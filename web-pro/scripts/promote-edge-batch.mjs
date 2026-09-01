/**
 * Sweep remaining edge pages: redirects + ticket detail promote.
 * Run: node scripts/promote-edge-batch.mjs
 */
import fs from 'node:fs';
import path from 'node:path';

const root = path.resolve('src/pages');

function write(rel, content) {
  const out = path.join(root, rel);
  fs.mkdirSync(path.dirname(out), { recursive: true });
  fs.writeFileSync(out, content, 'utf8');
  console.log(rel, { ok: true });
}

const redirects = [
  ['redirects/cicd-todo/index.tsx', '/workflow/inbox?domain=cicd'],
  ['redirects/dbmgmt-todo/index.tsx', '/workflow/inbox?domain=dbmgmt'],
  ['redirects/cicd-approval-flow/index.tsx', '/workflow/definitions?domain=cicd'],
  ['redirects/dbmgmt-approval-flow/index.tsx', '/workflow/definitions?domain=dbmgmt'],
  ['redirects/project-cluster-log/index.tsx', '/project-services?tab=cluster-log'],
];

for (const [rel, to] of redirects) {
  write(
    rel,
    `import { Navigate } from '@umijs/max';

/** Legacy menu/component compatibility redirect */
export default function LegacyRedirect() {
  return <Navigate to="${to}" replace />;
}
`,
  );
}

// ticket detail → PageContainer
{
  let s = fs.readFileSync(path.join(root, 'dbmgmt-ticket-detail-page.tsx'), 'utf8');
  const hadCRLF = s.includes('\r\n');
  s = s.replace(/\r\n/g, '\n');
  s = s.replaceAll('from "../', 'from "@/').replaceAll("from '../", "from '@/");
  if (!s.includes('@ant-design/pro-components')) {
    s = s.replace('\n', '\nimport { PageContainer } from "@ant-design/pro-components";\n');
  }
  s = s.replace('export function DbmgmtTicketDetailPage()', 'export default function DbmgmtTicketDetailPage()');
  s = s.replace(/\nexport default DbmgmtTicketDetailPage;\n?$/, '\n');

  const from = `  return (
    <Card
      loading={loading}
      title="工单任务详情"
      extra={
`;
  const to = `  return (
    <PageContainer
      header={{
        title: "工单任务详情",
        subTitle: "审批、执行与回滚",
        extra: (
`;
  if (!s.includes(from)) throw new Error('ticket-detail return missing');
  s = s.replace(from, to);

  // close extra={ Space...}  then Card children — original ends extra with `}\n    >`
  // After replace we have extra: ( <Space>... need ), }} > then Card without title
  s = s.replace(
    /(\n          <Button onClick=\{\(\) => navigate\(projectId \? `\/dbmgmt\/workflow\/history\?project=\$\{projectId\}` : "\/dbmgmt\/workflow\/history"\)\}>返回历史工单<\/Button>\n        <\/Space>\n      )\n    >\n/,
    `$1
        ),
      }}
    >
    <Card bordered={false} loading={loading}>
`,
  );

  if (!s.includes('<PageContainer') || s.includes('title="工单任务详情"')) {
    // fallback: if pattern failed, try looser
    if (s.includes('title="工单任务详情"')) {
      throw new Error('ticket-detail Card title still present — close pattern failed');
    }
  }

  const marker = '    </Card>\n  );\n}';
  const idx = s.lastIndexOf(marker);
  if (idx < 0) throw new Error('ticket-detail close missing');
  s = s.slice(0, idx) + '    </Card>\n    </PageContainer>\n  );\n}';

  if (hadCRLF) s = s.replace(/\n/g, '\r\n');
  write('dbmgmt/ticket-detail/index.tsx', s);
  const open = (s.match(/<PageContainer\b/g) || []).length;
  const close = (s.match(/<\/PageContainer>/g) || []).length;
  console.log('ticket-detail check', { open, close, ok: open === 1 && open === close });
  if (open !== close) process.exitCode = 1;
}

console.log('promote-edge-batch done');
