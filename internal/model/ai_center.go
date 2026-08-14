package model

import (
	"time"

	"gorm.io/gorm"
)

// AiLLMModel LLM 模型目录。
type AiLLMModel struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	Name          string         `json:"name" gorm:"size:128;not null;uniqueIndex"`
	Provider      string         `json:"provider" gorm:"size:64;not null;index"` // openai_compat|deepseek|anthropic|qwen
	BaseURL       string         `json:"base_url" gorm:"size:512"`
	APIKeyEnc     string         `json:"-" gorm:"column:api_key_enc;size:1024"`
	HasAPIKey     bool           `json:"has_api_key" gorm:"-"`
	ModelName     string         `json:"model_name" gorm:"size:128;not null"`
	ModelType     string         `json:"model_type" gorm:"size:64;default:chat"` // chat|embedding
	ModelVersion  string         `json:"model_version" gorm:"size:64"`
	Temperature   float64        `json:"temperature" gorm:"type:decimal(4,2);default:0.20"`
	MaxTokens     int            `json:"max_tokens" gorm:"not null;default:4096"`
	ContextLength int            `json:"context_length" gorm:"default:128000"`
	Enabled       bool           `json:"enabled" gorm:"not null;default:true;index"`
	IsDefault     bool           `json:"is_default" gorm:"not null;default:false;index"`
	Remark        string         `json:"remark" gorm:"size:512"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AiLLMModel) TableName() string { return "ai_llm_models" }

// AiPrompt Prompt 逻辑定义。
type AiPrompt struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Code      string         `json:"code" gorm:"size:128;not null;uniqueIndex"` // system/ops-agent
	Name      string         `json:"name" gorm:"size:128;not null"`
	Type      string         `json:"type" gorm:"size:32;not null;index"` // system|diagnosis|...
	Scene     string         `json:"scene" gorm:"size:128;index"`
	Enabled   bool           `json:"enabled" gorm:"not null;default:true;index"`
	Remark    string         `json:"remark" gorm:"size:512"`
	CreatedBy uint           `json:"created_by"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AiPrompt) TableName() string { return "ai_prompts" }

// AiPromptVersion Prompt 版本。
type AiPromptVersion struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	PromptID  uint      `json:"prompt_id" gorm:"not null;index;uniqueIndex:uk_ai_prompt_ver"`
	Version   int       `json:"version" gorm:"not null;uniqueIndex:uk_ai_prompt_ver"`
	Content   string    `json:"content" gorm:"type:longtext;not null"`
	Changelog string    `json:"changelog" gorm:"size:512"`
	IsCurrent bool      `json:"is_current" gorm:"not null;default:false;index"`
	CreatedBy uint      `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

func (AiPromptVersion) TableName() string { return "ai_prompt_versions" }

// AiKnowledgeBase 知识库。
type AiKnowledgeBase struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Code      string         `json:"code" gorm:"size:64;not null;uniqueIndex"` // kb_ops
	Name      string         `json:"name" gorm:"size:128;not null"`
	Category  string         `json:"category" gorm:"size:64;index"` // ops|sop|case|...
	Remark    string         `json:"remark" gorm:"size:512"`
	Enabled   bool           `json:"enabled" gorm:"not null;default:true;index"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AiKnowledgeBase) TableName() string { return "ai_knowledge_bases" }

