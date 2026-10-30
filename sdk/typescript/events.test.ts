import { test } from "node:test";
import assert from "node:assert/strict";
import crypto from "node:crypto";
import { mkdtempSync } from "node:fs";
import http from "node:http";
import { tmpdir } from "node:os";
import { join } from "node:path";

import { DurableQueue, EventClient } from "./events.ts";
import type { RheovelaEvent } from "./events.ts";

const WS_GUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11";

function frame(payload: unknown): Buffer {
  const body = Buffer.from(JSON.stringify(payload));
  return Buffer.concat([Buffer.from([0x81, body.length]), body]);
}

async function startServer(
  onConnection: (socket: import("node:net").Socket, url: string) => void,
): Promise<{ server: http.Server; port: number; close: () => Promise<void> }> {
  const sockets = new Set<import("node:net").Socket>();
  const server = http.createServer();
  server.on("upgrade", (req, socket) => {
    sockets.add(socket);
    socket.on("close", () => sockets.delete(socket));
    socket.on("data", (data) => {
      if ((data[0] & 0x0f) === 0x8) {
        socket.write(Buffer.from([0x88, 0]));
        socket.destroy();
      }
    });
    const key = String(req.headers["sec-websocket-key"] || "");
    const accept = crypto
      .createHash("sha1")
      .update(key + WS_GUID)
      .digest("base64");
    socket.write(
      "HTTP/1.1 101 Switching Protocols\r\n" +
        "Upgrade: websocket\r\n" +
        "Connection: Upgrade\r\n" +
        `Sec-WebSocket-Accept: ${accept}\r\n\r\n`,
    );
    onConnection(socket, req.url || "");
  });
  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
  const port = (server.address() as { port: number }).port;
  return {
    server,
    port,
    close: async () => {
      for (const socket of sockets) socket.destroy();
      await new Promise<void>((resolve) => server.close(() => resolve()));
    },
  };
}

async function waitUntil(check: () => boolean, timeoutMs = 3000): Promise<void> {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    if (check()) return;
    await new Promise((resolve) => setTimeout(resolve, 10));
  }
  throw new Error("condition not met before timeout");
}

function stateDir(): string {
  return mkdtempSync(join(tmpdir(), "rheovela-events-"));
}

test("queue persists and acknowledges", () => {
  const dir = stateDir();
  const queue = new DurableQueue(dir);
  queue.push({ seq: 2, type: "b" });
  queue.push({ seq: 1, type: "a" });
  queue.push({ seq: 2, type: "b" });

  const reopened = new DurableQueue(dir);
  assert.deepEqual(
    reopened.pending().map((item) => item.seq),
    [1, 2],
  );
  reopened.ack(1);
  assert.equal(reopened.lastSeq, 1);
  assert.deepEqual(
    new DurableQueue(dir).pending().map((item) => item.seq),
    [2],
  );
});

test("client receives and persists", async () => {
  const server = await startServer((socket) => {
    socket.write(frame({ seq: 1, type: "a" }));
    socket.write(frame({ seq: 2, type: "b" }));
  });
  const received: number[] = [];
  const controller = new AbortController();
  const client = new EventClient({
    url: `http://127.0.0.1:${server.port}`,
    streamId: "S",
    stateDir: stateDir(),
    handler: (event: RheovelaEvent) => {
      received.push(event.seq);
    },
    retryDelayMs: 10,
    reconnectMinMs: 10,
  });
  const task = client.run(controller.signal);
  await waitUntil(() => received.length >= 2);
  controller.abort();
  await task;
  assert.deepEqual(received, [1, 2]);
  assert.equal(client.lastSeq, 2);
  assert.deepEqual(client.pending(), []);
  await server.close();
});

test("client reconnects with last seq", async () => {
  const afterSeqs: string[] = [];
  let connections = 0;
  const server = await startServer((socket, url) => {
    connections += 1;
    afterSeqs.push(new URL(url, "http://localhost").searchParams.get("after_seq") || "");
    if (connections === 1) {
      socket.write(frame({ seq: 1, type: "a" }));
      socket.destroy();
      return;
    }
    socket.write(frame({ seq: 2, type: "b" }));
  });
  const received: number[] = [];
  const controller = new AbortController();
  const client = new EventClient({
    url: `http://127.0.0.1:${server.port}`,
    streamId: "S",
    stateDir: stateDir(),
    handler: (event) => {
      received.push(event.seq);
    },
    retryDelayMs: 10,
    reconnectMinMs: 10,
  });
  const task = client.run(controller.signal);
  await waitUntil(() => received.length >= 2);
  controller.abort();
  await task;
  assert.deepEqual(received, [1, 2]);
  assert.ok(connections >= 2);
  assert.equal(afterSeqs[0], "");
  assert.equal(afterSeqs[1], "1");
  await server.close();
});

test("client retries handler failures", async () => {
  const server = await startServer((socket) => {
    socket.write(frame({ seq: 1, type: "a" }));
  });
  const calls: number[] = [];
  const errors: Error[] = [];
  const controller = new AbortController();
  const client = new EventClient({
    url: `http://127.0.0.1:${server.port}`,
    streamId: "S",
    stateDir: stateDir(),
    handler: (event) => {
      calls.push(event.seq);
      if (calls.length === 1) throw new Error("transient");
    },
    retryDelayMs: 10,
    reconnectMinMs: 10,
    onError: (error) => errors.push(error),
  });
  const task = client.run(controller.signal);
  await waitUntil(() => calls.length >= 2 && client.lastSeq === 1);
  controller.abort();
  await task;
  assert.ok(calls.length >= 2);
  assert.ok(errors.length > 0);
  assert.deepEqual(client.pending(), []);
  await server.close();
});
