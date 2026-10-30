//! Rheovela Rust worker SDK (std-only, zero external crates).
//!
//! Implements a minimal HTTP/1.1 client over `std::net::TcpStream` and a
//! worker for the Rheovela Worker HTTP API (`rheo serve`):
//!
//! - `GET  /api/v1/work-items[?instance=]` — list ready work items
//! - `GET  /api/v1/work-items/{id}` — get one work item (result/approval)
//! - `POST /api/v1/work-items/{id}/claim` — claim (returns fencing token)
//! - `POST /api/v1/work-items/{id}/heartbeat` — renew lease
//! - `POST /api/v1/work-items/{id}/complete` — complete (token-guarded)
//! - `POST /api/v1/work-items/{id}/request-approval` — request approval
//! - `POST /api/v1/work-items/{id}/approve` — approve a pending item
//! - `POST /api/v1/work-items/{id}/reject` — reject a pending item
//! - `POST /api/v1/work-items/{id}/cancel` — cancel an item
//! - `POST /api/v1/work-items/{id}/fail` — fail with error message
//!
//! JSON handling is deliberately minimal: field values are extracted by
//! scanning for top-level keys. Arbitrary JSON values (`result`, `final_args`)
//! are carried as raw JSON strings and embedded verbatim in request bodies.

use std::io::{Read, Write};
use std::net::TcpStream;
use std::time::Duration;

pub mod events;

/// Approval state attached to a work item.
#[derive(Debug, Clone, PartialEq)]
pub struct Approval {
    pub state: String,
    pub pending_stage: Option<String>,
    /// Raw JSON value supplied by the approval requester.
    pub final_args: Option<String>,
    pub reason: Option<String>,
    pub requested_by: Option<String>,
    pub requested_at: Option<String>,
    pub decided_by: Option<String>,
    pub decided_at: Option<String>,
    pub comment: Option<String>,
    pub checkpoint_ref: Option<String>,
    pub effect_cursor: Option<String>,
}

impl Approval {
    /// Parse an approval object. Missing fields become empty/default values.
    pub fn parse(s: &str) -> Approval {
        Approval {
            state: json_field(s, "state").unwrap_or_default(),
            pending_stage: json_field(s, "pending_stage"),
            final_args: json_value(s, "final_args").filter(|v| v != "null"),
            reason: json_field(s, "reason"),
            requested_by: json_field(s, "requested_by"),
            requested_at: json_field(s, "requested_at"),
            decided_by: json_field(s, "decided_by"),
            decided_at: json_field(s, "decided_at"),
            comment: json_field(s, "comment"),
            checkpoint_ref: json_field(s, "checkpoint_ref"),
            effect_cursor: json_field(s, "effect_cursor"),
        }
    }
}

/// A single work item returned by `poll`/`get`.
#[derive(Debug, Clone, PartialEq)]
pub struct WorkItem {
    pub id: String,
    pub instance_id: String,
    pub activity_id: String,
    pub state: String,
    /// Raw JSON value of the item's typed result.
    pub result: Option<String>,
    pub checkpoint_ref: String,
    pub effect_cursor: String,
    pub attempts: u32,
    pub max_attempts: u32,
    pub last_error: Option<String>,
    pub approval: Option<Approval>,
}

impl WorkItem {
    /// Parse a `WorkItem` from a JSON object. Returns None when `id` is absent.
    pub fn parse(s: &str) -> Option<WorkItem> {
        Some(WorkItem {
            id: json_field(s, "id")?,
            instance_id: json_field(s, "instance_id").unwrap_or_default(),
            activity_id: json_field(s, "activity_id").unwrap_or_default(),
            state: json_field(s, "state").unwrap_or_default(),
            result: json_value(s, "result").filter(|v| v != "null"),
            checkpoint_ref: json_field(s, "checkpoint_ref").unwrap_or_default(),
            effect_cursor: json_field(s, "effect_cursor").unwrap_or_default(),
            attempts: json_int(s, "attempts"),
            max_attempts: json_int(s, "max_attempts"),
            last_error: json_field(s, "last_error"),
            approval: json_object(s, "approval").map(|o| Approval::parse(&o)),
        })
    }
}

