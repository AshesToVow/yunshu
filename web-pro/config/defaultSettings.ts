import type { ProLayoutProps } from '@ant-design/pro-components';

const BRAND_NAME = '云枢运维平台';
const BRAND_PRIMARY = '#0d9488';

const Settings: ProLayoutProps & { logo?: string } = {
  navTheme: 'light',
  colorPrimary: BRAND_PRIMARY,
  layout: 'mix',
  contentWidth: 'Fluid',
  fixedHeader: true,
  fixSiderbar: true,
  colorWeak: false,
  title: BRAND_NAME,
  logo: undefined,
  iconfontUrl: '',
};

export default Settings;
