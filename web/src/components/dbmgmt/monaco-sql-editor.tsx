import Editor from "@monaco-editor/react";
import { useMemo } from "react";

export type MonacoSqlEditorProps = {
  value?: string;
  onChange?: (value: string) => void;
  readOnly?: boolean;
  height?: number | string;
};

export function MonacoSqlEditor({ value = "", onChange, readOnly = false, height = 320 }: MonacoSqlEditorProps) {
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
    }),
    [readOnly],
  );

  return (
    <div style={{ width: "100%", minWidth: 0, overflow: "hidden" }}>
      <Editor
        height={height}
        language="sql"
        theme="vs-dark"
        value={value}
        onChange={(v) => onChange?.(v ?? "")}
        options={options}
      />
    </div>
  );
}
