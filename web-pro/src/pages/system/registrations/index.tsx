import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  FormOutlined,
} from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { PageContainer, ProTable } from '@ant-design/pro-components';
import { Button, Form, Input, Modal, Radio, Tag, Typography, message } from 'antd';
import { useRef, useState } from 'react';
import {
  getRegistrations,
  reviewRegistration,
  type RegistrationRequestItem,
} from '@/services/registrations';

export default function RegistrationsPage() {
  const actionRef = useRef<ActionType>(undefined);
  const [reviewOpen, setReviewOpen] = useState(false);
  const [reviewTarget, setReviewTarget] = useState<RegistrationRequestItem | null>(null);
  const [reviewForm] = Form.useForm();
  const [submitting, setSubmitting] = useState(false);

  function statusTag(status: number) {
    switch (status) {
      case 0:
        return <Tag color="warning">待审核</Tag>;
      case 1:
        return <Tag color="success">已通过</Tag>;
      case 2:
        return <Tag color="error">已拒绝</Tag>;
      default:
        return <Tag>未知</Tag>;
    }
  }

  function openReview(record: RegistrationRequestItem) {
    setReviewTarget(record);
    reviewForm.resetFields();
    setReviewOpen(true);
  }

  async function handleReview() {
    if (!reviewTarget) return;
    const values = await reviewForm.validateFields();
    setSubmitting(true);
    try {
      await reviewRegistration(reviewTarget.id, values);
      message.success('审核完成');
      setReviewOpen(false);
      actionRef.current?.reload();
    } finally {
      setSubmitting(false);
    }
  }

  const columns: ProColumns<RegistrationRequestItem>[] = [
    {
      title: '状态',
      dataIndex: 'status',
      hideInTable: true,
      valueType: 'select',
      valueEnum: {
        0: { text: '待审核', status: 'Warning' },
        1: { text: '已通过', status: 'Success' },
        2: { text: '已拒绝', status: 'Error' },
      },
    },
    {
      title: '关键词',
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: { placeholder: '搜索用户名/邮箱' },
    },
    { title: 'ID', dataIndex: 'id', width: 70, search: false },
    { title: '用户名', dataIndex: 'username', search: false },
    { title: '邮箱', dataIndex: 'email', search: false },
    { title: '昵称', dataIndex: 'nickname', search: false },
    {
      title: '状态',
      dataIndex: 'status',
      search: false,
      render: (_, r) => statusTag(r.status),
    },
    { title: '申请时间', dataIndex: 'created_at', search: false },
    {
      title: '审核人',
      dataIndex: 'reviewer_username',
      search: false,
      render: (_, r) => r.reviewer_username || '-',
    },
    {
      title: '审核备注',
      dataIndex: 'review_comment',
      search: false,
      render: (_, r) => r.review_comment || '-',
    },
    {
      title: '操作',
      valueType: 'option',
      width: 120,
      render: (_, record) =>
        record.status === 0
          ? [
              <Button
                key="review"
                type="link"
                icon={<FormOutlined />}
                onClick={() => openReview(record)}
              >
                审核
              </Button>,
            ]
          : [<span key="done" style={{ color: '#999' }}>已完成</span>],
    },
  ];

  return (
    <PageContainer header={{ title: '注册审核', subTitle: '自助注册申请审批' }}>
      <ProTable<RegistrationRequestItem>
        rowKey="id"
        actionRef={actionRef}
        columns={columns}
        search={{ labelWidth: 'auto' }}
        request={async (params) => {
          const result = await getRegistrations({
            keyword: params.keyword as string | undefined,
            status: params.status as number | undefined,
            page: params.current,
            page_size: params.pageSize,
          });
          return {
            data: result.list,
            success: true,
            total: result.total,
          };
        }}
        pagination={{ defaultPageSize: 10, showSizeChanger: true }}
      />

      <Modal
        title={`审核注册申请 #${reviewTarget?.id}`}
        open={reviewOpen}
        onCancel={() => setReviewOpen(false)}
        onOk={() => void handleReview()}
        confirmLoading={submitting}
        destroyOnClose
        width={500}
      >
        <Form form={reviewForm} layout="vertical" initialValues={{ status: 1 }}>
          <Typography.Text strong>申请人信息</Typography.Text>
          <div style={{ marginBottom: 16, color: '#666' }}>
            用户名：{reviewTarget?.username} / 邮箱：{reviewTarget?.email} / 昵称：
            {reviewTarget?.nickname}
          </div>
          <Form.Item name="status" label="审核结果" rules={[{ required: true }]}>
            <Radio.Group>
              <Radio.Button value={1}>
                <CheckCircleOutlined /> 通过
              </Radio.Button>
              <Radio.Button value={2}>
                <CloseCircleOutlined /> 拒绝
              </Radio.Button>
            </Radio.Group>
          </Form.Item>
          <Form.Item name="comment" label="审核备注">
            <Input.TextArea rows={3} placeholder="请输入审核备注（选填）" />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  );
}
