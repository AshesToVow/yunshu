import {
  CloudDownloadOutlined,
  CloudUploadOutlined,
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
  RocketOutlined,
} from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Drawer,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Tooltip,
  Typography,
  message,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useMemo, useState } from "react";
import { PageTelemetryHeader } from "../components/page-telemetry-header";
import { useDictOptions } from "../hooks/use-dict-options";
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
} from "../services/cicd";
import {
  getProjectServerDetail,
  getProjectServers,
  getProjects,
  type ProjectItem,
  type ServerItem,
} from "../services/projects";
import { getClusters, type ClusterItem } from "../services/clusters";
import { getUsers } from "../services/users";
import type { UserItem } from "../types/api";
import { useAuth } from "../contexts/auth-context";
import { formatDateTime } from "../utils/format";

function serviceTypeLabel(v: string) {
  if (v === "frontend") return "前端服务";
  if (v === "backend") return "后端服务";
  return "容器化服务";
}

function buildResultColor(r?: string) {
  if (r === "success") return "success";
  if (r === "failure") return "error";
  if (r === "running") return "processing";
  return "default";
}

function ownerLabel(username: string | undefined, users: UserItem[]) {
  if (!username) return "—";
  const u = users.find((it) => it.username === username);
  if (!u) return username;
  return u.nickname ? `${u.nickname} (${u.username})` : u.username;
}

function ownerEmailPreview(username: string | undefined, users: UserItem[]) {
  if (!username) return "";
  const u = users.find((it) => it.username === username);
  return String(u?.email || "").trim();
}

function cicdAccess(row: CicdServiceItem) {
  return (
    row.access ?? {
      can_view: false,
      can_build: false,
      can_release: false,
      can_manage: false,
    }
  );
}

function isSuperAdminUser(u: UserItem | null | undefined): boolean {
  return Boolean(u?.roles?.some((r) => r.code === "super-admin"));
}

/** 与后端 projectacl.FullAccess 一致：超管或项目 owner/admin 可新建应用 */
function canCreateCicdService(isSuper: boolean, myProjectRole: string | null | undefined): boolean {
  if (isSuper) return true;
  const r = String(myProjectRole || "").toLowerCase();
  return r === "owner" || r === "admin";
}

const FRONTEND_RELEASE_OPS = [
  { value: "frontend_online", label: "服务上线" },
  { value: "frontend_rollback", label: "服务回滚" },
] as const;

const BACKEND_RELEASE_OPS = [
  { value: "backend_initial", label: "服务初次部署" },
  { value: "backend_update", label: "服务更新" },
] as const;

const CONTAINER_RELEASE_OPS = [
  { value: "service_online", label: "服务上线" },
  { value: "pod_update", label: "POD 更新" },
  { value: "container_rollback", label: "回滚" },
] as const;

const K8S_DEPLOY_CONFIG_TYPES = [
  { value: "使用deployment模板", label: "Deployment" },
  { value: "使用statefulset模板", label: "StatefulSet" },
  { value: "使用daemonset模板", label: "DaemonSet" },
] as const;

const K8S_DEPLOY_TEMPLATES = [
  { value: "基础模板", label: "基础模板" },
  { value: "通用微服务含skywalking", label: "通用微服务含 SkyWalking" },
] as const;

function releaseOpLabel(op: string) {
  return [...FRONTEND_RELEASE_OPS, ...BACKEND_RELEASE_OPS, ...CONTAINER_RELEASE_OPS].find((o) => o.value === op)?.label ?? op;
}

function parseServerIds(json?: string): number[] {
  if (!json) return [];
  try {
    const parsed = JSON.parse(json) as unknown;
    return Array.isArray(parsed) ? parsed.map((v) => Number(v)).filter((v) => Number.isFinite(v) && v > 0) : [];
  } catch {
    return [];
  }
}

function serverOptionLabel(s: Pick<ServerItem, "name" | "host">) {
  return `${s.name} (${s.host})`;
}

