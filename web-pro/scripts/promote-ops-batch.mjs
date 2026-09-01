/**
 * Promote remaining ops/project/alert/db/es pages into Pro routes.
 * Run: node scripts/promote-ops-batch.mjs
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

function save(outRel, s, hadCRLF, check = defaultCheck) {
  if (hadCRLF) s = s.replace(/\n/g, '\r\n');
  const out = path.join(root, outRel);
  fs.mkdirSync(path.dirname(out), { recursive: true });
  fs.writeFileSync(out, s, 'utf8');
  const ok = check(s);
  console.log(outRel, { ok });
  if (!ok) failed += 1;
}

function defaultCheck(s) {
  return (
    s.includes('export default') &&
    /[\u4e00-\u9fff]{2}/.test(s) &&
    !s.includes('from "../components') &&
    !s.includes('from "../services') &&
    !s.includes('from "../hooks') &&
    !s.includes('from "../contexts') &&
    !s.includes('from "../utils')
  );
}

function checkPC(s) {
  const open = (s.match(/<PageContainer\b/g) || []).length;
  const close = (s.match(/<\/PageContainer>/g) || []).length;
  return open >= 1 && open === close && defaultCheck(s);
}

function ensureImport(s, line) {
  if (s.includes(line.trim())) return s;
  const nl = s.indexOf('\n');
  return s.slice(0, nl + 1) + line + '\n' + s.slice(nl + 1);
}

function ensurePC(s) {
  return ensureImport(s, 'import { PageContainer } from "@ant-design/pro-components";');
}

function toDefault(s, name) {
  const re = new RegExp(`export function ${name}\\(`);
  if (!re.test(s)) throw new Error(`missing export function ${name}`);
  s = s.replace(re, `export default function ${name}(`);
  s = s.replace(new RegExp(`\\nexport default ${name};\\n?$`), '\n');
  return s;
}

function wrapShell(s, name) {
  s = ensureImport(s, 'import { LegacyShell } from "@/components/LegacyShell";');
  s = s.replace(
    `export default function ${name}(`,
    `export default function ${name}() {\n  return (\n    <LegacyShell>\n      <${name}Inner />\n    </LegacyShell>\n  );\n}\n\nfunction ${name}Inner(`,
  );
  return s;
}

function closeLast(s, from, to) {
  const idx = s.lastIndexOf(from);
  if (idx < 0) throw new Error(`close marker missing: ${JSON.stringify(from)}`);
  return s.slice(0, idx) + to + s.slice(idx + from.length);
}

// ---- simple Card → PageContainer ----
function wrapCardTitle(s, { title, subTitle, out, name, shell = false, localFix }) {
  let { s: src, hadCRLF } = typeof s === 'string' ? { s, hadCRLF: false } : s;
  src = ensurePC(src);
  src = toDefault(src, name);
  if (localFix) src = localFix(src);
  if (shell) src = wrapShell(src, name);

  // Common patterns
  const patterns = [
    {
      from: `  return (\n    <Card\n      className="table-card"\n      title="${title}"`,
      to: `  return (\n    <PageContainer header={{ title: "${title}", subTitle: ${JSON.stringify(subTitle || '')} }}>\n    <Card\n      className="table-card"\n      bordered={false}`,
    },
    {
      from: `  return (\n    <Card title="${title}"`,
      to: `  return (\n    <PageContainer header={{ title: "${title}", subTitle: ${JSON.stringify(subTitle || '')} }}>\n    <Card bordered={false}`,
    },
    {
      from: `  return (\n    <Card\n      title="${title}"`,
      to: `  return (\n    <PageContainer header={{ title: "${title}", subTitle: ${JSON.stringify(subTitle || '')} }}>\n    <Card\n      bordered={false}`,
    },
  ];
  let hit = false;
  for (const p of patterns) {
    if (src.includes(p.from)) {
      src = src.replace(p.from, p.to);
      hit = true;
      break;
    }
  }
  if (!hit) throw new Error(`${name}: Card title return not found for ${title}`);
  src = closeLast(src, '    </Card>\n  );\n}', '    </Card>\n    </PageContainer>\n  );\n}');
  save(out, src, hadCRLF, checkPC);
}

// mysql-backup
{
  let { s, hadCRLF } = load('mysql-backup-page.tsx');
  wrapCardTitle(
    { s, hadCRLF },
    {
      title: 'MySQL 备份',
      subTitle: '复用项目 SSH 凭据归档至 MinIO',
      out: 'ops/mysql-backup/index.tsx',
      name: 'MysqlBackupPage',
      shell: true,
    },
  );
}

// project-members
{
  let { s, hadCRLF } = load('project-members-page.tsx');
  wrapCardTitle(
    { s, hadCRLF },
    {
      title: '项目成员',
      subTitle: '项目内角色与资源授权（与全局 RBAC 独立）',
      out: 'project/members/index.tsx',
      name: 'ProjectMembersPage',
    },
  );
}

// service-catalog
{
  let { s, hadCRLF } = load('service-catalog-page.tsx');
  wrapCardTitle(
    { s, hadCRLF },
    {
      title: '服务目录',
      subTitle: '跨项目服务注册与绑定',
      out: 'project/service-catalog/index.tsx',
      name: 'ServiceCatalogPage',
    },
  );
}

// project-servers
{
  let { s, hadCRLF } = load('project-servers-page.tsx');
  s = s.replaceAll('from "./project-servers/', 'from "../../project-servers/');
  wrapCardTitle(
    { s, hadCRLF },
    {
      title: '服务器管理',
      subTitle: '自建与云服务器分组、同步与连接',
      out: 'project/servers/index.tsx',
      name: 'ProjectServersPage',
    },
  );
}

// Fix dbmgmt: write shared module + two route entries
{
  let { s, hadCRLF } = load('dbmgmt-console-page.tsx');
  s = ensurePC(s);
  s = s.replace(
    `  return (
    <Card title={pageTitle}>
`,
    `  return (
    <PageContainer header={{ title: pageTitle, subTitle: mode === "audit" ? "SQL 审计与历史" : "在线查询与执行" }}>
    <Card bordered={false}>
`,
  );
  const marker = '    </Card>\n  );\n}\n\nexport function DbmgmtSqlQueryPage()';
  if (!s.includes(marker)) throw new Error('dbmgmt-console close missing');
  s = s.replace(marker, '    </Card>\n    </PageContainer>\n  );\n}\n\nexport function DbmgmtSqlQueryPage()');
  save('dbmgmt/console-shared.tsx', s, hadCRLF, (x) => x.includes('PageContainer') && x.includes('DbmgmtConsolePage'));

  fs.mkdirSync(path.join(root, 'dbmgmt/sql-query'), { recursive: true });
  fs.writeFileSync(
    path.join(root, 'dbmgmt/sql-query/index.tsx'),
    `import { DbmgmtConsolePage } from '../console-shared';

export default function DbmgmtSqlQueryPage() {
  return <DbmgmtConsolePage mode="query" />;
}
`,
    'utf8',
  );
  console.log('dbmgmt/sql-query/index.tsx', { ok: true });

  fs.mkdirSync(path.join(root, 'dbmgmt/sql-audit'), { recursive: true });
  fs.writeFileSync(
    path.join(root, 'dbmgmt/sql-audit/index.tsx'),
    `import { DbmgmtConsolePage } from '../console-shared';

export default function DbmgmtSqlAuditPage() {
  return <DbmgmtConsolePage mode="audit" />;
}
`,
    'utf8',
  );
  console.log('dbmgmt/sql-audit/index.tsx', { ok: true });
}

// instance detail
{
  let { s, hadCRLF } = load('dbmgmt-instance-detail-page.tsx');
  s = ensurePC(s);
  s = toDefault(s, 'DbmgmtInstanceDetailPage');
  s = s.replace(
    `  return (
    <Card
      loading={loading}
      title={instance ? \`实例详情 · \${formatInstanceLabel(instance)}\` : "实例详情"}
`,
    `  return (
    <PageContainer
      header={{
        title: instance ? \`实例详情 · \${formatInstanceLabel(instance)}\` : "实例详情",
        subTitle: "库、用户与列脱敏",
      }}
    >
    <Card
      bordered={false}
      loading={loading}
`,
  );
  s = closeLast(s, '    </Card>\n  );\n}', '    </Card>\n    </PageContainer>\n  );\n}');
  save('dbmgmt/instance-detail/index.tsx', s, hadCRLF, checkPC);
}

// esmgmt console
{
  let { s, hadCRLF } = load('esmgmt-console-page.tsx');
  s = s.replace(/import \{ OpsPageHeader \} from "@\/components\/ops\/ops-page-header";\n/, '');
  s = ensurePC(s);
  s = toDefault(s, 'EsmgmtConsolePage');
  s = s.replace(
    `  return (
    <div className="page-stack">
      <OpsPageHeader
        title="ES REST 控制台"
        description="管理员受限代理：支持 GET/POST/PUT/DELETE/HEAD，禁止脚本执行与节点关机。"
        breadcrumbs={[{ title: "ES 管理控制台" }, { title: "REST 控制台" }]}
        extra={
          <Space>
            <Link to="/esmgmt/overview">集群概览</Link>
            <Link to="/esmgmt/connections">连接管理</Link>
          </Space>
        }
      />
`,
    `  return (
    <PageContainer
      header={{
        title: "ES REST 控制台",
        subTitle: "管理员受限代理：支持 GET/POST/PUT/DELETE/HEAD，禁止脚本执行与节点关机。",
        extra: (
          <Space>
            <Link to="/esmgmt/overview">集群概览</Link>
            <Link to="/esmgmt/connections">连接管理</Link>
          </Space>
        ),
      }}
    >
`,
  );
  s = closeLast(s, '    </div>\n  );\n}', '    </PageContainer>\n  );\n}');
  save('esmgmt/console/index.tsx', s, hadCRLF, checkPC);
}

// project-inspect
{
  let { s, hadCRLF } = load('project-inspect-page.tsx');
  s = s.replace(/import \{ OpsPageHeader \} from "@\/components\/ops\/ops-page-header";\n/, '');
  s = s.replaceAll('from "./inspect/', 'from "../../inspect/');
  s = ensurePC(s);
  s = toDefault(s, 'ProjectInspectPage');
  s = s.replace(
    `  return (
    <div className="page-stack project-inspect-page">
      <OpsPageHeader
        title="项目巡检"
        description="基于 Prometheus 采集指标定时/手动巡检，生成 HTML / Excel 报告；PDF 采用 html2canvas + jsPDF（与 PromAI 相同方案），样式与 HTML 预览一致。邮件默认可仅附 HTML。"
        breadcrumbs={[{ title: "项目运维" }, { title: "项目巡检" }]}
        meta={
          projectId ? (
            <Space wrap size="small">
              <Tag>{projectName || \`项目 #\${projectId}\`}</Tag>
              <Tag color={plan?.enabled ? "success" : "default"}>
                {plan?.enabled ? "定时已启用" : "定时未启用"}
              </Tag>
              <Tag>数据源 · {dsName}</Tag>
              {plan?.last_run_at ? <Tag>最近执行 · {formatDateTime(plan.last_run_at)}</Tag> : <Tag>尚未执行</Tag>}
              {recipients.length ? <Tag color="blue">邮件 · {recipients.length} 人</Tag> : <Tag>未配置收件人</Tag>}
            </Space>
          ) : null
        }
        extra={
          <>
            <Select
              style={{ width: 220 }}
              value={projectId || undefined}
              placeholder="选择项目"
              options={projects.map((p) => ({ label: p.name, value: p.id }))}
              onChange={(v) => setProjectId(v)}
              showSearch
              optionFilterProp="label"
            />
            <Button icon={<ReloadOutlined />} onClick={() => void refresh(projectId)} loading={loading}>
              刷新
            </Button>
            <Button
              type="primary"
              icon={<ThunderboltOutlined />}
              loading={running}
              disabled={!projectId}
              onClick={() => void handleImmediateRun()}
            >
              立即巡检
            </Button>
          </>
        }
      />
`,
    `  return (
    <PageContainer
      className="project-inspect-page"
      header={{
        title: "项目巡检",
        subTitle: "基于 Prometheus 指标定时/手动巡检，生成 HTML / Excel / PDF 报告",
        extra: (
          <>
            <Select
              style={{ width: 220 }}
              value={projectId || undefined}
              placeholder="选择项目"
              options={projects.map((p) => ({ label: p.name, value: p.id }))}
              onChange={(v) => setProjectId(v)}
              showSearch
              optionFilterProp="label"
            />
            <Button icon={<ReloadOutlined />} onClick={() => void refresh(projectId)} loading={loading}>
              刷新
            </Button>
            <Button
              type="primary"
              icon={<ThunderboltOutlined />}
              loading={running}
              disabled={!projectId}
              onClick={() => void handleImmediateRun()}
            >
              立即巡检
            </Button>
          </>
        ),
      }}
    >
`,
  );
  s = closeLast(s, '    </div>\n  );\n}', '    </PageContainer>\n  );\n}');
  save('project/inspect/index.tsx', s, hadCRLF, checkPC);
}

// project-logs
{
  let { s, hadCRLF } = load('project-logs-page.tsx');
  s = ensurePC(s);
  s = toDefault(s, 'ProjectLogsPage');
  s = s.replace(
    `  return (
    <div className="project-logs-page">
      <Card
        className="table-card project-logs-card"
        title="日志检索"
`,
    `  return (
    <PageContainer
      className="project-logs-page"
      header={{ title: "日志检索", subTitle: "主机 Agent 与集群 Loggie 统一检索" }}
    >
      <Card
        className="table-card project-logs-card"
        bordered={false}
`,
  );
  s = closeLast(s, '    </div>\n  );\n}', '    </PageContainer>\n  );\n}');
  save('project/logs/index.tsx', s, hadCRLF, checkPC);
}

// loggie-status
{
  let { s, hadCRLF } = load('loggie-status-page.tsx');
  s = ensurePC(s);
  s = toDefault(s, 'LoggieStatusPage');
  s = s.replace(
    `  return (
    <div className="loggie-status-page">
      <Card
        className="table-card"
        title="Agent 管理"
`,
    `  return (
    <PageContainer
      className="loggie-status-page"
      header={{ title: "Agent 管理", subTitle: "yunshu-agent 注册、热更与状态" }}
    >
      <Card
        className="table-card"
        bordered={false}
`,
  );
  s = closeLast(s, '    </div>\n  );\n}', '    </PageContainer>\n  );\n}');
  save('project/loggie-status/index.tsx', s, hadCRLF, checkPC);
}

// log-retention
{
  let { s, hadCRLF } = load('log-retention-page.tsx');
  s = ensurePC(s);
  s = toDefault(s, 'LogRetentionPage');
  s = s.replace(
    `  return (
    <div className="log-retention-page page-stack">
      <Tabs
`,
    `  return (
    <PageContainer
      className="log-retention-page"
      header={{ title: "日志保留", subTitle: "ES 索引保留与 Kafka 积压" }}
    >
      <Tabs
`,
  );
  s = closeLast(s, '    </div>\n  );\n}', '    </PageContainer>\n  );\n}');
  save('project/log-retention/index.tsx', s, hadCRLF, checkPC);
}

// collect-config
{
  let { s, hadCRLF } = load('project-collect-config-page.tsx');
  s = ensurePC(s);
  s = toDefault(s, 'ProjectCollectConfigPage');
  s = s.replace(
    'import { ProjectClusterLogPage } from "./project-cluster-log-page";\n',
    'import { ProjectClusterLogPage } from "@/pages/project-cluster-log-page";\n',
  );
  s = s.replace(
    'import { ProjectLogSourcesPage } from "./project-log-sources-page";\n',
    'import { ProjectLogSourcesPage } from "@/pages/project-log-sources-page";\n',
  );
  s = s.replace(
    'import { ProjectServicesPage } from "./project-services-page";\n',
    'import { ProjectServicesPage } from "@/pages/project-services-page";\n',
  );
  // after @/ rewrite the above might already be wrong — fix if already rewritten
  s = s.replaceAll('from "@/project-', 'from "@/pages/project-');
  // Actually load() already changed ./ to stay, and ../ to @/. Local ./ became unchanged then we replace.
  // If load already made them from "./project..." still there - good.
  s = s.replace(
    `  return (
    <Card className="table-card" title="服务与日志采集" styles={{ body: { paddingTop: 8 } }}>
`,
    `  return (
    <PageContainer header={{ title: "服务与日志采集", subTitle: "服务配置 · 主机日志源 · 集群采集" }}>
    <Card className="table-card" bordered={false} styles={{ body: { paddingTop: 8 } }}>
`,
  );
  s = closeLast(s, '    </Card>\n  );\n}', '    </Card>\n    </PageContainer>\n  );\n}');
  save('project/collect-config/index.tsx', s, hadCRLF, checkPC);
}

// service-portrait
{
  let { s, hadCRLF } = load('service-portrait-page.tsx');
  s = ensurePC(s);
  s = toDefault(s, 'ServicePortraitPage');
  s = s.replace(
    `  return (
    <Space direction="vertical" size="middle" style={{ width: "100%" }}>
`,
    `  return (
    <PageContainer header={{ title: "服务画像", subTitle: "健康分、入口与依赖关系" }}>
    <Space direction="vertical" size="middle" style={{ width: "100%" }}>
`,
  );
  // remove duplicate title on first Card
  s = s.replace(
    `      <Card
        title="服务画像"
        extra={
`,
    `      <Card
        bordered={false}
        extra={
`,
  );
  s = closeLast(s, '    </Space>\n  );\n}', '    </Space>\n    </PageContainer>\n  );\n}');
  save('project/service-portrait/index.tsx', s, hadCRLF, checkPC);
}

// ai-assistant
{
  let { s, hadCRLF } = load('ai-assistant-page.tsx');
  s = ensurePC(s);
  s = toDefault(s, 'AiAssistantPage');
  s = s.replace(
    `  return (
    <div style={{ display: "flex", gap: 12, alignItems: "stretch" }}>
`,
    `  return (
    <PageContainer header={{ title: "AI 运维助手", subTitle: "会话、工具调用与知识库命中" }}>
    <div style={{ display: "flex", gap: 12, alignItems: "stretch" }}>
`,
  );
  s = closeLast(s, '    </div>\n  );\n}', '    </div>\n    </PageContainer>\n  );\n}');
  save('ai/assistant/index.tsx', s, hadCRLF, checkPC);
}

// server-console — two returns with OpsPageHeader
{
  let { s, hadCRLF } = load('server-console-page.tsx');
  s = s.replace(/import \{ OpsPageHeader \} from "@\/components\/ops\/ops-page-header";\n/, '');
  s = ensurePC(s);
  s = toDefault(s, 'ServerConsolePage');
  s = s.replace(
    `    return (
      <div className="page-stack">
        <OpsPageHeader title="服务器控制台" description="终端、命令执行与文件传输" />
        <Card className="table-card">
          <Alert type="error" showIcon message="参数不完整" description="请从服务器管理页面点击“连接”进入。" />
        </Card>
      </div>
    );`,
    `    return (
      <PageContainer header={{ title: "服务器控制台", subTitle: "终端、命令执行与文件传输" }}>
        <Card className="table-card" bordered={false}>
          <Alert type="error" showIcon message="参数不完整" description="请从服务器管理页面点击“连接”进入。" />
        </Card>
      </PageContainer>
    );`,
  );
  s = s.replace(
    `  return (
    <div className="page-stack">
      <OpsPageHeader
        title="服务器控制台"
        description="SSH 终端、远程命令与文件传输；需对该服务器具备 SSH/执行授权。"
        breadcrumbs={[{ title: "项目运维" }, { title: "服务器控制台" }]}
        meta={
          <Space wrap size="small">
            <Tag color="blue">项目 #{projectId}</Tag>
            <Tag>服务器 #{serverId}</Tag>
            <Tag>{server?.source_type === "cloud" ? \`云 · \${server?.provider || "-"}\` : "自建"}</Tag>
            <Tag>{server?.host || "-"}</Tag>
            <Tag>
              {server?.auth_type === "password"
                ? "密码认证"
                : server?.auth_type === "key"
                  ? "密钥认证"
                  : server?.auth_type || "-"}
            </Tag>
          </Space>
        }
        extra={<Link to="/project-servers">返回服务器管理</Link>}
      />
`,
    `  return (
    <PageContainer
      header={{
        title: "服务器控制台",
        subTitle: "SSH 终端、远程命令与文件传输；需对该服务器具备 SSH/执行授权。",
        extra: <Link to="/project-servers">返回服务器管理</Link>,
      }}
    >
`,
  );
  s = closeLast(s, '    </div>\n  );\n}', '    </PageContainer>\n  );\n}');
  save('project/server-console/index.tsx', s, hadCRLF, (x) => {
    const open = (x.match(/<PageContainer\b/g) || []).length;
    const close = (x.match(/<\/PageContainer>/g) || []).length;
    return open === 2 && close === 2 && defaultCheck(x);
  });
}

// alert-config-center redirect
{
  let { s, hadCRLF } = load('alert-config-center-page.tsx');
  s = toDefault(s, 'AlertConfigCenterPage');
  save('alert/config-center/index.tsx', s, hadCRLF, (x) => x.includes('export default') && x.includes('Navigate'));
}

// alert-monitor: update layout to PageContainer + thin route page
{
  const layoutPath = path.join(root, 'alert-monitor/layout.tsx');
  let s = fs.readFileSync(layoutPath, 'utf8');
  const hadCRLF = s.includes('\r\n');
  s = s.replace(/\r\n/g, '\n');
  if (!s.includes('PageContainer')) {
    s = s.replace(
      'import { PageTelemetryHeader } from "../../components/page-telemetry-header";\n',
      'import { PageContainer } from "@ant-design/pro-components";\n',
    );
    s = s.replace(
      `  return (
    <div className="page-stack">
      <PageTelemetryHeader
        label="[ ALERT / ENGINE ]"
        title="告警平台"
        subtitle="数据源评测 · 规则中心 · 事件与降噪 · 通知（夜莺式引擎，无 Alertmanager）"
        meta={
          tab === "policies"
            ? [
                "路由 · 平台全局",
                \`分组 · \${groupLabel}\`,
                \`页签 · \${groupTabs.find((x) => x.key === tab)?.label || tab}\`,
                ctx.loading ? "同步中" : "已同步",
              ]
            : [
                ctx.projectContextId ? \`项目 · \${ctx.activeProjectName}\` : "项目 · 全部",
                \`分组 · \${groupLabel}\`,
                \`页签 · \${groupTabs.find((x) => x.key === tab)?.label || tab}\`,
                ctx.loading ? "同步中" : "已同步",
              ]
        }
      />
`,
      `  return (
    <PageContainer
      header={{
        title: "告警平台",
        subTitle: "数据源评测 · 规则中心 · 事件与降噪 · 通知（夜莺式引擎，无 Alertmanager）",
      }}
    >
`,
    );
    s = s.replace('    </div>\n  );\n}', '    </PageContainer>\n  );\n}');
    if (hadCRLF) s = s.replace(/\n/g, '\r\n');
    fs.writeFileSync(layoutPath, s, 'utf8');
    console.log('alert-monitor/layout.tsx', { ok: s.includes('PageContainer') });
  }

  const monitorPage = `// @ts-nocheck
export { AlertMonitorPlatformRoot as default } from '@/pages/alert-monitor/platform-provider';
`;
  fs.mkdirSync(path.join(root, 'alert/monitor'), { recursive: true });
  fs.writeFileSync(path.join(root, 'alert/monitor/index.tsx'), monitorPage, 'utf8');
  console.log('alert/monitor/index.tsx', { ok: true });
}

console.log(failed ? `promote-ops-batch FAILED (${failed})` : 'promote-ops-batch done');
process.exitCode = failed ? 1 : 0;
