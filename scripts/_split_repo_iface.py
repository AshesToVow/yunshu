from pathlib import Path
import re

src = Path("internal/repository/interfaces.go").read_text(encoding="utf-8")
header_end = src.find("// LogSourceRepo")
body = src[header_end:]
parts = re.split(r"(?=// \w+Repo is implemented)", body)
parts = [p for p in parts if p.strip()]
blocks = {}
for p in parts:
    m = re.match(r"// (\w+Repo) ", p)
    if not m:
        raise SystemExit(f"bad block start: {p[:80]!r}")
    blocks[m.group(1)] = p.strip() + "\n\n"
print("blocks", sorted(blocks))

groups = {
    "log_repo_iface.go": ["LogSourceRepo", "LogRetentionRepo", "LoggieAgentRepo"],
    "user_repo_iface.go": [
        "UserRepo",
        "UserGroupRepo",
        "DepartmentRepo",
        "RegistrationRequestRepo",
        "LoginLogRepo",
    ],
    "rbac_repo_iface.go": [
        "RoleRepo",
        "PermissionRepo",
        "MenuRepo",
        "OperationLogRepo",
        "DictEntryRepo",
    ],
    "project_repo_iface.go": ["ProjectRepo", "ProjectMemberRepo"],
    "cmdb_repo_iface.go": [
        "ServerRepo",
        "ServerGroupRepo",
        "CloudAccountRepo",
        "ServiceRepo",
    ],
    "k8s_repo_iface.go": [
        "K8sClusterRepo",
        "K8sClusterAccessRepo",
        "K8sNamespaceAllowRepo",
        "K8sNamespaceDenyRepo",
    ],
    "dbmgmt_repo_iface.go": ["DbmgmtRepo", "MysqlBackupRepo"],
}
imports = {
    "log_repo_iface.go": 'import (\n\t"context"\n\t"time"\n\n\t"yunshu/internal/model"\n)',
    "user_repo_iface.go": 'import (\n\t"context"\n\n\t"yunshu/internal/model"\n\n\t"gorm.io/gorm"\n)',
    "rbac_repo_iface.go": 'import (\n\t"context"\n\n\t"yunshu/internal/model"\n)',
    "project_repo_iface.go": 'import (\n\t"context"\n\n\t"yunshu/internal/model"\n)',
    "cmdb_repo_iface.go": 'import (\n\t"context"\n\n\t"yunshu/internal/model"\n)',
    "k8s_repo_iface.go": 'import (\n\t"context"\n\n\t"yunshu/internal/model"\n\t"yunshu/internal/pkg/k8sauth"\n)',
    "dbmgmt_repo_iface.go": 'import (\n\t"context"\n\t"time"\n\n\t"yunshu/internal/model"\n)',
}

out = Path("internal/repository")
for fname, names in groups.items():
    chunks = []
    for n in names:
        if n not in blocks:
            raise SystemExit(f"missing {n}")
        chunks.append(blocks.pop(n))
    text = "package repository\n\n" + imports[fname] + "\n\n" + "".join(chunks)
    (out / fname).write_text(text, encoding="utf-8", newline="\n")
    print("wrote", fname)

if blocks:
    raise SystemExit(f"leftover blocks: {sorted(blocks)}")

Path("internal/repository/interfaces.go").unlink()
print("deleted interfaces.go")
