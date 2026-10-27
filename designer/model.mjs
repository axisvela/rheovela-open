const DELEGATION_KINDS = new Set(["actor", "role", "capability"]);

export function emptyWorkflow() {
  return {
    workflow_id: "new-workflow",
    title: "New Workflow",
    source: "designer",
    stages: [],
    transitions: [],
    compensations: [],
  };
}

export function cloneWorkflow(wf) {
  return JSON.parse(JSON.stringify(wf));
}

export function normalizeWorkflow(input) {
  const raw = typeof input === "string" ? JSON.parse(input) : cloneWorkflow(input);
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
    throw new Error("workflow must be a JSON object");
  }
  const wf = {
    ...raw,
    workflow_id: typeof raw.workflow_id === "string" ? raw.workflow_id : "new-workflow",
    title: typeof raw.title === "string" ? raw.title : "New Workflow",
    source: typeof raw.source === "string" && raw.source ? raw.source : "designer",
    stages: Array.isArray(raw.stages) ? raw.stages : [],
    transitions: Array.isArray(raw.transitions) ? raw.transitions : [],
    compensations: Array.isArray(raw.compensations) ? raw.compensations : [],
  };
  wf.stages = wf.stages.map((stage) => ({
    ...stage,
    inputs: Array.isArray(stage.inputs) ? stage.inputs : [],
    outputs: Array.isArray(stage.outputs) ? stage.outputs : [],
  }));
  return wf;
}

export function validateWorkflow(input) {
  let wf;
  try {
    wf = normalizeWorkflow(input);
  } catch (err) {
    return [err.message];
  }
  const errors = [];
  if (!wf.workflow_id.trim()) errors.push("workflow_id is required");
  if (!wf.title.trim()) errors.push("title is required");
  if (!wf.source.trim()) errors.push("source is required");
  if (wf.stages.length === 0) errors.push("stages must contain at least one stage");

  const ids = new Set();
  for (const stage of wf.stages) {
    if (!stage || typeof stage.id !== "string" || !stage.id.trim()) {
      errors.push("every stage needs a non-empty id");
      continue;
    }
    if (ids.has(stage.id)) errors.push(`duplicate stage id: ${stage.id}`);
    ids.add(stage.id);
    if (stage.delegation) {
      if (!DELEGATION_KINDS.has(stage.delegation.kind)) {
        errors.push(`stage ${stage.id}: delegation kind must be actor, role, or capability`);
      }
      if (!stage.delegation.delegatee) {
        errors.push(`stage ${stage.id}: delegation delegatee is required`);
      }
    }
    if (stage.evidence !== undefined && !Array.isArray(stage.evidence)) {
      errors.push(`stage ${stage.id}: evidence must be an array`);
    }
  }

  const adj = new Map();
  const indegree = new Map();
  for (const id of ids) {
    adj.set(id, []);
    indegree.set(id, 0);
  }
  for (const tr of wf.transitions) {
    if (!tr || !tr.from || !tr.to) {
      errors.push("every transition needs from and to");
      continue;
    }
    if (!ids.has(tr.from)) errors.push(`transition references unknown from stage: ${tr.from}`);
    if (!ids.has(tr.to)) errors.push(`transition references unknown to stage: ${tr.to}`);
    if (ids.has(tr.from) && ids.has(tr.to)) {
      adj.get(tr.from).push(tr.to);
      indegree.set(tr.to, indegree.get(tr.to) + 1);
    }
  }
  const queue = [];
  for (const [id, n] of indegree) {
    if (n === 0) queue.push(id);
  }
  let visited = 0;
  while (queue.length > 0) {
    const id = queue.shift();
    visited++;
    for (const next of adj.get(id)) {
      indegree.set(next, indegree.get(next) - 1);
      if (indegree.get(next) === 0) queue.push(next);
    }
  }
  if (visited !== ids.size) errors.push("transition graph contains a cycle");

  for (const c of wf.compensations) {
    if (!c || !c.for_activity) {
      errors.push("every compensation needs for_activity");
      continue;
    }
    if (!ids.has(c.for_activity)) {
      errors.push(`compensation references unknown stage: ${c.for_activity}`);
    }
    if (!c.compensate_with && !c.capability) {
      errors.push(`compensation for ${c.for_activity} needs compensate_with or capability`);
    }
  }
  return errors;
}

export function addStage(wf, id, title) {
  const next = normalizeWorkflow(wf);
  const stageID = String(id || "").trim();
  if (!stageID) throw new Error("stage id is required");
  next.stages.push({
    id: stageID,
    title: String(title || stageID),
    inputs: [],
    outputs: [],
  });
  return next;
}

