#!/usr/bin/env python3
import subprocess
import pathlib

raw = subprocess.check_output(
    ["git", "show", "HEAD:internal/service/dynamic_resource_service.go"]
)
text = raw.decode("utf-8")
text = text.replace("package service", "package k8s", 1)
text = text.replace('"yunshu/internal/service/svcerr"', 'bizerrors "yunshu/internal/pkg/errors"')
text = text.replace("svcerr.", "bizerrors.")
path = pathlib.Path("internal/service/k8s/dynamic_resource_service.go")
path.write_text(text, encoding="utf-8")
print("ok", path)
