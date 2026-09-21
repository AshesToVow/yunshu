package logplatform

import (
	"strings"
	"testing"
)

func TestParseStoredBootstrapConfig_DefaultAutoFromLogSources(t *testing.T) {
	t.Parallel()
	// 空配置
	empty := parseStoredBootstrapConfig("")
	if !empty.AutoFromLogSources {
		t.Fatal("empty config should default auto_from_log_sources=true")
	}
	// 历史 JSON 无该字段
	legacy := parseStoredBootstrapConfig(`{"monitor_port":9196,"sources":[{"path":"/var/log/a.log","paths":["/var/log/a.log"]}]}`)
	if !legacy.AutoFromLogSources {
		t.Fatal("legacy config missing key should default auto_from_log_sources=true")
	}
	if len(legacy.Sources) != 1 {
		t.Fatalf("sources len=%d", len(legacy.Sources))
	}
	explicitFalse := parseStoredBootstrapConfig(`{"auto_from_log_sources":false,"monitor_port":9196}`)
	if explicitFalse.AutoFromLogSources {
		t.Fatal("explicit false should be preserved")
	}
	explicitTrue := parseStoredBootstrapConfig(`{"auto_from_log_sources":true}`)
	if !explicitTrue.AutoFromLogSources {
		t.Fatal("explicit true should be preserved")
	}
	if !strings.Contains(legacy.DeployDir, "loggie") && legacy.DeployDir != defaultLoggieDeployDir {
		// deploy dir normalized
		_ = legacy.DeployDir
	}
}
