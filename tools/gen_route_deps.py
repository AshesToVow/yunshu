#!/usr/bin/env python3
"""Generate narrow RouteDeps interfaces and rewrite register_* to use getters."""
from __future__ import annotations

import re
from pathlib import Path

ROOT = Path("internal/router")

# field_name -> (MethodName, GoTypeExpr)
# GoTypeExpr is relative to handler package or special
FIELD_META = {
    "adminHandler": ("AdminHandler", "*handler.AdminHandler"),
    "aiHandler": ("AIHandler", "*handler.AIHandler"),
    "alertHandler": ("AlertHandler", "*handler.AlertHandler"),
    "alertInhibitionHandler": ("AlertInhibitionHandler", "*handler.AlertInhibitionHandler"),
    "alertPlatformHandler": ("AlertPlatformHandler", "*handler.AlertPlatformHandler"),
    "alertReceiverGroupHandler": ("AlertReceiverGroupHandler", "*handler.AlertReceiverGroupHandler"),
    "alertSubscriptionHandler": ("AlertSubscriptionHandler", "*handler.AlertSubscriptionHandler"),
    "authHandler": ("AuthHandler", "*handler.AuthHandler"),
    "cicdHandler": ("CicdHandler", "*handler.CicdHandler"),
    "cloudExpiryRuleHandler": ("CloudExpiryRuleHandler", "*handler.CloudExpiryRuleHandler"),
    "clusterHandler": ("ClusterHandler", "*handler.ClusterHandler"),
    "clusterLogHandler": ("ClusterLogHandler", "*handler.ClusterLogHandler"),
    "cmdbHandler": ("CMDBHandler", "*handler.CMDBHandler"),
    "configHandler": ("ConfigHandler", "*handler.ConfigHandler"),
    "crHandler": ("CRHandler", "*handler.CRHandler"),
    "crdHandler": ("CRDHandler", "*handler.CRDHandler"),
    "dbmgmtHandler": ("DbmgmtHandler", "*handler.DbmgmtHandler"),
    "departmentHandler": ("DepartmentHandler", "*handler.DepartmentHandler"),
    "dictEntryHandler": ("DictEntryHandler", "*handler.DictEntryHandler"),
    "esmgmtHandler": ("EsmgmtHandler", "*handler.EsmgmtHandler"),
    "eventHandler": ("EventHandler", "*handler.EventHandler"),
    "helmHandler": ("HelmHandler", "*handler.HelmHandler"),
    "ingressHandler": ("IngressHandler", "*handler.IngressHandler"),
    "inspectHandler": ("InspectHandler", "*handler.InspectHandler"),
    "k8sDiscoveryHandler": ("K8sDiscoveryHandler", "*handler.K8sDiscoveryHandler"),
    "k8sEventForwardHandler": ("K8sEventForwardHandler", "*handler.K8sEventForwardHandler"),
    "k8sHPAHandler": ("K8sHPAHandler", "*handler.K8sHPAHandler"),
    "k8sNamespaceAllowHandler": ("K8sNamespaceAllowHandler", "*handler.K8sNamespaceAllowHandler"),
    "k8sNamespaceDenyHandler": ("K8sNamespaceDenyHandler", "*handler.K8sNamespaceDenyHandler"),
    "k8sResourceWatchHandler": ("K8sResourceWatchHandler", "*handler.K8sResourceWatchHandler"),
    "k8sScopedPolicyHandler": ("K8sScopedPolicyHandler", "*handler.K8sScopedPolicyHandler"),
    "k8sSearchHandler": ("K8sSearchHandler", "*handler.K8sSearchHandler"),
    "logPlatformHandler": ("LogPlatformHandler", "*handler.LogPlatformHandler"),
    "loggieHandler": ("LoggieHandler", "*handler.LoggieHandler"),
    "loginLogHandler": ("LoginLogHandler", "*handler.LoginLogHandler"),
    "menuHandler": ("MenuHandler", "*handler.MenuHandler"),
    "mysqlBackupHandler": ("MysqlBackupHandler", "*handler.MysqlBackupHandler"),
    "namespaceHandler": ("NamespaceHandler", "*handler.NamespaceHandler"),
    "networkPolicyHandler": ("NetworkPolicyHandler", "*handler.NetworkPolicyHandler"),
    "nodeHandler": ("NodeHandler", "*handler.NodeHandler"),
    "opLogHandler": ("OperationLogHandler", "*handler.OperationLogHandler"),
    "overviewHandler": ("OverviewHandler", "*handler.OverviewHandler"),
    "permissionHandler": ("PermissionHandler", "*handler.PermissionHandler"),
    "platformFeatures": ("PlatformFeaturesHandler", "*handler.PlatformFeaturesHandler"),
    "platformTplHandler": ("PlatformTemplateHandler", "*handler.PlatformTemplateHandler"),
    "pluginHandler": ("PluginHandler", "*handler.PluginHandler"),
    "podHandler": ("PodHandler", "*handler.PodHandler"),
    "policyHandler": ("PolicyHandler", "*handler.PolicyHandler"),
    "projectCatalogHandler": ("ProjectCatalogHandler", "*handler.ProjectCatalogHandler"),
    "projectHandler": ("ProjectHandler", "*handler.ProjectHandler"),
    "rbacHandler": ("RBACHandler", "*handler.RBACHandler"),
    "regHandler": ("RegistrationHandler", "*handler.RegistrationHandler"),
    "roleHandler": ("RoleHandler", "*handler.RoleHandler"),
    "serviceAccountHandler": ("ServiceAccountHandler", "*handler.ServiceAccountHandler"),
    "serviceResourceHandler": ("ServiceResourceHandler", "*handler.ServiceResourceHandler"),
    "storageHandler": ("StorageHandler", "*handler.StorageHandler"),
    "systemHandler": ("SystemHandler", "*handler.SystemHandler"),
    "userGroupHandler": ("UserGroupHandler", "*handler.UserGroupHandler"),
    "userHandler": ("UserHandler", "*handler.UserHandler"),
    "workflowHandler": ("WorkflowHandler", "*handler.WorkflowHandler"),
    "workloadHandler": ("WorkloadHandler", "*handler.WorkloadHandler"),
}

