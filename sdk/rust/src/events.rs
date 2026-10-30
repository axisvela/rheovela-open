//! Durable WebSocket event client (std-only).
//!
//! Mirrors the Go/Python/TypeScript/Java clients: file-backed cursor and
//! pending queue, `after_seq` reconnect, exponential backoff, at-least-once
//! handler delivery. Only `ws://` is supported (TLS needs an external crate).

use std::collections::VecDeque;
use std::fs;
use std::io::{self, Read, Write};
use std::net::{Shutdown, TcpStream};
use std::path::{Path, PathBuf};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Mutex;
use std::time::{Duration, SystemTime, UNIX_EPOCH};

#[derive(Debug, Clone, PartialEq)]
pub struct Event {
    pub seq: u64,
    pub type_: String,
    pub actor_id: String,
    pub payload: Option<String>,
    pub wall_time: String,
    pub raw: String,
}

pub struct DurableQueue {
    dir: PathBuf,
    last_seq: u64,
    pending: VecDeque<Event>,
}

impl DurableQueue {
    pub fn open(dir: &Path) -> io::Result<Self> {
        fs::create_dir_all(dir)?;
        let mut queue = DurableQueue {
            dir: dir.to_path_buf(),
            last_seq: 0,
            pending: VecDeque::new(),
        };
        queue.load()?;
        Ok(queue)
    }

    pub fn last_seq(&self) -> u64 {
        self.last_seq
    }

    pub fn pending(&self) -> Vec<Event> {
        self.pending.iter().cloned().collect()
    }

    pub fn push(&mut self, event: Event) -> io::Result<()> {
        if event.seq <= self.last_seq || self.pending.iter().any(|item| item.seq == event.seq) {
            return Ok(());
        }
        self.pending.push_back(event.clone());
        let mut events: Vec<_> = self.pending.iter().cloned().collect();
        events.sort_by_key(|item| item.seq);
        self.pending = events.into();
        use std::fs::OpenOptions;
        let mut file = OpenOptions::new()
            .create(true)
            .append(true)
            .open(self.pending_path())?;
        writeln!(file, "{}", event.raw)?;
        file.sync_all()
    }

    pub fn ack(&mut self, seq: u64) -> io::Result<()> {
        if seq > self.last_seq {
            self.last_seq = seq;
        }
        self.pending.retain(|event| event.seq > seq);
        self.rewrite_pending()?;
        self.write_sequence()
    }

    pub fn set_floor(&mut self, seq: u64) -> io::Result<()> {
        if !self.pending.is_empty() || seq <= self.last_seq {
            return Ok(());
        }
        self.last_seq = seq;
        self.write_sequence()
    }

    fn pending_path(&self) -> PathBuf {
        self.dir.join("events-pending.jsonl")
    }

    fn sequence_path(&self) -> PathBuf {
        self.dir.join("events-last-seq")
    }

    fn load(&mut self) -> io::Result<()> {
        if let Ok(raw) = fs::read_to_string(self.sequence_path()) {
            self.last_seq = raw.trim().parse().unwrap_or(0);
        }
        let raw = match fs::read_to_string(self.pending_path()) {
            Ok(raw) => raw,
            Err(error) if error.kind() == io::ErrorKind::NotFound => return Ok(()),
            Err(error) => return Err(error),
        };
        for line in raw.lines().filter(|line| !line.trim().is_empty()) {
            let event = parse_event(line);
            if event.seq > self.last_seq {
                self.pending.push_back(event);
            }
        }
        let mut events: Vec<_> = self.pending.iter().cloned().collect();
        events.sort_by_key(|item| item.seq);
        self.pending = events.into();
        Ok(())
    }

    fn rewrite_pending(&self) -> io::Result<()> {
        let mut out = String::new();
        for event in &self.pending {
            out.push_str(&event.raw);
            out.push('\n');
        }
        atomic_write(&self.pending_path(), out.as_bytes())
    }

    fn write_sequence(&self) -> io::Result<()> {
        atomic_write(&self.sequence_path(), self.last_seq.to_string().as_bytes())
    }
}

fn atomic_write(path: &Path, data: &[u8]) -> io::Result<()> {
    let tmp = path.with_extension("tmp");
    {
        let mut file = fs::File::create(&tmp)?;
        file.write_all(data)?;
        file.sync_all()?;
    }
    fs::rename(tmp, path)
}

