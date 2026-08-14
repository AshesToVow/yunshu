package knowledge

import "strings"

// Module 知识库功能模块标识。
const (
	ModuleAI     = "ai"
	ModuleK8s    = "k8s"
	ModuleCICD   = "cicd"
	ModuleAlert  = "alert"
	ModuleLog    = "log"
	ModuleLinux  = "linux"
	ModuleEsmgmt = "esmgmt"
	ModuleCMDB   = "cmdb"
	ModuleDB     = "dbmgmt"
)

// Doc 内嵌知识文档。
type Doc struct {
	Module  string
	Source  string
	Title   string
	Content string
}

// InferModules 根据用户问题关键词推断相关功能模块。
func InferModules(query string) []string {
	q := strings.ToLower(query)
	type rule struct {
		module string
		keys   []string
	}
	rules := []rule{
		{ModuleK8s, []string{
			"k8s", "kubernetes", "pod", "deployment", "namespace", "命名空间", "集群", "排障",
			"crashloop", "imagepull", "pending", "helm", "节点", "oom", "重启", "容器",
			"服务挂了", "挂了", "探针", "调度", "副本", "扩容", "缩容", "workload",
		}},
		{ModuleCICD, []string{
			"cicd", "ci/cd", "jenkins", "构建", "发布", "流水线", "build", "release",
			"制品", "镜像推送", "上线", "编译失败", "打包失败", "harbor 推",
		}},
		{ModuleAlert, []string{
			"告警", "alert", "prometheus", "alertmanager", "通知", "抑制", "指纹",
			"fingerprint", "订阅", "没收到告警", "告警噪音", "钉钉", "企微",
		}},
		{ModuleLog, []string{
			"日志", "log", "loggie", "kafka", "检索日志", "查日志", "没有日志", "采集",
			"daemonset 日志", "agent 日志",
		}},
		{ModuleLinux, []string{
			"linux", "磁盘", "disk", "df", "inode", "磁盘满", "磁盘打满", "空间不足",
			"内存", "mem", "load", "负载", "cpu 高", "主机探测", "statvfs",
		}},
		{ModuleEsmgmt, []string{"esmgmt", "es 管理", "索引备份", "es 连接", "elasticsearch 集群"}},
		{ModuleCMDB, []string{"cmdb", "主机", "服务器", "资产", "ssh", "云账号", "服务器管理"}},
		{ModuleDB, []string{"数据库", "mysql", "dbmgmt", "授权", "备份库", "sql"}},
		{ModuleAI, []string{"ai 助手", "操作审批", "知识库", "tool calling", "怎么用助手"}},
	}
	seen := map[string]struct{}{}
	var out []string
	for _, r := range rules {
		for _, k := range r.keys {
			if strings.Contains(q, strings.ToLower(k)) {
				if _, ok := seen[r.module]; !ok {
					seen[r.module] = struct{}{}
					out = append(out, r.module)
				}
				break
			}
		}
	}
	return out
}
