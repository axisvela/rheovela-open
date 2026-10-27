import { CoreAPIError, importBPMN, listWorkflows, loadWorkflow, publishWorkflow } from "./core.mjs";
import {
  addStage,
  addTransition,
  emptyWorkflow,
  layoutWorkflow,
  normalizeWorkflow,
  removeStage,
  removeTransition,
  serializeWorkflow,
  updateStage,
  validateWorkflow,
} from "./model.mjs";

const $ = (id) => document.getElementById(id);
let wf = emptyWorkflow();
let editingStageID = null;

function splitCSV(value) {
  return String(value || "")
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);
}

const example = {
  workflow_id: "ExpenseApproval",
  title: "Expense Approval",
  source: "designer",
  stages: [
    { id: "validate", title: "Validate", inputs: [], outputs: [] },
    {
      id: "approve",
      title: "Approve",
      inputs: [],
      outputs: [],
      evidence: ["approval_receipt"],
      delegation: {
        kind: "role",
        delegatee: "cost-center-lead",
        scopes: ["approve", "reject"],
        budget: 500,
      },
    },
    { id: "pay", title: "Pay", inputs: [], outputs: [] },
  ],
  transitions: [
    { from: "validate", to: "approve", condition: "amount >= 1000" },
    { from: "validate", to: "pay", condition: "amount < 1000" },
  ],
  compensations: [{ for_activity: "pay", capability: "payroll.reverse" }],
};

function setStatus(message, isError = false) {
  const el = $("status");
  el.textContent = message;
  el.classList.toggle("error", isError);
}

function setCoreStatus(message, isError = false) {
  const el = $("core-status");
  el.textContent = message;
  el.classList.toggle("error", isError);
}

function coreOptions() {
  return {
    url: $("core-url").value,
    tenant: $("core-tenant").value.trim(),
    token: $("core-token").value.trim(),
  };
}

function describeError(err) {
  if (err instanceof CoreAPIError) return `core ${err.status}: ${err.body}`;
  return err?.message || String(err);
}

function renderMeta() {
  $("workflow-id").value = wf.workflow_id;
  $("workflow-title").value = wf.title;
  $("workflow-source").value = wf.source;
}

function renderStages() {
  const tbody = $("stage-rows");
  tbody.replaceChildren();
  for (const stage of wf.stages) {
    const tr = document.createElement("tr");
    const id = document.createElement("td");
    id.textContent = stage.id;
    const title = document.createElement("td");
    title.textContent = stage.title || "";
    const meta = document.createElement("td");
    const bits = [];
    if (stage.delegation) {
      bits.push(`delegate ${stage.delegation.kind}:${stage.delegation.delegatee}`);
    }
    if (Array.isArray(stage.evidence) && stage.evidence.length) {
      bits.push(`evidence: ${stage.evidence.join(", ")}`);
    }
    meta.textContent = bits.join(" · ");
    const actions = document.createElement("td");
    const edit = document.createElement("button");
    edit.type = "button";
    edit.className = "secondary";
    edit.textContent = "Edit";
    edit.addEventListener("click", () => openStageEditor(stage.id));
    const remove = document.createElement("button");
    remove.type = "button";
    remove.className = "danger";
    remove.textContent = "Remove";
    remove.addEventListener("click", () => {
      wf = removeStage(wf, stage.id);
      if (editingStageID === stage.id) closeStageEditor();
      renderAll();
      setStatus(`Removed stage ${stage.id}.`);
    });
    actions.append(edit, remove);
    tr.append(id, title, meta, actions);
    tbody.append(tr);
  }
}