pub struct Config {
    pub url: String,
    pub stream_id: String,
    pub state_dir: PathBuf,
    pub types: Vec<String>,
    pub correlation_key: String,
    pub predicate: String,
    pub after_seq: u64,
    pub reconnect_min: Duration,
    pub reconnect_max: Duration,
    pub retry_delay: Duration,
}

impl Config {
    pub fn new(url: impl Into<String>, stream_id: impl Into<String>, state_dir: impl Into<PathBuf>) -> Self {
        Config {
            url: url.into(),
            stream_id: stream_id.into(),
            state_dir: state_dir.into(),
            types: Vec::new(),
            correlation_key: String::new(),
            predicate: String::new(),
            after_seq: 0,
            reconnect_min: Duration::from_millis(200),
            reconnect_max: Duration::from_secs(30),
            retry_delay: Duration::from_secs(1),
        }
    }
}

pub struct Client {
    cfg: Config,
    queue: Mutex<DurableQueue>,
    handler: Box<dyn Fn(&Event) -> Result<(), String> + Send + Sync>,
    on_error: Option<Box<dyn Fn(&str) + Send + Sync>>,
    running: AtomicBool,
    active: Mutex<Option<TcpStream>>,
}

impl Client {
    pub fn new(
        cfg: Config,
        handler: Box<dyn Fn(&Event) -> Result<(), String> + Send + Sync>,
        on_error: Option<Box<dyn Fn(&str) + Send + Sync>>,
    ) -> io::Result<Self> {
        let mut queue = DurableQueue::open(&cfg.state_dir)?;
        if cfg.after_seq > 0 {
            queue.set_floor(cfg.after_seq)?;
        }
        Ok(Client {
            cfg,
            queue: Mutex::new(queue),
            handler,
            on_error,
            running: AtomicBool::new(true),
            active: Mutex::new(None),
        })
    }

    pub fn last_seq(&self) -> u64 {
        self.queue.lock().unwrap().last_seq()
    }

    pub fn pending(&self) -> Vec<Event> {
        self.queue.lock().unwrap().pending()
    }

    pub fn stop(&self) {
        self.running.store(false, Ordering::SeqCst);
        if let Some(stream) = self.active.lock().unwrap().take() {
            let _ = stream.shutdown(Shutdown::Both);
        }
    }

    pub fn run(&self) {
        let mut backoff = self.cfg.reconnect_min;
        while self.running.load(Ordering::SeqCst) {
            if let Err(error) = self.drain() {
                self.report(&error);
                sleep(self.cfg.retry_delay);
                continue;
            }
            match self.session() {
                Ok(()) => backoff = self.cfg.reconnect_min,
                Err(error) => {
                    if !self.running.load(Ordering::SeqCst) {
                        break;
                    }
                    self.report(&error);
                    sleep(backoff);
                    backoff = (backoff * 2).min(self.cfg.reconnect_max);
                }
            }
        }
    }

    fn drain(&self) -> Result<(), String> {
        let pending = self.queue.lock().unwrap().pending();
        for event in pending {
            (self.handler)(&event)?;
            self.queue
                .lock()
                .unwrap()
                .ack(event.seq)
                .map_err(|error| error.to_string())?;
        }
        Ok(())
    }

    fn session(&self) -> Result<(), String> {
        let (host, port, path) = parse_url(&self.cfg.url).map_err(|error| error.to_string())?;
        let mut stream = TcpStream::connect((host.as_str(), port)).map_err(|error| error.to_string())?;
        let wake = stream.try_clone().map_err(|error| error.to_string())?;
        *self.active.lock().unwrap() = Some(wake);
        let result = self.handshake_and_read(&mut stream, &host, port, &path);
        *self.active.lock().unwrap() = None;
        result
    }

