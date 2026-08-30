# -*- coding: utf-8 -*-
"""RF-03 step 4: extract rules/duty/assign/metric into state/use-rules-state.tsx"""
from __future__ import annotations

from pathlib import Path

ROOT = Path("web/src/pages/alert-monitor")
MAIN = ROOT / "use-alert-monitor-platform-state.tsx"
OUT = ROOT / "state" / "use-rules-state.tsx"

text = MAIN.read_text(encoding="utf-8")
lines = text.splitlines(keepends=True)

def idx(substr: str, start: int = 0) -> int:
    for i in range(start, len(lines)):
        if substr in lines[i]:
            return i
    raise SystemExit(f"not found: {substr!r} from {start}")

# --- slice boundaries (0-based) ---
# Rule cluster A: ruleList .. dutyUsersHint (before cloud hook), excluding bootstrap loading/users/projects/deptTree
i_rule_list = idx("const [ruleList, setRuleList]")
i_duty_hint = idx('const [dutyUsersHint, setDutyUsersHint]')
# include the dutyUsersHint line
i_a_end = i_duty_hint + 1  # exclusive end after blank? keep through dutyUsersHint

# Rule options that sit between silence and promql: ruleComparatorOptions + ruleLogicOptions
i_rule_cmp = idx("const ruleComparatorOptions")
i_rule_logic_end = idx("const ruleLogicOptions")
# find end of ruleLogicOptions useMemo (closing `], []);` then `);`)
i = i_rule_logic_end
while i < len(lines) and "]," not in lines[i]:
    i += 1
i_rule_logic_block_end = i + 2  # `], []);` and maybe blank — check
# Actually: `    [],\n  );\n` — find the `);` after ruleLogicOptions
for j in range(i_rule_logic_end, i_rule_logic_end + 15):
    if lines[j].strip() == ");":
        i_rule_logic_block_end = j + 1
        break

# Metric + all rule ops: from metricKeyword through removeBlk (before return)
i_metric = idx("const [metricKeyword, setMetricKeyword]")
i_return = idx("  return {")
# removeBlk ends just before blank lines then return
i_b_end = i_return  # exclusive

# Bootstrap + tab effects that must STAY in main (inside the metric..return range)
i_bootstrap = idx("const ruleDisplayList = ruleList")
# From ruleDisplayList through end of tab effect — keep in main
# Find end of second useEffect after bootstrap (the tab one)
# Structure:
#   const ruleDisplayList = ruleList;
#   useEffect(() => { bootstrap users/projects }, []);
#   useEffect(() => { tab loaders }, [deps]);
#   ... more rule functions continue after tab effect

# Find where rule functions resume after tab effect.
# After tab effect there's typically more code like openRuleCreate
i_open_rule = idx("function openRuleCreate", i_bootstrap)
# So KEEP in main: i_bootstrap .. i_open_rule (exclusive of openRuleCreate)
# MOVE: i_metric .. i_bootstrap, and i_open_rule .. i_return

print("A:", i_rule_list + 1, "-", i_a_end)
print("cmp/logic:", i_rule_cmp + 1, "-", i_rule_logic_block_end)
print("metric..bootstrap:", i_metric + 1, "-", i_bootstrap)
print("openRule..return:", i_open_rule + 1, "-", i_return)
print("bootstrap stay:", i_bootstrap + 1, "-", i_open_rule)

# Collect body parts for rules hook
part_a = lines[i_rule_list:i_a_end]
# filter bootstrap state from part_a
skip = (
    "const [loading, setLoading]",
    "const [users, setUsers]",
    "const [projects, setProjects]",
    "const [deptTree, setDeptTree]",
)
part_a_f = [l for l in part_a if not any(s in l for s in skip)]
# Also remove blank-only runs? keep as-is for fidelity

part_cmp = lines[i_rule_cmp:i_rule_logic_block_end]
part_m1 = lines[i_metric:i_bootstrap]
part_m2 = lines[i_open_rule:i_return]

body_lines = part_a_f + ["\n"] + part_cmp + ["\n"] + part_m1 + part_m2

# Dict opts: alertSeverity / threshold / template move into rules;
# promqlLabelKeyOpts stays in main (shared with silence) — pass as param.
# Currently these sit between cloud and datasource. We'll load severity/threshold/template in rules hook.

