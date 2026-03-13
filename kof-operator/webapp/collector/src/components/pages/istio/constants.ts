import { NodeObject } from "react-force-graph-2d";
import { GraphNode } from "./types";

export const COLOR: Record<string, string> = {
  Purple: "#6366f1",
  Orange: "#f59e0b",
  Green: "#10b981",
  Gray: "#94a3b8",
  White: "#FFFFFF",
  Black: "#020617",
};

export const ROLE_COLORS: Record<string, string> = {
  management: COLOR.Purple,
  regional: COLOR.Orange,
  child: COLOR.Green,
};

export const roleColor = (role: string): string => ROLE_COLORS[role] ?? COLOR.Gray;

export const NODE_RADIUS = 5;
export const LABEL_OFFSET = NODE_RADIUS + 5;

export const getNodeId = (node: string | number | NodeObject | undefined): string =>
  typeof node === "string"
    ? node
    : typeof node === "number"
      ? String(node)
      : node?.id !== undefined
        ? String(node.id)
        : "";

export function drawNode(
  node: NodeObject,
  ctx: CanvasRenderingContext2D,
  globalScale: number,
) {
  const n = node as GraphNode;
  const x = n.x ?? 0;
  const y = n.y ?? 0;
  const fontSize = Math.max(6 / globalScale, 3);

  ctx.beginPath();
  ctx.arc(x, y, NODE_RADIUS, 0, 2 * Math.PI);
  ctx.fillStyle = roleColor(n.role);
  ctx.fill();

  ctx.font = `${fontSize}px sans-serif`;
  ctx.fillStyle = COLOR.White;
  ctx.textAlign = "center";
  ctx.fillText(n.name, x, y + LABEL_OFFSET);
}

// Stable cache key matching IstioClusterEndpointsProvider.
export const nodeKey = (n: { id: string; namespace: string }): string => `${n.id}:${n.namespace}`;