# Already have getters for some — skip regenerating those files' methods if in existing deps
EXISTING_GETTERS = {
    "AIHandler", "EsmgmtHandler", "InspectHandler", "DbmgmtHandler", "CicdHandler",
    "CMDBHandler", "AlertHandler", "AlertPlatformHandler", "AlertSubscriptionHandler",
    "AlertInhibitionHandler", "AlertReceiverGroupHandler", "CloudExpiryRuleHandler",
    "PlatformFeaturesHandler", "WorkflowHandler",
}

MW_REPLACES = [
    ("d.authMiddleware", "d.AuthMiddleware()"),
    ("d.wsAuthMiddleware", "d.WSAuthMiddleware()"),
    ("d.authorize", "d.Authorize()"),
    ("d.k8sScopeAuthorize", "d.K8sScopeAuthorize()"),
    ("d.opAudit", "d.OpAudit()"),
    ("d.projectMemberRepo", "d.ProjectMemberRepo()"),
    ("d.projectRepo", "d.ProjectRepo()"),
    ("d.app.Logger", "d.AppLogger()"),
    ("d.app.Redis", "d.App().Redis"),
    ("d.app.DB", "d.App().DB"),
    ("d.app.Engine", "d.App().Engine"),
    ("d.app.Config", "d.App().Config"),
]

SPECS = {
    "backup": {
        "iface": "BackupRouteDeps",
        "file": "deps_backup.go",
        "register": "register_backup_routes.go",
        "embed": ["RouteMiddleware", "ProjectAccessDeps"],
        "fields": ["mysqlBackupHandler"],
    },
    "log_platform": {
        "iface": "LogPlatformRouteDeps",
        "file": "deps_log_platform.go",
        "register": "register_log_platform_routes.go",
        "embed": ["RouteMiddleware"],
        "fields": ["logPlatformHandler", "loggieHandler"],
    },
    "project": {
        "iface": "ProjectRouteDeps",
        "file": "deps_project.go",
        "register": "register_project_routes.go",
        "embed": ["RouteMiddleware", "ProjectAccessDeps"],
        "fields": [
            "projectHandler", "projectCatalogHandler",
            "logPlatformHandler", "loggieHandler", "clusterLogHandler",
        ],
    },
    "workflow": {
        "iface": "WorkflowRouteDeps",
        "file": "deps_workflow.go",
        "register": "register_workflow_routes.go",
        "embed": ["RouteMiddleware"],
        "fields": ["workflowHandler"],
        "need_app": True,
    },
    "platform_tpl": {
        "iface": "PlatformTemplateRouteDeps",
        "file": "deps_platform_template.go",
        "register": "register_platform_template_routes.go",
        "embed": ["RouteMiddleware"],
        "fields": ["platformTplHandler"],
    },
    "core": {
        "iface": "CoreRouteDeps",
        "file": "deps_core.go",
        "register": "register_core_routes.go",
        "embed": ["RouteMiddleware"],
        "fields": [
            "systemHandler", "authHandler", "regHandler", "userHandler",
            "departmentHandler", "roleHandler", "userGroupHandler",
            "permissionHandler", "policyHandler", "adminHandler",
            "menuHandler", "dictEntryHandler", "loginLogHandler",
            "opLogHandler", "pluginHandler",
        ],
        "need_app": True,
    },
    "k8s": {
        "iface": "K8sRouteDeps",
        "file": "deps_k8s.go",
        "register": "register_k8s_routes.go",
        "embed": ["RouteMiddleware"],
        "fields": [
            "k8sScopedPolicyHandler", "k8sNamespaceDenyHandler", "k8sNamespaceAllowHandler",
            "clusterHandler", "k8sDiscoveryHandler", "k8sEventForwardHandler",
            "podHandler", "namespaceHandler", "nodeHandler", "workloadHandler",
            "configHandler", "storageHandler", "serviceResourceHandler",
            "ingressHandler", "networkPolicyHandler", "k8sHPAHandler",
            "helmHandler", "eventHandler", "crdHandler", "crHandler",
            "rbacHandler", "serviceAccountHandler", "overviewHandler",
            "k8sSearchHandler", "k8sResourceWatchHandler", "platformFeatures",
        ],
    },
}


