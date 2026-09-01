/**
 * Promote K8s pages into web-pro native routes (PageContainer / LegacyShell).
 * YamlCrudPage already uses PageContainer — those pages only need @/ imports + default export.
 * Run: node scripts/promote-k8s-batch.mjs
 */
import fs from 'node:fs';
import path from 'node:path';

const root = path.resolve('src/pages');
let failed = 0;

function load(srcRel) {
  let s = fs.readFileSync(path.join(root, srcRel), 'utf8');
  const hadCRLF = s.includes('\r\n');
  s = s.replace(/\r\n/g, '\n');
  s = s.replaceAll('from "../', 'from "@/').replaceAll("from '../", "from '@/");
  return { s, hadCRLF };
}

function save(outRel, s, hadCRLF, check) {
  if (hadCRLF) s = s.replace(/\n/g, '\r\n');
  const out = path.join(root, outRel);
  fs.mkdirSync(path.dirname(out), { recursive: true });
  fs.writeFileSync(out, s, 'utf8');
  const ok = check(s);
  console.log(outRel, { ok });
  if (!ok) failed += 1;
}

function ensureImport(s, line) {
  if (s.includes(line.trim())) return s;
  const nl = s.indexOf('\n');
  return s.slice(0, nl + 1) + line + '\n' + s.slice(nl + 1);
}

function toDefaultExport(s, exportName) {
  const re = new RegExp(`export function ${exportName}\\(`);
  if (!re.test(s)) throw new Error(`export function ${exportName} missing`);
  s = s.replace(re, `export default function ${exportName}(`);
  s = s.replace(new RegExp(`\\nexport default ${exportName};\\n?$`), '\n');
  return s;
}

function wrapLegacyShell(s, exportName) {
  s = ensureImport(s, 'import { LegacyShell } from "@/components/LegacyShell";');
  s = s.replace(
    `export default function ${exportName}(`,
    `export default function ${exportName}(`,
  );
  // Rename default export body to Inner, wrap with LegacyShell
  s = s.replace(
    `export default function ${exportName}(`,
    `export default function ${exportName}() {\n  return (\n    <LegacyShell>\n      <${exportName}Inner />\n    </LegacyShell>\n  );\n}\n\nfunction ${exportName}Inner(`,
  );
  return s;
}

function checkYaml(s) {
  return (
    s.includes('export default') &&
    s.includes('YamlCrudPage') &&
    /[\u4e00-\u9fff]{2}/.test(s) &&
    !s.includes('from "../')
  );
}

function checkPageContainer(s) {
  const open = (s.match(/<PageContainer\b/g) || []).length;
  const close = (s.match(/<\/PageContainer>/g) || []).length;
  return open === 1 && close === 1 && s.includes('export default') && /[\u4e00-\u9fff]{2}/.test(s);
}

/** YamlCrud list pages */
const YAML_CRUD = [
  ['namespaces-page.tsx', 'NamespacesPage', 'k8s/namespaces/index.tsx', false],
  ['configmaps-page.tsx', 'ConfigmapsPage', 'k8s/configmaps/index.tsx', false],
  ['secrets-page.tsx', 'SecretsPage', 'k8s/secrets/index.tsx', false],
  ['crds-page.tsx', 'CrdsPage', 'k8s/crds/index.tsx', false],
  ['ingress-classes-page.tsx', 'IngressClassesPage', 'k8s/ingress-classes/index.tsx', false],
  ['ingresses-page.tsx', 'IngressesPage', 'k8s/ingresses/index.tsx', false],
  ['k8s-services-page.tsx', 'K8sServicesPage', 'k8s/services/index.tsx', false],
  ['persistentvolumes-page.tsx', 'PersistentVolumesPage', 'k8s/persistentvolumes/index.tsx', false],
  ['persistentvolumeclaims-page.tsx', 'PersistentVolumeClaimsPage', 'k8s/persistentvolumeclaims/index.tsx', false],
  ['storageclasses-page.tsx', 'StorageClassesPage', 'k8s/storageclasses/index.tsx', false],
  ['serviceaccounts-page.tsx', 'ServiceaccountsPage', 'k8s/serviceaccounts/index.tsx', false],
  ['rbac-roles-page.tsx', 'RbacRolesPage', 'k8s/rbac/roles/index.tsx', false],
  ['rbac-rolebindings-page.tsx', 'RbacRoleBindingsPage', 'k8s/rbac/rolebindings/index.tsx', false],
  ['rbac-clusterroles-page.tsx', 'RbacClusterRolesPage', 'k8s/rbac/clusterroles/index.tsx', false],
  ['rbac-clusterrolebindings-page.tsx', 'RbacClusterRoleBindingsPage', 'k8s/rbac/clusterrolebindings/index.tsx', false],
  ['network-policies-page.tsx', 'NetworkPoliciesPage', 'k8s/network-policies/index.tsx', false],
  ['horizontal-pod-autoscalers-page.tsx', 'HorizontalPodAutoscalersPage', 'k8s/hpa/index.tsx', false],
  ['daemonsets-page.tsx', 'DaemonsetsPage', 'k8s/daemonsets/index.tsx', false],
  ['jobs-page.tsx', 'JobsPage', 'k8s/jobs/index.tsx', false],
  ['cronjobs-page.tsx', 'CronjobsPage', 'k8s/cronjobs/index.tsx', false],
  ['deployments-page.tsx', 'DeploymentsPage', 'k8s/deployments/index.tsx', true],
  ['statefulsets-page.tsx', 'StatefulsetsPage', 'k8s/statefulsets/index.tsx', true],
  ['nodes-page.tsx', 'NodesPage', 'k8s/nodes/index.tsx', true],
];

