// @ts-nocheck
import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import type { PropsWithChildren } from "react";
import { DEFAULT_ENABLED_PLUGINS, setPluginManifests } from "../modules/plugin-path";
import { listPlugins, type PluginInfo } from "../services/plugins";
import { useAuth } from "./auth-context";

interface PluginContextValue {
  loading: boolean;
  plugins: PluginInfo[];
  enabled: string[];
  isPluginEnabled: (name: string) => boolean;
  refreshPlugins: () => Promise<void>;
}

const PluginContext = createContext<PluginContextValue | undefined>(undefined);

export function PluginProvider({ children }: PropsWithChildren) {
  const { isAuthenticated } = useAuth();
  const [loading, setLoading] = useState(true);
  const [plugins, setPlugins] = useState<PluginInfo[]>([]);
  const [enabled, setEnabled] = useState<string[]>([...DEFAULT_ENABLED_PLUGINS]);

  const refreshPlugins = useCallback(async () => {
    if (!isAuthenticated) {
      setEnabled([...DEFAULT_ENABLED_PLUGINS]);
      setPlugins([]);
      setPluginManifests([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const data = await listPlugins();
      const list = data.plugins ?? [];
      setPlugins(list);
      setPluginManifests(list);
      setEnabled(data.enabled?.length ? data.enabled : [...DEFAULT_ENABLED_PLUGINS]);
    } catch {
      setEnabled([...DEFAULT_ENABLED_PLUGINS]);
      setPluginManifests([]);
    } finally {
      setLoading(false);
    }
  }, [isAuthenticated]);

  useEffect(() => {
    void refreshPlugins();
  }, [refreshPlugins]);

  const enabledSet = useMemo(() => new Set(enabled.map((n) => n.toLowerCase())), [enabled]);

  const isPluginEnabled = useCallback((name: string) => enabledSet.has(name.trim().toLowerCase()), [enabledSet]);

  const value = useMemo(
    () => ({
      loading,
      plugins,
      enabled,
      isPluginEnabled,
      refreshPlugins,
    }),
    [loading, plugins, enabled, isPluginEnabled, refreshPlugins],
  );

  return <PluginContext.Provider value={value}>{children}</PluginContext.Provider>;
}

export function usePlugins() {
  const ctx = useContext(PluginContext);
  if (!ctx) {
    throw new Error("usePlugins must be used within PluginProvider");
  }
  return ctx;
}
