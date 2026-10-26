"""Rheovela Python worker SDK (stdlib only).

Uses the core Worker HTTP API (rheo serve). Minimal, dependency-free.
"""
import json
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import dataclass, replace
from typing import Any, List, Optional


class WorkerError(Exception):
    pass


def _nonempty(value):
    return value is not None and value != ""


def _escape(value):
    return urllib.parse.quote(str(value), safe="")


@dataclass
class Approval:
    """Approval state attached to a work item."""

    state: str = ""
    pending_stage: str = ""
    final_args: Any = None
    reason: str = ""
    requested_by: str = ""
    requested_at: str = ""
    decided_by: str = ""
    decided_at: str = ""
    comment: str = ""
    checkpoint_ref: str = ""
    effect_cursor: str = ""

    @classmethod
    def from_dict(cls, data):
        if not data:
            return None
        return cls(
            state=data.get("state", ""),
            pending_stage=data.get("pending_stage", ""),
            final_args=data.get("final_args"),
            reason=data.get("reason", ""),
            requested_by=data.get("requested_by", ""),
            requested_at=data.get("requested_at", ""),
            decided_by=data.get("decided_by", ""),
            decided_at=data.get("decided_at", ""),
            comment=data.get("comment", ""),
            checkpoint_ref=data.get("checkpoint_ref", ""),
            effect_cursor=data.get("effect_cursor", ""),
        )


@dataclass
class WorkItem:
    """A claimable unit of work with lease/fencing semantics."""

    id: str = ""
    instance_id: str = ""
    activity_id: str = ""
    state: str = ""
    result: Any = None
    checkpoint_ref: str = ""
    effect_cursor: str = ""
    attempts: int = 0
    max_attempts: int = 0
    last_error: str = ""
    approval: Optional[Approval] = None

    @classmethod
    def from_dict(cls, data):
        data = data or {}
        return cls(
            id=data.get("id", ""),
            instance_id=data.get("instance_id", ""),
            activity_id=data.get("activity_id", ""),
            state=data.get("state", ""),
            result=data.get("result"),
            checkpoint_ref=data.get("checkpoint_ref", ""),
            effect_cursor=data.get("effect_cursor", ""),
            attempts=data.get("attempts", 0) or 0,
            max_attempts=data.get("max_attempts", 0) or 0,
            last_error=data.get("last_error", "") or "",
            approval=Approval.from_dict(data.get("approval")),
        )


@dataclass
class ApprovalRequest:
    """Request human approval before a work item advances."""

    actor: str = ""
    reason: str = ""
    final_args: Any = None
    checkpoint_ref: str = ""
    effect_cursor: str = ""


@dataclass
class WorkResult:
    """Outcome reported by a worker callback."""

    status: str = "success"
    error: str = ""
    outcome: str = ""
    result: Any = None
    checkpoint_ref: str = ""
    effect_cursor: str = ""
    approval: Optional[ApprovalRequest] = None

    @classmethod
    def success(cls, result=None, checkpoint_ref="", effect_cursor=""):
        return cls(
            status="success",
            result=result,
            checkpoint_ref=checkpoint_ref,
            effect_cursor=effect_cursor,
        )

    @classmethod
    def failure(cls, error):
        return cls(status="failure", error=error)

    @classmethod
    def request_approval(cls, approval):
        return cls(status="approval", approval=approval)