function openStageEditor(id) {
  const stage = wf.stages.find((s) => s.id === id);
  if (!stage) return;
  editingStageID = id;
  $("stage-editor").hidden = false;
  $("stage-editor-id").textContent = id;
  $("edit-stage-title").value = stage.title || "";
  const assignee = stage.assignee_spec || { kind: "open" };
  $("edit-assignee-kind").value = assignee.kind || "open";
  $("edit-assignee-value").value = assignee.role || assignee.actor_id || (assignee.required || [])[0] || "";
  $("edit-evidence").value = (stage.evidence || []).join(", ");
  $("edit-effect-key").value = stage.effect_key || "";
  $("edit-timeout").value = stage.timeout || "";
  $("edit-migration-policy").value = stage.migration_policy || "";
  const delegation = stage.delegation;
  $("edit-delegation-kind").value = delegation?.kind || "";
  $("edit-delegatee").value = delegation?.delegatee || "";
  $("edit-scopes").value = (delegation?.scopes || []).join(", ");
  $("edit-budget").value = String(delegation?.budget || 0);
  $("edit-delegation-evidence").value = (delegation?.evidence || []).join(", ");
  $("edit-stage-title").focus();
}

function closeStageEditor() {
  editingStageID = null;
  $("stage-editor").hidden = true;
}

function renderTransitionSelects() {
  const from = $("transition-from");
  const to = $("transition-to");
  const previousFrom = from.value;
  const previousTo = to.value;
  from.replaceChildren();
  to.replaceChildren();
  for (const stage of wf.stages) {
    const opt = document.createElement("option");
    opt.value = stage.id;
    opt.textContent = stage.id;
    from.append(opt);
    to.append(opt.cloneNode(true));
  }
  if (wf.stages.some((s) => s.id === previousFrom)) from.value = previousFrom;
  if (wf.stages.some((s) => s.id === previousTo)) to.value = previousTo;
}

function renderTransitions() {
  const tbody = $("transition-rows");
  tbody.replaceChildren();
  wf.transitions.forEach((tr, index) => {
    const row = document.createElement("tr");
    const from = document.createElement("td");
    from.textContent = tr.from;
    const to = document.createElement("td");
    to.textContent = tr.to;
    const condition = document.createElement("td");
    condition.textContent = tr.condition || "";
    const actions = document.createElement("td");
    const remove = document.createElement("button");
    remove.type = "button";
    remove.className = "danger";
    remove.textContent = "Remove";
    remove.addEventListener("click", () => {
      wf = removeTransition(wf, index);
      renderAll();
      setStatus("Removed transition.");
    });
    actions.append(remove);
    row.append(from, to, condition, actions);
    tbody.append(row);
  });
}

function renderGraph() {
  const host = $("graph");
  host.replaceChildren();
  if (wf.stages.length === 0) {
    host.textContent = "No stages yet.";
    return;
  }
  const layout = layoutWorkflow(wf);
  const nodeW = 170;
  const nodeH = 48;
  const gapX = 90;
  const gapY = 28;
  const pad = 20;
  const byLayer = new Map();
  for (const node of layout.nodes) {
    if (!byLayer.has(node.layer)) byLayer.set(node.layer, []);
    byLayer.get(node.layer).push(node);
  }
  let maxRows = 1;
  for (const nodes of byLayer.values()) maxRows = Math.max(maxRows, nodes.length);
  const width = pad * 2 + Math.max(1, layout.layerCount) * nodeW + Math.max(0, layout.layerCount - 1) * gapX;
  const height = pad * 2 + maxRows * nodeH + Math.max(0, maxRows - 1) * gapY;
  const svgNS = "http://www.w3.org/2000/svg";
  const svg = document.createElementNS(svgNS, "svg");
  svg.setAttribute("viewBox", `0 0 ${width} ${height}`);
  svg.setAttribute("width", String(width));
  svg.setAttribute("height", String(height));

  const pos = new Map();
  for (const node of layout.nodes) {
    const x = pad + node.layer * (nodeW + gapX);
    const y = pad + node.index * (nodeH + gapY);
    pos.set(node.id, { x, y });
  }

  for (const edge of layout.edges) {
    const a = pos.get(edge.from);
    const b = pos.get(edge.to);
    if (!a || !b) continue;
    const line = document.createElementNS(svgNS, "line");
    line.setAttribute("class", "edge");
    line.setAttribute("x1", String(a.x + nodeW));
    line.setAttribute("y1", String(a.y + nodeH / 2));
    line.setAttribute("x2", String(b.x));
    line.setAttribute("y2", String(b.y + nodeH / 2));
    svg.append(line);
    if (edge.condition) {
      const label = document.createElementNS(svgNS, "text");
      label.setAttribute("class", "edge-label");
      label.setAttribute("x", String((a.x + nodeW + b.x) / 2));
      label.setAttribute("y", String((a.y + b.y) / 2 + nodeH / 2 - 4));
      label.textContent = edge.condition.length > 24 ? edge.condition.slice(0, 21) + "..." : edge.condition;
      svg.append(label);
    }
  }

  for (const node of layout.nodes) {
    const p = pos.get(node.id);
    const group = document.createElementNS(svgNS, "g");
    group.setAttribute("class", "node");
    const rect = document.createElementNS(svgNS, "rect");
    rect.setAttribute("x", String(p.x));
    rect.setAttribute("y", String(p.y));
    rect.setAttribute("width", String(nodeW));
    rect.setAttribute("height", String(nodeH));
    rect.setAttribute("rx", "8");
    const title = document.createElementNS(svgNS, "text");
    title.setAttribute("x", String(p.x + 10));
    title.setAttribute("y", String(p.y + 19));
    title.textContent = node.title.length > 22 ? node.title.slice(0, 19) + "..." : node.title;
    const id = document.createElementNS(svgNS, "text");
    id.setAttribute("x", String(p.x + 10));
    id.setAttribute("y", String(p.y + 36));
    id.setAttribute("opacity", "0.65");
    id.textContent = node.id;
    group.append(rect, title, id);
    svg.append(group);
  }
  host.append(svg);
}

