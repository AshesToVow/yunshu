import { DeleteOutlined, EyeOutlined, FileTextOutlined, LinkOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { PageContainer, ProTable } from '@ant-design/pro-components';
import { useModel, useSearchParams } from '@umijs/max';
import { Button, Modal, Popconfirm, Space, Tag, Typography, message } from 'antd';
import { useEffect, useRef, useState } from 'react';
import { CicdReleaseDetailPanel } from '@/components/cicd-release-detail-panel';
import { LegacyShell } from '@/components/LegacyShell';
import {
  cicdReleaseStatusLabel,
  cicdReleaseStatusTagColor,
  cicdReleaseTypeLabel,
  cicdReleaseTypeTagColor,
} from '@/components/cicd-release-utils';
import { useDictOptions } from '@/hooks/use-dict-options';
import {
  deleteReleaseRun,
  executeReleaseRun,
  getReleaseRunLog,
  listReleaseRuns,
  type CicdReleaseRun,
} from '@/services/cicd';
import { getProjects, type ProjectItem } from '@/services/projects';
import { formatDateTime } from '@/utils/format';

export default function CicdReleaseRecordsPage() {
  return (
    <LegacyShell>
      <CicdReleaseRecordsInner />
    </LegacyShell>
  );
}

function CicdReleaseRecordsInner() {
  const actionRef = useRef<ActionType>(undefined);
  const logPreRef = useRef<HTMLPreElement>(null);
  const [searchParams] = useSearchParams();
  const { initialState } = useModel('@@initialState');
  const currentUserId = Number(
    (initialState?.currentUser as { id?: number; userid?: string } | undefined)?.id ??
      initialState?.currentUser?.userid,
  );

  const tenvOpts = useDictOptions('cicd_tenv');
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number | undefined>(() => {
    const n = Number(searchParams.get('project') || 0);
    return n > 0 ? n : undefined;
  });
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailRun, setDetailRun] = useState<CicdReleaseRun | null>(null);
  const [executeLoading, setExecuteLoading] = useState(false);
  const [logOpen, setLogOpen] = useState(false);
  const [logRun, setLogRun] = useState<CicdReleaseRun | null>(null);
  const [logText, setLogText] = useState('');
  const [logLoading, setLogLoading] = useState(false);

  useEffect(() => {
    void getProjects({ page: 1, page_size: 200 }).then((res) => {
      const list = res.list ?? [];
      setProjects(list);
      if (!projectId && list.length) setProjectId(list[0].id);
    });
  }, []);

  useEffect(() => {
    const releaseId = Number(searchParams.get('release') || 0);
    if (releaseId > 0 && projectId) {
      void listReleaseRuns(projectId, { page: 1, page_size: 50 }).then((res) => {
        const hit = res.list?.find((r) => r.id === releaseId);
        if (hit) {
          setDetailRun(hit);
          setDetailOpen(true);
        }
      });
    }
  }, [projectId, searchParams]);

  async function fetchLog(run: CicdReleaseRun, scroll = true) {
    if (!projectId) return;
    setLogLoading(true);
    try {
      const r = await getReleaseRunLog(projectId, run.id);
      setLogText(r.log || '（暂无日志）');
      if (scroll && logPreRef.current) {
        requestAnimationFrame(() => {
          logPreRef.current!.scrollTop = logPreRef.current!.scrollHeight;
        });
      }
    } finally {
      setLogLoading(false);
    }
  }

  useEffect(() => {
    if (!logOpen || !logRun || !projectId) return;
    const running = logRun.status === 'running' || logRun.status === 'pending';
    void fetchLog(logRun);
    if (!running) return;
    const timer = window.setInterval(() => {
      void fetchLog(logRun, true);
      void listReleaseRuns(projectId, { page: 1, page_size: 50 }).then((res) => {
        const updated = res.list?.find((r) => r.id === logRun.id);
        if (updated) setLogRun(updated);
      });
    }, 3000);
    return () => window.clearInterval(timer);
  }, [logOpen, logRun?.id, logRun?.status, projectId]);

  const columns: ProColumns<CicdReleaseRun>[] = [
    {
      title: '项目',
      dataIndex: 'project_id',
      hideInTable: true,
      valueType: 'select',
      initialValue: projectId,
      fieldProps: {
        options: projects.map((p) => ({ label: p.name, value: p.id })),
        showSearch: true,
        allowClear: false,
        onChange: (v: number) => setProjectId(v),
      },
    },
    { title: '工单名称', dataIndex: 'title', ellipsis: true, width: 200 },
    {
      title: '关键词',
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: { placeholder: '搜索工单名称' },
    },
    { title: '应用', dataIndex: 'service_name', width: 140, ellipsis: true, search: false },
    { title: '提交人', dataIndex: 'submitter_name', width: 96, search: false },
    {
      title: '类型',
      dataIndex: 'release_type',
      width: 120,
      search: false,
      render: (_, row) => (
        <Tag color={cicdReleaseTypeTagColor(row.release_type ?? '')}>
          {cicdReleaseTypeLabel(row.release_type ?? '') || '—'}
        </Tag>
      ),
    },
    {
      title: '环境',
      dataIndex: 'tenv',
      width: 100,
      valueType: 'select',
      fieldProps: { options: tenvOpts.map((o) => ({ label: o.label, value: o.value })), allowClear: true },
    },
    {
      title: '提交时间',
      dataIndex: 'started_at',
      width: 168,
      search: false,
      render: (_, row) => formatDateTime(row.started_at),
    },
    {
      title: '完成时间',
      dataIndex: 'finished_at',
      width: 168,
      search: false,
      render: (_, row) => formatDateTime(row.finished_at),
    },
    {
      title: '工单状态',
      dataIndex: 'status',
      width: 110,
      valueType: 'select',
      valueEnum: {
        success: { text: '执行成功', status: 'Success' },
        running: { text: '执行中', status: 'Processing' },
        failure: { text: '执行失败', status: 'Error' },
        pending_approval: { text: '待审核', status: 'Warning' },
        pending_execution: { text: '待执行', status: 'Warning' },
        rejected: { text: '已驳回', status: 'Default' },
        aborted: { text: '已中止', status: 'Default' },
      },
      render: (_, row) => (
        <Tag color={cicdReleaseStatusTagColor(row.status)}>{cicdReleaseStatusLabel(row.status)}</Tag>
      ),
    },
    {
      title: '操作',
      valueType: 'option',
      width: 200,
      fixed: 'right',
      render: (_, row) => [
        <Button
          key="detail"
          type="link"
          size="small"
          icon={<EyeOutlined />}
          onClick={() => {
            setDetailRun(row);
            setDetailOpen(true);
          }}
        >
          详情
        </Button>,
        <Button
          key="log"
          type="link"
          size="small"
          icon={<FileTextOutlined />}
          onClick={() => {
            setLogRun(row);
            setLogOpen(true);
          }}
        >
          发布日志
        </Button>,
        <Popconfirm
          key="del"
          title="确认删除该工单记录？"
          onConfirm={async () => {
            if (!projectId) return;
            await deleteReleaseRun(projectId, row.id);
            message.success('已删除');
            actionRef.current?.reload();
          }}
        >
          <Button type="link" size="small" danger icon={<DeleteOutlined />} />
        </Popconfirm>,
      ],
    },
  ];

  return (
    <PageContainer title="CD 历史工单" subTitle="常规发布与容器化发布执行记录">
      <ProTable<CicdReleaseRun>
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        scroll={{ x: 1200 }}
        params={{ projectId }}
        pagination={{ defaultPageSize: 10, showSizeChanger: true }}
        search={{ labelWidth: 'auto' }}
        request={async (params) => {
          const pid = Number(params.project_id || params.projectId || projectId);
          if (!pid) return { data: [], success: true, total: 0 };
          if (pid !== projectId) setProjectId(pid);
          const res = await listReleaseRuns(pid, {
            page: params.current,
            page_size: params.pageSize,
            status: params.status,
            keyword: params.keyword,
            tenv: params.tenv,
          });
          return { data: res.list ?? [], total: res.total ?? 0, success: true };
        }}
      />

      <Modal
        title={
          detailRun ? (
            <Space wrap>
              <span>工单详情</span>
              <Tag color={cicdReleaseStatusTagColor(detailRun.status)}>
                {cicdReleaseStatusLabel(detailRun.status)}
              </Tag>
              <Tag color={cicdReleaseTypeTagColor(detailRun.release_type ?? '')}>
                {cicdReleaseTypeLabel(detailRun.release_type ?? '') || '—'}
              </Tag>
            </Space>
          ) : (
            '工单详情'
          )
        }
        open={detailOpen}
        onCancel={() => {
          setDetailOpen(false);
          setDetailRun(null);
        }}
        width={920}
        style={{ top: 24 }}
        destroyOnClose
        footer={
          detailRun ? (
            <Space>
              {detailRun.jenkins_build_url ? (
                <Button icon={<LinkOutlined />} href={detailRun.jenkins_build_url} target="_blank" rel="noreferrer">
                  打开 Jenkins
                </Button>
              ) : null}
              {detailRun.status === 'pending_execution' &&
              currentUserId &&
              detailRun.submitter_user_id === currentUserId ? (
                <Button
                  type="primary"
                  loading={executeLoading}
                  onClick={() => {
                    if (!projectId) return;
                    setExecuteLoading(true);
                    void executeReleaseRun(projectId, detailRun.id)
                      .then(() => {
                        message.success('已触发发布执行');
                        setDetailOpen(false);
                        setDetailRun(null);
                        actionRef.current?.reload();
                      })
                      .finally(() => setExecuteLoading(false));
                  }}
                >
                  执行发布
                </Button>
              ) : null}
              <Button onClick={() => setDetailOpen(false)}>关闭</Button>
            </Space>
          ) : null
        }
      >
        {detailRun && projectId ? (
          <CicdReleaseDetailPanel
            projectId={projectId}
            runId={detailRun.id}
            onExecuted={() => {
              setDetailOpen(false);
              setDetailRun(null);
              actionRef.current?.reload();
            }}
          />
        ) : null}
      </Modal>

      <Modal
        title={
          <Space>
            <span>发布日志</span>
            {logRun?.title ? <Typography.Text type="secondary">— {logRun.title}</Typography.Text> : null}
            {logRun && (logRun.status === 'running' || logRun.status === 'pending') ? (
              <Tag color="processing">实时刷新中</Tag>
            ) : null}
          </Space>
        }
        open={logOpen}
        onCancel={() => {
          setLogOpen(false);
          setLogRun(null);
        }}
        width="92vw"
        style={{ top: 16, maxWidth: 1400 }}
        styles={{ body: { padding: '12px 16px' } }}
        destroyOnClose
        footer={
          <Space>
            <Button onClick={() => logRun && void fetchLog(logRun, false)} loading={logLoading}>
              刷新
            </Button>
            <Button type="primary" onClick={() => setLogOpen(false)}>
              关闭
            </Button>
          </Space>
        }
      >
        <pre
          ref={logPreRef}
          style={{
            height: 'calc(100vh - 220px)',
            minHeight: 480,
            overflow: 'auto',
            fontSize: 12,
            lineHeight: 1.5,
            margin: 0,
            padding: 12,
            background: '#1e1e1e',
            color: '#d4d4d4',
            borderRadius: 6,
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-word',
          }}
        >
          {logText}
        </pre>
      </Modal>
    </PageContainer>
  );
}
