package system

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"yunshu/internal/config"
	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/apiroute"
	"yunshu/internal/plugin"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const reclaimTestModel = `
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
`

type fakePermissionRepo struct {
	interfaces.PermissionRepository
	rows    []model.Permission
	deleted []uint
	created []model.Permission
}

func (f *fakePermissionRepo) ListAll(context.Context) ([]model.Permission, error) {
	return f.rows, nil
}

func (f *fakePermissionRepo) Delete(_ context.Context, p *model.Permission) error {
	f.deleted = append(f.deleted, p.ID)
	return nil
}

func (f *fakePermissionRepo) GetByResourceAction(context.Context, string, string) (*model.Permission, error) {
	return nil, gorm.ErrRecordNotFound
}

func (f *fakePermissionRepo) Create(_ context.Context, p *model.Permission) error {
	f.created = append(f.created, *p)
	return nil
}

func newReclaimEngine(t *testing.T, routes ...[2]string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	for _, r := range routes {
		engine.Handle(r[0], r[1], func(*gin.Context) {})
	}
	return engine
}

func newReclaimEnforcer(t *testing.T) *casbin.SyncedEnforcer {
	t.Helper()
	m, err := casbinmodel.NewModelFromString(reclaimTestModel)
	if err != nil {
		t.Fatalf("casbin model: %v", err)
	}
	enforcer, err := casbin.NewSyncedEnforcer(m)
	if err != nil {
		t.Fatalf("casbin enforcer: %v", err)
	}
	return enforcer
}

func hasPolicy(t *testing.T, enforcer *casbin.SyncedEnforcer, sub, obj, act string) bool {
	t.Helper()
	return enforcer.HasPolicy(sub, obj, act)
}

func TestReclaimOrphansRemovesStalePermissionsAndPolicies(t *testing.T) {
	t.Parallel()
	engine := newReclaimEngine(t, [2]string{"GET", "/api/v1/users"})
	repo := &fakePermissionRepo{rows: []model.Permission{
		{ID: 1, Resource: "/api/v1/users", Action: "GET", Description: apiroute.AutoSyncDescription("core")},
		{ID: 2, Resource: "/api/v1/legacy", Action: "GET", Description: apiroute.AutoSyncDescription("core")},
		{ID: 3, Resource: "/api/v1/roles", Action: "GET", Description: "角色列表"},
		{ID: 4, Resource: "/api/v1/custom", Action: "GET", Description: "运营自建权限"},
		{ID: 5, Resource: "/api/v1/dict/sensitive", Action: "GET", Description: "查看敏感字典"},
	}}
	enforcer := newReclaimEnforcer(t)
	for _, rule := range [][]string{
		{"ops", "/api/v1/users", "GET"},
		{"ops", "/api/v1/legacy", "GET"},
		{"ops", "/api/v1/roles", "GET"},
		{"ops", "/api/v1/ghost", "GET"},
	} {
		if _, err := enforcer.AddPolicy(rule[0], rule[1], rule[2]); err != nil {
			t.Fatalf("seed policy: %v", err)
		}
	}

	svc := &PermissionSyncService{permissionRepo: repo}
	result, err := svc.ReclaimOrphans(context.Background(), engine, enforcer)
	if err != nil {
		t.Fatalf("reclaim: %v", err)
	}
	if result.Aborted {
		t.Fatalf("unexpected abort: %s", result.Reason)
	}
	if result.DeletedPermissions != 2 {
		t.Fatalf("deleted permissions = %d, want 2 (%v)", result.DeletedPermissions, result.Samples)
	}
	if result.Skipped != 1 {
		t.Fatalf("skipped = %d, want 1 (manual row)", result.Skipped)
	}
	if got := fmt.Sprint(repo.deleted); got != "[2 3]" {
		t.Fatalf("deleted ids = %s, want [2 3]", got)
	}
	if !hasPolicy(t, enforcer, "ops", "/api/v1/users", "GET") {
		t.Fatal("live route policy must be kept")
	}
	for _, obj := range []string{"/api/v1/legacy", "/api/v1/roles", "/api/v1/ghost"} {
		if hasPolicy(t, enforcer, "ops", obj, "GET") {
			t.Fatalf("orphan policy %s should be removed", obj)
		}
	}
	if result.RemovedPolicies != 3 {
		t.Fatalf("removed policies = %d, want 3", result.RemovedPolicies)
	}
}

