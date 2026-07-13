import { DownloadOutlined, PlayCircleOutlined, PlusOutlined, ReloadOutlined, UploadOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Form, Input, Radio, Select, Space, Table, Tabs, Tag, Tree, Upload, message } from "antd";
import type { DataNode } from "antd/es/tree";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { CREATE_TABLE_SAMPLE, SQL_AUDIT_RULES, formatInstanceLabel } from "../components/dbmgmt/dbmgmt-ui-shared";
import { envLabel } from "../utils/dbmgmt-labels";
import { MonacoSqlEditor } from "../components/dbmgmt/monaco-sql-editor";
import {
  checkDbSql,
  executeDb,
  getEffectiveDbPermission,
  importDbSql,
  listDbColumns,
  listDbDatabases,
  listDbInstances,
  listDbTables,
  listDbExecutions,
  queryDb,
  type DbEffectivePermission,
  type DbInstance,
  type DbSqlCheckRow,
  type DbSqlExecution,
} from "../services/dbmgmt";
import { getProjects, type ProjectItem } from "../services/projects";
import { downloadQueryResultCsv, formatSqlCheckFailure, formatSqlCheckSummary, guessSqlRiskLevel, quoteSqlIdent, riskLevelColor, riskLevelLabel, sanitizeQuerySql } from "../utils/dbmgmt-console";
import { formatDateTime } from "../utils/format";

export type DbmgmtConsoleMode = "all" | "query" | "audit";

function formatSqlBasic(sql: string) {
  return sql
    .replace(/\s+/g, " ")
    .replace(/\b(SELECT|FROM|WHERE|JOIN|LEFT JOIN|RIGHT JOIN|INNER JOIN|GROUP BY|ORDER BY|LIMIT|INSERT|UPDATE|DELETE|CREATE|ALTER|DROP)\b/gi, "\n$1")
    .trim();
}

function instanceTreeTitle(inst: DbInstance) {
  return `${formatInstanceLabel(inst)} (${envLabel(inst.env)})`;
}

