package config

// CicdConfig CI/CD 平台连接（数据字典 cicd_* 优先覆盖）。
type CicdConfig struct {
	Enabled bool `mapstructure:"enabled"`

	Jenkins JenkinsConfig `mapstructure:"jenkins"`

	Jenkinsfile JenkinsfileConfig `mapstructure:"jenkinsfile"`

	MinIO CicdMinIOConfig `mapstructure:"minio"`

	Harbor HarborConfig `mapstructure:"harbor"`

	Credentials CicdCredentialsConfig `mapstructure:"credentials"`

	// Sonar SonarQube 质量门禁（数据字典 cicd_sonar_* 优先）。
	Sonar CicdSonarConfig `mapstructure:"sonar"`
	// Callback Jenkins 阶段/制品回调（HMAC；字典 cicd_jenkins_callback_*）。
	Callback CicdCallbackConfig `mapstructure:"callback"`

	RunSyncIntervalSeconds        int `mapstructure:"run_sync_interval_seconds"`
	DefaultWaitMins               int `mapstructure:"default_wait_mins"`
	DefaultArtifactRetain         int `mapstructure:"default_artifact_retain_count"`
	ApprovalSlaHours              int `mapstructure:"approval_sla_hours"`
	ApprovalReminderIntervalHours int `mapstructure:"approval_reminder_interval_hours"`
}

// CicdSonarConfig SonarQube 扫描与质量门禁。
type CicdSonarConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	URL       string `mapstructure:"url"`
	Token     string `mapstructure:"token"`
	GateBlock bool   `mapstructure:"gate_block"` // 门禁失败时是否拦截进入审批/发布
}

// CicdCallbackConfig Jenkins → Yunshu 回调鉴权与地址。
type CicdCallbackConfig struct {
	HMACSecret  string `mapstructure:"hmac_secret"`
	CallbackURL string `mapstructure:"callback_url"` // 完整回调 URL，注入 Jenkins 参数
}

type JenkinsConfig struct {
	BaseURL   string `mapstructure:"base_url"`
	Username  string `mapstructure:"username"`
	APIToken  string `mapstructure:"api_token"`
	JobFolder string `mapstructure:"job_folder"`
}

type JenkinsfileConfig struct {
	Repo       string `mapstructure:"repo"`
	Branch     string `mapstructure:"branch"`
	Front      string `mapstructure:"front"`
	Backend    string `mapstructure:"backend"`
	K8s        string `mapstructure:"k8s"`
	SharedLib  string `mapstructure:"shared_library_name"`
}

type CicdMinIOConfig struct {
	Endpoint       string `mapstructure:"endpoint"`
	BucketFrontend string `mapstructure:"bucket_frontend"`
	BucketBackend  string `mapstructure:"bucket_backend"`
	MCBin          string `mapstructure:"mc_bin"`
	MCAlias        string `mapstructure:"mc_alias"`
}

type HarborConfig struct {
	URL          string `mapstructure:"url"`
	HostIP       string `mapstructure:"host_ip"`
	ProjectGroup string `mapstructure:"project_group"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
}

type CicdCredentialsConfig struct {
	Git       string `mapstructure:"git"`
	SSHDeploy string `mapstructure:"ssh_deploy"`
	MinIO     string `mapstructure:"minio"`
	Harbor    string `mapstructure:"harbor"`
}

func DefaultCicdConfig() CicdConfig {
	return CicdConfig{
		Enabled: true,
		Jenkinsfile: JenkinsfileConfig{
			Repo:      "git@gitee.com:wxd_ops/jenkinsfile-new.git",
			Branch:    "main",
			Front:     "front.jenkinsfile",
			Backend:   "backend.jenkinsfile",
			K8s:       "cigroovy.jenkinsfile",
			SharedLib: "jenkins_share_libraries",
		},
		MinIO: CicdMinIOConfig{
			Endpoint:       "http://192.168.56.102:8021",
			BucketFrontend: "frontend-artifacts",
			BucketBackend:  "backend-artifacts",
			MCBin:          "/export/server/minio/mc",
			MCAlias:        "myminio",
		},
		Harbor: HarborConfig{
			URL:          "harbor.deploy.local",
			HostIP:       "10.10.10.103",
			ProjectGroup: "registry",
		},
		Credentials: CicdCredentialsConfig{
			Git:       "gitee_registry_ssh",
			SSHDeploy: "target-server-credential",
			MinIO:     "minio-credentials",
			Harbor:    "HARBOR_ID",
		},
		Sonar: CicdSonarConfig{
			Enabled:   false,
			GateBlock: true,
		},
		RunSyncIntervalSeconds:        15,
		DefaultWaitMins:               60,
		DefaultArtifactRetain:         10,
		ApprovalSlaHours:              24,
		ApprovalReminderIntervalHours: 4,
	}
}
