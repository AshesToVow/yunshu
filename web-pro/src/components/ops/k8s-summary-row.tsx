// @ts-nocheck
export type K8sSummaryItem = {
  label: string;
  value: number | string;
  accent?: string;
};

export type K8sSummaryRowProps = {
  items: K8sSummaryItem[];
};

export function K8sSummaryRow({ items }: K8sSummaryRowProps) {
  if (!items.length) return null;
  return (
    <div className="k8s-summary-row">
      {items.map((item) => (
        <div key={item.label} className="k8s-summary-row__item">
          <span className="k8s-summary-row__label">{item.label}</span>
          <span className="k8s-summary-row__value" style={item.accent ? { color: item.accent } : undefined}>
            {item.value}
          </span>
        </div>
      ))}
    </div>
  );
}
