// @ts-nocheck
import { PageContainer } from '@ant-design/pro-components';
import { Alert, Typography } from 'antd';
import { BRAND_NAME } from '@/constants/brand';

export default function WelcomePage() {
  return (
    <PageContainer title="总览">
      <Alert
        type="success"
        showIcon
        message={`${BRAND_NAME} · Ant Design Pro 新版前端`}
        description={
          <Typography.Paragraph style={{ marginBottom: 0 }}>
            当前为 <Typography.Text code>web-pro</Typography.Text> 迁移阶段。侧栏菜单来自后端动态配置；
            已迁移页面可直接使用，未迁移页面会显示占位提示。完整功能在过渡期仍可通过{' '}
            <Typography.Text code>web/</Typography.Text>（Vite 旧版）访问。
          </Typography.Paragraph>
        }
      />
    </PageContainer>
  );
}
