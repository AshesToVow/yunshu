package menu

import (
	"testing"

	"yunshu/internal/model"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
)

func TestDefaultPathBindingsCoversCatalogLeaves(t *testing.T) {
	for _, spec := range DefaultCatalog() {
		var walk func([]Spec)
		walk = func(items []Spec) {
			for _, it := range items {
				if it.Component != "" {
					path := normalizeMenuPath(it.Path)
					if _, ok := DefaultPathBindings()[path]; !ok {
						t.Errorf("missing default binding for catalog path %q", it.Path)
					}
				}
				if len(it.Children) > 0 {
					walk(it.Children)
				}
			}
		}
		walk([]Spec{spec})
	}
}

func TestUserCanAccessMenuEmptyBindings(t *testing.T) {
	if UserCanAccessMenu(nil, 1, nil) {
		t.Fatal("empty bindings should deny access")
	}
}

func TestDefaultPathBindings_AIAssistant(t *testing.T) {
	t.Parallel()
	eps := DefaultPathBindings()["/ai/assistant"]
	if len(eps) < 2 {
		t.Fatalf("ai/assistant should bind chat+sessions, got %#v", eps)
	}
}

func newMenuTestEnforcer(t *testing.T) *casbin.SyncedEnforcer {
	t.Helper()
	m, err := casbinmodel.NewModelFromString(`
[request_definition]
r = sub, obj, act
[policy_definition]
p = sub, obj, act
[role_definition]
g = _, _
[policy_effect]
e = some(where (p.eft == allow))
[matchers]
m = g(r.sub, p.sub) && keyMatch2(r.obj, p.obj) && r.act == p.act
`)
	if err != nil {
		t.Fatalf("model: %v", err)
	}
	e, err := casbin.NewSyncedEnforcer(m)
	if err != nil {
		t.Fatalf("enforcer: %v", err)
	}
	return e
}

func TestFilterMenusByAccess_KeepsParentWhenChildAllowed(t *testing.T) {
	t.Parallel()
	e := newMenuTestEnforcer(t)
	_, _ = e.AddGroupingPolicy("user:1", "role:ops")
	_, _ = e.AddPolicy("role:ops", "/api/v1/ai/chat", "POST")

	items := []model.Menu{
		{
			ID: 1, Path: "/ai", Name: "AI",
			Children: []model.Menu{
				{ID: 2, Path: "/ai/assistant", Name: "助手", Component: "ai-assistant-page"},
				{ID: 3, Path: "/ai/approvals", Name: "审批", Component: "ai-approvals-page"},
			},
		},
		{
			ID: 10, Path: "/system", Name: "系统",
			Children: []model.Menu{
				{ID: 11, Path: "/unknown-leaf", Name: "未知", Component: "unknown-page"},
			},
		},
	}
	store := NewBindingStore(nil, nil)
	out := FilterMenusByAccess(items, e, 1, store)
	if len(out) != 1 || out[0].Path != "/ai" {
		t.Fatalf("want only /ai parent, got %#v", out)
	}
	if len(out[0].Children) != 1 || out[0].Children[0].Path != "/ai/assistant" {
		t.Fatalf("want only assistant child, got %#v", out[0].Children)
	}
}
