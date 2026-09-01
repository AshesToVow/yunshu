// @ts-nocheck
import { create } from "zustand";
import { persist } from "zustand/middleware";

export type K8sContextState = {
  clusterId?: number;
  namespace: string;
  setClusterId: (id?: number) => void;
  setNamespace: (ns: string) => void;
};

export const useK8sContextStore = create<K8sContextState>()(
  persist(
    (set) => ({
      namespace: "default",
      setClusterId: (clusterId) => set({ clusterId }),
      setNamespace: (namespace) => set({ namespace }),
    }),
    {
      name: "yunshu-k8s-context",
      partialize: (state) => ({
        clusterId: state.clusterId,
        namespace: state.namespace,
      }),
    },
  ),
);
