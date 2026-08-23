package model

import (
	"time"

	"gorm.io/datatypes"
)

// --- 驱动与连接模式 ---

const (
	DbDriverMySQL    = "mysql"
	DbDriverPostgres = "postgres"

	DbConnectDirect    = "direct"
	DbConnectSSHTunnel = "ssh_tunnel"

	DbEnvDev  = "dev"
	DbEnvTest = "test"
	DbEnvProd = "prod"

	DbInstanceRolePrimary = "primary" // 主库
	DbInstanceRoleReplica = "replica" // 从库
)

// --- 授权主体 ---

const (
	DbPrincipalUser  = "user"
	DbPrincipalRole  = "role"
	DbPrincipalGroup = "group"
)

// --- 工单类型与状态 ---

const (
	DbTicketTypeAccessRequest = "access_request"
	DbTicketTypeSqlExecute    = "sql_execute"
	DbTicketTypeSqlImport     = "sql_import"

	DbAuditModeSystem = "system"
	DbAuditModeManual = "manual"

	DbRiskLow     = "low"
	DbRiskMedium  = "medium"
	DbRiskHigh    = "high"
	DbRiskBlocked = "blocked"

	DbTicketStatusDraft           = "draft"
	DbTicketStatusPendingApproval = "pending_approval"
	DbTicketStatusApproved        = "approved"
	DbTicketStatusExecuting       = "executing"
	DbTicketStatusSuccess         = "success"
	DbTicketStatusFailed          = "failed"
	DbTicketStatusRejected        = "rejected"
	DbTicketStatusPendingExecution = "pending_execution"

	DbAccessRequestStatusPending  = "pending"
	DbAccessRequestStatusApproved = "approved"
	DbAccessRequestStatusRejected = "rejected"
	DbAccessRequestStatusClosed   = "closed"

	DbApprovalStepPending  = "pending"
	DbApprovalStepApproved = "approved"
	DbApprovalStepRejected = "rejected"
	DbApprovalStepSkipped  = "skipped"
)

const (
	DbApprovalStageDBALead       = "dba_lead"
	DbApprovalStageSecurityLead  = "security_lead"
	DbApprovalStageOpsLead       = "ops_lead"
)

