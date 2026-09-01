import { LockOutlined, UserOutlined } from '@ant-design/icons';
import { LoginForm, ProFormText } from '@ant-design/pro-components';
import { Helmet, history, useModel } from '@umijs/max';
import { App, Button, Form, Space } from 'antd';
import React, { useCallback, useState } from 'react';
import { BRAND_NAME, BRAND_PRIMARY, BRAND_SUBTITLE } from '@/constants/brand';
import { getCurrentUser, passwordLogin, sendPasswordLoginCode } from '@/services/yunshu/auth';

const getSafeRedirectUrl = (redirect: string | null): string => {
  if (!redirect?.startsWith('/')) return '/welcome';
  if (redirect.startsWith('//')) return '/welcome';
  try {
    const parsed = new URL(redirect, window.location.origin);
    if (parsed.origin !== window.location.origin) return '/welcome';
    return `${parsed.pathname}${parsed.search}${parsed.hash}`;
  } catch {
    return '/welcome';
  }
};

export default function LoginPage() {
  const { message } = App.useApp();
  const { setInitialState } = useModel('@@initialState');
  const [form] = Form.useForm();
  const [captchaKey, setCaptchaKey] = useState('');
  const [captchaImage, setCaptchaImage] = useState<string | null>(null);
  const [loadingCaptcha, setLoadingCaptcha] = useState(false);

  const refreshCaptcha = useCallback(async (username?: string) => {
    if (!username?.trim()) {
      message.warning('请先输入用户名');
      return;
    }
    setLoadingCaptcha(true);
    try {
      const res = await sendPasswordLoginCode(username.trim());
      setCaptchaKey(res.captcha_key);
      setCaptchaImage(res.image);
    } catch (e: any) {
      message.error(e?.message || '获取验证码失败');
    } finally {
      setLoadingCaptcha(false);
    }
  }, [message]);

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: 'linear-gradient(145deg, #f8fafc 0%, #eef2f7 100%)',
        padding: 24,
      }}
    >
      <Helmet>
        <title>登录 - {BRAND_NAME}</title>
      </Helmet>
      <div style={{ width: '100%', maxWidth: 420 }}>
        <div style={{ textAlign: 'center', marginBottom: 24 }}>
          <div
            style={{
              width: 56,
              height: 56,
              margin: '0 auto 12px',
              borderRadius: 12,
              background: BRAND_PRIMARY,
              color: '#fff',
              display: 'grid',
              placeItems: 'center',
              fontWeight: 700,
              fontSize: 18,
            }}
          >
            YS
          </div>
          <h1 style={{ margin: 0, fontSize: 22, fontWeight: 600 }}>{BRAND_NAME}</h1>
          <p style={{ margin: '8px 0 0', color: '#64748b' }}>{BRAND_SUBTITLE}</p>
        </div>
        <LoginForm
          form={form}
          title="登录"
          subTitle="使用平台账号登录"
          onFinish={async (values) => {
            if (!captchaKey) {
              message.warning('请先获取验证码');
              return false;
            }
            try {
              await passwordLogin({
                username: values.username,
                password: values.password,
                captcha_key: captchaKey,
                code: values.captcha,
              });
              const user = await getCurrentUser();
              await setInitialState((s) => ({
                ...s,
                currentUser: {
                  name: user.nickname || user.username,
                  userid: String(user.id),
                  access: user.roles?.some((r: YunshuAPI.RoleItem) => r.code === 'super-admin') ? 'admin' : 'user',
                  ...user,
                },
              }));
              message.success('登录成功');
              const params = new URLSearchParams(window.location.search);
              const redirect = getSafeRedirectUrl(params.get('redirect'));
              history.replace(redirect);
              return true;
            } catch (e: any) {
              message.error(e?.message || '登录失败');
              void refreshCaptcha(values.username);
              return false;
            }
          }}
        >
          <ProFormText
            name="username"
            fieldProps={{ size: 'large', prefix: <UserOutlined /> }}
            placeholder="用户名"
            rules={[{ required: true, message: '请输入用户名' }]}
          />
          <ProFormText.Password
            name="password"
            fieldProps={{ size: 'large', prefix: <LockOutlined /> }}
            placeholder="密码"
            rules={[{ required: true, message: '请输入密码' }]}
          />
          <Space.Compact style={{ width: '100%' }}>
            <ProFormText
              name="captcha"
              fieldProps={{ size: 'large' }}
              placeholder="验证码"
              rules={[{ required: true, message: '请输入验证码' }]}
              style={{ flex: 1 }}
            />
            <Button
              size="large"
              loading={loadingCaptcha}
              onClick={() => refreshCaptcha(form.getFieldValue('username'))}
            >
              获取验证码
            </Button>
          </Space.Compact>
          {captchaImage ? (
            <img
              src={captchaImage}
              alt="captcha"
              style={{ height: 40, cursor: 'pointer', marginTop: 8 }}
              onClick={() => refreshCaptcha(form.getFieldValue('username'))}
            />
          ) : null}
        </LoginForm>
      </div>
    </div>
  );
}