header = '''/**
 * 告警监控平台：监控规则 / 值班 / 处理人 / 指标浏览器（rules Tab）状态（RF-03 第四步拆分产物）
 *
 * 从 `use-alert-monitor-platform-state.tsx` 原地搬迁，逐字保留语义。
 * `loadRules` 仍由主 Hook 的 Tab 副作用统一调用（与 datasources 一并拉取），
 * 因此这里只暴露方法、不自建 Tab 级 effect。
 *
 * `promqlLabelKeyOpts` 由主 Hook 传入（静默 Tab 也用同一字典，避免重复请求）。
 */
import {
  DeleteOutlined,
  EditOutlined,
  CalendarOutlined,
  TeamOutlined,
} from "@ant-design/icons";
import type { TreeSelectProps } from "antd";
import { Button, Form, Popconfirm, Space, Tag, message } from "antd";
import type { Dayjs } from "dayjs";
import dayjs from "dayjs";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { useDictOptions } from "../../../hooks/use-dict-options";
import { extractApiErrorMessage } from "../../../services/http";
import {
  createAlertMonitorRule,
  createDutyBlock,
  deleteAlertMonitorRule,
  deleteDutyBlock,
  getMonitorRuleAssignees,
  listAlertMonitorRules,
  listDutyBlocks,
  promInstantQuery,
  updateAlertMonitorRule,
  updateDutyBlock,
  upsertMonitorRuleAssignees,
  type AlertDutyBlockItem,
  type AlertDatasourceItem,
  type AlertMonitorRuleItem,
} from "../../../services/alert-platform";
import { stringifyPrettyJSON } from "../../../services/alert-mappers";
import type { ProjectItem } from "../../../services/projects";
import type { UserUpdatePayload } from "../../../types/api";
import { getUser, updateUser } from "../../../services/users";
import { formatDateTime } from "../../../utils/format";
import { DEFAULT_PAGE_SIZE } from "../../../utils/table-pagination";

import type {
  MetricLabelFilter,
  RuleBuilderCondition,
  RuleBuilderLogic,
  RuleComparator,
} from "../platform-provider-types";
import {
  buildPromSelectorExpr,
  detectPromFunctionKeyFromExpr,
  isValidPromLabelKey,
  parsePromSelectorExpr,
  unwrapPrometheusQueryData,
} from "../prom-parse";
import { buildRuleExprByConditions, parseRuleBuilderExpr, parseTemplatePresetPair } from "../rule-parse";

export function useAlertMonitorRulesState(params: {
  projectContextId?: number;
  projects: ProjectItem[];
  users: Array<{ label: string; value: number }>;
  deptTree: TreeSelectProps["treeData"];
  setTab: (tab: string) => void;
  dsList: AlertDatasourceItem[];
  promDsId?: number;
  /** 与静默 Tab 共用，由主 Hook 请求后传入 */
  promqlLabelKeyOpts: Array<{ label: string; value: string }>;
}) {
  const {
    projectContextId,
    projects,
    users,
    deptTree,
    setTab,
    dsList,
    promDsId,
    promqlLabelKeyOpts,
  } = params;

  const alertSeverityOpts = useDictOptions("alert_severity");
  const thresholdUnitDictOpts = useDictOptions("alert_threshold_unit");
  const ruleTemplatePresetDictOpts = useDictOptions("alert_rule_template_preset");

'''

# Indent body already has 2-space indent from original function body — good.
# Fix: body uses projects/users/deptTree/setTab/dsList/promDsId/promqlLabelKeyOpts from closure — now from params. OK.
# Fix: body used alertSeverityOpts / thresholdUnitDictOpts / ruleTemplatePresetDictOpts — now defined above. OK.

# Build return object keys from original return for rules-owned fields
# We'll have the hook return an object with all its public fields; main spreads them.

