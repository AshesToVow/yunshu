import fs from 'node:fs';
import path from 'node:path';

const root = path.resolve('src/pages');

function load(srcRel) {
  let s = fs.readFileSync(path.join(root, srcRel), 'utf8');
  const hadCRLF = s.includes('\r\n');
  s = s.replace(/\r\n/g, '\n');
  s = s.replaceAll('from "../', 'from "@/').replaceAll("from '../", "from '@/");
  return { s, hadCRLF };
}

function ensurePageContainerImport(s) {
  if (s.includes('@ant-design/pro-components')) return s;
  const nl = s.indexOf('\n');
  return s.slice(0, nl + 1) + 'import { PageContainer } from "@ant-design/pro-components";\n' + s.slice(nl + 1);
}

function save(outRel, s, hadCRLF) {
  if (hadCRLF) s = s.replace(/\n/g, '\r\n');
  const out = path.join(root, outRel);
  fs.mkdirSync(path.dirname(out), { recursive: true });
  fs.writeFileSync(out, s, 'utf8');
  const open = (s.match(/<PageContainer\b/g) || []).length;
  const close = (s.match(/<\/PageContainer>/g) || []).length;
  const ok = open === 1 && close === 1 && /[\u4e00-\u9fff]{2}/.test(s) && s.includes('export default');
  console.log(outRel, { ok, open, close, cn: /[\u4e00-\u9fff]{2}/.test(s) });
  if (!ok) process.exitCode = 1;
}

// ---- access requests (also query apply) ----
{
  let { s, hadCRLF } = load('dbmgmt-access-requests-page.tsx');
  s = ensurePageContainerImport(s);
  s = s.replace(
    'export function DbmgmtAccessRequestsPage({ preset = "all" }: { preset?: DbmgmtAccessRequestPreset }) {',
    'export default function DbmgmtAccessRequestsPage({ preset = "all" }: { preset?: DbmgmtAccessRequestPreset }) {',
  );
  // keep named query apply pointing to default
  s = s.replace(
    'export function DbmgmtQueryApplyPage() {\n  return <DbmgmtAccessRequestsPage preset="query" />;\n}',
    'export function DbmgmtQueryApplyPage() {\n  return <DbmgmtAccessRequestsPage preset="query" />;\n}\n\nexport { DbmgmtAccessRequestsPage };',
  );
  // If we already have export default function, the re-export line might conflict - fix:
  // Actually `export { DbmgmtAccessRequestsPage }` after default export of same name is invalid in some bundlers.
  // Better: don't re-export; QueryApplyPage uses the local function name which works for default export too in same file.
  s = s.replace('\n\nexport { DbmgmtAccessRequestsPage };', '');

  const from = `  return (
    <Card
      title={meta.title}
`;
  const to = `  return (
    <PageContainer header={{ title: meta.title, subTitle: meta.hint }}>
    <Card
      bordered={false}
      title={meta.showList === false ? undefined : meta.title}
`;
  if (!s.includes(from)) throw new Error('access-requests return block missing');
  s = s.replace(from, to);
  const marker = '    </Card>\n  );\n}\n\nexport function DbmgmtQueryApplyPage';
  if (!s.includes(marker)) throw new Error('access-requests close missing');
  s = s.replace(marker, '    </Card>\n    </PageContainer>\n  );\n}\n\nexport function DbmgmtQueryApplyPage');
  save('dbmgmt/access-requests/index.tsx', s, hadCRLF);
}

// query apply thin route page
{
  const content = `import DbmgmtAccessRequestsPage from '../access-requests';

export default function DbmgmtQueryApplyPage() {
  return <DbmgmtAccessRequestsPage preset="query" />;
}
`;
  fs.mkdirSync(path.join(root, 'dbmgmt/apply-query'), { recursive: true });
  fs.writeFileSync(path.join(root, 'dbmgmt/apply-query/index.tsx'), content, 'utf8');
  console.log('dbmgmt/apply-query/index.tsx', { ok: true });
}

// ---- database apply ----
{
  let { s, hadCRLF } = load('dbmgmt-database-apply-page.tsx');
  s = ensurePageContainerImport(s);
  s = s.replace('export function DbmgmtDatabaseApplyPage()', 'export default function DbmgmtDatabaseApplyPage()');
  const from = `  return (
    <Card title="数据库创建申请">
`;
  const to = `  return (
    <PageContainer header={{ title: "数据库创建申请", subTitle: "提交新建库工单" }}>
    <Card bordered={false}>
`;
  if (!s.includes(from)) throw new Error('database-apply return missing');
  s = s.replace(from, to);
  const marker = '    </Card>\n  );\n}';
  const idx = s.lastIndexOf(marker);
  if (idx < 0) throw new Error('database-apply close missing');
  s = s.slice(0, idx) + '    </Card>\n    </PageContainer>\n  );\n}';
  save('dbmgmt/apply-database/index.tsx', s, hadCRLF);
}

// ---- app user apply ----
{
  let { s, hadCRLF } = load('dbmgmt-app-user-apply-page.tsx');
  s = ensurePageContainerImport(s);
  s = s.replace('export function DbmgmtAppUserApplyPage()', 'export default function DbmgmtAppUserApplyPage()');
  // find Card title return
  const cardTitleMatch = s.match(/  return \(\n    <Card\n      title="[^"]+"/);
  if (!cardTitleMatch) {
    // try single-line Card title
    if (s.includes('  return (\n    <Card title=')) {
      s = s.replace(
        /  return \(\n    <Card title="([^"]+)">\n/,
        '  return (\n    <PageContainer header={{ title: "$1" }}>\n    <Card bordered={false}>\n',
      );
    } else {
      throw new Error('app-user-apply return Card not found');
    }
  } else {
    const title = (s.match(/title="([^"]+)"/) || [])[1] || '应用用户权限申请';
    s = s.replace(
      /  return \(\n    <Card\n      title="[^"]+"[^\n]*\n/,
      `  return (\n    <PageContainer header={{ title: "${title}" }}>\n    <Card\n      bordered={false}\n`,
    );
  }
  const marker = '    </Card>\n  );\n}';
  const idx = s.lastIndexOf(marker);
  if (idx < 0) throw new Error('app-user-apply close missing');
  s = s.slice(0, idx) + '    </Card>\n    </PageContainer>\n  );\n}';
  save('dbmgmt/apply-app-user/index.tsx', s, hadCRLF);
}

