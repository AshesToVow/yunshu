import YAML from "yaml";

export function validateYaml(text: string): string | null {
  const trimmed = text.trim();
  if (!trimmed) return null;
  try {
    YAML.parseAllDocuments(trimmed);
    return null;
  } catch (e) {
    return e instanceof Error ? e.message : "YAML 解析失败";
  }
}
