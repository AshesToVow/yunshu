import { useCallback, useEffect, useState } from "react";
import { getAIStatus } from "../services/ai";

/** 嵌入页 AI 按钮门控：查询 ai_enabled 与 Provider 配置。 */
export function useAiEnabled() {
  const [enabled, setEnabled] = useState<boolean | null>(null);
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const st = await getAIStatus();
      setEnabled(!!st?.enabled);
    } catch {
      // 请求失败（网络抖动/500）时保留上一次结果，不误判为未启用
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  return { enabled: enabled === true, loading, unknown: enabled === null, refresh };
}
