/** Semantic colors for charts / topology — aligned with DESIGN.md */
export const CHART_BRAND = "#0d9488";
export const CHART_INFO = "#0958d9";
export const CHART_SUCCESS = "#389e0d";
export const CHART_WARNING = "#d48806";
export const CHART_ERROR = "#cf1322";
export const CHART_MUTED = "#64748b";
export const CHART_SECONDARY = "#0f766e";

export const TOPOLOGY_NODE_COLORS: Record<string, string> = {
  Project: CHART_BRAND,
  ServerGroup: CHART_SECONDARY,
  Deployment: CHART_INFO,
  Ingress: CHART_SECONDARY,
};

export function topologyColorForKind(kind: string): string {
  return TOPOLOGY_NODE_COLORS[kind] ?? CHART_BRAND;
}
