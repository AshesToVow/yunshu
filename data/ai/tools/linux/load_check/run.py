#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""只读负载概况。兼容 Python 2.7 / 3.x。探测运行环境本机。"""
from __future__ import print_function
import json
import os
import sys

def main():
    out = {"ok": True, "platform": sys.platform, "note": "linux.load.check probes local runtime host"}
    try:
        load1, load5, load15 = os.getloadavg()
        out["load1"] = round(float(load1), 4)
        out["load5"] = round(float(load5), 4)
        out["load15"] = round(float(load15), 4)
    except Exception as e:
        out["loadavg_error"] = str(e)
    try:
        out["cpu_count"] = int(os.sysconf("SC_NPROCESSORS_ONLN"))
    except Exception:
        try:
            import multiprocessing
            out["cpu_count"] = int(multiprocessing.cpu_count())
        except Exception:
            out["cpu_count"] = None
    print(json.dumps(out, ensure_ascii=False))

if __name__ == "__main__":
    main()
