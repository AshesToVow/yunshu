import {
  BgColorsOutlined,
  BulbFilled,
  BulbOutlined,
  TranslationOutlined,
  LockOutlined,
  MailOutlined,
  SafetyCertificateOutlined,
  UserOutlined,
} from "@ant-design/icons";
import type { InputRef } from "antd";
import { Button, Form, Input, Modal, message } from "antd";
import { useEffect, useRef, useState, type CSSProperties } from "react";
import { useTranslation } from "react-i18next";
import { useLocation, useNavigate } from "react-router-dom";
import { sendEmailCode, sendPasswordLoginCode, registerByEmail } from "../services/auth";
import { extractApiErrorMessage } from "../services/http";
import type {
  EmailLoginPayload,
  PasswordLoginPayload,
  RegisterPayload,
  SendEmailCodePayload,
  SendPasswordLoginCodeResult,
} from "../types/api";
import { useAuth } from "../contexts/auth-context";
import { resolveAppLocale } from "../i18n";
import { resolveEmailFromForm } from "../utils/form-email";
import loginHeroImage from "../assets/login-hero.svg";
import { useAdminThemeStore } from "../stores/admin-theme-store";

type LoginAccent = "blue" | "violet" | "emerald" | "amber";

const LOGIN_ACCENT_COLORS: Record<LoginAccent, string> = {
  blue: "#2563eb",
  violet: "#7c3aed",
  emerald: "#0d9488",
  amber: "#d97706",
};

function accentKeyFromHex(hex: string): LoginAccent {
  const hit = (Object.entries(LOGIN_ACCENT_COLORS) as [LoginAccent, string][]).find(([, v]) => v === hex);
  return hit?.[0] ?? "emerald";
}

type AuthTabKey = "account" | "email";
type ButtonFxState = "idle" | "loading" | "success";

interface LocationState {
  from?: string;
}

function useCountdown(seconds: number, onTick: (next: number) => void) {
  useEffect(() => {
    if (seconds <= 0) return;
    const t = window.setTimeout(() => onTick(seconds - 1), 1000);
    return () => window.clearTimeout(t);
  }, [seconds, onTick]);
}

