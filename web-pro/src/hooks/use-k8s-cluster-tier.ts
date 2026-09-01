// @ts-nocheck
import { useEffect, useState } from "react";
import { getMyK8sAccess, K8S_ACCESS_RANK } from "../services/k8s-my-access";

export type K8sClusterTierCapabilities = {
  rank: number;
  preset: string;
  canRead: boolean;
  canExec: boolean;
  canMutate: boolean;
  loading: boolean;
};

/** 按当前用户档位控制 K8s 页面操作按钮可见性 */
export function useK8sClusterTier(clusterId?: number): K8sClusterTierCapabilities {
  const [rank, setRank] = useState(0);
  const [preset, setPreset] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!clusterId) {
      setRank(0);
      setPreset("");
      return;
    }
    let cancelled = false;
    setLoading(true);
    void getMyK8sAccess(clusterId)
      .then((res) => {
        if (cancelled) return;
        setRank(res.access_rank ?? 0);
        setPreset(res.access_preset ?? "");
      })
      .catch(() => {
        if (cancelled) return;
        setRank(0);
        setPreset("");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [clusterId]);

  return {
    rank,
    preset,
    canRead: rank >= K8S_ACCESS_RANK.readonly,
    canExec: rank >= K8S_ACCESS_RANK.readonlyExec,
    canMutate: rank >= K8S_ACCESS_RANK.admin,
    loading,
  };
}
