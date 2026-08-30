# -*- coding: utf-8 -*-
"""Improved: ASCII-skeleton encoding detection; build RESTORED from HEAD + real only."""
from __future__ import annotations

import difflib
import pathlib
import re

ROOT = pathlib.Path(r"d:/gocode/yunshu")
WORKING = ROOT / "web/src/pages/pod-page.tsx"
HEAD = ROOT / "web/src/pages/pod-page.HEAD.tsx"
RESTORED = ROOT / "web/src/pages/pod-page.RESTORED.tsx"

CJK_RE = re.compile(r"[\u4e00-\u9fff]")


def ascii_skeleton(line: str) -> str:
    """Drop non-ASCII and '?' (CJK / CJK-punct corruption); keep other ASCII."""
    out = []
    for ch in line.rstrip("\n"):
        o = ord(ch)
        if ch == "?":
            continue
        if o < 128:
            out.append(ch)
    # collapse spaces left by removals
    return re.sub(r"[ \t]+", " ", "".join(out)).strip()


def is_encoding_only_pair(head_line: str, work_line: str) -> bool:
    return ascii_skeleton(head_line) == ascii_skeleton(work_line)


def classify_hunk(head_lines: list[str], work_lines: list[str]) -> str:
    if not head_lines or not work_lines:
        return "real"
    if len(head_lines) == len(work_lines):
        flags = [is_encoding_only_pair(h, w) for h, w in zip(head_lines, work_lines)]
        if all(flags):
            return "encoding"
        if any(flags):
            return "mixed"
        return "real"
    h_sk = [ascii_skeleton(h) for h in head_lines]
    w_sk = [ascii_skeleton(w) for w in work_lines]
    sm = difflib.SequenceMatcher(a=h_sk, b=w_sk, autojunk=False)
    saw_non_enc = False
    for tag, i1, i2, j1, j2 in sm.get_opcodes():
        if tag == "equal":
            continue
        if tag == "replace" and (i2 - i1) == (j2 - j1):
            for hi, wi in zip(range(i1, i2), range(j1, j2)):
                if not is_encoding_only_pair(head_lines[hi], work_lines[wi]):
                    saw_non_enc = True
        else:
            saw_non_enc = True
    return "real" if saw_non_enc else "encoding"


def unicode_preview(s: str, limit: int = 120) -> str:
    s = s.rstrip()
    if len(s) > limit:
        s = s[:limit] + "..."
    return s.encode("unicode_escape").decode("ascii")


