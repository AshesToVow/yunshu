// @ts-nocheck
import { ReloadOutlined } from "@ant-design/icons";
import { Button, Input, Select, Space, Switch } from "antd";
import type { ReactNode } from "react";
import type { ClusterOption, NamespaceOption } from "../k8s/yaml-crud-page";

export type K8sPageToolbarProps = {
  clusterId?: number;
  namespace?: string;
  clusterOptions: ClusterOption[];
  namespaceOptions?: NamespaceOption[];
  needNamespace?: boolean;
  searchPlaceholder?: string;
  onClusterChange: (id: number) => void;
  onNamespaceChange?: (ns: string) => void;
  onSearch?: (keyword: string) => void;
  onRefresh?: () => void;
  watchLive?: boolean;
  onWatchChange?: (enabled: boolean) => void;
  extraLeft?: ReactNode;
  extraRight?: ReactNode;
  primaryAction?: ReactNode;
};

export function K8sPageToolbar({
  clusterId,
  namespace,
  clusterOptions,
  namespaceOptions,
  needNamespace = true,
  searchPlaceholder = "搜索名称",
  onClusterChange,
  onNamespaceChange,
  onSearch,
  onRefresh,
  watchLive,
  onWatchChange,
  extraLeft,
  extraRight,
  primaryAction,
}: K8sPageToolbarProps) {
  return (
    <div className="k8s-page-toolbar">
      <Space wrap className="k8s-page-toolbar__left">
        <Select
          placeholder="选择集群"
          className="k8s-page-toolbar__cluster"
          value={clusterId}
          onChange={onClusterChange}
          options={clusterOptions}
        />
        {needNamespace ? (
          <Select
            placeholder="命名空间"
            className="k8s-page-toolbar__namespace"
            value={namespace}
            onChange={onNamespaceChange}
            options={namespaceOptions}
            showSearch
            optionFilterProp="label"
          />
        ) : null}
        {onSearch ? (
          <Input.Search
            allowClear
            placeholder={searchPlaceholder}
            className="k8s-page-toolbar__search"
            onSearch={onSearch}
          />
        ) : null}
        {extraLeft}
      </Space>
      <Space wrap className="k8s-page-toolbar__right">
        {extraRight}
        {primaryAction}
        {onRefresh ? (
          <Button icon={<ReloadOutlined />} onClick={onRefresh}>
            刷新
          </Button>
        ) : null}
        {onWatchChange ? (
          <Space size={4}>
            <span className="k8s-page-toolbar__watch-label">实时 Watch</span>
            <Switch checked={watchLive} onChange={onWatchChange} />
          </Space>
        ) : null}
      </Space>
    </div>
  );
}
