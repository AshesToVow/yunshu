#!/usr/bin/env python3
"""Split router deps_*.go: interfaces → internal/routedeps, getters stay in router."""
from __future__ import annotations

import re
from pathlib import Path

ROUTER = Path("internal/router")
OUT = Path("internal/routedeps")

# Files that only have getters (no interface to move)
GETTER_ONLY = {"deps_app.go"}

# Order matters for Bundle embedding (leaf interfaces first in files is fine)
BUNDLE_IFACES = [
    "CoreRouteDeps",
    "K8sRouteDeps",
    "AlertRouteDeps",
    "ProjectRouteDeps",
    "LogPlatformRouteDeps",
    "CMDBRouteDeps",
    "BackupRouteDeps",
    "CicdRouteDeps",
    "DbmgmtRouteDeps",
    "InspectRouteDeps",
    "AIRouteDeps",
    "EsmgmtRouteDeps",
]


def split_file(path: Path) -> tuple[str | None, str | None, set[str]]:
    """Return (iface_block_without_package, getter_body, imports_needed_for_iface)."""
    text = path.read_text(encoding="utf-8")
    # strip package and imports
    m = re.match(
        r"package router\s+(?:import\s+(?:\(([^)]*)\)|\"([^\"]+)\")\s*)?(.*)",
        text,
        re.S,
    )
    if not m:
        raise SystemExit(f"parse fail {path}")
    import_block = m.group(1) or (f'\t"{m.group(2)}"' if m.group(2) else "")
    body = m.group(3).strip() + "\n"

    iface_parts = []
    getter_parts = []
    for block in re.split(r"(?=\n(?:type |func |var ))", "\n" + body):
        block = block.strip()
        if not block:
            continue
        if block.startswith("type ") and "interface" in block.split("{", 1)[0]:
            iface_parts.append(block)
        else:
            getter_parts.append(block)

    imports = set()
    for line in import_block.splitlines():
        line = line.strip().strip('"')
        if not line:
            continue
        # handle alias
        if " " in line and not line.startswith('"'):
            # e.g. logx "yunshu/..."
            imports.add(line if line.startswith('"') or " " in line else f'"{line}"')
        else:
            imports.add(line if line.startswith('"') else f'"{line}"')

    # re-parse imports properly
    imports = set()
    raw = import_block.strip()
    if raw:
        for line in raw.splitlines():
            line = line.strip()
            if line:
                imports.add(line)

    iface_src = "\n\n".join(iface_parts) if iface_parts else None
    getter_src = "\n\n".join(getter_parts) if getter_parts else None
    return iface_src, getter_src, imports


def iface_imports(iface_src: str, all_imports: set[str]) -> set[str]:
    needed = set()
    for imp in all_imports:
        # extract path
        path = re.search(r'"([^"]+)"', imp)
        if not path:
            continue
        pkg = path.group(1).rsplit("/", 1)[-1]
        alias_m = re.match(r"(\w+)\s+", imp)
        name = alias_m.group(1) if alias_m else pkg
        if name == "logx":
            if "logx." in iface_src or "*logx." in iface_src:
                needed.add(imp)
        elif f"{name}." in iface_src or name == "gin" and "gin." in iface_src:
            needed.add(imp)
        elif pkg == "handler" and "handler." in iface_src:
            needed.add(imp)
        elif pkg.endswith("bootstrap") and "bootstrap." in iface_src:
            needed.add(imp)
        elif pkg.endswith("interfaces") and "interfaces." in iface_src:
            needed.add(imp)
        elif pkg.endswith("logger") and ("logx." in iface_src or "logger." in iface_src):
            needed.add(imp)
    return needed


