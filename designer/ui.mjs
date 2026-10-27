import { CoreAPIError, importBPMN, listWorkflows, loadWorkflow, publishWorkflow } from "./core.mjs";
import { bpmnDiagramLayout } from "./diagram.mjs";
import { createHistory } from "./history.mjs";
import {
  addStage,
  addTransition,
  diffWorkflows,
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
const history = createHistory(wf);
let editingStageID = null;
let pendingImport = null;

function commit(next, message, isError = false) {
  wf = next;
  history.push(wf);
  renderAll();
  setStatus(message, isError);
}

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
      if (editingStageID === stage.id) closeStageEditor();
      commit(removeStage(wf, stage.id), `Removed stage ${stage.id}.`);
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
      commit(removeTransition(wf, index), "Removed transition.");
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

function renderHistoryButtons() {
  $("undo").disabled = !history.canUndo();
  $("redo").disabled = !history.canRedo();
}

function undo() {
  if (!history.canUndo()) return;
  closeStageEditor();
  wf = history.undo();
  renderAll();
  setStatus("Undo.");
}

function redo() {
  if (!history.canRedo()) return;
  closeStageEditor();
  wf = history.redo();
  renderAll();
  setStatus("Redo.");
}

function renderBPMNDiagram() {
  const panel = $("bpmn-di-panel");
  const host = $("bpmn-diagram");
  const layout = bpmnDiagramLayout(wf);
  host.replaceChildren();
  if (!layout) {
    panel.hidden = true;
    return;
  }
  panel.hidden = false;
  const svgNS = "http://www.w3.org/2000/svg";
  const svg = document.createElementNS(svgNS, "svg");
  svg.setAttribute("viewBox", `0 0 ${layout.width} ${layout.height}`);
  svg.setAttribute("width", String(Math.min(layout.width, 1100)));
  svg.setAttribute("height", String(Math.min(layout.height, 700)));
  for (const edge of layout.edges) {
    if (edge.points.length < 2) continue;
    const line = document.createElementNS(svgNS, "polyline");
    line.setAttribute("class", "di-edge");
    line.setAttribute("points", edge.points.map((point) => `${point.x},${point.y}`).join(" "));
    svg.append(line);
  }
  for (const shape of layout.shapes) {
    const group = document.createElementNS(svgNS, "g");
    group.setAttribute("class", `di-shape ${shape.element}`);
    const rect = document.createElementNS(svgNS, "rect");
    rect.setAttribute("x", String(shape.x));
    rect.setAttribute("y", String(shape.y));
    rect.setAttribute("width", String(shape.width));
    rect.setAttribute("height", String(shape.height));
    rect.setAttribute("rx", "6");
    group.append(rect);
    if (shape.width >= 30) {
      const label = document.createElementNS(svgNS, "text");
      label.setAttribute("x", String(shape.x + 6));
      label.setAttribute("y", String(shape.y + Math.min(18, Math.max(12, shape.height / 2))));
      label.textContent =
        shape.label.length > 28 ? `${shape.label.slice(0, 25)}...` : shape.label;
      group.append(label);
    }
    svg.append(group);
  }
  host.append(svg);
}

function renderImportPreview() {
  const panel = $("import-preview");
  const host = $("import-diff");
  if (!pendingImport) {
    panel.hidden = true;
    host.replaceChildren();
    return;
  }
  panel.hidden = false;
  const diff = diffWorkflows(wf, pendingImport.workflow);
  pendingImport.diff = diff;
  host.replaceChildren();

  const source = document.createElement("p");
  source.className = "muted";
  source.textContent = pendingImport.source
    ? `Pending import: ${pendingImport.source}`
    : "Pending import";
  host.append(source);
  if (diff.empty) {
    const empty = document.createElement("p");
    empty.textContent = "No changes.";
    host.append(empty);
    return;
  }

  const list = document.createElement("ul");
  const add = (text, className) => {
    const li = document.createElement("li");
    li.textContent = text;
    if (className) li.className = className;
    list.append(li);
  };
  for (const change of diff.workflow) {
    add(
      `workflow ${change.field}: ${JSON.stringify(change.before)} -> ${JSON.stringify(change.after)}`,
    );
  }
  for (const id of diff.stages.added) add(`+ stage ${id}`, "diff-add");
  for (const id of diff.stages.removed) add(`- stage ${id}`, "diff-remove");
  for (const change of diff.stages.changed) {
    add(`~ stage ${change.id} (${change.fields.join(", ")})`, "diff-change");
  }
  for (const key of diff.transitions.added) add(`+ transition ${key}`, "diff-add");
  for (const key of diff.transitions.removed) add(`- transition ${key}`, "diff-remove");
  for (const key of diff.compensations.added) add(`+ compensation ${key}`, "diff-add");
  for (const key of diff.compensations.removed) add(`- compensation ${key}`, "diff-remove");

  const bpmnBits = [];
  if (diff.bpmn.before !== diff.bpmn.after) {
    bpmnBits.push(`BPMN metadata present: ${diff.bpmn.before} -> ${diff.bpmn.after}`);
  }
  for (const [label, value] of [
    ["nodes", diff.bpmn.nodes],
    ["shapes", diff.bpmn.shapes],
    ["edges", diff.bpmn.edges],
    ["participants", diff.bpmn.participants],
    ["choreographies", diff.bpmn.choreographies],
    ["data associations", diff.bpmn.dataAssociations],
  ]) {
    if (value.added.length) bpmnBits.push(`+ ${label}: ${value.added.join(", ")}`);
    if (value.removed.length) bpmnBits.push(`- ${label}: ${value.removed.join(", ")}`);
  }
  if (bpmnBits.length) add(`BPMN: ${bpmnBits.join("; ")}`);
  host.append(list);
}

function previewImport(workflow, meta = {}) {
  pendingImport = {
    workflow: normalizeWorkflow(workflow),
    source: meta.source || "",
    meta,
  };
  renderImportPreview();
}

function clearImportPreview() {
  pendingImport = null;
  $("import-preview").hidden = true;
  $("import-diff").replaceChildren();
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
  renderBPMNDiagram();
  renderImportPreview();
  renderValidation();
  syncJSON();
  renderHistoryButtons();
}

function recordTextEdit() {
  if (JSON.stringify(history.current()) === JSON.stringify(wf)) return;
  history.push(wf);
  renderHistoryButtons();
}

$("workflow-id").addEventListener("input", (e) => {
  wf.workflow_id = e.target.value;
  renderValidation();
  syncJSON();
});
$("workflow-id").addEventListener("change", recordTextEdit);
$("workflow-title").addEventListener("input", (e) => {
  wf.title = e.target.value;
  renderValidation();
  syncJSON();
});
$("workflow-title").addEventListener("change", recordTextEdit);

$("stage-form").addEventListener("submit", (e) => {
  e.preventDefault();
  try {
    const next = addStage(wf, $("stage-id").value, $("stage-title").value);
    $("stage-id").value = "";
    $("stage-title").value = "";
    commit(next, "Stage added.");
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
    const next = updateStage(wf, editingStageID, {
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
    commit(next, `Updated stage ${id}.`);
  } catch (err) {
    setStatus(err.message, true);
  }
});

$("stage-editor-cancel").addEventListener("click", closeStageEditor);
$("undo").addEventListener("click", undo);
$("redo").addEventListener("click", redo);
$("apply-import").addEventListener("click", () => {
  if (!pendingImport) return;
  const pending = pendingImport;
  clearImportPreview();
  commit(pending.workflow, `Applied import${pending.source ? ` from ${pending.source}` : ""}.`);
});
$("discard-import").addEventListener("click", () => {
  const source = pendingImport?.source;
  clearImportPreview();
  setStatus(source ? `Discarded import from ${source}.` : "Discarded import.");
});

$("transition-form").addEventListener("submit", (e) => {
  e.preventDefault();
  const next = addTransition(wf, $("transition-from").value, $("transition-to").value, $("transition-condition").value);
  $("transition-condition").value = "";
  commit(next, "Transition added.");
});

$("new-workflow").addEventListener("click", () => {
  wf = emptyWorkflow();
  history.reset(wf);
  clearImportPreview();
  renderAll();
  setStatus("New workflow.");
});

$("load-example").addEventListener("click", () => {
  commit(normalizeWorkflow(example), "Example loaded.");
});

$("validate-json").addEventListener("click", () => {
  const errors = validateWorkflow(wf);
  renderValidation();
  setStatus(errors.length === 0 ? "Workflow is valid." : `${errors.length} validation error(s).`, errors.length > 0);
});

$("apply-json").addEventListener("click", () => {
  try {
    commit(normalizeWorkflow($("json-editor").value), "JSON applied.");
  } catch (err) {
    setStatus(`Invalid JSON: ${err.message}`, true);
  }
});

$("core-load").addEventListener("click", async () => {
  const { url, ...opts } = coreOptions();
  const id = $("core-workflow-id").value.trim() || wf.workflow_id;
  try {
    const loaded = await loadWorkflow(url, id, opts);
    commit(normalizeWorkflow(loaded), `Loaded ${$("core-workflow-id").value || id}.`);
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
    previewImport(result.workflow, {
      source: file.name,
      report: result.report,
      validation: result.validation,
    });
    $("core-result").textContent = JSON.stringify(
      { report: result.report, validation: result.validation },
      null,
      2,
    );
    const invalid = result.validation && result.validation.valid === false;
    setCoreStatus(
      invalid
        ? `Previewed ${file.name} with validation errors. Review the diff before applying.`
        : `Previewed ${file.name}. Review the diff and apply.`,
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
    previewImport(await file.text(), { source: file.name });
    setStatus(`Previewed ${file.name}. Review the diff and apply.`);
  } catch (err) {
    setStatus(`Import failed: ${err.message}`, true);
  }
  e.target.value = "";
});

document.addEventListener("keydown", (e) => {
  const tag = (e.target?.tagName || "").toLowerCase();
  if (tag === "input" || tag === "textarea" || tag === "select") return;
  if (!(e.ctrlKey || e.metaKey)) return;
  const key = e.key.toLowerCase();
  if (key === "z") {
    e.preventDefault();
    if (e.shiftKey) redo();
    else undo();
  } else if (key === "y") {
    e.preventDefault();
    redo();
  }
});

renderAll();
