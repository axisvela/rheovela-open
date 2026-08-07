"""Runnable Python worker example (stdlib only).

Starts a local fake Worker-API server (http.server) backed by an in-memory
dict, then processes its work items through the SDK Worker from
sdk/python/worker.py. task-1 succeeds, task-2 fails.

Run:
    python examples/python-worker/main.py
"""

import json
import os
import sys
import threading
from http.server import BaseHTTPRequestHandler, HTTPServer

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "..", "sdk", "python"))
from worker import Client, Worker, WorkerError


class FakeWorkerServer(HTTPServer):
    """In-memory Worker-API server (adapted from sdk/python/test_worker.py)."""

    def __init__(self):
        super().__init__(("127.0.0.1", 0), _Handler)
        self.items = {
            "task-1": {"id": "task-1", "instance_id": "inst-task-1",
                       "activity_id": "act", "state": "ready"},
            "task-2": {"id": "task-2", "instance_id": "inst-task-2",
                       "activity_id": "act", "state": "ready"},
        }
        self.tokens = {}
        self.records = []
        self.fail_records = []
        self.thread = threading.Thread(target=self.serve_forever, daemon=True)
        self.thread.start()

    @property
    def base_url(self):
        host, port = self.server_address[:2]
        return f"http://{host}:{port}"

    def close(self):
        self.shutdown()
        self.server_close()
        self.thread.join(timeout=5)


class _Handler(BaseHTTPRequestHandler):
    def log_message(self, *args):
        pass

    def _send(self, code, obj):
        body = json.dumps(obj).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def _read_body(self):
        length = int(self.headers.get("Content-Length", 0))
        if not length:
            return {}
        return json.loads(self.rfile.read(length).decode())

    def do_GET(self):
        srv = self.server
        if self.path == "/api/v1/work-items":
            self._send(200, list(srv.items.values()))
        else:
            self._send(404, {"error": "not found"})

    def do_POST(self):
        srv = self.server
        body = self._read_body()
        srv.records.append(("POST", self.path, body))
        item_id = self.path.split("/")[4]
        if self.path.endswith("/claim"):
            item = srv.items.get(item_id)
            if not item or item["state"] != "ready":
                self._send(409, {"error": "conflict"})
                return
            token = "tok-" + item_id
            srv.tokens[item_id] = token
            self._send(200, {"token": token, "lease": body.get("lease", "30s")})
        elif self.path.endswith("/complete"):
            item = srv.items.get(item_id)
            if not item or srv.tokens.get(item_id) != body.get("token"):
                self._send(409, {"error": "stale token"})
                return
            item["state"] = "done"
            self._send(200, {"ok": True})
        elif self.path.endswith("/fail"):
            item = srv.items.get(item_id)
            if not item or srv.tokens.get(item_id) != body.get("token"):
                self._send(409, {"error": "stale token"})
                return
            item["state"] = "failed"
            item["error"] = body.get("error", "failure")
            srv.fail_records.append(body)
            self._send(200, {"ok": True})
        elif self.path.endswith("/heartbeat"):
            self._send(200, {"ok": True})
        else:
            self._send(404, {"error": "not found"})


def run_example():
    srv = FakeWorkerServer()
    try:
        client = Client(srv.base_url)
        fn = lambda item: ("success", None) if item["id"] == "task-1" \
            else ("failure", "simulated failure")
        worker = Worker(client, fn, "30s")
        processed = worker.process_once()
        states = {i: srv.items[i]["state"] for i in srv.items}
        return processed, states
    finally:
        srv.close()


def main():
    try:
        processed, states = run_example()
    except WorkerError as e:
        print(f"error: {e}", file=sys.stderr)
        return 1
    print(f"processed={processed}")
    for item_id in sorted(states):
        print(f"  {item_id}: {states[item_id]}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
