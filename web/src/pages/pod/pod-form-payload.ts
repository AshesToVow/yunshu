/**
 * Pod 简单表单（simple mode）的表单值类型与「表单值 ⇄ K8s 结构」纯转换逻辑。
 *
 * 从 pod-page.tsx 原地抽出（RF-04 第一步），仅搬迁不改语义：
 * - buildPodPairs：键值对数组 → Record（去空格、丢弃空 key）
 * - buildPodAffinityPayload：表单 affinity → K8s affinity（提交方向）
 * - buildPodTolerationsPayload：表单 tolerations → 提交体 tolerations
 * - podAffinityToForm：Pod 详情 affinity → 表单 affinity（回填方向）
 */

export type PodPairForm = { key?: string; value?: string };

export type PodMatchExpressionForm = {
  key?: string;
  operator?: "In" | "NotIn" | "Exists" | "DoesNotExist" | "Gt" | "Lt";
  values?: string[];
};

export type PodNodeAffinityForm = {
  required?: Array<{ match_expressions?: PodMatchExpressionForm[] }>;
  preferred?: Array<{ weight?: number; match_expressions?: PodMatchExpressionForm[] }>;
};

export type PodTopologyTermForm = { match_labels?: PodPairForm[]; topology_key?: string };

export type PodTopologyPreferredForm = {
  weight?: number;
  match_labels?: PodPairForm[];
  topology_key?: string;
};

export type PodTopologyAffinityForm = {
  required?: PodTopologyTermForm[];
  preferred?: PodTopologyPreferredForm[];
};

export type PodAffinityForm = {
  node?: PodNodeAffinityForm;
  pod?: PodTopologyAffinityForm;
  pod_anti?: PodTopologyAffinityForm;
};

export type PodTolerationForm = {
  key?: string;
  operator?: "Equal" | "Exists";
  value?: string;
  effect?: "NoSchedule" | "PreferNoSchedule" | "NoExecute";
  toleration_seconds?: number;
};

export type PodSimpleFormValues = {
  name: string;
  image: string;
  command?: string;
  container_name?: string;
  image_pull_policy?: "Always" | "IfNotPresent" | "Never";
  restart_policy?: "Always" | "OnFailure" | "Never";
  port?: number;
  env_pairs?: PodPairForm[];
  label_pairs?: PodPairForm[];
  requests_cpu?: string;
  requests_memory?: string;
  limits_cpu?: string;
  limits_memory?: string;
  tolerations?: PodTolerationForm[];
  node_selector_pairs?: PodPairForm[];
  priority_class_name?: string;
  affinity?: PodAffinityForm;
};

/** 键值对数组 → Record；key 去空格后为空则丢弃，value 同样 trim。 */
export function buildPodPairs(list?: PodPairForm[]): Record<string, string> {
  const out: Record<string, string> = {};
  (list || []).forEach((item) => {
    const k = (item?.key || "").trim();
    const v = (item?.value || "").trim();
    if (k) out[k] = v;
  });
  return out;
}

function buildMatchExpressions(list?: PodMatchExpressionForm[]) {
  return (list || [])
    .map((e) => ({
      key: (e.key || "").trim(),
      operator: e.operator,
      values: (e.values || []).map((v) => String(v).trim()).filter(Boolean),
    }))
    .filter((e) => e.key && e.operator);
}

function buildPodAffinityTerms(list?: PodTopologyTermForm[]) {
  return (list || [])
    .map((it) => {
      const matchLabels = buildPodPairs(it.match_labels);
      const topologyKey = (it.topology_key || "").trim();
      if (!topologyKey || Object.keys(matchLabels).length === 0) return null;
      return {
        labelSelector: { matchLabels },
        topologyKey,
      };
    })
    .filter(Boolean);
}

function buildPodPreferredTerms(list?: PodTopologyPreferredForm[]) {
  return (list || [])
    .map((it) => {
      const matchLabels = buildPodPairs(it.match_labels);
      const topologyKey = (it.topology_key || "").trim();
      if (!topologyKey || Object.keys(matchLabels).length === 0) return null;
      return {
        weight: Math.min(100, Math.max(1, Number(it.weight || 1))),
        podAffinityTerm: {
          labelSelector: { matchLabels },
          topologyKey,
        },
      };
    })
    .filter(Boolean);
}

