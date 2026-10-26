//! Runnable demo + contract check. Serves a fake Worker HTTP API in-process,
//! exercises the v1.4.0 client surface, and fails if the recorded requests do
//! not match the expected endpoint/body sequence.

use std::io::{Read, Write};
use std::net::{TcpListener, TcpStream};
use std::sync::{Arc, Mutex};
use std::thread;

use rheovela_worker::{ApprovalRequest, Client, WorkResult, Worker};

const TASK1: &str = r#"{
  "id":"task-1","instance_id":"inst-1","activity_id":"act-1","state":"ready",
  "result":{"previous":true},"checkpoint_ref":"cp-0","effect_cursor":"cur-0",
  "attempts":2,"max_attempts":5,"last_error":"retry"
}"#;

const ITEMS: &str = r#"[
  {
    "id":"task-1","instance_id":"inst-1","activity_id":"act-1","state":"ready",
    "result":{"previous":true},"checkpoint_ref":"cp-0","effect_cursor":"cur-0",
    "attempts":2,"max_attempts":5,"last_error":"retry"
  },
  {
    "id":"task-2","instance_id":"inst-1","activity_id":"act-2",
    "state":"awaiting_approval","attempts":0,"max_attempts":1,"last_error":"",
    "approval":{
      "state":"pending","pending_stage":"review","final_args":{"seed":1},
      "reason":"needs review","requested_by":"worker","requested_at":"t0",
      "decided_by":"","decided_at":"","comment":"",
      "checkpoint_ref":"cp-req","effect_cursor":"cur-req"
    }
  }
]"#;

type Records = Arc<Mutex<Vec<String>>>;