/// Request human approval before a work item advances.
#[derive(Debug, Clone, PartialEq)]
pub struct ApprovalRequest {
    pub actor: String,
    pub reason: String,
    /// Raw JSON value embedded verbatim in the request body.
    pub final_args: Option<String>,
    pub checkpoint_ref: String,
    pub effect_cursor: String,
}

/// Outcome reported by a worker callback.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum WorkStatus {
    Success,
    Failure,
    Approval,
}

/// Rich result returned by the worker callback.
#[derive(Debug, Clone, PartialEq)]
pub struct WorkResult {
    pub status: WorkStatus,
    pub error: Option<String>,
    pub outcome: Option<String>,
    /// Raw JSON value embedded verbatim in the completion body.
    pub result: Option<String>,
    pub checkpoint_ref: Option<String>,
    pub effect_cursor: Option<String>,
    pub approval: Option<ApprovalRequest>,
}

impl WorkResult {
    /// Successful completion without a result payload.
    pub fn success() -> WorkResult {
        WorkResult {
            status: WorkStatus::Success,
            error: None,
            outcome: None,
            result: None,
            checkpoint_ref: None,
            effect_cursor: None,
            approval: None,
        }
    }

    /// Successful completion with a raw JSON result and optional cursors.
    pub fn success_with(
        result: Option<String>,
        checkpoint_ref: Option<String>,
        effect_cursor: Option<String>,
    ) -> WorkResult {
        WorkResult {
            result,
            checkpoint_ref,
            effect_cursor,
            ..WorkResult::success()
        }
    }

    /// Failed completion with an error message.
    pub fn failure(error: impl Into<String>) -> WorkResult {
        WorkResult {
            status: WorkStatus::Failure,
            error: Some(error.into()),
            ..WorkResult::success()
        }
    }