func TestReclaimOrphansAbortsWithoutRoutes(t *testing.T) {
	t.Parallel()
	repo := &fakePermissionRepo{rows: []model.Permission{
		{ID: 1, Resource: "/api/v1/legacy", Action: "GET", Description: apiroute.AutoSyncDescription("core")},
	}}
	svc := &PermissionSyncService{permissionRepo: repo}
	result, err := svc.ReclaimOrphans(context.Background(), newReclaimEngine(t), nil)
	if err != nil {
		t.Fatalf("reclaim: %v", err)
	}
	if !result.Aborted || result.Reason != "no live api routes" {
		t.Fatalf("want abort on empty engine, got %+v", result)
	}
	if len(repo.deleted) != 0 {
		t.Fatalf("must not delete on abort, deleted %v", repo.deleted)
	}
}

func TestReclaimOrphansAbortsOverThreshold(t *testing.T) {
	t.Parallel()
	engine := newReclaimEngine(t, [2]string{"GET", "/api/v1/users"})
	rows := make([]model.Permission, 0, defaultMaxReclaim+1)
	for i := 0; i < defaultMaxReclaim+1; i++ {
		rows = append(rows, model.Permission{
			ID: uint(i + 1), Resource: fmt.Sprintf("/api/v1/legacy-%d", i), Action: "GET",
			Description: apiroute.AutoSyncDescription("core"),
		})
	}
	repo := &fakePermissionRepo{rows: rows}
	svc := &PermissionSyncService{permissionRepo: repo}
	result, err := svc.ReclaimOrphans(context.Background(), engine, nil)
	if err != nil {
		t.Fatalf("reclaim: %v", err)
	}
	if !result.Aborted || result.Reason != "exceeds reclaim threshold" {
		t.Fatalf("want threshold abort, got %+v", result)
	}
	if len(repo.deleted) != 0 {
		t.Fatalf("must not delete on abort, deleted %d", len(repo.deleted))
	}
}

type reclaimGatePlugin struct {
	plugin.Base
}

func (p *reclaimGatePlugin) Name() string { return "reclaim-gate-test" }

func (p *reclaimGatePlugin) Manifest() plugin.Manifest {
	return plugin.Manifest{APIPrefixes: []string{"/api/v1/reclaim-gate"}}
}

var reclaimGateOnce sync.Once

func registerReclaimGatePlugin() {
	reclaimGateOnce.Do(func() { plugin.Register(&reclaimGatePlugin{}) })
}

func TestReclaimOrphansSkipsDisabledPluginPaths(t *testing.T) {
	registerReclaimGatePlugin()
	engine := newReclaimEngine(t, [2]string{"GET", "/api/v1/users"})
	repo := &fakePermissionRepo{rows: []model.Permission{
		{ID: 1, Resource: "/api/v1/reclaim-gate/things", Action: "GET", Description: apiroute.AutoSyncDescription("reclaim-gate-test")},
	}}
	svc := &PermissionSyncService{
		permissionRepo: repo,
		plugins:        &config.PluginsConfig{Enabled: []string{"core"}},
	}
	result, err := svc.ReclaimOrphans(context.Background(), engine, nil)
	if err != nil {
		t.Fatalf("reclaim: %v", err)
	}
	if result.DeletedPermissions != 0 || result.Skipped != 1 {
		t.Fatalf("disabled plugin path must be skipped, got %+v", result)
	}
	if len(repo.deleted) != 0 {
		t.Fatalf("must not delete disabled plugin permission, deleted %v", repo.deleted)
	}
}

// 插件 Manifest 前缀只有登记项、没有同名路由；回收删掉后下一轮同步会重建，撞唯一键。
func TestReclaimOrphansKeepsPluginManifestPrefix(t *testing.T) {
	registerReclaimGatePlugin()
	engine := newReclaimEngine(t, [2]string{"GET", "/api/v1/users"})
	repo := &fakePermissionRepo{rows: []model.Permission{
		{ID: 1, Resource: "/api/v1/reclaim-gate", Action: "GET", Description: apiroute.AutoSyncDescription("reclaim-gate-test")},
		{ID: 2, Resource: "/api/v1/legacy", Action: "GET", Description: apiroute.AutoSyncDescription("core")},
	}}
	svc := &PermissionSyncService{
		permissionRepo: repo,
		plugins:        &config.PluginsConfig{Enabled: []string{"core", "reclaim-gate-test"}},
	}
	result, err := svc.ReclaimOrphans(context.Background(), engine, nil)
	if err != nil {
		t.Fatalf("reclaim: %v", err)
	}
	if result.DeletedPermissions != 1 || result.Skipped != 1 {
		t.Fatalf("manifest prefix must be kept, got %+v", result)
	}
	if got := fmt.Sprint(repo.deleted); got != "[2]" {
		t.Fatalf("deleted ids = %s, want [2]", got)
	}
}

