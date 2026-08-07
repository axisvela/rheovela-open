"""Rheovela Python worker SDK (stdlib only).

Uses the core Worker HTTP API (rheo serve). Minimal, dependency-free.
"""
import json
import urllib.request
import urllib.error


class WorkerError(Exception):
    pass


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
                return json.loads(resp.read().decode())
        except urllib.error.HTTPError as e:
            try:
                msg = e.read().decode()
            finally:
                e.close()
            raise WorkerError(f"{e.code}: {msg}") from e

    def poll(self, instance_id=None):
        path = "/api/v1/work-items"
        if instance_id:
            path += "?instance=" + instance_id
        return self._request("GET", path)

    def claim(self, work_item_id, lease="30s"):
        return self._request("POST", f"/api/v1/work-items/{work_item_id}/claim",
                             {"lease": lease})

    def heartbeat(self, work_item_id, token, lease="30s"):
        return self._request("POST", f"/api/v1/work-items/{work_item_id}/heartbeat",
                             {"token": token, "lease": lease})

    def complete(self, work_item_id, token):
        return self._request("POST", f"/api/v1/work-items/{work_item_id}/complete",
                             {"token": token})

    def fail(self, work_item_id, token, error):
        return self._request("POST", f"/api/v1/work-items/{work_item_id}/fail",
                             {"token": token, "error": error})


class Worker:
    def __init__(self, client, fn, lease="30s"):
        self.client = client
        self.fn = fn  # fn(item) -> ("success"|"failure", error_message_or_None)
        self.lease = lease

    def process_once(self, instance_id=None):
        processed = 0
        for item in self.client.poll(instance_id):
            try:
                claim = self.client.claim(item["id"], self.lease)
                token = claim["token"]
            except WorkerError:
                continue  # claimed by another worker
            status, err = self.fn(item)
            try:
                if status == "success":
                    self.client.complete(item["id"], token)
                else:
                    self.client.fail(item["id"], token, err or "failure")
                processed += 1
            except WorkerError:
                pass  # lease expired / stale token
        return processed
