//! Runnable demo: an in-process fake Rheovela Worker API server plus the
//! std-only worker SDK. Prints the processed count and final item states.

use std::collections::HashMap;
use std::io::{Read, Write};
use std::net::{TcpListener, TcpStream};
use std::sync::{Arc, Mutex};
use std::thread;

use rheovela_worker::{Client, Worker};

const POLL_BODY: &str = r#"[
  {"id":"task-1","instance_id":"inst-1","activity_id":"act-1","state":"ready"},
  {"id":"task-2","instance_id":"inst-1","activity_id":"act-2","state":"ready"}
]"#;

type States = Arc<Mutex<HashMap<String, String>>>;

/// Minimal fake Worker HTTP API. Records the final state of each item.
fn serve_one(mut stream: TcpStream, states: States) {
    let mut buf = [0u8; 4096];
    let n = match stream.read(&mut buf) {
        Ok(n) if n > 0 => n,
        _ => return,
    };
    let raw = String::from_utf8_lossy(&buf[..n]).to_string();
    let mut lines = raw.lines();
    let request_line = lines.next().unwrap_or("");
    let mut parts = request_line.split_whitespace();
    let method = parts.next().unwrap_or("").to_string();
    let path = parts.next().unwrap_or("").to_string();
    let body = match raw.find("\r\n\r\n") {
        Some(i) => raw[i + 4..].to_string(),
        None => String::new(),
    };

    let (status, payload) = route(&method, &path, &body, &states);
    let payload = format!("{}\r\n", payload);
    let head = format!(
        "HTTP/1.1 {status}\r\nContent-Length: {}\r\nContent-Type: application/json\r\nConnection: close\r\n\r\n",
        payload.len()
    );
    let _ = stream.write_all(head.as_bytes());
    let _ = stream.write_all(payload.as_bytes());
}

fn route(method: &str, path: &str, _body: &str, states: &States) -> (u16, String) {
    if method == "GET" && path.starts_with("/api/v1/work-items") {
        return (200, POLL_BODY.to_string());
    }
    if method == "POST" && path.starts_with("/api/v1/work-items/") {
        let rest = &path["/api/v1/work-items/".len()..];
        if rest.ends_with("/claim") {
            return (200, "{\"token\":\"t1\"}".to_string());
        }
        if let Some(id) = rest.strip_suffix("/complete") {
            states.lock().unwrap().insert(id.to_string(), "done".to_string());
            return (200, "{}".to_string());
        }
        if let Some(id) = rest.strip_suffix("/fail") {
            states.lock().unwrap().insert(id.to_string(), "failed".to_string());
            return (200, "{}".to_string());
        }
        if rest.ends_with("/heartbeat") {
            return (200, "{}".to_string());
        }
    }
    (404, "{\"error\":\"not found\"}".to_string())
}

fn main() {
    let listener = TcpListener::bind("127.0.0.1:0").expect("bind fake server");
    let addr = listener.local_addr().expect("fake server addr");
    let states: States = Arc::new(Mutex::new(HashMap::new()));

    let server_states = states.clone();
    thread::spawn(move || {
        for stream in listener.incoming() {
            let Ok(stream) = stream else { continue };
            let states = server_states.clone();
            thread::spawn(move || serve_one(stream, states));
        }
    });

    let worker = Worker::new(Client::new(&format!("http://{addr}")));

    let processed = worker
        .process_once(
            |item| {
                if item.id == "task-1" {
                    Ok(())
                } else {
                    Err("boom".to_string())
                }
            },
            "30s",
        )
        .expect("process_once");

    println!("processed={processed}");

    let states = states.lock().unwrap();
    for id in ["task-1", "task-2"] {
        let state = states.get(id).map(|s| s.as_str()).unwrap_or("pending");
        println!("{id}: {state}");
    }
}
