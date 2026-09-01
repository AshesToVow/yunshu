// @ts-nocheck
import { useMemo, useState } from "react";
import type { ColumnsType, ColumnType } from "antd/es/table";

export const MIN_TABLE_COL_WIDTH = 72;

function columnKey<T>(col: ColumnType<T>, index: number): string {
  if (col.key != null) return String(col.key);
  if (col.dataIndex != null) {
    return Array.isArray(col.dataIndex) ? col.dataIndex.join(".") : String(col.dataIndex);
  }
  return `col-${index}`;
}

export function useResizableTableColumns<T>(columns: ColumnsType<T> | undefined): ColumnsType<T> {
  const [widths, setWidths] = useState<Record<string, number>>({});
  return useMemo(() => {
    return (columns ?? []).map((col, index) => {
      if (!col || typeof col !== "object" || Array.isArray(col) || "children" in col) {
        return col;
      }
      const typed = col as ColumnType<T>;
      const key = columnKey(typed, index);
      const baseWidth = typeof typed.width === "number" ? typed.width : 160;
      const width = widths[key] ?? baseWidth;
      return {
        ...typed,
        width,
        onHeaderCell: (column) => ({
          ...typed.onHeaderCell?.(column as never),
          width,
          onResize: (next: number) => setWidths((prev) => ({ ...prev, [key]: next })),
        }),
      };
    }) as ColumnsType<T>;
  }, [columns, widths]);
}