// AiKbDocument 知识文档。
type AiKbDocument struct {
	ID         uint           `json:"id" gorm:"primaryKey"`
	KBID       uint           `json:"kb_id" gorm:"not null;index"`
	Title      string         `json:"title" gorm:"size:256;not null"`
	Source     string         `json:"source" gorm:"size:256;index"`
	Version    string         `json:"version" gorm:"size:64"`
	Enabled    bool           `json:"enabled" gorm:"not null;default:true;index"`
	Confidence float64        `json:"confidence" gorm:"type:decimal(4,2);default:0.80"`
	Content    string         `json:"content" gorm:"type:longtext"`
	MetaJSON   string         `json:"meta_json" gorm:"type:mediumtext"`
	CreatedBy  uint           `json:"created_by"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AiKbDocument) TableName() string { return "ai_kb_documents" }

// AiKbChunk 文档切片。
type AiKbChunk struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	DocumentID  uint      `json:"document_id" gorm:"not null;index"`
	KBID        uint      `json:"kb_id" gorm:"not null;index"`
	Seq         int       `json:"seq" gorm:"not null"`
	HeadingPath string    `json:"heading_path" gorm:"size:512"`
	Content     string    `json:"content" gorm:"type:mediumtext;not null"`
	MetaJSON    string    `json:"meta_json" gorm:"type:text"`
	Embedding   []byte    `json:"-" gorm:"type:mediumblob"` // 预留向量
	CreatedAt   time.Time `json:"created_at"`
}

func (AiKbChunk) TableName() string { return "ai_kb_chunks" }

// AiIncidentCase 结构化故障案例。
type AiIncidentCase struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	CaseID        string         `json:"case_id" gorm:"size:64;uniqueIndex"`
	Title         string         `json:"title" gorm:"size:256;not null"`
	Category      string         `json:"category" gorm:"size:64;index"`
	Technology    string         `json:"technology" gorm:"size:64;index"`
	Symptom       string         `json:"symptom" gorm:"type:mediumtext"`
	Environment   string         `json:"environment" gorm:"type:text"`
	Diagnosis     string         `json:"diagnosis" gorm:"type:mediumtext"`
	RootCause     string         `json:"root_cause" gorm:"type:mediumtext"`
	Solution      string         `json:"solution" gorm:"type:mediumtext"`
	Verification  string         `json:"verification" gorm:"type:text"`
	Risk          string         `json:"risk" gorm:"type:text"`
	RelatedTools  string         `json:"related_tools" gorm:"type:text"` // JSON array
	RelatedSOP    string         `json:"related_sop" gorm:"type:text"`
	Source        string         `json:"source" gorm:"size:128"`
	Confidence    float64        `json:"confidence" gorm:"type:decimal(4,2);default:0.80"`
	Enabled       bool           `json:"enabled" gorm:"not null;default:true;index"`
	CreatedBy     uint           `json:"created_by"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AiIncidentCase) TableName() string { return "ai_incident_cases" }

// AiSOP 运维 SOP。
type AiSOP struct {
	ID               uint           `json:"id" gorm:"primaryKey"`
	Code             string         `json:"code" gorm:"size:64;uniqueIndex"`
	Title            string         `json:"title" gorm:"size:256;not null"`
	Scenario         string         `json:"scenario" gorm:"type:text"`
	Preconditions    string         `json:"preconditions" gorm:"type:text"`
	InputParams      string         `json:"input_params" gorm:"type:text"`
	CheckSteps       string         `json:"check_steps" gorm:"type:mediumtext"`
	ExecSteps        string         `json:"exec_steps" gorm:"type:mediumtext"`
	VerifySteps      string         `json:"verify_steps" gorm:"type:text"`
	ExceptionHandle  string         `json:"exception_handle" gorm:"type:text"`
	Rollback         string         `json:"rollback" gorm:"type:text"`
	Risk             string         `json:"risk" gorm:"type:text"`
	ApprovalNeeded   bool           `json:"approval_needed" gorm:"not null;default:false"`
	Enabled          bool           `json:"enabled" gorm:"not null;default:true;index"`
	CreatedBy        uint           `json:"created_by"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AiSOP) TableName() string { return "ai_sops" }

// AiToolDef Tool 注册定义。
type AiToolDef struct {
	ID                  uint           `json:"id" gorm:"primaryKey"`
	Name                string         `json:"name" gorm:"size:128;not null;uniqueIndex"`
	Description         string         `json:"description" gorm:"type:text"`
	Module              string         `json:"module" gorm:"size:64;index"` // k8s|log|cicd|script|...
	Runtime             string         `json:"runtime" gorm:"size:32;not null;default:builtin;index"` // builtin|script
	HandlerKey          string         `json:"handler_key" gorm:"size:128"`
	ScriptLang          string         `json:"script_lang" gorm:"size:32"` // python27|go|shell
	ScriptPath          string         `json:"script_path" gorm:"size:512"`
	TimeoutSec          int            `json:"timeout_sec" gorm:"not null;default:30"`
	InputSchemaJSON     string         `json:"input_schema_json" gorm:"type:mediumtext"`
	Permission          string         `json:"permission" gorm:"size:32;not null;default:READ_ONLY"` // READ_ONLY|WRITE
	RiskLevel           string         `json:"risk_level" gorm:"size:32;not null;default:LOW;index"`  // LOW|MEDIUM|HIGH|CRITICAL
	RequireConfirmation bool           `json:"require_confirmation" gorm:"not null;default:false"`
	AuditRequired       bool           `json:"audit_required" gorm:"not null;default:true"`
	Enabled             bool           `json:"enabled" gorm:"not null;default:true;index"`
	Remark              string         `json:"remark" gorm:"size:512"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AiToolDef) TableName() string { return "ai_tool_defs" }

