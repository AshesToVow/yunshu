#!/usr/bin/env python3
"""Replace svcerr imports/calls with bizerrors across internal/."""
import os
import re

ROOT = os.path.join(os.path.dirname(__file__), "..", "internal")


def fix_imports(text: str) -> str:
    if "svcerr" not in text and "yunshu/internal/service/svcerr" not in text:
        return text
    text = text.replace('"yunshu/internal/service/svcerr"', 'bizerrors "yunshu/internal/pkg/errors"')
    text = text.replace("yunshu/internal/service/svcerr", "yunshu/internal/pkg/errors")
    if "bizerrors" not in text:
        text = text.replace('"yunshu/internal/pkg/errors"', 'bizerrors "yunshu/internal/pkg/errors"', 1)
    text = re.sub(r"\bbizerrors bizerrors\b", "bizerrors", text)
    text = text.replace("svcerr.", "bizerrors.")
    return text


def main() -> None:
    for dirpath, _, files in os.walk(ROOT):
        for fn in files:
            if not fn.endswith(".go"):
                continue
            path = os.path.join(dirpath, fn)
            if "svcerr" in path:
                continue
            with open(path, encoding="utf-8") as f:
                orig = f.read()
            new = fix_imports(orig)
            if new != orig:
                with open(path, "w", encoding="utf-8", newline="\n") as f:
                    f.write(new)
                print(path)


if __name__ == "__main__":
    main()