def main() -> None:
    head_text = HEAD.read_text(encoding="utf-8")
    work_text = WORKING.read_text(encoding="utf-8")
    head_lines = head_text.splitlines(keepends=True)
    work_lines = work_text.splitlines(keepends=True)

    sm = difflib.SequenceMatcher(a=head_lines, b=work_lines, autojunk=False)
    real_ops: list[tuple] = []
    enc_line_pairs = 0

    print("=" * 72)
    print("REAL (non-encoding) HUNKS — detailed")
    print("=" * 72)

    for tag, i1, i2, j1, j2 in sm.get_opcodes():
        if tag == "equal":
            continue
        hchunk = head_lines[i1:i2]
        wchunk = work_lines[j1:j2]
        kind = classify_hunk(hchunk, wchunk)

        if kind == "encoding":
            enc_line_pairs += max(len(hchunk), len(wchunk))
            continue

        if kind == "mixed" and len(hchunk) == len(wchunk):
            for k, (hl, wl) in enumerate(zip(hchunk, wchunk)):
                if is_encoding_only_pair(hl, wl):
                    enc_line_pairs += 1
                    continue
                real_ops.append(("replace", i1 + k, i1 + k + 1, j1 + k, j1 + k + 1, [hl], [wl]))
                print(f"\n### REAL replace  HEAD L{i1+k+1}  WORK L{j1+k+1}")
                print(f"  -HEAD: {unicode_preview(hl)}")
                print(f"  +WORK: {unicode_preview(wl)}")
                print(f"  skH={ascii_skeleton(hl)!r}")
                print(f"  skW={ascii_skeleton(wl)!r}")
            continue

        # For unequal mixed/real, try to peel encoding-only line pairs via SequenceMatcher on skeletons
        if kind in ("real", "mixed") and hchunk and wchunk:
            h_sk = [ascii_skeleton(h) for h in hchunk]
            w_sk = [ascii_skeleton(w) for w in wchunk]
            inner = difflib.SequenceMatcher(a=h_sk, b=w_sk, autojunk=False)
            peeled_any = False
            for itag, a1, a2, b1, b2 in inner.get_opcodes():
                if itag == "equal":
                    # verify encoding-only
                    for hi, wi in zip(range(a1, a2), range(b1, b2)):
                        if is_encoding_only_pair(hchunk[hi], wchunk[wi]):
                            enc_line_pairs += 1
                        else:
                            # shouldn't happen on equal skeleton
                            real_ops.append(
                                (
                                    "replace",
                                    i1 + hi,
                                    i1 + hi + 1,
                                    j1 + wi,
                                    j1 + wi + 1,
                                    [hchunk[hi]],
                                    [wchunk[wi]],
                                )
                            )
                    continue
                peeled_any = True
                if itag == "replace" and (a2 - a1) == (b2 - b1):
                    for hi, wi in zip(range(a1, a2), range(b1, b2)):
                        if is_encoding_only_pair(hchunk[hi], wchunk[wi]):
                            enc_line_pairs += 1
                            continue
                        real_ops.append(
                            (
                                "replace",
                                i1 + hi,
                                i1 + hi + 1,
                                j1 + wi,
                                j1 + wi + 1,
                                [hchunk[hi]],
                                [wchunk[wi]],
                            )
                        )
                        print(f"\n### REAL replace  HEAD L{i1+hi+1}  WORK L{j1+wi+1}")
                        print(f"  -HEAD: {unicode_preview(hchunk[hi])}")
                        print(f"  +WORK: {unicode_preview(wchunk[wi])}")
                        print(f"  skH={ascii_skeleton(hchunk[hi])!r}")
                        print(f"  skW={ascii_skeleton(wchunk[wi])!r}")
                else:
                    real_ops.append(
                        (
                            itag,
                            i1 + a1,
                            i1 + a2,
                            j1 + b1,
                            j1 + b2,
                            hchunk[a1:a2],
                            wchunk[b1:b2],
                        )
                    )
                    print(f"\n### REAL {itag.upper()}  HEAD L{i1+a1+1}-{i1+a2}  WORK L{j1+b1+1}-{j1+b2}")
                    for hl in hchunk[a1:a2]:
                        print(f"  -HEAD: {unicode_preview(hl)}")
                    for wl in wchunk[b1:b2]:
                        print(f"  +WORK: {unicode_preview(wl)}")
            if peeled_any or True:
                continue

        real_ops.append((tag, i1, i2, j1, j2, hchunk, wchunk))
        print(f"\n### REAL {tag.upper()}  HEAD L{i1+1}-{i2}  WORK L{j1+1}-{j2}")
        for hl in hchunk:
            print(f"  -HEAD: {unicode_preview(hl)}")
        for wl in wchunk:
            print(f"  +WORK: {unicode_preview(wl)}")

    # Dedupe overlapping ops by (i1,i2,j1,j2)
    uniq = []
    seen = set()
    for op in real_ops:
        key = op[:5]
        if key in seen:
            continue
        seen.add(key)
        uniq.append(op)
    real_ops = uniq

    print("\n" + "=" * 72)
    print(f"encoding_line_pairs≈{enc_line_pairs}  real_ops={len(real_ops)}")
    print("=" * 72)

    # Apply real ops onto HEAD (reverse by HEAD start)
    out = list(head_lines)
    for tag, i1, i2, j1, j2, hchunk, wchunk in sorted(real_ops, key=lambda x: x[1], reverse=True):
        # Use working lines for real changes (they should be ASCII-clean for real logic)
        # If a working line still has ??, prefer HEAD if encoding-only vs that head line;
        # otherwise keep work (logic).
        repl = []
        for idx, wl in enumerate(wchunk):
            if "??" in wl and idx < len(hchunk) and is_encoding_only_pair(hchunk[idx], wl):
                repl.append(hchunk[idx])
            else:
                repl.append(wl)
        out[i1:i2] = repl

    restored = "".join(out)
    if restored and not restored.endswith("\n"):
        restored += "\n"
    RESTORED.write_bytes(restored.encode("utf-8"))

    rt = RESTORED.read_text(encoding="utf-8")
    cjk = len(CJK_RE.findall(rt))
    print("\nRESTORED verification:")
    print("  bytes", RESTORED.stat().st_size)
    print("  cjk", cjk, "(HEAD was", len(CJK_RE.findall(head_text)), ")")
    print("  lines", rt.count("\n") + 1)
    print("  Pod ???", '"Pod ???"' in rt)
    bad = [m.group(0) for m in re.finditer(r'"[^"]{0,50}"', rt) if "??" in m.group(0)]
    print("  strings_with_??", bad[:15], "count", len(bad))
    print("  @xterm/xterm", "@xterm/xterm" in rt)
    print("  xterm/css (old)", '"xterm/css/xterm.css"' in rt or "from \"xterm\"" in rt)
    print("  getToken", "getToken" in rt)
    print("  credentials: include", 'credentials: "include"' in rt)
    print("  Authorization Bearer token", "Authorization: token" in rt)
    print("  PodDetailPanel", "PodDetailPanel" in rt)
    print("  PodLogsPanel", "PodLogsPanel" in rt)

    # Diff restored vs HEAD ascii-only summary
    r_lines = rt.splitlines(keepends=True)
    sm2 = difflib.SequenceMatcher(a=head_lines, b=r_lines, autojunk=False)
    print("\n--- Diff RESTORED vs HEAD (should be only real changes) ---")
    for tag, i1, i2, j1, j2 in sm2.get_opcodes():
        if tag == "equal":
            continue
        print(f"{tag} HEAD {i1+1}-{i2} -> REST {j1+1}-{j2}")
        for hl in head_lines[i1:i2]:
            print(" -", unicode_preview(hl, 160))
        for rl in r_lines[j1:j2]:
            print(" +", unicode_preview(rl, 160))


if __name__ == "__main__":
    main()
