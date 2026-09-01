import fs from 'node:fs';
import path from 'node:path';

const root = path.resolve('src/pages');
let s = fs.readFileSync(path.join(root, 'workflow-definitions-page.tsx'), 'utf8');
const hadCRLF = s.includes('\r\n');
s = s.replace(/\r\n/g, '\n');
s = s.replaceAll('from "../', 'from "@/').replaceAll("from '../", "from '@/");

if (!s.includes('PageContainer')) {
  s = s.replace(
    'import { ArrowDownOutlined,',
    'import { PageContainer } from "@ant-design/pro-components";\nimport { ArrowDownOutlined,',
  );
}
s = s.replace('import { PageTelemetryHeader } from "@/components/page-telemetry-header";\n', '');
s = s.replace(
  'export function WorkflowDefinitionsPage()',
  'export default function WorkflowDefinitionsPage()',
);

const from = `  return (
    <div>
      <PageTelemetryHeader label="Workflow" title="审批流配置" subtitle="统一纳管各业务域审批节点与审批人用户组" />
`;
const to = `  return (
    <PageContainer
      header={{
        title: "审批流配置",
        subTitle: "统一纳管各业务域审批节点与审批人用户组",
      }}
    >
`;

if (!s.includes(from)) {
  console.error('from block missing');
  process.exit(1);
}
s = s.replace(from, to);

const marker = '    </div>\n  );\n}';
const lastDiv = s.lastIndexOf(marker);
if (lastDiv < 0) {
  console.error('close missing');
  process.exit(1);
}
s = s.slice(0, lastDiv) + '    </PageContainer>\n  );\n}';

if (hadCRLF) s = s.replace(/\n/g, '\r\n');
fs.writeFileSync(path.join(root, 'workflow/definitions/index.tsx'), s, 'utf8');

console.log({
  open: (s.match(/<PageContainer/g) || []).length,
  close: (s.match(/<\/PageContainer>/g) || []).length,
  tele: s.includes('PageTelemetryHeader'),
  cn: s.includes('审批流配置'),
});
