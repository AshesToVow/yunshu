package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const defaultTokenFile = "/var/lib/yunshu/agent.token"

func effectiveTokenFile(cfg Config) string {
	if p := strings.TrimSpace(cfg.TokenFile); p != "" {
		return p
	}
	if strings.TrimSpace(cfg.Token) != "" {
		return ""
	}
	if strings.TrimSpace(cfg.RegisterSecret) != "" {
		return defaultTokenFile
	}
	return ""
}

func loadTokenFile(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("token file path is empty")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(raw))
	if token == "" {
		return "", fmt.Errorf("token file %s is empty", path)
	}
	return token, nil
}

func saveTokenFile(path, token string) error {
	path = strings.TrimSpace(path)
	token = strings.TrimSpace(token)
	if path == "" || token == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(token+"\n"), 0o600)
}
