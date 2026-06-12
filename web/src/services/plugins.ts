import { getData, http } from "./http";

export interface PluginInfo {
  name: string;
  description: string;
  enabled: boolean;
}

export interface PluginListResult {
  plugins: PluginInfo[];
  enabled: string[];
  registered: string[];
}

export function listPlugins() {
  return getData<PluginListResult>(http.get("/plugins"));
}