for (const [src, name, out, shell] of YAML_CRUD) {
  let { s, hadCRLF } = load(src);
  s = toDefaultExport(s, name);
  if (shell) s = wrapLegacyShell(s, name);
  save(out, s, hadCRLF, checkYaml);
}

// ---- OpsPageHeader → PageContainer helpers ----
function stripOpsImport(s) {
  return s.replace(/import \{ OpsPageHeader \} from "@\/components\/ops\/ops-page-header";\n/, '');
}

function ensurePageContainer(s) {
  return ensureImport(s, 'import { PageContainer } from "@ant-design/pro-components";');
}

// events
{
  let { s, hadCRLF } = load('events-page.tsx');
  s = stripOpsImport(s);
  s = ensurePageContainer(s);
  s = toDefaultExport(s, 'EventsPage');
  s = s.replace(
    `  return (
    <div className="page-stack">
      <OpsPageHeader
        title="Event 事件"
        description="集群与命名空间内 Kubernetes 事件检索与聚合"
        breadcrumbs={[{ title: "Kubernetes" }, { title: "Events" }]}
      />
`,
    `  return (
    <PageContainer
      header={{
        title: "Event 事件",
        subTitle: "集群与命名空间内 Kubernetes 事件检索与聚合",
      }}
    >
`,
  );
  const marker = '    </div>\n  );\n}';
  const idx = s.lastIndexOf(marker);
  if (idx < 0) throw new Error('events close missing');
  s = s.slice(0, idx) + '    </PageContainer>\n  );\n}';
  save('k8s/events/index.tsx', s, hadCRLF, checkPageContainer);
}

// helm charts
{
  let { s, hadCRLF } = load('helm-charts-page.tsx');
  s = stripOpsImport(s);
  s = ensurePageContainer(s);
  s = toDefaultExport(s, 'HelmChartsPage');
  s = s.replace(
    `  return (
    <div className="page-stack">
      <OpsPageHeader
        title="Harbor Helm Chart"
        description="Jenkins 推送的 Chart 包列表与版本"
        extra={
          <Link to="/helm/releases">
            <Button type="primary" icon={<RocketOutlined />}>
              安装 Release
            </Button>
          </Link>
        }
        meta={
          <Typography.Text type="secondary">
            {info ? \`\${info.project} @ \${info.url}\` : "加载 Harbor 配置…"}
          </Typography.Text>
        }
      />
`,
    `  return (
    <PageContainer
      header={{
        title: "Harbor Helm Chart",
        subTitle: "Jenkins 推送的 Chart 包列表与版本",
        extra: (
          <Link to="/helm/releases">
            <Button type="primary" icon={<RocketOutlined />}>
              安装 Release
            </Button>
          </Link>
        ),
      }}
    >
`,
  );
  const marker = '    </div>\n  );\n}';
  const idx = s.lastIndexOf(marker);
  if (idx < 0) throw new Error('helm-charts close missing');
  s = s.slice(0, idx) + '    </PageContainer>\n  );\n}';
  save('k8s/helm/charts/index.tsx', s, hadCRLF, checkPageContainer);
}

