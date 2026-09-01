import { DeleteOutlined, ReloadOutlined } from '@ant-design/icons';
import type { ProColumns } from '@ant-design/pro-components';
import { PageContainer, ProTable } from '@ant-design/pro-components';
import { Link } from '@umijs/max';
import { Button, Popconfirm, Select, Space, Tag, Typography, message } from 'antd';
import { useCallback, useEffect, useState } from 'react';
import {
  deleteHarborArtifact,
  listHarborArtifacts,
  listHarborProjects,
  listHarborRepositories,
  listRegistries,
  type HarborRepoItem,
  type HarborTagItem,
  type ImageRegistryItem,
} from '@/services/cicd';
import { formatDateTime } from '@/utils/format';

export default function CicdImageBrowserPage() {
  const [registries, setRegistries] = useState<ImageRegistryItem[]>([]);
  const [registryId, setRegistryId] = useState<number>();
  const [projects, setProjects] = useState<{ name: string }[]>([]);
  const [harborProject, setHarborProject] = useState<string>();
  const [repos, setRepos] = useState<HarborRepoItem[]>([]);
  const [repository, setRepository] = useState<string>();
  const [artifacts, setArtifacts] = useState<HarborTagItem[]>([]);
  const [loading, setLoading] = useState(false);

  const selected = registries.find((r) => r.id === registryId);
  const isDocker = selected?.type === 'docker_registry';

  useEffect(() => {
    void listRegistries({ page: 1, page_size: 100 }).then((d) => {
      const list = d.list || [];
      setRegistries(list);
      const def = list.find((r) => r.is_default) || list[0];
      if (def) setRegistryId(def.id);
    });
  }, []);

  const loadProjects = useCallback(async () => {
    if (!registryId) return;
    const rows = await listHarborProjects({ registry_id: registryId });
    setProjects(rows || []);
    if (isDocker) {
      setHarborProject('_catalog');
    } else if (rows?.[0] && !harborProject) {
      setHarborProject(rows[0].name);
    }
  }, [registryId, isDocker, harborProject]);

  useEffect(() => {
    setHarborProject(undefined);
    setRepository(undefined);
    setRepos([]);
    setArtifacts([]);
    void loadProjects();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [registryId]);

  const loadRepos = useCallback(async () => {
    if (!registryId || !harborProject) return;
    setLoading(true);
    try {
      const rows = await listHarborRepositories({
        registry_id: registryId,
        harbor_project: harborProject,
      });
      setRepos(rows || []);
    } finally {
      setLoading(false);
    }
  }, [registryId, harborProject]);

  useEffect(() => {
    setRepository(undefined);
    setArtifacts([]);
    if (harborProject) void loadRepos();
  }, [harborProject, loadRepos]);

  const loadArtifacts = useCallback(async () => {
    if (!registryId || !repository) return;
    setLoading(true);
    try {
      const rows = await listHarborArtifacts({
        registry_id: registryId,
        harbor_project: isDocker ? undefined : harborProject,
        repository,
      });
      setArtifacts(rows || []);
    } finally {
      setLoading(false);
    }
  }, [registryId, repository, harborProject, isDocker]);

  useEffect(() => {
    if (repository) void loadArtifacts();
    else setArtifacts([]);
  }, [repository, loadArtifacts]);

  const columns: ProColumns<HarborTagItem>[] = [
    {
      title: 'Tags',
      dataIndex: 'tags',
      search: false,
      render: (_, row) =>
        row.tags?.length ? (
          <Space wrap>
            {row.tags.map((t) => (
              <Tag key={t}>{t}</Tag>
            ))}
          </Space>
        ) : (
          '—'
        ),
    },
    { title: 'Digest', dataIndex: 'digest', ellipsis: true, width: 280, search: false },
    {
      title: '大小',
      dataIndex: 'size',
      width: 100,
      search: false,
      render: (_, row) => (row.size ? `${(row.size / 1024 / 1024).toFixed(1)} MB` : '—'),
    },
    {
      title: '推送时间',
      dataIndex: 'push_time',
      width: 168,
      search: false,
      render: (_, row) => formatDateTime(row.push_time),
    },
    {
      title: '关联构建',
      dataIndex: 'linked_build_runs',
      width: 180,
      search: false,
      render: (_, row) =>
        row.linked_build_runs?.length ? (
          <Space wrap>
            {row.linked_build_runs.map((l) => (
              <Link key={l.id} to={`/cicd/build-records?project_id=${l.project_id}&run_id=${l.id}`}>
                #{l.build_number || l.id}
              </Link>
            ))}
          </Space>
        ) : (
          '—'
        ),
    },
    {
      title: '操作',
      valueType: 'option',
      width: 100,
      render: (_, row) => {
        const ref = row.tags?.[0] || row.digest;
        if (!ref || !repository) return [];
        return [
          <Popconfirm
            key="del"
            title={`确认删除 ${ref}？`}
            onConfirm={async () => {
              await deleteHarborArtifact({
                registry_id: registryId,
                harbor_project: isDocker ? undefined : harborProject,
                repository,
                reference: ref,
              });
              message.success('已删除');
              void loadArtifacts();
            }}
          >
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>,
        ];
      },
    },
  ];

  return (
    <PageContainer
      header={{
        title: '镜像浏览',
        subTitle: '按注册中心浏览 Project / Repository / Tag，并与构建记录互链',
      }}
    >
      <Space wrap style={{ marginBottom: 16 }}>
        <Select
          style={{ width: 220 }}
          placeholder="注册中心"
          value={registryId}
          options={registries.map((r) => ({ label: `${r.name} (${r.type})`, value: r.id }))}
          onChange={(v) => setRegistryId(v)}
        />
        {!isDocker && (
          <Select
            style={{ width: 200 }}
            placeholder="Harbor Project"
            value={harborProject}
            options={projects.map((p) => ({ label: p.name, value: p.name }))}
            onChange={setHarborProject}
            showSearch
          />
        )}
        <Select
          style={{ width: 280 }}
          placeholder="Repository"
          value={repository}
          options={repos.map((r) => ({
            label: `${r.name}${r.artifact_count != null ? ` (${r.artifact_count})` : ''}`,
            value: r.name,
          }))}
          onChange={setRepository}
          showSearch
        />
        <Button
          icon={<ReloadOutlined />}
          onClick={() => void (repository ? loadArtifacts() : loadRepos())}
        >
          刷新
        </Button>
      </Space>

      {!registryId ? (
        <Typography.Text type="secondary">请先在「镜像仓库注册中心」创建仓库</Typography.Text>
      ) : (
        <ProTable<HarborTagItem>
          rowKey={(r) => r.digest || r.tags?.join(',') || String(Math.random())}
          columns={columns}
          dataSource={artifacts}
          loading={loading}
          search={false}
          options={false}
          pagination={{ pageSize: 20 }}
          toolBarRender={false}
        />
      )}
    </PageContainer>
  );
}
