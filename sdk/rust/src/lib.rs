//! Rheovela Rust worker SDK (std-only, zero external crates).
//!
//! Implements a minimal HTTP/1.1 client over `std::net::TcpStream` and a
//! worker for the Rheovela Worker HTTP API (`rheo serve`):
//!
//! - `GET  /api/v1/work-items[?instance=]` — list ready work items
//! - `POST /api/v1/work-items/{id}/claim` — claim (returns fencing token)
//! - `POST /api/v1/work-items/{id}/heartbeat` — renew lease
//! - `POST /api/v1/work-items/{id}/complete` — complete (token-guarded)
//! - `POST /api/v1/work-items/{id}/fail` — fail with error message
//!
//! JSON handling is deliberately minimal: field values are extracted by
//! `"key":"value"` substring scanning, which is sufficient for the flat
//! fake-API object shape (`WorkItem`, `ClaimResponse`).

use std::io::{Read, Write};
use std::net::TcpStream;
use std::time::Duration;

/// A single work item returned by `poll`.
#[derive(Debug, Clone)]
pub struct WorkItem {
    pub id: String,
    pub instance_id: String,
    pub activity_id: String,
    pub state: String,
}

impl WorkItem {
    /// Parse a `WorkItem` from a flat JSON object such as
    /// `{"id":"task-1","instance_id":"i1","activity_id":"a1","state":"ready"}`.
    pub fn parse(s: &str) -> Option<WorkItem> {
        Some(WorkItem {
            id: json_field(s, "id")?,
            instance_id: json_field(s, "instance_id")?,
            activity_id: json_field(s, "activity_id")?,
            state: json_field(s, "state")?,
        })
    }
}

/// Error returned by the HTTP client or worker. `status` is the HTTP status
/// code for server responses (0 for transport-level failures).
#[derive(Debug, Clone)]
pub struct WorkerError {
    pub status: u16,
    pub message: String,
}

impl std::fmt::Display for WorkerError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "worker error (status {}): {}", self.status, self.message)
    }
}

impl std::error::Error for WorkerError {}

/// Minimal HTTP/1.1 client over a TCP socket.
#[derive(Debug, Clone)]
pub struct Client {
    base: String,
}

impl Client {
    pub fn new(base: &str) -> Client {
        Client {
            base: base.trim_end_matches('/').to_string(),
        }
    }

    /// Perform an HTTP/1.1 request. Returns the response body on any 2xx
    /// status; otherwise returns a `WorkerError` with the status code.
    pub fn request(
        &self,
        method: &str,
        path: &str,
        body: Option<&str>,
    ) -> Result<String, WorkerError> {
        let (host, port) = parse_base(&self.base);
        let body = body.unwrap_or("");

        let mut req = String::new();
        req.push_str(method);
        req.push(' ');
        req.push_str(path);
        req.push_str(" HTTP/1.1\r\n");
        if port == 80 {
            req.push_str(&format!("Host: {}\r\n", host));
        } else {
            req.push_str(&format!("Host: {}:{}\r\n", host, port));
        }
        req.push_str(&format!("Content-Length: {}\r\n", body.len()));
        req.push_str("Content-Type: application/json\r\n");
        req.push_str("Connection: close\r\n\r\n");
        req.push_str(body);

        let mut stream = TcpStream::connect((host.as_str(), port)).map_err(|e| WorkerError {
            status: 0,
            message: format!("connect {}:{}: {}", host, port, e),
        })?;
        // Guard against servers that never close the connection.
        let _ = stream.set_read_timeout(Some(Duration::from_secs(30)));
        stream
            .write_all(req.as_bytes())
            .map_err(|e| WorkerError {
                status: 0,
                message: format!("write request: {}", e),
            })?;

        let mut raw = String::new();
        stream
            .read_to_string(&mut raw)
            .map_err(|e| WorkerError {
                status: 0,
                message: format!("read response: {}", e),
            })?;

        // Split status line + headers from the body on the blank line.
        let (head, resp_body) = match raw.split_once("\r\n\r\n") {
            Some((h, b)) => (h.to_string(), b.trim().to_string()),
            None => (raw.clone(), String::new()),
        };

        let status_line = head.lines().next().unwrap_or("").to_string();
        let code: u16 = status_line
            .split_whitespace()
            .nth(1)
            .and_then(|c| c.parse().ok())
            .unwrap_or(0);

        if (200..300).contains(&code) {
            Ok(resp_body)
        } else {
            Err(WorkerError {
                status: code,
                message: if resp_body.is_empty() {
                    status_line
                } else {
                    resp_body
                },
            })
        }
    }
}

/// High-level worker over the Worker HTTP API.
#[derive(Debug, Clone)]
pub struct Worker {
    client: Client,
}