// helm releases
{
  let { s, hadCRLF } = load('helm-releases-page.tsx');
  s = stripOpsImport(s);
  s = ensurePageContainer(s);
  s = toDefaultExport(s, 'HelmReleasesPage');
  s = s.replace(
    `  return (
    <div className="page-stack">
      <OpsPageHeader
        title="Helm Release 管理"
        description="Harbor OCI Chart 安装、Release 查看与应急回滚"
        extra={<Link to="/helm/charts">Harbor Chart 目录</Link>}
        meta={
          <Space size="middle">
            <Typography.Text type="secondary">Release {statusSummary.total}</Typography.Text>
            <Typography.Text type="secondary">Deployed {statusSummary.deployed}</Typography.Text>
            {statusSummary.failed > 0 ? (
              <Typography.Text type="danger">Failed {statusSummary.failed}</Typography.Text>
            ) : null}
          </Space>
        }
      />
`,
    `  return (
    <PageContainer
      header={{
        title: "Helm Release 管理",
        subTitle: "Harbor OCI Chart 安装、Release 查看与应急回滚",
        extra: <Link to="/helm/charts">Harbor Chart 目录</Link>,
      }}
    >
`,
  );
  const marker = '    </div>\n  );\n}';
  const idx = s.lastIndexOf(marker);
  if (idx < 0) throw new Error('helm-releases close missing');
  s = s.slice(0, idx) + '    </PageContainer>\n  );\n}';
  save('k8s/helm/releases/index.tsx', s, hadCRLF, checkPageContainer);
}

// pods
{
  let { s, hadCRLF } = load('pod-page.tsx');
  s = stripOpsImport(s);
  s = ensurePageContainer(s);
  // local ./pod/* → ../../pod/* from k8s/pods/
  s = s.replaceAll('from "./pod/', 'from "../../pod/');
  s = toDefaultExport(s, 'PodPage');
  s = s.replace(
    `  return (
    <div className="page-stack">
      <OpsPageHeader
        title="Pod 管理"
        description="跨集群 Pod 生命周期、诊断、日志与终端会话"
        breadcrumbs={[{ title: "Kubernetes" }, { title: "Pod" }]}
        meta={
          <span>
            {clusterId ? \`集群 #\${clusterId} · \${namespace}\` : "未选择集群"} · {loading ? "加载中" : \`\${pods.length} 条\`}
          </span>
        }
      />
`,
    `  return (
    <PageContainer
      header={{
        title: "Pod 管理",
        subTitle: "跨集群 Pod 生命周期、诊断、日志与终端会话",
      }}
    >
`,
  );
  const marker = '    </div>\n  );\n}';
  const idx = s.lastIndexOf(marker);
  if (idx < 0) throw new Error('pods close missing');
  s = s.slice(0, idx) + '    </PageContainer>\n  );\n}';
  save('k8s/pods/index.tsx', s, hadCRLF, checkPageContainer);
}

// clusters
{
  let { s, hadCRLF } = load('cluster-page.tsx');
  s = stripOpsImport(s);
  s = ensurePageContainer(s);
  s = toDefaultExport(s, 'ClusterPage');
  s = s.replace(
    `  return (
    <div className="page-stack">
      <OpsPageHeader
        title="集群列表"
        description="纳管 Kubernetes 集群连接、授权矩阵与运行状态"
        meta={
          <Space size="middle">
            <Typography.Text type="secondary">共 {total} 个集群</Typography.Text>
            <Typography.Text type="secondary">{loading ? "同步中…" : "已同步"}</Typography.Text>
          </Space>
        }
        extra={
          <Space wrap>
            <Segmented
              value={viewMode}
              onChange={(v) => setViewMode(v as "table" | "cards")}
              options={[
                { label: "卡片", value: "cards", icon: <AppstoreOutlined /> },
                { label: "表格", value: "table", icon: <TableOutlined /> },
              ]}
            />
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
              新建集群
            </Button>
            <Button icon={<ReloadOutlined />} onClick={() => void loadClusters()}>
              刷新
            </Button>
          </Space>
        }
      />
`,
    `  return (
    <PageContainer
      header={{
        title: "集群列表",
        subTitle: "纳管 Kubernetes 集群连接、授权矩阵与运行状态",
        extra: (
          <Space wrap>
            <Segmented
              value={viewMode}
              onChange={(v) => setViewMode(v as "table" | "cards")}
              options={[
                { label: "卡片", value: "cards", icon: <AppstoreOutlined /> },
                { label: "表格", value: "table", icon: <TableOutlined /> },
              ]}
            />
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
              新建集群
            </Button>
            <Button icon={<ReloadOutlined />} onClick={() => void loadClusters()}>
              刷新
            </Button>
          </Space>
        ),
      }}
    >
`,
  );
  const marker = '    </div>\n  );\n}';
  const idx = s.lastIndexOf(marker);
  if (idx < 0) throw new Error('cluster close missing');
  s = s.slice(0, idx) + '    </PageContainer>\n  );\n}';
  save('k8s/clusters/index.tsx', s, hadCRLF, checkPageContainer);
}

