import { Empty, Tag, Typography } from "antd";
import { useMemo } from "react";
import {
  CHART_BRAND,
  CHART_ERROR,
  CHART_INFO,
  CHART_MUTED,
  CHART_SECONDARY,
  CHART_SUCCESS,
  CHART_WARNING,
} from "../../constants/chart-colors";

export type TopologyNode = {
  id: string;
  label: string;
  kind: string;
  state?: string;
  state_level?: "normal" | "progressing" | "abnormal" | string;
};

export type TopologyEdge = {
  from: string;
  to: string;
  kind?: string;
};

export type TopologyGraph = {
  nodes: TopologyNode[];
  edges: TopologyEdge[];
};

const KIND_COLORS: Record<string, string> = {
  Project: CHART_BRAND,
  ServerGroup: CHART_SECONDARY,
  Server: CHART_BRAND,
  AppService: CHART_SUCCESS,
  LogSource: CHART_WARNING,
  Deployment: CHART_INFO,
  ReplicaSet: CHART_INFO,
  StatefulSet: CHART_INFO,
  DaemonSet: CHART_INFO,
  Pod: CHART_SUCCESS,
  Service: CHART_BRAND,
  Ingress: CHART_SECONDARY,
};

function borderColor(level?: string): string {
  switch (level) {
    case "abnormal":
      return CHART_ERROR;
    case "progressing":
      return CHART_WARNING;
    default:
      return CHART_INFO;
  }
}

type LayoutNode = TopologyNode & { x: number; y: number; depth: number };

const KIND_DEPTH: Record<string, number> = {
  Ingress: 0,
  Service: 1,
  Deployment: 2,
  StatefulSet: 2,
  DaemonSet: 2,
  ReplicaSet: 3,
  Pod: 4,
};

function assignNodeDepths(graph: TopologyGraph): Map<string, number> {
  const depth = new Map<string, number>();
  const byId = new Map(graph.nodes.map((n) => [n.id, n]));

  for (const n of graph.nodes) {
    if (KIND_DEPTH[n.kind] !== undefined) {
      depth.set(n.id, KIND_DEPTH[n.kind]);
    }
  }

  let changed = true;
  for (let round = 0; round < graph.nodes.length + 2 && changed; round++) {
    changed = false;
    for (const e of graph.edges) {
      const fromD = depth.get(e.from);
      if (fromD === undefined) continue;
      const toNode = byId.get(e.to);
      const kindFloor = toNode ? KIND_DEPTH[toNode.kind] : undefined;
      const next = kindFloor !== undefined ? Math.max(fromD + 1, kindFloor) : fromD + 1;
      if ((depth.get(e.to) ?? -1) < next) {
        depth.set(e.to, next);
        changed = true;
      }
    }
  }

  for (const n of graph.nodes) {
    if (!depth.has(n.id)) {
      depth.set(n.id, KIND_DEPTH[n.kind] ?? 0);
    }
  }
  return depth;
}

function layoutGraph(graph: TopologyGraph | null | undefined): { nodes: LayoutNode[]; edges: TopologyEdge[]; width: number; height: number } {
  if (!graph?.nodes?.length) {
    return { nodes: [], edges: graph?.edges ?? [], width: 640, height: 240 };
  }
  const depth = assignNodeDepths(graph);
  const layers = new Map<number, TopologyNode[]>();
  for (const n of graph.nodes) {
    const d = depth.get(n.id) ?? 0;
    if (!layers.has(d)) layers.set(d, []);
    layers.get(d)!.push(n);
  }
  const nodeW = 148;
  const nodeH = 56;
  const gapX = 48;
  const gapY = 72;
  const maxLayer = Math.max(...Array.from(layers.keys()), 0);
  const maxCount = Math.max(...Array.from(layers.values()).map((v) => v.length), 1);
  const width = Math.max(720, (maxLayer + 1) * (nodeW + gapX) + 80);
  const height = Math.max(280, maxCount * (nodeH + gapY) + 80);
  const nodes: LayoutNode[] = [];
  layers.forEach((list, d) => {
    const totalH = list.length * nodeH + (list.length - 1) * gapY;
    let y = (height - totalH) / 2;
    list.forEach((n) => {
      nodes.push({ ...n, x: 40 + d * (nodeW + gapX), y, depth: d });
      y += nodeH + gapY;
    });
  });
  return { nodes, edges: graph.edges, width, height };
}

type Props = {
  graph: TopologyGraph | null | undefined;
  loading?: boolean;
  emptyText?: string;
};

export function TopologyGraphView({ graph, loading, emptyText = "暂无拓扑数据" }: Props) {
  const layout = useMemo(() => layoutGraph(graph), [graph]);
  const pos = useMemo(() => {
    const m = new Map<string, LayoutNode>();
    for (const n of layout.nodes) m.set(n.id, n);
    return m;
  }, [layout.nodes]);

  if (!loading && !graph?.nodes?.length) {
    return <Empty description={emptyText} />;
  }

  return (
    <div style={{ overflow: "auto", maxHeight: 520, border: "1px solid #f0f0f0", borderRadius: 8, background: "#fafafa" }}>
      <svg width={layout.width} height={layout.height} style={{ display: "block", minWidth: "100%" }}>
        {layout.edges.map((e) => {
          const from = pos.get(e.from);
          const to = pos.get(e.to);
          if (!from || !to) return null;
          const x1 = from.x + 148;
          const y1 = from.y + 28;
          const x2 = to.x;
          const y2 = to.y + 28;
          const mx = (x1 + x2) / 2;
          return (
            <g key={`${e.from}-${e.to}`}>
              <path d={`M${x1} ${y1} C ${mx} ${y1}, ${mx} ${y2}, ${x2} ${y2}`} fill="none" stroke="#bfbfbf" strokeWidth={1.5} markerEnd="url(#arrow)" />
              {e.kind ? (
                <text x={mx} y={(y1 + y2) / 2 - 4} textAnchor="middle" fontSize={10} fill="#8c8c8c">
                  {e.kind}
                </text>
              ) : null}
            </g>
          );
        })}
        <defs>
          <marker id="arrow" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto">
            <path d="M0,0 L6,3 L0,6 Z" fill="#bfbfbf" />
          </marker>
        </defs>
        {layout.nodes.map((n) => (
          <g key={n.id} transform={`translate(${n.x}, ${n.y})`}>
            <rect width={148} height={56} rx={8} fill="#fff" stroke={borderColor(n.state_level)} strokeWidth={2} />
            <text x={10} y={20} fontSize={12} fontWeight={600} fill={KIND_COLORS[n.kind] ?? CHART_MUTED}>
              {n.kind}
            </text>
            <text x={10} y={38} fontSize={11} fill="#262626">
              {n.label.length > 18 ? `${n.label.slice(0, 16)}…` : n.label}
            </text>
            {n.state ? (
              <text x={10} y={50} fontSize={10} fill="#8c8c8c">
                {n.state.length > 20 ? `${n.state.slice(0, 18)}…` : n.state}
              </text>
            ) : null}
          </g>
        ))}
      </svg>
      <div style={{ padding: "8px 12px", borderTop: "1px solid #f0f0f0" }}>
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          图例：
        </Typography.Text>
        <Tag color="blue" style={{ marginLeft: 8 }}>
          正常
        </Tag>
        <Tag color="gold">进行中</Tag>
        <Tag color="red">异常</Tag>
        <Typography.Text type="secondary" style={{ marginLeft: 12, fontSize: 12 }}>
          参考 KubeVela：按 owner/路由关系展示资源层级
        </Typography.Text>
      </div>
    </div>
  );
}
