// @ts-nocheck
import {
  Alert,
  Button,
  Card,
  Collapse,
  Input,
  Radio,
  Segmented,
  Select,
  Space,
  Table,
  Typography,
} from "antd";
import { PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { lazy, Suspense } from "react";
import { useAlertMonitor } from "../context";

const AlertInhibitionPanel = lazy(async () => {
  const mod = await import("../../alert-inhibition-panel");
  return { default: mod.AlertInhibitionPanel };
});

export function InhibitionTab() {
  const ctx = useAlertMonitor();
  return (
<AlertInhibitionPanel projectId={ctx.projectContextId} />
  );
}
