// @ts-nocheck
/**
 * K8s 资源表单创建抽屉的外壳组件。
 *
 * 从 k8s-resource-form-drawers.tsx 原地抽出（RF-11 第一步），仅搬迁不改语义。
 * 两种形态由 `embedded` 决定：
 * - embedded=false：完整 Drawer（右侧抽屉 + extra 区「取消 / 创建」按钮）
 * - embedded=true：只渲染 Form + 底部「创建」，用于嵌入 YamlCrudPage 创建抽屉的「表单」Tab
 *
 * 注意：`maskClosable={false}` 与 `destroyOnClose` 是刻意组合——防止误点遮罩丢失已填表单，
 * 同时保证下次打开时 Form 状态干净（配合 Form 的 preserve={false}）。
 */

import { Button, Drawer, Form, Space } from "antd";
import type { FormInstance } from "antd/es/form";

export function DrawerShellForm(props: {
  title: string;
  open: boolean;
  width?: number;
  form: FormInstance;
  onClose: () => void;
  loading: boolean;
  onSubmit: () => void;
  children: React.ReactNode;
  initialValues?: Record<string, unknown>;
  /** 仅表单 + 底部「创建」，不包 Drawer（嵌入 YamlCrudPage 创建抽屉的「表单」Tab） */
  embedded?: boolean;
}) {
  const { title, open, width = 720, form, onClose, loading, onSubmit, children, initialValues, embedded } = props;
  const formNode = (
    <Form form={form} layout="vertical" requiredMark="optional" preserve={false} scrollToFirstError initialValues={initialValues}>
      {children}
    </Form>
  );
  if (embedded) {
    return (
      <>
        {formNode}
        <Space style={{ marginTop: 16 }}>
          <Button type="primary" loading={loading} onClick={() => void onSubmit()}>
            创建
          </Button>
        </Space>
      </>
    );
  }
  return (
    <Drawer
      title={title}
      placement="right"
      width={width}
      open={open}
      onClose={onClose}
      destroyOnClose
      maskClosable={false}
      styles={{ body: { paddingBottom: 24 } }}
      extra={
        <Space>
          <Button onClick={onClose}>取消</Button>
          <Button type="primary" loading={loading} onClick={() => void onSubmit()}>
            创建
          </Button>
        </Space>
      }
    >
      {formNode}
    </Drawer>
  );
}
