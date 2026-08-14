import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";

export type WorkloadProgressKind = "Deployment" | "StatefulSet" | "DaemonSet" | "NodeDrain";

export type WorkloadProgressTask = {
  id: string;
  kind: WorkloadProgressKind;
  clusterId: number;
  namespace?: string;
  name: string;
  title: string;
  startedAt: number;
};

type Ctx = {
  tasks: WorkloadProgressTask[];
  track: (task: Omit<WorkloadProgressTask, "id" | "startedAt">) => string;
  dismiss: (id: string) => void;
  clear: () => void;
};

const WorkloadProgressContext = createContext<Ctx | null>(null);

export function WorkloadProgressProvider({ children }: { children: ReactNode }) {
  const [tasks, setTasks] = useState<WorkloadProgressTask[]>([]);

  const track = useCallback((task: Omit<WorkloadProgressTask, "id" | "startedAt">) => {
    const id = `${task.kind}-${task.clusterId}-${task.namespace || ""}-${task.name}-${Date.now()}`;
    setTasks((prev) => {
      const filtered = prev.filter(
        (t) =>
          !(
            t.kind === task.kind &&
            t.clusterId === task.clusterId &&
            t.namespace === task.namespace &&
            t.name === task.name
          ),
      );
      return [...filtered, { ...task, id, startedAt: Date.now() }].slice(-5);
    });
    return id;
  }, []);

  const dismiss = useCallback((id: string) => {
    setTasks((prev) => prev.filter((t) => t.id !== id));
  }, []);

  const clear = useCallback(() => setTasks([]), []);

  const value = useMemo(() => ({ tasks, track, dismiss, clear }), [tasks, track, dismiss, clear]);

  return <WorkloadProgressContext.Provider value={value}>{children}</WorkloadProgressContext.Provider>;
}

export function useWorkloadProgress() {
  const ctx = useContext(WorkloadProgressContext);
  if (!ctx) {
    throw new Error("useWorkloadProgress must be used within WorkloadProgressProvider");
  }
  return ctx;
}

/** 可选：页面未包 Provider 时不抛错 */
export function useWorkloadProgressOptional() {
  return useContext(WorkloadProgressContext);
}
