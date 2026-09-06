import { DeleteOutlined, EditOutlined, EyeOutlined, PlusOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Collapse, Drawer, Modal, Select, Space, Table, Tabs, Typography, message } from "antd";
import type { ColumnType, ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import YAML from "yaml";
import type { K8sDeleteOptions } from "../../services/service-factory";
import { extractApiErrorMessage } from "../../services/http";
import { K8sPageToolbar } from "../ops/k8s-page-toolbar";
import { OpsPageHeader } from "../ops/ops-page-header";
import { useK8sClusterTier } from "../../hooks/use-k8s-cluster-tier";
import { useK8sContext } from "../../hooks/use-k8s-context";
import { useK8sWatch } from "../../hooks/use-k8s-watch";
import { useEditGuardStore } from "../../stores/edit-guard-store";
import { AiYamlGeneratePanel } from "./ai-yaml-generate-panel";
import { K8sDeleteDialog } from "./k8s-delete-dialog";
import { MonacoYamlEditor, validateYaml } from "./monaco-yaml-editor";

function sumColumnWidths(columns: ColumnType<unknown>[], fallback = 120): number {
  return columns.reduce((sum, col) => {
    const w = col.width;
    if (typeof w === "number") return sum + w;
    if (typeof w === "string") {
      const n = Number.parseInt(w, 10);
      if (Number.isFinite(n)) return sum + n;
    }
    return sum + fallback;
  }, 0);
}

export type ClusterOption = { label: string; value: number; disabled?: boolean };
export type NamespaceOption = { label: string; value: string };

export type YamlCrudListArgs = {
  clusterId: number;
  namespace?: string;
  keyword?: string;
};

export type YamlCrudDetailArgs = {
  clusterId: number;
  namespace?: string;
  name: string;
};

export type YamlCrudApplyArgs = {
  clusterId: number;
  manifest: string;
};

export type YamlCrudPreviewResult = {
  dry_run_ok: boolean;
  message?: string;
  diffs?: Array<{
    kind: string;
    namespace: string;
    name: string;
    exists: boolean;
    unified?: string;
  }>;
};

export type YamlCrudDeleteArgs = {
  clusterId: number;
  namespace?: string;
  name: string;
} & K8sDeleteOptions;

export interface YamlCrudApi<TItem, TDetail> {
  list: (args: YamlCrudListArgs) => Promise<TItem[]>;
  detail: (args: YamlCrudDetailArgs) => Promise<TDetail>;
  apply?: (args: YamlCrudApplyArgs) => Promise<unknown>;
  /** 可选：Apply 前预检（Deployment/StatefulSet preview-apply） */
  previewApply?: (args: YamlCrudApplyArgs) => Promise<YamlCrudPreviewResult>;
  remove?: (args: YamlCrudDeleteArgs) => Promise<unknown>;
}

/** 工具栏及创建流程回调使用的上下文 */
export type YamlCrudToolbarCtx = {
  clusterId?: number;
  namespace?: string;
  reload: () => void;
};

/** 创建抽屉打开时传给子组件（含关闭外层抽屉） */
export type YamlCrudCreateCtx = YamlCrudToolbarCtx & {
  closeCreateDrawer: () => void;
};

export interface YamlCrudPageProps<TItem extends { name: string }, TDetail extends { yaml: string }> {
  title: string;
  needNamespace?: boolean;
  namespaceOptions?: NamespaceOption[];
  onLoadNamespaces?: (clusterId: number) => Promise<NamespaceOption[]>;
  columns: ColumnsType<TItem>;
  api: YamlCrudApi<TItem, TDetail>;
  extraRowActions?: (record: TItem, ctx: { clusterId: number; namespace?: string; reload: () => void }) => React.ReactNode;
  onEdit?: (record: TItem, ctx: { clusterId: number; namespace?: string; reload: () => void }) => void;
  detailExtra?: (detail: TDetail, yamlCtx?: { yaml: string; setYaml: (v: string) => void }) => React.ReactNode;
  createTemplate?: (ctx: { namespace?: string }) => string;
  /** AI 生成 YAML 时的资源 Kind，缺省从模板或标题推断 */
  aiResourceKind?: string;
  /** 点击「创建」打开右侧抽屉后调用（准备表单初始值等，如 prepareCreate） */
  onCreateDrawerOpen?: (ctx: YamlCrudCreateCtx) => void;
  /** 「表单创建」Tab 内容（与 Pod 页一致：与 YAML 同在一个创建抽屉内） */
  renderCreateFormTab?: (ctx: YamlCrudCreateCtx) => React.ReactNode;
  /** 集群/命名空间/搜索变化时回调，便于父组件同步 reload 引用 */
  onToolbarReady?: (ctx: YamlCrudToolbarCtx) => void;
  renderToolbarExtraRight?: (ctx: YamlCrudToolbarCtx) => React.ReactNode;
  renderDetail?: (detail: TDetail) => React.ReactNode;
  showEditButton?: boolean;
  confirmOverwrite?: boolean;
  disableMutations?: boolean;
  /** 操作列宽度，节点等页面操作较多时可加大；默认需容纳 详情/编辑/删除（及常见额外操作） */
  actionColumnWidth?: number;
  /** 启用 K8s SSE Watch 时传入资源短名，如 deployments、pods */
  watchResource?: string;
  /** 页头副标题 */
  description?: string;
  /** 页头右侧扩展 */
  headerExtra?: React.ReactNode;
  /** 表格上方摘要条（节点/命名空间统计等） */
  renderSummary?: (items: TItem[], ctx: YamlCrudToolbarCtx) => React.ReactNode;
  /** 表格横向滚动最小宽度；未设或小于列宽总和时自动按列宽 + 操作列计算 */
  tableScrollX?: number | string;
}

export function YamlCrudPage<TItem extends { name: string }, TDetail extends { yaml: string }>(props: YamlCrudPageProps<TItem, TDetail>) {
  const {
    title,
    needNamespace,
    columns,
    api,
    extraRowActions,
    onEdit,
    onLoadNamespaces,
    detailExtra,
    createTemplate,
    aiResourceKind,
    onCreateDrawerOpen,
    renderCreateFormTab,
    onToolbarReady,
    renderToolbarExtraRight,
    renderDetail,
    showEditButton = true,
    confirmOverwrite = true,
    disableMutations = false,
    actionColumnWidth = 360,
    watchResource,
    description,
    headerExtra,
    renderSummary,
    tableScrollX,
  } = props;
  const {
    clusterId,
    namespace,
    setClusterId,
    setNamespace,
    clusterOptions,
    namespaceOptions,
  } = useK8sContext({
    needNamespace: Boolean(needNamespace),
    syncUrl: true,
    onLoadNamespaces,
  });
  const { canMutate, loading: tierLoading } = useK8sClusterTier(clusterId);
  const mutationsDisabled = disableMutations || (!!clusterId && !tierLoading && !canMutate);
  const [keyword, setKeyword] = useState<string>("");
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<TItem[]>([]);
  const [listError, setListError] = useState<string>("");

  const [detailOpen, setDetailOpen] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detail, setDetail] = useState<TDetail | null>(null);
  const [detailName, setDetailName] = useState<string>("");
  const [createDrawerOpen, setCreateDrawerOpen] = useState(false);
  const [applyLoading, setApplyLoading] = useState(false);
  const [detailApplyLoading, setDetailApplyLoading] = useState(false);
  const [detailYaml, setDetailYaml] = useState("");
  const [manifest, setManifest] = useState<string>("");
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<TItem | null>(null);
  const [deleteLoading, setDeleteLoading] = useState(false);
  const [watchLive, setWatchLive] = useState(false);
  const beginEdit = useEditGuardStore((s) => s.beginEdit);
  const endEdit = useEditGuardStore((s) => s.endEdit);

  const closeCreateDrawer = useCallback(() => setCreateDrawerOpen(false), []);
  const reloadSeqRef = useRef(0);

  /** silent=true：Watch 触发的后台刷新，不亮 Table loading，避免闪烁 */
  async function reload(overrideKeyword?: string, opts?: { silent?: boolean }) {
    if (!clusterId) return;
    if (needNamespace && !namespace) return;
    const seq = ++reloadSeqRef.current;
    const silent = Boolean(opts?.silent);
    if (!silent) setLoading(true);
    try {
      const effectiveKeyword = (overrideKeyword ?? keyword).trim();
      const list = await api.list({ clusterId, namespace, keyword: effectiveKeyword || undefined });
      if (seq !== reloadSeqRef.current) return;
      setListError("");
      setData(list ?? []);
    } catch (err: unknown) {
      if (seq !== reloadSeqRef.current) return;
      setData([]);
      const msg = extractApiErrorMessage(err, "加载列表失败");
      setListError(msg);
      if (!silent) message.error(msg);
    } finally {
      if (!silent && seq === reloadSeqRef.current) setLoading(false);
    }
  }

  const onToolbarReadyRef = useRef(onToolbarReady);
  onToolbarReadyRef.current = onToolbarReady;

  useEffect(() => {
    const fn = onToolbarReadyRef.current;
    if (!fn) return;
    fn({
      clusterId,
      namespace,
      reload: () => void reload(),
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [clusterId, namespace, keyword]);

  useEffect(() => {
    void reload();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [clusterId, namespace]);

  useK8sWatch({
    enabled: Boolean(watchResource) && watchLive && Boolean(clusterId) && (!needNamespace || Boolean(namespace)),
    clusterId,
    namespace,
    resource: watchResource,
    requireNamespace: Boolean(needNamespace),
    onRefresh: () => void reload(undefined, { silent: true }),
    onDisabled: () => setWatchLive(false),
  });

  useEffect(() => {
    if (!detailOpen && !createDrawerOpen && !deleteDialogOpen) return;
    beginEdit();
    return () => endEdit();
  }, [detailOpen, createDrawerOpen, deleteDialogOpen, beginEdit, endEdit]);

  useEffect(() => {
    if (!detailOpen) {
      setDetailYaml("");
      return;
    }
    if (detail?.yaml != null) {
      setDetailYaml(detail.yaml);
    }
  }, [detailOpen, detail?.yaml]);

  const actionCol: ColumnsType<TItem>[number] = {
    title: "操作",
    key: "action",
    width: actionColumnWidth,
    fixed: "right",
    className: "yunshu-table-actions-cell",
    render: (_: unknown, record: TItem) => (
      <Space size={0} wrap className="yunshu-table-actions">
        <Button
          type="link"
          size="small"
          icon={<EyeOutlined />}
          onClick={() => {
            if (!clusterId) return;
            // 清理可能残留的 confirm/info 遮罩，避免遮住详情弹窗
            Modal.destroyAll();
            message.loading({ content: "正在加载详情...", key: "yaml-crud-detail", duration: 0 });
            setDetailOpen(true);
            setDetailName(record.name);
            setDetail(null);
            setDetailLoading(true);
            void (async () => {
              try {
                const d = await api.detail({ clusterId, namespace, name: record.name });
                setDetail(d);
              } catch (e) {
                const status = (e as any)?.response?.status;
                if (status === 403) {
                  message.error({ content: "无访问权限", key: "forbidden" });
                } else {
                message.error({
                  content: e instanceof Error ? e.message : "加载详情失败",
                  key: "yaml-crud-detail",
                });
                }
              } finally {
                setDetailLoading(false);
                message.destroy("yaml-crud-detail");
              }
            })();
          }}
        >
          详情
        </Button>
        {!mutationsDisabled && showEditButton && onEdit ? (
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => {
              if (!clusterId) return;
              onEdit(record, { clusterId, namespace, reload });
            }}
          >
            编辑
          </Button>
        ) : null}
        {extraRowActions?.(record, { clusterId: clusterId ?? 0, namespace, reload })}
        {!mutationsDisabled && api.remove ? (
          <Button
            danger
            type="link"
            size="small"
            icon={<DeleteOutlined />}
            onClick={() => {
              setDeleteTarget(record);
              setDeleteDialogOpen(true);
            }}
          >
            删除
          </Button>
        ) : null}
      </Space>
    ),
  };

  const resolvedScrollX = useMemo(() => {
    const computed = sumColumnWidths(columns as ColumnType<unknown>[]) + actionColumnWidth;
    if (tableScrollX === "max-content") return computed;
    if (typeof tableScrollX === "number") return Math.max(tableScrollX, computed);
    return computed;
  }, [columns, actionColumnWidth, tableScrollX]);

  const toolbarCtx: YamlCrudToolbarCtx = {
    clusterId,
    namespace,
    reload: () => void reload(),
  };

  const createCtx: YamlCrudCreateCtx = {
    ...toolbarCtx,
    closeCreateDrawer,
  };

  const hasFormTab = Boolean(renderCreateFormTab);
  const hasYamlTab = Boolean(api.apply);
  const canOpenCreate = !mutationsDisabled && (hasFormTab || hasYamlTab);

  function handleOpenCreateDrawer() {
    if (!clusterId) return;
    setManifest(createTemplate ? createTemplate({ namespace }) : "");
    setCreateDrawerOpen(true);
    onCreateDrawerOpen?.(createCtx);
  }

  async function applyYamlManifest(manifestText: string, onSuccess?: () => void | Promise<void>) {
    const applyFn = api.apply;
    if (!clusterId || !applyFn) return;
    if (validateYaml(manifestText)) {
      message.warning("请先修正 YAML 语法错误");
      return;
    }

    const doApply = async () => {
      setApplyLoading(true);
      setDetailApplyLoading(true);
      try {
        await applyFn({ clusterId, manifest: manifestText });
        message.success("应用成功");
        await onSuccess?.();
      } finally {
        setApplyLoading(false);
        setDetailApplyLoading(false);
      }
    };

    const runPreviewThen = async (next: () => Promise<void>) => {
      const previewFn = api.previewApply;
      if (!previewFn) {
        await next();
        return;
      }
      setApplyLoading(true);
      setDetailApplyLoading(true);
      try {
        const preview = await previewFn({ clusterId, manifest: manifestText });
        const diffs = preview.diffs ?? [];
        const unified = diffs
          .map((d) => d.unified?.trim())
          .filter(Boolean)
          .join("\n\n");
        const statusLine = preview.dry_run_ok
          ? "Server-side dry-run 通过"
          : `Dry-run 未通过：${preview.message || "未知错误"}`;
        Modal.confirm({
          title: "应用预检",
          width: 820,
          okText: preview.dry_run_ok ? "确认应用" : "仍要应用",
          okButtonProps: preview.dry_run_ok ? undefined : { danger: true },
          cancelText: "取消",
          content: (
            <Space direction="vertical" style={{ width: "100%" }} size="small">
              <Typography.Text type={preview.dry_run_ok ? "success" : "danger"}>{statusLine}</Typography.Text>
              {diffs.length ? (
                <Typography.Text type="secondary">
                  变更对象：
                  {diffs.map((d) => `${d.kind}/${d.namespace || "-"}/${d.name}${d.exists ? "" : " (新建)"}`).join("，")}
                </Typography.Text>
              ) : null}
              <Typography.Paragraph
                style={{
                  marginBottom: 0,
                  maxHeight: 360,
                  overflow: "auto",
                  whiteSpace: "pre-wrap",
                  fontFamily: "var(--ys-font-mono, monospace)",
                  fontSize: 12,
                  background: "var(--admin-surface, #f8fafc)",
                  padding: 12,
                  borderRadius: 8,
                }}
              >
                {unified || preview.message || "无文本 diff（可能为新建或无法解析）"}
              </Typography.Paragraph>
            </Space>
          ),
          onOk: async () => {
            await next();
          },
        });
      } catch (err: unknown) {
        message.error(extractApiErrorMessage(err, "预检失败"));
      } finally {
        setApplyLoading(false);
        setDetailApplyLoading(false);
      }
    };

    const confirmAndApply = async () => {
      if (!confirmOverwrite) {
        await runPreviewThen(doApply);
        return;
      }
      if (!manifestText.trim()) {
        await runPreviewThen(doApply);
        return;
      }

      try {
        const docs = YAML.parseAllDocuments(manifestText);
        let targetName: string | undefined;
        let targetNamespace: string | undefined;

        for (const doc of docs) {
          const v: any = doc.toJSON();
          const md = v?.metadata;
          const n = md?.name;
          if (n) {
            targetName = String(n);
            targetNamespace = md?.namespace ? String(md.namespace) : namespace ?? "default";
            break;
          }
        }

        if (!targetName) {
          await runPreviewThen(doApply);
          return;
        }

        let exists = false;
        try {
          await api.detail({ clusterId, namespace: targetNamespace, name: targetName });
          exists = true;
        } catch {
          exists = false;
        }

        if (!exists) {
          await runPreviewThen(doApply);
          return;
        }

        // 有 preview 时直接走预检弹窗；否则保留覆盖确认
        if (api.previewApply) {
          await runPreviewThen(doApply);
          return;
        }

        Modal.confirm({
          title: "检测到同名对象",
          content: `${targetNamespace}/${targetName} 已存在，确认覆盖吗？（apply 会直接更新）`,
          okText: "覆盖并应用",
          cancelText: "取消",
          onOk: async () => {
            await doApply();
          },
        });
        return;
      } catch {
        await runPreviewThen(doApply);
        return;
      }
    };

    void confirmAndApply();
  }

  async function submitCreateYaml() {
    await applyYamlManifest(manifest, async () => {
      closeCreateDrawer();
      await reload();
    });
  }

  async function submitDetailYaml() {
    await applyYamlManifest(detailYaml, async () => {
      setDetailOpen(false);
      await reload();
    });
  }

  const canApplyDetailYaml = Boolean(api.apply) && !mutationsDisabled;

  const resolvedAiKind = useMemo(() => {
    if (aiResourceKind?.trim()) return aiResourceKind.trim();
    const tpl = createTemplate?.({ namespace }) ?? "";
    try {
      const doc = YAML.parse(tpl) as { kind?: string } | null;
      if (doc?.kind) return String(doc.kind);
    } catch {
      /* ignore */
    }
    return title.replace(/管理$/, "").trim() || "Resource";
  }, [aiResourceKind, createTemplate, namespace, title]);

  const yamlCreatePanel = (
    <Space direction="vertical" style={{ width: "100%" }} size="middle">
      <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
        支持直接粘贴 Kubernetes YAML（底层使用 Kom SDK 的 apply）。打开时已预填模板（若有），可清空后自行编写，或用 AI 按描述生成。
      </Typography.Paragraph>
      <AiYamlGeneratePanel
        resourceKind={resolvedAiKind}
        namespace={namespace}
        clusterId={clusterId}
        hintYaml={manifest}
        onGenerated={setManifest}
      />
      <Space wrap>
        {createTemplate ? (
          <Button size="small" onClick={() => setManifest(createTemplate({ namespace }))}>
            填入模板
          </Button>
        ) : null}
        <Button size="small" onClick={() => setManifest("")}>
          清空内容
        </Button>
      </Space>
      <MonacoYamlEditor value={manifest} onChange={setManifest} height={420} />
      <Button type="primary" loading={applyLoading} onClick={() => void submitCreateYaml()}>
        创建
      </Button>
      <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
        提示：如果要修改现有对象，建议保留 name/namespace 并直接 apply。
      </Typography.Paragraph>
    </Space>
  );

  return (
    <div className="page-stack">
      <OpsPageHeader title={title} description={description} extra={headerExtra} />
      <Card className="table-card yaml-crud-card" bordered={false}>
      <K8sPageToolbar
        clusterId={clusterId}
        namespace={namespace}
        clusterOptions={clusterOptions}
        namespaceOptions={namespaceOptions}
        needNamespace={needNamespace}
        onClusterChange={setClusterId}
        onNamespaceChange={setNamespace}
        onSearch={(v) => {
          setKeyword(v);
          void reload(v);
        }}
        onRefresh={() => void reload()}
        watchLive={watchResource ? watchLive : undefined}
        onWatchChange={watchResource ? setWatchLive : undefined}
        extraRight={renderToolbarExtraRight ? renderToolbarExtraRight(toolbarCtx) : undefined}
        primaryAction={
          canOpenCreate ? (
            <Button type="primary" icon={<PlusOutlined />} disabled={!clusterId} onClick={handleOpenCreateDrawer}>
              创建
            </Button>
          ) : undefined
        }
      />
      {listError ? (
        <Alert
          type="error"
          showIcon
          style={{ marginBottom: 12 }}
          message="列表加载失败"
          description={listError}
          action={
            <Button size="small" onClick={() => void reload()}>
              重试
            </Button>
          }
        />
      ) : null}
      {clusterId && !tierLoading && !canMutate && !disableMutations ? (
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 12 }}
          message="当前集群档位为只读（或只读+Exec），已隐藏创建/编辑/删除；变更需 admin 档位"
        />
      ) : null}

      {renderSummary ? <div className="k8s-summary-row-wrap">{renderSummary(data, toolbarCtx)}</div> : null}

      <div className="k8s-table-scroll-host">
        <Table
          rowKey={(r) => (r as any).name}
          loading={loading}
          dataSource={data}
          pagination={{ pageSize: 10, showSizeChanger: true, pageSizeOptions: [10, 20, 50, 100], showQuickJumper: true }}
          columns={[...columns, actionCol]}
          scroll={{ x: resolvedScrollX }}
          tableLayout="fixed"
        />
      </div>

      {!mutationsDisabled && (hasFormTab || hasYamlTab) ? (
        <Drawer
          title={
            <Space direction="vertical" size={0}>
              <span>创建</span>
              {needNamespace ? (
                <Typography.Text type="secondary" style={{ fontSize: 13, fontWeight: "normal" }}>
                  目标命名空间：{namespace ?? "—"}
                </Typography.Text>
              ) : null}
            </Space>
          }
          placement="right"
          width={960}
          open={createDrawerOpen}
          onClose={closeCreateDrawer}
          destroyOnClose
          maskClosable={false}
          styles={{ body: { paddingBottom: 24 } }}
          zIndex={1200}
          extra={<Button onClick={closeCreateDrawer}>取消</Button>}
        >
          {hasFormTab && hasYamlTab ? (
            <Tabs
              defaultActiveKey="form"
              items={[
                { key: "form", label: "表单创建", children: renderCreateFormTab?.(createCtx) },
                { key: "yaml", label: "YAML 创建", children: yamlCreatePanel },
              ]}
            />
          ) : hasYamlTab ? (
            yamlCreatePanel
          ) : (
            renderCreateFormTab?.(createCtx)
          )}
        </Drawer>
      ) : null}

      <Drawer
        title={`${title} - 详情${detailName ? `：${detailName}` : ""}`}
        open={detailOpen}
        onClose={() => {
          setDetailOpen(false);
        }}
        destroyOnClose
        width={920}
        zIndex={1300}
        className="detail-edit-drawer"
        extra={
          canApplyDetailYaml ? (
            <Button type="primary" loading={detailApplyLoading} onClick={() => void submitDetailYaml()}>
              应用 YAML
            </Button>
          ) : null
        }
      >
        {detailLoading ? (
          <Typography.Text type="secondary">加载中...</Typography.Text>
        ) : detail ? (
          <Space direction="vertical" style={{ width: "100%" }} size="middle">
            {detailExtra?.(detail, { yaml: detailYaml, setYaml: setDetailYaml })}
            {renderDetail ? renderDetail(detail) : null}
            {!renderDetail && !detailExtra ? (
              <Typography.Paragraph copyable style={{ marginBottom: 0, whiteSpace: "pre-wrap" }}>
                {detail.yaml}
              </Typography.Paragraph>
            ) : null}
            <Collapse
              defaultActiveKey={["yaml"]}
              items={[
                {
                  key: "yaml",
                  label: "YAML",
                  children: (
                    <Space direction="vertical" style={{ width: "100%" }} size="middle">
                      {canApplyDetailYaml ? (
                        <AiYamlGeneratePanel
                          resourceKind={resolvedAiKind}
                          namespace={namespace}
                          clusterId={clusterId}
                          hintYaml={detailYaml}
                          onGenerated={setDetailYaml}
                        />
                      ) : null}
                      <MonacoYamlEditor
                        value={detailYaml}
                        onChange={canApplyDetailYaml ? setDetailYaml : undefined}
                        readOnly={!canApplyDetailYaml}
                        height={480}
                      />
                    </Space>
                  ),
                },
              ]}
            />
          </Space>
        ) : (
          <Typography.Text type="secondary">暂无数据</Typography.Text>
        )}
      </Drawer>

      <K8sDeleteDialog
        open={deleteDialogOpen}
        resourceName={deleteTarget?.name ?? ""}
        loading={deleteLoading}
        onCancel={() => {
          setDeleteDialogOpen(false);
          setDeleteTarget(null);
        }}
        onConfirm={async (deleteOpts) => {
          if (!clusterId || !deleteTarget) return;
          setDeleteLoading(true);
          try {
            await api.remove?.({
              clusterId,
              namespace,
              name: deleteTarget.name,
              ...deleteOpts,
            });
            message.success("删除成功");
            setDeleteDialogOpen(false);
            setDeleteTarget(null);
            await reload();
          } finally {
            setDeleteLoading(false);
          }
        }}
      />

      </Card>
    </div>
  );
}

