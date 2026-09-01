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
import type { DeploymentFormValues } from "./deployment";

export type JobFormValues = {
  name: string;
  namespace: string;
  restart_policy: "Never" | "OnFailure";
  container_name: string;
  image: string;
  image_pull_policy?: "Always" | "IfNotPresent" | "Never";
  command?: string;
  env_pairs?: EnvPair[];
  requests_cpu?: string;
  requests_memory?: string;
  limits_cpu?: string;
  limits_memory?: string;
  tolerations?: DeploymentFormValues["tolerations"];
  volumes?: DeploymentFormValues["volumes"];
  volume_mounts?: DeploymentFormValues["volume_mounts"];
  node_selector_pairs?: KVPair[];
  affinity_yaml?: string;
  parallelism?: number;
  completions?: number;
  backoff_limit?: number;
  active_deadline_seconds?: number;
  ttl_seconds_after_finished?: number;
  liveness_probe_type?: DeploymentFormValues["liveness_probe_type"];
  liveness_http_path?: DeploymentFormValues["liveness_http_path"];
  liveness_http_port?: DeploymentFormValues["liveness_http_port"];
  liveness_http_scheme?: DeploymentFormValues["liveness_http_scheme"];
  liveness_tcp_port?: DeploymentFormValues["liveness_tcp_port"];
  liveness_exec_command?: DeploymentFormValues["liveness_exec_command"];
  liveness_initial_delay_seconds?: DeploymentFormValues["liveness_initial_delay_seconds"];
  liveness_period_seconds?: DeploymentFormValues["liveness_period_seconds"];
  liveness_timeout_seconds?: DeploymentFormValues["liveness_timeout_seconds"];
  liveness_failure_threshold?: DeploymentFormValues["liveness_failure_threshold"];
  liveness_success_threshold?: DeploymentFormValues["liveness_success_threshold"];
  readiness_probe_type?: DeploymentFormValues["readiness_probe_type"];
  readiness_http_path?: DeploymentFormValues["readiness_http_path"];
  readiness_http_port?: DeploymentFormValues["readiness_http_port"];
  readiness_http_scheme?: DeploymentFormValues["readiness_http_scheme"];
  readiness_tcp_port?: DeploymentFormValues["readiness_tcp_port"];
  readiness_exec_command?: DeploymentFormValues["readiness_exec_command"];
  readiness_initial_delay_seconds?: DeploymentFormValues["readiness_initial_delay_seconds"];
  readiness_period_seconds?: DeploymentFormValues["readiness_period_seconds"];
  readiness_timeout_seconds?: DeploymentFormValues["readiness_timeout_seconds"];
  readiness_failure_threshold?: DeploymentFormValues["readiness_failure_threshold"];
  readiness_success_threshold?: DeploymentFormValues["readiness_success_threshold"];
  startup_probe_type?: DeploymentFormValues["startup_probe_type"];
  startup_http_path?: DeploymentFormValues["startup_http_path"];
  startup_http_port?: DeploymentFormValues["startup_http_port"];
  startup_http_scheme?: DeploymentFormValues["startup_http_scheme"];
  startup_tcp_port?: DeploymentFormValues["startup_tcp_port"];
  startup_exec_command?: DeploymentFormValues["startup_exec_command"];
  startup_initial_delay_seconds?: DeploymentFormValues["startup_initial_delay_seconds"];
  startup_period_seconds?: DeploymentFormValues["startup_period_seconds"];
  startup_timeout_seconds?: DeploymentFormValues["startup_timeout_seconds"];
  startup_failure_threshold?: DeploymentFormValues["startup_failure_threshold"];
  startup_success_threshold?: DeploymentFormValues["startup_success_threshold"];
};