def main() -> None:
    OUT.mkdir(parents=True, exist_ok=True)
    all_ifaces: list[tuple[str, str, set[str]]] = []  # name, src, imports
    alias_names: list[str] = []

    for path in sorted(ROUTER.glob("deps_*.go")):
        if path.name in GETTER_ONLY:
            continue
        iface_src, getter_src, imports = split_file(path)
        if iface_src:
            # find interface names
            names = re.findall(r"type (\w+) interface", iface_src)
            alias_names.extend(names)
            all_ifaces.append((path.stem, iface_src, iface_imports(iface_src, imports)))

            # write routedeps file
            imps = sorted(iface_imports(iface_src, imports))
            # embedded refs like RouteMiddleware don't need import
            imp_block = ""
            if imps:
                imp_block = "import (\n" + "\n".join(f"\t{i}" for i in imps) + "\n)\n\n"
            out = f"package routedeps\n\n{imp_block}{iface_src}\n"
            (OUT / f"{path.stem}.go").write_text(out, encoding="utf-8")
            print("iface", path.name, "→", OUT / f"{path.stem}.go")

        # rewrite router file to getters only + aliases assertion
        if getter_src is None and iface_src:
            # workflow-like: only interface, keep assertion file
            names = re.findall(r"type (\w+) interface", iface_src)
            asserts = "\n".join(f"var _ routedeps.{n} = (*RouteDeps)(nil)" for n in names)
            new_body = f'''package router

import "yunshu/internal/routedeps"

{asserts}
'''
            path.write_text(new_body, encoding="utf-8")
            print("assert-only", path.name)
            continue

        if getter_src:
            # detect imports needed for getters
            getter_imps = set()
            for imp in imports:
                path_m = re.search(r'"([^"]+)"', imp)
                if not path_m:
                    continue
                pkg = path_m.group(1).rsplit("/", 1)[-1]
                alias_m = re.match(r"(\w+)\s+", imp)
                name = alias_m.group(1) if alias_m else pkg
                if name == "logx" and "logx." in getter_src:
                    getter_imps.add(imp)
                elif f"{name}." in getter_src or (name == "gin" and "gin." in getter_src):
                    getter_imps.add(imp)
                elif "handler." in getter_src and "handler" in path_m.group(1):
                    getter_imps.add(imp)
                elif "bootstrap." in getter_src and "bootstrap" in path_m.group(1):
                    getter_imps.add(imp)
                elif "interfaces." in getter_src and "interfaces" in path_m.group(1):
                    getter_imps.add(imp)

            names = re.findall(r"type (\w+) interface", iface_src) if iface_src else []
            getter_imps.add('"yunshu/internal/routedeps"')
            asserts = "\n".join(f"var _ routedeps.{n} = (*RouteDeps)(nil)" for n in names)
            imps = sorted(getter_imps)
            if len(imps) == 1:
                imp_block = f"import {imps[0]}\n\n"
            else:
                imp_block = "import (\n" + "\n".join(f"\t{i}" for i in imps) + "\n)\n\n"
            new_body = f"package router\n\n{imp_block}{getter_src}\n"
            if asserts:
                new_body += f"\n{asserts}\n"
            path.write_text(new_body, encoding="utf-8")
            print("getters", path.name)

    # Bundle
    bundle = '''package routedeps

// Bundle 插件 HTTP 路由所需的完整依赖面（由 *router.RouteDeps 实现）。
// 放入独立包以打破 plugin ↔ router 循环，使 plugin.Runtime.Deps 可为具体接口。
type Bundle interface {
''' + "\n".join(f"\t{n}" for n in BUNDLE_IFACES) + "\n}\n"
    (OUT / "bundle.go").write_text(bundle, encoding="utf-8")
    print("wrote bundle.go")

    # Type aliases in router for Register* signatures
    aliases = '''package router

import "yunshu/internal/routedeps"

// 窄路由依赖类型别名（实现位于 routedeps，便于 Register* 签名稳定）。
type (
''' + "\n".join(f"\t{n} = routedeps.{n}" for n in sorted(set(alias_names))) + '''
\tBundle         = routedeps.Bundle
)

var _ routedeps.Bundle = (*RouteDeps)(nil)
'''
    (ROUTER / "deps_aliases.go").write_text(aliases, encoding="utf-8")
    print("wrote deps_aliases.go")


if __name__ == "__main__":
    main()
