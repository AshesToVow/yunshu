#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""只读内存概况。兼容 Python 2.7 / 3.x。探测运行环境本机，非远端主机。"""
from __future__ import print_function
import json
import os
import sys

def read_meminfo():
    path = "/proc/meminfo"
    if not os.path.exists(path):
        return None
    data = {}
    try:
        with open(path, "r") as f:
            for line in f:
                parts = line.split(":")
                if len(parts) < 2:
                    continue
                key = parts[0].strip()
                val = parts[1].strip().split()[0]
                try:
                    data[key] = int(val) * 1024  # kB -> bytes
                except Exception:
                    pass
    except Exception:
        return None
    return data

def main():
    out = {"ok": True, "platform": sys.platform, "note": "linux.mem.check probes local runtime host"}
    info = read_meminfo()
    if info:
        total = info.get("MemTotal", 0)
        avail = info.get("MemAvailable", info.get("MemFree", 0))
        out["total_bytes"] = total
        out["available_bytes"] = avail
        if total > 0:
            out["used_ratio"] = round(1.0 - float(avail) / float(total), 4)
    else:
        out["meminfo"] = "unavailable"
        out["hint"] = "non-Linux or /proc/meminfo missing"
    print(json.dumps(out, ensure_ascii=False))

if __name__ == "__main__":
    main()