// ---- esmgmt overview ----
{
  let { s, hadCRLF } = load('esmgmt-overview-page.tsx');
  s = ensurePageContainerImport(s);
  s = s.replace('export function EsmgmtOverviewPage()', 'export default function EsmgmtOverviewPage()');
  // already has export default at end sometimes
  s = s.replace(/\nexport default EsmgmtOverviewPage;\n?$/, '\n');
  const from = `  return (
    <Space direction="vertical" style={{ width: "100%" }} size="middle">
`;
  const to = `  return (
    <PageContainer header={{ title: "ES 集群概览", subTitle: "索引、备份、恢复与定时任务" }}>
    <Space direction="vertical" style={{ width: "100%" }} size="middle">
`;
  if (!s.includes(from)) throw new Error('esmgmt return missing');
  s = s.replace(from, to);
  const marker = '    </Space>\n  );\n}';
  const idx = s.lastIndexOf(marker);
  if (idx < 0) throw new Error('esmgmt close missing');
  s = s.slice(0, idx) + '    </Space>\n    </PageContainer>\n  );\n}';
  save('esmgmt/overview/index.tsx', s, hadCRLF);
}

// ---- ai center ----
{
  let { s, hadCRLF } = load('ai-center-page.tsx');
  s = ensurePageContainerImport(s);
  s = s.replace('import { OpsPageHeader } from "@/components/ops/ops-page-header";\n', '');
  s = s.replace('export function AiCenterPage()', 'export default function AiCenterPage()');
  s = s.replace(/\nexport default AiCenterPage;\n?$/, '\n');
  const from = `  return (
    <div className="page-stack">
      <OpsPageHeader
        title="AI 运维能力中心"
        description="模型、Prompt、工具、知识库与 Evaluation 统一管理；同步种子后可将知识写入 ES 供助手 RAG。"
        breadcrumbs={[{ title: "AI" }, { title: "能力中心" }]}
        extra={
`;
  // Keep extra toolbar by moving into PageContainer header.extra — extract until closing of OpsPageHeader is hard.
  // Simpler: PageContainer + keep extra buttons in a Space below header by replacing OpsPageHeader with fragment of extra only.
  if (!s.includes(from)) {
    // fallback looser
    if (!s.includes('<OpsPageHeader')) throw new Error('ai-center OpsPageHeader missing');
  }
  s = s.replace(
    from,
    `  return (
    <PageContainer
      header={{
        title: "AI 运维能力中心",
        subTitle: "模型、Prompt、工具、知识库与 Evaluation 统一管理；同步种子后可将知识写入 ES 供助手 RAG。",
        extra: (
`,
  );
  // OpsPageHeader closed with `/>` after extra={...} — convert closing
  // Original structure:
  // <OpsPageHeader ... extra={ <Space>...</Space> } />
  // After our replace, we have extra: ( <Space>... need to close with ), }} >
  // Find the OpsPageHeader self-close after extra
  s = s.replace(
    /(\n            <Button loading=\{loading\} onClick=\{\(\) => void handleEval\(true\)\}>\n              在线 Evaluation\n            <\/Button>\n          <\/Space>\n        )\n      \/>\n/,
    `$1
        ),
      }}
    >
`,
  );
  // If above didn't match, try generic
  if (s.includes('<OpsPageHeader') || s.includes('      />\n') && s.includes('page-stack')) {
    // try another pattern for closing OpsPageHeader
  }
  const closeMarker = '    </div>\n  );\n}';
  const idx = s.lastIndexOf(closeMarker);
  if (idx < 0) throw new Error('ai-center close missing');
  s = s.slice(0, idx) + '    </PageContainer>\n  );\n}';
  save('ai/center/index.tsx', s, hadCRLF);
}

// ---- platform templates ----
{
  let { s, hadCRLF } = load('platform-templates-page.tsx');
  s = ensurePageContainerImport(s);
  s = s.replace('import { PageTelemetryHeader } from "@/components/page-telemetry-header";\n', '');
  s = s.replace('export function PlatformTemplatesPage()', 'export default function PlatformTemplatesPage()');
  const from = `  return (
    <div>
      <PageTelemetryHeader label="Templates" title="模板中心" subtitle="CI/CD 片段 · 告警 · 巡检 · Loggie（MySQL 权威 + MinIO 镜像）" />
`;
  const to = `  return (
    <PageContainer
      header={{
        title: "模板中心",
        subTitle: "CI/CD 片段 · 告警 · 巡检 · Loggie（MySQL 权威 + MinIO 镜像）",
      }}
    >
`;
  if (!s.includes(from)) throw new Error('platform-templates return missing');
  s = s.replace(from, to);
  const marker = '    </div>\n  );\n}';
  const idx = s.lastIndexOf(marker);
  if (idx < 0) throw new Error('platform-templates close missing');
  s = s.slice(0, idx) + '    </PageContainer>\n  );\n}';
  save('platform/templates/index.tsx', s, hadCRLF);
}

console.log('promote-batch3 done');
