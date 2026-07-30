package cicd

import (
	"testing"

	"yunshu/internal/model"
)

func TestApplyApolloParams_ProjectOverride(t *testing.T) {
	params := map[string]string{"Tenv": "test"}
	applyApolloParams(params, BuildParamsInput{
		ApolloMeta:       "http://apollo.example/",
		ApolloNamespaces: "application,biz.yml",
	})
	if params["APOLLO_META"] != "http://apollo.example/" {
		t.Fatalf("APOLLO_META=%q", params["APOLLO_META"])
	}
	if params["APOLLO_ENV"] != "FAT" {
		t.Fatalf("expected FAT from tenv=test, got %q", params["APOLLO_ENV"])
	}
	if params["APOLLO_NAMESPACES"] != "application,biz.yml" {
		t.Fatalf("APOLLO_NAMESPACES=%q", params["APOLLO_NAMESPACES"])
	}
}

func TestApplyApolloParams_ExplicitEnv(t *testing.T) {
	params := map[string]string{"Tenv": "prod"}
	applyApolloParams(params, BuildParamsInput{
		ApolloMeta: "http://meta/",
		ApolloEnv:  "UAT",
	})
	if params["APOLLO_ENV"] != "UAT" {
		t.Fatalf("APOLLO_ENV=%q", params["APOLLO_ENV"])
	}
}

func TestApplyApolloParams_EmptySkipped(t *testing.T) {
	params := map[string]string{"Tenv": "dev"}
	applyApolloParams(params, BuildParamsInput{})
	if _, ok := params["APOLLO_META"]; ok {
		t.Fatal("should not set APOLLO_META when project apollo empty")
	}
}

func TestBuildJenkinsParams_ApolloOnContainerDeploy(t *testing.T) {
	params := BuildJenkinsParams(BuildParamsInput{
		Service: &model.CicdService{ServiceType: model.CicdServiceTypeBackend, Identifier: "demo"},
		DeployConfig: &model.CicdDeployConfig{
			DeployKind:   model.CicdDeployKindContainer,
			DeployMethod: "kubectl",
			Tenv:         "prod",
			K8sNamespace: "default",
			Replicas:     1,
		},
		Tenv:       "prod",
		ApolloMeta: "http://apollo-prod/",
		ApolloEnv:  "PRO",
	})
	if params["APOLLO_META"] != "http://apollo-prod/" {
		t.Fatalf("container path missing APOLLO_META: %v", params)
	}
	if params["APOLLO_ENV"] != "PRO" {
		t.Fatalf("APOLLO_ENV=%q", params["APOLLO_ENV"])
	}
}
