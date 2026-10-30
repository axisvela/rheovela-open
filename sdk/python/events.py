"""Durable WebSocket event client for Rheovela.

The client mirrors the Go ``sdk/go/events`` package: it persists its cursor and
pending events under ``state_dir``, reconnects with ``after_seq``, and delivers
events at least once until the handler succeeds.
"""

from __future__ import annotations

import asyncio
import dataclasses
import inspect
import json
import os
import tempfile
from typing import Any, Awaitable, Callable, Optional
from urllib.parse import parse_qsl, urlencode, urlsplit, urlunsplit

try:
    import websockets
except ImportError:  # pragma: no cover - surfaced by tests when unavailable
    websockets = None


@dataclasses.dataclass
class Event:
    seq: int
    type: str
    actor_id: str = ""
    payload: Any = None
    wall_time: str = ""

    @classmethod
    def from_dict(cls, raw: dict[str, Any]) -> "Event":
        return cls(
            seq=int(raw.get("seq") or 0),
            type=str(raw.get("type") or ""),
            actor_id=str(raw.get("actor_id") or ""),
            payload=raw.get("payload"),
            wall_time=str(raw.get("wall_time") or ""),
        )


class DurableQueue:
    """File-backed pending queue and cursor."""

    def __init__(self, directory: str):
        if not directory:
            raise ValueError("state_dir is required")
        os.makedirs(directory, exist_ok=True)
        self.path = os.path.join(directory, "events-state.json")
        self._last_seq = 0
        self._pending: list[Event] = []
        self._load()

    @property
    def last_seq(self) -> int:
        return self._last_seq

    def pending(self) -> list[Event]:
        return list(self._pending)

    def push(self, event: Event) -> None:
        if event.seq <= self._last_seq:
            return
        if any(item.seq == event.seq for item in self._pending):
            return
        self._pending.append(event)
        self._pending.sort(key=lambda item: item.seq)
        self._persist()

    def ack(self, seq: int) -> None:
        if seq > self._last_seq:
            self._last_seq = seq
        self._pending = [item for item in self._pending if item.seq > seq]
        self._persist()

    def set_floor(self, seq: int) -> None:
        if self._pending or seq <= self._last_seq:
            return
        self._last_seq = seq
        self._persist()

    def _load(self) -> None:
        try:
            with open(self.path, "r", encoding="utf-8") as handle:
                raw = json.load(handle)
        except FileNotFoundError:
            return
        self._last_seq = int(raw.get("last_seq") or 0)
        self._pending = [
            Event.from_dict(item)
            for item in raw.get("pending") or []
            if int(item.get("seq") or 0) > self._last_seq
        ]
        self._pending.sort(key=lambda item: item.seq)

    def _persist(self) -> None:
        raw = {
            "last_seq": self._last_seq,
            "pending": [dataclasses.asdict(item) for item in self._pending],
        }
        fd, tmp = tempfile.mkstemp(dir=os.path.dirname(self.path), prefix=".events-")
        try:
            with os.fdopen(fd, "w", encoding="utf-8") as handle:
                json.dump(raw, handle)
                handle.flush()
                os.fsync(handle.fileno())
            os.replace(tmp, self.path)
        finally:
            if os.path.exists(tmp):
                os.unlink(tmp)


class EventClient:
    """Reconnectable, durable WebSocket event subscriber."""

    def __init__(
        self,
        url: str,
        stream_id: str,
        state_dir: str,
        handler: Callable[[Event], Optional[Awaitable[None]]],
        *,
        types: Optional[list[str]] = None,
        correlation_key: str = "",
        predicate: str = "",
        after_seq: int = 0,
        reconnect_min: float = 0.2,
        reconnect_max: float = 30.0,
        retry_delay: float = 1.0,
        on_error: Optional[Callable[[Exception], None]] = None,
    ):
        if not url:
            raise ValueError("url is required")
        if not stream_id:
            raise ValueError("stream_id is required")
        if handler is None:
            raise ValueError("handler is required")
        self.url = url
        self.stream_id = stream_id
        self.handler = handler
        self.types = list(types or [])
        self.correlation_key = correlation_key
        self.predicate = predicate
        self.reconnect_min = reconnect_min
        self.reconnect_max = reconnect_max
        self.retry_delay = retry_delay
        self.on_error = on_error
        self.queue = DurableQueue(state_dir)
        if after_seq > 0:
            self.queue.set_floor(after_seq)
        self._attempts = 0

    @property
    def last_seq(self) -> int:
        return self.queue.last_seq

    def pending(self) -> list[Event]:
        return self.queue.pending()

    async def run(self) -> None:
        backoff = self.reconnect_min
        while True:
            try:
                await self._drain()
                await self._session()
                backoff = self.reconnect_min
            except asyncio.CancelledError:
                return
            except Exception as exc:  # noqa: BLE001 - surfaced via on_error
                if self.on_error:
                    self.on_error(exc)
                await asyncio.sleep(backoff)
                backoff = min(backoff * 2, self.reconnect_max)

    async def _drain(self) -> None:
        for event in self.queue.pending():
            await self._deliver(event)
            self.queue.ack(event.seq)

    async def _session(self) -> None:
        if websockets is None:
            raise RuntimeError("websockets package is required")
        async with websockets.connect(self._endpoint()) as socket:
            while True:
                raw = await socket.recv()
                event = Event.from_dict(json.loads(raw))
                if event.seq == 0:
                    continue
                self.queue.push(event)
                await self._deliver(event)
                self.queue.ack(event.seq)
                await socket.send(json.dumps({"type": "ack", "seq": event.seq}))

    async def _deliver(self, event: Event) -> None:
        result = self.handler(event)
        if inspect.isawaitable(result):
            await result

    def _endpoint(self) -> str:
        parts = urlsplit(self.url)
        scheme = "wss" if parts.scheme in ("https", "wss") else "ws"
        path = parts.path.rstrip("/") + "/api/v1/events/ws"
        query = [
            ("stream", self.stream_id),
            *parse_qsl(parts.query, keep_blank_values=True),
        ]
        if self.last_seq > 0:
            query.append(("after_seq", str(self.last_seq)))
        if self.types:
            query.append(("types", ",".join(self.types)))
        if self.correlation_key:
            query.append(("correlation_key", self.correlation_key))
        if self.predicate:
            query.append(("predicate", self.predicate))
        return urlunsplit((scheme, parts.netloc, path, urlencode(query), ""))
