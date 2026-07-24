package inspect

import "yunshu/internal/model"

// defaultTemplateItems 面向 Prometheus + Telegraf + Blackbox(+ Pushgateway) 的通用模板。
// 指标名对齐 Telegraf prometheus output / 常见自定义 inputs；无硬编码 IP。
// 中间件：先在服务器 telegraf.conf 配 inputs → Prometheus 拉到指标后，再在此启用对应巡检项。
func defaultTemplateItems() []model.InspectItem {
	type spec struct {
		Type, Name, Desc, Query, ThresholdType, Unit string
		Threshold                                    float64
		Sort                                         int
		Enabled                                      bool
	}
	items := []spec{
		// —— 主机：Telegraf inputs.cpu / mem / disk / system ——
		{"基础设施层", "CPU 使用率", "Telegraf inputs.cpu → cpu_usage_*；取非 idle 近似", `100 - cpu_usage_idle{cpu="cpu-total"}`, "greater", "%", 85, 10, true},
		{"基础设施层", "1 分钟负载", "Telegraf inputs.system → system_load1", `system_load1`, "greater", "", 5, 15, true},
		{"基础设施层", "内存使用率", "Telegraf inputs.mem → mem_used_percent", `mem_used_percent`, "greater", "%", 85, 20, true},
		{"基础设施层", "磁盘使用率", "Telegraf inputs.disk；按需改 path 过滤", `disk_used_percent{path=~"/|/data|/export"}`, "greater", "%", 80, 30, true},
		{"基础设施层", "inode 使用率", "Telegraf inputs.disk → disk_inodes_*", `(disk_inodes_used{path=~"/|/data|/export"} / disk_inodes_total{path=~"/|/data|/export"}) * 100`, "greater", "%", 90, 40, true},
		{"基础设施层", "TCP 已建立连接", "Telegraf inputs.netstat → netstat_tcp_established", `netstat_tcp_established`, "greater", "", 30000, 45, false},

		// —— 连通性：Blackbox ——
		{"基础设施层", "ICMP 连通性", "blackbox icmp；job 名按你们 scrape 配置改", `probe_success{job=~".*icmp.*|icmp"}`, "equal", "", 1, 50, true},
		{"基础设施层", "TCP/端口探测", "blackbox tcp；常见 job=Service_tcp 或 blackbox", `probe_success{job=~".*tcp.*|Service_tcp|blackbox"}`, "equal", "", 1, 60, true},
		{"基础设施层", "HTTP 探测", "blackbox http；无则关闭", `probe_success{job=~".*http.*"}`, "equal", "", 1, 65, false},

		// —— Pushgateway：仅示例，job/实例按你们推送约定改 ——
		{"基础设施层", "Pushgateway 批次存活", "示例：按需改 job/exported_job", `up{job=~".*pushgateway.*"}`, "equal", "", 1, 70, false},

		// —— K8s（有 kube-state-metrics 再开）——
		{"k8s集群", "Node Ready", "kube-state-metrics", `kube_node_status_condition{condition="Ready",status="true"}`, "equal", "", 1, 110, false},
		{"k8s集群", "Pod 非 Running", "示例：按命名空间改过滤", `sum by (namespace) (kube_pod_status_phase{phase!="Running",phase!="Succeeded"} > 0)`, "greater", "", 0, 120, false},

		// —— MySQL：Telegraf inputs.mysql / 自定义 mysqlping ——
		{"数据库监控", "MySQL 存活(mysqlping)", "Telegraf 自定义或 inputs；无则改用 mysql_up", `mysqlping_status`, "equal", "", 1, 210, true},
		{"数据库监控", "MySQL 存活(mysql_up)", "mysqld_exporter / telegraf 字段名差异时备选", `mysql_up`, "equal", "", 1, 215, false},
		{"数据库监控", "从库 SQL 线程", "Telegraf/exporter 从库指标；无从库请关闭", `mysql_slave_slave_sql_running`, "equal", "", 1, 220, false},
		{"数据库监控", "从库 IO 线程", "无从库请关闭", `mysql_slave_slave_io_running`, "equal", "", 1, 230, false},
		{"数据库监控", "主从延时", "无从库请关闭", `mysql_slave_seconds_behind_master`, "greater", "s", 30, 240, false},
		{"数据库监控", "从库 SQL 线程(备选名)", "mysqld_exporter 命名差异时启用", `mysql_slave_status_slave_sql_running`, "equal", "", 1, 221, false},
		{"数据库监控", "从库 IO 线程(备选名)", "mysqld_exporter 命名差异时启用", `mysql_slave_status_slave_io_running`, "equal", "", 1, 231, false},
		{"数据库监控", "主从延时(备选名)", "mysqld_exporter 命名差异时启用", `mysql_slave_status_seconds_behind_master`, "greater", "s", 30, 241, false},
		{"数据库监控", "MySQL 备份检查", "自定义 backup check 指标；无则关闭", `mysqlbackupcheck_status`, "equal", "", 1, 250, false},

		// —— Redis / ES：Telegraf inputs.redis / elasticsearch ——
		{"中间件层", "Redis 存活", "Telegraf inputs.redis 或 redis_exporter → redis_up", `redis_up`, "equal", "", 1, 310, true},
		{"中间件层", "Redis 内存使用", "inputs.redis → redis_used_memory / maxmemory 需按环境改", `redis_mem_fragmentation_ratio`, "greater", "", 3, 315, false},
		{"中间件层", "ES 集群红状态", "inputs.elasticsearch；无 ES 请关闭", `elasticsearch_cluster_health_status{color="red"}`, "equal", "", 0, 320, false},
	}
	out := make([]model.InspectItem, 0, len(items))
	for _, it := range items {
		out = append(out, model.InspectItem{
			ProjectID:     0,
			Type:          it.Type,
			Name:          it.Name,
			Description:   it.Desc,
			Query:         it.Query,
			Threshold:     it.Threshold,
			ThresholdType: it.ThresholdType,
			Unit:          it.Unit,
			Enabled:       it.Enabled,
			SortOrder:     it.Sort,
		})
	}
	return out
}
