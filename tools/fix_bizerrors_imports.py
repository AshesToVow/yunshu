#!/usr/bin/env python3
import os
import re

ROOT = os.path.join(os.path.dirname(__file__), "..", "internal")


def dedupe_imports(text: str) -> str:
    lines = text.split("\n")
    out = []
    in_import = False
    seen_biz = False
    for line in lines:
        if line.startswith("import ("):
            in_import = True
            seen_biz = False
            out.append(line)
            continue
        if in_import and line.strip() == ")":
            in_import = False
            out.append(line)
            continue
        if in_import and 'bizerrors "yunshu/internal/pkg/errors"' in line:
            if seen_biz:
                continue
            seen_biz = True
        out.append(line)
    return "\n".join(out)


def fix_internal_calls(text: str) -> str:
    # svcerr.Internal(ctx, comp, op, err, msgFmt) -> Internalf
    return re.sub(
        r"bizerrors\.Internal\((ctx, [^,]+, [^,]+, err,)",
        r"bizerrors.Internalf(\1",
        text,
    )


def main() -> None:
    for dirpath, _, files in os.walk(ROOT):
        for fn in files:
            if not fn.endswith(".go"):
                continue
            path = os.path.join(dirpath, fn)
            with open(path, encoding="utf-8") as f:
                text = f.read()
            new = fix_internal_calls(dedupe_imports(text))
            if new != text:
                with open(path, "w", encoding="utf-8", newline="\n") as f:
                    f.write(new)
                print(path)


if __name__ == "__main__":
    main()