func TestUpsertRoutePermissionUsesCatalogMetadata(t *testing.T) {
	t.Parallel()
	repo := &fakePermissionRepo{}
	svc := &PermissionSyncService{permissionRepo: repo}
	created, err := svc.upsertRoutePermission(context.Background(), "GET", "/api/v1/users", "core")
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if !created || len(repo.created) != 1 {
		t.Fatalf("want one created row, got created=%v rows=%v", created, repo.created)
	}
	row := repo.created[0]
	if row.Name != "用户列表" {
		t.Fatalf("catalog name not applied: %q", row.Name)
	}
	if apiroute.IsAutoSyncedDescription(row.Description) {
		t.Fatalf("catalog row must not be marked auto-synced: %q", row.Description)
	}
}

func TestUpsertRoutePermissionFallsBackToAutoName(t *testing.T) {
	t.Parallel()
	repo := &fakePermissionRepo{}
	svc := &PermissionSyncService{permissionRepo: repo}
	if _, err := svc.upsertRoutePermission(context.Background(), "GET", "/api/v1/not-in-catalog", "core"); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	row := repo.created[0]
	if !apiroute.IsAutoSyncedDescription(row.Description) {
		t.Fatalf("unknown route must keep auto-synced marker: %q", row.Description)
	}
	if row.Name != apiroute.DefaultName("GET", "/api/v1/not-in-catalog") {
		t.Fatalf("unexpected fallback name: %q", row.Name)
	}
}

// duplicatePermissionRepo 模拟唯一键冲突：活跃行查不到，但存在同键软删除行。
type duplicatePermissionRepo struct {
	interfaces.PermissionRepository
	deleted      *model.Permission
	restoredID   uint
	restoredName string
	restoredAct  string
}

func (f *duplicatePermissionRepo) Create(context.Context, *model.Permission) error {
	return errors.New("Error 1062 (23000): Duplicate entry for key 'idx_perm_resource_action'")
}

func (f *duplicatePermissionRepo) GetByResourceAction(context.Context, string, string) (*model.Permission, error) {
	return nil, gorm.ErrRecordNotFound
}

func (f *duplicatePermissionRepo) GetSoftDeletedByResourceAction(context.Context, string, string) (*model.Permission, error) {
	if f.deleted == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return f.deleted, nil
}

func (f *duplicatePermissionRepo) Restore(_ context.Context, id uint, name, _ string, action string) error {
	f.restoredID = id
	f.restoredName = name
	f.restoredAct = action
	return nil
}

func TestUpsertRoutePermissionRevivesSoftDeletedRow(t *testing.T) {
	t.Parallel()
	repo := &duplicatePermissionRepo{
		deleted: &model.Permission{
			ID:        42,
			Resource:  "/api/v1/users",
			Action:    "get",
			DeletedAt: gorm.DeletedAt{Time: time.Now(), Valid: true},
		},
	}
	svc := &PermissionSyncService{permissionRepo: repo}
	created, err := svc.upsertRoutePermission(context.Background(), "GET", "/api/v1/users", "core")
	if err != nil {
		t.Fatalf("upsert must revive instead of failing: %v", err)
	}
	if !created {
		t.Fatal("revived row should count as created")
	}
	if repo.restoredID != 42 {
		t.Fatalf("restored id = %d, want 42", repo.restoredID)
	}
	if repo.restoredName != "用户列表" {
		t.Fatalf("restore must apply catalog name, got %q", repo.restoredName)
	}
	if repo.restoredAct != "GET" {
		t.Fatalf("restore must normalize action, got %q", repo.restoredAct)
	}
}

func TestUpsertRoutePermissionReportsUnrelatedCreateError(t *testing.T) {
	t.Parallel()
	repo := &duplicatePermissionRepo{}
	svc := &PermissionSyncService{permissionRepo: repo}
	if _, err := svc.upsertRoutePermission(context.Background(), "GET", "/api/v1/users", "core"); err == nil {
		t.Fatal("create error without a soft-deleted row must surface")
	}
	if repo.restoredID != 0 {
		t.Fatalf("must not restore on unrelated failure, restored %d", repo.restoredID)
	}
}
