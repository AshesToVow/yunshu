import { useEffect, useMemo, useRef, useState } from "react";

export interface BarChartItem {
  label: string;
  value: number;
  color?: string;
}

interface BarChartProps {
  items: BarChartItem[];
  height?: number;
  darkMode?: boolean;
  valueLabel?: string;
}

export function BarChart({ items, height = 320, darkMode = false, valueLabel = "数量" }: BarChartProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const [viewWidth, setViewWidth] = useState(700);
  const padding = { top: 24, right: 16, bottom: 72, left: 40 };
  const plotH = height - padding.top - padding.bottom;
  const maxValue = Math.max(...items.map((it) => it.value), 1);
  const barGap = 12;
  const minBarWidth = 28;
  const contentWidth = Math.max(viewWidth, items.length * (minBarWidth + barGap) + padding.left + padding.right);

  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    const resize = () => setViewWidth(Math.max(360, Math.floor(el.clientWidth || 700)));
    resize();
    const observer = new ResizeObserver(resize);
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  const bars = useMemo(() => {
    const plotW = contentWidth - padding.left - padding.right;
    const barWidth = items.length > 0 ? Math.min(48, Math.max(minBarWidth, plotW / items.length - barGap)) : minBarWidth;
    const totalBarsWidth = items.length * barWidth + Math.max(0, items.length-1) * barGap;
    const startX = padding.left + Math.max(0, (plotW - totalBarsWidth) / 2);
    return items.map((item, i) => {
      const barH = (item.value / maxValue) * plotH;
      const x = startX + i * (barWidth + barGap);
      const y = padding.top + plotH - barH;
      return { ...item, x, y, barH, barWidth };
    });
  }, [contentWidth, items, maxValue, plotH]);

  const gridColor = darkMode ? "rgba(56, 189, 248, 0.14)" : "#eef3fb";
  const textColor = darkMode ? "#94a3b8" : "#64748b";
  const valueColor = darkMode ? "#f8fafc" : "#0f172a";
  const defaultColor = "#3b82f6";

  return (
    <div ref={containerRef} style={{ width: "100%", overflowX: "auto" }}>
      <svg viewBox={`0 0 ${contentWidth} ${height}`} width={contentWidth} height={height} style={{ display: "block" }}>
        {[0, 0.25, 0.5, 0.75, 1].map((t) => {
          const y = padding.top + t * plotH;
          const val = Math.round(maxValue * (1 - t));
          return (
            <g key={t}>
              <line x1={padding.left} x2={contentWidth - padding.right} y1={y} y2={y} stroke={gridColor} strokeWidth={1} />
              <text x={padding.left - 8} y={y + 4} textAnchor="end" fontSize={11} fill={textColor}>
                {val}
              </text>
            </g>
          );
        })}
        {bars.map((bar, idx) => (
          <g key={`${bar.label}-${idx}`}>
            <rect
              x={bar.x}
              y={bar.y}
              width={bar.barWidth}
              height={Math.max(bar.barH, bar.value > 0 ? 4 : 0)}
              rx={4}
              fill={bar.color || defaultColor}
              opacity={0.92}
            />
            {bar.value > 0 ? (
              <text x={bar.x + bar.barWidth / 2} y={bar.y - 8} textAnchor="middle" fontSize={12} fontWeight={700} fill={valueColor}>
                {bar.value}
              </text>
            ) : null}
            <text
              x={bar.x + bar.barWidth / 2}
              y={height - padding.bottom + 18}
              textAnchor="middle"
              fontSize={11}
              fill={textColor}
              transform={`rotate(-24, ${bar.x + bar.barWidth / 2}, ${height - padding.bottom + 18})`}
            >
              {bar.label}
            </text>
          </g>
        ))}
        <text x={padding.left} y={16} fontSize={11} fill={textColor}>
          {valueLabel}
        </text>
      </svg>
    </div>
  );
}
