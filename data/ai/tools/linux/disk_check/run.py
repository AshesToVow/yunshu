#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""只读磁盘/路径用量。兼容 Python 2.7 / 3.x。探测运行环境本机，非任意远端主机。"""
from __future__ import print_function
import json
import os
import sys

def main():
    raw = sys.stdin.read()
    try:
        args = json.loads(raw or "{}")
    except Exception:
        args = {}
    path = args.get("path") or ("C:\\" if sys.platform.startswith("win") else "/")
    out = {
        "ok": True,
        "path": path,
        "platform": sys.platform,
        "note": "linux.disk.check probes local runtime host; use server console for remote hosts",
    }
    if hasattr(os, "statvfs"):
        try:
            st = os.statvfs(path)
            total = float(st.f_blocks) * float(st.f_frsize)
            free = float(st.f_bfree) * float(st.f_frsize)
            out["total_bytes"] = int(total)
            out["free_bytes"] = int(free)
            if total > 0:
                out["used_ratio"] = round(1.0 - free / total, 4)
        except Exception as e:
            out["statvfs_error"] = str(e)
    else:
        out["cwd"] = os.getcwd()
        out["hint"] = "statvfs unavailable on this platform"
    print(json.dumps(out, ensure_ascii=False))

if __name__ == "__main__":
    main()
