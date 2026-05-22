#!/usr/bin/env python3
"""Move internal/service files into domain subpackages and generate root facades."""
from __future__ import annotations

import os
import re
import shutil

ROOT = os.path.join(os.path.dirname(__file__), "..", "internal", "service")

# target_dir -> list of filename prefixes or exact names (root only)
DOMAINS: dict[str, list[str]] = {
    "overview": ["overview_"],
    "mysqlbackup": ["mysql_backup"],
    "logplatform": ["log_agent_", "agent_discovery_"],
    "system": [
        "auth_", "user_", "role_", "permission_", "policy_", "menu_",
        "department_", "dict_entry_", "login_log_", "operation_log_",
        "registration_", "user_group_", "casbin_sync", "dto.go",
    ],
    "project": ["project_mgmt_", "cloud_provider_", "cloud_expiry_"],
    "k8s": ["k8s_"],
    "alert": ["alert_"],
}

# k8s excludes event forward admin (goes to k8s/eventforward)
K8S_EXCLUDE = {"k8s_event_forward_admin_service.go"}

PACKAGE_NAMES = {
    "overview": "overview",
    "mysqlbackup": "mysqlbackup",
    "logplatform": "logplatform",
    "system": "system",
    "project": "project",
    "k8s": "k8s",
    "k8s/eventforward": "eventforward",
    "alert": "alert",
}


def match_domain(filename: str) -> str | None:
    if filename in K8S_EXCLUDE:
        return "k8s/eventforward"
    for domain, patterns in DOMAINS.items():
        if domain == "k8s" and not any(filename.startswith(p) for p in patterns):
            continue
        if domain == "k8s" and any(filename.startswith(p) for p in patterns):
            return domain
        for p in patterns:
            if filename == p or filename.startswith(p):
                if domain == "k8s":
                    return domain
                return domain
    return None


def set_package(path: str, pkg: str) -> None:
    with open(path, encoding="utf-8") as f:
        text = f.read()
    text = re.sub(r"^package\s+\w+", f"package {pkg}", text, count=1, flags=re.M)
    with open(path, "w", encoding="utf-8", newline="\n") as f:
        f.write(text)


def move_root_files() -> dict[str, list[str]]:
    moved: dict[str, list[str]] = {}
    for fn in os.listdir(ROOT):
        if not fn.endswith(".go"):
            continue
        src = os.path.join(ROOT, fn)
        if not os.path.isfile(src):
            continue
        domain = match_domain(fn)
        if not domain:
            continue
        dest_dir = os.path.join(ROOT, domain.replace("/", os.sep))
        os.makedirs(dest_dir, exist_ok=True)
        dest = os.path.join(dest_dir, fn)
        if os.path.exists(dest):
            os.remove(dest)
        shutil.move(src, dest)
        pkg = PACKAGE_NAMES[domain]
        set_package(dest, pkg)
        moved.setdefault(domain, []).append(fn)
    return moved


def move_k8seventforward() -> None:
    src_dir = os.path.join(ROOT, "k8seventforward")
    if not os.path.isdir(src_dir):
        return
    dest_dir = os.path.join(ROOT, "k8s", "eventforward")
    os.makedirs(dest_dir, exist_ok=True)
    for fn in os.listdir(src_dir):
        if not fn.endswith(".go"):
            continue
        src = os.path.join(src_dir, fn)
        dest = os.path.join(dest_dir, fn)
        if os.path.exists(dest):
            os.remove(dest)
        shutil.move(src, dest)
        set_package(dest, "eventforward")
    try:
        os.rmdir(src_dir)
    except OSError:
        pass


def main() -> None:
    moved = move_root_files()
    move_k8seventforward()
    for domain, files in sorted(moved.items()):
        print(f"{domain}: {len(files)} files")
    print("done")


if __name__ == "__main__":
    main()
