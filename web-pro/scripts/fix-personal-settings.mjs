import fs from 'node:fs';
import path from 'node:path';

const root = path.resolve('src/pages');
let s = fs.readFileSync(path.join(root, 'personal-settings-page.tsx'), 'utf8');

// Normalize to LF for deterministic replaces, restore CRLF at end if needed
const hadCRLF = s.includes('\r\n');
s = s.replace(/\r\n/g, '\n');

s = s.replaceAll('from "../', 'from "@/').replaceAll("from '../", "from '@/");

if (!s.includes('PageContainer')) {
  s = s.replace(
    'import { Alert, Card, Form, Input, Menu, Button, message } from "antd";\n',
    'import { PageContainer } from "@ant-design/pro-components";\nimport { Alert, Card, Form, Input, Menu, Button, message } from "antd";\n',
  );
}

if (!s.includes('LegacyShell')) {
  s = s.replace(
    'import { useAuth } from "@/contexts/auth-context";\n',
    'import { LegacyShell } from "@/components/LegacyShell";\nimport { useAuth } from "@/contexts/auth-context";\n',
  );
}

if (s.includes('export function PersonalSettingsPage()')) {
  s = s.replace(
    'export function PersonalSettingsPage() {\n  const { user, refreshUser } = useAuth();',
    `export default function PersonalSettingsPage() {
  return (
    <LegacyShell>
      <PersonalSettingsInner />
    </LegacyShell>
  );
}

function PersonalSettingsInner() {
  const { user, refreshUser } = useAuth();`,
  );
}

if (!s.includes('<PageContainer')) {
  s = s.replace(
    '  return (\n    <Card className="table-card personal-settings-card">',
    '  return (\n    <PageContainer header={{ title: "个人设置", subTitle: "基本资料与密码修改" }}>\n    <Card className="table-card personal-settings-card" bordered={false}>',
  );
  s = s.replace('      </div>\n    </Card>\n  );\n}', '      </div>\n    </Card>\n    </PageContainer>\n  );\n}');
}

if (hadCRLF) s = s.replace(/\n/g, '\r\n');

const out = path.join(root, 'system/personal-settings/index.tsx');
fs.mkdirSync(path.dirname(out), { recursive: true });
fs.writeFileSync(out, s, 'utf8');

const ok =
  s.includes('个人设置') &&
  s.includes('基本设置') &&
  s.includes('LegacyShell') &&
  s.includes('PageContainer') &&
  s.includes('export default function PersonalSettingsPage');

console.log({
  ok,
  personal: s.includes('个人设置'),
  basic: s.includes('基本设置'),
  legacy: s.includes('LegacyShell'),
  pageC: s.includes('PageContainer'),
  def: s.includes('export default function PersonalSettingsPage'),
});
if (!ok) process.exit(1);
