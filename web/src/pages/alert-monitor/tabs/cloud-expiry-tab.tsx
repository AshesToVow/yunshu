import {
  Alert,
  Button,
  Card,
  Collapse,
  Input,
  Radio,
  Segmented,
  Select,
  Space,
  Table,
  Typography,
} from "antd";
import { PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { lazy, Suspense } from "react";
import { useAlertMonitor } from "../context";
import { tablePagination } from "../../../utils/table-pagination";


export function CloudExpiryTab() {
  const ctx = useAlertMonitor();
  return (
<Space direction="vertical" style={{ width: "100%" }} size="middle">
                <Space>
                  <Button type="primary" icon={<PlusOutlined />} onClick={ctx.openCloudExpiryCreate}>
                    新建云到期规则
                  </Button>
                  <Select
                    style={{ width: 160 }}
                    allowClear
                    placeholder="厂商筛选"
                    value={ctx.cloudExpiryProviderFilter || undefined}
                    onChange={(v) => ctx.setCloudExpiryProviderFilter(String(v || ""))}
                    options={[
                      { label: "全部厂商", value: "" },
                      { label: "阿里云", value: "alibaba" },
                      { label: "腾讯云", value: "tencent" },
                      { label: "京东云", value: "jd" },
                    ]}
                  />
                  <Input.Search
                    style={{ width: 240 }}
                    allowClear
                    placeholder="按规则名/地域搜索"
                    value={ctx.cloudExpiryKeyword}
                    onChange={(e) => ctx.setCloudExpiryKeyword(e.target.value)}
                    onSearch={(v) => ctx.setCloudExpiryKeyword(String(v || "").trim())}
                  />
                  <Button icon={<ReloadOutlined />} onClick={() => void ctx.loadCloudExpiryRules(ctx.projectContextId, ctx.cloudExpiryProviderFilter, ctx.cloudExpiryKeyword)}>
                    刷新
                  </Button>
                  <Button type="primary" loading={ctx.cloudExpiryEvaluating} onClick={() => void ctx.runCloudExpiryEvalNow()}>
                    立即执行一次评估
                  </Button>
                </Space>
                <Alert
                  type="info"
                  showIcon
                  message="云到期规则说明"
                  description="后台在「启用定时自动评估」且已填写 Cron 时拉取云实例到期时间：① 告警服务约每分钟检查一次是否到达各规则配置的 Cron（如 0 */2 * * * 表示每 2 小时整点评估一次）；② 与 YAML 中 alert.monitor_eval_cron_spec（仅用于内置 PromQL 监控规则）无关；③ 仅当剩余天数 ≤ 提前天数才 firing。须配置 security.encryption_key 且云账号可解密。订阅按 labels 匹配，route 等不会自动替代订阅节点。"
                />
                <Table rowKey="id" columns={ctx.cloudExpiryColumns} dataSource={ctx.cloudExpiryList} pagination={tablePagination()} scroll={{ x: 1280 }} />
              </Space>
  );
}
