import YAML from "yaml";
import {
  EnvPair,
  KVPair,
  ProbeType,
  envPairsToMap,
  kvPairsToMap,
  mapToEnvPairs,
  mapToKvPairs,
  parseExecCommandJson,
  parseIntOrStringPort,
  probeFromForm,
  probeToForm,
  safeGet,
  safeParseYaml,
  toNumberOrUndefined
} from "./helpers";

export type DeploymentFormValues = {
  name: string;
  namespace: string;
  replicas: number;
  container_name: string;
  image: string;
  image_pull_policy?: "Always" | "IfNotPresent" | "Never";
  port?: number;
  port_name?: string;
  command?: string;
  env_pairs?: EnvPair[];
  requests_cpu?: string;
  requests_memory?: string;
  limits_cpu?: string;
  limits_memory?: string;
  node_selector_pairs?: KVPair[];
  affinity_yaml?: string;
  strategy_type?: "RollingUpdate" | "Recreate";
  rolling_update_max_surge?: string;
  rolling_update_max_unavailable?: string;
  min_ready_seconds?: number;
  progress_deadline_seconds?: number;
  revision_history_limit?: number;
  tolerations?: Array<{
    key?: string;
    operator?: "Equal" | "Exists";
    value?: string;
    effect?: "NoSchedule" | "PreferNoSchedule" | "NoExecute";
    toleration_seconds?: number;
  }>;
  volumes?: Array<{
    name?: string;
    type?: "emptyDir" | "configMap" | "secret" | "pvc";
    source_name?: string;
  }>;
  volume_mounts?: Array<{
    name?: string;
    mount_path?: string;
    read_only?: boolean;
    sub_path?: string;
  }>;
  image_pull_secrets?: string[];
  liveness_probe_type?: ProbeType;
  liveness_http_path?: string;
  liveness_http_port?: string;
  liveness_http_scheme?: "HTTP" | "HTTPS";
  liveness_tcp_port?: string;
  liveness_exec_command?: string;
  liveness_initial_delay_seconds?: number;
  liveness_period_seconds?: number;
  liveness_timeout_seconds?: number;
  liveness_failure_threshold?: number;
  liveness_success_threshold?: number;
  readiness_probe_type?: ProbeType;
  readiness_http_path?: string;
  readiness_http_port?: string;
  readiness_http_scheme?: "HTTP" | "HTTPS";
  readiness_tcp_port?: string;
  readiness_exec_command?: string;
  readiness_initial_delay_seconds?: number;
  readiness_period_seconds?: number;
  readiness_timeout_seconds?: number;
  readiness_failure_threshold?: number;
  readiness_success_threshold?: number;
  startup_probe_type?: ProbeType;
  startup_http_path?: string;
  startup_http_port?: string;
  startup_http_scheme?: "HTTP" | "HTTPS";
  startup_tcp_port?: string;
  startup_exec_command?: string;
  startup_initial_delay_seconds?: number;
  startup_period_seconds?: number;
  startup_timeout_seconds?: number;
  startup_failure_threshold?: number;
  startup_success_threshold?: number;
};

