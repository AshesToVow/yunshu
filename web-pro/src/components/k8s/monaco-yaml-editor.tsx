// @ts-nocheck
import Editor, { type OnMount } from "@monaco-editor/react";
import { Alert } from "antd";
import { useCallback, useEffect, useMemo, useState } from "react";
import YAML from "yaml";

export type MonacoYamlEditorProps = {
  value?: string;
  onChange?: (value: string) => void;
  readOnly?: boolean;
  height?: number | string;
  className?: string;
};

function validateYaml(text: string): string | null {
  const trimmed = text.trim();
  if (!trimmed) return null;
  try {
    YAML.parseAllDocuments(trimmed);
    return null;
  } catch (e) {
    return e instanceof Error ? e.message : "YAML 解析失败";
  }
}

export function MonacoYamlEditor({
  value = "",
  onChange,
  readOnly = false,
  height = 360,
  className,
}: MonacoYamlEditorProps) {
  const [error, setError] = useState<string | null>(null);

  const runValidate = useCallback((text: string) => {
    setError(validateYaml(text));
  }, []);

  useEffect(() => {
    runValidate(value);
  }, [value, runValidate]);

  const options = useMemo(
    () => ({
      readOnly,
      minimap: { enabled: false },
      wordWrap: "on" as const,
      scrollBeyondLastLine: false,
      automaticLayout: true,
      fontSize: 13,
      lineNumbers: "on" as const,
      tabSize: 2,
      renderWhitespace: "selection" as const,
    }),
    [readOnly],
  );

  const handleMount: OnMount = (editor) => {
    editor.updateOptions({ readOnly });
  };

  return (
    <div className={className ? `monaco-yaml-editor ${className}` : "monaco-yaml-editor"}>
      {error ? (
        <Alert type="warning" showIcon message="YAML 语法提示" description={error} style={{ marginBottom: 8 }} />
      ) : null}
      <div className="monaco-yaml-editor__surface">
        <Editor
          height={height}
          defaultLanguage="yaml"
          theme="vs-dark"
          value={value}
          onChange={(next) => {
            const text = next ?? "";
            onChange?.(text);
            runValidate(text);
          }}
          onMount={handleMount}
          options={options}
        />
      </div>
    </div>
  );
}

export { validateYaml };
