package repository

import (
	"context"

	"yunshu/internal/model"
)

// AiApprovalListParams filters ai_tool_approvals listing.
type AiApprovalListParams struct {
	Status   string
	UserID   uint // when RestrictUser, filter by this user
	RestrictUser bool
	Offset   int
	Limit    int
}

// AiSessionListParams filters ai_chat_sessions listing.
type AiSessionListParams struct {
	UserID uint
	Offset int
	Limit  int
}

// AiInvestigationListParams filters ai_investigations listing.
type AiInvestigationListParams struct {
	UserID uint
	Kind   string
	Status string
	Offset int
	Limit  int
}

// AiRepo is implemented by *AiRepository.
type AiRepo interface {
	Transaction(ctx context.Context, fn func(AiRepo) error) error

	// --- Audit ---
	CreateAuditEvent(ctx context.Context, ev *model.AiAuditEvent) error

	// --- Prompt ---
	ListPrompts(ctx context.Context) ([]model.AiPrompt, error)
	GetPromptByID(ctx context.Context, id uint) (*model.AiPrompt, error)
	GetPromptByCode(ctx context.Context, code string) (*model.AiPrompt, error)
	GetPromptByCodeEnabled(ctx context.Context, code string) (*model.AiPrompt, error)
	CreatePrompt(ctx context.Context, row *model.AiPrompt) error
	UpdatePrompt(ctx context.Context, id uint, updates map[string]any) error
	DeletePromptCascade(ctx context.Context, id uint) error
	CreatePromptWithVersionTx(ctx context.Context, prompt *model.AiPrompt, version *model.AiPromptVersion) error
	ListPromptVersions(ctx context.Context, promptID uint) ([]model.AiPromptVersion, error)
	GetCurrentPromptVersion(ctx context.Context, promptID uint) (*model.AiPromptVersion, error)
	GetPromptVersionByID(ctx context.Context, promptID, versionID uint) (*model.AiPromptVersion, error)
	GetPromptMaxVersion(ctx context.Context, promptID uint) (int, error)
	PublishPromptVersion(ctx context.Context, promptID, userID uint, content, changelog string) (*model.AiPromptVersion, error)
	RollbackPromptVersion(ctx context.Context, promptID, versionID uint) error
	CountPromptVersions(ctx context.Context, promptID uint) (int64, error)
	ClearPromptCurrentVersions(ctx context.Context, promptID uint) error
	CountPrompts(ctx context.Context) (int64, error)
	SeedPromptVersionUpdate(ctx context.Context, promptID uint, content string, nextVersion int) error

	// --- LLM Model ---
	ListLLMModels(ctx context.Context) ([]model.AiLLMModel, error)
	ListEnabledLLMModels(ctx context.Context) ([]model.AiLLMModel, error)
	CountEnabledLLMModels(ctx context.Context) (int64, error)
	GetLLMModelByID(ctx context.Context, id uint) (*model.AiLLMModel, error)
	CreateLLMModel(ctx context.Context, row *model.AiLLMModel, clearDefault bool) error
	UpdateLLMModel(ctx context.Context, id uint, updates map[string]any, clearOtherDefault bool) error
	DeleteLLMModel(ctx context.Context, id uint) error
	SetDefaultLLMModel(ctx context.Context, id uint) error
	FindLLMModelByName(ctx context.Context, name string) (*model.AiLLMModel, error)
	FindLLMModelByProvider(ctx context.Context, provider string) (*model.AiLLMModel, error)
	FindDefaultLLMModel(ctx context.Context) (*model.AiLLMModel, error)
	FindFirstEnabledLLMModel(ctx context.Context) (*model.AiLLMModel, error)
	FindEmbeddingModel(ctx context.Context) (*model.AiLLMModel, error)

	// --- Tool ---
	ListTools(ctx context.Context) ([]model.AiToolDef, error)
	ListEnabledScriptTools(ctx context.Context) ([]model.AiToolDef, error)
	GetToolByID(ctx context.Context, id uint) (*model.AiToolDef, error)
	GetToolByName(ctx context.Context, name string) (*model.AiToolDef, error)
	CreateTool(ctx context.Context, row *model.AiToolDef) error
	UpdateTool(ctx context.Context, id uint, updates map[string]any) error
	DeleteToolRow(ctx context.Context, row *model.AiToolDef) error
	UpdateToolEnabled(ctx context.Context, id uint, enabled bool) error
	CountToolsByRuntime(ctx context.Context, runtime string) (int64, error)
	CountAllTools(ctx context.Context) (int64, error)
	CountToolsByName(ctx context.Context, name string) (int64, error)

	// --- SOP ---
	ListSOPs(ctx context.Context, limit int) ([]model.AiSOP, error)
	ListEnabledSOPs(ctx context.Context, limit int) ([]model.AiSOP, error)
	GetSOPByID(ctx context.Context, id uint) (*model.AiSOP, error)
	CreateSOP(ctx context.Context, row *model.AiSOP) error
	UpdateSOP(ctx context.Context, id uint, updates map[string]any) error
	DeleteSOP(ctx context.Context, id uint) error
	CountSOPs(ctx context.Context) (int64, error)
	CountSOPByCode(ctx context.Context, code string) (int64, error)

	// --- Incident Case ---
	ListIncidentCases(ctx context.Context, limit int) ([]model.AiIncidentCase, error)
	ListEnabledIncidentCases(ctx context.Context, limit int) ([]model.AiIncidentCase, error)
	GetIncidentCaseByID(ctx context.Context, id uint) (*model.AiIncidentCase, error)
	CreateIncidentCase(ctx context.Context, row *model.AiIncidentCase) error
	UpdateIncidentCase(ctx context.Context, id uint, updates map[string]any) error
	DeleteIncidentCase(ctx context.Context, id uint) error
	CountIncidentCases(ctx context.Context) (int64, error)
	CountIncidentCaseByCaseID(ctx context.Context, caseID string) (int64, error)

	// --- Knowledge Base ---
	ListKnowledgeBases(ctx context.Context) ([]model.AiKnowledgeBase, error)
	GetKnowledgeBaseByID(ctx context.Context, id uint) (*model.AiKnowledgeBase, error)
	GetKnowledgeBaseByCode(ctx context.Context, code string) (*model.AiKnowledgeBase, error)
	CreateKnowledgeBase(ctx context.Context, row *model.AiKnowledgeBase) error
	UpdateKnowledgeBase(ctx context.Context, id uint, updates map[string]any) error
	DeleteKnowledgeBaseCascade(ctx context.Context, id uint) error
	CountKnowledgeBases(ctx context.Context) (int64, error)

	// --- KB Document ---
	ListKBDocuments(ctx context.Context, kbID uint, limit int) ([]model.AiKbDocument, error)
	ListEnabledKBDocuments(ctx context.Context) ([]model.AiKbDocument, error)
	GetKBDocumentByID(ctx context.Context, id uint) (*model.AiKbDocument, error)
	CreateKBDocument(ctx context.Context, row *model.AiKbDocument) error
	UpdateKBDocument(ctx context.Context, id uint, updates map[string]any) error
	DeleteKBDocumentCascade(ctx context.Context, id uint) error
	DeleteKBDocumentChunks(ctx context.Context, documentID uint) error
	CountKBDocuments(ctx context.Context) (int64, error)
	CountKBDocumentBySource(ctx context.Context, kbID uint, source string) (int64, error)

	// --- KB Chunk ---
	DeleteChunksByDocumentID(ctx context.Context, documentID uint) error
	CreateChunk(ctx context.Context, ch *model.AiKbChunk) error
	ListRecentChunks(ctx context.Context, limit int) ([]model.AiKbChunk, error)
	ListChunksMissingEmbedding(ctx context.Context, limit int) ([]model.AiKbChunk, error)
	UpdateChunkEmbedding(ctx context.Context, chunkID uint, embedding []byte) error

	// --- Eval ---
	ListEvalCases(ctx context.Context) ([]model.AiEvalCase, error)
	ListEnabledEvalCases(ctx context.Context) ([]model.AiEvalCase, error)
	GetEvalCaseByID(ctx context.Context, id uint) (*model.AiEvalCase, error)
	CreateEvalCase(ctx context.Context, row *model.AiEvalCase) error
	UpdateEvalCase(ctx context.Context, id uint, updates map[string]any) error
	DeleteEvalCase(ctx context.Context, id uint) error
	CountEvalCases(ctx context.Context) (int64, error)
	CountEvalCaseByCaseCode(ctx context.Context, caseCode string) (int64, error)
	CreateEvalRun(ctx context.Context, run *model.AiEvalRun) error
	SaveEvalRun(ctx context.Context, run *model.AiEvalRun) error
	ListEvalRuns(ctx context.Context, limit int) ([]model.AiEvalRun, error)
	GetEvalRunByID(ctx context.Context, id uint) (*model.AiEvalRun, error)
	ListEvalResultsByRunID(ctx context.Context, runID uint) ([]model.AiEvalResult, error)
	CreateEvalResult(ctx context.Context, result *model.AiEvalResult) error

	// --- Approval ---
	CreateApproval(ctx context.Context, row *model.AiToolApproval) error
	DeleteApproval(ctx context.Context, id uint) error
	ListApprovals(ctx context.Context, p AiApprovalListParams) ([]model.AiToolApproval, int64, error)
	GetApprovalByID(ctx context.Context, id uint) (*model.AiToolApproval, error)
	SaveApproval(ctx context.Context, row *model.AiToolApproval) error
	ClaimApprovalExecution(ctx context.Context, id uint, fromStatuses []string) (int64, error)
	UpdateApprovalFields(ctx context.Context, id uint, fields map[string]any) error

	// --- Chat Session / Message ---
	CreateSession(ctx context.Context, row *model.AiChatSession) error
	ListSessions(ctx context.Context, p AiSessionListParams) ([]model.AiChatSession, int64, error)
	GetSessionByUser(ctx context.Context, userID, sessionID uint) (*model.AiChatSession, error)
	UpdateSession(ctx context.Context, id uint, updates map[string]any) error
	DeleteSessionCascade(ctx context.Context, sessionID uint) error
	ClearSessionMessages(ctx context.Context, sessionID uint, sessionUpdates map[string]any) error
	CountSessionMessages(ctx context.Context, sessionIDs []uint) (map[uint]int64, error)
	ListSessionMessages(ctx context.Context, sessionID uint, limit int) ([]model.AiChatMessage, error)
	CreateMessages(ctx context.Context, msgs []model.AiChatMessage) error
	CountChatSessions(ctx context.Context) (int64, error)

	// --- Investigation ---
	CreateInvestigation(ctx context.Context, row *model.AiInvestigation) error
	SaveInvestigation(ctx context.Context, row *model.AiInvestigation) error
	ListInvestigations(ctx context.Context, p AiInvestigationListParams) ([]model.AiInvestigation, int64, error)
	GetInvestigationByUser(ctx context.Context, userID, id uint) (*model.AiInvestigation, error)
	GetInvestigationByApprovalID(ctx context.Context, approvalID uint) (*model.AiInvestigation, error)
	FindInvestigationLinkingApproval(ctx context.Context, approvalID uint) (*model.AiInvestigation, error)
}

var _ AiRepo = (*AiRepository)(nil)
