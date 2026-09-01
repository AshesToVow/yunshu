import {
  LockOutlined,
  MailOutlined,
  MoonOutlined,
  SunOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { Helmet, history, useModel } from '@umijs/max';
import {
  App,
  Button,
  Form,
  Input,
  Modal,
  Space,
  Typography,
} from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import loginHero from '@/assets/login-hero.svg';
import { BRAND_NAME, BRAND_PRIMARY, BRAND_SUBTITLE } from '@/constants/brand';
import {
  emailLogin,
  getCurrentUser,
  passwordLogin,
  registerByEmail,
  sendEmailCode,
  sendPasswordLoginCode,
  toCaptchaDataUrl,
} from '@/services/yunshu/auth';

type AuthTab = 'account' | 'email';
type Accent = 'emerald' | 'blue' | 'violet' | 'amber';

const ACCENTS: { key: Accent; color: string }[] = [
  { key: 'emerald', color: '#0d9488' },
  { key: 'blue', color: '#2563eb' },
  { key: 'violet', color: '#7c3aed' },
  { key: 'amber', color: '#d97706' },
];

const INTRO_POINTS = ['Kubernetes 资源编排', 'CI/CD 发布流水线', 'CMDB 资产治理', '告警与值班联动'];

const getSafeRedirectUrl = (redirect: string | null): string => {
  if (!redirect?.startsWith('/')) return '/dashboard';
  if (redirect.startsWith('//')) return '/dashboard';
  try {
    const parsed = new URL(redirect, window.location.origin);
    if (parsed.origin !== window.location.origin) return '/dashboard';
    return `${parsed.pathname}${parsed.search}${parsed.hash}` || '/dashboard';
  } catch {
    return '/dashboard';
  }
};

function mapUser(user: YunshuAPI.UserItem): API.CurrentUser {
  return {
    name: user.nickname || user.username,
    userid: String(user.id),
    access: user.roles?.some((r: YunshuAPI.RoleItem) => r.code === 'super-admin') ? 'admin' : 'user',
    ...user,
  };
}

function errMessage(e: unknown, fallback: string) {
  if (e && typeof e === 'object' && 'message' in e && typeof (e as any).message === 'string') {
    return (e as any).message || fallback;
  }
  return fallback;
}

export default function LoginPage() {
  const { message } = App.useApp();
  const { setInitialState } = useModel('@@initialState');
  const [accountForm] = Form.useForm();
  const [emailForm] = Form.useForm();
  const [registerForm] = Form.useForm();

  const [tab, setTab] = useState<AuthTab>('account');
  const [theme, setTheme] = useState<'light' | 'dark'>('light');
  const [accent, setAccent] = useState<Accent>('emerald');
  const [submitting, setSubmitting] = useState(false);

  const [captchaKey, setCaptchaKey] = useState('');
  const [captchaImage, setCaptchaImage] = useState<string | null>(null);
  const [loadingCaptcha, setLoadingCaptcha] = useState(false);
  const [captchaCooldown, setCaptchaCooldown] = useState(0);

  const [emailCooldown, setEmailCooldown] = useState(0);
  const [registerOpen, setRegisterOpen] = useState(false);
  const [registerCooldown, setRegisterCooldown] = useState(0);
  const [registerSubmitting, setRegisterSubmitting] = useState(false);

  useEffect(() => {
    if (captchaCooldown <= 0) return;
    const t = window.setTimeout(() => setCaptchaCooldown((s) => s - 1), 1000);
    return () => window.clearTimeout(t);
  }, [captchaCooldown]);

  useEffect(() => {
    if (emailCooldown <= 0) return;
    const t = window.setTimeout(() => setEmailCooldown((s) => s - 1), 1000);
    return () => window.clearTimeout(t);
  }, [emailCooldown]);

  useEffect(() => {
    if (registerCooldown <= 0) return;
    const t = window.setTimeout(() => setRegisterCooldown((s) => s - 1), 1000);
    return () => window.clearTimeout(t);
  }, [registerCooldown]);

  const shellClass = useMemo(
    () => `gw-auth-shell is-${theme} gw-accent-${accent}`,
    [theme, accent],
  );

  const afterLogin = useCallback(
    async (user: YunshuAPI.UserItem) => {
      await setInitialState((s) => ({
        ...s,
        currentUser: mapUser(user),
      }));
      message.success('登录成功');
      const params = new URLSearchParams(window.location.search);
      history.replace(getSafeRedirectUrl(params.get('redirect')));
    },
    [message, setInitialState],
  );

  const refreshCaptcha = useCallback(
    async (username?: string, opts?: { silent?: boolean }) => {
      const name = (username ?? accountForm.getFieldValue('username') ?? '').trim();
      if (!name) {
        if (!opts?.silent) message.warning('请先输入用户名');
        return;
      }
      if (captchaCooldown > 0 && !opts?.silent) {
        message.warning(`验证码冷却中，请 ${captchaCooldown} 秒后再试`);
        return;
      }
      setLoadingCaptcha(true);
      try {
        const res = await sendPasswordLoginCode(name);
        setCaptchaKey(res.captcha_key);
        setCaptchaImage(toCaptchaDataUrl(res.image));
        if (res.cooldown_in > 0) setCaptchaCooldown(res.cooldown_in);
      } catch (e) {
        const msg = errMessage(e, '获取验证码失败');
        if (!opts?.silent) message.error(msg);
        // 冷却中保留已有图
        const m = /(\d+)\s*秒/.exec(msg);
        if (m) setCaptchaCooldown(Number(m[1]));
      } finally {
        setLoadingCaptcha(false);
      }
    },
    [accountForm, captchaCooldown, message],
  );

  const onAccountFinish = async (values: {
    username: string;
    password: string;
    captcha: string;
  }) => {
    if (!captchaKey) {
      message.warning('请先获取图形验证码');
      await refreshCaptcha(values.username);
      return;
    }
    setSubmitting(true);
    try {
      await passwordLogin({
        username: values.username.trim(),
        password: values.password,
        captcha_key: captchaKey,
        code: values.captcha.trim(),
      });
      const user = await getCurrentUser();
      await afterLogin(user);
    } catch (e) {
      message.error(errMessage(e, '登录失败'));
      accountForm.setFieldValue('captcha', undefined);
      await refreshCaptcha(values.username, { silent: true });
    } finally {
      setSubmitting(false);
    }
  };

  const sendLoginEmailCode = async () => {
    const email = (emailForm.getFieldValue('email') || '').trim();
    if (!email) {
      message.warning('请先输入邮箱');
      return;
    }
    if (emailCooldown > 0) return;
    try {
      const res = await sendEmailCode({ email, scene: 'login' });
      message.success('验证码已发送到邮箱');
      setEmailCooldown(res.cooldown_in || 60);
    } catch (e) {
      message.error(errMessage(e, '发送验证码失败'));
    }
  };

  const onEmailFinish = async (values: { email: string; code: string }) => {
    setSubmitting(true);
    try {
      await emailLogin({
        email: values.email.trim(),
        code: values.code.trim(),
      });
      const user = await getCurrentUser();
      await afterLogin(user);
    } catch (e) {
      message.error(errMessage(e, '登录失败'));
    } finally {
      setSubmitting(false);
    }
  };

  const sendRegisterEmailCode = async () => {
    const email = (registerForm.getFieldValue('email') || '').trim();
    if (!email) {
      message.warning('请先输入邮箱');
      return;
    }
    if (registerCooldown > 0) return;
    try {
      const res = await sendEmailCode({ email, scene: 'register' });
      message.success('验证码已发送到邮箱');
      setRegisterCooldown(res.cooldown_in || 60);
    } catch (e) {
      message.error(errMessage(e, '发送验证码失败'));
    }
  };

  const onRegisterFinish = async (values: {
    username: string;
    email: string;
    nickname: string;
    password: string;
    code: string;
  }) => {
    setRegisterSubmitting(true);
    try {
      const res = await registerByEmail({
        username: values.username.trim(),
        email: values.email.trim(),
        nickname: values.nickname.trim(),
        password: values.password,
        code: values.code.trim(),
      });
      message.success(res.message || '注册申请已提交，请等待管理员审核');
      setRegisterOpen(false);
      registerForm.resetFields();
    } catch (e) {
      message.error(errMessage(e, '注册失败'));
    } finally {
      setRegisterSubmitting(false);
    }
  };

  return (
    <div className={shellClass} style={{ ['--login-accent' as string]: BRAND_PRIMARY }}>
      <Helmet>
        <title>登录 - {BRAND_NAME}</title>
      </Helmet>

      <div className="gw-auth-shell__ambient" aria-hidden>
        <div className="gw-auth-shell__orb gw-auth-shell__orb--1" />
        <div className="gw-auth-shell__orb gw-auth-shell__orb--2" />
        <div className="gw-auth-shell__orb gw-auth-shell__orb--3" />
      </div>

      <div className="gw-auth-brand">
        <span className="gw-auth-brand__logoDot" />
        {BRAND_NAME}
      </div>

      <Space className="gw-auth-toolbar" size={4}>
        <Button
          type="text"
          className="gw-auth-toolbar__btn"
          icon={theme === 'light' ? <MoonOutlined /> : <SunOutlined />}
          onClick={() => setTheme((t) => (t === 'light' ? 'dark' : 'light'))}
          aria-label="切换主题"
        />
        {ACCENTS.map((a) => (
          <button
            key={a.key}
            type="button"
            className={`gw-auth-dot${accent === a.key ? ' is-active' : ''}`}
            style={{ background: a.color }}
            aria-label={`主题色 ${a.key}`}
            onClick={() => setAccent(a.key)}
          />
        ))}
      </Space>

      <main className="gw-auth-main">
        <div className="gw-auth-frame">
          <section className="gw-auth-story">
            <img className="gw-auth-story__hero" src={loginHero} alt="" />
            <h1 className="gw-auth-story__title">{BRAND_NAME}</h1>
            <p className="gw-auth-story__desc">
              {BRAND_SUBTITLE}。统一编排集群、发布、资产与告警，服务值班与交付现场。
            </p>
            <ul className="gw-auth-story__points">
              {INTRO_POINTS.map((p) => (
                <li key={p}>{p}</li>
              ))}
            </ul>
          </section>

          <section className="gw-auth-panel">
            <div className="gw-auth-card" role="region" aria-label="登录面板">
              <div className="gw-auth-card__header">
                <div className="gw-auth-card__title">欢迎回来</div>
                <div className="gw-auth-card__sub">登录云枢运维平台，继续你的值班与发布工作</div>
              </div>

              <div className="gw-auth-tabs" role="tablist" aria-label="登录方式">
                <button
                  type="button"
                  role="tab"
                  aria-selected={tab === 'account'}
                  className={`gw-auth-tabs__item${tab === 'account' ? ' is-active' : ''}`}
                  onClick={() => setTab('account')}
                >
                  账号密码
                </button>
                <button
                  type="button"
                  role="tab"
                  aria-selected={tab === 'email'}
                  className={`gw-auth-tabs__item${tab === 'email' ? ' is-active' : ''}`}
                  onClick={() => setTab('email')}
                >
                  邮箱登录
                </button>
              </div>

              {tab === 'account' ? (
                <>
                  <p className="login-light-card__hint">输入用户名后点击验证码图片刷新；验证码为 4 位数字。</p>
                  <Form form={accountForm} layout="vertical" onFinish={onAccountFinish} requiredMark={false}>
                    <Form.Item
                      name="username"
                      rules={[{ required: true, message: '请输入用户名' }]}
                    >
                      <Input
                        size="large"
                        prefix={<UserOutlined />}
                        placeholder="用户名"
                        autoComplete="username"
                        onBlur={() => {
                          const u = accountForm.getFieldValue('username');
                          if (u?.trim() && !captchaImage) void refreshCaptcha(u, { silent: true });
                        }}
                      />
                    </Form.Item>
                    <Form.Item
                      name="password"
                      rules={[{ required: true, message: '请输入密码' }]}
                    >
                      <Input.Password
                        size="large"
                        prefix={<LockOutlined />}
                        placeholder="密码"
                        autoComplete="current-password"
                      />
                    </Form.Item>
                    <Form.Item label="图形验证码" required style={{ marginBottom: 8 }}>
                      <Space.Compact style={{ width: '100%' }}>
                        <Form.Item
                          name="captcha"
                          noStyle
                          rules={[
                            { required: true, message: '请输入验证码' },
                            { len: 4, message: '验证码为 4 位数字' },
                          ]}
                        >
                          <Input size="large" placeholder="4 位验证码" maxLength={4} style={{ flex: 1 }} />
                        </Form.Item>
                        <Button
                          size="large"
                          loading={loadingCaptcha}
                          disabled={captchaCooldown > 0}
                          onClick={() => void refreshCaptcha()}
                          style={{ width: 120 }}
                        >
                          {captchaCooldown > 0 ? `${captchaCooldown}s` : '获取验证码'}
                        </Button>
                      </Space.Compact>
                    </Form.Item>
                    {captchaImage ? (
                      <img
                        src={captchaImage}
                        alt="captcha"
                        title="点击刷新"
                        style={{
                          height: 48,
                          marginBottom: 16,
                          borderRadius: 6,
                          cursor: loadingCaptcha || captchaCooldown > 0 ? 'default' : 'pointer',
                          opacity: loadingCaptcha ? 0.6 : 1,
                        }}
                        onClick={() => {
                          if (!loadingCaptcha && captchaCooldown <= 0) void refreshCaptcha();
                        }}
                      />
                    ) : (
                      <Typography.Text type="secondary" style={{ display: 'block', marginBottom: 16 }}>
                        点击「获取验证码」显示图形码
                      </Typography.Text>
                    )}
                    <Button
                      className="login-submitBtn"
                      type="primary"
                      htmlType="submit"
                      block
                      size="large"
                      loading={submitting}
                    >
                      登录
                    </Button>
                  </Form>
                </>
              ) : (
                <>
                  <p className="login-light-card__hint">向已绑定邮箱发送 6 位验证码，无需密码即可登录。</p>
                  <Form form={emailForm} layout="vertical" onFinish={onEmailFinish} requiredMark={false}>
                    <Form.Item
                      name="email"
                      rules={[
                        { required: true, message: '请输入邮箱' },
                        { type: 'email', message: '邮箱格式不正确' },
                      ]}
                    >
                      <Input size="large" prefix={<MailOutlined />} placeholder="邮箱" autoComplete="email" />
                    </Form.Item>
                    <Form.Item label="邮箱验证码" required>
                      <Space.Compact style={{ width: '100%' }}>
                        <Form.Item
                          name="code"
                          noStyle
                          rules={[
                            { required: true, message: '请输入验证码' },
                            { len: 6, message: '验证码为 6 位数字' },
                          ]}
                        >
                          <Input size="large" placeholder="6 位验证码" maxLength={6} style={{ flex: 1 }} />
                        </Form.Item>
                        <Button
                          size="large"
                          disabled={emailCooldown > 0}
                          onClick={() => void sendLoginEmailCode()}
                          style={{ width: 120 }}
                        >
                          {emailCooldown > 0 ? `${emailCooldown}s` : '发送验证码'}
                        </Button>
                      </Space.Compact>
                    </Form.Item>
                    <Button
                      className="login-submitBtn"
                      type="primary"
                      htmlType="submit"
                      block
                      size="large"
                      loading={submitting}
                    >
                      登录
                    </Button>
                  </Form>
                </>
              )}

              <div style={{ marginTop: 16, textAlign: 'center' }}>
                <Typography.Link
                  className="login-light-registerLink"
                  onClick={() => setRegisterOpen(true)}
                >
                  没有账号？注册申请
                </Typography.Link>
              </div>
            </div>
          </section>
        </div>
      </main>

      <Modal
        className="login-registerModal"
        title="用户注册申请"
        open={registerOpen}
        onCancel={() => setRegisterOpen(false)}
        footer={null}
        destroyOnHidden
        width={440}
      >
        <Typography.Paragraph type="secondary" style={{ marginTop: 0 }}>
          提交后需管理员在「注册审核」中批准方可登录。
        </Typography.Paragraph>
        <Form form={registerForm} layout="vertical" onFinish={onRegisterFinish} requiredMark={false}>
          <Form.Item
            name="username"
            label="用户名"
            rules={[
              { required: true, message: '请输入用户名' },
              { min: 3, message: '至少 3 个字符' },
            ]}
          >
            <Input prefix={<UserOutlined />} placeholder="登录用户名" autoComplete="off" />
          </Form.Item>
          <Form.Item
            name="nickname"
            label="昵称"
            rules={[{ required: true, message: '请输入昵称' }]}
          >
            <Input placeholder="显示名称" />
          </Form.Item>
          <Form.Item
            name="email"
            label="邮箱"
            rules={[
              { required: true, message: '请输入邮箱' },
              { type: 'email', message: '邮箱格式不正确' },
            ]}
          >
            <Input prefix={<MailOutlined />} placeholder="用于接收验证码" autoComplete="email" />
          </Form.Item>
          <Form.Item
            name="password"
            label="密码"
            rules={[
              { required: true, message: '请输入密码' },
              { min: 6, message: '至少 6 位' },
            ]}
          >
            <Input.Password prefix={<LockOutlined />} placeholder="登录密码" autoComplete="new-password" />
          </Form.Item>
          <Form.Item label="邮箱验证码" required>
            <Space.Compact style={{ width: '100%' }}>
              <Form.Item
                name="code"
                noStyle
                rules={[
                  { required: true, message: '请输入验证码' },
                  { len: 6, message: '验证码为 6 位数字' },
                ]}
              >
                <Input placeholder="6 位验证码" maxLength={6} style={{ flex: 1 }} />
              </Form.Item>
              <Button disabled={registerCooldown > 0} onClick={() => void sendRegisterEmailCode()}>
                {registerCooldown > 0 ? `${registerCooldown}s` : '发送验证码'}
              </Button>
            </Space.Compact>
          </Form.Item>
          <Button type="primary" htmlType="submit" block loading={registerSubmitting} className="login-submitBtn">
            提交注册申请
          </Button>
        </Form>
      </Modal>
    </div>
  );
}
