// @ts-nocheck
import { PageContainer } from '@ant-design/pro-components';
import { Alert, Typography } from 'antd';
import { useLocation } from '@umijs/max';

type Props = {
  menuName?: string;
  componentField?: string;
  path?: string;
};

/** 尚未从 web/ 迁移的菜单页占位 */
export default function MigrationPlaceholder({ menuName, componentField, path }: Props) {
  const location = useLocation();
  const displayPath = path || location.pathname;

  return (
    <PageContainer title={menuName || '功能迁移中'}>
      <Alert
        type="info"
        showIcon
        message="该页面正在迁移至 Ant Design Pro 新版前端（web-pro）"
        description={
          <Typography.Paragraph style={{ marginBottom: 0 }}>
            当前路径：<Typography.Text code>{displayPath}</Typography.Text>
            {componentField ? (
              <>
                <br />
                菜单组件：<Typography.Text code>{componentField}</Typography.Text>
              </>
            ) : null}
            <br />
            迁移进度见仓库文档 <Typography.Text code>docs/web-pro-migration.md</Typography.Text>。
            开发阶段可继续使用旧版 <Typography.Text code>web/</Typography.Text> 访问完整功能。
          </Typography.Paragraph>
        }
      />
    </PageContainer>
  );
}