# Identify which identifiers from return are owned by rules
rules_return_keys = [
    "activeProjectName",
    "alertSeverityOpts",
    "applyMetricSelectorToRuleExpr",
    "applyRuleAnnotationPreset",
    "applyRuleBuilderToExpr",
    "applyStepwisePromQL",
    "assignForm",
    "assignOpen",
    "assignSubmitting",
    "assignUserIds",
    "assignUsersHint",
    "blkColumns",
    "blkCurrent",
    "blkForm",
    "blkModalOpen",
    "blkSubmitting",
    "blkUserIds",
    "blockList",
    "commonLabelKeyOptions",
    "copyDutyBlocksFromSelectedRule",
    "copyDutyLoading",
    "copyDutyRuleOptions",
    "copySourceRuleId",
    "dutyModalOpen",
    "dutyRuleId",
    "dutyUsersHint",
    "insertPromFunctionToExpr",
    "labelValueLoading",
    "labelValueOptions",
    "loadLabelValuesForRule",
    "loadMetricOptionsForRule",
    "loadRules",
    "onRuleTableChange",
    "metricKeyword",
    "metricLabelFilters",
    "metricLoading",
    "metricOptions",
    "openBlkCreate",
    "openRuleCreate",
    "openRuleCreateFromObject",
    "projectOptions",
    "promFunctionTemplates",
    "ruleColumns",
    "ruleComparatorOptions",
    "ruleConditions",
    "ruleCurrent",
    "ruleDisplayList",
    "ruleEnabledFilter",
    "ruleEnabledStats",
    "rulePage",
    "rulePageSize",
    "ruleTotal",
    "rulesLoading",
    "ruleForm",
    "ruleLogic",
    "ruleLogicOptions",
    "ruleModalOpen",
    "ruleSeverityOptions",
    "ruleSubmitting",
    "ruleTemplatePresetOptions",
    "selectedMetric",
    "selectedPromFunc",
    "selectedPromFuncMeta",
    "setAssignOpen",
    "setBlkModalOpen",
    "setCopySourceRuleId",
    "setDutyModalOpen",
    "setMetricKeyword",
    "setMetricLabelFilters",
    "setRuleConditions",
    "setRuleEnabledFilter",
    "setRuleLogic",
    "setRuleModalOpen",
    "setSelectedMetric",
    "setSelectedPromFunc",
    "submitAssign",
    "submitBlk",
    "submitRule",
    "thresholdUnit",
    "thresholdUnitOptions",
    "tryFillRuleBuilderFromExpr",
    "users",  # wait - users stays in main and is returned; rules receives it
    "removeBlk",
    "openDuty",
]

# Check which of these actually exist in original return
ret_block = "".join(lines[i_return:])
owned = []
for k in rules_return_keys:
    if f"\n    {k}," in ret_block or f"\n    {k}\n" in ret_block or ret_block.strip().startswith(f"{k},"):
        # also match at start after {
        owned.append(k)
    elif f"    {k}," in ret_block:
        owned.append(k)

# users is returned from main (bootstrap), not from rules — remove if present
owned = [k for k in owned if k != "users"]
# deptTree similarly stays main
owned = [k for k in owned if k != "deptTree"]

# Also need ruleDisplayList - it's defined in the STAY section. Move definition into rules:
# We'll add `const ruleDisplayList = ruleList;` at end of rules body before return.

footer_keys = ",\n".join(f"    {k}" for k in owned)
# Ensure ruleDisplayList is in body
if "ruleDisplayList" not in "".join(body_lines):
    body_lines.append("  const ruleDisplayList = ruleList;\n")

footer = f'''
  return {{
{footer_keys},
  }};
}}
'''

# dayjs.locale may be needed — main already sets it; rules uses Dayjs type and dayjs() — OK if locale set in main.

OUT.write_text(header + "".join(body_lines) + footer, encoding="utf-8")
print(f"Wrote {OUT} ({OUT.stat().st_size} bytes, {len(OUT.read_text(encoding='utf-8').splitlines())} lines)")
print(f"Return keys: {len(owned)}")
for k in owned:
    print(" ", k)

# --- Rewrite MAIN ---
# 1. Remove part_a rule state (keep bootstrap)
# 2. Remove alertSeverity/threshold/template dict opts (keep promqlLabelKeyOpts)
# 3. Remove ruleComparator/ruleLogic
# 4. Remove metric..bootstrap (keep from ruleDisplayList bootstrap+tab)
# 5. Remove openRule..return body
# 6. Add useAlertMonitorRulesState composition
# 7. Update return to spread rules fields

new_lines: list[str] = []

# Rebuild main from scratch by splicing
# Keep header imports but trim unused — do a second pass for imports

# Find export function start
i_fn = idx("export function useAlertMonitorPlatformState")

# We'll construct new function body programmatically by reading sections to keep.

keep_before_rule = lines[:i_rule_list]  # includes up to applyDefaultPromDatasource

# bootstrap state only
bootstrap_block = [
    "  const [loading, setLoading] = useState(false);\n",
    "  const [users, setUsers] = useState<Array<{ label: string; value: number }>>([]);\n",
    "  const [projects, setProjects] = useState<ProjectItem[]>([]);\n",
    '  const [deptTree, setDeptTree] = useState<TreeSelectProps["treeData"]>([]);\n',
    "\n",
]

