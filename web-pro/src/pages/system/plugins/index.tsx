import type { ProColumns } from '@ant-design/pro-components';
import { PageContainer, ProTable } from '@ant-design/pro-components';
import { Tag } from 'antd';
import { LegacyShell } from '@/components/LegacyShell';
import { usePlugins } from '@/contexts/plugin-context';
import type { PluginInfo } from '@/services/plugins';

export default function PluginsPage() {
  return (
    <LegacyShell>
      <PluginsInner />
    </LegacyShell>
  );
}

function PluginsInner() {
  const { plugins, enabled, loading } = usePlugins();

  const columns: ProColumns<PluginInfo>[] = [
    {
      title: '插件',
      dataIndex: 'name',
      search: false,
      render: (_, row) => <Tag color="blue">{row.name}</Tag>,
    },
    {
      title: '说明',
      dataIndex: 'description',
      ellipsis: true,
      search: false,
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      width: 120,
      search: false,
      render: (_, row) =>
        row.enabled ? <Tag color="success">已启用</Tag> : <Tag>未启用</Tag>,
    },
  ];

  return (
    <PageContainer
      header={{
        title: '业务插件',
        subTitle: `与后端 config.yaml plugins.enabled 同步。当前启用：${
          enabled.length ? enabled.join(', ') : 'NONE'
        }`,
      }}
    >
      <ProTable<PluginInfo>
        rowKey="name"
        loading={loading}
        columns={columns}
        dataSource={plugins}
        search={false}
        options={false}
        pagination={false}
        toolBarRender={false}
      />
    </PageContainer>
  );
}
