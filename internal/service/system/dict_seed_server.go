package system

// serverDictSeeds 服务器管理域内置字典种子：分组类别、操作系统、SSH 认证与凭据/端口模板。
func serverDictSeeds() []DictEntryCreateRequest {
	return []DictEntryCreateRequest{
		{DictType: "server_group_category", Label: "自建服务器", Value: "self_hosted", Sort: intRef(1), Status: 1, Remark: "服务器分组类别"},
		{DictType: "server_group_category", Label: "云服务器", Value: "cloud", Sort: intRef(2), Status: 1, Remark: "服务器分组类别"},
		{DictType: "server_os_type", Label: "Linux", Value: "linux", Sort: intRef(1), Status: 1, Remark: "服务器操作系统类型"},
		{DictType: "server_os_type", Label: "Windows", Value: "windows", Sort: intRef(2), Status: 1, Remark: "服务器操作系统类型"},
		{DictType: "server_auth_type", Label: "密码", Value: "password", Sort: intRef(1), Status: 1, Remark: "服务器 SSH 认证方式"},
		{DictType: "server_auth_type", Label: "私钥", Value: "key", Sort: intRef(2), Status: 1, Remark: "服务器 SSH 认证方式"},
		{DictType: "server_ssh_username", Label: "root", Value: "root", Sort: intRef(1), Status: 1, Remark: "服务器 SSH 用户名模板"},
		{DictType: "server_ssh_username", Label: "admin", Value: "admin", Sort: intRef(2), Status: 1, Remark: "服务器 SSH 用户名模板"},
		{DictType: "server_ssh_password", Label: "默认密码模板（示例）", Value: "change-me-password", Sort: intRef(1), Status: 1, Remark: "服务器 SSH 密码模板，生产建议改为真实值"},
		{DictType: "server_port", Label: "SSH 默认端口 22", Value: "22", Sort: intRef(1), Status: 1, Remark: "服务器连接端口模板"},
		{DictType: "server_port", Label: "RDP 默认端口 3389", Value: "3389", Sort: intRef(2), Status: 1, Remark: "服务器连接端口模板"},
		{DictType: "cmdb_max_transfer_file_mb", Label: "服务器文件传输上限(MB)", Value: "50", Sort: intRef(1), Status: 1, Remark: "服务器操作台 SFTP 上传/下载单文件上限"},
	}
}