    fn handshake_and_read(&self, stream: &mut TcpStream, host: &str, port: u16, path: &str) -> Result<(), String> {
        let key = ws_key();
        let request = format!(
            "GET {} HTTP/1.1\r\nHost: {}:{}\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: {}\r\nSec-WebSocket-Version: 13\r\n\r\n",
            endpoint_path(self, path),
            host,
            port,
            key
        );
        stream.write_all(request.as_bytes()).map_err(|error| error.to_string())?;
        read_handshake(stream)?;
        let mut partial = Vec::new();
        loop {
            if !self.running.load(Ordering::SeqCst) {
                return Ok(());
            }
            let (fin, opcode, payload) = read_frame(stream).map_err(|error| error.to_string())?;
            match opcode {
                0x1 | 0x0 => {
                    partial.extend_from_slice(&payload);
                    if fin {
                        let raw = String::from_utf8_lossy(&partial).to_string();
                        partial.clear();
                        let event = parse_event(&raw);
                        if event.seq > 0 {
                            self.queue
                                .lock()
                                .unwrap()
                                .push(event.clone())
                                .map_err(|error| error.to_string())?;
                            (self.handler)(&event)?;
                            self.queue
                                .lock()
                                .unwrap()
                                .ack(event.seq)
                                .map_err(|error| error.to_string())?;
                            let ack = format!("{{\"type\":\"ack\",\"seq\":{}}}", event.seq);
                            write_frame(stream, 0x1, ack.as_bytes()).map_err(|error| error.to_string())?;
                        }
                    }
                }
                0x9 => {
                    write_frame(stream, 0xA, &payload).map_err(|error| error.to_string())?;
                }
                0x8 => return Ok(()),
                _ => {}
            }
        }
    }

    fn report(&self, error: &str) {
        if let Some(on_error) = &self.on_error {
            on_error(error);
        }
    }
}

fn endpoint_path(client: &Client, base: &str) -> String {
    let mut path = format!("{}/api/v1/events/ws?stream={}", base.trim_end_matches('/'), pct(&client.cfg.stream_id));
    let last = client.last_seq();
    if last > 0 {
        path.push_str(&format!("&after_seq={}", last));
    }
    if !client.cfg.types.is_empty() {
        path.push_str("&types=");
        path.push_str(&pct(&client.cfg.types.join(",")));
    }
    if !client.cfg.correlation_key.is_empty() {
        path.push_str("&correlation_key=");
        path.push_str(&pct(&client.cfg.correlation_key));
    }
    if !client.cfg.predicate.is_empty() {
        path.push_str("&predicate=");
        path.push_str(&pct(&client.cfg.predicate));
    }
    path
}

fn parse_url(url: &str) -> Result<(String, u16, String), String> {
    let rest = url
        .strip_prefix("ws://")
        .or_else(|| url.strip_prefix("http://"))
        .ok_or_else(|| "events: only ws:// or http:// is supported".to_string())?;
    let (authority, path) = match rest.split_once('/') {
        Some((authority, tail)) => (authority, format!("/{}", tail)),
        None => (rest, "/".to_string()),
    };
    let (host, port) = match authority.rsplit_once(':') {
        Some((host, port)) => (host.to_string(), port.parse().map_err(|_| "events: bad port".to_string())?),
        None => (authority.to_string(), 80),
    };
    if host.is_empty() {
        return Err("events: bad host".to_string());
    }
    Ok((host, port, path))
}

fn read_handshake(stream: &mut TcpStream) -> Result<(), String> {
    let mut buf = Vec::new();
    let mut byte = [0u8; 1];
    while !buf.ends_with(b"\r\n\r\n") {
        match stream.read(&mut byte) {
            Ok(0) => return Err("events: handshake closed".to_string()),
            Ok(_) => buf.push(byte[0]),
            Err(error) => return Err(error.to_string()),
        }
        if buf.len() > 16 * 1024 {
            return Err("events: handshake too large".to_string());
        }
    }
    let text = String::from_utf8_lossy(&buf);
    if !text.starts_with("HTTP/1.1 101") && !text.starts_with("HTTP/1.0 101") {
        return Err(format!("events: handshake rejected: {}", text.lines().next().unwrap_or("")));
    }
    Ok(())
}

