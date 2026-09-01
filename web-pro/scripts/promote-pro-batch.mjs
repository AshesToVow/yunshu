import fs from 'node:fs';
import path from 'node:path';

const root = path.resolve('src/pages');

function write(rel, content) {
  const abs = path.join(root, rel);
  fs.mkdirSync(path.dirname(abs), { recursive: true });
  fs.writeFileSync(abs, content, 'utf8');
  console.log('wrote', rel);
}

function rewriteImports(s) {
  return s.replaceAll('from "../', 'from "@/').replaceAll("from '../", "from '@/");
}

// ---- alert channels ----
{
  let s = fs.readFileSync(path.join(root, 'alert-channels-page.tsx'), 'utf8');
  s = rewriteImports(s);
  s = s.replace('import { PageTelemetryHeader } from "@/components/page-telemetry-header";\n', '');
  s = s.replace(
    'import { DeleteOutlined, EditOutlined, PlusOutlined, SendOutlined } from "@ant-design/icons";\n',
    'import { DeleteOutlined, EditOutlined, PlusOutlined, SendOutlined } from "@ant-design/icons";\nimport { PageContainer } from "@ant-design/pro-components";\n',
  );
  s = s.replace('export function AlertChannelsPage()', 'export default function AlertChannelsPage()');
  s = s.replace(
    `  return (
    <div className="page-stack">
      <PageTelemetryHeader
        label="[ ALERT / CHANNEL ]"
        title="Webhook 告警通道"
        subtitle="配置 Webhook 投递端点、超时策略与告警模板绑定"
        meta={[
          \`COUNT / \${list.length}\`,
          loading ? "SYNC / PENDING" : "SYNC / OK",
        ]}
      />
    <Card className="table-card">`,
    `  return (
    <PageContainer
      header={{
        title: "Webhook 告警通道",
        subTitle: "配置 Webhook 投递端点、超时策略与告警模板绑定",
      }}
    >
    <Card className="table-card" bordered={false}>`,
  );
  s = s.replace(
    `    </Card>
    </div>
  );
}`,
    `    </Card>
    </PageContainer>
  );
}`,
  );
  write('alert/channels/index.tsx', s);
  if (!s.includes('告警通道') || s.includes('PageTelemetryHeader')) {
    throw new Error('channels rewrite failed');
  }
}

// ---- alert duty ----
{
  let s = fs.readFileSync(path.join(root, 'alert-duty-page.tsx'), 'utf8');
  s = rewriteImports(s);
  s = s.replace('import { PageTelemetryHeader } from "@/components/page-telemetry-header";\n', '');
  s = s.replace(
    'import { DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined, SwapOutlined } from "@ant-design/icons";\n',
    'import { DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined, SwapOutlined } from "@ant-design/icons";\nimport { PageContainer } from "@ant-design/pro-components";\n',
  );
  s = s.replace('export function AlertDutyPage()', 'export default function AlertDutyPage()');
  s = s.replace(
    `  return (
    <div className="page-stack">
      <PageTelemetryHeader
        label="[ ALERT / DUTY ]"
        title="值班总览"
        subtitle="班次甘特、日历排班与告警规则维度筛选"
        meta={[
          \`MODE / \${viewMode === "day" ? "DAY" : "WEEK"}\`,
          \`ANCHOR / \${anchorDate.format("YYYY-MM-DD")}\`,
        ]}
        extra={
          <Space wrap>
            <Select allowClear style={{ width: 240 }} options={projectOptions} value={projectId} onChange={setProjectId} placeholder="项目维度（可选）" />
            <Select allowClear style={{ width: 320 }} options={ruleOptions} value={ruleId} onChange={setRuleId} placeholder="规则维度（可选）" />
            <Segmented
              value={viewMode}
              options={[
                { label: "按天", value: "day" },
                { label: "按周", value: "week" },
              ]}
              onChange={(v) => setViewMode(v as "day" | "week")}
            />
            <DatePicker value={anchorDate} onChange={(v) => setAnchorDate(v || dayjs())} />
            <Button icon={<ReloadOutlined />} onClick={() => { void loadBlocks(); void loadCalendar(); }}>
              刷新
            </Button>
            <Button onClick={() => (window.location.href = "/alert-monitor-platform/rules")}>去配置规则</Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
              新增值班
            </Button>
          </Space>
        }
      />
    <Card className="table-card">`,
    `  return (
    <PageContainer
      header={{
        title: "值班总览",
        subTitle: "班次甘特、日历排班与告警规则维度筛选",
        extra: (
          <Space wrap>
            <Select allowClear style={{ width: 240 }} options={projectOptions} value={projectId} onChange={setProjectId} placeholder="项目维度（可选）" />
            <Select allowClear style={{ width: 320 }} options={ruleOptions} value={ruleId} onChange={setRuleId} placeholder="规则维度（可选）" />
            <Segmented
              value={viewMode}
              options={[
                { label: "按天", value: "day" },
                { label: "按周", value: "week" },
              ]}
              onChange={(v) => setViewMode(v as "day" | "week")}
            />
            <DatePicker value={anchorDate} onChange={(v) => setAnchorDate(v || dayjs())} />
            <Button icon={<ReloadOutlined />} onClick={() => { void loadBlocks(); void loadCalendar(); }}>
              刷新
            </Button>
            <Button onClick={() => (window.location.href = "/alert-monitor-platform/rules")}>去配置规则</Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
              新增值班
            </Button>
          </Space>
        ),
      }}
    >
    <Card className="table-card" bordered={false}>`,
  );
  s = s.replace(
    `    </Card>
    </div>
  );
}

function parseUintArrayJSON`,
    `    </Card>
    </PageContainer>
  );
}

function parseUintArrayJSON`,
  );
  write('alert/duty/index.tsx', s);
  if (!s.includes('值班总览') || s.includes('PageTelemetryHeader')) {
    throw new Error('duty rewrite failed');
  }
}

// ---- personal settings ----
{
  let s = fs.readFileSync(path.join(root, 'personal-settings-page.tsx'), 'utf8');
  s = rewriteImports(s);
  s = s.replace(
    'import { Alert, Card, Form, Input, Menu, Button, message } from "antd";\n',
    'import { PageContainer } from "@ant-design/pro-components";\nimport { Alert, Card, Form, Input, Menu, Button, message } from "antd";\n',
  );
  s = s.replace(
    'import { useAuth } from "@/contexts/auth-context";\n',
    'import { LegacyShell } from "@/components/LegacyShell";\nimport { useAuth } from "@/contexts/auth-context";\n',
  );
  s = s.replace(
    `export function PersonalSettingsPage() {
  const { user, refreshUser } = useAuth();`,
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
  s = s.replace(
    `  return (
    <Card className="table-card personal-settings-card">`,
    `  return (
    <PageContainer header={{ title: "个人设置", subTitle: "基本资料与密码修改" }}>
    <Card className="table-card personal-settings-card" bordered={false}>`,
  );
  s = s.replace(
    `      </div>
    </Card>
  );
}`,
    `      </div>
    </Card>
    </PageContainer>
  );
}`,
  );
  write('system/personal-settings/index.tsx', s);
  if (!s.includes('个人设置') || !s.includes('LegacyShell')) {
    throw new Error('personal-settings rewrite failed');
  }
}

console.log('promote-pro-batch done');