# From cloud hook comment through silence end, but remove severity/threshold/template opts
# Original: after dutyUsersHint blank, cloud block starts
i_cloud_comment = idx("云资源到期规则", i_a_end - 5 if i_a_end > 5 else 0)
# safer: find useAlertMonitorCloudExpiryState destructure start
i_cloud_const = None
for i in range(i_a_end, i_rule_cmp):
    if "useAlertMonitorCloudExpiryState" in lines[i]:
        # walk back to const {
        for j in range(i, max(0, i - 25), -1):
            if lines[j].lstrip().startswith("const {"):
                i_cloud_const = j
                break
            if "云资源到期" in lines[j] or "cloud-expiry" in lines[j]:
                i_cloud_const = j
                break
        break
if i_cloud_const is None:
    raise SystemExit("cloud block not found")

# From cloud through silence end (line with useAlertMonitorSilenceState ...);
i_sil_call = idx("useAlertMonitorSilenceState({")
i_sil_end = i_sil_call
while i_sil_end < len(lines) and not lines[i_sil_end].rstrip().endswith(");"):
    i_sil_end += 1
i_sil_end += 1

# Between cloud-start and silence: remove the three dict opts that move to rules
mid = lines[i_cloud_const:i_sil_end]
mid_f = []
for l in mid:
    if 'useDictOptions("alert_severity")' in l:
        continue
    if 'useDictOptions("alert_threshold_unit")' in l:
        continue
    if 'useDictOptions("alert_rule_template_preset")' in l:
        continue
    mid_f.append(l)

# Keep promqlLabelKeyOpts — it's in mid. Good.
# After silence was ruleComparator — skip those (moved)
# Keep promql console block
i_prom_comment = None
for i in range(i_rule_logic_block_end, i_metric):
    if "PromQL" in lines[i] or "useAlertMonitorPromqlConsoleState" in lines[i]:
        # include comment lines above
        j = i
        while j > i_rule_logic_block_end and (lines[j - 1].strip().startswith("//") or lines[j - 1].strip() == ""):
            j -= 1
            if "PromQL" in lines[j] or "promql" in lines[j]:
                i_prom_comment = j
                break
        if i_prom_comment is None:
            i_prom_comment = i
        break
if i_prom_comment is None:
    i_prom_comment = idx("useAlertMonitorPromqlConsoleState")

i_prom_ref = idx("setPromDsIdRef.current = setPromDsId")
i_prom_end = i_prom_ref + 1

prom_block = lines[i_prom_comment:i_prom_end]

# Stay: bootstrap effect + tab effect (from ruleDisplayList through just before openRuleCreate)
# But ruleDisplayList moves to rules — keep only the two useEffects
stay_effects = lines[i_bootstrap:i_open_rule]
# Remove `const ruleDisplayList = ruleList;\n` from stay
stay_effects_f = [l for l in stay_effects if "ruleDisplayList" not in l]

# Composition call for rules — after prom ref, before effects
rules_compose = '''
  // 监控规则 / 值班 / 处理人 / 指标浏览器（rules Tab）
  // 状态与操作已下沉到 state/use-rules-state.tsx；`loadRules` 仍由本文件 Tab 副作用统一调用
  const rules = useAlertMonitorRulesState({
    projectContextId,
    projects,
    users,
    deptTree,
    setTab,
    dsList,
    promDsId,
    promqlLabelKeyOpts,
  });
  const {
    activeProjectName,
    alertSeverityOpts,
    applyMetricSelectorToRuleExpr,
    applyRuleAnnotationPreset,
    applyRuleBuilderToExpr,
    applyStepwisePromQL,
    assignForm,
    assignOpen,
    assignSubmitting,
    assignUserIds,
    assignUsersHint,
    blkColumns,
    blkCurrent,
    blkForm,
    blkModalOpen,
    blkSubmitting,
    blkUserIds,
    blockList,
    commonLabelKeyOptions,
    copyDutyBlocksFromSelectedRule,
    copyDutyLoading,
    copyDutyRuleOptions,
    copySourceRuleId,
    dutyModalOpen,
    dutyRuleId,
    dutyUsersHint,
    insertPromFunctionToExpr,
    labelValueLoading,
    labelValueOptions,
    loadLabelValuesForRule,
    loadMetricOptionsForRule,
    loadRules,
    onRuleTableChange,
    metricKeyword,
    metricLabelFilters,
    metricLoading,
    metricOptions,
    openBlkCreate,
    openRuleCreate,
    openRuleCreateFromObject,
    projectOptions,
    promFunctionTemplates,
    ruleColumns,
    ruleComparatorOptions,
    ruleConditions,
    ruleCurrent,
    ruleDisplayList,
    ruleEnabledFilter,
    ruleEnabledStats,
    rulePage,
    rulePageSize,
    ruleTotal,
    rulesLoading,
    ruleForm,
    ruleLogic,
    ruleLogicOptions,
    ruleModalOpen,
    ruleSeverityOptions,
    ruleSubmitting,
    ruleTemplatePresetOptions,
    selectedMetric,
    selectedPromFunc,
    selectedPromFuncMeta,
    setAssignOpen,
    setBlkModalOpen,
    setCopySourceRuleId,
    setDutyModalOpen,
    setMetricKeyword,
    setMetricLabelFilters,
    setRuleConditions,
    setRuleEnabledFilter,
    setRuleLogic,
    setRuleModalOpen,
    setSelectedMetric,
    setSelectedPromFunc,
    submitAssign,
    submitBlk,
    submitRule,
    thresholdUnit,
    thresholdUnitOptions,
    tryFillRuleBuilderFromExpr,
    removeBlk,
    openDuty,
  } = rules;

'''