/** 表单 affinity → K8s affinity；全部子项为空时返回 undefined（与原页面逻辑一致）。 */
export function buildPodAffinityPayload(form?: PodAffinityForm): Record<string, unknown> | undefined {
  if (!form) return undefined;

  const nodeRequiredTerms =
    form.node?.required
      ?.map((t) => ({ matchExpressions: buildMatchExpressions(t.match_expressions) }))
      .filter((t) => t.matchExpressions.length > 0) || [];

  const nodePreferred =
    form.node?.preferred
      ?.map((p) => ({
        weight: Math.min(100, Math.max(1, Number(p.weight || 1))),
        preference: { matchExpressions: buildMatchExpressions(p.match_expressions) },
      }))
      .filter((p) => p.preference.matchExpressions.length > 0) || [];

  const podRequired = buildPodAffinityTerms(form.pod?.required);
  const podPreferred = buildPodPreferredTerms(form.pod?.preferred);
  const podAntiRequired = buildPodAffinityTerms(form.pod_anti?.required);
  const podAntiPreferred = buildPodPreferredTerms(form.pod_anti?.preferred);

  const affinity: any = {};
  if (nodeRequiredTerms.length > 0 || nodePreferred.length > 0) {
    affinity.nodeAffinity = {};
    if (nodeRequiredTerms.length > 0) {
      affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution = {
        nodeSelectorTerms: nodeRequiredTerms,
      };
    }
    if (nodePreferred.length > 0) {
      affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution = nodePreferred;
    }
  }
  if (podRequired.length > 0 || podPreferred.length > 0) {
    affinity.podAffinity = {};
    if (podRequired.length > 0) affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution = podRequired;
    if (podPreferred.length > 0) affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution = podPreferred;
  }
  if (podAntiRequired.length > 0 || podAntiPreferred.length > 0) {
    affinity.podAntiAffinity = {};
    if (podAntiRequired.length > 0) {
      affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution = podAntiRequired;
    }
    if (podAntiPreferred.length > 0) {
      affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution = podAntiPreferred;
    }
  }
  return Object.keys(affinity).length > 0 ? affinity : undefined;
}

/** 表单 tolerations → 提交体；key 为空的行整条丢弃。 */
export function buildPodTolerationsPayload(list?: PodTolerationForm[]) {
  return (list || [])
    .filter((item) => (item.key || "").trim() !== "")
    .map((item) => ({
      key: (item.key || "").trim(),
      operator: item.operator || "Equal",
      value: (item.value || "").trim(),
      effect: item.effect,
      toleration_seconds: item.toleration_seconds,
    }));
}

function parsePodTerms(list: any[]): PodTopologyTermForm[] {
  return (list || []).map((t: any) => {
    const ml = t.labelSelector?.matchLabels || {};
    return {
      topology_key: t.topologyKey,
      match_labels: Object.entries(ml).map(([key, value]) => ({ key, value: value as string })),
    };
  });
}

function parsePodPreferred(list: any[]): PodTopologyPreferredForm[] {
  return (list || []).map((p: any) => {
    const term = p.podAffinityTerm || {};
    const ml = term.labelSelector?.matchLabels || {};
    return {
      weight: p.weight,
      topology_key: term.topologyKey,
      match_labels: Object.entries(ml).map(([key, value]) => ({ key, value: value as string })),
    };
  });
}

/** Pod 详情 affinity → 表单 affinity（编辑态回填）。 */
export function podAffinityToForm(affinity: unknown): PodAffinityForm {
  const a: any = affinity || {};
  const out: PodAffinityForm = {};
  if (a.nodeAffinity) {
    const reqTerms = a.nodeAffinity?.requiredDuringSchedulingIgnoredDuringExecution?.nodeSelectorTerms || [];
    const pref = a.nodeAffinity?.preferredDuringSchedulingIgnoredDuringExecution || [];
    out.node = {
      required: reqTerms.map((t: any) => ({
        match_expressions: (t.matchExpressions || []).map((e: any) => ({
          key: e.key,
          operator: e.operator,
          values: e.values || [],
        })),
      })),
      preferred: pref.map((p: any) => ({
        weight: p.weight,
        match_expressions: (p.preference?.matchExpressions || []).map((e: any) => ({
          key: e.key,
          operator: e.operator,
          values: e.values || [],
        })),
      })),
    };
  }
  if (a.podAffinity) {
    out.pod = {
      required: parsePodTerms(a.podAffinity.requiredDuringSchedulingIgnoredDuringExecution || []),
      preferred: parsePodPreferred(a.podAffinity.preferredDuringSchedulingIgnoredDuringExecution || []),
    };
  }
  if (a.podAntiAffinity) {
    out.pod_anti = {
      required: parsePodTerms(a.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution || []),
      preferred: parsePodPreferred(a.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution || []),
    };
  }
  return out;
}