async function mergeServersWithSelected(
  projectId: number,
  list: ServerItem[],
  selectedIds: number[],
): Promise<{ servers: ServerItem[]; unresolvedIds: number[] }> {
  const byId = new Map<number, ServerItem>();
  for (const s of list) {
    byId.set(Number(s.id), s);
  }
  const missing = selectedIds.filter((id) => !byId.has(id));
  if (missing.length === 0) {
    return { servers: list, unresolvedIds: [] };
  }
  // 探测请求不弹全局 toast，由调用方汇总成「发布主机」场景提示，避免重复弹窗
  const details = await Promise.all(
    missing.map((id) =>
      getProjectServerDetail(projectId, id, { silentErrorToast: true }).catch(() => null),
    ),
  );
  for (const d of details) {
    if (d) {
      byId.set(Number(d.id), d);
    }
  }
  // 保持列表接口顺序，缺失的已选主机追加在末尾，便于回显名称
  const merged = [...list];
  const unresolvedIds: number[] = [];
  for (const id of missing) {
    const s = byId.get(id);
    if (s && !merged.some((it) => Number(it.id) === id)) {
      merged.push(s);
    } else if (!s) {
      unresolvedIds.push(id);
    }
  }
  return { servers: merged, unresolvedIds };
}

function nodesStatusTag(status?: string) {
  const s = status || "—";
  const color =
    s === "正常" || s === "启用"
      ? "success"
      : s === "部分异常"
        ? "warning"
        : s === "异常" || s === "已停用"
          ? "error"
          : "default";
  return <Tag color={color}>{s}</Tag>;
}

function defaultCiFormValues(svc: CicdServiceItem) {
  const isFront = svc.service_type === "frontend";
  return {
    ref_type: "branch",
    ref_name: "main",
    language_type: isFront ? "frontend" : "custom",
    build_type: isFront ? "npm" : "mvn",
    build_shell: isFront ? "run build" : "clean package -DskipTests",
    build_path: isFront ? "dist" : "target",
    npm_install_mode: "install",
    node_version: "node24",
    java_tool_name: "jdk8",
    project_name: svc.identifier,
    description: svc.name,
  };
}

