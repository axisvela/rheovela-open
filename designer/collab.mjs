// Transport adapter between the Designer CRDT (crdt.mjs) and the EE
// collaboration service. CRDT semantics stay in crdt.mjs; this module only maps
// createSync ops to the EE HTTP contract.

export function createEECollabClient({
  baseUrl,
  documentId,
  headers = {},
  fetchImpl = globalThis.fetch,
} = {}) {
  if (!baseUrl) throw new Error("collab: baseUrl is required");
  if (!documentId) throw new Error("collab: documentId is required");

  const root = String(baseUrl).replace(/\/+$/, "");
  const documentURL = `${root}/api/v1/ee/collab/documents/${encodeURIComponent(documentId)}`;
  const presenceURL = `${root}/api/v1/ee/collab/presence`;

  async function request(url, init = {}) {
    if (typeof fetchImpl !== "function") {
      throw new Error("collab: fetch is not available");
    }
    const response = await fetchImpl(url, {
      ...init,
      headers: {
        "content-type": "application/json",
        ...headers,
        ...(init.headers || {}),
      },
    });
    if (!response.ok) {
      throw new Error(`collab: HTTP ${response.status}`);
    }
    return response.json();
  }

  return {
    sendOps(ops) {
      return request(`${documentURL}/ops`, {
        method: "POST",
        body: JSON.stringify({ ops }),
      });
    },
    getDocument() {
      return request(documentURL, { method: "GET" });
    },
    heartbeat(clientId, cursor) {
      const body = { client_id: clientId };
      if (cursor !== undefined) body.cursor = cursor;
      return request(presenceURL, { method: "POST", body: JSON.stringify(body) });
    },
    presence() {
      return request(presenceURL, { method: "GET" });
    },
  };
}
