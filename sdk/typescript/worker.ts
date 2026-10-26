/** Rheovela TypeScript worker SDK (no runtime deps; Node 18+ fetch). */

export interface Approval {
  state: string;
  pending_stage?: string;
  final_args?: unknown;
  reason?: string;
  requested_by?: string;
  requested_at?: string;
  decided_by?: string;
  decided_at?: string;
  comment?: string;
  checkpoint_ref?: string;
  effect_cursor?: string;
}

export interface WorkItem {
  id: string;
  instance_id: string;
  activity_id: string;
  state: string;
  result?: unknown;
  checkpoint_ref?: string;
  effect_cursor?: string;
  attempts?: number;
  max_attempts?: number;
  last_error?: string;
  approval?: Approval;
}

export interface ApprovalRequest {
  actor?: string;
  reason: string;
  final_args?: unknown;
  checkpoint_ref?: string;
  effect_cursor?: string;
}

export interface WorkResult {
  status: "success" | "failure" | "approval";
  error?: string;
  outcome?: string;
  result?: unknown;
  checkpoint_ref?: string;
  effect_cursor?: string;
  approval?: ApprovalRequest;
}

export interface CompleteOptions {
  result?: unknown;
  checkpoint_ref?: string;
  effect_cursor?: string;
}

export type WorkFn = (item: WorkItem) => Promise<WorkResult>;

export function success(
  result?: unknown,
  checkpoint_ref = "",
  effect_cursor = "",
): WorkResult {
  return { status: "success", result, checkpoint_ref, effect_cursor };
}

export function failure(error: string): WorkResult {
  return { status: "failure", error };
}

export function requestApproval(approval: ApprovalRequest): WorkResult {
  return { status: "approval", approval };
}

function nonempty(value: unknown): boolean {
  return value !== undefined && value !== null && value !== "";
}

export class WorkerError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.status = status;
    this.name = "WorkerError";
  }
}

export class Client {
  private baseUrl: string;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

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

  private itemPath(id: string): string {
    return `/api/v1/work-items/${encodeURIComponent(id)}`;
  }

  poll(instanceId?: string): Promise<WorkItem[]> {
    const path = instanceId
      ? `/api/v1/work-items?instance=${encodeURIComponent(instanceId)}`
      : "/api/v1/work-items";
    return this.req("GET", path);
  }

  getWorkItem(id: string): Promise<WorkItem> {
    return this.req("GET", this.itemPath(id));
  }

  claim(id: string, lease?: string): Promise<{ token: string; lease: string }> {
    return this.req("POST", `${this.itemPath(id)}/claim`, { lease });
  }

  heartbeat(id: string, token: string, lease?: string): Promise<{ ok: boolean }> {
    return this.req("POST", `${this.itemPath(id)}/heartbeat`, { token, lease });
  }

  complete(
    id: string,
    token: string,
    options: CompleteOptions = {},
  ): Promise<{ ok: boolean }> {
    const body: Record<string, unknown> = { token };
    if (nonempty(options.result)) body.result = options.result;
    if (options.checkpoint_ref) body.checkpoint_ref = options.checkpoint_ref;
    if (options.effect_cursor) body.effect_cursor = options.effect_cursor;
    return this.req("POST", `${this.itemPath(id)}/complete`, body);
  }

  requestApproval(
    id: string,
    token: string,
    approval: ApprovalRequest,
  ): Promise<{ ok: boolean }> {
    const body: Record<string, unknown> = {
      token,
      reason: approval.reason,
    };
    if (approval.actor) body.actor = approval.actor;
    if (nonempty(approval.final_args)) body.final_args = approval.final_args;
    if (approval.checkpoint_ref) body.checkpoint_ref = approval.checkpoint_ref;
    if (approval.effect_cursor) body.effect_cursor = approval.effect_cursor;
    return this.req("POST", `${this.itemPath(id)}/request-approval`, body);
  }

  approve(
    id: string,
    actor: string,
    comment = "",
    final_args?: unknown,
  ): Promise<{ ok: boolean }> {
    const body: Record<string, unknown> = { actor, comment };
    if (nonempty(final_args)) body.final_args = final_args;
    return this.req("POST", `${this.itemPath(id)}/approve`, body);
  }

  reject(id: string, actor: string, reason: string): Promise<{ ok: boolean }> {
    return this.req("POST", `${this.itemPath(id)}/reject`, { actor, reason });
  }

  cancel(id: string): Promise<{ ok: boolean }> {
    return this.req("POST", `${this.itemPath(id)}/cancel`, {});
  }

  fail(id: string, token: string, error: string): Promise<{ ok: boolean }> {
    return this.req("POST", `${this.itemPath(id)}/fail`, { token, error });
  }
}

export class Worker {
  private client: Client;
  private fn: WorkFn;
  private lease: string;

  constructor(client: Client, fn: WorkFn, lease = "30s") {
    this.client = client;
    this.fn = fn;
    this.lease = lease;
  }

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
        if (res.status === "success") {
          await this.client.complete(item.id, token, {
            result: res.result,
            checkpoint_ref: res.checkpoint_ref,
            effect_cursor: res.effect_cursor,
          });
        } else if (res.status === "approval") {
          const approval = res.approval;
          if (!approval) {
            await this.client.fail(
              item.id,
              token,
              "approval result is missing approval request",
            );
          } else {
            const merged: ApprovalRequest = {
              ...approval,
              checkpoint_ref: approval.checkpoint_ref || res.checkpoint_ref,
              effect_cursor: approval.effect_cursor || res.effect_cursor,
            };
            try {
              await this.client.requestApproval(item.id, token, merged);
            } catch (err) {
              await this.client.fail(item.id, token, String(err));
            }
          }
        } else {
          await this.client.fail(item.id, token, res.error ?? "failure");
        }
        processed++;
      } catch {
        /* stale token / expired lease */
      }
    }
    return processed;
  }
}