export function CicdServicesPage() {
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
  const selectedLanguageType = Form.useWatch("language_type", ciForm) as string | undefined;
  const selectedTemplate = pipelineTemplates.find((t) => t.language_type === selectedLanguageType);

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

  const isFrontend = ciService?.service_type === "frontend" || deployService?.service_type === "frontend";

  return (
    <div className="page-stack">
      <PageTelemetryHeader
        label="[ CI/CD ]"
        title="应用服务"
        subtitle="CI 打包仅构建上传 MinIO；发布按操作类型部署制品（京雀式）"
        meta={[`PROJECT / ${projectId ?? "—"}`, `SERVICES / ${total}`]}
      />

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

      <Modal
        title={editingService ? "编辑应用" : "新建应用"}
        open={serviceModalOpen}
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
      >
        <Form form={serviceForm} layout="vertical">
          <Form.Item name="identifier" label="唯一标识符" rules={[{ required: true }]}>
            <Input placeholder="cityos-account" disabled={!!editingService} />
          </Form.Item>
          <Form.Item name="name" label="应用名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="service_type" label="应用类型" rules={[{ required: true }]}>
            <Select options={pipelineTypes.map((o) => ({ label: o.label, value: o.value }))} />
          </Form.Item>
          <Form.Item name="owner" label="Owner" extra="从平台用户列表选择，保存为用户名">
            <Select
              showSearch
              allowClear
              placeholder="选择负责人"
              optionFilterProp="label"
              options={userOptions.map((u) => ({
                value: u.username,
                label: `${u.nickname || u.username} (${u.username})`,
              }))}
            />
          </Form.Item>
          <Form.Item name="product_line" label="产品线">
            <Input />
          </Form.Item>
          <Form.Item name="jenkins_job" label="Jenkins Job 名" extra="留空则自动生成 cicd-p{projectId}-{identifier}">
            <Input />
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Input.TextArea rows={2} />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        title={`编辑 CI 配置 — ${ciService?.name ?? ""}`}
        width={560}
        open={ciDrawerOpen}
        onClose={() => setCiDrawerOpen(false)}
        extra={
          <Button type="primary" onClick={() => void saveCiConfig()}>
            确认
          </Button>
        }
      >
        <Alert
          type="warning"
          showIcon
          message="提示：填写业务仓库与构建参数后保存；Yunshu 将自动在 Jenkins 创建 Pipeline Job（Pipeline script from SCM），并引用数据字典中的 jenkinsfile-new。"
          style={{ marginBottom: 16 }}
        />
        <Form form={ciForm} layout="vertical">
          <Form.Item name="git_url" label="仓库地址" rules={[{ required: true }]}>
            <Input placeholder="git@gitee.com:org/repo.git" />
          </Form.Item>
          <Form.Item name="ref_type" label="分支或 TAG">
            <Select options={[{ label: "branch", value: "branch" }, { label: "tag", value: "tag" }]} />
          </Form.Item>
          <Form.Item
            name="ref_name"
            label="分支或 TAG 名称"
            rules={[{ required: true }]}
            extra="须与远端仓库实际分支一致（Gitee 常见默认分支为 master，勿填 main 除非仓库确有 main）"
          >
            <Input placeholder="master" />
          </Form.Item>
          <Form.Item
            name="language_type"
            label="流水线语言模板"
            rules={[{ required: true }]}
            extra={
              selectedTemplate?.script_path
                ? `将使用 Script Path：${selectedTemplate.script_path}`
                : selectedLanguageType === "custom"
                  ? "自定义：按服务类型选择 front/backend/k8s Jenkinsfile"
                  : selectedTemplate?.description
            }
          >
            <Select
              options={(pipelineTemplates.length
                ? pipelineTemplates
                : [
                    { language_type: "go", name: "Go" },
                    { language_type: "java", name: "Java" },
                    { language_type: "frontend", name: "前端" },
                    { language_type: "python", name: "Python" },
                    { language_type: "custom", name: "自定义" },
                  ]
              ).map((t) => ({
                label: t.name,
                value: t.language_type,
              }))}
            />
          </Form.Item>
          <Form.Item name="build_type" label="打包模板类型" rules={[{ required: true }]}>
            <Select
              options={(isFrontend ? frontBuildTypes : backBuildTypes).map((o) => ({
                label: o.label,
                value: o.value,
              }))}
            />
          </Form.Item>
          <Form.Item
            name="build_shell"
            label="打包参数"
            extra={isFrontend ? "如 run build:prod（不要写 npm/yarn 前缀）" : "如 clean package -DskipTests"}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="build_path"
            label={isFrontend ? "静态资源目录路径" : "服务包路径"}
            extra={isFrontend ? "前端构建输出目录，如 dist / build" : "JAR 搜索目录，默认 target"}
          >
            <Input />
          </Form.Item>
          {!isFrontend && (
            <>
              <Form.Item name="project_name" label="项目名(projectName)">
                <Input />
              </Form.Item>
              <Form.Item name="java_tool_name" label="JDK 工具名">
                <Input placeholder="jdk8" />
              </Form.Item>
              <Form.Item name="server_port" label="服务端口">
                <Input />
              </Form.Item>
            </>
          )}
          {isFrontend && (
            <>
              <Form.Item
                name="node_version"
                label="Node.js 工具"
                extra="与 Jenkins → Global Tool Configuration 中 Node 安装名称一致（如 node24、node20、node18）"
              >
                <Select
                  options={[
                    { label: "Node 24 (node24)", value: "node24" },
                    { label: "Node 20 LTS (node20)", value: "node20" },
                    { label: "Node 18 LTS (node18)", value: "node18" },
                  ]}
                />
              </Form.Item>
              <Form.Item name="npm_install_mode" label="依赖安装">
                <Select options={npmInstallModes.map((o) => ({ label: o.label, value: o.value }))} />
              </Form.Item>
              <Form.Item name="clean_npm_cache" label="清理缓存" valuePropName="checked">
                <Switch />
              </Form.Item>
              <Form.Item name="clean_node_modules" label="删除 node_modules" valuePropName="checked">
                <Switch />
              </Form.Item>
            </>
          )}
          <Form.Item name="version" label="版本号">
            <Input />
          </Form.Item>
          <Form.Item name="description" label="描述信息" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
        </Form>
      </Drawer>

      <Modal
        title={
          editingDeployConfig
            ? `编辑${deployKind === "regular" ? "常规" : "容器化"}发布配置`
            : deployKind === "regular"
              ? "新增常规发布配置"
              : "容器化发布配置"
        }
        open={deployWizardOpen}
        width={640}
        onCancel={() => {
          setDeployWizardOpen(false);
          setEditingDeployConfig(null);
        }}
        footer={
          <Space>
            <Button onClick={() => setDeployWizardOpen(false)}>取消</Button>
            {deployStep > 0 && <Button onClick={() => setDeployStep((s) => s - 1)}>上一步</Button>}
            {deployStep < (deployKind === "regular" ? 1 : 1) ? (
              <Button type="primary" onClick={() => void goDeployNextStep()}>
                下一步
              </Button>
            ) : (
              <Button type="primary" onClick={() => void submitDeployConfig()}>
                {editingDeployConfig ? "保存" : "确认"}
              </Button>
            )}
          </Space>
        }
      >
        <Form form={deployForm} layout="vertical">
        <div style={{ display: deployStep === 0 ? "block" : "none" }}>
            <Alert type="warning" showIcon message="新建发布配置将按已录入的应用生成新的发布配置信息！" style={{ marginBottom: 16 }} />
            <Form.Item name="name" label="配置名称" rules={[{ required: true, message: "请填写配置名称" }]}>
              <Input placeholder="如 k8s-demo-生产环境" />
            </Form.Item>
            <Form.Item name="tenv" label="发布环境" rules={[{ required: true }]} extra="同一应用下同类型发布配置每个环境仅允许一条">
              <Select
                options={tenvOpts.map((o) => ({
                  label: usedTenvSet.has(String(o.value)) ? `${o.label}（已配置）` : o.label,
                  value: o.value,
                  disabled: usedTenvSet.has(String(o.value)),
                }))}
              />
            </Form.Item>
            <Form.Item name="audit_enabled" label="发布审核" valuePropName="checked">
              <Switch checkedChildren="开启" unCheckedChildren="关闭" />
            </Form.Item>
            <Form.Item name="importance" label="重要级别">
              <Select allowClear options={importanceLevels.map((o) => ({ label: o.label, value: o.value }))} />
            </Form.Item>
            {deployKind === "regular" ? (
              <Form.Item name="server_port" label="服务端口">
                <InputNumber min={1} max={65535} style={{ width: "100%" }} />
              </Form.Item>
            ) : (
              <>
                <Form.Item name="deploy_action" label="默认操作类型" extra="发布时可覆盖">
                  <Select options={deployActions.map((o) => ({ label: o.label, value: o.value }))} />
                </Form.Item>
              </>
            )}
        </div>
        {deployKind === "container" && (
        <div style={{ display: deployStep === 1 ? "block" : "none" }}>
            <Alert type="info" showIcon message="配置 K8s 集群、命名空间与镜像信息" style={{ marginBottom: 16 }} />
            <Form.Item name="k8s_cluster_id" label="K8s 集群" rules={[{ required: true, message: "请选择集群" }]}>
              <Select
                showSearch
                optionFilterProp="label"
                options={clusters.map((c) => ({ label: c.name, value: c.id }))}
                placeholder="选择已接入的 K8s 集群"
              />
            </Form.Item>
            <Form.Item name="k8s_namespace" label="Namespace" rules={[{ required: true, message: "请填写命名空间" }]}>
              <Input placeholder="default" />
            </Form.Item>
            <Form.Item name="deploy_method" label="部署方式" rules={[{ required: true }]}>
              <Select options={[{ label: "kubectl", value: "kubectl" }, { label: "helm", value: "helm" }]} />
            </Form.Item>
            {deployMethodWatch === "helm" ? (
              <Alert
                type="success"
                showIcon
                style={{ marginBottom: 16 }}
                message="Helm 部署：已对齐「Application + base charts」目录架构"
                description={
                  <Space direction="vertical" size={8} style={{ width: "100%" }}>
                    <Typography.Text type="secondary">
                      下载 zip 解压到仓库根目录，得到 <Typography.Text code>helm/</Typography.Text>
                      （含 charts/deployment-base 等公共模块、config-files、多环境 values）与可选{" "}
                      <Typography.Text code>setup/</Typography.Text>
                      。研发只改 values.yaml；Jenkins 使用 <Typography.Text code>helm/Chart.yaml</Typography.Text>
                      。默认写入 Consul 注册：注解 <Typography.Text code>consul.register/enabled=true</Typography.Text>
                      、<Typography.Text code>consul.register/service.name</Typography.Text>
                      与标签 <Typography.Text code>yunshu-metrics: tag</Typography.Text>
                      （不需要时在 values 里关 <Typography.Text code>deployment-base.consulRegister.enabled</Typography.Text>）。
                    </Typography.Text>
                    <Button
                      type="primary"
                      icon={<CloudDownloadOutlined />}
                      loading={helmScaffoldLoading}
                      onClick={() => void handleDownloadHelmScaffold()}
                    >
                      下载 Helm 脚手架
                    </Button>
                  </Space>
                }
              />
            ) : null}
            {deployMethodWatch !== "helm" ? (
              <>
                <Form.Item name="deploy_config_type" label="工作负载类型" rules={[{ required: true }]}>
                  <Select options={K8S_DEPLOY_CONFIG_TYPES.map((o) => ({ label: o.label, value: o.value }))} />
                </Form.Item>
                <Form.Item name="deploy_config_template" label="部署模板" rules={[{ required: true }]} extra="共享库 k8s-basic / k8s-skywalking 的 Pod 模板须含 Consul 必填项：consul.register/enabled、service.name、标签 yunshu-metrics=tag">
                  <Select options={K8S_DEPLOY_TEMPLATES.map((o) => ({ label: o.label, value: o.value }))} />
                </Form.Item>
              </>
            ) : (
              <>
                <Form.Item name="deploy_config_type" hidden>
                  <Input />
                </Form.Item>
                <Form.Item name="deploy_config_template" hidden>
                  <Input />
                </Form.Item>
              </>
            )}
            <Form.Item name="replicas" label="副本数" rules={[{ required: true }]}>
              <InputNumber min={1} max={100} style={{ width: "100%" }} />
            </Form.Item>
            <Form.Item
              name="deploy_strategy"
              label="发布策略"
              rules={[{ required: true }]}
              extra="金丝雀/蓝绿：Jenkins 接收 deployStrategy 等参数；平台侧可在发布详情中晋级/中止"
            >
              <Select
                options={[
                  { label: "滚动发布", value: "rolling" },
                  { label: "金丝雀发布", value: "canary" },
                  { label: "蓝绿发布", value: "blue_green" },
                ]}
              />
            </Form.Item>
            <Form.Item noStyle shouldUpdate={(prev, cur) => prev.deploy_strategy !== cur.deploy_strategy}>
              {({ getFieldValue }) => {
                const strategy = getFieldValue("deploy_strategy");
                if (strategy === "canary") {
                  return (
                    <>
                      <Form.Item name="canary_replicas" label="金丝雀初始副本" rules={[{ required: true }]}>
                        <InputNumber min={1} max={100} style={{ width: "100%" }} />
                      </Form.Item>
                      <Form.Item name="canary_percent" label="金丝雀流量占比(%)" rules={[{ required: true }]}>
                        <InputNumber min={1} max={100} style={{ width: "100%" }} />
                      </Form.Item>
                      <Form.Item
                        name="canary_steps_json"
                        label="晋级步骤(%)"
                        rules={[{ required: true }]}
                        extra="逗号分隔，如 10,50,100"
                      >
                        <Input placeholder="10,50,100" />
                      </Form.Item>
                    </>
                  );
                }
                if (strategy === "blue_green") {
                  return (
                    <Form.Item
                      name="blue_green_service"
                      label="蓝绿 Service 名"
                      extra="留空则使用工作负载名；切换 selector 标签 yunshu.io/color"
                    >
                      <Input placeholder="可选，默认与工作负载同名" />
                    </Form.Item>
                  );
                }
                return null;
              }}
            </Form.Item>
            <Form.Item name="container_port" label="容器端口" rules={[{ required: true }]}>
              <InputNumber min={1} max={65535} style={{ width: "100%" }} />
            </Form.Item>
            <Form.Item name="image_name" label="镜像名" rules={[{ required: true }]} extra="Harbor 仓库中的镜像名称">
              <Input />
            </Form.Item>
            <Form.Item name="image_tag" label="默认镜像 Tag">
              <Input placeholder="latest（CI 构建时会追加时间戳）" />
            </Form.Item>
        </div>
        )}
        {deployKind === "regular" && (
        <div style={{ display: deployStep === 1 ? "block" : "none" }}>
            <Alert type="warning" showIcon message="需要选择部署的目标主机及部署路径" style={{ marginBottom: 16 }} />
            <Form.Item name="dest_path" label="部署路径" rules={[{ required: true }]}>
              <Input placeholder="/export/icity/app-name" />
            </Form.Item>
            <Form.Item name="server_ids" label="发布主机" rules={[{ required: true, message: "请选择发布主机" }]}>
              <Select
                mode="multiple"
                allowClear
                showSearch
                optionFilterProp="label"
                options={serverOptions}
                placeholder="请选择发布主机"
                tagRender={(props) => {
                  const { value, closable, onClose } = props;
                  const text = serverLabelById.get(Number(value)) ?? serverOptions.find((o) => o.value === Number(value))?.label;
                  return (
                    <Tag closable={closable} onClose={onClose} style={{ marginInlineEnd: 4 }}>
                      {text || String(value)}
                    </Tag>
                  );
                }}
              />
            </Form.Item>
            <Form.Item name="artifact_retain_count" label="历史版本数量">
              <InputNumber min={1} max={100} style={{ width: "100%" }} />
            </Form.Item>
            <Form.Item name="deploy_user" label="部署用户">
              <Input />
            </Form.Item>
            <Form.Item name="deploy_group" label="部署属组" hidden>
              <Input />
            </Form.Item>
            {!isFrontend && (
              <>
                <Form.Item name="run_user" label="运行用户" extra="后端 JAR/进程的运行用户">
                  <Input />
                </Form.Item>
                <Form.Item name="start_script_type" label="启动脚本类型" extra="后端部署时在目标机生成 bin/launch.sh">
                  <Select options={startScriptTypes.map((o) => ({ label: o.label, value: o.value }))} />
                </Form.Item>
                <Form.Item noStyle shouldUpdate={(p, c) => p.start_script_type !== c.start_script_type}>
                  {({ getFieldValue }) =>
                    getFieldValue("start_script_type") === "自定义脚本" ? (
                      <Form.Item
                        name="custom_script_content"
                        label="自定义 launch.sh 内容"
                        rules={[{ required: true, message: "请填写自定义启动脚本" }]}
                        extra="整段 bash 脚本，将写入目标机 destPath/bin/launch.sh"
                      >
                        <Input.TextArea rows={6} placeholder="#!/bin/bash&#10;..." />
                      </Form.Item>
                    ) : null
                  }
                </Form.Item>
              </>
            )}
            {isFrontend && (
              <Alert
                type="info"
                showIcon
                message="前端静态资源部署：制品解压到部署路径即可，由 Nginx 等 Web 服务器提供访问，无需 launch.sh 启动脚本。"
                style={{ marginBottom: 8 }}
              />
            )}
            <Form.Item name="clean_deploy_dir" label="部署前清空目录" valuePropName="checked">
              <Switch />
            </Form.Item>
        </div>
        )}
        </Form>
      </Modal>

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

      <Modal title={`发布 — ${releaseService?.name}`} open={releaseModalOpen} onCancel={() => setReleaseModalOpen(false)} onOk={() => void doTriggerRelease()}>
        <Form form={releaseForm} layout="vertical">
          {releaseDeployConfig?.audit_enabled ? (
            <Alert
              type="warning"
              showIcon
              message="该环境已开启发布审核"
              description="提交后将进入「待办列表 → 待审核」，审批通过后再进入「待执行」由审核人/运维执行 Jenkins 发布。"
              style={{ marginBottom: 16 }}
            />
          ) : null}
          <Form.Item name="deploy_kind" hidden>
            <Input />
          </Form.Item>
          <Form.Item name="title" label="任务名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="deploy_config_id" label="发布配置" rules={[{ required: true }]}>
            <Select
              options={(expandedDeploys[releaseService?.id ?? 0] ?? []).map((c) => ({
                label: `${c.name} (${c.tenv})`,
                value: c.id,
              }))}
            />
          </Form.Item>
          <Form.Item noStyle shouldUpdate={(prev, cur) => prev.deploy_kind !== cur.deploy_kind}>
            {({ getFieldValue, setFieldsValue }) => {
              const kind = getFieldValue("deploy_kind");
              const svcType = releaseService?.service_type;
              if (kind === "container") {
                const op = getFieldValue("release_operation");
                const isRollback = op === "container_rollback";
                return (
                  <>
                    <Form.Item
                      name="release_operation"
                      label="操作类型"
                      rules={[{ required: true, message: "请选择操作类型" }]}
                    >
                      <Select
                        options={CONTAINER_RELEASE_OPS.map((o) => ({ label: o.label, value: o.value }))}
                        onChange={(v) => {
                          const name = releaseService?.name ?? "应用";
                          setFieldsValue({
                            title: `${name}-${releaseOpLabel(String(v))}`,
                            publish_mode: v === "container_rollback" ? "回滚" : "制品发布",
                          });
                        }}
                      />
                    </Form.Item>
                    {!isRollback ? (
                      <>
                        <Form.Item name="publish_mode" hidden initialValue="制品发布">
                          <Input />
                        </Form.Item>
                        <Form.Item
                          name="build_run_id"
                          label="CI 构建镜像"
                          rules={[{ required: true, message: "请选择已成功构建的镜像" }]}
                          extra="须先执行 CI 打包并成功推送 Harbor"
                        >
                          <Select
                            showSearch
                            loading={releaseBuildRunsLoading}
                            placeholder={releaseBuildRunsLoading ? "加载构建记录…" : releaseBuildRuns.length ? "选择镜像" : "暂无可用镜像，请先 CI 打包"}
                            options={releaseBuildRuns.map((r) => ({
                              value: r.id,
                              label: `#${r.build_number} ${r.image_address}${r.branch_name ? ` · ${r.branch_name}` : ""}`,
                            }))}
                            onChange={(id) => {
                              const run = releaseBuildRuns.find((r) => r.id === id);
                              setFieldsValue({ image_address: run?.image_address });
                            }}
                          />
                        </Form.Item>
                        <Form.Item name="image_address" hidden>
                          <Input />
                        </Form.Item>
                      </>
                    ) : (
                      <Alert type="warning" showIcon message="将回滚到 K8s 上一版本（Helm/kubectl）" />
                    )}
                  </>
                );
              }
              const opOptions = svcType === "frontend" ? FRONTEND_RELEASE_OPS : BACKEND_RELEASE_OPS;
              const op = getFieldValue("release_operation");
              const opExtra =
                op === "frontend_rollback"
                  ? "选择 MinIO 中的历史制品包进行回滚"
                  : op === "backend_update"
                    ? "部署所选制品；选最新包为上线，选历史包即为回滚"
                    : op === "backend_initial"
                      ? "首次部署将清空目标目录后解压制品"
                      : "部署所选 MinIO 制品到目标服务器";
              return (
                <>
                  <Form.Item
                    name="release_operation"
                    label="操作类型"
                    rules={[{ required: true, message: "请选择操作类型" }]}
                  >
                    <Select
                      options={opOptions.map((o) => ({ label: o.label, value: o.value }))}
                      onChange={(v) => {
                        const name = releaseService?.name ?? "应用";
                        setFieldsValue({ title: `${name}-${releaseOpLabel(String(v))}` });
                      }}
                    />
                  </Form.Item>
                  <Form.Item
                    name="artifact_name"
                    label="MinIO 制品包"
                    rules={[{ required: true, message: "请选择要部署的制品" }]}
                    extra={opExtra}
                  >
                    <Select
                      showSearch
                      loading={releaseArtifactsLoading}
                      placeholder={releaseArtifactsLoading ? "加载制品列表…" : releaseArtifacts.length ? "选择制品" : "暂无制品，请先 CI 打包"}
                      options={releaseArtifacts.map((a) => ({
                        value: a.name,
                        label: `${a.name}${a.last_modified ? ` · ${a.last_modified}` : ""}`,
                      }))}
                      filterOption={(input, option) => String(option?.label ?? "").toLowerCase().includes(input.toLowerCase())}
                    />
                  </Form.Item>
                </>
              );
            }}
          </Form.Item>
          <Alert
            type="info"
            showIcon
            message="构建通知邮件"
            description={
              ownerEmailPreview(releaseService?.owner, userOptions)
                ? `Jenkins 构建/部署结果将发送至 Owner 邮箱：${ownerEmailPreview(releaseService?.owner, userOptions)}（可在用户管理维护 email；须配置 Jenkins 邮件扩展与 SMTP）`
                : "未找到 Owner 邮箱：请在用户管理为应用 Owner 填写 email，并在数据字典配置 mail_* 与 Jenkins 邮件插件"
            }
          />
        </Form>
      </Modal>
    </div>
  );
}