export function DbmgmtConsolePage({ mode = "all" }: { mode?: DbmgmtConsoleMode }) {
  const isQueryMode = mode === "query" || mode === "all";
  const isAuditMode = mode === "audit" || mode === "all";
  const pageTitle = mode === "query" ? "SQL查询" : mode === "audit" ? "SQL审核" : "SQL 控制台";
  const [auditTab, setAuditTab] = useState<"sql" | "file">("sql");
  const [auditType, setAuditType] = useState<"system" | "manual">("system");
  const [backupChoice, setBackupChoice] = useState<"yes" | "no">("yes");
  const [changeDesc, setChangeDesc] = useState("");
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [instances, setInstances] = useState<DbInstance[]>([]);
  const [instanceId, setInstanceId] = useState<number>();
  const [database, setDatabase] = useState<string>();
  const [selectedTable, setSelectedTable] = useState<string>();
  const [treeData, setTreeData] = useState<DataNode[]>([]);
  const [sql, setSql] = useState("SELECT 1");
  const [columns, setColumns] = useState<string[]>([]);
  const [columnDefs, setColumnDefs] = useState<{ title: string; dataIndex: string }[]>([]);
  const [rawRows, setRawRows] = useState<unknown[][]>([]);
  const [rows, setRows] = useState<Record<string, unknown>[]>([]);
  const [tableColumns, setTableColumns] = useState<{ name: string; data_type: string; nullable: boolean; comment?: string }[]>([]);
  const [loading, setLoading] = useState(false);
  const [perm, setPerm] = useState<DbEffectivePermission | null>(null);
  const [riskMsg, setRiskMsg] = useState<string>();
  const [importSql, setImportSql] = useState("");
  const [importFileName, setImportFileName] = useState("");
  const [checkOpen, setCheckOpen] = useState(false);
  const [checkRows, setCheckRows] = useState<DbSqlCheckRow[]>([]);
  const [checkSummary, setCheckSummary] = useState<string>();
  const [checkPassed, setCheckPassed] = useState(false);
  const [importCheckPassed, setImportCheckPassed] = useState(false);
  const [queryTab, setQueryTab] = useState<"history" | "result" | "structure">("history");
  const [historyRows, setHistoryRows] = useState<DbSqlExecution[]>([]);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [historyTotal, setHistoryTotal] = useState(0);
  const [historyPage, setHistoryPage] = useState(1);
  const [historyKeyword, setHistoryKeyword] = useState("");

  const currentInstance = useMemo(() => instances.find((i) => i.id === instanceId), [instances, instanceId]);
  const writeRisk = useMemo(() => guessSqlRiskLevel(sql), [sql]);
  const isReadOnlySql = writeRisk === "low";
  const canWriteSql = useMemo(
    () => !perm || perm.can_manage || perm.can_dml || perm.can_ddl || perm.can_import,
    [perm],
  );
  const projectName = projects.find((p) => p.id === projectId)?.name ?? "数据库";

  useEffect(() => {
    void getProjects({ page: 1, page_size: 200 }).then((res) => {
      setProjects(res.list ?? []);
      if (res.list?.length) setProjectId(res.list[0].id);
    });
  }, []);

  const loadInstances = useCallback(async () => {
    if (!projectId) return;
    const res = await listDbInstances(projectId, { page: 1, page_size: 200 });
    const list = res.list ?? [];
    setInstances(list);
  }, [projectId]);

  useEffect(() => {
    void loadInstances();
  }, [loadInstances]);

  const buildInstanceTree = useCallback(async () => {
    if (!projectId) return;
    const byDriver = new Map<string, DbInstance[]>();
    for (const inst of instances) {
      const d = (inst.driver || "mysql").toLowerCase();
      if (!byDriver.has(d)) byDriver.set(d, []);
      byDriver.get(d)!.push(inst);
    }
    const driverNodes: DataNode[] = [];
    for (const [driver, insts] of byDriver.entries()) {
      const children: DataNode[] = [];
      for (const inst of insts) {
        const instKey = `inst:${inst.id}`;
        let dbChildren: DataNode[] = [];
        if (instanceId === inst.id) {
          const dbs = await listDbDatabases(projectId, inst.id).catch(() => []);
          dbChildren = await Promise.all(
            (dbs ?? []).map(async (d) => {
              const tables = database === d.name ? await listDbTables(projectId, inst.id, d.name).catch(() => []) : [];
              return {
                title: d.name,
                key: `db:${inst.id}:${d.name}`,
                children: (tables ?? []).map((t) => ({ title: t.name, key: `table:${inst.id}:${d.name}.${t.name}`, isLeaf: true })),
              };
            }),
          );
        }
        children.push({ title: instanceTreeTitle(inst), key: instKey, children: dbChildren });
      }
      driverNodes.push({ title: driver, key: `driver:${driver}`, children });
    }
    setTreeData([{ title: projectName, key: "root", children: driverNodes }]);
  }, [projectId, instances, instanceId, database, projectName]);

  useEffect(() => {
    void buildInstanceTree();
  }, [buildInstanceTree]);

  const refreshMeta = useCallback(async () => {
    if (!projectId || !instanceId) return;
    const p = await getEffectiveDbPermission(projectId, instanceId);
    setPerm(p);
    if (database && selectedTable) {
      const cols = await listDbColumns(projectId, instanceId, database, selectedTable).catch(() => []);
      setTableColumns(cols ?? []);
    }
  }, [projectId, instanceId, database, selectedTable]);

  useEffect(() => {
    void refreshMeta();
  }, [refreshMeta]);

  const loadHistory = useCallback(async () => {
    if (!projectId) return;
    setHistoryLoading(true);
    try {
      const res = await listDbExecutions(projectId, {
        query_only: 1,
        instance_id: instanceId ?? "",
        page: historyPage,
        page_size: 10,
      });
      setHistoryRows(res.list ?? []);
      setHistoryTotal(res.total ?? 0);
    } finally {
      setHistoryLoading(false);
    }
  }, [projectId, instanceId, historyPage]);

  useEffect(() => {
    if (mode === "query" && queryTab === "history") void loadHistory();
  }, [mode, queryTab, loadHistory]);

  const onTreeSelect = async (keys: React.Key[]) => {
    const k = String(keys[0] ?? "");
    if (k.startsWith("inst:")) {
      const id = Number(k.slice(5));
      setInstanceId(id);
      setDatabase(undefined);
      setSelectedTable(undefined);
      return;
    }
    if (k.startsWith("db:")) {
      const parts = k.split(":");
      const id = Number(parts[1]);
      const dbName = parts.slice(2).join(":");
      setInstanceId(id);
      setDatabase(dbName);
      setSelectedTable(undefined);
      return;
    }
    if (k.startsWith("table:")) {
      const m = k.match(/^table:(\d+):([^.]+)\.(.+)$/);
      if (!m) return;
      const iid = Number(m[1]);
      const dbn = m[2];
      const tbl = m[3];
      const inst = instances.find((i) => i.id === iid);
      const drv = (inst?.driver || "mysql").toLowerCase();
      setInstanceId(iid);
      setDatabase(dbn);
      setSelectedTable(tbl);
      setSql(`SELECT * FROM ${quoteSqlIdent(tbl, drv)} LIMIT 100`);
      if (projectId) {
        const cols = await listDbColumns(projectId, iid, dbn, tbl).catch(() => []);
        setTableColumns(cols ?? []);
      }
    }
  };

  useEffect(() => {
    setCheckPassed(false);
    setCheckOpen(false);
    setCheckSummary(undefined);
    setCheckRows([]);
  }, [sql, database, instanceId]);

  const runQuery = async () => {
    if (!projectId || !instanceId) {
      message.warning("请先选择实例");
      return;
    }
    if (isQueryMode && perm && !perm.can_query && !perm.can_manage) {
      message.error("当前账号无 SQL 查询权限");
      return;
    }
    if (!database) {
      message.warning("请先选择要查询的数据库");
      return;
    }
    setLoading(true);
    setRiskMsg(undefined);
    try {
      const querySql = sanitizeQuerySql(sql);
      if (!querySql.trim()) {
        message.warning("请输入查询语句");
        return;
      }
      const res = await queryDb(projectId, instanceId, { database, sql: querySql });
      const cols = res.columns ?? [];
      const dataRows = (res.rows ?? []).map((r) => (Array.isArray(r) ? r : []));
      setColumns(cols);
      setColumnDefs(cols.map((c, i) => ({ title: c, dataIndex: String(i) })));
      setRawRows(dataRows);
      setRows(dataRows.map((r) => Object.fromEntries(r.map((v, i) => [String(i), v]))));
      if (res.truncated) message.warning("结果已截断");
      setQueryTab("result");
    } catch (e) {
      message.error(e instanceof Error ? e.message : "查询失败");
    } finally {
      setLoading(false);
    }
  };

  const runCheck = async () => {
    if (!projectId || !instanceId) return;
    const instanceDDL = /^\s*CREATE\s+DATABASE\b/i.test(sql.trim()) || /^\s*DROP\s+DATABASE\b/i.test(sql.trim());
    if (!database && !instanceDDL) {
      message.warning("请先选择数据库");
      return;
    }
    try {
      const res = await checkDbSql(projectId, instanceId, { database, sql });
      setCheckRows(res.rows ?? []);
      setCheckSummary(formatSqlCheckSummary(res));
      setCheckOpen(true);
      const passed = (res.error_count ?? 0) === 0;
      setCheckPassed(passed);
      if (!passed) message.error(formatSqlCheckFailure(res));
      else if ((res.warning_count ?? 0) > 0) message.warning("预检有警告，请确认后执行");
      else message.success("预检通过");
    } catch (e) {
      message.error(e instanceof Error ? e.message : "预检失败");
      setCheckPassed(false);
    }
  };

  const runImportCheck = async () => {
    if (!projectId || !instanceId || !database) {
      message.warning("请先选择数据库");
      return;
    }
    if (!importSql.trim()) {
      message.warning("请上传 SQL 文件");
      return;
    }
    try {
      const res = await checkDbSql(projectId, instanceId, { database, sql: importSql });
      setCheckRows(res.rows ?? []);
      setCheckSummary(formatSqlCheckSummary(res));
      setCheckOpen(true);
      const passed = (res.error_count ?? 0) === 0;
      setImportCheckPassed(passed);
      if (!passed) message.error(formatSqlCheckFailure(res));
      else message.success("文件 SQL 预检通过");
    } catch (e) {
      message.error(e instanceof Error ? e.message : "预检失败");
      setImportCheckPassed(false);
    }
  };

  const runExecute = async () => {
    if (!projectId || !instanceId) {
      message.warning("请先选择实例");
      return;
    }
    if (perm && !canWriteSql) {
      message.error("当前账号无 SQL 变更权限（DML/DDL/导入）");
      return;
    }
    const instanceDDL = /^\s*CREATE\s+DATABASE\b/i.test(sql.trim()) || /^\s*DROP\s+DATABASE\b/i.test(sql.trim());
    if (!database && !instanceDDL) {
      message.warning("请先选择数据库");
      return;
    }
    if (mode === "audit" && !checkPassed) {
      message.warning("请先通过 SQL 检测");
      return;
    }
    if (mode === "audit" && isReadOnlySql) {
      message.warning("SELECT/SHOW 等只读语句请使用「SQL 查询」页面；SQL 审核仅支持数据变更语句");
      return;
    }
    if (mode === "audit" && !changeDesc.trim()) {
      message.warning("请填写变更描述");
      return;
    }
    if (mode !== "audit" && writeRisk === "blocked") {
      message.error("当前 SQL 命中阻断规则，无法执行");
      return;
    }
    try {
      const res = await executeDb(projectId, instanceId, {
        database,
        sql,
        reason: changeDesc || undefined,
        audit_mode: auditType,
        is_backup: backupChoice === "yes",
      });
      if (res.status === "pending_approval") {
        setRiskMsg(`已创建审批工单 #${res.ticket_id}，请到历史工单查看`);
      } else {
        message.success(`执行成功，影响行数 ${res.rows_affected ?? 0}`);
      }
    } catch (e) {
      message.error(e instanceof Error ? e.message : "执行失败");
    }
  };

  const submitImport = async () => {
    if (!projectId || !instanceId || !database) {
      message.warning("请先选择数据库");
      return;
    }
    if (perm && !canWriteSql) {
      message.error("当前账号无 SQL 文件导入权限");
      return;
    }
    if (!importCheckPassed) {
      message.warning("请先对 SQL 文件内容执行检测并通过后再提交");
      return;
    }
    if (!importSql.trim()) {
      message.warning("请上传 SQL 文件");
      return;
    }
    if (!changeDesc.trim()) {
      message.warning("请填写变更概述");
      return;
    }
    const res = await importDbSql(projectId, instanceId, {
      database,
      sql: importSql,
      reason: changeDesc,
      audit_mode: auditType,
      is_backup: backupChoice === "yes",
      sql_file_ref: importFileName || undefined,
    });
    if (res.status === "pending_approval") {
      message.info(res.message ?? `已创建审批工单 #${res.ticket_id}`);
    } else {
      message.success(res.message ?? "导入执行成功");
    }
  };

  const exportCsv = () => {
    if (!columns.length || !rawRows.length) {
      message.warning("暂无查询结果可导出");
      return;
    }
    downloadQueryResultCsv(`query-${database ?? "db"}-${Date.now()}.csv`, columns, rawRows);
    message.success("已导出 CSV");
  };

  const filteredHistory = useMemo(() => {
    const kw = historyKeyword.trim().toLowerCase();
    if (!kw) return historyRows;
    return historyRows.filter((r) => {
      const inst = instances.find((i) => i.id === r.instance_id);
      const name = inst ? formatInstanceLabel(inst) : "";
      return (
        name.toLowerCase().includes(kw) ||
        (r.database_name ?? "").toLowerCase().includes(kw) ||
        (r.sql_excerpt ?? "").toLowerCase().includes(kw)
      );
    });
  }, [historyRows, historyKeyword, instances]);

  const selectedDbPath = database
    ? `${projectName}/${(currentInstance?.driver || "mysql").toLowerCase()}/${currentInstance ? instanceTreeTitle(currentInstance) : ""}/${database}`
    : "/";

  return (
    <Card title={pageTitle}>
      <div className="dbmgmt-console-layout">
        <div className="dbmgmt-console-tree">
          <div style={{ marginBottom: 8, fontWeight: 600 }}>{mode === "audit" ? "数据库列表" : "请选择数据库（层级最右端可查看表）"}</div>
          <Select
            style={{ width: "100%", marginBottom: 8 }}
            value={projectId}
            options={projects.map((p) => ({ value: p.id, label: p.name }))}
            onChange={(v: number) => {
              setProjectId(v);
              setInstanceId(undefined);
              setDatabase(undefined);
            }}
          />
          <Tree showLine defaultExpandAll treeData={treeData} onSelect={(keys) => void onTreeSelect(keys)} />
        </div>
        <div className="dbmgmt-console-main">
          {mode === "query" ? (
            <>
              <Tabs
                type="editable-card"
                hideAdd
                items={[{ key: "1", label: "查询 1", closable: false }]}
                tabBarExtraContent={<Button type="text" icon={<PlusOutlined />} disabled />}
              />
              {!instanceId ? (
                <Alert type="info" showIcon message="当前未选择实例信息!" style={{ marginBottom: 12 }} />
              ) : null}
              <Space style={{ marginBottom: 12 }} wrap>
                <Button
                  type="primary"
                  icon={<PlayCircleOutlined />}
                  loading={loading}
                  disabled={perm != null && !perm.can_query && !perm.can_manage}
                  onClick={() => void runQuery()}
                >
                  查询
                </Button>
                <Button onClick={() => setSql(formatSqlBasic(sql))}>格式化</Button>
                <Link to="/dbmgmt/apply/query">
                  <Button>查询权限申请</Button>
                </Link>
                <Button icon={<ReloadOutlined />} onClick={() => void loadHistory()} />
                <Button icon={<DownloadOutlined />} disabled={!rawRows.length} onClick={exportCsv}>
                  导出 CSV
                </Button>
              </Space>
              <MonacoSqlEditor value={sql} onChange={setSql} height={240} />
            </>
          ) : null}

          {mode === "audit" ? (
            <>
              <Tabs
                activeKey={auditTab}
                onChange={(k) => setAuditTab(k as "sql" | "file")}
                items={[
                  { key: "sql", label: "sql变更" },
                  { key: "file", label: "sql文件变更" },
                ]}
                style={{ marginBottom: 8 }}
              />
              <div style={{ marginBottom: 8 }}>当前选择的库：{selectedDbPath}</div>
              {auditTab === "sql" ? (
                <MonacoSqlEditor value={sql} onChange={(v) => { setSql(v); setCheckPassed(false); }} height={220} />
              ) : (
                <Space direction="vertical" style={{ width: "100%" }} size="middle">
                  <Upload
                    accept=".sql,.txt"
                    maxCount={1}
                    beforeUpload={(file) => {
                      setImportFileName(file.name);
                      void file.text().then((t) => {
                        setImportSql(t);
                        setImportCheckPassed(false);
                      });
                      return false;
                    }}
                    onRemove={() => {
                      setImportFileName("");
                      setImportSql("");
                      setImportCheckPassed(false);
                    }}
                  >
                    <Button icon={<UploadOutlined />}>选取文件</Button>
                  </Upload>
                  {importFileName ? <span style={{ color: "#666" }}>已选文件：{importFileName}</span> : null}
                  <span style={{ color: "#999" }}>请上传.sql脚本，文件名为英文</span>
                </Space>
              )}
              <Form layout="horizontal" labelCol={{ span: 4 }} wrapperCol={{ span: 16 }} style={{ marginTop: 16 }}>
                <Form.Item label="变更描述" required>
                  <Input placeholder="输入工单名称" value={changeDesc} onChange={(e) => setChangeDesc(e.target.value)} />
                </Form.Item>
                <Form.Item label="审核" required>
                  <Radio.Group value={auditType} onChange={(e) => setAuditType(e.target.value)}>
                    <Radio value="system">系统审核</Radio>
                    <Radio value="manual">人工审核</Radio>
                  </Radio.Group>
                </Form.Item>
                <Form.Item label="备份" required>
                  <Radio.Group value={backupChoice} onChange={(e) => setBackupChoice(e.target.value)}>
                    <Radio value="yes">是</Radio>
                    <Radio value="no">否</Radio>
                  </Radio.Group>
                </Form.Item>
              </Form>
              {perm && !canWriteSql && mode === "audit" ? (
                <Alert type="warning" showIcon message="当前账号无 SQL 变更权限，请先在「权限申请」中申请 DML/DDL 或导入权限。" style={{ marginBottom: 12 }} />
              ) : null}
              {auditTab === "sql" && isReadOnlySql ? (
                <Alert
                  type="info"
                  showIcon
                  style={{ marginTop: 8 }}
                  message="当前为只读查询语句，请切换到「SQL 查询」页面执行；SQL 审核用于 INSERT/UPDATE/DELETE/DDL 等变更。"
                />
              ) : null}
              <Space style={{ marginTop: 8 }} wrap>
                {auditTab === "sql" ? (
                  <>
                    <Button type="primary" style={{ background: "#fa8c16", borderColor: "#fa8c16" }} onClick={() => void runCheck()}>
                      sql检测
                    </Button>
                    <Button
                      type="primary"
                      disabled={!checkPassed || isReadOnlySql || !canWriteSql}
                      onClick={() => void runExecute()}
                    >
                      提交sql
                    </Button>
                  </>
                ) : (
                  <>
                    <Button type="primary" style={{ background: "#fa8c16", borderColor: "#fa8c16" }} onClick={() => void runImportCheck()}>
                      sql检测
                    </Button>
                    <Button
                      type="primary"
                      disabled={!importCheckPassed || !database || !importSql.trim() || !changeDesc.trim() || !canWriteSql}
                      onClick={() => void submitImport()}
                    >
                      提交
                    </Button>
                  </>
                )}
              </Space>
              <div style={{ marginTop: 24 }}>
                <div style={{ fontWeight: 600, marginBottom: 8 }}>正确建表样例:（标记红词句必须有, 否则通不过审核!）</div>
                <pre style={{ background: "#f5f5f5", padding: 12, borderRadius: 6, fontSize: 12 }}>{CREATE_TABLE_SAMPLE}</pre>
                <div style={{ fontWeight: 600, marginTop: 12 }}>基本规范</div>
                <ol style={{ paddingLeft: 20, color: "#666" }}>
                  {SQL_AUDIT_RULES.map((r) => (
                    <li key={r}>{r}</li>
                  ))}
                </ol>
              </div>
            </>
          ) : null}

          {mode === "all" ? (
            <>
              <MonacoSqlEditor value={sql} onChange={setSql} height={280} />
              <Space style={{ marginTop: 12 }} wrap>
                {isQueryMode ? (
                  <Button
                  type="primary"
                  icon={<PlayCircleOutlined />}
                  loading={loading}
                  disabled={perm != null && !perm.can_query && !perm.can_manage}
                  onClick={() => void runQuery()}
                >
                    查询
                  </Button>
                ) : null}
                {isAuditMode && !isReadOnlySql ? (
                  <>
                    <Button onClick={() => void runCheck()} disabled={!canWriteSql}>
                      SQL 检查
                    </Button>
                    <Button type="primary" danger disabled={!canWriteSql} onClick={() => void runExecute()}>
                      提交 SQL
                    </Button>
                  </>
                ) : null}
              </Space>
            </>
          ) : null}

          {isQueryMode && mode === "query" ? (
            <Tabs
              style={{ marginTop: 16 }}
              activeKey={queryTab}
              onChange={(k) => setQueryTab(k as "history" | "result" | "structure")}
              items={[
                {
                  key: "history",
                  label: "查询历史",
                  children: (
                    <>
                      <Space style={{ marginBottom: 12 }}>
                        <Input
                          placeholder="请输入实例名称、SQL、数据库"
                          value={historyKeyword}
                          onChange={(e) => setHistoryKeyword(e.target.value)}
                          style={{ width: 280 }}
                        />
                        <Button type="primary" onClick={() => void loadHistory()}>
                          搜索
                        </Button>
                        <Button icon={<ReloadOutlined />} onClick={() => { setHistoryKeyword(""); void loadHistory(); }}>
                          刷新重置
                        </Button>
                      </Space>
                      <Table
                        size="small"
                        loading={historyLoading}
                        rowKey="id"
                        dataSource={filteredHistory}
                        onRow={(r) => ({
                          onDoubleClick: () => {
                            if (r.instance_id) setInstanceId(r.instance_id);
                            if (r.database_name) setDatabase(r.database_name);
                            if (r.sql_excerpt) setSql(r.sql_excerpt);
                          },
                        })}
                        pagination={{
                          current: historyPage,
                          pageSize: 10,
                          total: historyTotal,
                          showTotal: (t) => `共 ${t} 条`,
                          onChange: (p) => setHistoryPage(p),
                        }}
                        columns={[
                          { title: "ID", dataIndex: "id", width: 60 },
                          { title: "开始时间", dataIndex: "created_at", width: 170, render: (v?: string) => (v ? formatDateTime(v) : "—") },
                          {
                            title: "实例名称",
                            width: 200,
                            render: (_, r) => {
                              const inst = instances.find((i) => i.id === r.instance_id);
                              return inst ? formatInstanceLabel(inst) : "—";
                            },
                          },
                          { title: "数据库名称", dataIndex: "database_name", width: 140 },
                          {
                            title: "执行语句（双击粘贴至上方）",
                            dataIndex: "sql_excerpt",
                            ellipsis: true,
                          },
                          { title: "影响行数", dataIndex: "rows_affected", width: 90 },
                          {
                            title: "耗时",
                            dataIndex: "duration_ms",
                            width: 100,
                            render: (v?: number) => (v != null ? `${(v / 1000).toFixed(6)}` : "—"),
                          },
                        ]}
                      />
                    </>
                  ),
                },
                {
                  key: "result",
                  label: "查询结果",
                  children: (
                    <Table
                      size="small"
                      scroll={{ x: true }}
                      rowKey={(_, i) => String(i)}
                      columns={columnDefs}
                      dataSource={rows}
                      pagination={{ pageSize: 50 }}
                      locale={{ emptyText: "查询无结果" }}
                    />
                  ),
                },
                {
                  key: "structure",
                  label: "查询表结构",
                  children: selectedTable && database ? (
                    <Table
                      size="small"
                      rowKey="name"
                      dataSource={tableColumns}
                      columns={[
                        { title: "字段", dataIndex: "name" },
                        { title: "类型", dataIndex: "data_type" },
                        { title: "可空", dataIndex: "nullable", render: (v: boolean) => (v ? "是" : "否") },
                        { title: "注释", dataIndex: "comment", render: (v?: string) => v || "—" },
                      ]}
                    />
                  ) : (
                    <Alert type="info" message="请在左侧树中选择表以查看结构" />
                  ),
                },
              ]}
            />
          ) : null}

          {perm && !perm.can_query && isQueryMode && mode === "query" ? (
            <Alert type="warning" showIcon message={<>无查询权限，可 <Link to="/dbmgmt/apply/query">申请查询权限</Link></>} style={{ marginTop: 12 }} />
          ) : null}
          {riskMsg ? <Alert type="info" showIcon message={riskMsg} style={{ marginTop: 12 }} /> : null}
        </div>
      </div>

      {checkOpen ? (
        <div style={{ marginTop: 16 }}>
          {checkSummary ? (
            <Alert
              type={(checkPassed ? "success" : "error") as "success" | "error"}
              showIcon
              message={checkSummary}
              style={{ marginBottom: 12 }}
            />
          ) : null}
          <Table
            size="small"
            rowKey={(_, i) => String(i)}
            pagination={false}
            scroll={{ x: 800 }}
            dataSource={checkRows}
            columns={[
              { title: "#", dataIndex: "order_id", width: 48 },
              { title: "阶段", dataIndex: "stage", width: 100 },
              {
                title: "级别",
                dataIndex: "error_level",
                width: 72,
                render: (v: number) => (v === 2 ? <Tag color="red">错误</Tag> : v === 1 ? <Tag color="orange">警告</Tag> : <Tag color="green">通过</Tag>),
              },
              { title: "信息", dataIndex: "error_message", ellipsis: true },
              { title: "SQL", dataIndex: "sql", ellipsis: true },
            ]}
          />
        </div>
      ) : null}

      <style>{`
        .dbmgmt-console-layout {
          display: grid;
          grid-template-columns: minmax(220px, 260px) minmax(0, 1fr);
          gap: 16px;
          width: 100%;
        }
        .dbmgmt-console-tree {
          min-width: 0;
          max-height: 720px;
          overflow: auto;
          border: 1px solid var(--ant-color-border-secondary, #f0f0f0);
          border-radius: 8px;
          padding: 8px;
        }
        .dbmgmt-console-main { min-width: 0; }
      `}</style>
    </Card>
  );
}

export function DbmgmtSqlQueryPage() {
  return <DbmgmtConsolePage mode="query" />;
}

export function DbmgmtSqlAuditPage() {
  return <DbmgmtConsolePage mode="audit" />;
}
