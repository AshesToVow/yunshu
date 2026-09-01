export default [
  {
    path: '/user',
    layout: false,
    routes: [
      {
        name: '登录',
        path: '/user/login',
        component: './user/login',
      },
    ],
  },
  {
    path: '/workflow/inbox',
    component: './workflow/inbox',
  },
  {
    path: '/login-logs',
    component: './core/login-logs',
  },
  {
    path: '/operation-logs',
    component: './core/operation-logs',
  },
  {
    path: '/users',
    component: './system/users',
  },
  {
    path: '/departments',
    component: './system/departments',
  },
  {
    path: '/roles',
    component: './system/roles',
  },
  {
    path: '/banned-ips',
    component: './system/banned-ips',
  },
  {
    path: '/menus',
    component: './system/menus',
  },
  {
    path: '/permissions',
    component: './system/permissions',
  },
  {
    path: '/policies',
    component: './system/policies',
  },
  {
    path: '/cicd/release-records',
    component: './cicd/release-records',
  },
  {
    path: '/cicd/build-records',
    component: './cicd/build-records',
  },
  {
    path: '/cicd/services',
    component: './cicd/services',
  },
  {
    path: '/cicd/registries',
    component: './cicd/registries',
  },
  {
    path: '/cicd/image-browser',
    component: './cicd/image-browser',
  },
  {
    path: '/dbmgmt/instances',
    component: './dbmgmt/instances',
  },
  {
    path: '/dict-entries',
    component: './system/dict-entries',
  },
  {
    path: '/projects',
    component: './projects',
  },
  {
    path: '/dbmgmt/workflow/history',
    component: './dbmgmt/tickets',
  },
  {
    path: '/dbmgmt/grants',
    component: './dbmgmt/grants',
  },
  {
    path: '/dbmgmt/apply/query-grants',
    component: './dbmgmt/grants',
  },
  {
    path: '/esmgmt/connections',
    component: './esmgmt/connections',
  },
  {
    path: '/user-groups',
    component: './system/user-groups',
  },
  {
    path: '/registrations',
    component: './system/registrations',
  },
  {
    path: '/alert-channels',
    component: './alert/channels',
  },
  {
    path: '/alert-duty',
    component: './alert/duty',
  },
  {
    path: '/alert-maintenance',
    component: './alert/maintenance',
  },
  {
    path: '/ai/approvals',
    component: './ai/approvals',
  },
  {
    path: '/ai/investigations',
    component: './ai/investigations',
  },
  {
    path: '/personal-settings',
    component: './system/personal-settings',
  },
  {
    path: '/plugins',
    component: './system/plugins',
  },
  {
    path: '/dashboard',
    component: './dashboard',
  },
  {
    path: '/dbmgmt/audit',
    component: './dbmgmt/audit',
  },
  {
    path: '/alert-quality',
    component: './alert/quality',
  },
  {
    path: '/alert-events',
    component: './alert/events',
  },
  {
    path: '/workflow/definitions',
    component: './workflow/definitions',
  },
  {
    path: '/dbmgmt/access-requests/all',
    component: './dbmgmt/access-requests',
  },
  {
    path: '/dbmgmt/apply/query',
    component: './dbmgmt/apply-query',
  },
  {
    path: '/dbmgmt/apply/database',
    component: './dbmgmt/apply-database',
  },
  {
    path: '/dbmgmt/apply/app-user',
    component: './dbmgmt/apply-app-user',
  },
  {
    path: '/esmgmt/overview',
    component: './esmgmt/overview',
  },
  {
    path: '/ai/center',
    component: './ai/center',
  },
  {
    path: '/platform-templates',
    component: './platform/templates',
  },
  {
    path: '/workflow/tickets',
    component: './workflow/tickets',
  },
  { path: '/clusters', component: './k8s/clusters' },
  { path: '/pods', component: './k8s/pods' },
  { path: '/namespaces', component: './k8s/namespaces' },
  { path: '/nodes', component: './k8s/nodes' },
  { path: '/component-status', component: './k8s/component-status' },
  { path: '/cluster-api-resources', component: './k8s/cluster-api-resources' },
  { path: '/horizontal-pod-autoscalers', component: './k8s/hpa' },
  { path: '/k8s-resource-topology', component: './k8s/resource-topology' },
  { path: '/deployments', component: './k8s/deployments' },
  { path: '/statefulsets', component: './k8s/statefulsets' },
  { path: '/daemonsets', component: './k8s/daemonsets' },
  { path: '/cronjobs', component: './k8s/cronjobs' },
  { path: '/jobs', component: './k8s/jobs' },
  { path: '/configmaps', component: './k8s/configmaps' },
  { path: '/secrets', component: './k8s/secrets' },
  { path: '/ingresses', component: './k8s/ingresses' },
  { path: '/ingress-classes', component: './k8s/ingress-classes' },
  { path: '/events', component: './k8s/events' },
  { path: '/k8s-services', component: './k8s/services' },
  { path: '/persistentvolumes', component: './k8s/persistentvolumes' },
  { path: '/persistentvolumeclaims', component: './k8s/persistentvolumeclaims' },
  { path: '/storageclasses', component: './k8s/storageclasses' },
  { path: '/crds', component: './k8s/crds' },
  { path: '/crs', component: './k8s/crs' },
  { path: '/rbac/roles', component: './k8s/rbac/roles' },
  { path: '/rbac/rolebindings', component: './k8s/rbac/rolebindings' },
  { path: '/rbac/clusterroles', component: './k8s/rbac/clusterroles' },
  { path: '/rbac/clusterrolebindings', component: './k8s/rbac/clusterrolebindings' },
  { path: '/serviceaccounts', component: './k8s/serviceaccounts' },
  { path: '/network-policies', component: './k8s/network-policies' },
  { path: '/helm/charts', component: './k8s/helm/charts' },
  { path: '/helm/releases', component: './k8s/helm/releases' },
  { path: '/k8s-scoped-policies', component: './k8s/scoped-policies' },
  { path: '/k8s-cr-templates', component: './k8s/cr-templates' },
  { path: '/k8s/event-forward', component: './k8s/event-forward' },
  { path: '/project-servers', component: './project/servers' },
  { path: '/project-inspect', component: './project/inspect' },
  { path: '/project-members', component: './project/members' },
  { path: '/project-logs', component: './project/logs' },
  { path: '/project-services', component: './project/collect-config' },
  { path: '/project-log-sources', component: './project/collect-config' },
  { path: '/service-catalog', component: './project/service-catalog' },
  { path: '/service-portrait', component: './project/service-portrait' },
  { path: '/log-retention', component: './project/log-retention' },
  { path: '/loggie-status', component: './project/loggie-status' },
  { path: '/server-console', component: './project/server-console' },
  { path: '/mysql-backup', component: './ops/mysql-backup' },
  { path: '/dbmgmt/sql/query', component: './dbmgmt/sql-query' },
  { path: '/dbmgmt/console', component: './dbmgmt/sql-query' },
  { path: '/dbmgmt/sql/audit', component: './dbmgmt/sql-audit' },
  { path: '/dbmgmt/instances/:instanceId', component: './dbmgmt/instance-detail' },
  { path: '/esmgmt/console', component: './esmgmt/console' },
  { path: '/ai/assistant', component: './ai/assistant' },
  { path: '/alert-config-center', component: './alert/config-center' },
  {
    path: '/alert-monitor-platform',
    redirect: '/alert-monitor-platform/history',
  },
  {
    path: '/alert-monitor-platform/:tab',
    component: './alert/monitor',
  },
  { path: '/cicd/todo', component: './redirects/cicd-todo' },
  { path: '/dbmgmt/todo', component: './redirects/dbmgmt-todo' },
  { path: '/cicd/approval-flow', component: './redirects/cicd-approval-flow' },
  { path: '/dbmgmt/approval-flow', component: './redirects/dbmgmt-approval-flow' },
  { path: '/dbmgmt/workflow/pending', component: './redirects/dbmgmt-todo' },
  { path: '/dbmgmt/workflow/tickets/:ticketId', component: './dbmgmt/ticket-detail' },
  {
    path: '/*',
    component: './dynamic-menu',
  },
  {
    path: '*',
    layout: false,
    component: './exception/404',
  },
];
