import { ExperimentOutlined, ReloadOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { PageContainer, ProTable } from '@ant-design/pro-components';
import { Link, useSearchParams } from '@umijs/max';
import { Alert, Button, Card, Descriptions, Space, Tag, Typography, message } from 'antd';
import { useEffect, useRef, useState } from 'react';
import {
  getAIInvestigation,
  listAIInvestigations,
  type AIInvestigation,
  type AIInvestigationReport,
} from '@/services/ai';
import { extractApiErrorMessage } from '@/services/http';

function parseReport(row?: AIInvestigation | null): AIInvestigationReport | null {
  if (!row?.report_json) return null;
  try {
    return JSON.parse(row.report_json) as AIInvestigationReport;
  } catch {
    return null;
  }
}

function statusColor(s: string) {
  switch (s) {
    case 'done':
      return 'success';
    case 'failed':
      return 'error';
    case 'analyzing':
    case 'collecting':
      return 'processing';
    case 'awaiting_approval':
      return 'warning';
    default:
      return 'default';
  }
}

export default function AiInvestigationsPage() {
  const actionRef = useRef<ActionType>(undefined);
  const [searchParams] = useSearchParams();
  const focusId = Number(searchParams.get('id') || 0);
  const [selected, setSelected] = useState<AIInvestigation | null>(null);

  async function openDetail(id: number) {
    try {
      const row = await getAIInvestigation(id);
      setSelected(row);
    } catch (e) {
      message.error(extractApiErrorMessage(e, '加载调查详情失败'));
    }
  }

  useEffect(() => {
    if (focusId > 0) void openDetail(focusId);
  }, [focusId]);

  const report = parseReport(selected);

  const columns: ProColumns<AIInvestigation>[] = [
    {
      title: '类型',
      dataIndex: 'kind',
      hideInTable: true,
      valueType: 'select',
      valueEnum: {
        alert: { text: '告警' },
        pod: { text: 'Pod' },
        cicd: { text: 'CI/CD' },
        chat: { text: '对话' },
      },
      fieldProps: { allowClear: true, placeholder: '类型' },
    },
    { title: 'ID', dataIndex: 'id', width: 70, search: false },
    { title: '标题', dataIndex: 'title', ellipsis: true, search: false },
    {
      title: '类型',
      dataIndex: 'kind',
      width: 90,
      search: false,
      render: (_, row) => <Tag>{row.kind}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 120,
      search: false,
      render: (_, row) => <Tag color={statusColor(row.status)}>{row.status}</Tag>,
    },
    { title: '更新时间', dataIndex: 'updated_at', width: 180, search: false },
    {
      title: '操作',
      valueType: 'option',
      width: 100,
      render: (_, row) => [
        <Button
          key="detail"
          type="link"
          size="small"
          icon={<ExperimentOutlined />}
          onClick={() => void openDetail(row.id)}
        >
          详情
        </Button>,
      ],
    },
  ];

  return (
    <PageContainer
      header={{
        title: 'AI 调查',
        subTitle: '告警 / Pod / CI 等场景的采集→分析→报告闭环',
        extra: (
          <Space>
            <Link to="/ai/assistant">运维助手</Link>
            <Button icon={<ReloadOutlined />} onClick={() => actionRef.current?.reload()}>
              刷新
            </Button>
          </Space>
        ),
      }}
    >
      <ProTable<AIInvestigation>
        rowKey="id"
        actionRef={actionRef}
        columns={columns}
        search={{ labelWidth: 'auto' }}
        request={async (params) => {
          try {
            const res = await listAIInvestigations({
              kind: (params.kind as string) || undefined,
              page: params.current ?? 1,
              page_size: params.pageSize ?? 20,
            });
            return {
              data: res?.list || [],
              success: true,
              total: res?.total || 0,
            };
          } catch (e) {
            message.error(extractApiErrorMessage(e, '加载调查列表失败'));
            return { data: [], success: false, total: 0 };
          }
        }}
        pagination={{ defaultPageSize: 20 }}
      />

      {selected ? (
        <Card
          style={{ marginTop: 16 }}
          title={`调查 #${selected.id} · ${selected.title}`}
          extra={
            <Button type="link" onClick={() => setSelected(null)}>
              关闭
            </Button>
          }
        >
          <Descriptions size="small" column={2} style={{ marginBottom: 12 }}>
            <Descriptions.Item label="状态">
              <Tag color={statusColor(selected.status)}>{selected.status}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="类型">{selected.kind}</Descriptions.Item>
            <Descriptions.Item label="项目">{selected.project_id || '-'}</Descriptions.Item>
            <Descriptions.Item label="集群">{selected.cluster_id || '-'}</Descriptions.Item>
            <Descriptions.Item label="命名空间">{selected.namespace || '-'}</Descriptions.Item>
            <Descriptions.Item label="资源/指纹">
              {selected.resource || selected.fingerprint || '-'}
            </Descriptions.Item>
          </Descriptions>
          {selected.error_msg ? (
            <Alert type="error" showIcon message={selected.error_msg} style={{ marginBottom: 12 }} />
          ) : null}
          {report ? (
            <Space direction="vertical" style={{ width: '100%' }} size="middle">
              <Alert type="info" showIcon message="摘要" description={report.summary || '（无）'} />
              {report.root_causes?.length ? (
                <Card size="small" title="可能根因">
                  <pre style={{ margin: 0, whiteSpace: 'pre-wrap', fontSize: 12 }}>
                    {JSON.stringify(report.root_causes, null, 2)}
                  </pre>
                </Card>
              ) : null}
              {report.actions?.length ? (
                <Card size="small" title="建议动作">
                  <pre style={{ margin: 0, whiteSpace: 'pre-wrap', fontSize: 12 }}>
                    {JSON.stringify(report.actions, null, 2)}
                  </pre>
                </Card>
              ) : null}
              {report.evidence?.length ? (
                <Card size="small" title="证据">
                  <pre
                    style={{
                      margin: 0,
                      whiteSpace: 'pre-wrap',
                      fontSize: 12,
                      maxHeight: 280,
                      overflow: 'auto',
                    }}
                  >
                    {JSON.stringify(report.evidence, null, 2)}
                  </pre>
                </Card>
              ) : null}
              {report.raw_reply ? (
                <Card size="small" title="原始回复">
                  <Typography.Paragraph style={{ whiteSpace: 'pre-wrap', marginBottom: 0 }}>
                    {report.raw_reply}
                  </Typography.Paragraph>
                </Card>
              ) : null}
            </Space>
          ) : (
            <Typography.Text type="secondary">暂无结构化报告</Typography.Text>
          )}
        </Card>
      ) : null}
    </PageContainer>
  );
}