// DbInstance 项目内数据库实例。
type DbInstance struct {
	ID        uint   `json:"id" gorm:"primaryKey"`
	ProjectID uint   `json:"project_id" gorm:"not null;index:idx_db_inst_proj"`
	Name      string `json:"name" gorm:"size:128;not null"`
	Env       string `json:"env" gorm:"size:16;not null;default:'dev';index"`
	Driver    string `json:"driver" gorm:"size:32;not null;default:'mysql'"`
	ConnectMode string `json:"connect_mode" gorm:"size:32;not null;default:'direct'"`

	Host     string `json:"host" gorm:"size:255;not null;default:'127.0.0.1'"`
	Port     int    `json:"port" gorm:"not null;default:3306"`
	Database string `json:"database" gorm:"size:128;comment:默认库（可选）"`

	ServerID      *uint  `json:"server_id,omitempty" gorm:"index;comment:SSH 隧道关联 CMDB Server"`
	SSHLocalHost  string `json:"ssh_local_host" gorm:"size:64;default:'127.0.0.1'"`
	SSHLocalPort  int    `json:"ssh_local_port" gorm:"default:0;comment:本地映射端口，0=动态"`

	Username    string `json:"username" gorm:"size:128;not null"`
	EncPassword string `json:"-" gorm:"type:text;comment:加密凭据"`
	SSLMode     string `json:"ssl_mode" gorm:"size:32;default:'disable';comment:PostgreSQL sslmode"`
	ExtraJSON   string `json:"extra_json" gorm:"type:text;comment:驱动扩展参数 JSON"`

	ReadOnly              bool `json:"read_only" gorm:"not null;default:false;comment:只读副本，禁止 DML/DDL"`
	Role                  string `json:"role" gorm:"size:16;not null;default:'primary';index;comment:primary=主库 replica=从库"`
	PrimaryInstanceID     *uint  `json:"primary_instance_id,omitempty" gorm:"index;comment:从库关联主库实例 ID"`
	RequireTicketForDML   bool `json:"require_ticket_for_dml" gorm:"not null;default:true"`
	OwnerUserID           *uint `json:"owner_user_id,omitempty" gorm:"index"`

	Status      string     `json:"status" gorm:"size:32;default:'unknown'"`
	LastPingAt  *time.Time `json:"last_ping_at,omitempty"`
	LastPingOK  bool       `json:"last_ping_ok" gorm:"default:false"`

	Tags   string `json:"tags" gorm:"size:512"`
	Remark string `json:"remark" gorm:"size:512"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (DbInstance) TableName() string { return "db_instances" }

// DbAccessGrant 长期库表级授权。
type DbAccessGrant struct {
	ID            uint   `json:"id" gorm:"primaryKey"`
	ProjectID     uint   `json:"project_id" gorm:"not null;index"`
	InstanceID    uint   `json:"instance_id" gorm:"not null;index"`
	PrincipalKind string `json:"principal_kind" gorm:"size:16;not null;index"`
	PrincipalRef  string `json:"principal_ref" gorm:"size:128;not null;index"`

	DatabaseName   string `json:"database_name" gorm:"size:128;comment:空=整实例"`
	TableNamesJSON string `json:"table_names_json" gorm:"type:text;comment:JSON 数组，[\"*\"]=整库"`

	CanConnect bool `json:"can_connect" gorm:"not null;default:true"`
	CanQuery   bool `json:"can_query" gorm:"not null;default:false"`
	CanDML     bool `json:"can_dml" gorm:"not null;default:false"`
	CanDDL     bool `json:"can_ddl" gorm:"not null;default:false"`
	CanExport  bool `json:"can_export" gorm:"not null;default:false"`
	CanImport  bool `json:"can_import" gorm:"not null;default:false"`
	CanManage  bool `json:"can_manage" gorm:"not null;default:false"`

	PrivilegesJSON string `json:"privileges_json" gorm:"type:text;comment:细粒度权限 JSON 数组"`

	QueryLimitNum int `json:"query_limit_num" gorm:"not null;default:1000;comment:查询行数上限，仅查询类授权有效"`

	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Remark    string     `json:"remark" gorm:"size:512"`

	CreatedByUserID *uint     `json:"created_by_user_id,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (DbAccessGrant) TableName() string { return "db_access_grants" }

// DbApprovalFlowStage 项目级审批流模板。
type DbApprovalFlowStage struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	ProjectID   uint      `json:"project_id" gorm:"not null;uniqueIndex:uk_db_flow_stage,priority:1"`
	StageKey    string    `json:"stage_key" gorm:"size:32;not null;uniqueIndex:uk_db_flow_stage,priority:2"`
	StageName   string    `json:"stage_name" gorm:"size:64;not null"`
	SortOrder   int       `json:"sort_order" gorm:"not null;default:0"`
	Enabled     bool      `json:"enabled" gorm:"not null;default:true"`
	UserGroupID *uint     `json:"user_group_id,omitempty" gorm:"index"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (DbApprovalFlowStage) TableName() string { return "db_approval_flow_stages" }

// DbAccessRequest 权限申请工单。
type DbAccessRequest struct {
	ID              uint   `json:"id" gorm:"primaryKey"`
	ProjectID       uint   `json:"project_id" gorm:"not null;index"`
	InstanceID      uint   `json:"instance_id" gorm:"not null;index"`
	RequesterUserID uint   `json:"requester_user_id" gorm:"not null;index"`
	RequesterName   string `json:"requester_name" gorm:"size:64"`

	DatabaseName   string `json:"database_name" gorm:"size:128"`
	TableNamesJSON string `json:"table_names_json" gorm:"type:text"`

	CanConnect bool `json:"can_connect"`
	CanQuery   bool `json:"can_query"`
	CanDML     bool `json:"can_dml"`
	CanDDL     bool `json:"can_ddl"`
	CanExport  bool `json:"can_export"`
	CanImport  bool `json:"can_import"`

	PrivilegesJSON string `json:"privileges_json" gorm:"type:text;comment:细粒度权限 JSON 数组"`
	MetaJSON       string `json:"meta_json" gorm:"type:text;comment:建库等扩展字段 JSON"`

	QueryLimitNum int `json:"query_limit_num" gorm:"not null;default:1000"`

	Reason string `json:"reason" gorm:"size:1024"`
	Status string `json:"status" gorm:"size:32;not null;default:'pending';index"`

	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (DbAccessRequest) TableName() string { return "db_access_requests" }

// DbAccessRequestStep 权限申请审批步骤。
type DbAccessRequestStep struct {
	ID              uint       `json:"id" gorm:"primaryKey"`
	AccessRequestID uint       `json:"access_request_id" gorm:"not null;index"`
	StageKey        string     `json:"stage_key" gorm:"size:32;not null"`
	StageName       string     `json:"stage_name" gorm:"size:64;not null"`
	SortOrder       int        `json:"sort_order" gorm:"not null;default:0"`
	Status          string     `json:"status" gorm:"size:32;not null;default:'pending';index:idx_db_access_step_status_activated,priority:1"`
	UserGroupID     *uint      `json:"user_group_id,omitempty"`
	ReviewerUserID  *uint      `json:"reviewer_user_id,omitempty"`
	ReviewerName    string     `json:"reviewer_name" gorm:"size:64"`
	ReviewComment   string     `json:"review_comment" gorm:"size:512"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
	ActivatedAt     *time.Time `json:"activated_at,omitempty" gorm:"index:idx_db_access_step_status_activated,priority:2"`
	LastRemindedAt  *time.Time `json:"last_reminded_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (DbAccessRequestStep) TableName() string { return "db_access_request_steps" }

// DbSqlTicket SQL 执行/导入工单。
type DbSqlTicket struct {
	ID              uint   `json:"id" gorm:"primaryKey"`
	ProjectID       uint   `json:"project_id" gorm:"not null;index"`
	InstanceID      uint   `json:"instance_id" gorm:"not null;index"`
	TicketType      string `json:"ticket_type" gorm:"size:32;not null;index"`
	SubmitterUserID uint   `json:"submitter_user_id" gorm:"not null;index"`
	SubmitterName   string `json:"submitter_name" gorm:"size:64"`

	DatabaseName string `json:"database_name" gorm:"size:128"`
	SqlText      string `json:"sql_text" gorm:"type:text"`
	SqlFileRef   string `json:"sql_file_ref" gorm:"size:512"`
	AuditMode    string `json:"audit_mode" gorm:"size:16;not null;default:'system';comment:system=系统审核 manual=人工审核"`

	RiskLevel     string `json:"risk_level" gorm:"size:16;not null;default:'low'"`
	SyntaxType    int    `json:"syntax_type" gorm:"not null;default:0;comment:0其他 1DDL 2DML"`
	IsBackup      bool   `json:"is_backup" gorm:"not null;default:true"`
	ParsedOpsJSON string `json:"parsed_ops_json" gorm:"type:text"`
	ReviewJSON    string `json:"review_json" gorm:"type:text;comment:goInception预检结果JSON"`
	ExecuteJSON   string `json:"execute_json" gorm:"type:text;comment:goInception执行结果JSON"`
	Reason        string `json:"reason" gorm:"size:1024"`
	Status        string `json:"status" gorm:"size:32;not null;default:'draft';index"`

	RequestJSON string `json:"request_json" gorm:"type:text;comment:审批通过后执行快照"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (DbSqlTicket) TableName() string { return "db_sql_tickets" }

// DbSqlTicketStep SQL 工单审批步骤。
type DbSqlTicketStep struct {
	ID             uint       `json:"id" gorm:"primaryKey"`
	TicketID       uint       `json:"ticket_id" gorm:"not null;index"`
	StageKey       string     `json:"stage_key" gorm:"size:32;not null"`
	StageName      string     `json:"stage_name" gorm:"size:64;not null"`
	SortOrder      int        `json:"sort_order" gorm:"not null;default:0"`
	Status         string     `json:"status" gorm:"size:32;not null;default:'pending';index:idx_db_sql_step_status_activated,priority:1"`
	UserGroupID    *uint      `json:"user_group_id,omitempty"`
	ReviewerUserID *uint      `json:"reviewer_user_id,omitempty"`
	ReviewerName   string     `json:"reviewer_name" gorm:"size:64"`
	ReviewComment  string     `json:"review_comment" gorm:"size:512"`
	ReviewedAt     *time.Time `json:"reviewed_at,omitempty"`
	ActivatedAt    *time.Time `json:"activated_at,omitempty" gorm:"index:idx_db_sql_step_status_activated,priority:2"`
	LastRemindedAt *time.Time `json:"last_reminded_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (DbSqlTicketStep) TableName() string { return "db_sql_ticket_steps" }

// DbSqlExecution 实际 SQL 执行审计。
type DbSqlExecution struct {
	ID             uint   `json:"id" gorm:"primaryKey"`
	ProjectID      uint   `json:"project_id" gorm:"not null;index"`
	TicketID       *uint  `json:"ticket_id,omitempty" gorm:"index"`
	InstanceID     uint   `json:"instance_id" gorm:"not null;index"`
	ExecutorUserID uint   `json:"executor_user_id" gorm:"not null;index"`
	ExecutorName   string `json:"executor_name" gorm:"size:64"`
	DatabaseName   string `json:"database_name" gorm:"size:128"`

	StatementHash      string `json:"statement_hash" gorm:"size:64;index"`
	SqlExcerpt         string `json:"sql_excerpt" gorm:"type:text"`
	RowsAffected       int64  `json:"rows_affected"`
	DurationMs         int64  `json:"duration_ms"`
	ResultPreviewJSON  string `json:"result_preview_json" gorm:"type:text"`
	ErrorMessage       string `json:"error_message" gorm:"type:text"`
	RiskLevel          string `json:"risk_level" gorm:"size:16"`

	CreatedAt time.Time `json:"created_at"`
}

func (DbSqlExecution) TableName() string { return "db_sql_executions" }

// DbAuditLog 统一审计（实例变更、授权、控制台登录等）。
type DbAuditLog struct {
	ID         uint           `json:"id" gorm:"primaryKey"`
	ProjectID  uint           `json:"project_id" gorm:"not null;index"`
	InstanceID *uint          `json:"instance_id,omitempty" gorm:"index"`
	ActorUserID uint          `json:"actor_user_id" gorm:"not null;index"`
	ActorName   string         `json:"actor_name" gorm:"size:64"`
	Action      string         `json:"action" gorm:"size:64;not null;index"`
	DetailJSON  datatypes.JSON `json:"detail_json" gorm:"type:json"`
	IP          string         `json:"ip" gorm:"size:64"`
	CreatedAt   time.Time      `json:"created_at"`
}

func (DbAuditLog) TableName() string { return "db_audit_logs" }

const (
	DbAppUserApplyNewUser  = "new_user"
	DbAppUserApplyAddPriv  = "add_priv"
	DbAppUserApplyAddIP    = "add_ip"
	DbAppUserApplyRevoke   = "revoke"

	DbAppUserPrivGlobal    = "global"
	DbAppUserPrivDatabase  = "database"
)

// DbAppUserRequest MySQL 应用账号权限申请（对齐 smartdbs appuser_apply）。
type DbAppUserRequest struct {
	ID              uint   `json:"id" gorm:"primaryKey"`
	ProjectID       uint   `json:"project_id" gorm:"not null;index"`
	InstanceID      uint   `json:"instance_id" gorm:"not null;index"`
	RequesterUserID uint   `json:"requester_user_id" gorm:"not null;index"`
	RequesterName   string `json:"requester_name" gorm:"size:64"`

	ApplyType    string `json:"apply_type" gorm:"size:32;not null;index"`
	MySQLUser    string `json:"mysql_user" gorm:"size:64;not null"`
	MySQLHost    string `json:"mysql_host" gorm:"size:64;not null;default:'%'"`
	DatabaseName string `json:"database_name" gorm:"size:128"`
	PrivLevel    string `json:"priv_level" gorm:"size:16;not null;default:'database'"`
	PrivilegesJSON string `json:"privileges_json" gorm:"type:text"`
	GrantHosts   string `json:"grant_hosts" gorm:"size:1024;comment:授权 IP，分号分隔"`

	Reason string `json:"reason" gorm:"size:1024"`
	Status string `json:"status" gorm:"size:32;not null;default:'pending';index"`

	ExecuteError string `json:"execute_error" gorm:"type:text"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (DbAppUserRequest) TableName() string { return "db_app_user_requests" }

// DbAppUserRequestStep 应用账号申请审批步骤。
type DbAppUserRequestStep struct {
	ID                uint       `json:"id" gorm:"primaryKey"`
	AppUserRequestID  uint       `json:"app_user_request_id" gorm:"not null;index"`
	StageKey          string     `json:"stage_key" gorm:"size:32;not null"`
	StageName         string     `json:"stage_name" gorm:"size:64;not null"`
	SortOrder         int        `json:"sort_order" gorm:"not null;default:0"`
	Status            string     `json:"status" gorm:"size:32;not null;default:'pending'"`
	UserGroupID       *uint      `json:"user_group_id,omitempty"`
	ReviewerUserID    *uint      `json:"reviewer_user_id,omitempty"`
	ReviewerName      string     `json:"reviewer_name" gorm:"size:64"`
	ReviewComment     string     `json:"review_comment" gorm:"size:512"`
	ReviewedAt        *time.Time `json:"reviewed_at,omitempty"`
	ActivatedAt       *time.Time `json:"activated_at,omitempty"`
	LastRemindedAt    *time.Time `json:"last_reminded_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (DbAppUserRequestStep) TableName() string { return "db_app_user_request_steps" }

// DbInstanceAccount 实例上已创建的应用账号（审批执行后落库）。
type DbInstanceAccount struct {
	ID               uint   `json:"id" gorm:"primaryKey"`
	ProjectID        uint   `json:"project_id" gorm:"not null;index"`
	InstanceID       uint   `json:"instance_id" gorm:"not null;index"`
	AppUserRequestID *uint  `json:"app_user_request_id,omitempty" gorm:"index"`
	Username         string `json:"username" gorm:"size:64;not null;index"`
	Host             string `json:"host" gorm:"size:64;not null;default:'%'"`
	EncPassword      string `json:"-" gorm:"type:text"`
	GrantsSummary    string `json:"grants_summary" gorm:"type:text"`
	Remark           string `json:"remark" gorm:"size:512"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (DbInstanceAccount) TableName() string { return "db_instance_accounts" }

// DbColumnMaskRule 列级脱敏规则。
type DbColumnMaskRule struct {
	ID         uint   `json:"id" gorm:"primaryKey"`
	InstanceID uint   `json:"instance_id" gorm:"not null;index:idx_db_mask_inst"`
	SchemaName string `json:"schema_name" gorm:"size:64;not null;default:'';index:idx_db_mask_inst"`
	MaskTable  string `json:"table_name" gorm:"column:table_name;size:128;not null;index:idx_db_mask_inst"`
	ColumnName string `json:"column_name" gorm:"size:128;not null"`
	MaskType   string `json:"mask_type" gorm:"size:32;not null;default:partial;comment:hash|partial|redact"`
	Pattern    string `json:"pattern" gorm:"size:64;comment:partial 时保留前后字符数如 3,4"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (DbColumnMaskRule) TableName() string { return "db_column_mask_rules" }