function renderValidation() {
  const list = $("validation");
  list.replaceChildren();
  const errors = validateWorkflow(wf);
  list.classList.toggle("ok", errors.length === 0);
  if (errors.length === 0) {
    const li = document.createElement("li");
    li.textContent = "Workflow is valid.";
    list.append(li);
    return;
  }
  for (const error of errors) {
    const li = document.createElement("li");
    li.textContent = error;
    list.append(li);
  }
}

function syncJSON() {
  $("json-editor").value = serializeWorkflow(wf);
}

function renderAll() {
  renderMeta();
  renderStages();
  renderTransitionSelects();
  renderTransitions();
  renderGraph();
  renderValidation();
  syncJSON();
}

$("workflow-id").addEventListener("input", (e) => {
  wf.workflow_id = e.target.value;
  renderValidation();
  syncJSON();
});
$("workflow-title").addEventListener("input", (e) => {
  wf.title = e.target.value;
  renderValidation();
  syncJSON();
});

$("stage-form").addEventListener("submit", (e) => {
  e.preventDefault();
  try {
    wf = addStage(wf, $("stage-id").value, $("stage-title").value);
    $("stage-id").value = "";
    $("stage-title").value = "";
    renderAll();
    setStatus("Stage added.");
  } catch (err) {
    setStatus(err.message, true);
  }
});

$("stage-editor-form").addEventListener("submit", (e) => {
  e.preventDefault();
  if (!editingStageID) return;
  const assigneeKind = $("edit-assignee-kind").value;
  const assigneeValue = $("edit-assignee-value").value.trim();
  let assignee = { kind: "open" };
  if (assigneeKind === "role") assignee = { kind: "role", role: assigneeValue };
  if (assigneeKind === "capability") assignee = { kind: "capability", required: assigneeValue ? [assigneeValue] : [] };
  if (assigneeKind === "actor") assignee = { kind: "actor", actor_id: assigneeValue };
  const delegationKind = $("edit-delegation-kind").value;
  const delegation = delegationKind
    ? {
        kind: delegationKind,
        delegatee: $("edit-delegatee").value.trim(),
        scopes: splitCSV($("edit-scopes").value),
        budget: Number($("edit-budget").value || 0),
        evidence: splitCSV($("edit-delegation-evidence").value),
      }
    : null;
  try {
    wf = updateStage(wf, editingStageID, {
      title: $("edit-stage-title").value,
      assignee,
      evidence: splitCSV($("edit-evidence").value),
      effectKey: $("edit-effect-key").value.trim(),
      timeout: $("edit-timeout").value.trim(),
      migrationPolicy: $("edit-migration-policy").value,
      delegation,
    });
    const id = editingStageID;
    closeStageEditor();
    renderAll();
    setStatus(`Updated stage ${id}.`);
  } catch (err) {
    setStatus(err.message, true);
  }
});

