package prompts

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed *.md
var fs embed.FS

// Load 读取内嵌 prompt 模板并用 vars 做简单 {{key}} 替换。
func Load(name string, vars map[string]string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("prompt name required")
	}
	if !strings.HasSuffix(name, ".md") {
		name += ".md"
	}
	raw, err := fs.ReadFile(name)
	if err != nil {
		return "", fmt.Errorf("load prompt %s: %w", name, err)
	}
	out := string(raw)
	for k, v := range vars {
		out = strings.ReplaceAll(out, "{{"+k+"}}", v)
	}
	return out, nil
}