def emit_getter(field: str) -> str:
    method, typ = FIELD_META[field]
    return f"""func (d *RouteDeps) {method}() {typ} {{
\tif d == nil {{
\t\treturn nil
\t}}
\treturn d.{field}
}}
"""


def write_deps(spec: dict) -> None:
    iface = spec["iface"]
    methods = []
    getters = []
    for f in spec["fields"]:
        method, typ = FIELD_META[f]
        methods.append(f"\t{method}() {typ}")
        if method not in EXISTING_GETTERS:
            getters.append(emit_getter(f))
    need_app = spec.get("need_app")
    embeds = "\n".join(f"\t{e}" for e in spec["embed"])
    extra_iface = ""
    extra_import = ""
    if need_app:
        extra_iface = "\tApp() *bootstrap.App\n"
        extra_import = '\n\t"yunshu/internal/bootstrap"\n'
        # App getter written once in deps_app.go
    body = f"""package router

import ({extra_import}
\t"yunshu/internal/handler"
)

// {iface} 插件/模块路由窄依赖。
type {iface} interface {{
{embeds}
{extra_iface}{chr(10).join(methods)}
}}

{chr(10).join(getters)}
var _ {iface} = (*RouteDeps)(nil)
"""
    # clean empty import lines
    body = body.replace("import (\n\n\t", "import (\n\t")
    if need_app:
        body = body.replace(
            'import (\n\t"yunshu/internal/bootstrap"\n\n\t"yunshu/internal/handler"\n)',
            'import (\n\t"yunshu/internal/bootstrap"\n\t"yunshu/internal/handler"\n)',
        )
    (ROOT / spec["file"]).write_text(body, encoding="utf-8")
    print("wrote", spec["file"])


def rewrite_register(spec: dict) -> None:
    path = ROOT / spec["register"]
    text = path.read_text(encoding="utf-8")
    # signature
    text = re.sub(
        rf"func Register\w+Routes\(api \*gin\.RouterGroup, d \*?RouteDeps\)",
        f"func RegisterPLACEHOLDER(api *gin.RouterGroup, d {spec['iface']})",
        text,
        count=1,
    )
    # restore Register name from original
    m = re.search(r"func (Register\w+Routes)", path.read_text(encoding="utf-8"))
    # re-read original name from file before we broke it - fix:
    orig = path.read_text(encoding="utf-8") if False else None
    # Fix: do signature replace properly
    text = path.read_text(encoding="utf-8")
    text = re.sub(
        r"func (Register\w+Routes)\(api \*gin\.RouterGroup, d \*?RouteDeps\)",
        rf"func \1(api *gin.RouterGroup, d {spec['iface']})",
        text,
        count=1,
    )
    for old, new in MW_REPLACES:
        text = text.replace(old, new)
    # handler fields — longest first to avoid partial issues
    for field in sorted(spec["fields"], key=len, reverse=True):
        method, _ = FIELD_META[field]
        text = text.replace(f"d.{field}", f"d.{method}()")
    # nil checks like d.WorkflowHandler() == nil already if field replaced
    # d.app == nil -> d.App() == nil
    text = text.replace("d.app == nil", "d.App() == nil")
    text = text.replace("d.app != nil", "d.App() != nil")
    # &d.App().Config.Plugins if was &d.app.Config.Plugins
    path.write_text(text, encoding="utf-8")
    print("rewrote", spec["register"])


def write_app_getter() -> None:
    p = ROOT / "deps_app.go"
    if p.exists():
        return
    p.write_text(
        '''package router

import "yunshu/internal/bootstrap"

func (d *RouteDeps) App() *bootstrap.App {
	if d == nil {
		return nil
	}
	return d.app
}
''',
        encoding="utf-8",
    )
    print("wrote deps_app.go")


def main() -> None:
    write_app_getter()
    for spec in SPECS.values():
        write_deps(spec)
        rewrite_register(spec)


if __name__ == "__main__":
    main()
