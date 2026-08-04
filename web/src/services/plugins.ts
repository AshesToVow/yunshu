import { getData, http } from "./http";

export interface PluginManifest {
  menu_path_prefixes?: string[];
  api_prefixes?: string[];
  depends_on?: string[];
  workers?: string[];
}

export interface PluginInfo {
  name: string;
  description: string;
  enabled: boolean;
  manifest?: PluginManifest;
}

export interface PluginListResult {
  plugins: PluginInfo[];
  enabled: string[];
  registered: string[];
}

export function listPlugins() {
  return getData<PluginListResult>(http.get("/plugins"));
}
