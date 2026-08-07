"""Tests for the Python worker SDK against a fake Worker HTTP server."""

import json
import threading
import unittest
from http.server import BaseHTTPRequestHandler, HTTPServer

import worker


class Handler(BaseHTTPRequestHandler):
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
            self._send(200, srv.poll_items)
        else:
            self._send(404, {"error": "not found"})

    def do_POST(self):
        srv = self.server
        body = self._read_body()
        srv.records.append(("POST", self.path, body))
        if self.path.endswith("/claim"):
            item_id = self.path.split("/")[4]
            if item_id in srv.conflict_ids:
                self._send(409, {"error": "conflict"})
                return
            self._send(200, {"token": srv.token, "lease": body.get("lease", "30s")})
        elif self.path.endswith("/complete"):
            self._send(200, {"ok": True})
        elif self.path.endswith("/fail"):
            srv.fail_records.append(body)
            self._send(200, {"ok": True})
        elif self.path.endswith("/heartbeat"):
            self._send(200, {"ok": True})
        else:
            self._send(404, {"error": "not found"})


class FakeServer(HTTPServer):
    def __init__(self):
        super().__init__(("127.0.0.1", 0), Handler)
        self.poll_items = []
        self.conflict_ids = set()
        self.token = "tok-1"
        self.records = []
        self.fail_records = []
        self.thread = threading.Thread(target=self.serve_forever, daemon=True)
        self.thread.start()

    @property
    def base_url(self):
        host, port = self.server_address[:2]
        return f"http://{host}:{port}"


class WorkerTest(unittest.TestCase):
    def setUp(self):
        self.srv = FakeServer()

    def tearDown(self):
        self.srv.shutdown()
        self.srv.server_close()
        self.srv.thread.join(timeout=5)

    def test_claim_then_complete(self):
        self.srv.poll_items = [{"id": "wi1"}]
        client = worker.Client(self.srv.base_url)
        calls = []

        def fn(item):
            calls.append(item)
            return ("success", None)

        n = worker.Worker(client, fn).process_once()

        self.assertEqual(n, 1)
        self.assertEqual(calls, [{"id": "wi1"}])
        posts = [r for r in self.srv.records if r[0] == "POST"]
        self.assertEqual([r[1] for r in posts],
                         ["/api/v1/work-items/wi1/claim",
                          "/api/v1/work-items/wi1/complete"])
        self.assertEqual(posts[1][2], {"token": "tok-1"})

    def test_claim_conflict_skipped(self):
        self.srv.poll_items = [{"id": "wi1"}]
        self.srv.conflict_ids = {"wi1"}
        client = worker.Client(self.srv.base_url)
        n = worker.Worker(client, lambda item: ("success", None)).process_once()

        self.assertEqual(n, 0)
        posts = [r for r in self.srv.records if r[0] == "POST"]
        self.assertEqual([r[1] for r in posts], ["/api/v1/work-items/wi1/claim"])

    def test_fail_records_error(self):
        self.srv.poll_items = [{"id": "wi1"}]
        client = worker.Client(self.srv.base_url)
        n = worker.Worker(client, lambda item: ("failure", "boom")).process_once()

        self.assertEqual(n, 1)
        self.assertEqual(self.srv.fail_records, [{"token": "tok-1", "error": "boom"}])


if __name__ == "__main__":
    unittest.main()