# Filter owned list against what's in rules_compose — openDuty/removeBlk may not exist in return
# Check original return for removeBlk / openDuty / thresholdUnit / tryFillRuleBuilderFromExpr
orig_ret = "".join(lines[i_return:])
def in_ret(name: str) -> bool:
    return f"    {name}," in orig_ret or f"    {name}\n" in orig_ret

# Rebuild rules_compose destructuring from owned that are in original return
# Plus any that must be used by tab effect: loadRules, ruleEnabledFilter
extra_for_effects = ["loadRules", "ruleEnabledFilter"]
owned_final = []
for k in owned:
    if in_ret(k) or k in extra_for_effects:
        owned_final.append(k)
# ensure loadRules and ruleEnabledFilter
for k in extra_for_effects:
    if k not in owned_final:
        owned_final.append(k)

# Also check openDuty - might be named differently
for cand in ["openDuty", "openDutyModal", "removeBlk", "thresholdUnit", "thresholdUnitOptions", "tryFillRuleBuilderFromExpr"]:
    if in_ret(cand) and cand not in owned_final:
        owned_final.append(cand)
    print(f"in_ret({cand})={in_ret(cand)}")

destructure = ",\n".join(f"    {k}" for k in owned_final)
rules_compose = f'''
  // 监控规则 / 值班 / 处理人 / 指标浏览器（rules Tab）
  // 状态与操作已下沉到 state/use-rules-state.tsx；`loadRules` 仍由本文件 Tab 副作用统一调用
  const rules = useAlertMonitorRulesState({{
    projectContextId,
    projects,
    users,
    deptTree,
    setTab,
    dsList,
    promDsId,
    promqlLabelKeyOpts,
  }});
  const {{
{destructure},
  }} = rules;

'''

# Fix rules hook return to only include owned_final (+ anything used internally doesn't need export)
# Rewrite OUT return section
out_text = OUT.read_text(encoding="utf-8")
# Replace return block
import re
out_text2, n = re.subn(
    r"\n  return \{.*?\n  \};\n\}\n?\Z",
    "\n  return {\n" + ",\n".join(f"    {k}" for k in owned_final) + ",\n  };\n}\n",
    out_text,
    count=1,
    flags=re.S,
)
if n != 1:
    raise SystemExit(f"failed to rewrite rules return, n={n}")
OUT.write_text(out_text2, encoding="utf-8")

# Original return block — keep structure, fields still listed (destructured into scope)
# After effects, just the return from original
ret_lines = lines[i_return:]

# Assemble new main
# Update imports: add useAlertMonitorRulesState; remove unused icons/services later via tsc

# Patch keep_before_rule imports
head = "".join(keep_before_rule)
if "useAlertMonitorRulesState" not in head:
    head = head.replace(
        'import { useAlertMonitorSilenceState } from "./state/use-silence-state";',
        'import { useAlertMonitorSilenceState } from "./state/use-silence-state";\n'
        'import { useAlertMonitorRulesState } from "./state/use-rules-state";',
    )

new_main = (
    head
    + "".join(bootstrap_block)
    + "".join(mid_f)
    + "\n"
    + "".join(prom_block)
    + rules_compose
    + "".join(stay_effects_f)
    + "\n"
    + "".join(ret_lines)
)

MAIN.write_text(new_main, encoding="utf-8")
print(f"Wrote {MAIN} ({len(new_main.splitlines())} lines)")
print("Done. Run tsc to validate.")
