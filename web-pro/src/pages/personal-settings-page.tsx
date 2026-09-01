// @ts-nocheck
import { Alert, Card, Form, Input, Menu, Button, message } from "antd";
import type { MenuProps } from "antd";
import { useEffect, useMemo, useState, useCallback } from "react";
import { useNavigate, useSearchParams } from '@umijs/max';
import { useAuth } from "../contexts/auth-context";
import { changePassword, getPasswordPolicy, updateProfile } from "../services/auth";
import { clearAuthStorage } from "../services/storage";
import type { PasswordPolicy } from "../types/api";

type SettingsTab = "basic" | "password";

export function PersonalSettingsPage() {
  const { user, refreshUser } = useAuth();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const forcePassword = searchParams.get("force") === "password" || Boolean(user?.must_change_password);
  const [tab, setTab] = useState<SettingsTab>(forcePassword ? "password" : "basic");
  const [profileLoading, setProfileLoading] = useState(false);
  const [passwordLoading, setPasswordLoading] = useState(false);
  const [policy, setPolicy] = useState<PasswordPolicy | null>(null);
  const [profileForm] = Form.useForm<{ nickname: string; email?: string; phone?: string }>();
  const [passwordForm] = Form.useForm<{ old_password: string; new_password: string; confirm_password: string }>();

  const [oldPassword, setOldPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");

  const menuItems = useMemo<MenuProps["items"]>(() => {
    const items: MenuProps["items"] = [];
    if (!forcePassword) {
      items.push({ key: "basic", label: "基本设置" });
    }
    items.push({ key: "password", label: "修改密码" });
    return items;
  }, [forcePassword]);

  useEffect(() => {
    if (forcePassword) {
      setTab("password");
    }
  }, [forcePassword]);

  useEffect(() => {
    void getPasswordPolicy()
      .then(setPolicy)
      .catch(() => setPolicy(null));
  }, []);

  useEffect(() => {
    profileForm.setFieldsValue({
      nickname: user?.nickname ?? "",
      email: user?.email ?? "",
      phone: user?.phone ?? "",
    });
  }, [profileForm, user?.nickname, user?.email, user?.phone]);

  useEffect(() => {
    if (tab === "password") {
      setOldPassword("");
      setNewPassword("");
      setConfirmPassword("");
      passwordForm.resetFields();
    }
  }, [tab, passwordForm]);

  async function submitProfile() {
    const values = await profileForm.validateFields();
    setProfileLoading(true);
    try {
      await updateProfile({
        nickname: values.nickname,
        email: values.email?.trim() || undefined,
        phone: values.phone?.trim() || undefined,
      });
      await refreshUser();
      message.success("基本信息已更新");
    } finally {
      setProfileLoading(false);
    }
  }

  const handleSubmitPassword = useCallback(async () => {
    if (!oldPassword) {
      message.error("请输入旧密码");
      return;
    }
    if (!newPassword) {
      message.error("请输入新密码");
      return;
    }
    if (newPassword !== confirmPassword) {
      message.error("两次输入的新密码不一致");
      return;
    }
    const minLen = policy?.min_length ?? 8;
    if (newPassword.length < minLen) {
      message.error(`新密码至少 ${minLen} 位`);
      return;
    }

    setPasswordLoading(true);
    try {
      await changePassword({
        old_password: oldPassword,
        new_password: newPassword,
      });
      message.success("密码修改成功，请重新登录");
      clearAuthStorage();
      setTimeout(() => {
        navigate("/login");
      }, 1200);
    } catch (err: any) {
      const errorMessage = err?.response?.data?.message || err?.message || "密码修改失败";
      message.error(errorMessage);
    } finally {
      setPasswordLoading(false);
    }
  }, [oldPassword, newPassword, confirmPassword, navigate, policy?.min_length]);

  return (
    <Card className="table-card personal-settings-card">
      <div className="personal-settings">
        <aside className="personal-settings__sidebar">
          <Menu
            mode="inline"
            selectedKeys={[tab]}
            items={menuItems}
            onClick={(info) => {
              if (forcePassword && info.key !== "password") return;
              setTab(info.key as SettingsTab);
            }}
            className="personal-settings__menu"
          />
        </aside>
        <section className="personal-settings__content">
          {tab === "basic" && !forcePassword ? (
            <div>
              <h3 className="personal-settings__title">基本设置</h3>
              <Form
                form={profileForm}
                layout="vertical"
                initialValues={{ nickname: user?.nickname ?? "", email: user?.email ?? "", phone: user?.phone ?? "" }}
                className="personal-settings__form"
                autoComplete="off"
              >
                <Form.Item label="昵称" name="nickname" rules={[{ required: true, message: "请输入昵称" }]}>
                  <Input placeholder="请输入昵称" autoComplete="off" />
                </Form.Item>
                <Form.Item label="账号">
                  <Input value={user?.username ?? ""} disabled />
                </Form.Item>
                <Form.Item label="邮箱" name="email" rules={[{ type: "email", message: "请输入正确邮箱地址" }]}>
                  <Input placeholder="请输入邮箱地址" autoComplete="off" />
                </Form.Item>
                <Form.Item label="手机号" name="phone" extra="选填；与钉钉/企微一致时，作为监控规则处理人可被 @ 提醒">
                  <Input placeholder="选填" maxLength={20} autoComplete="off" />
                </Form.Item>
                <Button type="primary" loading={profileLoading} onClick={() => void submitProfile()}>
                  更新基本信息
                </Button>
              </Form>
            </div>
          ) : (
            <div>
              <h3 className="personal-settings__title">修改密码</h3>
              {forcePassword ? (
                <Alert
                  type="warning"
                  showIcon
                  style={{ marginBottom: 16 }}
                  message="密码已过期或需强制修改"
                  description={
                    policy
                      ? `请按策略修改密码后重新登录。要求：${policy.hint}；有效期：${policy.expiry_hint}`
                      : "请修改密码后重新登录。"
                  }
                />
              ) : null}
              {policy && !forcePassword ? (
                <Alert
                  type="info"
                  showIcon
                  style={{ marginBottom: 16 }}
                  message={`密码策略：${policy.hint}；过期：${policy.expiry_hint}`}
                />
              ) : null}
              <Form form={passwordForm} layout="vertical" className="personal-settings__form" autoComplete="off">
                <Form.Item label="旧密码" rules={[{ required: true, message: "请输入旧密码" }]}>
                  <Input.Password
                    value={oldPassword}
                    onChange={(e) => setOldPassword(e.target.value)}
                    placeholder="请输入旧密码"
                    autoComplete="off"
                  />
                </Form.Item>
                <Form.Item
                  label="新密码"
                  rules={[
                    { required: true, message: "请输入新密码" },
                    { min: policy?.min_length ?? 8, message: `新密码至少 ${policy?.min_length ?? 8} 位` },
                  ]}
                  extra={policy?.hint}
                >
                  <Input.Password
                    value={newPassword}
                    onChange={(e) => setNewPassword(e.target.value)}
                    placeholder="请输入新密码"
                    autoComplete="off"
                  />
                </Form.Item>
                <Form.Item label="确认密码" rules={[{ required: true, message: "请再次输入新密码" }]}>
                  <Input.Password
                    value={confirmPassword}
                    onChange={(e) => setConfirmPassword(e.target.value)}
                    placeholder="请再次输入新密码"
                    autoComplete="off"
                  />
                </Form.Item>
                <Button type="primary" loading={passwordLoading} onClick={() => void handleSubmitPassword()}>
                  更新密码
                </Button>
              </Form>
            </div>
          )}
        </section>
      </div>
    </Card>
  );
}
