from pathlib import Path
import re

text = Path("cmd/seed.go").read_text(encoding="utf-8")
m = re.search(r"func defaultPermissions\(\) \[\]model\.Permission \{\n\treturn \[\]model\.Permission\{(.*?)\n\t\}\n\}", text, re.S)
if not m:
    raise SystemExit("defaultPermissions not found")
body = m.group(1)
# each entry is one {Name: ...},
entries = re.findall(r"\{Name: .*?\},", body, re.S)
print("entries", len(entries))

def group_of(entry: str) -> str:
    rm = re.search(r'Resource: "([^"]+)"', entry)
    if not rm:
        return "platform"
    r = rm.group(1)
    if r.startswith("/api/v1/alerts"):
        return "alert"
    if r.startswith("/api/v1/ai"):
        return "ai"
    if r.startswith("/api/v1/esmgmt") or r.startswith("/api/v1/log-platform"):
        return "log"
    if r.startswith("/api/v1/projects"):
        # project + nested cicd/dbmgmt/inspect under projects
        if "/dbmgmt" in r or "/mysql-backup" in r:
            return "dbmgmt"
        if "/cicd" in r or "/jenkins" in r or "/pipeline" in r or "/releases" in r or "/builds" in r:
            return "cicd"
        if "/inspect" in r:
            return "inspect"
        return "project"
    k8s_prefixes = (
        "/api/v1/k8s",
        "/api/v1/clusters",
        "/api/v1/pods",
        "/api/v1/nodes",
        "/api/v1/namespaces",
        "/api/v1/deployments",
        "/api/v1/statefulsets",
        "/api/v1/daemonsets",
        "/api/v1/jobs",
        "/api/v1/cronjobs",
        "/api/v1/ingresses",
        "/api/v1/helm",
        "/api/v1/rbac",
        "/api/v1/serviceaccounts",
        "/api/v1/configmaps",
        "/api/v1/secrets",
        "/api/v1/k8s-services",
        "/api/v1/persistentvolumes",
        "/api/v1/persistentvolumeclaims",
        "/api/v1/storageclasses",
        "/api/v1/horizontal-pod-autoscalers",
        "/api/v1/network-policies",
        "/api/v1/crds",
        "/api/v1/crs",
        "/api/v1/k8s-cr-templates",
        "/api/v1/events",
        "/api/v1/k8s-policies",
        "/api/v1/k8s-namespace-",
    )
    if any(r.startswith(p) for p in k8s_prefixes):
        return "k8s"
    if r.startswith("/api/v1/registries") or r.startswith("/api/v1/pipeline-templates") or r.startswith("/api/v1/inspect"):
        return "cicd"
    return "system"

groups = {}
for e in entries:
    g = group_of(e)
    groups.setdefault(g, []).append(e.strip())

for g, items in sorted(groups.items()):
    print(g, len(items))

header = '''package cmd

import "yunshu/internal/model"

'''

def write_group(name: str, func: str, items: list[str]):
    lines = [
        "package cmd",
        "",
        'import "yunshu/internal/model"',
        "",
        f"func {func}() []model.Permission {{",
        "\treturn []model.Permission{",
    ]
    for it in items:
        lines.append("\t\t" + it)
    lines.extend(["\t}", "}", ""])
    Path(f"cmd/seed_permissions_{name}.go").write_text("\n".join(lines) + "\n", encoding="utf-8", newline="\n")
    print("wrote", name, len(items))

mapping = {
    "system": "seedPermissionsSystem",
    "k8s": "seedPermissionsK8s",
    "alert": "seedPermissionsAlert",
    "project": "seedPermissionsProject",
    "cicd": "seedPermissionsCicd",
    "dbmgmt": "seedPermissionsDbmgmt",
    "ai": "seedPermissionsAI",
    "log": "seedPermissionsLog",
    "inspect": "seedPermissionsInspect",
}

for name, func in mapping.items():
    write_group(name, func, groups.get(name, []))

# replace defaultPermissions in seed.go
new_fn = '''func defaultPermissions() []model.Permission {
	out := make([]model.Permission, 0, 700)
	out = append(out, seedPermissionsSystem()...)
	out = append(out, seedPermissionsK8s()...)
	out = append(out, seedPermissionsAlert()...)
	out = append(out, seedPermissionsProject()...)
	out = append(out, seedPermissionsCicd()...)
	out = append(out, seedPermissionsDbmgmt()...)
	out = append(out, seedPermissionsAI()...)
	out = append(out, seedPermissionsLog()...)
	out = append(out, seedPermissionsInspect()...)
	return out
}'''

text2 = re.sub(
    r"func defaultPermissions\(\) \[\]model\.Permission \{.*?\n\}",
    new_fn,
    text,
    count=1,
    flags=re.S,
)
if text2 == text:
    raise SystemExit("failed to replace defaultPermissions")
Path("cmd/seed.go").write_text(text2, encoding="utf-8", newline="\n")
print("updated seed.go, lines", text2.count("\n")+1)
