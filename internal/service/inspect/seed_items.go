package inspect

import "yunshu/internal/model"

// defaultTemplateItems 清洗后的通用巡检模板（去掉环境硬编码 IP）。
func defaultTemplateItems() []model.InspectItem {
	items := []struct {
		Type, Name, Desc, Query, ThresholdType, Unit string
		Threshold                                    float64
		Sort                                         int
	}{
		{"基础设施层", "CPU 使用率", "节点 CPU 使用率", `100 - (avg by (instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)`, "greater", "%", 85, 10},
		{"基础设施层", "内存使用率", "节点内存使用率", `(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100`, "greater", "%", 85, 20},
		{"基础设施层", "根分区磁盘使用率", "根路径磁盘使用率", `(1 - (node_filesystem_avail_bytes{mountpoint="/",fstype!~"tmpfs|overlay"} / node_filesystem_size_bytes{mountpoint="/",fstype!~"tmpfs|overlay"})) * 100`, "greater", "%", 80, 30},
		{"基础设施层", "inode 使用率", "根路径 inode", `(node_filesystem_files_used{mountpoint="/"} / node_filesystem_files{mountpoint="/"}) * 100`, "greater", "%", 90, 40},
		{"基础设施层", "ICMP 连通性", "Blackbox ICMP", `probe_success{job=~".*icmp.*|icmp"}`, "equal", "", 1, 50},
		{"基础设施层", "TCP 探测", "Blackbox TCP", `probe_success{job=~".*tcp.*|Service_tcp"}`, "equal", "", 1, 60},
		{"k8s集群", "Node Ready", "节点 Ready", `kube_node_status_condition{condition="Ready",status="true"}`, "equal", "", 1, 110},
		{"k8s集群", "kube-apiserver", "API Server up", `up{job=~".*apiserver.*"}`, "equal", "", 1, 120},
		{"k8s集群", "kube-scheduler", "Scheduler up", `up{job=~".*scheduler.*"}`, "equal", "", 1, 130},
		{"k8s集群", "kube-controller", "Controller up", `up{job=~".*controller.*"}`, "equal", "", 1, 140},
		{"k8s集群", "coredns", "CoreDNS up", `up{job=~".*coredns.*|.*dns.*"}`, "equal", "", 1, 150},
		{"k8s集群", "kubelet", "Kubelet up", `up{job=~".*kubelet.*"}`, "equal", "", 1, 160},
		{"数据库监控", "MySQL 存活", "mysqld_exporter up / mysql_up", `mysql_up`, "equal", "", 1, 210},
		{"数据库监控", "从库 SQL 线程", "Slave SQL running", `mysql_slave_status_slave_sql_running`, "equal", "", 1, 220},
		{"数据库监控", "从库 IO 线程", "Slave IO running", `mysql_slave_status_slave_io_running`, "equal", "", 1, 230},
		{"数据库监控", "主从延时", "Seconds behind master", `mysql_slave_status_seconds_behind_master`, "greater", "s", 30, 240},
		{"中间件层", "Redis 存活", "redis_up", `redis_up`, "equal", "", 1, 310},
		{"中间件层", "ES 集群健康", "green=0 yellow=1 red=2（按导出器约定可调整）", `elasticsearch_cluster_health_status{color="red"}`, "equal", "", 0, 320},
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
			Enabled:       true,
			SortOrder:     it.Sort,
		})
	}
	return out
}