export function updateStage(wf, id, patch = {}) {
  const next = normalizeWorkflow(wf);
  const stage = next.stages.find((s) => s.id === id);
  if (!stage) throw new Error(`unknown stage: ${id}`);
  if (patch.title !== undefined) stage.title = String(patch.title);
  if (patch.assignee !== undefined) stage.assignee_spec = patch.assignee;
  if (patch.evidence !== undefined) stage.evidence = [...patch.evidence];
  if (patch.effectKey !== undefined) {
    if (patch.effectKey) stage.effect_key = patch.effectKey;
    else delete stage.effect_key;
  }
  if (patch.timeout !== undefined) {
    if (patch.timeout) stage.timeout = patch.timeout;
    else delete stage.timeout;
  }
  if (patch.migrationPolicy !== undefined) {
    if (patch.migrationPolicy) {
      stage.migration_policy = patch.migrationPolicy;
    } else {
      delete stage.migration_policy;
    }
  }
  if (patch.delegation !== undefined) {
    if (patch.delegation === null) {
      delete stage.delegation;
    } else {
      stage.delegation = patch.delegation;
    }
  }
  return next;
}

export function removeStage(wf, id) {
  const next = normalizeWorkflow(wf);
  next.stages = next.stages.filter((stage) => stage.id !== id);
  next.transitions = next.transitions.filter((tr) => tr.from !== id && tr.to !== id);
  next.compensations = next.compensations.filter((c) => c.for_activity !== id);
  if (next.bpmn) {
    next.bpmn.nodes = (next.bpmn.nodes || []).filter((n) => n.id !== id && n.attached_to !== id);
    next.bpmn.internal_flows = (next.bpmn.internal_flows || []).filter((f) => f.from !== id && f.to !== id);
    next.bpmn.lanes = (next.bpmn.lanes || []).map((lane) => ({
      ...lane,
      node_ids: (lane.node_ids || []).filter((nodeID) => nodeID !== id),
    }));
    if (next.bpmn.diagram) {
      next.bpmn.diagram.shapes = (next.bpmn.diagram.shapes || []).filter(
        (shape) => shape.bpmn_element !== id,
      );
    }
  }
  return next;
}

export function addTransition(wf, from, to, condition = "") {
  const next = normalizeWorkflow(wf);
  const tr = { from: String(from || "").trim(), to: String(to || "").trim() };
  if (String(condition).trim()) tr.condition = String(condition).trim();
  next.transitions.push(tr);
  return next;
}

export function removeTransition(wf, index) {
  const next = normalizeWorkflow(wf);
  next.transitions.splice(index, 1);
  return next;
}

export function serializeWorkflow(wf) {
  return JSON.stringify(normalizeWorkflow(wf), null, 2);
}

export function layoutWorkflow(input) {
  const wf = normalizeWorkflow(input);
  const ids = wf.stages.map((s) => s.id);
  const idSet = new Set(ids);
  const adj = new Map(ids.map((id) => [id, []]));
  const indegree = new Map(ids.map((id) => [id, 0]));
  for (const tr of wf.transitions) {
    if (!idSet.has(tr.from) || !idSet.has(tr.to)) continue;
    adj.get(tr.from).push(tr.to);
    indegree.set(tr.to, indegree.get(tr.to) + 1);
  }
  const layer = new Map(ids.map((id) => [id, 0]));
  const queue = ids.filter((id) => indegree.get(id) === 0);
  const seen = new Set();
  while (queue.length > 0) {
    const id = queue.shift();
    seen.add(id);
    for (const next of adj.get(id)) {
      layer.set(next, Math.max(layer.get(next), layer.get(id) + 1));
      indegree.set(next, indegree.get(next) - 1);
      if (indegree.get(next) === 0) queue.push(next);
    }
  }
  for (const id of ids) {
    if (!seen.has(id)) layer.set(id, 0);
  }
  const byLayer = new Map();
  for (const stage of wf.stages) {
    const l = layer.get(stage.id);
    if (!byLayer.has(l)) byLayer.set(l, []);
    byLayer.get(l).push(stage);
  }
  const nodes = [];
  for (const [l, stages] of [...byLayer.entries()].sort((a, b) => a[0] - b[0])) {
    stages.forEach((stage, index) => {
      nodes.push({ id: stage.id, title: stage.title || stage.id, layer: l, index });
    });
  }
  return {
    nodes,
    edges: wf.transitions
      .filter((tr) => idSet.has(tr.from) && idSet.has(tr.to))
      .map((tr) => ({ from: tr.from, to: tr.to, condition: tr.condition || "" })),
    layerCount: byLayer.size,
  };
}