export function LoginPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const { passwordLoginAction, emailLoginAction } = useAuth();

  const [tab, setTab] = useState<AuthTabKey>("account");
  const [registerOpen, setRegisterOpen] = useState(false);

  const [submitting, setSubmitting] = useState(false);
  const [sendingCode, setSendingCode] = useState(false);

  const [passwordCodeCountdown, setPasswordCodeCountdown] = useState(0);
  const [emailCodeCountdown, setEmailCodeCountdown] = useState(0);
  const [registerCodeCountdown, setRegisterCodeCountdown] = useState(0);

  const [captchaKey, setCaptchaKey] = useState("");
  const [captchaImage, setCaptchaImage] = useState<string | null>(null);

  const [buttonFx, setButtonFx] = useState<ButtonFxState>("idle");
  const themeMode = useAdminThemeStore((s) => s.mode);
  const setThemeMode = useAdminThemeStore((s) => s.setMode);
  const storeAccent = useAdminThemeStore((s) => s.accent);
  const setStoreAccent = useAdminThemeStore((s) => s.setAccent);
  const darkMode = themeMode !== "light";
  const { t, i18n } = useTranslation();
  const [paletteOpen, setPaletteOpen] = useState(false);
  const accent = accentKeyFromHex(storeAccent);
  const isZh = resolveAppLocale(i18n.language) === "zh-CN";

  const [passwordForm] = Form.useForm<PasswordLoginPayload>();
  const [emailForm] = Form.useForm<EmailLoginPayload>();
  const [registerForm] = Form.useForm<RegisterPayload>();
  const emailInputRef = useRef<InputRef>(null);
  const registerEmailInputRef = useRef<InputRef>(null);

  useCountdown(passwordCodeCountdown, setPasswordCodeCountdown);
  useCountdown(emailCodeCountdown, setEmailCodeCountdown);
  useCountdown(registerCodeCountdown, setRegisterCodeCountdown);

  useEffect(() => {
    document.documentElement.style.setProperty("--login-accent", LOGIN_ACCENT_COLORS[accent]);
  }, [accent]);

  const fromPath = (location.state as LocationState | null)?.from ?? "/";

  async function refreshPasswordCaptcha(options?: {
    silent?: boolean;
    requireUsername?: boolean;
    /** 登录失败后强制换新图：先清空旧图，避免冷却失败时仍展示已作废验证码 */
    afterLoginFailure?: boolean;
  }) {
    const silent = options?.silent === true;
    const requireUsername = options?.requireUsername !== false;
    const afterLoginFailure = options?.afterLoginFailure === true;
    const zh = isZh;
    try {
      const username = passwordForm.getFieldValue("username");
      if (!username) {
        if (requireUsername && !silent) {
          message.warning(zh ? "请先输入用户名" : "Please enter username first");
        }
        return false;
      }

      if (afterLoginFailure) {
        setCaptchaKey("");
        setCaptchaImage(null);
        passwordForm.setFieldsValue({ captcha_key: undefined, code: undefined });
      }

      setSendingCode(true);
      const result: SendPasswordLoginCodeResult = await sendPasswordLoginCode({ username });
      setCaptchaKey(result.captcha_key);
      setCaptchaImage(result.image);
      passwordForm.setFieldsValue({ captcha_key: result.captcha_key, code: undefined });
      if (!silent) {
        message.success(zh ? "验证码已生成" : "Captcha generated");
      }
      if (!afterLoginFailure) {
        setPasswordCodeCountdown(60);
      }
      return true;
    } catch (e) {
      if (!silent) {
        message.error(extractApiErrorMessage(e, zh ? "生成验证码失败" : "Failed to generate captcha"));
      }
      if (afterLoginFailure) {
        setCaptchaKey("");
        setCaptchaImage(null);
        passwordForm.setFieldsValue({ captcha_key: undefined, code: undefined });
      }
      return false;
    } finally {
      setSendingCode(false);
    }
  }

  async function handleSendPasswordCode() {
    await refreshPasswordCaptcha();
  }

  async function handleSendEmailCode() {
    try {
      const email = await resolveEmailFromForm(emailForm, emailInputRef);
      if (!email) {
        return;
      }

      setSendingCode(true);
      const payload: SendEmailCodePayload = { email, scene: "login" };
      await sendEmailCode(payload);
      message.success("验证码已发送到您的邮箱，请查收");
      setEmailCodeCountdown(60);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "发送验证码失败"));
    } finally {
      setSendingCode(false);
    }
  }

  async function handleSendRegisterCode() {
    try {
      const email = await resolveEmailFromForm(registerForm, registerEmailInputRef);
      if (!email) {
        return;
      }

      setSendingCode(true);
      const payload: SendEmailCodePayload = { email, scene: "register" };
      await sendEmailCode(payload);
      message.success("验证码已发送到您的邮箱，请查收");
      setRegisterCodeCountdown(60);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "发送验证码失败"));
    } finally {
      setSendingCode(false);
    }
  }

  async function runLogin<TPayload>(
    action: (payload: TPayload) => Promise<unknown>,
    payload: TPayload,
    onError?: () => void | Promise<void>,
  ) {
    setSubmitting(true);
    setButtonFx("loading");

    try {
      await action(payload);
      message.success("登录成功");
      setButtonFx("success");
      window.setTimeout(() => navigate(fromPath, { replace: true }), 520);
    } catch (e) {
      setButtonFx("idle");
      message.error(extractApiErrorMessage(e, "登录失败"));
      await onError?.();
    } finally {
      window.setTimeout(() => setButtonFx("idle"), 1200);
      setSubmitting(false);
    }
  }

  async function handlePasswordLogin(values: PasswordLoginPayload) {
    const payload: PasswordLoginPayload = {
      ...values,
      captcha_key: values.captcha_key || captchaKey,
    };
    void runLogin(passwordLoginAction as (p: PasswordLoginPayload) => Promise<unknown>, payload, async () => {
      await refreshPasswordCaptcha({ silent: true, requireUsername: false, afterLoginFailure: true });
    });
  }

  async function handleEmailLogin(values: EmailLoginPayload) {
    void runLogin(emailLoginAction as (p: EmailLoginPayload) => Promise<unknown>, values);
  }

  async function handleRegister(values: RegisterPayload) {
    setSubmitting(true);
    try {
      const payload: RegisterPayload = {
        ...values,
        code: String(values.code ?? "")
          .trim()
          .replace(/[^\d]/g, ""),
      };
      const result = await registerByEmail(payload);
      message.success(result?.message || "注册申请已提交，请等待管理员审核");
      setRegisterOpen(false);
      registerForm.resetFields();
    } catch (e) {
      message.error(extractApiErrorMessage(e, "注册失败"));
    } finally {
      setSubmitting(false);
    }
  }

  const submitButtonLabel = t("login.submit");
  const cardTitle = t("login.welcome");
  const cardSubTitle = t("login.welcomeSub");
  const appTitle = t("brand.name");
  const introTitle = t("login.introTitle");
  const introDesc = t("login.introDesc");
  const introFeatures = isZh
    ? ["Kubernetes 资源编排", "项目管理", "CI/CD 发布流水线", "日志平台", "CMDB 资产治理", "告警与值班联动"]
    : [
        "Kubernetes orchestration",
        "Project management",
        "CI/CD pipelines",
        "Log platform",
        "CMDB governance",
        "Alerts & on-call",
      ];

  function renderAuthTabs() {
    return (
      <div className="gw-auth-tabs" role="tablist" aria-label={t("login.method")}>
        <button
          type="button"
          className={`gw-auth-tabs__item ${tab === "account" ? "is-active" : ""}`}
          onClick={() => setTab("account")}
          role="tab"
          aria-selected={tab === "account"}
        >
          {t("login.accountLogin")}
        </button>
        <button
          type="button"
          className={`gw-auth-tabs__item ${tab === "email" ? "is-active" : ""}`}
          onClick={() => setTab("email")}
          role="tab"
          aria-selected={tab === "email"}
        >
          {t("login.emailLogin")}
        </button>
      </div>
    );
  }

  function renderFormCard() {
    return (
      <section className="gw-auth-card" role="region" aria-label={t("login.panel")}>
        <div className="gw-auth-card__header">
          <div className="gw-auth-card__title">{cardTitle}</div>
          <div className="gw-auth-card__sub">{cardSubTitle}</div>
        </div>

        {renderAuthTabs()}

        <div className="login-light-card__hint">
          {tab === "account" ? t("login.accountHint") : t("login.emailHint")}
        </div>

        {tab === "account" ? (
          <Form<PasswordLoginPayload> form={passwordForm} layout="vertical" onFinish={handlePasswordLogin} size="large">
            <Form.Item
              label={isZh ? "用户名" : "Username"}
              name="username"
              rules={[{ required: true, message: isZh ? "请输入用户名" : "Please enter username" }]}
            >
              <Input
                prefix={<UserOutlined />}
                placeholder={isZh ? "请输入用户名" : "Please enter username"}
                autoComplete="off"
              />
            </Form.Item>

            <Form.Item
              label={isZh ? "密码" : "Password"}
              name="password"
              rules={[{ required: true, message: isZh ? "请输入密码" : "Please enter password" }]}
            >
              <Input.Password
                prefix={<LockOutlined />}
                placeholder={isZh ? "请输入密码" : "Please enter password"}
                autoComplete="new-password"
              />
            </Form.Item>

            <Form.Item label={isZh ? "验证码" : "Code"}>
              <div className="login-captchaRow">
                <Form.Item
                  name="code"
                  rules={[
                    { required: true, message: isZh ? "请输入验证码" : "Please enter code" },
                    { len: 4, message: isZh ? "验证码为4位数字" : "4 digits required" },
                    { pattern: /^\d+$/, message: isZh ? "验证码必须为数字" : "Digits only" },
                  ]}
                  style={{ margin: 0, flex: 1 }}
                >
                  <Input prefix={<SafetyCertificateOutlined />} placeholder={isZh ? "验证码" : "Code"} maxLength={4} />
                </Form.Item>

                <div
                  className={`login-captchaWave ${sendingCode ? "is-loading" : ""}`}
                  onClick={() => void handleSendPasswordCode()}
                  role="button"
                  tabIndex={0}
                  aria-label={isZh ? "点击刷新验证码" : "Refresh captcha"}
                >
                  {captchaImage ? (
                    <img
                      className="login-captchaWave__img"
                      src={`data:image/png;base64,${captchaImage}`}
                      alt={isZh ? "验证码图片" : "Captcha image"}
                    />
                  ) : (
                    <span className="login-captchaWave__placeholder">{isZh ? "生成验证码" : "Generate"}</span>
                  )}
                </div>
              </div>
            </Form.Item>

            <Form.Item
              name="captcha_key"
              style={{ display: "none" }}
              rules={[{ required: true, message: isZh ? "请先生成验证码" : "Please generate code first" }]}
            >
              <Input type="hidden" />
            </Form.Item>

            <div className="login-submitRow">
              <Button
                htmlType="submit"
                className="login-submitBtn"
                disabled={submitting || !captchaKey}
                data-fx={buttonFx}
              >
                <span className="login-submitBtn__label">{submitButtonLabel}</span>
                <span className="login-submitBtn__spinner" aria-hidden="true" />
              </Button>
            </div>
          </Form>
        ) : (
          <Form<EmailLoginPayload> form={emailForm} layout="vertical" onFinish={handleEmailLogin} size="large">
            <Form.Item
              label={isZh ? "邮箱" : "Email"}
              name="email"
              rules={[
                { required: true, type: "email", message: isZh ? "请输入正确的邮箱地址" : "Please enter valid email" },
              ]}
            >
              <Input
                ref={emailInputRef}
                prefix={<MailOutlined />}
                placeholder={isZh ? "请输入邮箱地址" : "Please enter email"}
                autoComplete="email"
              />
            </Form.Item>

            <Form.Item
              label={isZh ? "验证码" : "Code"}
              name="code"
              rules={[{ required: true, message: isZh ? "请输入验证码" : "Please enter code" }]}
            >
              <Input prefix={<SafetyCertificateOutlined />} placeholder={isZh ? "邮箱验证码" : "Email code"} />
            </Form.Item>

            <div className="login-light-inlineAction">
              <Button
                type="default"
                className="login-light-secondaryBtn"
                onClick={() => void handleSendEmailCode()}
                loading={sendingCode}
                disabled={emailCodeCountdown > 0}
              >
                {emailCodeCountdown > 0
                  ? `${emailCodeCountdown}s ${isZh ? "后重发" : "to resend"}`
                  : isZh
                    ? "发送邮箱验证码"
                    : "Send Email Code"}
              </Button>
            </div>

            <div className="login-submitRow">
              <Button htmlType="submit" className="login-submitBtn" disabled={submitting} data-fx={buttonFx}>
                <span className="login-submitBtn__label">{submitButtonLabel}</span>
                <span className="login-submitBtn__spinner" aria-hidden="true" />
              </Button>
            </div>
          </Form>
        )}

        <div className="login-light-card__footer">
          <button type="button" className="login-light-registerLink" onClick={() => setRegisterOpen(true)}>
            {isZh ? "注册用户" : "Register User"}
          </button>
        </div>
      </section>
    );
  }

  return (
    <div
      className={`gw-auth-shell ${darkMode ? "is-dark" : "is-light"} gw-accent-${accent}`}
      style={{ "--login-accent": LOGIN_ACCENT_COLORS[accent] } as CSSProperties}
    >
      <div className="gw-auth-shell__ambient" aria-hidden="true">
        <span className="gw-auth-shell__orb gw-auth-shell__orb--1" />
        <span className="gw-auth-shell__orb gw-auth-shell__orb--2" />
        <span className="gw-auth-shell__orb gw-auth-shell__orb--3" />
      </div>
      <div className="gw-auth-brand">
        <span className="gw-auth-brand__logoDot" />
        <span>{appTitle}</span>
      </div>
      <div className="gw-auth-toolbar">
        <button
          type="button"
          className={`gw-auth-toolbar__btn ${paletteOpen ? "is-active" : ""}`}
          onClick={() => setPaletteOpen((v) => !v)}
        >
          <BgColorsOutlined />
        </button>
        {paletteOpen ? (
          <div className="gw-auth-toolbar__panel gw-auth-toolbar__panel--palette">
            {(Object.keys(LOGIN_ACCENT_COLORS) as LoginAccent[]).map((key) => (
              <button
                key={key}
                type="button"
                className={`gw-auth-dot ${accent === key ? "is-active" : ""}`}
                style={{ background: LOGIN_ACCENT_COLORS[key] }}
                onClick={() => setStoreAccent(LOGIN_ACCENT_COLORS[key])}
                aria-label={key}
              />
            ))}
          </div>
        ) : null}
        <button
          type="button"
          className="gw-auth-toolbar__btn"
          title={t("app.language")}
          onClick={() => {
            const next = isZh ? "en-US" : "zh-CN";
            window.localStorage.setItem("app-locale", next);
            document.documentElement.lang = next === "en-US" ? "en" : "zh-CN";
            void i18n.changeLanguage(next);
          }}
        >
          <TranslationOutlined />
        </button>
        <button
          type="button"
          className="gw-auth-toolbar__btn"
          onClick={() => setThemeMode(darkMode ? "light" : "dark")}
        >
          {darkMode ? <BulbOutlined /> : <BulbFilled />}
        </button>
      </div>

      <div className="gw-auth-main">
        <div className="gw-auth-frame">
          <aside className="gw-auth-story" aria-label={isZh ? "平台介绍" : "Platform intro"}>
            <img
              className="gw-auth-story__hero"
              src={loginHeroImage}
              alt={isZh ? "云枢运维平台插画" : "Yunshu Ops illustration"}
            />
            <h1 className="gw-auth-story__title">{introTitle}</h1>
            <p className="gw-auth-story__desc">{introDesc}</p>
            <ul className="gw-auth-story__points" aria-label={isZh ? "平台能力" : "Platform capabilities"}>
              {introFeatures.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          </aside>
          <div className="gw-auth-panel">{renderFormCard()}</div>
        </div>
      </div>

      <Modal
        open={registerOpen}
        title={isZh ? "注册账号" : "Register Account"}
        footer={null}
        onCancel={() => setRegisterOpen(false)}
        destroyOnClose
        centered
        width={520}
        className="login-registerModal"
      >
        <Form<RegisterPayload>
          form={registerForm}
          layout="vertical"
          onFinish={handleRegister}
          size="large"
          autoComplete="off"
        >
          <Form.Item
            label={isZh ? "用户名" : "Username"}
            name="username"
            rules={[{ required: true, min: 3, max: 64, message: isZh ? "用户名长度为3-64个字符" : "3-64 characters" }]}
          >
            <Input
              prefix={<UserOutlined />}
              placeholder={isZh ? "请输入用户名" : "Please enter username"}
              autoComplete="off"
            />
          </Form.Item>
          <Form.Item
            label={isZh ? "邮箱" : "Email"}
            name="email"
            rules={[
              { required: true, type: "email", message: isZh ? "请输入正确的邮箱地址" : "Please enter valid email" },
            ]}
          >
            <Input
              ref={registerEmailInputRef}
              prefix={<MailOutlined />}
              placeholder={isZh ? "请输入邮箱地址" : "Please enter email"}
              autoComplete="email"
            />
          </Form.Item>
          <Form.Item
            label={isZh ? "昵称" : "Nickname"}
            name="nickname"
            rules={[{ required: true, max: 128, message: isZh ? "请输入昵称" : "Please enter nickname" }]}
          >
            <Input
              prefix={<UserOutlined />}
              placeholder={isZh ? "请输入昵称" : "Please enter nickname"}
              autoComplete="off"
            />
          </Form.Item>
          <Form.Item
            label={isZh ? "密码" : "Password"}
            name="password"
            rules={[{ required: true, min: 6, max: 64, message: isZh ? "密码长度为6-64个字符" : "6-64 characters" }]}
          >
            <Input.Password
              prefix={<LockOutlined />}
              placeholder={isZh ? "请输入密码" : "Please enter password"}
              autoComplete="new-password"
            />
          </Form.Item>

          <Form.Item
            label={isZh ? "验证码" : "Code"}
            name="code"
            getValueFromEvent={(event) =>
              String(event?.target?.value ?? "")
                .replace(/[^\d]/g, "")
                .slice(0, 6)
            }
            rules={[
              { required: true, message: isZh ? "请输入验证码" : "Please enter code" },
              { pattern: /^\d{6}$/, message: isZh ? "验证码为6位数字" : "Code must be 6 digits" },
            ]}
          >
            <Input
              prefix={<SafetyCertificateOutlined />}
              placeholder={isZh ? "请输入验证码" : "Please enter code"}
              maxLength={6}
              autoComplete="off"
            />
          </Form.Item>
          <Button
            type="primary"
            ghost
            className="login-sendBtn"
            onClick={() => void handleSendRegisterCode()}
            loading={sendingCode}
            disabled={registerCodeCountdown > 0}
            style={{ marginTop: -10, marginBottom: 16 }}
          >
            {registerCodeCountdown > 0 ? `${registerCodeCountdown}s` : isZh ? "发送验证码" : "Send Code"}
          </Button>

          <Button htmlType="submit" type="primary" block disabled={submitting}>
            {isZh ? "提交注册" : "Submit Registration"}
          </Button>
        </Form>
      </Modal>
    </div>
  );
}