fn read_frame(stream: &mut TcpStream) -> io::Result<(bool, u8, Vec<u8>)> {
    let mut header = [0u8; 2];
    stream.read_exact(&mut header)?;
    let fin = header[0] & 0x80 != 0;
    let opcode = header[0] & 0x0f;
    let masked = header[1] & 0x80 != 0;
    let mut len = (header[1] & 0x7f) as usize;
    if len == 126 {
        let mut ext = [0u8; 2];
        stream.read_exact(&mut ext)?;
        len = u16::from_be_bytes(ext) as usize;
    } else if len == 127 {
        let mut ext = [0u8; 8];
        stream.read_exact(&mut ext)?;
        len = u64::from_be_bytes(ext) as usize;
    }
    let mask = if masked {
        let mut mask = [0u8; 4];
        stream.read_exact(&mut mask)?;
        Some(mask)
    } else {
        None
    };
    let mut payload = vec![0u8; len];
    stream.read_exact(&mut payload)?;
    if let Some(mask) = mask {
        for (index, byte) in payload.iter_mut().enumerate() {
            *byte ^= mask[index % 4];
        }
    }
    Ok((fin, opcode, payload))
}

fn write_frame(stream: &mut TcpStream, opcode: u8, payload: &[u8]) -> io::Result<()> {
    let mut header = vec![0x80 | opcode];
    let mask = ws_mask();
    if payload.len() < 126 {
        header.push(0x80 | payload.len() as u8);
    } else if payload.len() <= u16::MAX as usize {
        header.push(0x80 | 126);
        header.extend_from_slice(&(payload.len() as u16).to_be_bytes());
    } else {
        header.push(0x80 | 127);
        header.extend_from_slice(&(payload.len() as u64).to_be_bytes());
    }
    header.extend_from_slice(&mask);
    stream.write_all(&header)?;
    let mut body = payload.to_vec();
    for (index, byte) in body.iter_mut().enumerate() {
        *byte ^= mask[index % 4];
    }
    stream.write_all(&body)
}

fn sleep(duration: Duration) {
    std::thread::sleep(duration);
}

fn now_nanos() -> u64 {
    SystemTime::now().duration_since(UNIX_EPOCH).unwrap_or_default().as_nanos() as u64
}

fn ws_key() -> String {
    base64(&now_nanos().to_be_bytes().repeat(2)[..16])
}

fn ws_mask() -> [u8; 4] {
    let value = now_nanos() ^ (std::process::id() as u64) << 32;
    value.to_be_bytes()[4..8].try_into().unwrap()
}

fn pct(value: &str) -> String {
    let mut out = String::new();
    for byte in value.bytes() {
        if byte.is_ascii_alphanumeric() || matches!(byte, b'-' | b'_' | b'.' | b'~') {
            out.push(byte as char);
        } else {
            out.push_str(&format!("%{:02X}", byte));
        }
    }
    out
}

fn base64(data: &[u8]) -> String {
    const TABLE: &[u8; 64] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    let mut out = String::new();
    for chunk in data.chunks(3) {
        let b = [chunk[0], *chunk.get(1).unwrap_or(&0), *chunk.get(2).unwrap_or(&0)];
        out.push(TABLE[(b[0] >> 2) as usize] as char);
        out.push(TABLE[(((b[0] & 3) << 4) | (b[1] >> 4)) as usize] as char);
        if chunk.len() > 1 {
            out.push(TABLE[(((b[1] & 0xf) << 2) | (b[2] >> 6)) as usize] as char);
        } else {
            out.push('=');
        }
        if chunk.len() > 2 {
            out.push(TABLE[(b[2] & 0x3f) as usize] as char);
        } else {
            out.push('=');
        }
    }
    out
}

