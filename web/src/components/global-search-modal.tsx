import { SearchOutlined } from "@ant-design/icons";
import { Input, List, Modal, Tag, Typography } from "antd";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { searchK8s, type K8sSearchItem } from "../services/k8s-search";

type Props = {
  open: boolean;
  onClose: () => void;
};

export function GlobalSearchModal({ open, onClose }: Props) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [q, setQ] = useState("");
  const [loading, setLoading] = useState(false);
  const [items, setItems] = useState<K8sSearchItem[]>([]);
  const seqRef = useRef(0);

  useEffect(() => {
    if (!open) {
      setQ("");
      setItems([]);
      return;
    }
    const kw = q.trim();
    if (kw.length < 2) {
      setItems([]);
      return;
    }
    const seq = ++seqRef.current;
    const timer = window.setTimeout(() => {
      setLoading(true);
      void searchK8s({ q: kw, limit: 30 })
        .then((list) => {
          if (seq !== seqRef.current) return;
          setItems(list ?? []);
        })
        .catch(() => {
          if (seq !== seqRef.current) return;
          setItems([]);
        })
        .finally(() => {
          if (seq === seqRef.current) setLoading(false);
        });
    }, 300);
    return () => window.clearTimeout(timer);
  }, [open, q]);

  function openItem(it: K8sSearchItem) {
    const path = it.link_path || "/pods";
    const url = new URL(path, window.location.origin);
    navigate(`${url.pathname}${url.search}`);
    onClose();
  }

  return (
    <Modal title={t("app.search")} open={open} onCancel={onClose} footer={null} width={640} destroyOnClose>
      <Input
        autoFocus
        prefix={<SearchOutlined />}
        placeholder={t("app.searchPlaceholder")}
        value={q}
        onChange={(e) => setQ(e.target.value)}
      />
      <List
        style={{ marginTop: 12, maxHeight: 420, overflow: "auto" }}
        loading={loading}
        dataSource={items}
        locale={{
          emptyText: q.trim().length < 2 ? t("app.searchHint") : t("app.searchEmpty"),
        }}
        renderItem={(it) => (
          <List.Item style={{ cursor: "pointer" }} onClick={() => openItem(it)}>
            <List.Item.Meta
              title={
                <span>
                  <Tag>{it.type}</Tag>
                  {it.name}
                  <Typography.Text type="secondary" style={{ marginLeft: 8 }}>
                    {it.cluster_name} / {it.namespace}
                  </Typography.Text>
                </span>
              }
            />
          </List.Item>
        )}
      />
    </Modal>
  );
}
