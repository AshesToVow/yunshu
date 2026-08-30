from pathlib import Path
import re
from collections import Counter

text = Path("cmd/seed.go").read_text(encoding="utf-8")
rs = re.findall(r'Resource: "([^"]+)"', text)
prefixes = Counter()
for r in rs:
    parts = r.split("/")
    key = "/".join(parts[:4]) if len(parts) >= 4 else r
    prefixes[key] += 1
for k, v in prefixes.most_common():
    print(f"{v:3} {k}")
print("total", len(rs))
