import { normalizeWorkflow } from "./model.mjs";

function number(value) {
  const n = Number(value);
  return Number.isFinite(n) ? n : 0;
}

/**
 * Normalize BPMN DI shapes and edges into a renderable layout.
 *
 * Returns null when the workflow has no diagram metadata. Coordinates are
 * shifted so the top-left bound starts at `padding`.
 */
export function bpmnDiagramLayout(input, padding = 20) {
  let wf;
  try {
    wf = normalizeWorkflow(input);
  } catch {
    return null;
  }
  const diagram = wf.bpmn && wf.bpmn.diagram;
  if (!diagram) return null;
  const rawShapes = Array.isArray(diagram.shapes) ? diagram.shapes : [];
  const rawEdges = Array.isArray(diagram.edges) ? diagram.edges : [];
  if (rawShapes.length === 0 && rawEdges.length === 0) return null;

  const stageTitles = new Map(wf.stages.map((stage) => [stage.id, stage.title || stage.id]));
  const nodeNames = new Map(
    (Array.isArray(wf.bpmn.nodes) ? wf.bpmn.nodes : []).map((node) => [
      node.id,
      node.name || node.id,
    ]),
  );

  const shapes = rawShapes.map((shape) => {
    const element = shape.bpmn_element || "";
    return {
      id: shape.id,
      element,
      x: number(shape.x),
      y: number(shape.y),
      width: Math.max(0, number(shape.width)),
      height: Math.max(0, number(shape.height)),
      label: stageTitles.get(element) || nodeNames.get(element) || element || shape.id,
    };
  });
  const edges = rawEdges.map((edge) => ({
    id: edge.id,
    element: edge.bpmn_element || "",
    points: (Array.isArray(edge.waypoints) ? edge.waypoints : []).map((point) => ({
      x: number(point.x),
      y: number(point.y),
    })),
  }));

  let minX = Infinity;
  let minY = Infinity;
  let maxX = -Infinity;
  let maxY = -Infinity;
  for (const shape of shapes) {
    minX = Math.min(minX, shape.x);
    minY = Math.min(minY, shape.y);
    maxX = Math.max(maxX, shape.x + shape.width);
    maxY = Math.max(maxY, shape.y + shape.height);
  }
  for (const edge of edges) {
    for (const point of edge.points) {
      minX = Math.min(minX, point.x);
      minY = Math.min(minY, point.y);
      maxX = Math.max(maxX, point.x);
      maxY = Math.max(maxY, point.y);
    }
  }
  if (!Number.isFinite(minX)) {
    minX = 0;
    minY = 0;
    maxX = 0;
    maxY = 0;
  }

  const offsetX = padding - minX;
  const offsetY = padding - minY;
  return {
    width: Math.max(1, maxX - minX + padding * 2),
    height: Math.max(1, maxY - minY + padding * 2),
    shapes: shapes.map((shape) => ({
      ...shape,
      x: shape.x + offsetX,
      y: shape.y + offsetY,
    })),
    edges: edges.map((edge) => ({
      ...edge,
      points: edge.points.map((point) => ({
        x: point.x + offsetX,
        y: point.y + offsetY,
      })),
    })),
  };
}