    /// Request human approval.
    pub fn approval(approval: ApprovalRequest) -> WorkResult {
        WorkResult {
            status: WorkStatus::Approval,
            approval: Some(approval),
            ..WorkResult::success()
        }
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

    /// GET `/api/v1/work-items/{id}` — fetch one work item.
    pub fn get(&self, id: &str) -> Result<WorkItem, WorkerError> {
        let body = self
            .client
            .request("GET", &format!("/api/v1/work-items/{}", id), None)?;
        WorkItem::parse(&body).ok_or_else(|| WorkerError {
            status: 200,
            message: "work item response missing id".to_string(),
        })
    }

    /// POST `/api/v1/work-items/{id}/claim` — claim and return the fencing token.
    pub fn claim(&self, id: &str, lease: &str) -> Result<String, WorkerError> {
        self.claim_with_options(id, lease, None, None, None)
    }

    /// POST claim with optional actor/tools/credential guards.
    pub fn claim_with_options(
        &self,
        id: &str,
        lease: &str,
        actor: Option<&str>,
        tools_json: Option<&str>,
        credential: Option<&str>,
    ) -> Result<String, WorkerError> {
        let mut body = format!("{{\"lease\":\"{}\"", json_escape(lease));
        if let Some(actor) = actor.filter(|v| !v.is_empty()) {
            body.push_str(",\"actor\":\"");
            body.push_str(&json_escape(actor));
            body.push('"');
        }
        if let Some(tools) = tools_json.filter(|v| !v.is_empty()) {
            body.push_str(",\"tools\":");
            body.push_str(tools);
        }
        if let Some(credential) = credential.filter(|v| !v.is_empty()) {
            body.push_str(",\"credential\":\"");
            body.push_str(&json_escape(credential));
            body.push('"');
        }
        body.push('}');
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
        self.complete_with_result(id, token, None, None, None)
    }

    /// POST `/api/v1/work-items/{id}/complete` with an optional raw JSON
    /// result and checkpoint/effect references. Empty values are omitted, so
    /// a token-only body remains compatible with older servers.
    pub fn complete_with_result(
        &self,
        id: &str,
        token: &str,
        result: Option<&str>,
        checkpoint: Option<&str>,
        effect: Option<&str>,
    ) -> Result<(), WorkerError> {
        self.complete_with_evidence(id, token, result, None, 0, checkpoint, effect)
    }

    /// Complete with optional result/evidence JSON, cost, and cursors.
    pub fn complete_with_evidence(
        &self,
        id: &str,
        token: &str,
        result: Option<&str>,
        evidence: Option<&str>,
        cost: u64,
        checkpoint: Option<&str>,
        effect: Option<&str>,
    ) -> Result<(), WorkerError> {
        let mut body = format!("{{\"token\":\"{}\"", json_escape(token));
        if let Some(result) = result.filter(|v| !v.is_empty()) {
            body.push_str(",\"result\":");
            body.push_str(result);
        }
        if let Some(evidence) = evidence.filter(|v| !v.is_empty()) {
            body.push_str(",\"evidence\":");
            body.push_str(evidence);
        }
        if cost != 0 {
            body.push_str(&format!(",\"cost\":{}", cost));
        }
        if let Some(checkpoint) = checkpoint.filter(|v| !v.is_empty()) {
            body.push_str(",\"checkpoint_ref\":\"");
            body.push_str(&json_escape(checkpoint));
            body.push('"');
        }
        if let Some(effect) = effect.filter(|v| !v.is_empty()) {
            body.push_str(",\"effect_cursor\":\"");
            body.push_str(&json_escape(effect));
            body.push('"');
        }
        body.push('}');
        self.client
            .request("POST", &format!("/api/v1/work-items/{}/complete", id), Some(&body))?;
        Ok(())
    }

    /// POST `/api/v1/work-items/{id}/request-approval`.
    pub fn request_approval(
        &self,
        id: &str,
        token: &str,
        req: &ApprovalRequest,
    ) -> Result<(), WorkerError> {
        let mut body = format!(
            "{{\"token\":\"{}\",\"reason\":\"{}\"",
            json_escape(token),
            json_escape(&req.reason)
        );
        if !req.actor.is_empty() {
            body.push_str(",\"actor\":\"");
            body.push_str(&json_escape(&req.actor));
            body.push('"');
        }
        if let Some(final_args) = req.final_args.as_deref().filter(|v| !v.is_empty()) {
            body.push_str(",\"final_args\":");
            body.push_str(final_args);
        }
        if !req.checkpoint_ref.is_empty() {
            body.push_str(",\"checkpoint_ref\":\"");
            body.push_str(&json_escape(&req.checkpoint_ref));
            body.push('"');
        }
        if !req.effect_cursor.is_empty() {
            body.push_str(",\"effect_cursor\":\"");
            body.push_str(&json_escape(&req.effect_cursor));
            body.push('"');
        }
        body.push('}');
        self.client.request(
            "POST",
            &format!("/api/v1/work-items/{}/request-approval", id),
            Some(&body),
        )?;
        Ok(())
    }

    /// POST `/api/v1/work-items/{id}/approve`.
    pub fn approve(
        &self,
        id: &str,
        actor: &str,
        comment: &str,
        final_args: Option<&str>,
    ) -> Result<(), WorkerError> {
        let mut body = format!(
            "{{\"actor\":\"{}\",\"comment\":\"{}\"",
            json_escape(actor),
            json_escape(comment)
        );
        if let Some(final_args) = final_args.filter(|v| !v.is_empty()) {
            body.push_str(",\"final_args\":");
            body.push_str(final_args);
        }
        body.push('}');
        self.client
            .request("POST", &format!("/api/v1/work-items/{}/approve", id), Some(&body))?;
        Ok(())
    }

    /// POST `/api/v1/work-items/{id}/reject`.
    pub fn reject(&self, id: &str, actor: &str, reason: &str) -> Result<(), WorkerError> {
        let body = format!(
            "{{\"actor\":\"{}\",\"reason\":\"{}\"}}",
            json_escape(actor),
            json_escape(reason)
        );
        self.client
            .request("POST", &format!("/api/v1/work-items/{}/reject", id), Some(&body))?;
        Ok(())
    }

    /// POST `/api/v1/work-items/{id}/cancel`.
    pub fn cancel(&self, id: &str) -> Result<(), WorkerError> {
        self.client.request(
            "POST",
            &format!("/api/v1/work-items/{}/cancel", id),
            Some("{}"),
        )?;
        Ok(())
    }

    /// POST `/api/v1/work-items/{id}/fail` — mark the item failed.
    pub fn fail(&self, id: &str, token: &str, err: &str) -> Result<(), WorkerError> {
        let body = format!(
            "{{\"token\":\"{}\",\"error\":\"{}\"}}",
            json_escape(token),
            json_escape(err)
        );
        self.client
            .request("POST", &format!("/api/v1/work-items/{}/fail", id), Some(&body))?;
        Ok(())
    }

    /// One poll-then-process pass: poll → claim each item (skip items that
    /// cannot be claimed, e.g. already claimed by another worker) → run `f` →
    /// settle the returned `WorkResult`. Returns the number of processed items.
    ///
    /// - `success` completes with result/cursors when present, else token-only.
    /// - `failure` fails with the error (default `"failure"`).
    /// - `approval` requests approval; a missing request fails the item.
    pub fn process_once(
        &self,
        f: impl Fn(&WorkItem) -> WorkResult,
        lease: &str,
    ) -> Result<usize, WorkerError> {
        let items = self.poll(None)?;
        let mut processed = 0;
        for item in &items {
            let token = match self.claim(&item.id, lease) {
                Ok(t) => t,
                Err(_) => continue, // not claimable (claimed / expired / gone)
            };
            let res = f(item);
            let settled = match res.status {
                WorkStatus::Success => {
                    if is_nonempty(&res.result)
                        || is_nonempty(&res.checkpoint_ref)
                        || is_nonempty(&res.effect_cursor)
                    {
                        self.complete_with_result(
                            &item.id,
                            &token,
                            res.result.as_deref(),
                            res.checkpoint_ref.as_deref(),
                            res.effect_cursor.as_deref(),
                        )
                    } else {
                        self.complete(&item.id, &token)
                    }
                }
                WorkStatus::Failure => self.fail(
                    &item.id,
                    &token,
                    res.error.as_deref().unwrap_or("failure"),
                ),
                WorkStatus::Approval => match res.approval.as_ref() {
                    None => self.fail(
                        &item.id,
                        &token,
                        "approval result is missing approval request",
                    ),
                    Some(req) => {
                        let merged = ApprovalRequest {
                            actor: req.actor.clone(),
                            reason: req.reason.clone(),
                            final_args: req.final_args.clone(),
                            checkpoint_ref: if req.checkpoint_ref.is_empty() {
                                res.checkpoint_ref.clone().unwrap_or_default()
                            } else {
                                req.checkpoint_ref.clone()
                            },
                            effect_cursor: if req.effect_cursor.is_empty() {
                                res.effect_cursor.clone().unwrap_or_default()
                            } else {
                                req.effect_cursor.clone()
                            },
                        };
                        match self.request_approval(&item.id, &token, &merged) {
                            Ok(()) => Ok(()),
                            Err(e) => self.fail(&item.id, &token, &e.to_string()),
                        }
                    }
                },
            };
            if settled.is_ok() {
                processed += 1;
            }
        }
        Ok(processed)
    }
}

fn is_nonempty(value: &Option<String>) -> bool {
    matches!(value.as_deref(), Some(v) if !v.is_empty())
}

/// Parse `http://host[:port]` into `(host, port)`. Defaults to port 80.
fn parse_base(base: &str) -> (String, u16) {
    let b = base.strip_prefix("http://").unwrap_or(base);
    match b.rsplit_once(':') {
        Some((h, p)) => (h.to_string(), p.parse().unwrap_or(80)),
        None => (b.to_string(), 80),
    }
}

/// Extract the raw JSON value of a top-level `"key"` (quoted string, object,
/// array, number, boolean, or null). Returns None if the key is absent.
pub fn json_value(s: &str, key: &str) -> Option<String> {
    let colon = index_of_key(s, key)?;
    let bytes = s.as_bytes();
    let mut start = colon + 1;
    while start < bytes.len() && (bytes[start] as char).is_whitespace() {
        start += 1;
    }
    if start >= bytes.len() {
        return None;
    }
    match bytes[start] {
        b'"' => {
            let mut end = start + 1;
            let mut escaped = false;
            while end < bytes.len() {
                if escaped {
                    escaped = false;
                } else if bytes[end] == b'\\' {
                    escaped = true;
                } else if bytes[end] == b'"' {
                    end += 1;
                    break;
                }
                end += 1;
            }
            Some(s[start..end].to_string())
        }
        b'{' | b'[' => {
            let end = matching_bracket(s, start)?;
            Some(s[start..=end].to_string())
        }
        _ => {
            let mut end = start;
            while end < bytes.len() {
                let c = bytes[end];
                if c == b','
                    || c == b'}'
                    || c == b']'
                    || (c as char).is_whitespace()
                {
                    break;
                }
                end += 1;
            }
            if end == start {
                None
            } else {
                Some(s[start..end].to_string())
            }
        }
    }
}

/// Extract a nested JSON object for a top-level `"key"`, or None.
pub fn json_object(s: &str, key: &str) -> Option<String> {
    json_value(s, key).filter(|v| v.starts_with('{'))
}

/// Extract the decoded value of a top-level `"key":"value"`, honoring
/// backslash escapes inside the value. Returns None if the key is absent.
pub fn json_field(s: &str, key: &str) -> Option<String> {
    let raw = json_value(s, key)?;
    if raw.len() >= 2 && raw.starts_with('"') && raw.ends_with('"') {
        Some(unescape(&raw[1..raw.len() - 1]))
    } else {
        None
    }
}

fn json_int(s: &str, key: &str) -> u32 {
    json_value(s, key)
        .and_then(|v| v.trim().parse::<u32>().ok())
        .unwrap_or(0)
}

/// Find `"key":` at the top level of a JSON object. Returns the colon index.
fn index_of_key(s: &str, key: &str) -> Option<usize> {
    let bytes = s.as_bytes();
    let mut depth = 0i32;
    let mut i = 0;
    while i < bytes.len() {
        let c = bytes[i];
        if c == b'"' {
            let content_start = i + 1;
            let mut end = content_start;
            let mut escaped = false;
            while end < bytes.len() {
                if escaped {
                    escaped = false;
                } else if bytes[end] == b'\\' {
                    escaped = true;
                } else if bytes[end] == b'"' {
                    break;
                }
                end += 1;
            }
            let content = &s[content_start..end];
            let mut colon = end + 1;
            while colon < bytes.len() && (bytes[colon] as char).is_whitespace() {
                colon += 1;
            }
            if depth == 1
                && colon < bytes.len()
                && bytes[colon] == b':'
                && content == key
            {
                return Some(colon);
            }
            i = end + 1;
            continue;
        }
        if c == b'{' || c == b'[' {
            depth += 1;
        } else if c == b'}' || c == b']' {
            depth -= 1;
        }
        i += 1;
    }
    None
}

fn matching_bracket(s: &str, start: usize) -> Option<usize> {
    let bytes = s.as_bytes();
    let open = *bytes.get(start)?;
    let close = if open == b'{' { b'}' } else { b']' };
    let mut depth = 0i32;
    let mut in_string = false;
    let mut i = start;
    while i < bytes.len() {
        let c = bytes[i];
        if in_string {
            if c == b'\\' {
                i += 2;
                continue;
            }
            if c == b'"' {
                in_string = false;
            }
        } else if c == b'"' {
            in_string = true;
        } else if c == open {
            depth += 1;
        } else if c == close {
            depth -= 1;
            if depth == 0 {
                return Some(i);
            }
        }
        i += 1;
    }
    None
}

fn unescape(s: &str) -> String {
    let chars: Vec<char> = s.chars().collect();
    let mut out = String::with_capacity(s.len());
    let mut i = 0;
    while i < chars.len() {
        let c = chars[i];
        if c == '\\' && i + 1 < chars.len() {
            let n = chars[i + 1];
            match n {
                'n' => {
                    out.push('\n');
                    i += 2;
                }
                'r' => {
                    out.push('\r');
                    i += 2;
                }
                't' => {
                    out.push('\t');
                    i += 2;
                }
                'b' => {
                    out.push('\u{08}');
                    i += 2;
                }
                'f' => {
                    out.push('\u{0c}');
                    i += 2;
                }
                'u' => {
                    if i + 5 < chars.len() {
                        let hex: String = chars[i + 2..i + 6].iter().collect();
                        if let Ok(code) = u32::from_str_radix(&hex, 16) {
                            if let Some(ch) = char::from_u32(code) {
                                out.push(ch);
                            }
                        }
                        i += 6;
                    } else {
                        out.push(n);
                        i += 2;
                    }
                }
                _ => {
                    out.push(n);
                    i += 2;
                }
            }
        } else {
            out.push(c);
            i += 1;
        }
    }
    out
}

fn json_escape(s: &str) -> String {
    let mut out = String::with_capacity(s.len());
    for c in s.chars() {
        match c {
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            '\u{08}' => out.push_str("\\b"),
            '\u{0c}' => out.push_str("\\f"),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            '\t' => out.push_str("\\t"),
            c if (c as u32) < 0x20 => out.push_str(&format!("\\u{:04x}", c as u32)),
            c => out.push(c),
        }
    }
    out
}

/// Parse a JSON array of objects into `WorkItem`s by scanning balanced
/// `{` ... `}` blocks. Tolerant of surrounding whitespace and unknown fields.
pub fn parse_items(s: &str) -> Vec<WorkItem> {
    let bytes = s.as_bytes();
    let mut items = Vec::new();
    let mut i = 0;
    while i < bytes.len() {
        if bytes[i] == b'{' {
            if let Some(j) = matching_bracket(s, i) {
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

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn json_value_handles_raw_values() {
        let body = r#"{"obj":{"a":1},"arr":[1,2],"str":"hi","num":42,"flag":true,"none":null}"#;
        assert_eq!(json_value(body, "obj").as_deref(), Some(r#"{"a":1}"#));
        assert_eq!(json_value(body, "arr").as_deref(), Some("[1,2]"));
        assert_eq!(json_value(body, "str").as_deref(), Some(r#""hi""#));
        assert_eq!(json_value(body, "num").as_deref(), Some("42"));
        assert_eq!(json_value(body, "flag").as_deref(), Some("true"));
        assert_eq!(json_value(body, "none").as_deref(), Some("null"));
        assert_eq!(json_value(body, "missing"), None);
    }

    #[test]
    fn json_field_unescapes_strings() {
        let body = r#"{"s":"a\"b\nc"}"#;
        assert_eq!(json_field(body, "s").as_deref(), Some("a\"b\nc"));
    }

    #[test]
    fn parses_nested_approval_and_top_level_fields() {
        let body = r#"{"id":"wi1","instance_id":"i1","activity_id":"a1","state":"awaiting_approval","result":{"ok":true},"checkpoint_ref":"cp","effect_cursor":"cur","attempts":2,"max_attempts":5,"last_error":"retry","approval":{"state":"pending","pending_stage":"review","final_args":{"x":1},"reason":"why","requested_by":"agent","requested_at":"t0","decided_by":"human","decided_at":"t1","comment":"ok","checkpoint_ref":"cp2","effect_cursor":"cur2"}}"#;
        let item = WorkItem::parse(body).expect("parse work item");
        assert_eq!(item.id, "wi1");
        assert_eq!(item.instance_id, "i1");
        assert_eq!(item.activity_id, "a1");
        assert_eq!(item.state, "awaiting_approval");
        assert_eq!(item.result.as_deref(), Some(r#"{"ok":true}"#));
        assert_eq!(item.checkpoint_ref, "cp");
        assert_eq!(item.effect_cursor, "cur");
        assert_eq!(item.attempts, 2);
        assert_eq!(item.max_attempts, 5);
        assert_eq!(item.last_error.as_deref(), Some("retry"));

        let approval = item.approval.expect("approval present");
        assert_eq!(approval.state, "pending");
        assert_eq!(approval.pending_stage.as_deref(), Some("review"));
        assert_eq!(approval.final_args.as_deref(), Some(r#"{"x":1}"#));
        assert_eq!(approval.reason.as_deref(), Some("why"));
        assert_eq!(approval.requested_by.as_deref(), Some("agent"));
        assert_eq!(approval.requested_at.as_deref(), Some("t0"));
        assert_eq!(approval.decided_by.as_deref(), Some("human"));
        assert_eq!(approval.decided_at.as_deref(), Some("t1"));
        assert_eq!(approval.comment.as_deref(), Some("ok"));
        assert_eq!(approval.checkpoint_ref.as_deref(), Some("cp2"));
        assert_eq!(approval.effect_cursor.as_deref(), Some("cur2"));
    }

    #[test]
    fn json_object_requires_an_object() {
        let body = r#"{"approval":{"state":"pending"},"id":"wi1"}"#;
        assert_eq!(
            json_object(body, "approval").as_deref(),
            Some(r#"{"state":"pending"}"#)
        );
        assert_eq!(json_object(body, "id"), None);
        assert_eq!(json_object(body, "missing"), None);
    }

    #[test]
    fn work_result_helpers() {
        let ok = WorkResult::success();
        assert_eq!(ok.status, WorkStatus::Success);
        assert!(ok.result.is_none());
        assert!(ok.error.is_none());
        assert!(ok.approval.is_none());

        let rich = WorkResult::success_with(
            Some(r#"{"a":1}"#.to_string()),
            Some("cp".to_string()),
            Some("cur".to_string()),
        );
        assert_eq!(rich.status, WorkStatus::Success);
        assert_eq!(rich.result.as_deref(), Some(r#"{"a":1}"#));
        assert_eq!(rich.checkpoint_ref.as_deref(), Some("cp"));
        assert_eq!(rich.effect_cursor.as_deref(), Some("cur"));

        let failed = WorkResult::failure("boom");
        assert_eq!(failed.status, WorkStatus::Failure);
        assert_eq!(failed.error.as_deref(), Some("boom"));

        let req = ApprovalRequest {
            actor: "agent".to_string(),
            reason: "review".to_string(),
            final_args: Some(r#"{"x":1}"#.to_string()),
            checkpoint_ref: "cp".to_string(),
            effect_cursor: "cur".to_string(),
        };
        let approval = WorkResult::approval(req);
        assert_eq!(approval.status, WorkStatus::Approval);
        assert_eq!(approval.approval.expect("approval").reason, "review");
    }
}
