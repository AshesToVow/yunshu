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

function save(outRel, s, hadCRLF) {
  if (hadCRLF) s = s.replace(/\n/g, '\r\n');
  const out = path.join(root, outRel);
  fs.mkdirSync(path.dirname(out), { recursive: true });
  fs.writeFileSync(out, s, 'utf8');
  const ok =
    (s.match(/<PageContainer/g) || []).length === 1 &&
    (s.match(/<\/PageContainer>/g) || []).length === 1 &&
    /[\u4e00-\u9fff]{2}/.test(s) &&
    s.includes('export default');
  console.log(outRel, { ok, opens: (s.match(/<PageContainer/g) || []).length, closes: (s.match(/<\/PageContainer>/g) || []).length });
  if (!ok) process.exitCode = 1;
}

// dashboard
{
  let { s, hadCRLF } = load('dashboard-page.tsx');
  s = s.replace(
    'import {\n  AlertOutlined,',
    'import { PageContainer } from "@ant-design/pro-components";\nimport {\n  AlertOutlined,',
  );
  s = s.replace('import { OpsPageHeader } from "@/components/ops/ops-page-header";\n', '');
  s = s.replace('export function DashboardPage()', 'export default function DashboardPage()');
  s = s.replace(
    `  return (
    <div className="page-stack dashboard-page">
      <OpsPageHeader
        title={t("dashboard.title")}
        description={t("dashboard.subtitle")}
        meta={
          <Typography.Text type="secondary">
            {loading ? t("dashboard.syncPending") : loadError ? "同步失败" : t("dashboard.syncLive")}
          </Typography.Text>
        }
      />

`,
    `  return (
    <PageContainer
      header={{
        title: t("dashboard.title"),
        subTitle: t("dashboard.subtitle"),
      }}
      className="dashboard-page"
    >
`,
  );
  // replace final closing </div> of page-stack only
  const lastDiv = s.lastIndexOf('    </div>\n  );\n}');
  if (lastDiv < 0) throw new Error('dashboard close not found');
  s = s.slice(0, lastDiv) + '    </PageContainer>\n  );\n}';
  save('dashboard/index.tsx', s, hadCRLF);
}

// workflow definitions
{
  let { s, hadCRLF } = load('workflow-definitions-page.tsx');
  s = s.replace(
    'import { ArrowDownOutlined,',
    'import { PageContainer } from "@ant-design/pro-components";\nimport { ArrowDownOutlined,',
  );
  s = s.replace('import { PageTelemetryHeader } from "@/components/page-telemetry-header";\n', '');
  s = s.replace('export function WorkflowDefinitionsPage()', 'export default function WorkflowDefinitionsPage()');
  s = s.replace(
    `  return (
    <div className="page-stack">
      <PageTelemetryHeader label="Workflow" title="审批流配置" subtitle="统一纳管各业务域审批节点与审批人用户组" />
`,
    `  return (
    <PageContainer
      header={{
        title: "审批流配置",
        subTitle: "统一纳管各业务域审批节点与审批人用户组",
      }}
    >
`,
  );
  const lastDiv = s.lastIndexOf('    </div>\n  );\n}');
  if (lastDiv < 0) throw new Error('workflow close not found');
  s = s.slice(0, lastDiv) + '    </PageContainer>\n  );\n}';
  save('workflow/definitions/index.tsx', s, hadCRLF);
}

console.log('fix promote done');
