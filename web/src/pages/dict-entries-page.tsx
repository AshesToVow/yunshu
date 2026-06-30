import { DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Form, Input, InputNumber, Modal, Popconfirm, Segmented, Select, Space, Table, Tag, Tooltip, Typography, message } from "antd";
import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import {
  DICT_CATEGORY_META,
  DICT_CATEGORY_TABS,
  buildGroupedDictTypeSelectOptions,
  getDictCategoryLabel,
  matchesDictCategory,
  resolveDictCategory,
  type DictCategoryId,
} from "../constants/dict-types";
import { createDictEntry, deleteDictEntry, getDictEntries, updateDictEntry, type DictEntryItem, type DictPayload } from "../services/dict";
import { formatDateTime } from "../utils/format";

const defaultQuery = { keyword: "", dict_type: "", category: "all" as DictCategoryId, page: 1, page_size: 10 };

function parseCategory(raw: string | null): DictCategoryId {
  const value = String(raw || "").trim() as DictCategoryId;
  return DICT_CATEGORY_TABS.some((tab) => tab.id === value) ? value : "all";
}

export function DictEntriesPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const initialDictType = String(searchParams.get("dict_type") || "").trim();
  const initialKeyword = String(searchParams.get("keyword") || "").trim();
  const initialCategory = parseCategory(searchParams.get("category"));
  const [list, setList] = useState<DictEntryItem[]>([]);
  const [total, setTotal] = useState(0);
  const [query, setQuery] = useState(() => ({
    ...defaultQuery,
    dict_type: initialDictType,
    keyword: initialKeyword,
    category: initialCategory,
  }));
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [current, setCurrent] = useState<DictEntryItem | null>(null);
  const [open, setOpen] = useState(false);
  const [form] = Form.useForm<DictPayload>();
  const formDictType = Form.useWatch("dict_type", form) as string | undefined;
  const autoSortLocked = !current && String(formDictType || "") === "alert_promql_label_key";
  const isAlertLabelGovernedDictType = String(formDictType || "").trim() === "alert_promql_label_key";

  const extraDictTypes = useMemo(() => {
    const values = new Set<string>();
    for (const item of list) {
      const v = String(item.dict_type || "").trim();
      if (v) values.add(v);
    }
    if (current?.dict_type) values.add(current.dict_type);
    if (formDictType) values.add(String(formDictType).trim());
    return Array.from(values);
  }, [list, current?.dict_type, formDictType]);

  const groupedDictTypeOptions = useMemo(
    () => buildGroupedDictTypeSelectOptions(query.category, extraDictTypes),
    [query.category, extraDictTypes],
  );

  const activeCategoryMeta = query.category !== "all" ? DICT_CATEGORY_META[query.category] : null;

  useEffect(() => {
    void loadData(query);
  }, [query]);

  useEffect(() => {
    const dt = String(searchParams.get("dict_type") || "").trim();
    const kw = String(searchParams.get("keyword") || "").trim();
    const cat = parseCategory(searchParams.get("category"));
    setQuery((prev) => {
      if (prev.dict_type === dt && prev.keyword === kw && prev.category === cat) return prev;
      return { ...prev, dict_type: dt, keyword: kw, category: cat, page: 1 };
    });
  }, [searchParams]);

  useEffect(() => {
    if (!open || current || !formDictType) return;
    let cancelled = false;
    void (async () => {
      try {
        const result = await getDictEntries({ dict_type: formDictType, page: 1, page_size: 500 });
        if (cancelled) return;
        const maxSort = (result.list ?? []).reduce((max, it) => Math.max(max, Number(it.sort || 0)), 0);
        const currentSort = form.getFieldValue("sort");
        if (currentSort == null || Number.isNaN(Number(currentSort)) || autoSortLocked) {
          form.setFieldValue("sort", maxSort + 1);
        }
      } catch {
        if (!cancelled && autoSortLocked) {
          form.setFieldValue("sort", 1);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [open, current, formDictType, form, autoSortLocked]);

  function syncSearchParams(next: typeof query) {
    const params = new URLSearchParams();
    if (next.category && next.category !== "all") params.set("category", next.category);
    if (next.dict_type) params.set("dict_type", next.dict_type);
    if (next.keyword) params.set("keyword", next.keyword);
    setSearchParams(params, { replace: true });
  }

  async function loadData(next = query) {
    setLoading(true);
    try {
      const result = await getDictEntries({
        ...next,
        category: next.category !== "all" ? next.category : undefined,
      });
      setList(result.list ?? []);
      setTotal(result.total);
    } finally {
      setLoading(false);
    }
  }

  function changeCategory(category: DictCategoryId) {
    setQuery((prev) => {
      const nextDictType =
        prev.dict_type && !matchesDictCategory(prev.dict_type, category) ? "" : prev.dict_type;
      const next = { ...prev, category, dict_type: nextDictType, page: 1 };
      syncSearchParams(next);
      return next;
    });
  }

  function openCreate() {
    setCurrent(null);
    form.resetFields();
    form.setFieldsValue({ status: 1, sort: 0 });
    setOpen(true);
  }

  function openEdit(record: DictEntryItem) {
    setCurrent(record);
    form.setFieldsValue({
      dict_type: record.dict_type,
      label: record.label,
      value: record.value,
      sort: record.sort,
      status: record.status,
      remark: record.remark,
    });
    setOpen(true);
  }

  async function submit() {
    const values = await form.validateFields();
    setSubmitting(true);
    try {
      if (current) {
        await updateDictEntry(current.id, values);
        message.success("字典条目已更新");
      } else {
        await createDictEntry(values);
        message.success("字典条目创建成功");
      }
      setOpen(false);
      await loadData();
    } finally {
      setSubmitting(false);
    }
  }

  async function remove(record: DictEntryItem) {
    await deleteDictEntry(record.id);
    message.success(`已删除条目 ${record.label}`);
    await loadData();
  }

  return (
    <Card className="table-card">
      <Space direction="vertical" size="middle" style={{ width: "100%", marginBottom: 16 }}>
        <Segmented
          value={query.category}
          options={DICT_CATEGORY_TABS.map((tab) => ({ label: tab.label, value: tab.id }))}
          onChange={(value) => changeCategory(value as DictCategoryId)}
        />
        {activeCategoryMeta ? (
          <Typography.Text type="secondary">{activeCategoryMeta.description}</Typography.Text>
        ) : null}
      </Space>

      <div className="toolbar">
        <Space wrap>
          <Input.Search
            allowClear
            defaultValue={query.keyword}
            placeholder="搜索标签/值/备注"
            style={{ width: 260 }}
            onSearch={(keyword) => {
              setQuery((prev) => {
                const next = { ...prev, keyword, page: 1 };
                syncSearchParams(next);
                return next;
              });
            }}
          />
          <Select
            allowClear
            showSearch
            placeholder="按字典类型筛选"
            style={{ width: 280 }}
            value={query.dict_type || undefined}
            options={groupedDictTypeOptions}
            filterOption={(input, option) => String(option?.label ?? "").toLowerCase().includes(input.toLowerCase())}
            onChange={(v) => {
              setQuery((prev) => {
                const next = { ...prev, dict_type: String(v || ""), page: 1 };
                syncSearchParams(next);
                return next;
              });
            }}
          />
        </Space>
        <div className="toolbar__actions">
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            新建条目
          </Button>
          <Button icon={<ReloadOutlined />} onClick={() => void loadData()}>
            刷新
          </Button>
        </div>
      </div>

      <Table
        rowKey="id"
        loading={loading}
        dataSource={list}
        pagination={{
          current: query.page,
          pageSize: query.page_size,
          total,
          showSizeChanger: true,
          pageSizeOptions: [10, 20, 50, 100],
          showQuickJumper: true,
          onChange: (page, pageSize) => setQuery((prev) => ({ ...prev, page, page_size: pageSize })),
        }}
        scroll={{ x: 1200 }}
        columns={[
          { title: "ID", dataIndex: "id", width: 80 },
          {
            title: "分类",
            key: "category",
            width: 120,
            render: (_: unknown, record: DictEntryItem) => {
              const category = resolveDictCategory(record.dict_type);
              const meta = DICT_CATEGORY_META[category];
              return <Tag color={meta.color}>{meta.label}</Tag>;
            },
          },
          { title: "字典类型", dataIndex: "dict_type", width: 220, render: (v: string) => <Tag color="geekblue">{v}</Tag> },
          { title: "标签", dataIndex: "label", width: 200, ellipsis: true },
          {
            title: "值",
            dataIndex: "value",
            width: 280,
            render: (v: string) => (
              <Tooltip placement="topLeft" title={v}>
                <Typography.Text ellipsis style={{ maxWidth: 260, display: "block" }}>
                  {v || "—"}
                </Typography.Text>
              </Tooltip>
            ),
          },
          { title: "排序", dataIndex: "sort", width: 80 },
          { title: "状态", dataIndex: "status", width: 90, render: (v: number) => (v === 1 ? <Tag color="success">启用</Tag> : <Tag>停用</Tag>) },
          { title: "备注", dataIndex: "remark", render: (v?: string) => v || "-" },
          { title: "更新时间", dataIndex: "updated_at", width: 180, render: formatDateTime },
          {
            title: "操作",
            key: "action",
            width: 180,
            render: (_: unknown, record: DictEntryItem) => (
              <Space>
                <Button type="link" icon={<EditOutlined />} onClick={() => openEdit(record)}>
                  编辑
                </Button>
                <Popconfirm title="确认删除该条目吗？" onConfirm={() => void remove(record)}>
                  <Button type="link" danger icon={<DeleteOutlined />}>
                    删除
                  </Button>
                </Popconfirm>
              </Space>
            ),
          },
        ]}
      />

      <Modal
        title={current ? `编辑字典条目 #${current.id}` : "新建字典条目"}
        open={open}
        onCancel={() => setOpen(false)}
        onOk={() => void submit()}
        confirmLoading={submitting}
        destroyOnClose
      >
        <Form form={form} layout="vertical" initialValues={{ status: 1, sort: 0 }}>
          <Form.Item
            label="字典类型"
            name="dict_type"
            rules={[{ required: true, message: "请选择字典类型" }]}
            extra={
              <>
                <Typography.Paragraph type="secondary" style={{ marginBottom: 0, fontSize: 12 }}>
                  字典类型按分类统一管理；新增配置时请优先从分组下拉选择，避免同义多名导致代码读取不到。
                </Typography.Paragraph>
                {formDictType ? (
                  <Typography.Paragraph type="secondary" style={{ marginBottom: 0, fontSize: 12 }}>
                    当前分类：{getDictCategoryLabel(resolveDictCategory(formDictType))}
                  </Typography.Paragraph>
                ) : null}
                {isAlertLabelGovernedDictType ? (
                  <Typography.Paragraph type="warning" style={{ marginBottom: 0, fontSize: 12 }}>
                    告警标签键已统一以 `alert_promql_label_key` 为唯一来源，策略匹配与静默匹配都读取该类型。
                  </Typography.Paragraph>
                ) : null}
              </>
            }
          >
            <Select
              showSearch
              placeholder="请选择字典类型"
              options={buildGroupedDictTypeSelectOptions("all", extraDictTypes)}
              filterOption={(input, option) => String(option?.label ?? "").toLowerCase().includes(input.toLowerCase())}
            />
          </Form.Item>
          <Form.Item label="标签" name="label" rules={[{ required: true, message: "请输入标签" }]}>
            <Input placeholder="例如 启用" />
          </Form.Item>
          <Form.Item
            label="值"
            name="value"
            rules={[{ required: true, message: "请输入值" }]}
            extra="支持完整 kubeconfig（含证书）；库字段为 MEDIUMTEXT，服务端上限约 16MB。极长内容建议仍用集群管理直接粘贴。"
          >
            <Input.TextArea rows={4} placeholder="例如 1 / GET / 多行 yaml" style={{ fontFamily: "ui-monospace, Menlo, Consolas, monospace" }} />
          </Form.Item>
          <Space style={{ width: "100%" }} size="middle">
            <Form.Item
              label="排序"
              name="sort"
              style={{ width: 140, marginBottom: 0 }}
              extra={autoSortLocked ? "该类型自动按当前最大序号+1分配" : undefined}
            >
              <InputNumber min={0} style={{ width: "100%" }} disabled={autoSortLocked} />
            </Form.Item>
            <Form.Item label="状态" name="status" style={{ width: 160, marginBottom: 0 }}>
              <Select options={[{ label: "启用", value: 1 }, { label: "停用", value: 0 }]} />
            </Form.Item>
          </Space>
          <Form.Item label="备注" name="remark" style={{ marginTop: 12 }}>
            <Input.TextArea rows={3} />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
