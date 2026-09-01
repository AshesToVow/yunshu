// @ts-nocheck
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

export type StatefulSetFormValues = {
  name: string;
  namespace: string;
  service_name: string;
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
  update_strategy_type?: "RollingUpdate" | "OnDelete";
  rolling_update_partition?: number;
  revision_history_limit?: number;
  tolerations?: DeploymentFormValues["tolerations"];
  volumes?: DeploymentFormValues["volumes"];
  volume_mounts?: DeploymentFormValues["volume_mounts"];
};

export function buildStatefulSetYaml(v: StatefulSetFormValues): string {
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
  const nodeSelector = kvPairsToMap(v.node_selector_pairs);
  const affinity = safeParseYaml(String(v.affinity_yaml ?? "").trim() || "");

  const obj: any = {
    apiVersion: "apps/v1",
    kind: "StatefulSet",
    metadata: { name: v.name, namespace: v.namespace },
    spec: {
      serviceName: v.service_name || `${v.name}-headless`,
      replicas: v.replicas,
      updateStrategy: v.update_strategy_type
        ? {
            type: v.update_strategy_type,
            ...(v.update_strategy_type === "RollingUpdate" && typeof v.rolling_update_partition === "number"
              ? { rollingUpdate: { partition: v.rolling_update_partition } }
              : {}),
          }
        : undefined,
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
            },
          ],
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

export function statefulSetYamlToForm(yaml: string): StatefulSetFormValues | null {
  const obj: any = safeParseYaml(yaml);
  if (!obj || obj.kind !== "StatefulSet") return null;
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
  return {
    name: String(obj?.metadata?.name ?? ""),
    namespace: String(obj?.metadata?.namespace ?? "default"),
    service_name: String(obj?.spec?.serviceName ?? ""),
    replicas: Number(obj?.spec?.replicas ?? 1),
    container_name: String(c?.name ?? ""),
    image: String(c?.image ?? ""),
    image_pull_policy: (c?.imagePullPolicy as "Always" | "IfNotPresent" | "Never" | undefined) ?? undefined,
    port: typeof port === "number" ? port : undefined,
    port_name: c?.ports?.[0]?.name ? String(c.ports[0].name) : undefined,
    command: cmd || undefined,
    env_pairs: envPairs,
    requests_cpu: resReq?.cpu ? String(resReq.cpu) : undefined,
    requests_memory: resReq?.memory ? String(resReq.memory) : undefined,
    limits_cpu: resLim?.cpu ? String(resLim.cpu) : undefined,
    limits_memory: resLim?.memory ? String(resLim.memory) : undefined,
    node_selector_pairs: mapToKvPairs(obj?.spec?.template?.spec?.nodeSelector),
    affinity_yaml: obj?.spec?.template?.spec?.affinity ? YAML.stringify(obj.spec.template.spec.affinity) : undefined,
    update_strategy_type: obj?.spec?.updateStrategy?.type as StatefulSetFormValues["update_strategy_type"],
    rolling_update_partition: toNumberOrUndefined(obj?.spec?.updateStrategy?.rollingUpdate?.partition),
    revision_history_limit: toNumberOrUndefined(obj?.spec?.revisionHistoryLimit),
    tolerations,
    volumes,
    volume_mounts: volumeMounts,
  };
}

export function statefulSetObjToForm(obj: any): StatefulSetFormValues | null {
  if (!obj) return null;
  const c = safeGet(obj, "spec.template.spec.containers[0]") ?? null;
  if (!c) return null;
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
  return {
    name: String(safeGet(obj, "metadata.name") ?? ""),
    namespace: String(safeGet(obj, "metadata.namespace") ?? "default"),
    service_name: String(safeGet(obj, "spec.serviceName") ?? ""),
    replicas: Number(safeGet(obj, "spec.replicas") ?? 1),
    container_name: String(c?.name ?? ""),
    image: String(c?.image ?? ""),
    image_pull_policy: c?.imagePullPolicy as any,
    port: typeof port === "number" ? port : undefined,
    port_name: c?.ports?.[0]?.name ? String(c.ports[0].name) : undefined,
    command: cmd || undefined,
    env_pairs: envPairs,
    requests_cpu: resReq?.cpu ? String(resReq.cpu) : undefined,
    requests_memory: resReq?.memory ? String(resReq.memory) : undefined,
    limits_cpu: resLim?.cpu ? String(resLim.cpu) : undefined,
    limits_memory: resLim?.memory ? String(resLim.memory) : undefined,
    node_selector_pairs: mapToKvPairs(safeGet(obj, "spec.template.spec.nodeSelector")),
    affinity_yaml: safeGet(obj, "spec.template.spec.affinity") ? YAML.stringify(safeGet(obj, "spec.template.spec.affinity")) : undefined,
    update_strategy_type: safeGet(obj, "spec.updateStrategy.type") as StatefulSetFormValues["update_strategy_type"],
    rolling_update_partition: toNumberOrUndefined(safeGet(obj, "spec.updateStrategy.rollingUpdate.partition")),
    revision_history_limit: toNumberOrUndefined(safeGet(obj, "spec.revisionHistoryLimit")),
    tolerations,
    volumes,
    volume_mounts: volumeMounts,
  };
}

