import pathlib, re, subprocess, os

root = pathlib.Path(r"d:/gocode/yunshu")
working = root / "web/src/pages/pod-page.tsx"
head_out = root / "web/src/pages/pod-page.HEAD.tsx"

# 1) working tree
b = working.read_bytes()
t = b.decode("utf-8")
print("working_cjk", len(re.findall(r"[\u4e00-\u9fff]", t)))
print("working_lines", t.count("\n") + 1)
print("working_bytes", len(b))
samples = [m.group(0) for m in re.finditer(r'"[^"]{0,40}"', t) if "?" in m.group(0)][:8]
print("working_sample", repr(samples))

# 2) HEAD via git show (binary-safe)
proc = subprocess.run(
    ["git", "-C", str(root), "show", "HEAD:web/src/pages/pod-page.tsx"],
    capture_output=True,
)
if proc.returncode != 0:
    print("git_show_error", proc.stderr.decode("utf-8", errors="replace"))
else:
    hb = proc.stdout
    head_out.write_bytes(hb)
    ht = hb.decode("utf-8")
    print("head_cjk", len(re.findall(r"[\u4e00-\u9fff]", ht)))
    print("head_lines", ht.count("\n") + 1)
    print("head_bytes", len(hb))
    hs = [m.group(0) for m in re.finditer(r'"[^"]{0,40}"', ht) if any("\u4e00" <= c <= "\u9fff" for c in m.group(0))][:5]
    hq = [m.group(0) for m in re.finditer(r'"[^"]{0,40}"', ht) if "?" in m.group(0)][:5]
    print("head_sample_cjk", repr(hs))
    print("head_sample_q", repr(hq))

# 3) git diff --stat and log
for args in (
    ["git", "-C", str(root), "diff", "--stat", "HEAD", "--", "web/src/pages/pod-page.tsx"],
    ["git", "-C", str(root), "log", "-1", "--", "web/src/pages/pod-page.tsx"],
):
    print("---", " ".join(args[3:]))
    r = subprocess.run(args, capture_output=True, text=True, encoding="utf-8", errors="replace")
    print(r.stdout or r.stderr)

# 4) backups
print("--- backups *pod-page*")
for p in root.rglob("*pod-page*"):
    try:
        print(f"{p} | {p.stat().st_size}")
    except OSError as e:
        print(p, e)

# 5) logs panel
lp = root / "web/src/components/pod/pod-logs-panel.tsx"
if lp.exists():
    lt = lp.read_text(encoding="utf-8")
    print("logs_panel_cjk", len(re.findall(r"[\u4e00-\u9fff]", lt)))
    print("logs_panel_bytes", lp.stat().st_size)
else:
    print("logs_panel_missing")