export function buildDeploymentYaml(v: DeploymentFormValues): string {
  const envMap = envPairsToMap(v.env_pairs);
  const env = Object.keys(envMap).length
    ? Object.entries(envMap).map(([name, value]) => ({ name, value }))
    : undefined;

  const imagePullSecrets =
    (v.image_pull_secrets ?? []).filter(Boolean).map((name) => ({ name: String(name).trim() })).filter((s) => !!s.name);
  const tolerations =
    (v.tolerations ?? [])
      .map((t) => ({
        key: String(t.key ?? "").trim(),
        operator: t.operator || "Equal",
        value: String(t.value ?? "").trim(),
        effect: t.effect || undefined,
        tolerationSeconds: typeof t.toleration_seconds === "number" ? t.toleration_seconds : undefined,
      }))
      .filter((t) => t.key || t.operator === "Exists")
      .map((t) => ({
        key: t.key || undefined,
        operator: t.operator,
        value: t.operator === "Exists" ? undefined : t.value || undefined,
        effect: t.effect,
        tolerationSeconds: t.tolerationSeconds,
      })) || undefined;

  const volumes =
    (v.volumes ?? [])
      .map((it) => {
        const name = String(it.name ?? "").trim();
        if (!name) return null;
        const type = it.type || "emptyDir";
        if (type === "configMap") {
          return { name, configMap: { name: String(it.source_name ?? "").trim() || name } };
        }
        if (type === "secret") {
          return { name, secret: { secretName: String(it.source_name ?? "").trim() || name } };
        }
        if (type === "pvc") {
          return { name, persistentVolumeClaim: { claimName: String(it.source_name ?? "").trim() || name } };
        }
        return { name, emptyDir: {} };
      })
      .filter(Boolean);

  const volumeMounts =
    (v.volume_mounts ?? [])
      .map((m) => ({
        name: String(m.name ?? "").trim(),
        mountPath: String(m.mount_path ?? "").trim(),
        readOnly: !!m.read_only,
        subPath: String(m.sub_path ?? "").trim() || undefined,
      }))
      .filter((m) => m.name && m.mountPath) || undefined;

  const resources: any = {};
  if (v.requests_cpu || v.requests_memory) {
    resources.requests = {};
    if (v.requests_cpu) resources.requests.cpu = v.requests_cpu;
    if (v.requests_memory) resources.requests.memory = v.requests_memory;
  }
  if (v.limits_cpu || v.limits_memory) {
    resources.limits = {};
    if (v.limits_cpu) resources.limits.cpu = v.limits_cpu;
    if (v.limits_memory) resources.limits.memory = v.limits_memory;
  }

  const livenessProbe = probeFromForm("liveness", v);
  const readinessProbe = probeFromForm("readiness", v);
  const startupProbe = probeFromForm("startup", v);
  const nodeSelector = kvPairsToMap(v.node_selector_pairs);
  const affinity = safeParseYaml(String(v.affinity_yaml ?? "").trim() || "");

  const obj: any = {
    apiVersion: "apps/v1",
    kind: "Deployment",
    metadata: { name: v.name, namespace: v.namespace },
    spec: {
      replicas: v.replicas,
      strategy: v.strategy_type
        ? {
            type: v.strategy_type,
            ...(v.strategy_type === "RollingUpdate"
              ? {
                  rollingUpdate: {
                    maxSurge: String(v.rolling_update_max_surge ?? "").trim() || undefined,
                    maxUnavailable: String(v.rolling_update_max_unavailable ?? "").trim() || undefined,
                  },
                }
              : {}),
          }
        : undefined,
      minReadySeconds: typeof v.min_ready_seconds === "number" ? v.min_ready_seconds : undefined,
      progressDeadlineSeconds: typeof v.progress_deadline_seconds === "number" ? v.progress_deadline_seconds : undefined,
      revisionHistoryLimit: typeof v.revision_history_limit === "number" ? v.revision_history_limit : undefined,
      selector: { matchLabels: { app: v.name } },
      template: {
        metadata: { labels: { app: v.name } },
        spec: {
          containers: [
            {
              name: v.container_name || v.name,
              image: v.image,
              imagePullPolicy: v.image_pull_policy || undefined,
              ports: v.port ? [{ name: String(v.port_name ?? "").trim() || undefined, containerPort: v.port }] : undefined,
              command: v.command?.trim() ? ["sh", "-c", v.command.trim()] : undefined,
              env,
              resources: Object.keys(resources).length ? resources : undefined,
              volumeMounts,
                  ...(livenessProbe ? { livenessProbe } : {}),
                  ...(readinessProbe ? { readinessProbe } : {}),
                  ...(startupProbe ? { startupProbe } : {}),
            },
          ],
              imagePullSecrets: imagePullSecrets.length ? imagePullSecrets : undefined,
          volumes: volumes.length ? volumes : undefined,
          tolerations: tolerations?.length ? tolerations : undefined,
          nodeSelector: Object.keys(nodeSelector).length ? nodeSelector : undefined,
          affinity: affinity || undefined,
        },
      },
    },
  };
  return YAML.stringify(obj);
}

