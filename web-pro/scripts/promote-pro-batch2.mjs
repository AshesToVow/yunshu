import fs from 'node:fs';
import path from 'node:path';

const root = path.resolve('src/pages');

function promote({
  srcRel,
  outRel,
  title,
  subTitle,
  exportName,
  useLegacyShell = false,
  removeTelemetry = true,
  removeOpsHeader = false,
}) {
  let s = fs.readFileSync(path.join(root, srcRel), 'utf8');
  const hadCRLF = s.includes('\r\n');
  s = s.replace(/\r\n/g, '\n');
  s = s.replaceAll('from "../', 'from "@/').replaceAll("from '../", "from '@/");

  if (!s.includes('@ant-design/pro-components')) {
    // insert after first import line
    const firstNl = s.indexOf('\n');
    s =
      s.slice(0, firstNl + 1) +
      'import { PageContainer } from "@ant-design/pro-components";\n' +
      s.slice(firstNl + 1);
  }

  if (useLegacyShell && !s.includes('LegacyShell')) {
    s = s.replace(
      'import { PageContainer } from "@ant-design/pro-components";\n',
      'import { PageContainer } from "@ant-design/pro-components";\nimport { LegacyShell } from "@/components/LegacyShell";\n',
    );
  }

  if (removeTelemetry) {
    s = s.replace(/import \{ PageTelemetryHeader \} from "@\/components\/page-telemetry-header";\n/, '');
  }
  if (removeOpsHeader) {
    s = s.replace(/import \{ OpsPageHeader \} from "@\/components\/ops\/ops-page-header";\n/, '');
  }

  // named -> default (+ optional LegacyShell). Keep function signature/params.
  const namedRe = new RegExp(`export function ${exportName}(\\([^)]*\\))`);
  if (namedRe.test(s)) {
    if (useLegacyShell) {
      s = s.replace(
        namedRe,
        `export default function ${exportName}$1 {
  return (
    <LegacyShell>
      <${exportName}Inner />
    </LegacyShell>
  );
}

function ${exportName}Inner$1`,
      );
    } else {
      s = s.replace(namedRe, `export default function ${exportName}$1`);
    }
  }

  // Replace common page-stack + telemetry header openings is page-specific; do light wrap if still Card/div root
  if (!s.includes('<PageContainer')) {
    // OpsPageHeader pattern (dashboard)
    if (s.includes('<OpsPageHeader')) {
      s = s.replace(
        /return \(\s*<div className="page-stack">\s*<OpsPageHeader[\s\S]*?\/>/,
        `return (
    <PageContainer header={{ title: ${JSON.stringify(title)}, subTitle: ${JSON.stringify(subTitle)} }}>`,
      );
      // remove closing page-stack div before final );
      s = s.replace(/\n    <\/div>\n  \);\n}/, '\n    </PageContainer>\n  );\n}');
    } else if (s.includes('<PageTelemetryHeader')) {
      s = s.replace(
        /return \(\s*<div className="page-stack">\s*<PageTelemetryHeader[\s\S]*?\/>/,
        `return (
    <PageContainer header={{ title: ${JSON.stringify(title)}, subTitle: ${JSON.stringify(subTitle)} }}>`,
      );
      s = s.replace(/\n    <\/div>\n  \);\n}/, '\n    </PageContainer>\n  );\n}');
    } else if (s.includes('return (\n    <Space direction="vertical"')) {
      // alert-quality style
      s = s.replace(
        'return (\n    <Space direction="vertical" size="middle" style={{ width: "100%" }}>',
        `return (
    <PageContainer header={{ title: ${JSON.stringify(title)}, subTitle: ${JSON.stringify(subTitle)} }}>
    <Space direction="vertical" size="middle" style={{ width: "100%" }}>`,
      );
      s = s.replace(/\n    <\/Space>\n  \);\n}/, '\n    </Space>\n    </PageContainer>\n  );\n}');
    }
  }

  if (hadCRLF) s = s.replace(/\n/g, '\r\n');
  const out = path.join(root, outRel);
  fs.mkdirSync(path.dirname(out), { recursive: true });
  fs.writeFileSync(out, s, 'utf8');

  const ok = s.includes('PageContainer') && /[\u4e00-\u9fff]{2}/.test(s) && s.includes('export default');
  console.log(outRel, {
    ok,
    pageC: s.includes('PageContainer'),
    cn: /[\u4e00-\u9fff]{2}/.test(s),
    def: s.includes('export default'),
  });
  if (!ok) process.exitCode = 1;
}

promote({
  srcRel: 'alert-quality-page.tsx',
  outRel: 'alert/quality/index.tsx',
  title: '告警质量治理',
  subTitle: '噪声、覆盖与响应质量概览',
  exportName: 'AlertQualityPage',
});

promote({
  srcRel: 'dashboard-page.tsx',
  outRel: 'dashboard/index.tsx',
  title: '运营总览',
  subTitle: '平台资源与发布态势',
  exportName: 'DashboardPage',
  removeOpsHeader: true,
});

promote({
  srcRel: 'workflow-definitions-page.tsx',
  outRel: 'workflow/definitions/index.tsx',
  title: '审批流程定义',
  subTitle: '按域配置审批阶段与用户组',
  exportName: 'WorkflowDefinitionsPage',
});

console.log('promote-batch2 done');
