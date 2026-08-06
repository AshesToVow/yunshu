package menu

import "strings"

// EntryPermission 菜单入口所需的 API 权限（通常为页面列表 GET）。
type EntryPermission struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

func (e EntryPermission) Key() string {
	return e.Resource + "::" + strings.ToUpper(strings.TrimSpace(e.Action))
}

// DefaultPathBindings 内置菜单 path → 入口权限映射（与 catalog 同步维护）。
func DefaultPathBindings() map[string][]EntryPermission {
	raw := map[string]EntryPermission{
		"/": {Resource: "/api/v1/overview", Action: "GET"},

		"/alert-channels":              {"/api/v1/alerts/channels", "GET"},
		"/alert-monitor-platform":      {"/api/v1/alerts/datasources", "GET"},
		"/alert-duty":                  {"/api/v1/alerts/duty-blocks", "GET"},
		"/alert-maintenance":           {"/api/v1/alerts/maintenance-windows", "GET"},
		"/alert-quality":               {"/api/v1/alerts/quality-report", "GET"},

		"/projects":                    {"/api/v1/projects", "GET"},
		"/project-members":             {"/api/v1/projects/:id/members", "GET"},
		"/project-servers":             {"/api/v1/projects/:id/servers", "GET"},
		"/project-inspect":             {"/api/v1/projects/:id/inspect/plan", "GET"},
		"/service-catalog":             {"/api/v1/projects/:id/service-catalog", "GET"},
		"/service-portrait":            {"/api/v1/projects/:id/service-catalog", "GET"},

		"/project-services":            {"/api/v1/projects/:id/services", "GET"},
		"/project-log-sources":         {"/api/v1/projects/:id/log-sources", "GET"}, // 兼容旧菜单 path
		"/project-logs":                {"/api/v1/projects/:id/logs/search", "GET"},
		"/log-retention":               {"/api/v1/log-platform/retention", "GET"},
		"/loggie-status":               {"/api/v1/projects/:id/loggie/status", "GET"},

		"/dbmgmt/apply/database":       {"/api/v1/projects/:id/dbmgmt/tickets", "GET"},
		"/dbmgmt/apply/query":          {"/api/v1/projects/:id/dbmgmt/access-requests", "GET"},
		"/dbmgmt/apply/app-user":       {"/api/v1/projects/:id/dbmgmt/app-user-requests", "GET"},
		"/dbmgmt/apply/query-grants":   {"/api/v1/projects/:id/dbmgmt/grants", "GET"},
		"/dbmgmt/instances":            {"/api/v1/projects/:id/dbmgmt/instances", "GET"},
		"/dbmgmt/sql/query":            {"/api/v1/projects/:id/dbmgmt/instances", "GET"},
		"/dbmgmt/sql/audit":            {"/api/v1/projects/:id/dbmgmt/tickets", "GET"},
		"/dbmgmt/workflow/pending":     {"/api/v1/projects/:id/dbmgmt/tickets", "GET"},
		"/dbmgmt/workflow/history":     {"/api/v1/projects/:id/dbmgmt/tickets", "GET"},
		"/dbmgmt/approval-flow":        {"/api/v1/projects/:id/dbmgmt/approval-flow", "GET"},
		"/dbmgmt/audit":                {"/api/v1/projects/:id/dbmgmt/audit-logs", "GET"},
		"/dbmgmt/grants":               {"/api/v1/projects/:id/dbmgmt/grants", "GET"},

		"/mysql-backup": {"/api/v1/projects/:id/mysql-backup/instances", "GET"},

		"/users":              {"/api/v1/users", "GET"},
		"/user-groups":        {"/api/v1/user-groups", "GET"},
		"/departments":        {"/api/v1/departments/tree", "GET"},
		"/roles":              {"/api/v1/roles", "GET"},
		"/permissions":        {"/api/v1/permissions", "GET"},
		"/policies":           {"/api/v1/policies", "GET"},
		"/k8s-scoped-policies": {"/api/v1/k8s-policies", "GET"},
		"/registrations":      {"/api/v1/registrations", "GET"},
		"/menus":              {"/api/v1/menus/tree", "GET"},
		"/dict-entries":       {"/api/v1/dict/entries", "GET"},
		"/login-logs":         {"/api/v1/login-logs", "GET"},
		"/operation-logs":     {"/api/v1/operation-logs", "GET"},
		"/banned-ips":         {"/api/v1/security/banned-ips", "GET"},

		"/clusters":                    {"/api/v1/clusters", "GET"},
		"/namespaces":                  {"/api/v1/namespaces", "GET"},
		"/nodes":                       {"/api/v1/nodes", "GET"},
		"/component-status":            {"/api/v1/clusters/:id/component-statuses", "GET"},
		"/pods":                        {"/api/v1/pods", "GET"},
		"/deployments":                 {"/api/v1/deployments", "GET"},
		"/statefulsets":                {"/api/v1/statefulsets", "GET"},
		"/daemonsets":                  {"/api/v1/daemonsets", "GET"},
		"/cronjobs":                    {"/api/v1/cronjobs", "GET"},
		"/jobs":                        {"/api/v1/jobs", "GET"},
		"/configmaps":                  {"/api/v1/configmaps", "GET"},
		"/secrets":                     {"/api/v1/secrets", "GET"},
		"/k8s-services":                {"/api/v1/k8s-services", "GET"},
		"/persistentvolumes":           {"/api/v1/persistentvolumes", "GET"},
		"/persistentvolumeclaims":      {"/api/v1/persistentvolumeclaims", "GET"},
		"/storageclasses":              {"/api/v1/storageclasses", "GET"},
		"/ingresses":                   {"/api/v1/ingresses", "GET"},
		"/ingress-classes":             {"/api/v1/ingresses/classes", "GET"},
		"/network-policies":            {"/api/v1/network-policies", "GET"},
		"/rbac/roles":                  {"/api/v1/rbac/roles", "GET"},
		"/rbac/rolebindings":           {"/api/v1/rbac/rolebindings", "GET"},
		"/rbac/clusterroles":           {"/api/v1/rbac/clusterroles", "GET"},
		"/rbac/clusterrolebindings":    {"/api/v1/rbac/clusterrolebindings", "GET"},
		"/events":                      {"/api/v1/events", "GET"},
		"/k8s/event-forward":           {"/api/v1/k8s/event-forward/rules", "GET"},
		"/serviceaccounts":             {"/api/v1/serviceaccounts", "GET"},
		"/cluster-api-resources":       {"/api/v1/clusters/:id/api-resources", "GET"},
		"/horizontal-pod-autoscalers":  {"/api/v1/horizontal-pod-autoscalers", "GET"},
		"/helm/releases":               {"/api/v1/helm/releases", "GET"},
		"/helm/charts":                 {"/api/v1/helm/harbor/charts", "GET"},
		"/k8s-resource-topology":       {"/api/v1/k8s/topology", "GET"},

		"/cicd/services":        {"/api/v1/projects/:id/cicd/services", "GET"},
		"/cicd/todo":            {"/api/v1/projects/:id/cicd/release-runs", "GET"},
		"/cicd/approval-flow":   {"/api/v1/projects/:id/cicd/approval-flow", "GET"},
		"/cicd/build-records":   {"/api/v1/projects/:id/cicd/build-runs", "GET"},
		"/cicd/release-records": {"/api/v1/projects/:id/cicd/release-runs", "GET"},
		"/cicd/registries":      {"/api/v1/registries", "GET"},
		"/cicd/image-browser":   {"/api/v1/registries", "GET"},

		"/crds": {"/api/v1/crds", "GET"},
		"/crs":  {"/api/v1/crs", "GET"},
	}

	out := make(map[string][]EntryPermission, len(raw))
	for path, perm := range raw {
		out[normalizeMenuPath(path)] = []EntryPermission{perm}
	}
	// 「服务与日志源」整合页：具备任一列表权限即可进入菜单
	out["/project-services"] = []EntryPermission{
		{Resource: "/api/v1/projects/:id/services", Action: "GET"},
		{Resource: "/api/v1/projects/:id/log-sources", Action: "GET"},
	}
	return out
}

// PermissionToMenuPaths 反向索引：permission key → 关联菜单 path。
func PermissionToMenuPaths() map[string][]string {
	rev := make(map[string][]string)
	for path, perms := range DefaultPathBindings() {
		for _, p := range perms {
			rev[p.Key()] = append(rev[p.Key()], path)
		}
	}
	return rev
}

// MenuLink 权限关联的菜单摘要。
type MenuLink struct {
	Path string `json:"path"`
	Name string `json:"name,omitempty"`
}