#[cfg(test)]
fn sha1(data: &[u8]) -> [u8; 20] {
    let mut h: [u32; 5] = [0x67452301, 0xEFCDAB89, 0x98BADCFE, 0x10325476, 0xC3D2E1F0];
    let bit_len = (data.len() as u64) * 8;
    let mut message = data.to_vec();
    message.push(0x80);
    while message.len() % 64 != 56 {
        message.push(0);
    }
    message.extend_from_slice(&bit_len.to_be_bytes());
    for chunk in message.chunks(64) {
        let mut w = [0u32; 80];
        for index in 0..16 {
            w[index] = u32::from_be_bytes([
                chunk[index * 4],
                chunk[index * 4 + 1],
                chunk[index * 4 + 2],
                chunk[index * 4 + 3],
            ]);
        }
        for index in 16..80 {
            w[index] = (w[index - 3] ^ w[index - 8] ^ w[index - 14] ^ w[index - 16]).rotate_left(1);
        }
        let (mut a, mut b, mut c, mut d, mut e) = (h[0], h[1], h[2], h[3], h[4]);
        for (index, word) in w.iter().enumerate() {
            let (f, k) = match index {
                0..=19 => ((b & c) | ((!b) & d), 0x5A827999),
                20..=39 => (b ^ c ^ d, 0x6ED9EBA1),
                40..=59 => ((b & c) | (b & d) | (c & d), 0x8F1BBCDC),
                _ => (b ^ c ^ d, 0xCA62C1D6),
            };
            let temp = a
                .rotate_left(5)
                .wrapping_add(f)
                .wrapping_add(e)
                .wrapping_add(k)
                .wrapping_add(*word);
            e = d;
            d = c;
            c = b.rotate_left(30);
            b = a;
            a = temp;
        }
        h[0] = h[0].wrapping_add(a);
        h[1] = h[1].wrapping_add(b);
        h[2] = h[2].wrapping_add(c);
        h[3] = h[3].wrapping_add(d);
        h[4] = h[4].wrapping_add(e);
    }
    let mut out = [0u8; 20];
    for (index, value) in h.iter().enumerate() {
        out[index * 4..index * 4 + 4].copy_from_slice(&value.to_be_bytes());
    }
    out
}

pub fn parse_event(raw: &str) -> Event {
    Event {
        seq: raw_value(raw, "seq").and_then(|v| v.parse().ok()).unwrap_or(0),
        type_: parse_string(raw, "type"),
        actor_id: parse_string(raw, "actor_id"),
        payload: raw_value(raw, "payload"),
        wall_time: parse_string(raw, "wall_time"),
        raw: raw.to_string(),
    }
}

fn parse_string(raw: &str, key: &str) -> String {
    match raw_value(raw, key) {
        Some(value) if value.starts_with('"') && value.ends_with('"') => unescape(&value[1..value.len() - 1]),
        _ => String::new(),
    }
}

fn raw_value(raw: &str, key: &str) -> Option<String> {
    let needle = format!("\"{}\"", key);
    let at = raw.find(&needle)?;
    let colon = raw[at + needle.len()..].find(':')? + at + needle.len();
    let bytes = raw.as_bytes();
    let mut index = colon + 1;
    while index < bytes.len() && (bytes[index] as char).is_whitespace() {
        index += 1;
    }
    if index >= bytes.len() {
        return None;
    }
    let first = bytes[index] as char;
    if first == '"' {
        let mut end = index + 1;
        while end < bytes.len() {
            if bytes[end] == b'\\' {
                end += 2;
                continue;
            }
            if bytes[end] == b'"' {
                return Some(raw[index..=end].to_string());
            }
            end += 1;
        }
        return None;
    }
    if first == '{' || first == '[' {
        let mut depth = 0usize;
        let mut end = index;
        while end < bytes.len() {
            match bytes[end] as char {
                '"' => {
                    end += 1;
                    while end < bytes.len() {
                        if bytes[end] == b'\\' {
                            end += 2;
                            continue;
                        }
                        if bytes[end] == b'"' {
                            break;
                        }
                        end += 1;
                    }
                }
                '{' | '[' => depth += 1,
                '}' | ']' => {
                    depth -= 1;
                    if depth == 0 {
                        return Some(raw[index..=end].to_string());
                    }
                }
                _ => {}
            }
            end += 1;
        }
        return None;
    }
    let end = raw[index..].find(|c| c == ',' || c == '}').map(|i| index + i).unwrap_or(bytes.len());
    Some(raw[index..end].trim().to_string())
}