// component-status
{
  let { s, hadCRLF } = load('component-status-page.tsx');
  s = ensurePageContainer(s);
  s = toDefaultExport(s, 'ComponentStatusPage');
  s = s.replace(
    `  return (
    <Card className="table-card" title="组件状态">
`,
    `  return (
    <PageContainer header={{ title: "组件状态", subTitle: "集群控制面组件健康探测" }}>
    <Card className="table-card" bordered={false}>
`,
  );
  const marker = '    </Card>\n  );\n}';
  const idx = s.lastIndexOf(marker);
  if (idx < 0) throw new Error('component-status close missing');
  s = s.slice(0, idx) + '    </Card>\n    </PageContainer>\n  );\n}';
  save('k8s/component-status/index.tsx', s, hadCRLF, checkPageContainer);
}

// cluster-api-resources
{
  let { s, hadCRLF } = load('cluster-api-resources-page.tsx');
  s = ensurePageContainer(s);
  s = toDefaultExport(s, 'ClusterApiResourcesPage');
  s = s.replace(
    `  return (
    <div>
      <Card className="table-card" title="API 资源发现（kubectl api-resources）">
`,
    `  return (
    <PageContainer header={{ title: "API 资源发现", subTitle: "kubectl api-resources 等价能力" }}>
      <Card className="table-card" bordered={false}>
`,
  );
  const marker = '    </div>\n  );\n}';
  const idx = s.lastIndexOf(marker);
  if (idx < 0) throw new Error('cluster-api-resources close missing');
  s = s.slice(0, idx) + '    </PageContainer>\n  );\n}';
  save('k8s/cluster-api-resources/index.tsx', s, hadCRLF, checkPageContainer);
}

// crs
{
  let { s, hadCRLF } = load('crs-page.tsx');
  s = ensurePageContainer(s);
  s = toDefaultExport(s, 'CrsPage');
  s = s.replace(
    `  return (
    <Card className="table-card" title="CR 实例管理">
`,
    `  return (
    <PageContainer header={{ title: "CR 实例管理", subTitle: "自定义资源实例列表与 YAML 应用" }}>
    <Card className="table-card" bordered={false}>
`,
  );
  const marker = '    </Card>\n  );\n}';
  const idx = s.lastIndexOf(marker);
  if (idx < 0) throw new Error('crs close missing');
  s = s.slice(0, idx) + '    </Card>\n    </PageContainer>\n  );\n}';
  save('k8s/crs/index.tsx', s, hadCRLF, checkPageContainer);
}

// topology
{
  let { s, hadCRLF } = load('k8s-resource-topology-page.tsx');
  s = ensurePageContainer(s);
  s = toDefaultExport(s, 'K8sResourceTopologyPage');
  s = s.replace(
    `  return (
    <Card
      title="资源拓扑图"
      extra={
        <TypographyHint />
      }
    >
`,
    `  return (
    <PageContainer
      header={{
        title: "资源拓扑图",
        subTitle: "Ingress → Service → Workload → ReplicaSet → Pod",
        extra: <TypographyHint />,
      }}
    >
    <Card bordered={false}>
`,
  );
  const marker = '    </Card>\n  );\n}';
  const idx = s.lastIndexOf(marker);
  if (idx < 0) throw new Error('topology close missing');
  s = s.slice(0, idx) + '    </Card>\n    </PageContainer>\n  );\n}';
  save('k8s/resource-topology/index.tsx', s, hadCRLF, checkPageContainer);
}