class Client:
    def __init__(self, base_url):
        self.base_url = base_url.rstrip("/")

    def _request(self, method, path, body=None):
        url = self.base_url + path
        data = json.dumps(body).encode() if body is not None else None
        req = urllib.request.Request(url, data=data, method=method,
                                     headers={"Content-Type": "application/json"})
        try:
            with urllib.request.urlopen(req) as resp:
                raw = resp.read().decode()
                return json.loads(raw) if raw else None
        except urllib.error.HTTPError as e:
            try:
                msg = e.read().decode()
            finally:
                e.close()
            raise WorkerError(f"{e.code}: {msg}") from e

    def poll(self, instance_id=None):
        path = "/api/v1/work-items"
        if instance_id:
            path += "?instance=" + _escape(instance_id)
        data = self._request("GET", path) or []
        return [WorkItem.from_dict(item) for item in data]

    def get_work_item(self, work_item_id):
        path = f"/api/v1/work-items/{_escape(work_item_id)}"
        return WorkItem.from_dict(self._request("GET", path))

    def claim(self, work_item_id, lease="30s"):
        return self._request("POST", f"/api/v1/work-items/{_escape(work_item_id)}/claim",
                             {"lease": lease})

    def heartbeat(self, work_item_id, token, lease="30s"):
        return self._request("POST", f"/api/v1/work-items/{_escape(work_item_id)}/heartbeat",
                             {"token": token, "lease": lease})

    def complete(self, work_item_id, token, result=None,
                 checkpoint_ref="", effect_cursor=""):
        body = {"token": token}
        if _nonempty(result):
            body["result"] = result
        if checkpoint_ref:
            body["checkpoint_ref"] = checkpoint_ref
        if effect_cursor:
            body["effect_cursor"] = effect_cursor
        return self._request("POST", f"/api/v1/work-items/{_escape(work_item_id)}/complete",
                             body)

    def request_approval(self, work_item_id, token, approval_request):
        body = {"token": token, "reason": approval_request.reason}
        if approval_request.actor:
            body["actor"] = approval_request.actor
        if _nonempty(approval_request.final_args):
            body["final_args"] = approval_request.final_args
        if approval_request.checkpoint_ref:
            body["checkpoint_ref"] = approval_request.checkpoint_ref
        if approval_request.effect_cursor:
            body["effect_cursor"] = approval_request.effect_cursor
        return self._request(
            "POST",
            f"/api/v1/work-items/{_escape(work_item_id)}/request-approval",
            body,
        )

    def approve(self, work_item_id, actor, comment="", final_args=None):
        body = {"actor": actor, "comment": comment}
        if _nonempty(final_args):
            body["final_args"] = final_args
        return self._request("POST", f"/api/v1/work-items/{_escape(work_item_id)}/approve",
                             body)

    def reject(self, work_item_id, actor, reason):
        return self._request("POST", f"/api/v1/work-items/{_escape(work_item_id)}/reject",
                             {"actor": actor, "reason": reason})

    def cancel(self, work_item_id):
        return self._request("POST", f"/api/v1/work-items/{_escape(work_item_id)}/cancel",
                             {})

    def fail(self, work_item_id, token, error):
        return self._request("POST", f"/api/v1/work-items/{_escape(work_item_id)}/fail",
                             {"token": token, "error": error})


class Worker:
    def __init__(self, client, fn, lease="30s"):
        self.client = client
        self.fn = fn  # fn(item) -> WorkResult
        self.lease = lease

    def process_once(self, instance_id=None):
        processed = 0
        for item in self.client.poll(instance_id):
            try:
                claim = self.client.claim(item.id, self.lease)
                token = claim["token"]
            except WorkerError:
                continue  # claimed by another worker
            res = self.fn(item)
            try:
                if res.status == "success":
                    self.client.complete(item.id, token, res.result,
                                         res.checkpoint_ref, res.effect_cursor)
                elif res.status == "approval":
                    req = res.approval
                    if req is None:
                        self.client.fail(
                            item.id, token,
                            "approval result is missing approval request",
                        )
                    else:
                        req = replace(
                            req,
                            checkpoint_ref=req.checkpoint_ref or res.checkpoint_ref,
                            effect_cursor=req.effect_cursor or res.effect_cursor,
                        )
                        try:
                            self.client.request_approval(item.id, token, req)
                        except WorkerError as e:
                            self.client.fail(item.id, token, str(e))
                else:
                    self.client.fail(item.id, token, res.error or "failure")
                processed += 1
            except WorkerError:
                pass  # lease expired / stale token
        return processed