export function deploymentYamlToForm(yaml: string): DeploymentFormValues | null {
  const obj: any = safeParseYaml(yaml);
  if (!obj || obj.kind !== "Deployment") return null;
  const c = obj?.spec?.template?.spec?.containers?.[0] ?? {};
  const envPairs = mapToEnvPairs(
    Array.isArray(c.env)
      ? Object.fromEntries(c.env.filter((e: any) => e?.name).map((e: any) => [String(e.name), String(e.value ?? "")]))
      : undefined,
  );
  const port = c?.ports?.[0]?.containerPort;
  let cmd = "";
  if (Array.isArray(c.command) && c.command.length >= 3 && c.command[0] === "sh" && c.command[1] === "-c") {
    cmd = String(c.command.slice(2).join(" "));
  }
  const resReq = c?.resources?.requests ?? {};
  const resLim = c?.resources?.limits ?? {};
  const tolerations =
    Array.isArray(obj?.spec?.template?.spec?.tolerations) && obj.spec.template.spec.tolerations.length
      ? obj.spec.template.spec.tolerations.map((t: any) => ({
          key: t?.key,
          operator: (t?.operator || "Equal") as "Equal" | "Exists",
          value: t?.value,
          effect: t?.effect as "NoSchedule" | "PreferNoSchedule" | "NoExecute" | undefined,
          toleration_seconds: typeof t?.tolerationSeconds === "number" ? t.tolerationSeconds : undefined,
        }))
      : [{ key: "", operator: "Equal", value: "", effect: undefined }];
  const volumes =
    Array.isArray(obj?.spec?.template?.spec?.volumes) && obj.spec.template.spec.volumes.length
      ? obj.spec.template.spec.volumes.map((v: any) => ({
          name: v?.name,
          type: v?.configMap ? "configMap" : v?.secret ? "secret" : v?.persistentVolumeClaim ? "pvc" : "emptyDir",
          source_name: v?.configMap?.name || v?.secret?.secretName || v?.persistentVolumeClaim?.claimName || "",
        }))
      : [{ name: "", type: "emptyDir", source_name: "" }];
  const volumeMounts =
    Array.isArray(c?.volumeMounts) && c.volumeMounts.length
      ? c.volumeMounts.map((m: any) => ({
          name: m?.name,
          mount_path: m?.mountPath,
          read_only: !!m?.readOnly,
          sub_path: m?.subPath,
        }))
      : [{ name: "", mount_path: "", read_only: false, sub_path: "" }];

  const imagePullSecrets =
    Array.isArray(obj?.spec?.template?.spec?.imagePullSecrets) && obj.spec.template.spec.imagePullSecrets.length
      ? obj.spec.template.spec.imagePullSecrets.map((s: any) => String(s?.name ?? "")).filter((x: string) => !!x)
      : [];

  const lp = c?.livenessProbe;
  const rp = c?.readinessProbe;
  const sp = c?.startupProbe;

  return {
    name: String(obj?.metadata?.name ?? ""),
    namespace: String(obj?.metadata?.namespace ?? "default"),
    replicas: Number(obj?.spec?.replicas ?? 1),
    container_name: String(c?.name ?? ""),
    image: String(c?.image ?? ""),
    image_pull_policy: (c?.imagePullPolicy as "Always" | "IfNotPresent" | "Never" | undefined) ?? undefined,
    image_pull_secrets: imagePullSecrets,
    port: typeof port === "number" ? port : undefined,
    port_name: c?.ports?.[0]?.name ? String(c.ports[0].name) : undefined,
    command: cmd || undefined,
    env_pairs: envPairs,
    requests_cpu: resReq?.cpu ? String(resReq.cpu) : undefined,
    requests_memory: resReq?.memory ? String(resReq.memory) : undefined,
    limits_cpu: resLim?.cpu ? String(resLim.cpu) : undefined,
    limits_memory: resLim?.memory ? String(resLim.memory) : undefined,
    tolerations,
    volumes,
    volume_mounts: volumeMounts,
    node_selector_pairs: mapToKvPairs(obj?.spec?.template?.spec?.nodeSelector),
    affinity_yaml: obj?.spec?.template?.spec?.affinity ? YAML.stringify(obj.spec.template.spec.affinity) : undefined,
    strategy_type: obj?.spec?.strategy?.type as DeploymentFormValues["strategy_type"],
    rolling_update_max_surge: obj?.spec?.strategy?.rollingUpdate?.maxSurge != null ? String(obj.spec.strategy.rollingUpdate.maxSurge) : undefined,
    rolling_update_max_unavailable:
      obj?.spec?.strategy?.rollingUpdate?.maxUnavailable != null ? String(obj.spec.strategy.rollingUpdate.maxUnavailable) : undefined,
    revision_history_limit: toNumberOrUndefined(obj?.spec?.revisionHistoryLimit),
    ...probeToForm("liveness", lp),
    ...probeToForm("readiness", rp),
    ...probeToForm("startup", sp),
  };
}

