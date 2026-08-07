/** Rheovela TypeScript worker SDK (no runtime deps; Node 18+ fetch). */
export interface WorkItem {
  id: string;
  instance_id: string;
  activity_id: string;
  state: string;
}

export type WorkResult = { status: "success" | "failure"; error?: string };

export type WorkFn = (item: WorkItem) => Promise<WorkResult>;

export class WorkerError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
    this.name = "WorkerError";
  }
}

export class Client {
  constructor(private baseUrl: string) {}

  private async req(method: string, path: string, body?: unknown): Promise<any> {
    const res = await fetch(this.baseUrl + path, {
      method,
      headers: { "Content-Type": "application/json" },
      body: body ? JSON.stringify(body) : undefined,
    });
    const text = await res.text();
    if (!res.ok) throw new WorkerError(res.status, text);
    return text ? JSON.parse(text) : {};
  }

  poll(instanceId?: string): Promise<WorkItem[]> {
    const path = instanceId
      ? `/api/v1/work-items?instance=${encodeURIComponent(instanceId)}`
      : "/api/v1/work-items";
    return this.req("GET", path);
  }

  claim(id: string, lease?: string): Promise<{ token: string; lease: string }> {
    return this.req("POST", `/api/v1/work-items/${id}/claim`, { lease });
  }

  heartbeat(id: string, token: string, lease?: string): Promise<{ ok: boolean }> {
    return this.req("POST", `/api/v1/work-items/${id}/heartbeat`, { token, lease });
  }

  complete(id: string, token: string): Promise<{ ok: boolean }> {
    return this.req("POST", `/api/v1/work-items/${id}/complete`, { token });
  }

  fail(id: string, token: string, error: string): Promise<{ ok: boolean }> {
    return this.req("POST", `/api/v1/work-items/${id}/fail`, { token, error });
  }
}

export class Worker {
  constructor(
    private client: Client,
    private fn: WorkFn,
    private lease = "30s",
  ) {}

  async processOnce(instanceId?: string): Promise<number> {
    let processed = 0;
    for (const item of await this.client.poll(instanceId)) {
      let token: string;
      try {
        token = (await this.client.claim(item.id, this.lease)).token;
      } catch {
        continue; // claimed by another worker
      }
      const res = await this.fn(item);
      try {
        if (res.status === "success") await this.client.complete(item.id, token);
        else await this.client.fail(item.id, token, res.error ?? "failure");
        processed++;
      } catch {
        /* stale token / expired lease */
      }
    }
    return processed;
  }
}