fn unescape(value: &str) -> String {
    let mut out = String::new();
    let mut chars = value.chars();
    while let Some(c) = chars.next() {
        if c != '\\' {
            out.push(c);
            continue;
        }
        match chars.next() {
            Some('n') => out.push('\n'),
            Some('r') => out.push('\r'),
            Some('t') => out.push('\t'),
            Some('b') => out.push('\u{8}'),
            Some('f') => out.push('\u{c}'),
            Some('"') => out.push('"'),
            Some('\\') => out.push('\\'),
            Some('/') => out.push('/'),
            Some('u') => {
                let hex: String = chars.by_ref().take(4).collect();
                if let Ok(code) = u32::from_str_radix(&hex, 16) {
                    if let Some(ch) = char::from_u32(code) {
                        out.push(ch);
                    }
                }
            }
            Some(other) => out.push(other),
            None => {}
        }
    }
    out
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Read;
    use std::net::TcpListener;
    use std::sync::atomic::AtomicU64;
    use std::sync::mpsc;
    use std::sync::Arc;
    use std::thread;

    const GUID: &str = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11";

    fn unique_dir(name: &str) -> PathBuf {
        let dir = std::env::temp_dir().join(format!("rheovela-{}-{}", name, now_nanos()));
        let _ = fs::remove_dir_all(&dir);
        dir
    }

    fn read_until(stream: &mut TcpStream, marker: &[u8]) -> Vec<u8> {
        let mut buf = Vec::new();
        let mut byte = [0u8; 1];
        while !buf.ends_with(marker) {
            if stream.read(&mut byte).unwrap_or(0) == 0 {
                break;
            }
            buf.push(byte[0]);
        }
        buf
    }

    fn server_frame(stream: &mut TcpStream, opcode: u8, text: &str) {
        let payload = text.as_bytes();
        stream.write_all(&[0x80 | opcode, payload.len() as u8]).unwrap();
        stream.write_all(payload).unwrap();
        stream.flush().unwrap();
    }

    fn start_server<F>(connections: usize, handler: F) -> (String, thread::JoinHandle<()>)
    where
        F: Fn(usize, String, TcpStream) + Send + Sync + 'static,
    {
        let listener = TcpListener::bind("127.0.0.1:0").unwrap();
        let addr = listener.local_addr().unwrap();
        let handle = thread::spawn(move || {
            for index in 1..=connections {
                let (mut stream, _) = listener.accept().unwrap();
                let request = read_until(&mut stream, b"\r\n\r\n");
                let text = String::from_utf8_lossy(&request);
                let target = text.lines().next().unwrap_or("").split(' ').nth(1).unwrap_or("/");
                let after = target
                    .split_once('?')
                    .map(|(_, query)| {
                        query
                            .split('&')
                            .find_map(|pair| pair.strip_prefix("after_seq="))
                            .unwrap_or("")
                            .to_string()
                    })
                    .unwrap_or_default();
                let key = text
                    .lines()
                    .find_map(|line| line.strip_prefix("Sec-WebSocket-Key:"))
                    .unwrap_or("")
                    .trim();
                let accept = base64(&sha1(format!("{}{}", key, GUID).as_bytes()));
                let response = format!(
                    "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: {}\r\n\r\n",
                    accept
                );
                stream.write_all(response.as_bytes()).unwrap();
                handler(index, after, stream);
            }
        });
        (format!("ws://{}", addr), handle)
    }

    #[test]
    fn queue_persists_and_acknowledges() {
        let dir = unique_dir("queue");
        let mut queue = DurableQueue::open(&dir).unwrap();
        queue.push(parse_event(r#"{"seq":2,"type":"b"}"#)).unwrap();
        queue.push(parse_event(r#"{"seq":1,"type":"a"}"#)).unwrap();
        assert_eq!(queue.pending().len(), 2);
        assert_eq!(queue.pending()[0].seq, 1);
        queue.ack(1).unwrap();
        let reopened = DurableQueue::open(&dir).unwrap();
        assert_eq!(reopened.last_seq(), 1);
        assert_eq!(reopened.pending().len(), 1);
        assert_eq!(reopened.pending()[0].seq, 2);
        let _ = fs::remove_dir_all(dir);
    }

    #[test]
    fn client_receives_and_persists() {
        let dir = unique_dir("receive");
        let (url, handle) = start_server(1, |_, _, mut stream| {
            server_frame(&mut stream, 0x1, r#"{"seq":1,"type":"a"}"#);
            server_frame(&mut stream, 0x1, r#"{"seq":2,"type":"b"}"#);
            let mut buf = [0u8; 1];
            while stream.read(&mut buf).unwrap_or(0) > 0 {}
        });
        let (tx, rx) = mpsc::channel();
        let client = Arc::new(
            Client::new(
                {
                    let mut cfg = Config::new(url, "S", dir);
                    cfg.retry_delay = Duration::from_millis(10);
                    cfg.reconnect_min = Duration::from_millis(10);
                    cfg
                },
                Box::new(move |event| {
                    tx.send(event.seq).unwrap();
                    Ok(())
                }),
                None,
            )
            .unwrap(),
        );
        let runner = {
            let client = Arc::clone(&client);
            thread::spawn(move || client.run())
        };
        assert_eq!(rx.recv_timeout(Duration::from_secs(3)).unwrap(), 1);
        assert_eq!(rx.recv_timeout(Duration::from_secs(3)).unwrap(), 2);
        client.stop();
        runner.join().unwrap();
        handle.join().unwrap();
        assert_eq!(client.last_seq(), 2);
        assert!(client.pending().is_empty());
    }

    #[test]
    fn client_reconnects_with_last_seq() {
        let dir = unique_dir("reconnect");
        let after_seqs = Arc::new(Mutex::new(Vec::new()));
        let seen = Arc::clone(&after_seqs);
        let (url, handle) = start_server(2, move |index, after, mut stream| {
            seen.lock().unwrap().push(after);
            if index == 1 {
                server_frame(&mut stream, 0x1, r#"{"seq":1,"type":"a"}"#);
                server_frame(&mut stream, 0x8, "");
                return;
            }
            server_frame(&mut stream, 0x1, r#"{"seq":2,"type":"b"}"#);
            let mut buf = [0u8; 1];
            while stream.read(&mut buf).unwrap_or(0) > 0 {}
        });
        let (tx, rx) = mpsc::channel();
        let client = Arc::new(
            Client::new(
                {
                    let mut cfg = Config::new(url, "S", dir);
                    cfg.retry_delay = Duration::from_millis(10);
                    cfg.reconnect_min = Duration::from_millis(10);
                    cfg
                },
                Box::new(move |event| {
                    tx.send(event.seq).unwrap();
                    Ok(())
                }),
                None,
            )
            .unwrap(),
        );
        let runner = {
            let client = Arc::clone(&client);
            thread::spawn(move || client.run())
        };
        assert_eq!(rx.recv_timeout(Duration::from_secs(3)).unwrap(), 1);
        assert_eq!(rx.recv_timeout(Duration::from_secs(3)).unwrap(), 2);
        client.stop();
        runner.join().unwrap();
        handle.join().unwrap();
        assert_eq!(&after_seqs.lock().unwrap()[..2], &["".to_string(), "1".to_string()]);
    }

    #[test]
    fn client_retries_handler_failures() {
        let dir = unique_dir("retry");
        let (url, handle) = start_server(1, |_, _, mut stream| {
            server_frame(&mut stream, 0x1, r#"{"seq":1,"type":"a"}"#);
            let mut buf = [0u8; 1];
            while stream.read(&mut buf).unwrap_or(0) > 0 {}
        });
        let calls = Arc::new(AtomicU64::new(0));
        let errors = Arc::new(AtomicU64::new(0));
        let call_count = Arc::clone(&calls);
        let error_count = Arc::clone(&errors);
        let client = Arc::new(
            Client::new(
                {
                    let mut cfg = Config::new(url, "S", dir);
                    cfg.retry_delay = Duration::from_millis(10);
                    cfg.reconnect_min = Duration::from_millis(10);
                    cfg
                },
                Box::new(move |_| {
                    if call_count.fetch_add(1, Ordering::SeqCst) == 0 {
                        return Err("transient".to_string());
                    }
                    Ok(())
                }),
                Some(Box::new(move |_| {
                    error_count.fetch_add(1, Ordering::SeqCst);
                })),
            )
            .unwrap(),
        );
        let runner = {
            let client = Arc::clone(&client);
            thread::spawn(move || client.run())
        };
        let deadline = SystemTime::now() + Duration::from_secs(3);
        while SystemTime::now() < deadline {
            if calls.load(Ordering::SeqCst) >= 2 && client.last_seq() == 1 && client.pending().is_empty() {
                break;
            }
            thread::sleep(Duration::from_millis(10));
        }
        client.stop();
        runner.join().unwrap();
        handle.join().unwrap();
        assert!(calls.load(Ordering::SeqCst) >= 2);
        assert!(errors.load(Ordering::SeqCst) > 0);
        assert_eq!(client.last_seq(), 1);
    }
}
