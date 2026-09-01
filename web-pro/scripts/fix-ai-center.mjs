import fs from 'node:fs';

const p = 'src/pages/ai/center/index.tsx';
let s = fs.readFileSync(p, 'utf8');
const had = s.includes('\r\n');
s = s.replace(/\r\n/g, '\n');

const bad = `          </Space>
        }
      />
      <Card className="table-card">`;

const good = `          </Space>
        ),
      }}
    >
      <Card className="table-card" bordered={false}>`;

if (!s.includes(bad)) {
  console.error('bad block missing');
  const i = s.indexOf('</Space>');
  console.log(JSON.stringify(s.slice(i, i + 160)));
  process.exit(1);
}

s = s.replace(bad, good);
if (had) s = s.replace(/\n/g, '\r\n');
fs.writeFileSync(p, s, 'utf8');
console.log({
  open: (s.match(/<PageContainer\b/g) || []).length,
  close: (s.match(/<\/PageContainer>/g) || []).length,
  leftoverSelfClose: s.includes('        }\n      />\n      <Card'),
});