// scoped policies
{
  let { s, hadCRLF } = load('k8s-scoped-policies-page.tsx');
  s = ensurePageContainer(s);
  s = toDefaultExport(s, 'K8sScopedPoliciesPage');
  s = s.replace(
    `  return (
    <div>
`,
    `  return (
    <PageContainer header={{ title: "Kubernetes 集群访问档位", subTitle: "数据库维护档位，不经 Casbin" }}>
`,
  );
  // remove Card title duplication — keep card as content
  s = s.replace(
    `      <Card
        className="table-card"
        title="Kubernetes 集群访问档位（数据库维护，不经 Casbin）"
`,
    `      <Card
        className="table-card"
        bordered={false}
`,
  );
  const marker = '    </div>\n  );\n}';
  const idx = s.lastIndexOf(marker);
  if (idx < 0) throw new Error('scoped-policies close missing');
  s = s.slice(0, idx) + '    </PageContainer>\n  );\n}';
  save('k8s/scoped-policies/index.tsx', s, hadCRLF, checkPageContainer);
}

// CR templates
{
  let { s, hadCRLF } = load('k8s-cr-templates-page.tsx');
  s = stripOpsImport(s);
  s = ensurePageContainer(s);
  s = toDefaultExport(s, 'K8sCrTemplatesPage');
  // OpsPageHeader with extra → PageContainer; structure uses fragment
  if (!s.includes('<OpsPageHeader')) throw new Error('cr-templates OpsPageHeader missing');
  // Extract: return ( <> <OpsPageHeader ... /> <Card>...
  s = s.replace(
    `  return (
  <>
    <OpsPageHeader
      title="K8s CR/YAML 模板库"
      description="预置常用 CR 清单，可在自定义资源页复制 body 后应用；project_id=0 为全局模板。"
      extra={
`,
    `  return (
    <PageContainer
      header={{
        title: "K8s CR/YAML 模板库",
        subTitle: "预置常用 CR 清单，可在自定义资源页复制 body 后应用；project_id=0 为全局模板。",
        extra: (
`,
  );
  // Close OpsPageHeader after extra Space — find `/>` after extra block
  // Original ends extra with `}\n    />` then Card
  s = s.replace(
    /(\n          <Button type="primary"[^\n]*\n            新建模板\n          <\/Button>\n        <\/Space>\n      )\n    \/>\n/,
    `$1
        ),
      }}
    >
`,
  );
  if (s.includes('<OpsPageHeader') || s.includes('OpsPageHeader')) {
    // fallback: looser close
    const closeIdx = s.indexOf('/>\n', s.indexOf('新建模板'));
    if (closeIdx > 0 && s.includes('OpsPageHeader')) {
      throw new Error('cr-templates OpsPageHeader close failed');
    }
  }
  // Close fragment → PageContainer
  s = s.replace(/\n  <\/>\n  \);\n}/, '\n    </PageContainer>\n  );\n}');
  // Also try `</>`
  if (s.includes('</>')) {
    const frag = s.lastIndexOf('</>');
    s = s.slice(0, frag) + '</PageContainer>' + s.slice(frag + 3);
  }
  save('k8s/cr-templates/index.tsx', s, hadCRLF, checkPageContainer);
}

// event forward
{
  let { s, hadCRLF } = load('k8s-event-forward-page.tsx');
  s = ensurePageContainer(s);
  s = toDefaultExport(s, 'K8sEventForwardPage');
  s = s.replace(
    `export default function K8sEventForwardPage() {
  return (
    <Card className="table-card" title="K8s Event 多集群转发">
`,
    `export default function K8sEventForwardPage() {
  return (
    <PageContainer header={{ title: "K8s Event 多集群转发", subTitle: "规则与 Worker 参数" }}>
    <Card className="table-card" bordered={false}>
`,
  );
  const marker = '    </Card>\n  );\n}';
  const idx = s.lastIndexOf(marker);
  if (idx < 0) throw new Error('event-forward close missing');
  s = s.slice(0, idx) + '    </Card>\n    </PageContainer>\n  );\n}';
  save('k8s/event-forward/index.tsx', s, hadCRLF, checkPageContainer);
}

console.log(failed ? `promote-k8s-batch FAILED (${failed})` : 'promote-k8s-batch done');
process.exitCode = failed ? 1 : 0;
