// @ts-nocheck
import { useCallback, useEffect, useState } from "react";
import { getMenuTree, type MenuItem } from "../services/menus";
import { filterMenuTreeByPlugins } from "../modules/filter-menu";
import { usePlugins } from "../contexts/plugin-context";

export function useMenuTree(opts?: { skipPluginFilter?: boolean }) {
  const { isPluginEnabled, loading: pluginsLoading } = usePlugins();
  const [menus, setMenus] = useState<MenuItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const tree = await getMenuTree();
      const filtered = opts?.skipPluginFilter ? tree : filterMenuTreeByPlugins(tree, isPluginEnabled);
      setMenus(filtered);
    } catch (e) {
      setError(e);
      setMenus([]);
    } finally {
      setLoading(false);
    }
  }, [isPluginEnabled, opts?.skipPluginFilter]);

  useEffect(() => {
    if (pluginsLoading && !opts?.skipPluginFilter) {
      return;
    }
    void refresh();
  }, [refresh, pluginsLoading, opts?.skipPluginFilter]);

  return { menus, loading: loading || (pluginsLoading && !opts?.skipPluginFilter), error, refresh };
}
