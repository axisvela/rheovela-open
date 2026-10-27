export class CoreAPIError extends Error {
  constructor(status, body) {
    super(`core API returned ${status}: ${body}`);
    this.name = "CoreAPIError";
    this.status = status;
    this.body = body;
  }
}

function baseURL(value) {
  return String(value || "").replace(/\/+$/, "");
}

function headers(options) {
  const out = { "Content-Type": "application/json" };
  if (options.tenant) out["X-Tenant-ID"] = options.tenant;
  if (options.token) out.Authorization = `Bearer ${options.token}`;
  return out;
}

async function request(fetchImpl, url, init) {
  const res = await fetchImpl(url, init);
  const text = await res.text();
  if (!res.ok) {
    throw new CoreAPIError(res.status, text);
  }
  return text ? JSON.parse(text) : null;
}

export async function listWorkflows(url, options = {}) {
  const fetchImpl = options.fetchImpl || fetch;
  return request(fetchImpl, `${baseURL(url)}/api/v1/workflows`, {
    method: "GET",
    headers: headers(options),
  });
}

export async function loadWorkflow(url, id, options = {}) {
  const fetchImpl = options.fetchImpl || fetch;
  return request(fetchImpl, `${baseURL(url)}/api/v1/workflows/${encodeURIComponent(id)}`, {
    method: "GET",
    headers: headers(options),
  });
}

export async function publishWorkflow(url, wf, options = {}) {
  const fetchImpl = options.fetchImpl || fetch;
  return request(fetchImpl, `${baseURL(url)}/api/v1/workflows`, {
    method: "POST",
    headers: headers(options),
    body: JSON.stringify(wf),
  });
}
