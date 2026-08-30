package system

import (
	"fmt"
	"strings"
	"sync"

	"yunshu/internal/interfaces"
	"yunshu/internal/pkg/constants"
)

// 与 MySQL MEDIUMTEXT 上限一致的量级；在 binding 层不设 max，避免大 kubeconfig 被 ShouldBindJSON 拒绝。
const dictEntryValueMaxBytes = 16 << 20 // 16 MiB

func validateDictEntryValueBytes(v string) error {
	if len(v) > dictEntryValueMaxBytes {
		return constants.ErrBadRequestWithMsg(fmt.Sprintf(constants.ErrFmtd1b9788a27bb, len(v), dictEntryValueMaxBytes))
	}
	return nil
}

func intRef(v int) *int {
	p := v
	return &p
}

// dictEntrySort 将 JSON 中的 null/省略映射为 0（前端 InputNumber 清空会提交 null，不能直接绑定到 int）。
func dictEntrySort(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

type DictEntryListQuery struct {
	DictType string `form:"dict_type"`
	Category string `form:"category"`
	Keyword  string `form:"keyword"`
	Status   *int   `form:"status"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type DictEntryCreateRequest struct {
	DictType string `json:"dict_type" binding:"required,max=64"`
	Label    string `json:"label" binding:"required,max=128"`
	Value    string `json:"value" binding:"required"`
	Sort     *int   `json:"sort"`
	Status   int    `json:"status" binding:"oneof=0 1"`
	Remark   string `json:"remark" binding:"omitempty,max=512"`
}

type DictEntryUpdateRequest struct {
	DictType string `json:"dict_type" binding:"required,max=64"`
	Label    string `json:"label" binding:"required,max=128"`
	Value    string `json:"value" binding:"required"`
	Sort     *int   `json:"sort"`
	Status   int    `json:"status" binding:"oneof=0 1"`
	Remark   string `json:"remark" binding:"omitempty,max=512"`
}

type DictEntryOption struct {
	ID        uint   `json:"id"`
	Label     string `json:"label"`
	Value     string `json:"value"` // 非敏感为真实值；敏感类型为脱敏预览（明文仅 POST reveal-value）
	Sensitive bool   `json:"sensitive"`
}

type DictEntryService struct {
	repo     interfaces.DictEntryRepository
	initOnce sync.Once
}

const (
	dictTypeAlertPromQLLabelKey     = "alert_promql_label_key"
	dictTypeAlertSilenceMatcherName = "alert_silence_matcher_name"
)

func canonicalDictType(dictType string) string {
	t := strings.ToLower(strings.TrimSpace(dictType))
	if t == dictTypeAlertSilenceMatcherName {
		return dictTypeAlertPromQLLabelKey
	}
	return t
}

func NewDictEntryService(repo interfaces.DictEntryRepository) *DictEntryService {
	return &DictEntryService{repo: repo}
}
