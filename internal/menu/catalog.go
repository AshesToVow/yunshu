package menu

// DefaultCatalog 平台内置菜单树（单一数据源；seed 与升级同步均使用此定义）。
func DefaultCatalog() []Spec {
	return []Spec{
		{
			Path: "/", Name: "总览页面", Icon: "PieChartOutlined", Sort: 1, Component: "dashboard-page", Status: 1,
		},
		{
			Path: "/alert-notify", Name: "告警通知", Icon: "BellOutlined", Sort: 2, Status: 1,
			Children: []Spec{
				{Path: "/alert-channels", Name: "Webhook 告警通道", Icon: "NotificationOutlined", Sort: 1, Component: "alert-channels-page", Status: 1},
				{Path: "/alert-monitor-platform", Name: "告警监控平台", Icon: "MonitorOutlined", Sort: 2, Component: "alert-monitor-platform-page", Status: 1},
				{Path: "/alert-duty", Name: "值班总览", Icon: "ScheduleOutlined", Sort: 3, Component: "alert-duty-page", Status: 1},
				{Path: "/alert-maintenance", Name: "维护窗口", Icon: "CalendarOutlined", Sort: 4, Component: "alert-maintenance-page", Status: 1},
			},
		},
		{
			Path: "/project-management", Name: "项目管理", Icon: "ProjectOutlined", Sort: 3, Status: 1,
			Children: []Spec{
				{Path: "/projects", Name: "项目列表", Icon: "AppstoreOutlined", Sort: 1, Component: "projects-page", Status: 1},
				{Path: "/project-members", Name: "项目成员", Icon: "TeamOutlined", Sort: 2, Component: "project-members-page", Status: 1},
				{Path: "/project-servers", Name: "服务器管理", Icon: "HddOutlined", Sort: 3, Component: "project-servers-page", Status: 1},
			},
		},
		{
			Path: "/log-platform", Name: "日志平台", Icon: "FileTextOutlined", Sort: 4, Status: 1,
			Children: []Spec{
				{Path: "/project-services", Name: "服务与日志源", Icon: "SettingOutlined", Sort: 1, Component: "project-collect-config-page", Status: 1},
				{Path: "/project-logs", Name: "日志检索", Icon: "FileTextOutlined", Sort: 2, Component: "project-logs-page", Status: 1},
				{Path: "/log-retention", Name: "保留策略", Icon: "HistoryOutlined", Sort: 3, Component: "log-retention-page", Status: 1},
				{Path: "/loggie-status", Name: "Agent 管理", Icon: "CloudServerOutlined", Sort: 4, Component: "loggie-status-page", Status: 1},
			},
		},
		{
			Path: "/dbmgmt", Name: "数据库管理", Icon: "DatabaseOutlined", Sort: 5, Status: 1,
			Children: []Spec{
				{
					Path: "/dbmgmt/apply", Name: "资源申请", Icon: "FormOutlined", Sort: 1, Status: 1,
					Children: []Spec{
						{Path: "/dbmgmt/apply/database", Name: "数据库创建申请", Icon: "PlusCircleOutlined", Sort: 1, Component: "dbmgmt-database-apply-page", Status: 1},
						{Path: "/dbmgmt/apply/query", Name: "平台查询权限申请", Icon: "SearchOutlined", Sort: 2, Component: "dbmgmt-query-apply-page", Status: 1},
						{Path: "/dbmgmt/apply/app-user", Name: "应用用户权限申请", Icon: "UserAddOutlined", Sort: 3, Component: "dbmgmt-app-user-apply-page", Status: 1},
						{Path: "/dbmgmt/apply/query-grants", Name: "查询权限管理", Icon: "SafetyOutlined", Sort: 4, Component: "dbmgmt-query-grants-page", Status: 1},
					},
				},
				{
					Path: "/dbmgmt/resource", Name: "资源管理", Icon: "DatabaseOutlined", Sort: 2, Status: 1,
					Children: []Spec{
						{Path: "/dbmgmt/instances", Name: "实例管理", Icon: "DatabaseOutlined", Sort: 1, Component: "dbmgmt-instances-page", Status: 1},
					},
				},
				{
					Path: "/dbmgmt/sql", Name: "SQL 操作", Icon: "CodeOutlined", Sort: 3, Status: 1,
					Children: []Spec{
						{Path: "/dbmgmt/sql/query", Name: "SQL 查询", Icon: "SearchOutlined", Sort: 1, Component: "dbmgmt-sql-query-page", Status: 1},
						{Path: "/dbmgmt/sql/audit", Name: "SQL 审核", Icon: "AuditOutlined", Sort: 2, Component: "dbmgmt-sql-audit-page", Status: 1},
					},
				},
				{
					Path: "/dbmgmt/workflow", Name: "工单管理", Icon: "FileTextOutlined", Sort: 4, Status: 1,
					Children: []Spec{
						{Path: "/dbmgmt/workflow/pending", Name: "待审核", Icon: "UnorderedListOutlined", Sort: 1, Component: "dbmgmt-workflow-pending-page", Status: 1},
						{Path: "/dbmgmt/workflow/history", Name: "历史工单", Icon: "HistoryOutlined", Sort: 2, Component: "dbmgmt-workflow-history-page", Status: 1},
						{Path: "/dbmgmt/approval-flow", Name: "审批流配置", Icon: "SettingOutlined", Sort: 3, Component: "dbmgmt-approval-flow-page", Status: 1},
					},
				},
				{Path: "/dbmgmt/audit", Name: "审计日志", Icon: "HistoryOutlined", Sort: 5, Component: "dbmgmt-audit-page", Status: 1},
				{Path: "/dbmgmt/grants", Name: "授权管理", Icon: "SafetyOutlined", Sort: 6, Component: "dbmgmt-grants-page", Status: 1, Hidden: true},
				{Path: "/mysql-backup", Name: "MySQL 备份", Icon: "CloudServerOutlined", Sort: 7, Component: "mysql-backup-page", Status: 1},
			},
		},
		{
			Path: "/system", Name: "系统管理", Icon: "SettingOutlined", Sort: 6, Status: 1,
			Children: []Spec{
				{Path: "/users", Name: "用户管理", Icon: "TeamOutlined", Sort: 1, Component: "users-page", Status: 1},
				{Path: "/user-groups", Name: "用户组管理", Icon: "UserOutlined", Sort: 2, Component: "user-groups-page", Status: 1},
				{Path: "/departments", Name: "组织架构", Icon: "ApartmentOutlined", Sort: 3, Component: "departments-page", Status: 1},
				{Path: "/roles", Name: "角色模板", Icon: "ApartmentOutlined", Sort: 4, Component: "roles-page", Status: 1},
				{Path: "/permissions", Name: "API管理", Icon: "ApiOutlined", Sort: 5, Component: "permissions-page", Status: 1},
				{Path: "/policies", Name: "授权管理", Icon: "AuditOutlined", Sort: 6, Component: "policies-page", Status: 1},
				{Path: "/k8s-scoped-policies", Name: "K8s 集群访问档位", Icon: "AuditOutlined", Sort: 7, Component: "k8s-scoped-policies-page", Status: 1},
				{Path: "/registrations", Name: "注册审核", Icon: "CheckCircleOutlined", Sort: 8, Component: "registrations-page", Status: 1},
				{Path: "/menus", Name: "菜单管理", Icon: "MenuOutlined", Sort: 9, Component: "menus-page", Status: 1},
				{Path: "/dict-entries", Name: "数据字典", Icon: "DatabaseOutlined", Sort: 10, Component: "dict-entries-page", Status: 1},
				{Path: "/login-logs", Name: "登录日志", Icon: "LoginOutlined", Sort: 11, Component: "login-logs-page", Status: 1},
				{Path: "/operation-logs", Name: "操作历史", Icon: "HistoryOutlined", Sort: 12, Component: "operation-logs-page", Status: 1},
				{Path: "/banned-ips", Name: "封禁 IP 管理", Icon: "ApiOutlined", Sort: 13, Component: "banned-ips-page", Status: 1},
			},
		},
		{
			Path: "/kubernetes", Name: "Kubernetes 容器管理", Icon: "KubernetesOutlined", Sort: 7, Status: 1,
			Children: []Spec{
				{Path: "/clusters", Name: "集群管理", Icon: "KubernetesOutlined", Sort: 1, Component: "cluster-page", Status: 1},
				{Path: "/namespaces", Name: "命名空间管理", Icon: "AppstoreOutlined", Sort: 2, Component: "namespaces-page", Status: 1},
				{Path: "/nodes", Name: "Node 管理", Icon: "HddOutlined", Sort: 3, Component: "nodes-page", Status: 1},
				{Path: "/component-status", Name: "组件状态", Icon: "HeartOutlined", Sort: 4, Component: "component-status-page", Status: 1},
				{Path: "/pods", Name: "Pod 管理", Icon: "KubernetesOutlined", Sort: 5, Component: "pod-page", Status: 1},
				{Path: "/deployments", Name: "Deployment 管理", Icon: "DeploymentUnitOutlined", Sort: 6, Component: "deployments-page", Status: 1},
				{Path: "/statefulsets", Name: "StatefulSet 管理", Icon: "ClusterOutlined", Sort: 7, Component: "statefulsets-page", Status: 1},
				{Path: "/daemonsets", Name: "DaemonSet 管理", Icon: "ApiOutlined", Sort: 8, Component: "daemonsets-page", Status: 1},
				{Path: "/cronjobs", Name: "CronJob 管理", Icon: "ClockCircleOutlined", Sort: 9, Component: "cronjobs-page", Status: 1},
				{Path: "/jobs", Name: "Job 管理", Icon: "ThunderboltOutlined", Sort: 10, Component: "jobs-page", Status: 1},
				{Path: "/configmaps", Name: "ConfigMap 管理", Icon: "FileTextOutlined", Sort: 11, Component: "configmaps-page", Status: 1},
				{Path: "/secrets", Name: "Secret 管理", Icon: "SafetyOutlined", Sort: 12, Component: "secrets-page", Status: 1},
				{Path: "/k8s-services", Name: "Service 管理", Icon: "ApartmentOutlined", Sort: 13, Component: "k8s-services-page", Status: 1},
				{Path: "/persistentvolumes", Name: "PersistentVolume", Icon: "DatabaseOutlined", Sort: 14, Component: "persistentvolumes-page", Status: 1},
				{Path: "/persistentvolumeclaims", Name: "PersistentVolumeClaim", Icon: "HddOutlined", Sort: 15, Component: "persistentvolumeclaims-page", Status: 1},
				{Path: "/storageclasses", Name: "StorageClass", Icon: "FolderOpenOutlined", Sort: 16, Component: "storageclasses-page", Status: 1},
				{Path: "/ingresses", Name: "Ingress 管理", Icon: "GatewayOutlined", Sort: 17, Component: "ingresses-page", Status: 1},
				{Path: "/ingress-classes", Name: "IngressClass 入口类", Icon: "GatewayOutlined", Sort: 18, Component: "ingress-classes-page", Status: 1},
				{Path: "/network-policies", Name: "网络策略管理", Icon: "DeploymentUnitOutlined", Sort: 19, Component: "network-policies-page", Status: 1},
				{Path: "/rbac/roles", Name: "RBAC - Role", Icon: "SafetyCertificateOutlined", Sort: 20, Component: "rbac-roles-page", Status: 1},
				{Path: "/rbac/rolebindings", Name: "RBAC - RoleBinding", Icon: "SafetyCertificateOutlined", Sort: 21, Component: "rbac-rolebindings-page", Status: 1},
				{Path: "/rbac/clusterroles", Name: "RBAC - ClusterRole", Icon: "SafetyCertificateOutlined", Sort: 22, Component: "rbac-clusterroles-page", Status: 1},
				{Path: "/rbac/clusterrolebindings", Name: "RBAC - ClusterRoleBinding", Icon: "SafetyCertificateOutlined", Sort: 23, Component: "rbac-clusterrolebindings-page", Status: 1},
				{Path: "/events", Name: "Event 事件", Icon: "FileSearchOutlined", Sort: 24, Component: "events-page", Status: 1},
				{Path: "/k8s/event-forward", Name: "K8s Event 转发", Icon: "ShareAltOutlined", Sort: 25, Component: "k8s-event-forward-page", Status: 1},
				{Path: "/serviceaccounts", Name: "ServiceAccount 管理", Icon: "SafetyCertificateOutlined", Sort: 26, Component: "serviceaccounts-page", Status: 1},
				{Path: "/cluster-api-resources", Name: "API 资源发现", Icon: "UnorderedListOutlined", Sort: 27, Component: "cluster-api-resources-page", Status: 1},
				{Path: "/horizontal-pod-autoscalers", Name: "HPA 弹性伸缩", Icon: "LineChartOutlined", Sort: 28, Component: "horizontal-pod-autoscalers-page", Status: 1},
				{Path: "/helm/releases", Name: "Helm Release", Icon: "RocketOutlined", Sort: 29, Component: "helm-releases-page", Status: 1},
				{Path: "/helm/charts", Name: "Harbor Chart", Icon: "CloudServerOutlined", Sort: 30, Component: "helm-charts-page", Status: 1},
				{Path: "/k8s-resource-topology", Name: "资源拓扑图", Icon: "DeploymentUnitOutlined", Sort: 31, Component: "k8s-resource-topology-page", Status: 1},
			},
		},
		{
			Path: "/cicd", Name: "CI/CD", Icon: "RocketOutlined", Sort: 8, Status: 1,
			Children: []Spec{
				{Path: "/cicd/services", Name: "应用服务", Icon: "AppstoreOutlined", Sort: 1, Component: "cicd-services-page", Status: 1},
				{Path: "/cicd/todo", Name: "待办列表", Icon: "UnorderedListOutlined", Sort: 2, Component: "cicd-todo-page", Status: 1},
				{Path: "/cicd/approval-flow", Name: "审批管理", Icon: "AuditOutlined", Sort: 3, Component: "cicd-approval-flow-page", Status: 1},
				{Path: "/cicd/build-records", Name: "CI 打包记录", Icon: "CloudUploadOutlined", Sort: 4, Component: "cicd-build-records-page", Status: 1},
				{Path: "/cicd/release-records", Name: "CD 历史工单", Icon: "DeploymentUnitOutlined", Sort: 5, Component: "cicd-release-records-page", Status: 1},
			},
		},
		{
			Path: "/kubernetes-crd", Name: "Kubernetes CRD 管理", Icon: "BranchesOutlined", Sort: 9, Status: 1,
			Children: []Spec{
				{Path: "/crds", Name: "CRD 管理", Icon: "BranchesOutlined", Sort: 1, Component: "crds-page", Status: 1},
				{Path: "/crs", Name: "CR 实例管理", Icon: "DatabaseOutlined", Sort: 2, Component: "crs-page", Status: 1},
			},
		},
	}
}
