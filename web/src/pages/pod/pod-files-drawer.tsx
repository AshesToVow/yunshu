/**
 * Pod 文件管理抽屉（RF-04 拆分产物）
 * 从 pod-page.tsx 原地搬迁 JSX。
 */
import { DeleteOutlined, DownloadOutlined, ReloadOutlined, UploadOutlined } from "@ant-design/icons";
import { Button, Divider, Drawer, Input, Popconfirm, Space, Table, Typography, message } from "antd";
import type { MutableRefObject, RefObject } from "react";
import { deletePodFile, downloadPodFile, readPodFile, uploadPodFile, type PodFileItem, type PodItem } from "../../services/pods";

export type PodFilesDrawerProps = {
  fileOpen: boolean;
  setFileOpen: (v: boolean) => void;
  selected: PodItem | null;
  clusterId: number | undefined;
  filePath: string;
  setFilePath: (v: string) => void;
  fileList: PodFileItem[];
  fileLoading: boolean;
  fileContent: string;
  setFileContent: (v: string) => void;
  fileInputRef: RefObject<HTMLInputElement | null> | MutableRefObject<HTMLInputElement | null>;
  loadFiles: (pod: PodItem, path: string) => void | Promise<void>;
};

export function PodFilesDrawer({
  fileOpen,
  setFileOpen,
  selected,
  clusterId,
  filePath,
  setFilePath,
  fileList,
  fileLoading,
  fileContent,
  setFileContent,
  fileInputRef,
  loadFiles,
}: PodFilesDrawerProps) {
  return (
        <Drawer
          title={selected ? `Pod 文件管理 - ${selected.namespace}/${selected.name}` : "Pod 文件管理"}
          open={fileOpen}
          onClose={() => setFileOpen(false)}
          width={920}
        >
          <Space wrap style={{ marginBottom: 12 }}>
            <Input
              value={filePath}
              onChange={(e) => setFilePath(e.target.value)}
              placeholder="目录路径，例如 / /tmp /var/log"
              style={{ width: 360 }}
            />
            <Button onClick={() => selected && void loadFiles(selected, filePath)} icon={<ReloadOutlined />}>
              刷新目录
            </Button>
            <Button
              icon={<UploadOutlined />}
              onClick={() => fileInputRef.current?.click()}
              disabled={!selected}
            >
              上传到当前目录
            </Button>
            <input
            ref={fileInputRef as RefObject<HTMLInputElement>}
            type="file"
              style={{ display: "none" }}
              onChange={(e) => {
                const f = e.target.files?.[0];
                if (!f || !selected || !clusterId) return;
                void (async () => {
                  await uploadPodFile({
                    cluster_id: clusterId,
                    namespace: selected.namespace,
                    name: selected.name,
                    path: filePath || "/",
                    file: f,
                  });
                  message.success("上传成功");
                  await loadFiles(selected, filePath);
                })();
                e.currentTarget.value = "";
              }}
            />
          </Space>
          <Table
            rowKey={(r) => r.path}
            loading={fileLoading}
            dataSource={fileList}
            size="small"
            pagination={{ pageSize: 8 }}
            columns={[
              { title: "名称", dataIndex: "name" },
              { title: "类型", dataIndex: "type", width: 100 },
              { title: "大小", dataIndex: "size", width: 110 },
              { title: "权限", dataIndex: "permissions", width: 120 },
              { title: "修改时间", dataIndex: "mod_time", width: 140 },
              {
                title: "操作",
                width: 280,
                render: (_: unknown, row: PodFileItem) => (
                  <Space>
                    {row.is_dir ? (
                      <Button type="link" onClick={() => selected && void loadFiles(selected, row.path)}>
                        进入
                      </Button>
                    ) : (
                      <>
                        <Button
                          type="link"
                          onClick={() => {
                            if (!selected || !clusterId) return;
                            void (async () => {
                              const res = await readPodFile({
                                cluster_id: clusterId,
                                namespace: selected.namespace,
                                name: selected.name,
                                path: row.path,
                              });
                              setFileContent(res.content || "");
                            })();
                          }}
                        >
                          查看
                        </Button>
                        <Button
                          type="link"
                          icon={<DownloadOutlined />}
                          onClick={() => {
                            if (!selected || !clusterId) return;
                            void (async () => {
                              const blob = await downloadPodFile({
                                cluster_id: clusterId,
                                namespace: selected.namespace,
                                name: selected.name,
                                path: row.path,
                              });
                              const url = window.URL.createObjectURL(blob);
                              const a = document.createElement("a");
                              a.href = url;
                              a.download = row.name;
                              document.body.appendChild(a);
                              a.click();
                              a.remove();
                              window.URL.revokeObjectURL(url);
                            })();
                          }}
                        >
                          下载
                        </Button>
                      </>
                    )}
                    <Popconfirm
                      title={`确认删除 ${row.path} ?`}
                      onConfirm={() => {
                        if (!selected || !clusterId) return;
                        void (async () => {
                          await deletePodFile({
                            cluster_id: clusterId,
                            namespace: selected.namespace,
                            name: selected.name,
                            path: row.path,
                          });
                          message.success("删除成功");
                          await loadFiles(selected, filePath);
                        })();
                      }}
                    >
                      <Button type="link" danger icon={<DeleteOutlined />}>删除</Button>
                    </Popconfirm>
                  </Space>
                ),
              },
            ]}
          />
          <Divider />
          <Typography.Text strong>文件内容预览</Typography.Text>
          <Input.TextArea rows={14} value={fileContent} readOnly style={{ marginTop: 8 }} placeholder="点击“查看”显示文本内容" />
        </Drawer>
  );
}
