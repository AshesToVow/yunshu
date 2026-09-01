// @ts-nocheck
import type { ReactNode } from "react";
import type { TablePaginationConfig } from "antd/es/table";

/** 与后端 `internal/pkg/pagination.Normalize` 默认一致：默认 10，上限 100 */
export const DEFAULT_PAGE_SIZE = 10;

/** 列表表格统一可选：10 / 20 / 50 / 100 条每页 */
export const PAGE_SIZE_OPTIONS = ["10", "20", "50", "100"] as const;

export type TablePaginationInput = {
  current?: number;
  pageSize?: number;
  total?: number;
  onChange?: (page: number, pageSize: number) => void;
  /** 默认 true */
  showQuickJumper?: boolean;
  /** 默认展示「共 N 条」；传 false 关闭；或自定义 */
  showTotal?: boolean | ((total: number, range: [number, number]) => ReactNode);
};

/**
 * Ant Design Table 标准分页配置，避免各页重复写 pageSizeOptions。
 * 服务端分页时传 current / total / onChange；纯前端分页只传 pageSize 即可。
 */
export function tablePagination(opts: TablePaginationInput = {}): TablePaginationConfig {
  const showTotal =
    opts.showTotal === false
      ? undefined
      : typeof opts.showTotal === "function"
        ? opts.showTotal
        : ((t: number) => `共 ${t} 条`);

  return {
    current: opts.current,
    pageSize: opts.pageSize ?? DEFAULT_PAGE_SIZE,
    total: opts.total,
    showSizeChanger: true,
    pageSizeOptions: [...PAGE_SIZE_OPTIONS],
    showQuickJumper: opts.showQuickJumper ?? true,
    showTotal,
    onChange: opts.onChange,
  };
}
