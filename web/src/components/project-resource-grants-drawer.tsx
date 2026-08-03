import { Button, Checkbox, Drawer, Popconfirm, Select, Space, Table, Tabs, Typography, message } from "antd";
import { useEffect, useMemo, useState } from "react";
import {
  bootstrapCicdAccessGrants,
  bootstrapServerAccessGrants,
  bulkUpsertCicdAccessGrants,
  bulkUpsertServerAccessGrants,
  deleteCicdAccessGrant,
  deleteServerAccessGrant,
  listCicdAccessGrants,
  listServerAccessGrants,
  type CicdAccessGrantItem,
  type ServerAccessGrantItem,
} from "../services/project-resource-grants";
import { listCicdServices, type CicdServiceItem } from "../services/cicd";
import { getProjectServers, type ServerItem } from "../services/projects";
import type { ProjectMemberItem } from "../services/projects";

type Props = {
  open: boolean;
  projectId: number;
  members: ProjectMemberItem[];
  onClose: () => void;
};

export function ProjectResourceGrantsDrawer({ open, projectId, members, onClose }: Props) {
  const [tab, setTab] = useState("server");
  const [loading, setLoading] = useState(false);
  const [serverGrants, setServerGrants] = useState<ServerAccessGrantItem[]>([]);
  const [cicdGrants, setCicdGrants] = useState<CicdAccessGrantItem[]>([]);
  const [servers, setServers] = useState<ServerItem[]>([]);
  const [services, setServices] = useState<CicdServiceItem[]>([]);
  const [userId, setUserId] = useState<number>();
  const [resourceIds, setResourceIds] = useState<number[]>([]);
  const [caps, setCaps] = useState({ view: true, exec: true, build: true, release: true, manage: false });

  const memberOptions = useMemo(
    () =>
      members
        .filter((m) => m.role !== "owner" && m.role !== "admin")
        .map((m) => ({
          value: m.user_id,
          label: `${m.nickname || m.username || m.user_id} (${m.role})`,
        })),
    [members],
  );

  async function reload() {
    setLoading(true);
    try {
      const [sg, cg, sv, svc] = await Promise.all([
        listServerAccessGrants(projectId),
        listCicdAccessGrants(projectId),
        getProjectServers(projectId, { page: 1, page_size: 1000 }),
        listCicdServices(projectId, { page: 1, page_size: 1000 }),
      ]);
      setServerGrants(sg);
      setCicdGrants(cg);
      setServers(sv.list ?? []);
      setServices(svc.list ?? []);
    } catch (e) {
      message.error(e instanceof Error ? e.message : "加载授权失败");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    if (open) void reload();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, projectId]);

  async function onGrant() {
    if (!userId || resourceIds.length === 0) {
      message.warning("请选择成员与资源");
      return;
    }
    try {
      if (tab === "server") {
        await bulkUpsertServerAccessGrants(projectId, {
          user_id: userId,
          server_ids: resourceIds,
          can_view: caps.view,
          can_exec: caps.exec,
          can_manage: caps.manage,
        });
      } else {
        await bulkUpsertCicdAccessGrants(projectId, {
          user_id: userId,
          service_ids: resourceIds,
          can_view: caps.view,
          can_build: caps.build,
          can_release: caps.release,
          can_manage: caps.manage,
        });
      }
      message.success("已保存授权");
      setResourceIds([]);
      await reload();
    } catch (e) {
      message.error(e instanceof Error ? e.message : "保存失败");
    }
  }

  async function onBootstrap() {
    try {
      if (tab === "server") {
        const r = await bootstrapServerAccessGrants(projectId);
        message.success(`服务器授权迁移完成：写入 ${r.grants_upserted ?? 0} 条`);
      } else {
        const r = await bootstrapCicdAccessGrants(projectId);
        message.success(`CI/CD 授权迁移完成：写入 ${r.grants_upserted ?? 0} 条`);
      }
      await reload();
    } catch (e) {
      message.error(e instanceof Error ? e.message : "迁移失败");
    }
  }

  return (
    <Drawer title="项目资源授权" width={860} open={open} onClose={onClose} destroyOnClose>
      <Typography.Paragraph type="secondary" style={{ marginTop: 0 }}>
        owner/admin 默认可见全部资源，无需授权。普通成员/只读成员仅能访问下方已授权的服务器或 CI/CD 应用。
      </Typography.Paragraph>
      <Tabs
        activeKey={tab}
        onChange={(k) => {
          setTab(k);
          setResourceIds([]);
        }}
        items={[
          { key: "server", label: "服务器" },
          { key: "cicd", label: "CI/CD 应用" },
        ]}
      />
      <Space wrap style={{ marginBottom: 12, width: "100%" }}>
        <Select
          placeholder="选择成员"
          style={{ minWidth: 220 }}
          options={memberOptions}
          value={userId}
          onChange={setUserId}
          showSearch
          optionFilterProp="label"
        />
        <Select
          mode="multiple"
          placeholder={tab === "server" ? "选择服务器" : "选择应用"}
          style={{ minWidth: 320 }}
          value={resourceIds}
          onChange={setResourceIds}
          options={
            tab === "server"
              ? servers.map((s) => ({ value: s.id, label: `${s.name} (${s.host})` }))
              : services.map((s) => ({ value: s.id, label: `${s.name} (${s.identifier})` }))
          }
          optionFilterProp="label"
        />
        {tab === "server" ? (
          <Checkbox.Group
            options={[
              { label: "查看", value: "view" },
              { label: "SSH/执行", value: "exec" },
              { label: "管理", value: "manage" },
            ]}
            value={[caps.view && "view", caps.exec && "exec", caps.manage && "manage"].filter(Boolean) as string[]}
            onChange={(vals) =>
              setCaps((c) => ({
                ...c,
                view: vals.includes("view") || vals.includes("exec") || vals.includes("manage"),
                exec: vals.includes("exec") || vals.includes("manage"),
                manage: vals.includes("manage"),
              }))
            }
          />
        ) : (
          <Checkbox.Group
            options={[
              { label: "查看", value: "view" },
              { label: "构建", value: "build" },
              { label: "发布", value: "release" },
              { label: "管理", value: "manage" },
            ]}
            value={
              [caps.view && "view", caps.build && "build", caps.release && "release", caps.manage && "manage"].filter(
                Boolean,
              ) as string[]
            }
            onChange={(vals) =>
              setCaps((c) => ({
                ...c,
                view: vals.includes("view") || vals.includes("build") || vals.includes("release") || vals.includes("manage"),
                build: vals.includes("build") || vals.includes("manage"),
                release: vals.includes("release") || vals.includes("manage"),
                manage: vals.includes("manage"),
              }))
            }
          />
        )}
        <Button type="primary" onClick={() => void onGrant()}>
          授予
        </Button>
        <Popconfirm title="给全部非 admin 成员授予当前全部资源（view+操作）？" onConfirm={() => void onBootstrap()}>
          <Button>一键迁移存量</Button>
        </Popconfirm>
      </Space>
      {tab === "server" ? (
        <Table
          rowKey="id"
          loading={loading}
          size="small"
          dataSource={serverGrants}
          pagination={{ pageSize: 10 }}
          columns={[
            { title: "成员", render: (_: unknown, r: ServerAccessGrantItem) => r.nickname || r.username || r.principal_ref },
            { title: "服务器", render: (_: unknown, r: ServerAccessGrantItem) => `${r.server_name || r.server_id} ${r.server_host || ""}` },
            {
              title: "权限",
              render: (_: unknown, r: ServerAccessGrantItem) =>
                [r.can_view && "查看", r.can_exec && "执行", r.can_manage && "管理"].filter(Boolean).join(" / "),
            },
            {
              title: "操作",
              width: 80,
              render: (_: unknown, r: ServerAccessGrantItem) => (
                <Popconfirm title="删除该授权？" onConfirm={() => void deleteServerAccessGrant(projectId, r.id).then(reload)}>
                  <Button type="link" danger size="small">
                    删除
                  </Button>
                </Popconfirm>
              ),
            },
          ]}
        />
      ) : (
        <Table
          rowKey="id"
          loading={loading}
          size="small"
          dataSource={cicdGrants}
          pagination={{ pageSize: 10 }}
          columns={[
            { title: "成员", render: (_: unknown, r: CicdAccessGrantItem) => r.nickname || r.username || r.principal_ref },
            { title: "应用", render: (_: unknown, r: CicdAccessGrantItem) => r.service_name || r.service_id },
            {
              title: "权限",
              render: (_: unknown, r: CicdAccessGrantItem) =>
                [r.can_view && "查看", r.can_build && "构建", r.can_release && "发布", r.can_manage && "管理"]
                  .filter(Boolean)
                  .join(" / "),
            },
            {
              title: "操作",
              width: 80,
              render: (_: unknown, r: CicdAccessGrantItem) => (
                <Popconfirm title="删除该授权？" onConfirm={() => void deleteCicdAccessGrant(projectId, r.id).then(reload)}>
                  <Button type="link" danger size="small">
                    删除
                  </Button>
                </Popconfirm>
              ),
            },
          ]}
        />
      )}
    </Drawer>
  );
}