impl Worker {
    pub fn new(client: Client) -> Worker {
        Worker { client }
    }

    /// GET `/api/v1/work-items[?instance=]` — list ready work items.
    pub fn poll(&self, instance: Option<&str>) -> Result<Vec<WorkItem>, WorkerError> {
        let path = match instance {
            Some(i) => format!("/api/v1/work-items?instance={}", i),
            None => "/api/v1/work-items".to_string(),
        };
        let body = self.client.request("GET", &path, None)?;
        Ok(parse_items(&body))
    }

    /// POST `/api/v1/work-items/{id}/claim` — claim and return the fencing token.
    pub fn claim(&self, id: &str, lease: &str) -> Result<String, WorkerError> {
        let body = format!("{{\"lease\":\"{}\"}}", lease);
        let resp = self.client.request(
            "POST",
            &format!("/api/v1/work-items/{}/claim", id),
            Some(&body),
        )?;
        json_field(&resp, "token").ok_or_else(|| WorkerError {
            status: 200,
            message: "claim response missing token".to_string(),
        })
    }

    /// POST `/api/v1/work-items/{id}/complete` — mark the item done.
    pub fn complete(&self, id: &str, token: &str) -> Result<(), WorkerError> {
        let body = format!("{{\"token\":\"{}\"}}", token);
        self.client
            .request("POST", &format!("/api/v1/work-items/{}/complete", id), Some(&body))?;
        Ok(())
    }

    /// POST `/api/v1/work-items/{id}/fail` — mark the item failed.
    pub fn fail(&self, id: &str, token: &str, err: &str) -> Result<(), WorkerError> {
        let body = format!("{{\"token\":\"{}\",\"error\":\"{}\"}}", token, err);
        self.client
            .request("POST", &format!("/api/v1/work-items/{}/fail", id), Some(&body))?;
        Ok(())
    }

    /// One poll-then-process pass: poll → claim each item (skip items that
    /// cannot be claimed, e.g. already claimed by another worker) → run `f` →
    /// complete on Ok or fail on Err. Returns the number of processed items.
    pub fn process_once(
        &self,
        f: impl Fn(&WorkItem) -> Result<(), String>,
        lease: &str,
    ) -> Result<usize, WorkerError> {
        let items = self.poll(None)?;
        let mut processed = 0;
        for item in &items {
            let token = match self.claim(&item.id, lease) {
                Ok(t) => t,
                Err(_) => continue, // not claimable (claimed / expired / gone)
            };
            match f(item) {
                Ok(()) => self.complete(&item.id, &token)?,
                Err(e) => self.fail(&item.id, &token, &e)?,
            }
            processed += 1;
        }
        Ok(processed)
    }
}

/// Parse `http://host[:port]` into `(host, port)`. Defaults to port 80.
fn parse_base(base: &str) -> (String, u16) {
    let b = base.strip_prefix("http://").unwrap_or(base);
    match b.rsplit_once(':') {
        Some((h, p)) => (h.to_string(), p.parse().unwrap_or(80)),
        None => (b.to_string(), 80),
    }
}

/// Extract the value of `"key":"value"` from a flat JSON string, honoring
/// backslash escapes inside the value. Returns None if the key is absent.
pub fn json_field(s: &str, key: &str) -> Option<String> {
    let needle = format!("\"{}\"", key);
    let start = s.find(&needle)? + needle.len();
    let rest = s[start..].trim_start();
    let rest = rest.strip_prefix(':')?.trim_start();
    let rest = rest.strip_prefix('"')?;
    let mut out = String::new();
    let mut it = rest.chars();
    while let Some(c) = it.next() {
        if c == '\\' {
            out.push(it.next()?);
        } else if c == '"' {
            return Some(out);
        } else {
            out.push(c);
        }
    }
    None
}

/// Parse a JSON array of flat objects into `WorkItem`s by scanning balanced
/// `{` ... `}` blocks. Tolerant of surrounding whitespace and unknown fields.
pub fn parse_items(s: &str) -> Vec<WorkItem> {
    let bytes = s.as_bytes();
    let mut items = Vec::new();
    let mut i = 0;
    while i < bytes.len() {
        if bytes[i] == b'{' {
            let mut depth = 0u32;
            let mut j = i;
            while j < bytes.len() {
                if bytes[j] == b'{' {
                    depth += 1;
                } else if bytes[j] == b'}' {
                    depth -= 1;
                    if depth == 0 {
                        break;
                    }
                }
                j += 1;
            }
            if j < bytes.len() {
                if let Some(item) = WorkItem::parse(&s[i..=j]) {
                    items.push(item);
                }
                i = j + 1;
                continue;
            }
        }
        i += 1;
    }
    items
}
