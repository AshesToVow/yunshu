package cicd

import (
	"testing"

	"yunshu/internal/config"
)

func TestApplyHarborParams_ProjectOverride(t *testing.T) {
	cfg := config.CicdConfig{
		Harbor: config.HarborConfig{
			URL:          "global.harbor.local",
			ProjectGroup: "registry",
			HostIP:       "10.0.0.1",
		},
		Credentials: config.CicdCredentialsConfig{Harbor: "HARBOR_ID"},
	}
	params := map[string]string{}
	applyHarborParams(params, cfg, "https://team.harbor.com/", "team-a")
	if params["HARBOR_URL"] != "team.harbor.com" {
		t.Fatalf("HARBOR_URL=%q", params["HARBOR_URL"])
	}
	if params["PROJECT_GROUP"] != "team-a" {
		t.Fatalf("PROJECT_GROUP=%q", params["PROJECT_GROUP"])
	}
	// partial override: only project name
	params2 := map[string]string{}
	applyHarborParams(params2, cfg, "", "only-project")
	if params2["HARBOR_URL"] != "global.harbor.local" {
		t.Fatalf("expected global url, got %q", params2["HARBOR_URL"])
	}
	if params2["PROJECT_GROUP"] != "only-project" {
		t.Fatalf("PROJECT_GROUP=%q", params2["PROJECT_GROUP"])
	}
}