export function buildJobYaml(v: JobFormValues): string {
  const envMap = envPairsToMap(v.env_pairs);
  const env = Object.keys(envMap).length
    ? Object.entries(envMap).map(([name, value]) => ({ name, value }))
    : undefined;
  const tolerations =
    (v.tolerations ?? [])
      .map((t) => ({
        key: String(t?.key ?? "").trim(),
        operator: t?.operator || "Equal",
        value: String(t?.value ?? "").trim(),
        effect: t?.effect || undefined,
        tolerationSeconds: typeof t?.toleration_seconds === "number" ? t.toleration_seconds : undefined,
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
        const name = String(it?.name ?? "").trim();
        if (!name) return null;
        const type = it?.type || "emptyDir";
        if (type === "configMap") {
          return { name, configMap: { name: String(it?.source_name ?? "").trim() || name } };
        }
        if (type === "secret") {
          return { name, secret: { secretName: String(it?.source_name ?? "").trim() || name } };
        }
        if (type === "pvc") {
          return { name, persistentVolumeClaim: { claimName: String(it?.source_name ?? "").trim() || name } };
        }
        return { name, emptyDir: {} };
      })
      .filter(Boolean);

  const volumeMounts =
    (v.volume_mounts ?? [])
      .map((m) => ({
        name: String(m?.name ?? "").trim(),
        mountPath: String(m?.mount_path ?? "").trim(),
        readOnly: !!m?.read_only,
        subPath: String(m?.sub_path ?? "").trim() || undefined,
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
    apiVersion: "batch/v1",
    kind: "Job",
    metadata: { name: v.name, namespace: v.namespace },
    spec: {
      parallelism: typeof v.parallelism === "number" ? v.parallelism : undefined,
      completions: typeof v.completions === "number" ? v.completions : undefined,
      backoffLimit: typeof v.backoff_limit === "number" ? v.backoff_limit : undefined,
      activeDeadlineSeconds: typeof v.active_deadline_seconds === "number" ? v.active_deadline_seconds : undefined,
      ttlSecondsAfterFinished: typeof v.ttl_seconds_after_finished === "number" ? v.ttl_seconds_after_finished : undefined,
      template: {
        spec: {
          restartPolicy: v.restart_policy,
          tolerations: tolerations?.length ? tolerations : undefined,
          nodeSelector: Object.keys(nodeSelector).length ? nodeSelector : undefined,
          affinity: affinity || undefined,
          volumes: volumes.length ? volumes : undefined,
          containers: [
            {
              name: v.container_name || v.name,
              image: v.image,
              imagePullPolicy: v.image_pull_policy || undefined,
              command: v.command?.trim() ? ["sh", "-c", v.command.trim()] : undefined,
              env,
              resources: Object.keys(resources).length ? resources : undefined,
              volumeMounts,
              ...(livenessProbe ? { livenessProbe } : {}),
              ...(readinessProbe ? { readinessProbe } : {}),
              ...(startupProbe ? { startupProbe } : {}),
            },
          ],
        },
      },
    },
  };
  return YAML.stringify(obj);
}

export function jobYamlToForm(yaml: string): JobFormValues | null {
  const obj: any = safeParseYaml(yaml);
  if (!obj || obj.kind !== "Job") return null;
  const c = obj?.spec?.template?.spec?.containers?.[0] ?? {};
  const envPairs = mapToEnvPairs(
    Array.isArray(c.env)
      ? Object.fromEntries(c.env.filter((e: any) => e?.name).map((e: any) => [String(e.name), String(e.value ?? "")]))
      : undefined,
  );
  let cmd = "";
  if (Array.isArray(c.command) && c.command.length >= 3 && c.command[0] === "sh" && c.command[1] === "-c") {
    cmd = String(c.command.slice(2).join(" "));
  }
  const rp = String(obj?.spec?.template?.spec?.restartPolicy ?? "Never");
  const restart_policy = rp === "OnFailure" ? "OnFailure" : "Never";
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
  const lp = c?.livenessProbe;
  const rp2 = c?.readinessProbe;
  const sp = c?.startupProbe;
  return {
    name: String(obj?.metadata?.name ?? ""),
    namespace: String(obj?.metadata?.namespace ?? "default"),
    restart_policy,
    container_name: String(c?.name ?? ""),
    image: String(c?.image ?? ""),
    image_pull_policy: (c?.imagePullPolicy as "Always" | "IfNotPresent" | "Never" | undefined) ?? undefined,
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
    parallelism: toNumberOrUndefined(safeGet(obj, "spec.parallelism")),
    completions: toNumberOrUndefined(safeGet(obj, "spec.completions")),
    backoff_limit: toNumberOrUndefined(safeGet(obj, "spec.backoffLimit")),
    active_deadline_seconds: toNumberOrUndefined(safeGet(obj, "spec.activeDeadlineSeconds")),
    ttl_seconds_after_finished: toNumberOrUndefined(safeGet(obj, "spec.ttlSecondsAfterFinished")),
    ...probeToForm("liveness", lp),
    ...probeToForm("readiness", rp2),
    ...probeToForm("startup", sp),
  };
}

export function jobObjToForm(obj: any): JobFormValues | null {
  if (!obj) return null;
  const c = safeGet(obj, "spec.template.spec.containers[0]") ?? null;
  if (!c) return null;
  const envPairs = mapToEnvPairs(
    Array.isArray(c.env)
      ? Object.fromEntries(c.env.filter((e: any) => e?.name).map((e: any) => [String(e.name), String(e.value ?? "")]))
      : undefined,
  );
  let cmd = "";
  if (Array.isArray(c.command) && c.command.length >= 3 && c.command[0] === "sh" && c.command[1] === "-c") {
    cmd = String(c.command.slice(2).join(" "));
  }
  const rp = String(safeGet(obj, "spec.template.spec.restartPolicy") ?? "Never");
  const restart_policy: JobFormValues["restart_policy"] = rp === "OnFailure" ? "OnFailure" : "Never";
  const resReq = c?.resources?.requests ?? {};
  const resLim = c?.resources?.limits ?? {};
  const tolerations =
    Array.isArray(safeGet(obj, "spec.template.spec.tolerations")) && safeGet(obj, "spec.template.spec.tolerations")?.length
      ? (safeGet(obj, "spec.template.spec.tolerations") ?? []).map((t: any) => ({
          key: t?.key,
          operator: (t?.operator || "Equal") as "Equal" | "Exists",
          value: t?.value,
          effect: t?.effect as any,
          toleration_seconds: typeof t?.tolerationSeconds === "number" ? t.tolerationSeconds : undefined,
        }))
      : [];
  const volumes =
    Array.isArray(safeGet(obj, "spec.template.spec.volumes")) && safeGet(obj, "spec.template.spec.volumes")?.length
      ? (safeGet(obj, "spec.template.spec.volumes") ?? []).map((v: any) => ({
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
  const lp = c?.livenessProbe;
  const rp2 = c?.readinessProbe;
  const sp = c?.startupProbe;
  return {
    name: String(safeGet(obj, "metadata.name") ?? ""),
    namespace: String(safeGet(obj, "metadata.namespace") ?? "default"),
    restart_policy,
    container_name: String(c?.name ?? ""),
    image: String(c?.image ?? ""),
    image_pull_policy: c?.imagePullPolicy as any,
    command: cmd || undefined,
    env_pairs: envPairs,
    requests_cpu: resReq?.cpu ? String(resReq.cpu) : undefined,
    requests_memory: resReq?.memory ? String(resReq.memory) : undefined,
    limits_cpu: resLim?.cpu ? String(resLim.cpu) : undefined,
    limits_memory: resLim?.memory ? String(resLim.memory) : undefined,
    tolerations,
    volumes,
    volume_mounts: volumeMounts,
    ...probeToForm("liveness", lp),
    ...probeToForm("readiness", rp2),
    ...probeToForm("startup", sp),
  };
}