$("stage-editor-cancel").addEventListener("click", closeStageEditor);

$("transition-form").addEventListener("submit", (e) => {
  e.preventDefault();
  wf = addTransition(wf, $("transition-from").value, $("transition-to").value, $("transition-condition").value);
  $("transition-condition").value = "";
  renderAll();
  setStatus("Transition added.");
});

$("new-workflow").addEventListener("click", () => {
  wf = emptyWorkflow();
  renderAll();
  setStatus("New workflow.");
});

$("load-example").addEventListener("click", () => {
  wf = normalizeWorkflow(example);
  renderAll();
  setStatus("Example loaded.");
});

$("validate-json").addEventListener("click", () => {
  const errors = validateWorkflow(wf);
  renderValidation();
  setStatus(errors.length === 0 ? "Workflow is valid." : `${errors.length} validation error(s).`, errors.length > 0);
});

$("apply-json").addEventListener("click", () => {
  try {
    wf = normalizeWorkflow($("json-editor").value);
    renderAll();
    setStatus("JSON applied.");
  } catch (err) {
    setStatus(`Invalid JSON: ${err.message}`, true);
  }
});

$("core-load").addEventListener("click", async () => {
  const { url, ...opts } = coreOptions();
  const id = $("core-workflow-id").value.trim() || wf.workflow_id;
  try {
    const loaded = await loadWorkflow(url, id, opts);
    wf = normalizeWorkflow(loaded);
    renderAll();
    $("core-workflow-id").value = wf.workflow_id;
    setCoreStatus(`Loaded ${wf.workflow_id}.`);
  } catch (err) {
    setCoreStatus(`Load failed: ${describeError(err)}`, true);
  }
});

$("core-publish").addEventListener("click", async () => {
  const errors = validateWorkflow(wf);
  if (errors.length > 0) {
    renderValidation();
    setCoreStatus(`Cannot publish: ${errors.length} validation error(s).`, true);
    return;
  }
  const { url, ...opts } = coreOptions();
  try {
    const result = await publishWorkflow(url, wf, opts);
    setCoreStatus(`Published ${result.workflow_id}@${result.version}.`);
    $("core-result").textContent = JSON.stringify(result, null, 2);
  } catch (err) {
    setCoreStatus(`Publish failed: ${describeError(err)}`, true);
  }
});

$("core-list").addEventListener("click", async () => {
  const { url, ...opts } = coreOptions();
  try {
    const list = await listWorkflows(url, opts);
    $("core-result").textContent = JSON.stringify(list, null, 2);
    setCoreStatus(`${list.length} workflow(s).`);
  } catch (err) {
    setCoreStatus(`List failed: ${describeError(err)}`, true);
  }
});

$("export-json").addEventListener("click", () => {
  const blob = new Blob([serializeWorkflow(wf)], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `${wf.workflow_id || "workflow"}.json`;
  a.click();
  URL.revokeObjectURL(url);
});

$("import-bpmn").addEventListener("change", async (e) => {
  const file = e.target.files?.[0];
  if (!file) return;
  const { url, ...opts } = coreOptions();
  try {
    const result = await importBPMN(url, await file.text(), opts);
    wf = normalizeWorkflow(result.workflow);
    renderAll();
    $("core-result").textContent = JSON.stringify(
      { report: result.report, validation: result.validation },
      null,
      2,
    );
    const invalid = result.validation && result.validation.valid === false;
    setCoreStatus(
      invalid ? `Imported ${file.name} with validation errors.` : `Imported ${file.name}.`,
      invalid,
    );
  } catch (err) {
    setCoreStatus(`BPMN import failed: ${describeError(err)}`, true);
  }
  e.target.value = "";
});

$("import-json").addEventListener("change", async (e) => {
  const file = e.target.files?.[0];
  if (!file) return;
  try {
    wf = normalizeWorkflow(await file.text());
    renderAll();
    setStatus(`Imported ${file.name}.`);
  } catch (err) {
    setStatus(`Import failed: ${err.message}`, true);
  }
  e.target.value = "";
});

renderAll();
