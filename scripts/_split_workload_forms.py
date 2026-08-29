from pathlib import Path

src_path = Path("web/src/components/k8s/workload-forms.tsx")
lines = src_path.read_text(encoding="utf-8").splitlines(keepends=True)
# 1-indexed ranges inclusive
# helpers: 4-180 (skip first 3 import lines - rewrite)
# Actually keep helpers as everything before DeploymentFormValues

text = "".join(lines)
# Find markers
markers = {
    "helpers_end": text.find("export type DeploymentFormValues"),
    "stateful": text.find("export type StatefulSetFormValues"),
    "job_type": text.find("export type JobFormValues"),
    "daemon": text.find("export type DaemonSetFormValues"),
    "cron": text.find("export type CronJobFormValues"),
    "build_job": text.find("export function buildJobYaml"),
    "env_pairs": text.find("export function EnvPairsFormItem"),
}
for k, v in markers.items():
    if v < 0:
        raise SystemExit(f"missing marker {k}")

# Split by character positions into logical chunks from original file lines
# Convert char pos to line index
def line_of(pos: int) -> int:
    return text[:pos].count("\n")  # 0-based

h_end = line_of(markers["helpers_end"])
ss = line_of(markers["stateful"])
job_t = line_of(markers["job_type"])
ds = line_of(markers["daemon"])
cj = line_of(markers["cron"])
bj = line_of(markers["build_job"])
ep = line_of(markers["env_pairs"])

# helpers body without original imports (lines 0..h_end-1 include imports+helpers)
helper_body = "".join(lines[3:h_end])  # skip antd/yaml imports
deployment = "".join(lines[h_end:ss])
statefulset = "".join(lines[ss:job_t])
# JobFormValues is before DaemonSet; DaemonSet block; CronJob; then buildJobYaml which is Job builders
# Structure: JobFormValues type, DaemonSet..., CronJob..., buildJobYaml...
# Better ranges:
# job type only then daemon then cron then job builders
# From job_t to ds = JobFormValues type only
# ds to cj = DaemonSet
# cj to bj = CronJob (includes cron builders)
# bj to ep = Job builders

job_type_block = "".join(lines[job_t:ds])
daemon = "".join(lines[ds:cj])
cron = "".join(lines[cj:bj])
job_builders = "".join(lines[bj:ep])
forms = "".join(lines[ep:])

out = Path("web/src/components/k8s/workload-forms")
out.mkdir(parents=True, exist_ok=True)

helpers_ts = '''import YAML from "yaml";

''' + helper_body

# Export private helpers that other modules need
# Convert `function` / `type` at top level to export where needed
# Heuristic: export all top-level function and type in helpers
import re
def exportize(src: str) -> str:
    src = re.sub(r"^type ", "export type ", src, flags=re.M)
    src = re.sub(r"^function ", "export function ", src, flags=re.M)
    src = re.sub(r"^const ", "export const ", src, flags=re.M)
    return src

helpers_ts = exportize(helpers_ts)

# helper imports used by domain files
HELPER_IMPORT = '''import {
  envPairsToMap,
  mapToEnvPairs,
  safeParseYaml,
  safeGet,
  toNumberOrUndefined,
  probeToForm,
  formProbeToK8s,
  kvPairsToMap,
  mapToKvPairs,
  type EnvPair,
} from "./helpers";
'''

# Check which helper names exist
helper_names = set(re.findall(r"^export (?:function|type|const) (\w+)", helpers_ts, flags=re.M))
print("helper exports", sorted(helper_names))

# Build HELPER_IMPORT dynamically from names that domain files reference
# Safer: export * from helpers in each file that needs it - but tree-shaking
# Use: import * as H from "./helpers" - too invasive
# Instead import all exported helpers:

all_helpers = ",\n  ".join(sorted(helper_names))
HELPER_IMPORT = f"import {{\n  {all_helpers}\n}} from \"./helpers\";\n"

def needs_antd(src: str) -> bool:
    return "Form" in src or "Input" in src or "Button" in src or "Drawer" in src or "Alert" in src

def write(name: str, content: str, extra_imports: str = ""):
    (out / name).write_text(content, encoding="utf-8", newline="\n")
    print("wrote", name, "chars", len(content))

write("helpers.ts", helpers_ts)

dep_imports = '''import YAML from "yaml";
''' + HELPER_IMPORT + "\n"
write("deployment.ts", dep_imports + deployment)

write("statefulset.ts", '''import YAML from "yaml";
''' + HELPER_IMPORT + "\n" + statefulset)

write(
    "job.ts",
    '''import YAML from "yaml";
''' + HELPER_IMPORT + "\n" + job_type_block + job_builders,
)

write("daemonset.ts", '''import YAML from "yaml";
''' + HELPER_IMPORT + "\n" + daemon)

write("cronjob.ts", '''import YAML from "yaml";
''' + HELPER_IMPORT + "\n" + cron)

form_imports = '''import { Alert, Button, Card, Col, Divider, Drawer, Form, Input, InputNumber, Row, Select, Space, Typography } from "antd";
import type { FormInstance } from "antd";
''' + HELPER_IMPORT + "\n"
write("form-items.tsx", form_imports + forms)

index = '''export * from "./helpers";
export * from "./deployment";
export * from "./statefulset";
export * from "./daemonset";
export * from "./job";
export * from "./cronjob";
export * from "./form-items";
'''
write("index.ts", index)

# Replace original with re-export barrel
src_path.write_text('export * from "./workload-forms";\n', encoding="utf-8", newline="\n")
print("replaced workload-forms.tsx with re-export")
