import fs from 'node:fs';

const files = [
  'dbmgmt/access-requests',
  'dbmgmt/apply-database',
  'dbmgmt/apply-app-user',
  'dbmgmt/apply-query',
  'esmgmt/overview',
  'ai/center',
  'platform/templates',
];

for (const f of files) {
  const s = fs.readFileSync(`src/pages/${f}/index.tsx`, 'utf8');
  const open = (s.match(/<PageContainer\b/g) || []).length;
  const close = (s.match(/<\/PageContainer>/g) || []).length;
  const cn = /[\u4e00-\u9fff]{2}/.test(s);
  const def = s.includes('export default');
  console.log(f, { open, close, cn, def, ok: (f === 'dbmgmt/apply-query' ? def : open === 1 && close === 1 && cn && def) });
}