// AiEvalCase 评估用例。
type AiEvalCase struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	Suite           string         `json:"suite" gorm:"size:64;index;default:default"`
	CaseCode        string         `json:"case_code" gorm:"size:64;not null;uniqueIndex"`
	Title           string         `json:"title" gorm:"size:256"`
	InputQuestion   string         `json:"input_question" gorm:"type:mediumtext;not null"`
	ExpectKeywords  string         `json:"expect_keywords" gorm:"type:text"`  // JSON array
	ForbidKeywords  string         `json:"forbid_keywords" gorm:"type:text"`  // JSON array
	ExpectTools     string         `json:"expect_tools" gorm:"type:text"`     // JSON array
	ExpectRisk      string         `json:"expect_risk" gorm:"size:32"`
	ScoreWeight     int            `json:"score_weight" gorm:"not null;default:10"`
	Enabled         bool           `json:"enabled" gorm:"not null;default:true"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AiEvalCase) TableName() string { return "ai_eval_cases" }

// AiEvalRun 评估运行。
type AiEvalRun struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Suite        string    `json:"suite" gorm:"size:64;index"`
	Status       string    `json:"status" gorm:"size:32;index"` // running|done|failed
	TotalScore   float64   `json:"total_score"`
	MaxScore     float64   `json:"max_score"`
	Summary      string    `json:"summary" gorm:"type:text"`
	CreatedBy    uint      `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
}

func (AiEvalRun) TableName() string { return "ai_eval_runs" }

// AiEvalResult 单用例结果。
type AiEvalResult struct {
	ID        uint    `json:"id" gorm:"primaryKey"`
	RunID     uint    `json:"run_id" gorm:"not null;index"`
	CaseID    uint    `json:"case_id" gorm:"not null;index"`
	CaseCode  string  `json:"case_code" gorm:"size:64"`
	Passed    bool    `json:"passed"`
	Score     float64 `json:"score"`
	MaxScore  float64 `json:"max_score"`
	Detail    string  `json:"detail" gorm:"type:mediumtext"`
	Reply     string  `json:"reply" gorm:"type:mediumtext"`
}

func (AiEvalResult) TableName() string { return "ai_eval_results" }

// AiAuditEvent AI 域审计事件。
type AiAuditEvent struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"index"`
	SessionID uint      `json:"session_id" gorm:"index"`
	Action    string    `json:"action" gorm:"size:64;index"` // chat|tool|rag|approve|eval
	ToolName  string    `json:"tool_name" gorm:"size:128;index"`
	RiskLevel string    `json:"risk_level" gorm:"size:32"`
	OK        bool      `json:"ok"`
	DetailJSON string   `json:"detail_json" gorm:"type:mediumtext"`
	CreatedAt time.Time `json:"created_at" gorm:"index"`
}

func (AiAuditEvent) TableName() string { return "ai_audit_events" }
