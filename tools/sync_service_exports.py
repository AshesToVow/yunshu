#!/usr/bin/env python3
"""Add missing type aliases to internal/service/exports.go from handler references."""
import re
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
EXPORTS = ROOT / "internal/service/exports.go"
PKGS = {
    "system": "yunshu/internal/service/system",
    "project": "yunshu/internal/service/project",
    "k8s": "yunshu/internal/service/k8s",
    "eventforward": "yunshu/internal/service/k8s/eventforward",
    "logplatform": "yunshu/internal/service/logplatform",
    "mysqlbackup": "yunshu/internal/service/mysqlbackup",
    "alert": "yunshu/internal/service/alert",
    "overview": "yunshu/internal/service/overview",
}


def exported_symbols(pkg_import: str) -> set[str]:
    out = subprocess.run(
        ["go", "doc", "-all", pkg_import],
        cwd=ROOT,
        capture_output=True,
        text=True,
        errors="replace",
    )
    text = out.stdout + out.stderr
    names = set()
    for kind in ("type", "func", "const", "var"):
        for m in re.finditer(rf"^{kind} (\w+)", text, re.MULTILINE):
            names.add(m.group(1))
    return names


def referenced_symbols() -> set[str]:
    names = set()
    for pat in [
        ROOT / "internal/handler",
        ROOT / "internal/middleware",
        ROOT / "internal/grpc",
        ROOT / "internal/router",
    ]:
        if not pat.exists():
            continue
        for f in pat.rglob("*.go"):
            text = f.read_text(encoding="utf-8", errors="replace")
            for m in re.finditer(r"service\.([A-Z][A-Za-z0-9]+)", text):
                names.add(m.group(1))
    return names


def find_pkg(name: str, cache: dict[str, set[str]]) -> str | None:
    for pkg, syms in cache.items():
        if name in syms:
            return pkg
    return None


def main():
    cache = {pkg: exported_symbols(imp) for pkg, imp in PKGS.items()}
    refs = referenced_symbols()
    text = EXPORTS.read_text(encoding="utf-8")
    existing = set(re.findall(r"\b([A-Z][A-Za-z0-9]+)\s*=", text))
    missing = sorted(refs - existing)
    lines = []
    by_pkg: dict[str, list[str]] = {}
    for n in missing:
        pkg = find_pkg(n, cache)
        if not pkg:
            print("SKIP (not found):", n)
            continue
        by_pkg.setdefault(pkg, []).append(f"\t{n} = {pkg}.{n}")
    if not by_pkg:
        print("nothing to add")
        return
    block = "\n\n// --- auto-synced DTO aliases ---\ntype (\n"
    for pkg in ("k8s", "eventforward", "alert", "mysqlbackup", "system", "project", "logplatform", "overview"):
        if pkg in by_pkg:
            block += "\n".join(sorted(by_pkg[pkg])) + "\n"
    block += ")\n"
    if "// --- auto-synced DTO aliases ---" in text:
        text = re.sub(
            r"\n// --- auto-synced DTO aliases ---.*?\n\)\n",
            "\n" + block,
            text,
            flags=re.DOTALL,
        )
    else:
        text = text.rstrip() + "\n" + block
    EXPORTS.write_text(text, encoding="utf-8")
    print("added", sum(len(v) for v in by_pkg.values()), "aliases")


if __name__ == "__main__":
    main()
