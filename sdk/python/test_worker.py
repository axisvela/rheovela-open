"""Tests for the Python worker SDK against a fake Worker HTTP server."""

import json
import threading
import unittest
import urllib.parse
from http.server import BaseHTTPRequestHandler, HTTPServer

import worker


FULL_ITEM = {
    "id": "wi1",
    "instance_id": "i1",
    "activity_id": "a1",
    "state": "ready",
    "result": {"ok": True},
    "checkpoint_ref": "ckpt-1",
    "effect_cursor": "cur-1",
    "attempts": 2,
    "max_attempts": 5,
    "last_error": "boom",
    "approval": {
        "state": "pending",
        "pending_stage": "review",
        "final_args": {"x": 1},
        "reason": "needs review",
        "requested_by": "agent",
        "requested_at": "2026-10-14T00:00:00Z",
        "decided_by": "human",
        "decided_at": "2026-10-14T01:00:00Z",
        "comment": "looks good",
        "checkpoint_ref": "ckpt-1",
        "effect_cursor": "cur-1",
    },
}


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
        path = urllib.parse.urlsplit(self.path).path
        srv.records.append(("GET", self.path, None))
        if path == "/api/v1/work-items":
            self._send(200, srv.poll_items)
        elif path.startswith("/api/v1/work-items/"):
            item_id = path.split("/")[4]
            item = srv.work_items.get(item_id)
            if item is None:
                self._send(404, {"error": "not found"})
            else:
                self._send(200, item)
        else:
            self._send(404, {"error": "not found"})

    def do_POST(self):
        srv = self.server
        body = self._read_body()
        srv.records.append(("POST", self.path, body))
        parts = urllib.parse.urlsplit(self.path).path.split("/")
        item_id = parts[4]
        action = parts[5]
        if action == "claim":
            if item_id in srv.conflict_ids:
                self._send(409, {"error": "conflict"})
                return
            self._send(200, {"token": srv.token, "lease": body.get("lease", "30s")})
        elif action == "complete":
            srv.complete_records.append(body)
            self._send(200, {"ok": True})
        elif action == "request-approval":
            srv.approval_records.append(body)
            self._send(200, {"ok": True})
        elif action == "approve":
            srv.approve_records.append(body)
            self._send(200, {"ok": True})
        elif action == "reject":
            srv.reject_records.append(body)
            self._send(200, {"ok": True})
        elif action == "cancel":
            srv.cancel_records.append(body)
            self._send(200, {"ok": True})
        elif action == "fail":
            srv.fail_records.append(body)
            self._send(200, {"ok": True})
        elif action == "heartbeat":
            self._send(200, {"ok": True})
        else:
            self._send(404, {"error": "not found"})


