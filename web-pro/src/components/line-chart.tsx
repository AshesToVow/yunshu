// @ts-nocheck
import { useEffect, useMemo, useRef, useState } from "react";

export interface LineSeries {
  name?: string;
  data: number[];
  color?: string;
}

interface LineChartProps {
  data?: number[];
  series?: LineSeries[];
  labels?: string[];
  height?: number;
  showLegend?: boolean;
  darkMode?: boolean;
  /** Y 轴标题，默认「数量」 */
  yAxisLabel?: string;
}

const PLOT = { top: 28, right: 20, bottom: 52, left: 44 };
const MIN_POINT_GAP = 28;

function xTickIndices(n: number, maxTicks = 10): number[] {
  if (n <= 0) return [];
  if (n <= maxTicks) return Array.from({ length: n }, (_, i) => i);
  const step = Math.max(1, Math.ceil((n - 1) / (maxTicks - 1)));
  const ticks: number[] = [];
  for (let i = 0; i < n; i += step) ticks.push(i);
  if (ticks[ticks.length - 1] !== n - 1) ticks.push(n - 1);
  return ticks;
}

export function LineChart({
  data,
  series,
  labels,
  height = 220,
  showLegend = true,
  darkMode = false,
  yAxisLabel = "数量",
}: LineChartProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const svgRef = useRef<SVGSVGElement | null>(null);
  const [viewWidth, setViewWidth] = useState(700);
  const [hoverIndex, setHoverIndex] = useState<number | null>(null);
  const [hoverX, setHoverX] = useState(0);
  const [hoverClient, setHoverClient] = useState<{ x: number; y: number } | null>(null);

  const normalizedSeries: LineSeries[] =
    series && series.length > 0
      ? series
      : [{ name: undefined, data: data ?? [], color: "#2563eb" }];

  const nPoints = useMemo(() => Math.max(...normalizedSeries.map((s) => s.data.length), 0), [normalizedSeries]);

  const safeLabels = useMemo(() => {
    if (!labels?.length || nPoints <= 0) return labels;
    return labels;
  }, [labels, nPoints]);

  const allValues = normalizedSeries.flatMap((s) => s.data);
  const max = Math.max(...allValues, 1);
  const min = Math.min(...allValues, 0);

  const contentWidth = Math.max(viewWidth, nPoints * MIN_POINT_GAP + PLOT.left + PLOT.right);
  const plotW = contentWidth - PLOT.left - PLOT.right;
  const plotH = height - PLOT.top - PLOT.bottom;

  const gridColor = darkMode ? "rgba(56, 189, 248, 0.14)" : "#eef3fb";
  const textColor = darkMode ? "#94a3b8" : "#64748b";
  const valueColor = darkMode ? "#f8fafc" : "#0f172a";

  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    const resize = () => setViewWidth(Math.max(360, Math.floor(el.clientWidth || 700)));
    resize();
    const observer = new ResizeObserver(resize);
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  function toXY(v: number, i: number, n: number) {
    const x = PLOT.left + (i * plotW) / Math.max(1, n - 1);
    const y = PLOT.top + ((max - v) * plotH) / Math.max(1, max - min);
    return { x, y };
  }

  function indexToX(i: number, n: number) {
    return PLOT.left + (i * plotW) / Math.max(1, n - 1);
  }

  function nearestIndexFromX(x: number, n: number) {
    if (n <= 1) return 0;
    const clamped = Math.max(PLOT.left, Math.min(PLOT.left + plotW, x));
    const t = (clamped - PLOT.left) / Math.max(1, plotW);
    return Math.max(0, Math.min(n - 1, Math.round(t * (n - 1))));
  }

  const xTicks = useMemo(() => xTickIndices(nPoints), [nPoints]);

  const tooltipLeft = useMemo(() => {
    if (!hoverClient || !containerRef.current) return 0;
    const cw = containerRef.current.clientWidth;
    const tw = 240;
    let left = hoverClient.x + 12;
    if (left + tw > cw - 8) left = hoverClient.x - tw - 12;
    return Math.max(8, Math.min(left, cw - tw - 8));
  }, [hoverClient]);

  return (
    <div ref={containerRef} style={{ width: "100%", overflowX: "auto", overflowY: "hidden", position: "relative" }}>
      {showLegend && normalizedSeries.length > 1 ? (
        <div style={{ display: "flex", gap: 12, flexWrap: "wrap", padding: "0 2px 10px" }}>
          {normalizedSeries.map((s, idx) => {
            const color = s.color || ["#2563eb", "#10b981", "#f59e0b", "#ef4444"][idx % 4];
            return (
              <div
                key={idx}
                style={{ display: "flex", alignItems: "center", gap: 8, fontSize: 12, color: darkMode ? "#94a3b8" : "#475569" }}
              >
                <span style={{ width: 10, height: 10, borderRadius: 999, background: color }} />
                <span>{s.name || `Series ${idx + 1}`}</span>
              </div>
            );
          })}
        </div>
      ) : null}

      {hoverIndex !== null && hoverClient ? (
        <div
          style={{
            position: "absolute",
            left: tooltipLeft,
            top: Math.max(8, hoverClient.y - 8),
            transform: "translateY(-100%)",
            pointerEvents: "none",
            zIndex: 10,
            background: darkMode ? "rgba(15, 23, 42, 0.92)" : "rgba(255,255,255,0.98)",
            border: darkMode ? "1px solid rgba(56, 189, 248, 0.25)" : "1px solid #e6eefc",
            borderRadius: 10,
            padding: "10px 12px",
            boxShadow: "0 10px 30px rgba(15, 23, 42, 0.12)",
            minWidth: 200,
            maxWidth: 280,
          }}
        >
          <div style={{ fontSize: 12, color: darkMode ? "#cbd5e1" : "#475569", marginBottom: 6, fontWeight: 600 }}>
            {safeLabels?.[Math.min(hoverIndex, (safeLabels?.length ?? 1) - 1)] ?? `#${hoverIndex + 1}`}
          </div>
          <div style={{ display: "grid", gap: 6 }}>
            {normalizedSeries.map((s, sIdx) => {
              const color = s.color || ["#2563eb", "#10b981", "#f59e0b", "#ef4444"][sIdx % 4];
              const v = s.data[hoverIndex];
              return (
                <div key={sIdx} style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 12 }}>
                  <div style={{ display: "flex", alignItems: "center", gap: 8, minWidth: 100 }}>
                    <span style={{ width: 8, height: 8, borderRadius: 999, background: color }} />
                    <span style={{ fontSize: 12, color: darkMode ? "#94a3b8" : "#334155" }}>{s.name || `Series ${sIdx + 1}`}</span>
                  </div>
                  <span style={{ fontSize: 13, color: valueColor, fontWeight: 700 }}>{typeof v === "number" ? v : "-"}</span>
                </div>
              );
            })}
          </div>
        </div>
      ) : null}

      <svg
        ref={svgRef}
        viewBox={`0 0 ${contentWidth} ${height}`}
        width={contentWidth}
        height={height}
        style={{ display: "block", cursor: nPoints > 0 ? "crosshair" : "default" }}
        onMouseLeave={() => {
          setHoverIndex(null);
          setHoverClient(null);
        }}
        onMouseMove={(e) => {
          if (!svgRef.current || nPoints <= 0) return;
          const rect = svgRef.current.getBoundingClientRect();
          const px = e.clientX - rect.left;
          const py = e.clientY - rect.top;
          const vx = (px / Math.max(1, rect.width)) * contentWidth;
          const idx = nearestIndexFromX(vx, nPoints);
          setHoverIndex(idx);
          setHoverX(indexToX(idx, nPoints));
          const containerRect = containerRef.current?.getBoundingClientRect();
          const offsetX = containerRect ? e.clientX - containerRect.left : px;
          setHoverClient({ x: offsetX, y: py + (showLegend && normalizedSeries.length > 1 ? 26 : 0) });
        }}
      >
        {/* Y 轴网格 + 刻度 */}
        {[0, 0.25, 0.5, 0.75, 1].map((t) => {
          const y = PLOT.top + t * plotH;
          const val = Math.round(max * (1 - t) + min * t);
          return (
            <g key={t}>
              <line x1={PLOT.left} x2={contentWidth - PLOT.right} y1={y} y2={y} stroke={gridColor} strokeWidth={1} />
              <text x={PLOT.left - 8} y={y + 4} textAnchor="end" fontSize={11} fill={textColor}>
                {val}
              </text>
            </g>
          );
        })}

        {/* Y 轴标题 */}
        <text x={12} y={PLOT.top + plotH / 2} fontSize={11} fill={textColor} transform={`rotate(-90, 12, ${PLOT.top + plotH / 2})`} textAnchor="middle">
          {yAxisLabel}
        </text>

        {/* X 轴时间刻度 */}
        {xTicks.map((i) => {
          const x = indexToX(i, nPoints);
          const label = safeLabels?.[i] ?? `${i + 1}`;
          return (
            <g key={`x-${i}`}>
              <line x1={x} x2={x} y1={PLOT.top + plotH} y2={PLOT.top + plotH + 4} stroke={textColor} strokeWidth={1} opacity={0.5} />
              <text
                x={x}
                y={height - 10}
                textAnchor="middle"
                fontSize={10}
                fill={textColor}
                transform={nPoints > 12 ? `rotate(-35, ${x}, ${height - 10})` : undefined}
              >
                {label}
              </text>
            </g>
          );
        })}

        {/* X 轴基线 */}
        <line
          x1={PLOT.left}
          x2={contentWidth - PLOT.right}
          y1={PLOT.top + plotH}
          y2={PLOT.top + plotH}
          stroke={darkMode ? "rgba(56, 189, 248, 0.25)" : "#cbd5e1"}
          strokeWidth={1}
        />

        {hoverIndex !== null && nPoints > 0 ? (
          <line
            x1={hoverX}
            x2={hoverX}
            y1={PLOT.top}
            y2={PLOT.top + plotH}
            stroke={darkMode ? "rgba(56, 189, 248, 0.35)" : "#c7d7f7"}
            strokeWidth={1}
            strokeDasharray="4 4"
          />
        ) : null}

        {normalizedSeries.map((s, sIdx) => {
          const color = s.color || ["#2563eb", "#10b981", "#f59e0b", "#ef4444"][sIdx % 4];
          const pts = s.data.map((v, i) => {
            const p = toXY(v, i, s.data.length);
            return `${p.x},${p.y}`;
          }).join(" ");

          return (
            <g key={sIdx}>
              <polyline fill="none" stroke={color} strokeWidth={2} strokeLinejoin="round" strokeLinecap="round" points={pts} />
              {s.data.map((v, idx) => {
                const p = toXY(v, idx, s.data.length);
                const isHover = hoverIndex === idx;
                const showLabel = isHover || (v > 0 && (idx === s.data.length - 1 || nPoints <= 15));
                return (
                  <g key={idx}>
                    <circle cx={p.x} cy={p.y} r={isHover ? 4.5 : 3.5} fill={color} opacity={isHover ? 1 : 0.9} />
                    {isHover ? <circle cx={p.x} cy={p.y} r={8} fill={color} opacity={0.15} /> : null}
                    {showLabel && v > 0 ? (
                      <text x={p.x} y={p.y - 10} textAnchor="middle" fontSize={11} fontWeight={700} fill={valueColor}>
                        {v}
                      </text>
                    ) : null}
                  </g>
                );
              })}
            </g>
          );
        })}
      </svg>
    </div>
  );
}

export default LineChart;