fn main() {
    let listener = TcpListener::bind("127.0.0.1:0").expect("bind fake server");
    let addr = listener.local_addr().expect("fake server addr");
    let records: Records = Arc::new(Mutex::new(Vec::new()));

    let server_records = records.clone();
    thread::spawn(move || {
        for stream in listener.incoming() {
            let Ok(stream) = stream else { continue };
            let records = server_records.clone();
            thread::spawn(move || serve_one(stream, records));
        }
    });

    let worker = Worker::new(Client::new(&format!("http://{addr}")));

    let processed = worker
        .process_once(
            |item| {
                if item.id == "task-1" {
                    check(
                        item.result.as_deref() == Some(r#"{"previous":true}"#),
                        "task-1 result",
                    );
                    check(item.checkpoint_ref == "cp-0", "task-1 checkpoint_ref");
                    check(item.effect_cursor == "cur-0", "task-1 effect_cursor");
                    check(item.attempts == 2, "task-1 attempts");
                    check(item.max_attempts == 5, "task-1 max_attempts");
                    check(
                        item.last_error.as_deref() == Some("retry"),
                        "task-1 last_error",
                    );
                    check(item.approval.is_none(), "task-1 approval");
                    WorkResult::success_with(
                        Some(r#"{"answer":42}"#.to_string()),
                        Some("cp-1".to_string()),
                        Some("cur-1".to_string()),
                    )
                } else if item.id == "task-2" {
                    let approval = item.approval.as_ref().expect("task-2 approval");
                    check(
                        item.state == "awaiting_approval",
                        "task-2 state",
                    );
                    check(
                        approval.state == "pending",
                        "task-2 approval.state",
                    );
                    check(
                        approval.pending_stage.as_deref() == Some("review"),
                        "task-2 approval.pending_stage",
                    );
                    check(
                        approval.final_args.as_deref() == Some(r#"{"seed":1}"#),
                        "task-2 approval.final_args",
                    );
                    check(
                        approval.requested_by.as_deref() == Some("worker"),
                        "task-2 approval.requested_by",
                    );
                    check(item.attempts == 0, "task-2 attempts");
                    check(item.max_attempts == 1, "task-2 max_attempts");
                    let mut result = WorkResult::approval(ApprovalRequest {
                        actor: "worker".to_string(),
                        reason: "needs review".to_string(),
                        final_args: Some(r#"{"plan":"b"}"#.to_string()),
                        checkpoint_ref: String::new(),
                        effect_cursor: String::new(),
                    });
                    result.checkpoint_ref = Some("cp-res".to_string());
                    result.effect_cursor = Some("cur-res".to_string());
                    result
                } else {
                    WorkResult::failure(format!("unexpected item {}", item.id))
                }
            },
            "30s",
        )
        .expect("process_once");

    check(processed == 2, &format!("processed expected 2 but was {processed}"));

    let fetched = worker.get("task-1").expect("get task-1");
    check(fetched.id == "task-1", "get id");
    check(fetched.attempts == 2, "get attempts");

    worker
        .approve("task-2", "alice", "lgtm", Some(r#"{"plan":"c"}"#))
        .expect("approve");
    worker.reject("task-3", "bob", "nope").expect("reject");
    worker.cancel("task-4").expect("cancel");

    let expected: Vec<String> = vec![
        "GET /api/v1/work-items".to_string(),
        "POST /api/v1/work-items/task-1/claim {\"lease\":\"30s\"}".to_string(),
        "POST /api/v1/work-items/task-1/complete {\"token\":\"tok-1\",\"result\":{\"answer\":42},\"checkpoint_ref\":\"cp-1\",\"effect_cursor\":\"cur-1\"}".to_string(),
        "POST /api/v1/work-items/task-2/claim {\"lease\":\"30s\"}".to_string(),
        "POST /api/v1/work-items/task-2/request-approval {\"token\":\"tok-1\",\"reason\":\"needs review\",\"actor\":\"worker\",\"final_args\":{\"plan\":\"b\"},\"checkpoint_ref\":\"cp-res\",\"effect_cursor\":\"cur-res\"}".to_string(),
        "GET /api/v1/work-items/task-1".to_string(),
        "POST /api/v1/work-items/task-2/approve {\"actor\":\"alice\",\"comment\":\"lgtm\",\"final_args\":{\"plan\":\"c\"}}".to_string(),
        "POST /api/v1/work-items/task-3/reject {\"actor\":\"bob\",\"reason\":\"nope\"}".to_string(),
        "POST /api/v1/work-items/task-4/cancel {}".to_string(),
    ];

    let actual = records.lock().unwrap().clone();
    check(
        actual == expected,
        &format!("records mismatch\nexpected={expected:?}\nactual={actual:?}"),
    );

    println!("processed={processed}");
    println!("ok");
}

fn check(ok: bool, message: &str) {
    if !ok {
        panic!("{message}");
    }
}

fn serve_one(mut stream: TcpStream, records: Records) {
    let Some(raw) = read_request(&mut stream) else {
        return;
    };
    let mut lines = raw.lines();
    let request_line = lines.next().unwrap_or("");
    let mut parts = request_line.split_whitespace();
    let method = parts.next().unwrap_or("");
    let path = parts.next().unwrap_or("");
    let body = match raw.find("\r\n\r\n") {
        Some(i) => raw[i + 4..].to_string(),
        None => String::new(),
    };

    let (status, payload) = route(method, path, &body, &records);
    let payload = format!("{payload}\r\n");
    let head = format!(
        "HTTP/1.1 {status}\r\nContent-Length: {}\r\nContent-Type: application/json\r\nConnection: close\r\n\r\n",
        payload.len()
    );
    let _ = stream.write_all(head.as_bytes());
    let _ = stream.write_all(payload.as_bytes());
}

fn route(method: &str, path: &str, body: &str, records: &Records) -> (u16, String) {
    let record = if body.is_empty() {
        format!("{method} {path}")
    } else {
        format!("{method} {path} {body}")
    };
    records.lock().unwrap().push(record);

    if method == "GET" && path == "/api/v1/work-items" {
        (200, ITEMS.to_string())
    } else if method == "GET" && path.starts_with("/api/v1/work-items/") {
        (200, TASK1.to_string())
    } else if method == "POST" && path.ends_with("/claim") {
        (200, "{\"token\":\"tok-1\",\"lease\":\"30s\"}".to_string())
    } else if method == "POST"
        && (path.ends_with("/complete")
            || path.ends_with("/request-approval")
            || path.ends_with("/approve")
            || path.ends_with("/reject")
            || path.ends_with("/cancel")
            || path.ends_with("/fail"))
    {
        (200, "{\"ok\":true}".to_string())
    } else {
        (404, "{\"error\":\"not found\"}".to_string())
    }
}

fn read_request(stream: &mut TcpStream) -> Option<String> {
    let mut buf = Vec::new();
    let mut chunk = [0u8; 4096];
    loop {
        let n = match stream.read(&mut chunk) {
            Ok(0) => break,
            Ok(n) => n,
            Err(_) => return None,
        };
        buf.extend_from_slice(&chunk[..n]);
        if let Some(pos) = find_bytes(&buf, b"\r\n\r\n") {
            let head = String::from_utf8_lossy(&buf[..pos]);
            let len = content_length(&head);
            if buf.len() >= pos + 4 + len {
                break;
            }
        }
    }
    if buf.is_empty() {
        None
    } else {
        Some(String::from_utf8_lossy(&buf).to_string())
    }
}

fn find_bytes(haystack: &[u8], needle: &[u8]) -> Option<usize> {
    haystack.windows(needle.len()).position(|w| w == needle)
}

fn content_length(head: &str) -> usize {
    head.lines()
        .find_map(|line| {
            let (name, value) = line.split_once(':')?;
            if name.trim().eq_ignore_ascii_case("content-length") {
                value.trim().parse().ok()
            } else {
                None
            }
        })
        .unwrap_or(0)
}
