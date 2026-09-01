// @ts-nocheck
import {
  CloudUploadOutlined,
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
  RocketOutlined,
} from '@ant-design/icons';
import { PageContainer } from '@ant-design/pro-components';
import {
  Alert,
  Button,
  Card,
  Form,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { LegacyShell } from '@/components/LegacyShell';
import { useAuth } from '@/contexts/auth-context';
import { useDictOptions } from '@/hooks/use-dict-options';
import {
  createCicdService,
  createDeployConfig,
  deleteCicdService,
  deleteDeployConfig,
  downloadHelmScaffold,
  downloadHelmScaffoldPreview,
  getCiConfig,
  listCicdServices,
  listCicdArtifacts,
  listBuildRuns,
  listDeployConfigs,
  triggerBuild,
  triggerRelease,
  updateCicdService,
  updateDeployConfig,
  upsertCiConfig,
  listPipelineTemplates,
  type CicdArtifactItem,
  type CicdBuildRun,
  type CicdDeployConfig,
  type CicdPipelineTemplate,
  type CicdServiceItem,
} from '@/services/cicd';
import { getClusters, type ClusterItem } from '@/services/clusters';
import {
  getProjectServers,
  getProjects,
  type ProjectItem,
  type ServerItem,
} from '@/services/projects';
import { getUsers } from '@/services/users';
import type { UserItem } from '@/types/api';
import { formatDateTime } from '@/utils/format';
import { canCreateCicdService, cicdAccess, isSuperAdminUser } from '../access';
import { CiConfigDrawer } from '../ci-config-drawer';
import { DeployConfigModal } from '../deploy-config-modal';
import {
  buildResultColor,
  nodesStatusTag,
  ownerEmailPreview,
  ownerLabel,
  serviceTypeLabel,
} from '../display';
import { defaultCiFormValues } from '../form-defaults';
import { ReleaseDrawer } from '../release-drawer';
import { releaseOpLabel } from '../release-ops';
import { ServiceFormDrawer } from '../service-form-drawer';
import { mergeServersWithSelected, parseServerIds, serverOptionLabel } from '../servers';

export default function CicdServicesPage() {
  return (
    <LegacyShell>
      <CicdServicesInner />
    </LegacyShell>
  );
}

function CicdServicesInner() {
  const { user: currentUser } = useAuth();
  const isSuper = useMemo(() => isSuperAdminUser(currentUser), [currentUser]);
  const pipelineTypes = useDictOptions("cicd_pipeline_type");
  const publishModes = useDictOptions("cicd_publish_mode");
  const tenvOpts = useDictOptions("cicd_tenv");
  const frontBuildTypes = useDictOptions("cicd_build_type_frontend");
  const backBuildTypes = useDictOptions("cicd_build_type_backend");
  const npmInstallModes = useDictOptions("cicd_npm_install_mode");
  const deployActions = useDictOptions("cicd_deploy_action");
  const startScriptTypes = useDictOptions("cicd_start_script_type");
  const importanceLevels = useDictOptions("cicd_importance_level");
  const [pipelineTemplates, setPipelineTemplates] = useState<CicdPipelineTemplate[]>([]);

  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [userOptions, setUserOptions] = useState<UserItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [loading, setLoading] = useState(false);
  const [services, setServices] = useState<CicdServiceItem[]>([]);
  const [total, setTotal] = useState(0);
  const [keyword, setKeyword] = useState("");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  const selectedProject = useMemo(
    () => projects.find((p) => p.id === projectId),
    [projects, projectId],
  );
  const canCreateService = useMemo(
    () => Boolean(projectId && canCreateCicdService(isSuper, selectedProject?.my_project_role)),
    [projectId, isSuper, selectedProject?.my_project_role],
  );

  const [serviceModalOpen, setServiceModalOpen] = useState(false);
  const [editingService, setEditingService] = useState<CicdServiceItem | null>(null);
  const [serviceForm] = Form.useForm();

  const [ciDrawerOpen, setCiDrawerOpen] = useState(false);
  const [ciService, setCiService] = useState<CicdServiceItem | null>(null);
  const [ciForm] = Form.useForm();

  const [deployWizardOpen, setDeployWizardOpen] = useState(false);
  const [deployStep, setDeployStep] = useState(0);
  const [deployService, setDeployService] = useState<CicdServiceItem | null>(null);
  const [editingDeployConfig, setEditingDeployConfig] = useState<CicdDeployConfig | null>(null);
  const [deployKind, setDeployKind] = useState<"regular" | "container">("regular");
  const [deployForm] = Form.useForm();
  const deployMethodWatch = Form.useWatch("deploy_method", deployForm) as string | undefined;
  const [helmScaffoldLoading, setHelmScaffoldLoading] = useState(false);
  const [servers, setServers] = useState<ServerItem[]>([]);
  const [clusters, setClusters] = useState<ClusterItem[]>([]);
  const serverOptions = useMemo(
    () => servers.map((s) => ({ label: serverOptionLabel(s), value: Number(s.id) })),
    [servers],
  );
  const serverLabelById = useMemo(() => {
    const m = new Map<number, string>();
    for (const s of servers) {
      m.set(Number(s.id), serverOptionLabel(s));
    }
    return m;
  }, [servers]);

  const [expandedDeploys, setExpandedDeploys] = useState<Record<number, CicdDeployConfig[]>>({});

  const [buildModalOpen, setBuildModalOpen] = useState(false);
  const [buildService, setBuildService] = useState<CicdServiceItem | null>(null);
  const [buildPreview, setBuildPreview] = useState<{ branch: string; tenv: string } | null>(null);

  const [releaseModalOpen, setReleaseModalOpen] = useState(false);
  const [releaseService, setReleaseService] = useState<CicdServiceItem | null>(null);
  const [releaseArtifacts, setReleaseArtifacts] = useState<CicdArtifactItem[]>([]);
  const [releaseArtifactsLoading, setReleaseArtifactsLoading] = useState(false);
  const [releaseBuildRuns, setReleaseBuildRuns] = useState<CicdBuildRun[]>([]);
  const [releaseBuildRunsLoading, setReleaseBuildRunsLoading] = useState(false);
  const [releaseDeployConfig, setReleaseDeployConfig] = useState<CicdDeployConfig | null>(null);
  const [releaseForm] = Form.useForm();

  const loadProjects = useCallback(async () => {
    const res = await getProjects({ page: 1, page_size: 200 });
    setProjects(res.list ?? []);
    if (!projectId && res.list?.length) {
      setProjectId(res.list[0].id);
    }
  }, [projectId]);

  const loadUsers = useCallback(async () => {
    try {
      const res = await getUsers({ page: 1, page_size: 500 });
      setUserOptions((res.list ?? []).filter((u) => u.status === 1));
    } catch {
      setUserOptions([]);
    }
  }, []);

  const loadServices = useCallback(async () => {
    if (!projectId) return;
    setLoading(true);
    try {
      const res = await listCicdServices(projectId, { page, page_size: pageSize, keyword });
      setServices(res.list ?? []);
      setTotal(res.total ?? 0);
    } finally {
      setLoading(false);
    }
  }, [projectId, page, pageSize, keyword]);

  const handleDownloadHelmScaffold = useCallback(async () => {
    if (!projectId) {
      message.warning("请先选择项目");
      return;
    }
    const values = deployForm.getFieldsValue([
      "image_name",
      "replicas",
      "container_port",
    ]) as { image_name?: string; replicas?: number; container_port?: number };
    const chartName = deployService?.identifier || deployService?.name || "app";
    const imageName = String(values.image_name || chartName).trim();
    const params = {
      chart_name: chartName,
      image_repository: imageName.includes("/") ? imageName : undefined,
      replica_count: Number(values.replicas) || 1,
      container_port: Number(values.container_port) || 8080,
      service_port: Number(values.container_port) || 8080,
    };
    setHelmScaffoldLoading(true);
    try {
      let blob: Blob;
      if (deployService?.id) {
        blob = await downloadHelmScaffold(projectId, deployService.id, params);
      } else {
        blob = await downloadHelmScaffoldPreview(projectId, {
          chart_name: params.chart_name,
          image_repository: params.image_repository,
          replica_count: params.replica_count,
          container_port: params.container_port,
          service_port: params.service_port,
        });
      }
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `${chartName}-helm-scaffold.zip`;
      a.click();
      URL.revokeObjectURL(url);
      message.success("已下载：解压后提交 helm/（含 charts/*-base、config-files）与可选 setup/");
    } catch {
      // http 拦截器已 toast
    } finally {
      setHelmScaffoldLoading(false);
    }
  }, [projectId, deployService, deployForm]);

  useEffect(() => {
    void loadProjects();
    void loadUsers();
    void listPipelineTemplates().then((rows) => setPipelineTemplates(rows || [])).catch(() => setPipelineTemplates([]));
  }, [loadProjects, loadUsers]);

  useEffect(() => {
    void loadServices();
  }, [loadServices]);

  const loadDeployConfigs = useCallback(
    async (serviceId: number) => {
      if (!projectId) return [];
      const rows = await listDeployConfigs(projectId, serviceId);
      setExpandedDeploys((prev) => ({ ...prev, [serviceId]: rows }));
      return rows;
    },
    [projectId],
  );

  const openBuildModal = async (row: CicdServiceItem) => {
    if (!projectId) return;
    setBuildService(row);
    setBuildPreview(null);
    setBuildModalOpen(true);
    try {
      const [ciView, deploys] = await Promise.all([
        getCiConfig(projectId, row.id),
        loadDeployConfigs(row.id),
      ]);
      setBuildPreview({
        branch: ciView.config?.ref_name?.trim() || "main",
        tenv: deploys.find((c) => c.tenv)?.tenv || "dev",
      });
    } catch {
      setBuildPreview({ branch: "main", tenv: "dev" });
    }
  };

  const openCiConfig = async (svc: CicdServiceItem) => {
    if (!projectId) return;
    setCiService(svc);
    setCiDrawerOpen(true);
    ciForm.resetFields();
    const view = await getCiConfig(projectId, svc.id);
    if (view.configured && view.config) {
      ciForm.setFieldsValue(view.config);
    } else {
      ciForm.setFieldsValue(defaultCiFormValues(svc));
    }
  };

  const saveCiConfig = async () => {
    if (!projectId || !ciService) return;
    const values = await ciForm.validateFields();
    const result = await upsertCiConfig(projectId, ciService.id, values);
    if (result.jenkins_sync_error) {
      message.warning(`CI 配置已保存，但 Jenkins Job 同步失败：${result.jenkins_sync_error}`);
    } else if (result.jenkins_sync?.created) {
      message.success(`CI 配置已保存，已在 Jenkins 创建 Job：${result.jenkins_sync.job_name}`);
    } else if (result.jenkins_sync?.updated) {
      message.success(`CI 配置已保存，已更新 Jenkins Job：${result.jenkins_sync.job_name}`);
    } else {
      message.success("CI 配置已保存");
    }
    setCiDrawerOpen(false);
    void loadServices();
  };

  const openDeployWizard = async (
    svc: CicdServiceItem,
    kind: "regular" | "container",
    editCfg?: CicdDeployConfig,
  ) => {
    if (!projectId) return;
    setDeployService(svc);
    setDeployKind(kind);
    setEditingDeployConfig(editCfg ?? null);
    setDeployStep(0);
    deployForm.resetFields();
    const [srvRes, clusterRes] = await Promise.all([
      getProjectServers(projectId, { page: 1, page_size: 500 }),
      getClusters({ page: 1, page_size: 200 }),
      loadDeployConfigs(svc.id),
    ]);
    const selectedServerIds = editCfg ? parseServerIds(editCfg.server_ids_json) : [];
    const { servers: mergedServers, unresolvedIds } = await mergeServersWithSelected(
      projectId,
      srvRes.list ?? [],
      selectedServerIds,
    );
    setServers(mergedServers);
    if (unresolvedIds.length > 0) {
      message.warning(
        `发布配置关联的主机不存在或已移除（ID: ${unresolvedIds.join(", ")}），请重新选择发布主机`,
      );
    }
    setClusters(clusterRes.list ?? []);
    const ciView = await getCiConfig(projectId, svc.id);
    const base = {
      name: `${svc.name}-${kind === "regular" ? "常规" : "容器"}`,
      deploy_kind: kind,
      tenv: "dev",
      dest_path: svc.service_type === "frontend" ? `/export/frontend/${svc.identifier}` : `/export/icity/${svc.identifier}`,
      artifact_retain_count: 10,
      deploy_user: svc.service_type === "frontend" ? "nginx" : "root",
      deploy_group: svc.service_type === "frontend" ? "nginx" : "root",
      run_user: "app",
      start_script_type: "脚本模板",
      deploy_action: "服务更新",
      deploy_method: "kubectl",
      deploy_config_type: "使用deployment模板",
      deploy_config_template: "基础模板",
      deploy_strategy: "rolling",
      canary_replicas: 1,
      canary_percent: 10,
      canary_steps_json: "10,50,100",
      blue_green_service: "",
      server_port: 8080,
      replicas: 1,
      container_port: 8080,
      image_name: svc.identifier,
      image_tag: "latest",
    };
    if (editCfg) {
      deployForm.setFieldsValue({
        ...editCfg,
        server_ids: selectedServerIds,
      });
    } else {
      deployForm.setFieldsValue({
        ...base,
        ...(ciView.configured && ciView.config ? { git_url: ciView.config.git_url } : {}),
      });
    }
    setDeployWizardOpen(true);
  };

  const usedTenvSet = useMemo(() => {
    if (!deployService) return new Set<string>();
    const configs = expandedDeploys[deployService.id] ?? [];
    return new Set(
      configs
        .filter((c) => c.deploy_kind === deployKind && c.id !== editingDeployConfig?.id)
        .map((c) => String(c.tenv)),
    );
  }, [deployService, deployKind, editingDeployConfig, expandedDeploys]);

  const goDeployNextStep = async () => {
    const step0Fields =
      deployKind === "regular"
        ? (["name", "tenv", "audit_enabled", "importance", "server_port"] as const)
        : (["name", "tenv", "audit_enabled", "importance", "deploy_action"] as const);
    await deployForm.validateFields([...step0Fields]);
    setDeployStep((s) => s + 1);
  };

  const submitDeployConfig = async () => {
    if (!projectId || !deployService) return;
    const allFields =
      deployKind === "regular"
        ? ([
            "name",
            "tenv",
            "audit_enabled",
            "importance",
            "server_port",
            "dest_path",
            "server_ids",
            "artifact_retain_count",
            "deploy_user",
            "deploy_group",
            "clean_deploy_dir",
          ] as const)
        : ([
            "name",
            "tenv",
            "audit_enabled",
            "importance",
            "deploy_action",
            "k8s_namespace",
            "k8s_cluster_id",
            "deploy_method",
            "deploy_config_type",
            "deploy_config_template",
            "deploy_strategy",
            "replicas",
            "container_port",
            "image_name",
            "image_tag",
          ] as const);
    await deployForm.validateFields([...allFields]);
    const values = deployForm.getFieldsValue();
    const payload = { ...values, deploy_kind: deployKind };
    if (editingDeployConfig) {
      await updateDeployConfig(projectId, deployService.id, editingDeployConfig.id, payload);
      message.success("发布配置已更新");
    } else {
      await createDeployConfig(projectId, deployService.id, payload);
      message.success("发布配置已创建");
    }
    setDeployWizardOpen(false);
    setEditingDeployConfig(null);
    void loadDeployConfigs(deployService.id);
    void loadServices();
  };

  const doTriggerBuild = async () => {
    if (!projectId || !buildService) return;
    await triggerBuild(projectId, buildService.id, {});
    message.success("已触发 CI 打包");
    setBuildModalOpen(false);
    void loadServices();
  };

  const openReleaseModal = async (svc: CicdServiceItem, cfg: CicdDeployConfig) => {
    if (!projectId) return;
    const isContainer = cfg.deploy_kind === "container";
    const defaultOp = isContainer ? "pod_update" : svc.service_type === "frontend" ? "frontend_online" : "backend_update";
    setReleaseService(svc);
    setReleaseDeployConfig(cfg);
    setReleaseArtifacts([]);
    setReleaseBuildRuns([]);
    releaseForm.setFieldsValue({
      deploy_config_id: cfg.id,
      deploy_kind: cfg.deploy_kind,
      title: `${svc.name}-${releaseOpLabel(defaultOp)}`,
      release_operation: defaultOp,
      artifact_name: undefined,
      build_run_id: undefined,
      image_address: undefined,
      publish_mode: "制品发布",
    });
    setReleaseModalOpen(true);
    if (isContainer) {
      setReleaseBuildRunsLoading(true);
      try {
        const res = await listBuildRuns(projectId, { service_id: svc.id, page: 1, page_size: 50 });
        const runs = (res.list ?? []).filter(
          (r) =>
            r.build_result === "success" &&
            r.image_address &&
            !r.image_address.includes("/inbound-agent") &&
            !r.image_address.includes("/jenkins/inbound-agent"),
        );
        setReleaseBuildRuns(runs);
        if (runs.length === 1) {
          releaseForm.setFieldsValue({ build_run_id: runs[0].id, image_address: runs[0].image_address });
        }
      } catch {
        setReleaseBuildRuns([]);
      } finally {
        setReleaseBuildRunsLoading(false);
      }
    } else {
      setReleaseArtifactsLoading(true);
      try {
        const list = await listCicdArtifacts(projectId, svc.id);
        setReleaseArtifacts(list ?? []);
        if ((list ?? []).length === 1) {
          releaseForm.setFieldValue("artifact_name", list[0].name);
        } else if ((list ?? []).length === 0) {
          message.warning("暂无 MinIO 制品，请先完成 CI 打包并确认已上传到制品桶");
        }
      } catch {
        setReleaseArtifacts([]);
        // 全局 toast 已展示接口错误（如 MinIO 未配置/连不上）；此处仅清空选项
      } finally {
        setReleaseArtifactsLoading(false);
      }
    }
  };

  const doTriggerRelease = async () => {
    if (!projectId || !releaseService) return;
    const values = await releaseForm.validateFields();
    const run = await triggerRelease(projectId, releaseService.id, values);
    if (run.status === "pending_approval") {
      message.success("已提交发布申请，请至「待办列表」等待审核");
    } else {
      message.success("已提交发布工单");
    }
    setReleaseModalOpen(false);
  };

  const columns = useMemo<ColumnsType<CicdServiceItem>>(
    () => [
      { title: "应用名称", dataIndex: "name", width: 160 },
      { title: "唯一标识符", dataIndex: "identifier", width: 140 },
      { title: "产品线", dataIndex: "product_line", width: 120, render: (v) => v || "—" },
      {
        title: "应用类型",
        dataIndex: "service_type",
        width: 110,
        render: (v) => <Tag>{serviceTypeLabel(String(v))}</Tag>,
      },
      { title: "Owner", dataIndex: "owner", width: 120, render: (v) => ownerLabel(v, userOptions) },
      {
        title: "最近构建",
        width: 120,
        render: (_, row) =>
          row.last_build_result ? (
            <Tag color={buildResultColor(row.last_build_result)}>{row.last_build_result}</Tag>
          ) : (
            "—"
          ),
      },
      {
        title: "操作",
        key: "actions",
        fixed: "right",
        width: 340,
        render: (_, row) => {
          const access = cicdAccess(row);
          return (
          <Space size={4} wrap>
            <Button
              type="link"
              size="small"
              disabled={!access.can_manage}
              onClick={() => openCiConfig(row)}
            >
              {row.has_ci_config ? "编辑CI配置" : "新增CI配置"}
            </Button>
            <Button
              type="link"
              size="small"
              icon={<CloudUploadOutlined />}
              disabled={!access.can_build}
              onClick={() => void openBuildModal(row)}
            >
              CI打包
            </Button>
            <Button
              type="link"
              size="small"
              disabled={!access.can_manage}
              onClick={() => openDeployWizard(row, "regular")}
            >
              非容器化发布
            </Button>
            <Button
              type="link"
              size="small"
              disabled={!access.can_manage}
              onClick={() => openDeployWizard(row, "container")}
            >
              容器化发布
            </Button>
            <Popconfirm
              title="确认删除该应用？"
              disabled={!access.can_manage}
              onConfirm={() => void deleteCicdService(projectId!, row.id).then(loadServices)}
            >
              <Button type="link" size="small" danger icon={<DeleteOutlined />} disabled={!access.can_manage} />
            </Popconfirm>
          </Space>
          );
        },
      },
    ],
    [projectId, userOptions],
  );

  const expandedRowRender = (row: CicdServiceItem) => {
    const configs = expandedDeploys[row.id];
    if (!configs) {
      void loadDeployConfigs(row.id);
      return <Typography.Text type="secondary">加载发布配置…</Typography.Text>;
    }
    if (!configs.length) {
      return <Typography.Text type="secondary">暂无发布配置，请点击「非容器化发布」或「容器化发布」新建。</Typography.Text>;
    }
    return (
      <Table
        size="small"
        rowKey="id"
        pagination={false}
        dataSource={configs}
        columns={[
          { title: "配置名称", dataIndex: "name" },
          { title: "环境", dataIndex: "tenv", render: (v) => <Tag>{v}</Tag> },
          {
            title: "类型",
            dataIndex: "deploy_kind",
            render: (v) => (v === "container" ? "容器化" : "常规"),
          },
          { title: "部署路径/Namespace", render: (_, c) => c.dest_path || c.k8s_namespace || "—" },
          {
            title: "节点数",
            width: 80,
            render: (_, c) =>
              c.deploy_kind === "container" ? `${c.server_count ?? c.replicas ?? 1} 副本` : `${c.server_count ?? 0} 台`,
          },
          {
            title: "状态",
            width: 100,
            render: (_, c) => nodesStatusTag(c.nodes_status ?? (c.status === 1 ? "启用" : "已停用")),
          },
          {
            title: "审核",
            dataIndex: "audit_enabled",
            width: 72,
            render: (v) => (v ? "开启" : "关闭"),
          },
          {
            title: "操作",
            width: 200,
            render: (_, c) => {
              const access = cicdAccess(row);
              return (
              <Space size={4}>
                <Button
                  type="link"
                  size="small"
                  icon={<RocketOutlined />}
                  disabled={!access.can_release}
                  onClick={() => void openReleaseModal(row, c)}
                >
                  发布
                </Button>
                <Button
                  type="link"
                  size="small"
                  icon={<EditOutlined />}
                  disabled={!access.can_manage}
                  onClick={() => void openDeployWizard(row, c.deploy_kind === "container" ? "container" : "regular", c)}
                >
                  编辑
                </Button>
                <Popconfirm
                  title="删除该发布配置？"
                  disabled={!access.can_manage}
                  onConfirm={() =>
                    void deleteDeployConfig(projectId!, row.id, c.id).then(() => loadDeployConfigs(row.id))
                  }
                >
                  <Button type="link" size="small" danger disabled={!access.can_manage}>
                    删除
                  </Button>
                </Popconfirm>
              </Space>
              );
            },
          },
        ]}
      />
    );
  };

  return (
    <PageContainer
      header={{
        title: '应用服务',
        subTitle: 'CI 打包仅构建上传 MinIO；发布按操作类型部署制品（京雀式）',
      }}
    >
      <Alert
        type="info"
        showIcon
        message="使用前请在「数据字典」配置 cicd_jenkins_base_url、cicd_jenkins_api_token；容器化应用另需配置 cicd_harbor_url、cicd_harbor_host_ip、cicd_harbor_credential_id、cicd_harbor_project_group（Harbor 凭据须在 Jenkins 凭据库中预先创建）。保存 CI 配置后 Yunshu 会自动在 Jenkins 创建/更新 Pipeline Job。"
        style={{ marginBottom: 12 }}
      />

      <Card bordered={false}>
        <Space wrap style={{ marginBottom: 16 }}>
          <Select
            style={{ width: 220 }}
            placeholder="选择项目"
            value={projectId}
            options={projects.map((p) => ({ label: p.name, value: p.id }))}
            onChange={(v) => {
              setProjectId(v);
              setPage(1);
            }}
          />
          <Input.Search
            allowClear
            placeholder="搜索应用名称/标识符"
            style={{ width: 240 }}
            onSearch={(v) => {
              setKeyword(v);
              setPage(1);
            }}
          />
          <Button icon={<ReloadOutlined />} onClick={() => void loadServices()}>
            刷新
          </Button>
          <Tooltip title={canCreateService ? undefined : "仅项目 owner/admin 或超级管理员可新建应用"}>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              disabled={!canCreateService}
              onClick={() => {
                if (!canCreateService) {
                  message.error("当前账号无新建应用权限");
                  return;
                }
                setEditingService(null);
                serviceForm.resetFields();
                serviceForm.setFieldsValue({
                  service_type: "backend",
                  status: 1,
                  owner: currentUser?.username ?? undefined,
                });
                setServiceModalOpen(true);
              }}
            >
              新建应用
            </Button>
          </Tooltip>
        </Space>

        <div className="k8s-table-scroll-host">
        <Table<CicdServiceItem>
          rowKey="id"
          loading={loading}
          columns={columns}
          dataSource={services}
          scroll={{ x: 1200 }}
          expandable={{ expandedRowRender, onExpand: (_, r) => void loadDeployConfigs(r.id) }}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            onChange: (p, ps) => {
              setPage(p);
              setPageSize(ps);
            },
          }}
        />
        </div>
      </Card>

      <ServiceFormDrawer
        open={serviceModalOpen}
        editingService={editingService}
        form={serviceForm}
        pipelineTypes={pipelineTypes}
        userOptions={userOptions}
        onCancel={() => setServiceModalOpen(false)}
        onOk={async () => {
          if (!projectId) return;
          const values = await serviceForm.validateFields();
          if (editingService) {
            if (!cicdAccess(editingService).can_manage) {
              message.error("当前账号无编辑该应用权限");
              return;
            }
            await updateCicdService(projectId, editingService.id, values);
          } else {
            if (!canCreateService) {
              message.error("当前账号无新建应用权限");
              return;
            }
            await createCicdService(projectId, values);
          }
          message.success("已保存");
          setServiceModalOpen(false);
          void loadServices();
        }}
      />

      <CiConfigDrawer
        open={ciDrawerOpen}
        ciService={ciService}
        form={ciForm}
        pipelineTemplates={pipelineTemplates}
        frontBuildTypes={frontBuildTypes}
        backBuildTypes={backBuildTypes}
        npmInstallModes={npmInstallModes}
        onClose={() => setCiDrawerOpen(false)}
        onSave={saveCiConfig}
      />

      <DeployConfigModal
        open={deployWizardOpen}
        deployKind={deployKind}
        deployStep={deployStep}
        editingDeployConfig={editingDeployConfig}
        deployService={deployService}
        form={deployForm}
        tenvOpts={tenvOpts}
        usedTenvSet={usedTenvSet}
        importanceLevels={importanceLevels}
        deployActions={deployActions}
        deployMethodWatch={deployMethodWatch}
        clusters={clusters}
        helmScaffoldLoading={helmScaffoldLoading}
        serverOptions={serverOptions}
        serverLabelById={serverLabelById}
        startScriptTypes={startScriptTypes}
        onCancel={() => {
          setDeployWizardOpen(false);
          setEditingDeployConfig(null);
        }}
        onPrevStep={() => setDeployStep((s) => s - 1)}
        onNextStep={goDeployNextStep}
        onSubmit={submitDeployConfig}
        onDownloadHelmScaffold={handleDownloadHelmScaffold}
      />

      <Modal title={`CI 打包 — ${buildService?.name}`} open={buildModalOpen} onCancel={() => setBuildModalOpen(false)} onOk={() => void doTriggerBuild()}>
        <Alert
          type="info"
          showIcon
          message={
            expandedDeploys[buildService?.id ?? 0]?.some((c) => c.deploy_kind === "container")
              ? "容器化应用：拉代码 → 编译 → 构建镜像并推送 Harbor（不部署到集群）"
              : "CI 打包仅执行：拉代码 → 编译 → 上传 MinIO，不会部署到服务器"
          }
          description={
            buildPreview
              ? `分支：${buildPreview.branch}（来自 CI 配置） · 环境：${buildPreview.tenv}（来自发布配置）`
              : "加载配置中…"
          }
          style={{ marginBottom: 16 }}
        />
        <Alert
          type="info"
          showIcon
          message="构建通知"
          description={
            ownerEmailPreview(buildService?.owner, userOptions)
              ? `打包结果邮件将发送至：${ownerEmailPreview(buildService?.owner, userOptions)}`
              : "请为应用 Owner 在用户管理中配置 email，并确保 Jenkins 已配置 SMTP"
          }
        />
      </Modal>

      <ReleaseDrawer
        open={releaseModalOpen}
        releaseService={releaseService}
        releaseDeployConfig={releaseDeployConfig}
        releaseArtifacts={releaseArtifacts}
        releaseArtifactsLoading={releaseArtifactsLoading}
        releaseBuildRuns={releaseBuildRuns}
        releaseBuildRunsLoading={releaseBuildRunsLoading}
        expandedDeploys={expandedDeploys}
        form={releaseForm}
        userOptions={userOptions}
        onCancel={() => setReleaseModalOpen(false)}
        onOk={doTriggerRelease}
      />
    </PageContainer>
  );
}
