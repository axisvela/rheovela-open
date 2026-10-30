/** Rheovela TypeScript durable WebSocket event client (no runtime deps). */

import {
  existsSync,
  mkdirSync,
  readFileSync,
  renameSync,
  writeFileSync,
} from "node:fs";
import { dirname, join } from "node:path";

export interface RheovelaEvent {
  seq: number;
  type: string;
  actor_id?: string;
  payload?: unknown;
  wall_time?: string;
}

export interface EventClientOptions {
  url: string;
  streamId: string;
  stateDir: string;
  handler: (event: RheovelaEvent) => void | Promise<void>;
  types?: string[];
  correlationKey?: string;
  predicate?: string;
  afterSeq?: number;
  reconnectMinMs?: number;
  reconnectMaxMs?: number;
  retryDelayMs?: number;
  onError?: (error: Error) => void;
}

interface PersistedState {
  last_seq: number;
  pending: RheovelaEvent[];
}

export class DurableQueue {
  readonly path: string;
  private lastSeqValue = 0;
  private pendingEvents: RheovelaEvent[] = [];

  constructor(directory: string) {
    if (!directory) throw new Error("events: stateDir is required");
    mkdirSync(directory, { recursive: true });
    this.path = join(directory, "events-state.json");
    this.load();
  }

  get lastSeq(): number {
    return this.lastSeqValue;
  }

  pending(): RheovelaEvent[] {
    return [...this.pendingEvents];
  }

  push(event: RheovelaEvent): void {
    if (!event.seq || event.seq <= this.lastSeqValue) return;
    if (this.pendingEvents.some((item) => item.seq === event.seq)) return;
    this.pendingEvents.push(event);
    this.pendingEvents.sort((a, b) => a.seq - b.seq);
    this.persist();
  }

  ack(seq: number): void {
    if (seq > this.lastSeqValue) this.lastSeqValue = seq;
    this.pendingEvents = this.pendingEvents.filter((item) => item.seq > seq);
    this.persist();
  }

  setFloor(seq: number): void {
    if (this.pendingEvents.length > 0 || seq <= this.lastSeqValue) return;
    this.lastSeqValue = seq;
    this.persist();
  }

  private load(): void {
    if (!existsSync(this.path)) return;
    const raw = JSON.parse(readFileSync(this.path, "utf8")) as PersistedState;
    this.lastSeqValue = Number(raw.last_seq || 0);
    this.pendingEvents = (raw.pending || [])
      .filter((item) => Number(item.seq || 0) > this.lastSeqValue)
      .sort((a, b) => a.seq - b.seq);
  }

  private persist(): void {
    const state: PersistedState = {
      last_seq: this.lastSeqValue,
      pending: this.pendingEvents,
    };
    const tmp = `${this.path}.tmp`;
    mkdirSync(dirname(this.path), { recursive: true });
    writeFileSync(tmp, JSON.stringify(state), { encoding: "utf8" });
    renameSync(tmp, this.path);
  }
}

export class EventClient {
  readonly queue: DurableQueue;
  private readonly options: Required<
    Pick<
      EventClientOptions,
      | "reconnectMinMs"
      | "reconnectMaxMs"
      | "retryDelayMs"
      | "types"
      | "correlationKey"
      | "predicate"
    >
  > &
    EventClientOptions;

  constructor(options: EventClientOptions) {
    if (!options.url) throw new Error("events: url is required");
    if (!options.streamId) throw new Error("events: streamId is required");
    if (!options.handler) throw new Error("events: handler is required");
    this.options = {
      ...options,
      types: options.types || [],
      correlationKey: options.correlationKey || "",
      predicate: options.predicate || "",
      reconnectMinMs: options.reconnectMinMs ?? 200,
      reconnectMaxMs: options.reconnectMaxMs ?? 30_000,
      retryDelayMs: options.retryDelayMs ?? 1_000,
    };
    this.queue = new DurableQueue(options.stateDir);
    if ((options.afterSeq || 0) > 0) this.queue.setFloor(options.afterSeq!);
  }

  get lastSeq(): number {
    return this.queue.lastSeq;
  }

  pending(): RheovelaEvent[] {
    return this.queue.pending();
  }

  async run(signal?: AbortSignal): Promise<void> {
    let backoff = this.options.reconnectMinMs;
    while (!signal?.aborted) {
      try {
        await this.drain();
        await this.session(signal);
        backoff = this.options.reconnectMinMs;
      } catch (error) {
        if (signal?.aborted) return;
        this.report(error as Error);
        await sleep(backoff, signal);
        backoff = Math.min(backoff * 2, this.options.reconnectMaxMs);
      }
    }
  }

  private async drain(): Promise<void> {
    for (const event of this.queue.pending()) {
      await this.options.handler(event);
      this.queue.ack(event.seq);
    }
  }

  private session(signal?: AbortSignal): Promise<void> {
    return new Promise((resolve, reject) => {
      const WS = (globalThis as { WebSocket?: typeof WebSocket }).WebSocket;
      if (!WS) {
        reject(new Error("events: WebSocket is not available"));
        return;
      }
      const socket = new WS(this.endpoint());
      let chain = Promise.resolve();
      const abort = () => {
        try {
          socket.close();
        } catch {
          // ignore close errors
        }
        resolve();
      };
      signal?.addEventListener("abort", abort, { once: true });
      socket.onopen = () => undefined;
      socket.onerror = () => reject(new Error("events: websocket error"));
      socket.onclose = () => {
        signal?.removeEventListener("abort", abort);
        resolve();
      };
      socket.onmessage = (message: MessageEvent) => {
        chain = chain
          .then(async () => {
            const event = JSON.parse(String(message.data)) as RheovelaEvent;
            if (!event.seq) return;
            this.queue.push(event);
            await this.options.handler(event);
            this.queue.ack(event.seq);
            socket.send(JSON.stringify({ type: "ack", seq: event.seq }));
          })
          .catch((error) => {
            socket.close();
            reject(error as Error);
          });
      };
    });
  }

  private endpoint(): string {
    const parsed = new URL(this.options.url);
    parsed.protocol = parsed.protocol === "https:" || parsed.protocol === "wss:" ? "wss:" : "ws:";
    parsed.pathname = `${parsed.pathname.replace(/\/$/, "")}/api/v1/events/ws`;
    parsed.searchParams.set("stream", this.options.streamId);
    if (this.queue.lastSeq > 0) {
      parsed.searchParams.set("after_seq", String(this.queue.lastSeq));
    }
    if (this.options.types.length > 0) {
      parsed.searchParams.set("types", this.options.types.join(","));
    }
    if (this.options.correlationKey) {
      parsed.searchParams.set("correlation_key", this.options.correlationKey);
    }
    if (this.options.predicate) {
      parsed.searchParams.set("predicate", this.options.predicate);
    }
    return parsed.toString();
  }

  private report(error: Error): void {
    if (this.options.onError) this.options.onError(error);
  }
}

function sleep(ms: number, signal?: AbortSignal): Promise<void> {
  return new Promise((resolve) => {
    const timer = setTimeout(resolve, ms);
    signal?.addEventListener(
      "abort",
      () => {
        clearTimeout(timer);
        resolve();
      },
      { once: true },
    );
  });
}
