import { CloseOutlined, LinkOutlined } from "@ant-design/icons";
import { Button, Card, Progress, Space, Tag, Typography } from "antd";
import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import {
  useWorkloadProgress,
  type WorkloadProgressTask,
} from "../contexts/workload-progress-context";
import { listEvents, type EventItem } from "../services/events";
import { getNodeDrainStatus } from "../services/nodes";
import {
  getDeploymentRolloutStatus,
  listStatefulSetPods,
  type DeploymentRolloutStatus,
} from "../services/workloads";

type PanelState = {
  percent: number;
  status: "active" | "success" | "exception";
  detail: string;
  events: EventItem[];
  done: boolean;
};

const emptyState: PanelState = {
  percent: 0,
  status: "active",
  detail: "加载中…",
  events: [],
  done: false,
};

async function pollTask(task: WorkloadProgressTask): Promise<PanelState> {
  if (task.kind === "NodeDrain") {
    const st = await getNodeDrainStatus(task.clusterId, task.name);
    const totalHint = Math.max(st.remaining + 1, 1);
    const doneCount = st.drained ? totalHint : Math.max(0, totalHint - st.remaining - 1);
    return {
      percent: st.drained ? 100 : Math.min(95, Math.round((doneCount / totalHint) * 100)),
      status: st.drained ? "success" : "active",
      detail: st.message,
      events: [],
      done: st.drained,
    };
  }

  if (task.kind === "Deployment" && task.namespace) {
    const [st, events] = await Promise.all([
      getDeploymentRolloutStatus(task.clusterId, task.namespace, task.name),
      listEvents({
        cluster_id: task.clusterId,
        namespace: task.namespace,
        kind: "Deployment",
        name: task.name,
        limit: 8,
      }).catch(() => [] as EventItem[]),
    ]);
    return fromRollout(st, events);
  }

  if (task.kind === "StatefulSet" && task.namespace) {
    const [pods, events] = await Promise.all([
      listStatefulSetPods(task.clusterId, task.namespace, task.name),
      listEvents({
        cluster_id: task.clusterId,
        namespace: task.namespace,
        kind: "StatefulSet",
        name: task.name,
        limit: 8,
      }).catch(() => [] as EventItem[]),
    ]);
    const total = pods.length || 1;
    const ready = pods.filter((p) => p.phase === "Running").length;
    const complete = ready >= total && total > 0;
    return {
      percent: Math.round((ready / total) * 100),
      status: complete ? "success" : "active",
      detail: `就绪 Pod ${ready}/${pods.length}`,
      events: events || [],
      done: complete,
    };
  }

  return { ...emptyState, detail: "暂不支持该资源类型的进度追踪" };
}

function fromRollout(st: DeploymentRolloutStatus, events: EventItem[]): PanelState {
  const percent =
    st.replicas > 0 ? Math.round((st.ready_replicas / st.replicas) * 100) : st.complete ? 100 : 0;
  return {
    percent,
    status: st.complete ? "success" : "active",
    detail: `就绪 ${st.ready_replicas}/${st.replicas} · 已更新 ${st.updated_replicas} · 可用 ${st.available_replicas}${
      st.complete ? " · 完成" : st.progressing ? " · 滚动中" : ""
    }`,
    events: events || [],
    done: st.complete,
  };
}

function TaskCard({ task, onDismiss }: { task: WorkloadProgressTask; onDismiss: () => void }) {
  const [state, setState] = useState<PanelState>(emptyState);

  useEffect(() => {
    let cancelled = false;
    let timer: number | undefined;

    const tick = async () => {
      try {
        const next = await pollTask(task);
        if (cancelled) return;
        setState(next);
        if (next.done) {
          timer = window.setTimeout(() => {
            if (!cancelled) onDismiss();
          }, 8000);
          return;
        }
      } catch {
        if (!cancelled) {
          setState((s) => ({ ...s, detail: "进度刷新失败，稍后重试…", status: "exception" }));
        }
      }
      if (!cancelled) {
        timer = window.setTimeout(() => void tick(), 2500);
      }
    };

    void tick();
    return () => {
      cancelled = true;
      if (timer) window.clearTimeout(timer);
    };
  }, [task, onDismiss]);

  const link =
    task.kind === "NodeDrain"
      ? "/nodes"
      : task.kind === "Deployment"
        ? "/deployments"
        : task.kind === "StatefulSet"
          ? "/statefulsets"
          : undefined;

  return (
    <Card
      size="small"
      title={
        <Space size={6}>
          <Tag color={state.done ? "success" : "processing"}>{task.kind}</Tag>
          <Typography.Text ellipsis style={{ maxWidth: 180 }}>
            {task.title}
          </Typography.Text>
        </Space>
      }
      extra={
        <Button type="text" size="small" icon={<CloseOutlined />} onClick={onDismiss} />
      }
      style={{
        width: 360,
        boxShadow: "0 8px 24px rgba(0,0,0,0.18)",
        borderRadius: 10,
      }}
      styles={{ body: { paddingTop: 8 } }}
    >
      <Progress percent={state.percent} status={state.status} size="small" />
      <Typography.Paragraph type="secondary" style={{ marginBottom: 8, fontSize: 12 }}>
        {state.detail}
      </Typography.Paragraph>
      {state.events.length > 0 ? (
        <div style={{ maxHeight: 120, overflow: "auto", marginBottom: 8 }}>
          {state.events.slice(0, 5).map((e, i) => (
            <Typography.Text
              key={`${e.reason}-${i}`}
              type={e.type === "Warning" ? "danger" : "secondary"}
              style={{ display: "block", fontSize: 11, lineHeight: 1.4 }}
            >
              [{e.reason}] {e.message}
            </Typography.Text>
          ))}
        </div>
      ) : null}
      {link ? (
        <Link to={link}>
          <Button type="link" size="small" icon={<LinkOutlined />} style={{ padding: 0 }}>
            前往资源页
          </Button>
        </Link>
      ) : null}
    </Card>
  );
}

export function WorkloadProgressFloat() {
  const { tasks, dismiss } = useWorkloadProgress();
  if (!tasks.length) return null;

  return (
    <div
      style={{
        position: "fixed",
        right: 20,
        bottom: 20,
        zIndex: 1100,
        display: "flex",
        flexDirection: "column",
        gap: 10,
        pointerEvents: "none",
      }}
    >
      {tasks.map((t) => (
        <div key={t.id} style={{ pointerEvents: "auto" }}>
          <TaskCard task={t} onDismiss={() => dismiss(t.id)} />
        </div>
      ))}
    </div>
  );
}
