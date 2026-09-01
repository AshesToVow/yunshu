import { forwardRef, type MouseEvent, type ThHTMLAttributes } from "react";
import { Table, type TableProps } from "antd";
import { MIN_TABLE_COL_WIDTH, useResizableTableColumns } from "../hooks/use-resizable-table-columns";

type ResizableTableProps<T extends object> = TableProps<T> & {
  wrapHost?: boolean;
};

type HeaderCellProps = ThHTMLAttributes<HTMLTableCellElement> & {
  width?: number;
  onResize?: (width: number) => void;
};

const ResizableHeaderCell = forwardRef<HTMLTableCellElement, HeaderCellProps>(function ResizableHeaderCell(
  props,
  ref,
) {
  const { onResize, width, children, style, onClick, ...rest } = props;
  const handleMouseDown = (e: MouseEvent<HTMLSpanElement>) => {
    if (!onResize || width == null) return;
    e.preventDefault();
    e.stopPropagation();
    const startX = e.clientX;
    const startW = width;
    const onMove = (ev: globalThis.MouseEvent) => {
      onResize(Math.max(MIN_TABLE_COL_WIDTH, startW + ev.clientX - startX));
    };
    const onUp = () => {
      document.removeEventListener("mousemove", onMove);
      document.removeEventListener("mouseup", onUp);
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";
    document.addEventListener("mousemove", onMove);
    document.addEventListener("mouseup", onUp);
  };
  return (
    <th
      {...rest}
      ref={ref}
      onClick={onClick}
      style={{ ...style, width, minWidth: width, position: "relative", overflow: "visible" }}
    >
      {children}
      {typeof width === "number" && onResize ? (
        <span
          className="yunshu-table-col-resizer"
          role="separator"
          aria-orientation="vertical"
          aria-label="拖动调整列宽"
          title="拖动调整列宽"
          onMouseDown={handleMouseDown}
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
          }}
        />
      ) : null}
    </th>
  );
});

export function ResizableTable<T extends object>({ wrapHost = true, ...props }: ResizableTableProps<T>) {
  const columns = useResizableTableColumns(props.columns);
  const totalX = columns.reduce((sum, col) => {
    if (col && typeof col === "object" && !Array.isArray(col) && typeof col.width === "number") {
      return sum + col.width;
    }
    return sum;
  }, 0);
  const prevScroll = props.scroll;
  const prevX = prevScroll && typeof prevScroll === "object" && typeof prevScroll.x === "number" ? prevScroll.x : 0;
  const table = (
    <Table<T>
      {...props}
      columns={columns}
      tableLayout="fixed"
      scroll={{
        ...(typeof prevScroll === "object" ? prevScroll : {}),
        x: Math.max(prevX, totalX),
      }}
      components={{
        ...props.components,
        header: {
          ...props.components?.header,
          cell: ResizableHeaderCell,
        },
      }}
    />
  );
  if (!wrapHost) return table;
  return <div className="k8s-table-scroll-host">{table}</div>;
}