export function deploymentObjToForm(obj: any): DeploymentFormValues | null {
  if (!obj) return null;
  // Some JSON objects from API may not include kind/type meta; rely on structure presence instead.
  const container0 = safeGet(obj, "spec.template.spec.containers[0]");
  if (!container0) return null;
  const c = safeGet(obj, "spec.template.spec.containers[0]") ?? {};
  const envPairs = mapToEnvPairs(
    Array.isArray(c.env)
      ? Object.fromEntries(c.env.filter((e: any) => e?.name).map((e: any) => [String(e.name), String(e.value ?? "")]))
      : undefined,
  );
  const port = c?.ports?.[0]?.containerPort;
  let cmd = "";
  if (Array.isArray(c.command) && c.command.length >= 3 && c.command[0] === "sh" && c.command[1] === "-c") {
    cmd = String(c.command.slice(2).join(" "));
  }
  const resReq = c?.resources?.requests ?? {};
  const resLim = c?.resources?.limits ?? {};
  const tolerations =
    Array.isArray(safeGet(obj, "spec.template.spec.tolerations")) && safeGet(obj, "spec.template.spec.tolerations").length
      ? safeGet(obj, "spec.template.spec.tolerations").map((t: any) => ({
          key: t?.key,
          operator: (t?.operator || "Equal") as "Equal" | "Exists",
          value: t?.value,
          effect: t?.effect as any,
          toleration_seconds: typeof t?.tolerationSeconds === "number" ? t.tolerationSeconds : undefined,
        }))
      : [];
  const volumes =
    Array.isArray(safeGet(obj, "spec.template.spec.volumes")) && safeGet(obj, "spec.template.spec.volumes").length
      ? safeGet(obj, "spec.template.spec.volumes").map((v: any) => ({
          name: v?.name,
          type: v?.configMap ? "configMap" : v?.secret ? "secret" : v?.persistentVolumeClaim ? "pvc" : "emptyDir",
          source_name: v?.configMap?.name || v?.secret?.secretName || v?.persistentVolumeClaim?.claimName || "",
        }))
      : [];
  const volumeMounts =
    Array.isArray(c?.volumeMounts) && c.volumeMounts.length
      ? c.volumeMounts.map((m: any) => ({
          name: m?.name,
          mount_path: m?.mountPath,
          read_only: !!m?.readOnly,
          sub_path: m?.subPath,
        }))
      : [];

  const imagePullSecrets =
    Array.isArray(safeGet(obj, "spec.template.spec.imagePullSecrets")) && safeGet(obj, "spec.template.spec.imagePullSecrets")?.length
      ? (safeGet(obj, "spec.template.spec.imagePullSecrets") ?? []).map((s: any) => String(s?.name ?? "")).filter((x: string) => !!x)
      : [];

  const lp = c?.livenessProbe;
  const rp = c?.readinessProbe;
  const sp = c?.startupProbe;

  return {
    name: String(safeGet(obj, "metadata.name") ?? ""),
    namespace: String(safeGet(obj, "metadata.namespace") ?? "default"),
    replicas: Number(safeGet(obj, "spec.replicas") ?? 1),
    container_name: String(c?.name ?? ""),
    image: String(c?.image ?? ""),
    image_pull_policy: c?.imagePullPolicy as any,
    image_pull_secrets: imagePullSecrets,
    port: typeof port === "number" ? port : undefined,
    port_name: c?.ports?.[0]?.name ? String(c.ports[0].name) : undefined,
    command: cmd || undefined,
    env_pairs: envPairs,
    requests_cpu: resReq?.cpu ? String(resReq.cpu) : undefined,
    requests_memory: resReq?.memory ? String(resReq.memory) : undefined,
    limits_cpu: resLim?.cpu ? String(resLim.cpu) : undefined,
    limits_memory: resLim?.memory ? String(resLim.memory) : undefined,
    tolerations,
    volumes,
    volume_mounts: volumeMounts,
    node_selector_pairs: mapToKvPairs(safeGet(obj, "spec.template.spec.nodeSelector")),
    affinity_yaml: safeGet(obj, "spec.template.spec.affinity") ? YAML.stringify(safeGet(obj, "spec.template.spec.affinity")) : undefined,
    strategy_type: safeGet(obj, "spec.strategy.type") as DeploymentFormValues["strategy_type"],
    rolling_update_max_surge: safeGet(obj, "spec.strategy.rollingUpdate.maxSurge") != null ? String(safeGet(obj, "spec.strategy.rollingUpdate.maxSurge")) : undefined,
    rolling_update_max_unavailable:
      safeGet(obj, "spec.strategy.rollingUpdate.maxUnavailable") != null ? String(safeGet(obj, "spec.strategy.rollingUpdate.maxUnavailable")) : undefined,
    min_ready_seconds: toNumberOrUndefined(safeGet(obj, "spec.minReadySeconds")),
    progress_deadline_seconds: toNumberOrUndefined(safeGet(obj, "spec.progressDeadlineSeconds")),
    revision_history_limit: toNumberOrUndefined(safeGet(obj, "spec.revisionHistoryLimit")),
    ...probeToForm("liveness", lp),
    ...probeToForm("readiness", rp),
    ...probeToForm("startup", sp),
  };
}

export function qosFromResources(v: Pick<DeploymentFormValues, "requests_cpu" | "requests_memory" | "limits_cpu" | "limits_memory">): "BestEffort" | "Burstable" | "Guaranteed" {
  const rc = String(v.requests_cpu ?? "").trim();
  const rm = String(v.requests_memory ?? "").trim();
  const lc = String(v.limits_cpu ?? "").trim();
  const lm = String(v.limits_memory ?? "").trim();
  const hasReq = !!(rc || rm);
  const hasLim = !!(lc || lm);
  if (!hasReq && !hasLim) return "BestEffort";
  const guaranteed = rc && rm && lc && lm && rc === lc && rm === lm;
  return guaranteed ? "Guaranteed" : "Burstable";
}

