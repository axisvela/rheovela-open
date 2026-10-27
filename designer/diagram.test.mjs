import test from "node:test";
import assert from "node:assert/strict";

import { bpmnDiagramLayout } from "./diagram.mjs";

function workflowWithDiagram(diagram, extra = {}) {
  return {
    workflow_id: "di",
    title: "DI",
    source: "bpmn",
    stages: [{ id: "t1", title: "Review", inputs: [], outputs: [] }],
    transitions: [],
    bpmn: {
      nodes: [{ id: "t1", element: "task", name: "Review task" }],
      diagram,
      ...extra,
    },
  };
}

test("missing diagram metadata returns null", () => {
  assert.equal(bpmnDiagramLayout({ workflow_id: "x", title: "X", source: "designer", stages: [] }), null);
  assert.equal(bpmnDiagramLayout({ bpmn: { diagram: { shapes: [], edges: [] } } }), null);
});

test("shapes and edges are normalized and offset into the viewport", () => {
  const layout = bpmnDiagramLayout(
    workflowWithDiagram({
      shapes: [
        { id: "s1", bpmn_element: "t1", x: 100, y: 50, width: 120, height: 60 },
        { id: "s2", bpmn_element: "end", x: 400, y: 200, width: 36, height: 36 },
      ],
      edges: [
        {
          id: "e1",
          bpmn_element: "f1",
          waypoints: [
            { x: 220, y: 80 },
            { x: 400, y: 218 },
          ],
        },
      ],
    }),
  );
  assert.ok(layout);
  assert.equal(layout.shapes[0].x, 20);
  assert.equal(layout.shapes[0].y, 20);
  assert.equal(layout.shapes[0].label, "Review");
  assert.equal(layout.shapes[1].x, 320);
  assert.equal(layout.shapes[1].y, 170);
  assert.equal(layout.edges[0].points[0].x, 140);
  assert.equal(layout.edges[0].points[1].x, 320);
  assert.ok(layout.width >= 356);
  assert.ok(layout.height >= 206);
});

test("missing node names fall back to element ids and invalid numbers to zero", () => {
  const layout = bpmnDiagramLayout(
    workflowWithDiagram({
      shapes: [{ id: "s1", bpmn_element: "ghost", x: "x", y: null, width: -10, height: "5" }],
      edges: [],
    }),
  );
  assert.ok(layout);
  assert.equal(layout.shapes[0].label, "ghost");
  assert.equal(layout.shapes[0].x, 20);
  assert.equal(layout.shapes[0].width, 0);
  assert.equal(layout.shapes[0].height, 5);
});