class FakeServer(HTTPServer):
    def __init__(self):
        super().__init__(("127.0.0.1", 0), Handler)
        self.poll_items = []
        self.work_items = {}
        self.conflict_ids = set()
        self.token = "tok-1"
        self.records = []
        self.complete_records = []
        self.approval_records = []
        self.approve_records = []
        self.reject_records = []
        self.cancel_records = []
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
            return worker.WorkResult.success()

        n = worker.Worker(client, fn).process_once()

        self.assertEqual(n, 1)
        self.assertEqual([item.id for item in calls], ["wi1"])
        posts = [r for r in self.srv.records if r[0] == "POST"]
        self.assertEqual([r[1] for r in posts],
                         ["/api/v1/work-items/wi1/claim",
                          "/api/v1/work-items/wi1/complete"])
        self.assertEqual(self.srv.complete_records, [{"token": "tok-1"}])

    def test_complete_sends_evidence_and_cost(self):
        self.srv.poll_items = [{"id": "wi1"}]
        client = worker.Client(self.srv.base_url)
        n = worker.Worker(
            client,
            lambda item: worker.WorkResult.success(
                result={"ok": True}, evidence={"receipt": "r-1"}, cost=12
            ),
            claim_options={"actor": "alice", "tools": ["erp"],
                           "credential": "cred-1"},
        ).process_once()

        self.assertEqual(n, 1)
        self.assertEqual(self.srv.complete_records, [{
            "token": "tok-1",
            "actor": "alice",
            "result": {"ok": True},
            "evidence": {"receipt": "r-1"},
            "cost": 12,
        }])

    def test_claim_conflict_skipped(self):
        self.srv.poll_items = [{"id": "wi1"}]
        self.srv.conflict_ids = {"wi1"}
        client = worker.Client(self.srv.base_url)
        n = worker.Worker(
            client, lambda item: worker.WorkResult.success()
        ).process_once()

        self.assertEqual(n, 0)
        posts = [r for r in self.srv.records if r[0] == "POST"]
        self.assertEqual([r[1] for r in posts], ["/api/v1/work-items/wi1/claim"])

    def test_fail_records_error(self):
        self.srv.poll_items = [{"id": "wi1"}]
        client = worker.Client(self.srv.base_url)
        n = worker.Worker(
            client, lambda item: worker.WorkResult.failure("boom")
        ).process_once()

        self.assertEqual(n, 1)
        self.assertEqual(self.srv.fail_records,
                         [{"token": "tok-1", "error": "boom"}])

    def test_worker_success_sends_result_and_cursors(self):
        self.srv.poll_items = [{"id": "wi1"}]
        client = worker.Client(self.srv.base_url)

        def fn(item):
            return worker.WorkResult.success(
                result={"ok": True},
                checkpoint_ref="ckpt-1",
                effect_cursor="cur-1",
            )

        n = worker.Worker(client, fn).process_once()

        self.assertEqual(n, 1)
        self.assertEqual(self.srv.complete_records, [{
            "token": "tok-1",
            "result": {"ok": True},
            "checkpoint_ref": "ckpt-1",
            "effect_cursor": "cur-1",
        }])

    def test_complete_with_result_and_cursors(self):
        client = worker.Client(self.srv.base_url)

        client.complete("wi1", "tok-1", result={"ok": True},
                        checkpoint_ref="ckpt-1", effect_cursor="cur-1")

        self.assertEqual(self.srv.complete_records, [{
            "token": "tok-1",
            "result": {"ok": True},
            "checkpoint_ref": "ckpt-1",
            "effect_cursor": "cur-1",
        }])

    def test_get_work_item_parses_fields(self):
        self.srv.work_items["wi1"] = FULL_ITEM
        client = worker.Client(self.srv.base_url)

        item = client.get_work_item("wi1")

        self.assertIsInstance(item, worker.WorkItem)
        self.assertEqual(item.id, "wi1")
        self.assertEqual(item.instance_id, "i1")
        self.assertEqual(item.activity_id, "a1")
        self.assertEqual(item.state, "ready")
        self.assertEqual(item.result, {"ok": True})
        self.assertEqual(item.checkpoint_ref, "ckpt-1")
        self.assertEqual(item.effect_cursor, "cur-1")
        self.assertEqual(item.attempts, 2)
        self.assertEqual(item.max_attempts, 5)
        self.assertEqual(item.last_error, "boom")
        self.assertIsNotNone(item.approval)
        self.assertEqual(item.approval.state, "pending")
        self.assertEqual(item.approval.pending_stage, "review")
        self.assertEqual(item.approval.final_args, {"x": 1})
        self.assertEqual(item.approval.reason, "needs review")
        self.assertEqual(item.approval.requested_by, "agent")
        self.assertEqual(item.approval.requested_at, "2026-10-14T00:00:00Z")
        self.assertEqual(item.approval.decided_by, "human")
        self.assertEqual(item.approval.decided_at, "2026-10-14T01:00:00Z")
        self.assertEqual(item.approval.comment, "looks good")
        self.assertEqual(item.approval.checkpoint_ref, "ckpt-1")
        self.assertEqual(item.approval.effect_cursor, "cur-1")

    def test_request_approval_body(self):
        client = worker.Client(self.srv.base_url)
        req = worker.ApprovalRequest(
            actor="agent",
            reason="over limit",
            final_args={"amount": 10},
            checkpoint_ref="ckpt-1",
            effect_cursor="cur-1",
        )

        client.request_approval("wi1", "tok-1", req)

        self.assertEqual(self.srv.approval_records, [{
            "token": "tok-1",
            "actor": "agent",
            "reason": "over limit",
            "final_args": {"amount": 10},
            "checkpoint_ref": "ckpt-1",
            "effect_cursor": "cur-1",
        }])

    def test_approve_reject_cancel(self):
        client = worker.Client(self.srv.base_url)

        client.approve("wi1", "human", comment="ok",
                       final_args={"approved": True})
        client.reject("wi2", "human", "no")
        client.cancel("wi3")

        posts = [r for r in self.srv.records if r[0] == "POST"]
        self.assertEqual([r[1] for r in posts], [
            "/api/v1/work-items/wi1/approve",
            "/api/v1/work-items/wi2/reject",
            "/api/v1/work-items/wi3/cancel",
        ])
        self.assertEqual(self.srv.approve_records, [{
            "actor": "human",
            "comment": "ok",
            "final_args": {"approved": True},
        }])
        self.assertEqual(self.srv.reject_records,
                         [{"actor": "human", "reason": "no"}])
        self.assertEqual(self.srv.cancel_records, [{}])

    def test_global_poll_returns_claimable_items(self):
        self.srv.poll_items = [
            FULL_ITEM,
            {"id": "wi2", "instance_id": "i2", "activity_id": "a2",
             "state": "claimed"},
        ]
        client = worker.Client(self.srv.base_url)

        items = client.poll()

        self.assertIsInstance(items[0], worker.WorkItem)
        self.assertEqual([item.id for item in items], ["wi1", "wi2"])
        self.assertEqual([item.state for item in items], ["ready", "claimed"])
        self.assertEqual(items[0].result, {"ok": True})
        self.assertIsNotNone(items[0].approval)
        self.assertEqual(items[0].approval.state, "pending")
        self.assertIsNone(items[1].approval)
        gets = [r for r in self.srv.records if r[0] == "GET"]
        self.assertEqual(gets[0][1], "/api/v1/work-items")

    def test_poll_filters_by_instance(self):
        client = worker.Client(self.srv.base_url)

        self.assertEqual(client.poll("inst 1"), [])

        gets = [r for r in self.srv.records if r[0] == "GET"]
        self.assertEqual(gets[0][1], "/api/v1/work-items?instance=inst%201")

    def test_worker_approval_path_merges_cursors(self):
        self.srv.poll_items = [{"id": "wi1"}]
        client = worker.Client(self.srv.base_url)
        req = worker.ApprovalRequest(
            actor="agent", reason="over limit", final_args={"amount": 10}
        )

        def fn(item):
            return worker.WorkResult(
                status="approval",
                approval=req,
                checkpoint_ref="ckpt-1",
                effect_cursor="cur-1",
            )

        n = worker.Worker(client, fn).process_once()

        self.assertEqual(n, 1)
        posts = [r for r in self.srv.records if r[0] == "POST"]
        self.assertEqual([r[1] for r in posts], [
            "/api/v1/work-items/wi1/claim",
            "/api/v1/work-items/wi1/request-approval",
        ])
        self.assertEqual(self.srv.approval_records, [{
            "token": "tok-1",
            "actor": "agent",
            "reason": "over limit",
            "final_args": {"amount": 10},
            "checkpoint_ref": "ckpt-1",
            "effect_cursor": "cur-1",
        }])
        self.assertEqual(req.checkpoint_ref, "")
        self.assertEqual(req.effect_cursor, "")

    def test_worker_approval_without_request_fails(self):
        self.srv.poll_items = [{"id": "wi1"}]
        client = worker.Client(self.srv.base_url)

        n = worker.Worker(
            client,
            lambda item: worker.WorkResult(status="approval"),
        ).process_once()

        self.assertEqual(n, 1)
        self.assertEqual(self.srv.fail_records, [{
            "token": "tok-1",
            "error": "approval result is missing approval request",
        }])


if __name__ == "__main__":
    unittest.main()
