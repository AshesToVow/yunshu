package system

import (
	"context"
	"strings"

	"yunshu/internal/dictcategory"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/dictmask"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/pagination"

	"gorm.io/gorm"
)

func (s *DictEntryService) List(ctx context.Context, query DictEntryListQuery) (*pagination.Result[DictEntryListItem], error) {
	s.ensureBuiltins(ctx)
	query.DictType = canonicalDictType(query.DictType)
	var excludeTypes []string
	if query.ExcludeSensitive {
		excludeTypes = s.sensitiveDictTypes(ctx)
	}
	page, pageSize := pagination.Normalize(query.Page, query.PageSize)
	list, total, err := s.repo.List(ctx, query.DictType, query.Keyword, dictcategory.NormalizeID(query.Category), query.Status, excludeTypes, page, pageSize)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "dict", "List", err)
	}
	items := make([]DictEntryListItem, 0, len(list))
	for _, item := range list {
		items = append(items, maskDictEntryListItem(item))
	}
	return &pagination.Result[DictEntryListItem]{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// sensitiveDictTypes 返回当前库中属于「敏感凭据」的字典类型。
func (s *DictEntryService) sensitiveDictTypes(ctx context.Context) []string {
	types, err := s.repo.ListDistinctTypes(ctx)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(types))
	for _, t := range types {
		if dictmask.SensitiveDictType(t) {
			out = append(out, t)
		}
	}
	return out
}

// Get 返回字典条目原始记录（含明文），仅供服务端鉴权与敏感值提交判定使用。
func (s *DictEntryService) Get(ctx context.Context, id uint) (*model.DictEntry, error) {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrNotFoundWithMsg(constants.ErrMsg094b285159a4)
		}
		return nil, bizerrors.Pass(ctx, "dict", "Get", err)
	}
	return item, nil
}

func (s *DictEntryService) Create(ctx context.Context, req DictEntryCreateRequest) (*model.DictEntry, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	s.ensureBuiltins(ctx)
	rawVal := req.Value
	if err := validateDictEntryValueBytes(rawVal); err != nil {
		return nil, bizerrors.Pass(ctx, "dict", "Create", err)
	}
	item := model.DictEntry{
		DictType: canonicalDictType(req.DictType),
		Label:    strings.TrimSpace(req.Label),
		Value:    strings.TrimSpace(rawVal),
		Sort:     dictEntrySort(req.Sort),
		Status:   req.Status,
		Remark:   strings.TrimSpace(req.Remark),
	}
	if item.DictType == "" || item.Label == "" || item.Value == "" {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg0adab830348f)
	}
	if isSingletonDictType(item.DictType) {
		if exists, err := s.repo.ExistsByType(ctx, item.DictType, 0); err == nil && exists {
			return nil, constants.ErrBadRequestWithMsg("该字典类型为单值型，仅允许一条记录")
		}
	}
	if exists, err := s.repo.ExistsByTypeLabel(ctx, item.DictType, item.Label, 0); err == nil && exists {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg7e043b9a81af)
	}
	if exists, err := s.repo.ExistsByTypeValue(ctx, item.DictType, item.Value, 0); err == nil && exists {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg9ea86777037d)
	}
	if err := s.repo.Create(ctx, &item); err != nil {
		return nil, bizerrors.Pass(ctx, "dict", "Create", err)
	}
	return &item, nil
}

func (s *DictEntryService) Update(ctx context.Context, id uint, req DictEntryUpdateRequest) (*model.DictEntry, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrNotFoundWithMsg(constants.ErrMsg094b285159a4)
		}
		return nil, bizerrors.Pass(ctx, "dict", "Update", err)
	}
	rawVal := req.Value
	if err := validateDictEntryValueBytes(rawVal); err != nil {
		return nil, bizerrors.Pass(ctx, "dict", "Update", err)
	}
	oldValue := strings.TrimSpace(item.Value)
	oldType := canonicalDictType(item.DictType)
	oldLabel := strings.TrimSpace(item.Label)
	oldSensitive := dictmask.SensitiveDictType(item.DictType)
	item.DictType = strings.TrimSpace(req.DictType)
	item.DictType = canonicalDictType(item.DictType)
	item.Label = strings.TrimSpace(req.Label)
	// 列表返回的是脱敏预览：前端原样回传时视为“值未修改”，保留库中明文。
	// 若需要真正改成与预览相同的字面量，需通过新建条目完成。
	item.Value = resolveDictEntrySubmittedValue(rawVal, oldValue, oldSensitive, dictmask.SensitiveDictType(item.DictType))
	item.Sort = dictEntrySort(req.Sort)
	item.Status = req.Status
	item.Remark = strings.TrimSpace(req.Remark)
	if item.DictType == "" || item.Label == "" || item.Value == "" {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg0adab830348f)
	}
	if isBuiltinDictEntry(oldType, oldLabel) && (item.DictType != oldType || item.Label != oldLabel) {
		return nil, constants.ErrBadRequestWithMsg("内置字典项不允许修改字典类型或标签，可调整值、状态或备注")
	}
	// 敏感→普通迁移会把库中明文凭据暴露在普通字典列表里，一律拒绝；反向（普通→敏感）允许。
	if oldSensitive && !dictmask.SensitiveDictType(item.DictType) {
		return nil, constants.ErrBadRequestWithMsg("敏感字典项不允许迁移为普通字典类型，请新建普通条目并停用原条目")
	}
	if item.DictType != oldType && isSingletonDictType(item.DictType) {
		if exists, err2 := s.repo.ExistsByType(ctx, item.DictType, item.ID); err2 == nil && exists {
			return nil, constants.ErrBadRequestWithMsg("该字典类型为单值型，仅允许一条记录")
		}
	}
	if exists, err2 := s.repo.ExistsByTypeLabel(ctx, item.DictType, item.Label, item.ID); err2 == nil && exists {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg47f29b52ac8f)
	}
	if exists, err2 := s.repo.ExistsByTypeValue(ctx, item.DictType, item.Value, item.ID); err2 == nil && exists {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg1ffcbfd43034)
	}
	if err = s.repo.Update(ctx, item); err != nil {
		return nil, bizerrors.Pass(ctx, "dict", "Update", err)
	}
	return item, nil
}

func (s *DictEntryService) Delete(ctx context.Context, id uint) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return constants.ErrNotFoundWithMsg(constants.ErrMsg094b285159a4)
		}
		return bizerrors.Pass(ctx, "dict", "Delete", err)
	}
	if isBuiltinDictEntry(item.DictType, item.Label) {
		return constants.ErrBadRequestWithMsg("内置字典项不允许删除，可停用或修改值")
	}
	return s.repo.Delete(ctx, id)
}

func (s *DictEntryService) Options(ctx context.Context, dictType string) ([]DictEntryOption, error) {
	s.ensureBuiltins(ctx)
	canon := canonicalDictType(dictType)
	list, err := s.repo.ListByTypeEnabled(ctx, canon)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "dict", "Options", err)
	}
	sensitiveType := dictmask.SensitiveDictType(canon)
	options := make([]DictEntryOption, 0, len(list))
	for _, item := range list {
		v := item.Value
		if sensitiveType {
			v = dictmask.Preview(item.Value)
		}
		options = append(options, DictEntryOption{
			ID:        item.ID,
			Label:     item.Label,
			Value:     v,
			Sensitive: sensitiveType,
		})
	}
	return options, nil
}

// RevealValue 返回敏感字典条目的明文（仅用于表单从字典填充；需配合独立审计策略）。
func (s *DictEntryService) RevealValue(ctx context.Context, id uint) (string, error) {
	s.ensureBuiltins(ctx)
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", constants.ErrNotFoundWithMsg(constants.ErrMsg094b285159a4)
		}
		return "", bizerrors.Pass(ctx, "dict", "RevealValue", err)
	}
	dt := canonicalDictType(item.DictType)
	if !dictmask.SensitiveDictType(dt) {
		return "", constants.ErrBadRequestWithMsg("该字典类型不支持通过此接口获取明文")
	}
	return item.Value, nil
}

// maskDictEntryListItem 敏感类型的 value 仅返回脱敏预览，避免列表/审计链路泄漏明文。
func maskDictEntryListItem(item model.DictEntry) DictEntryListItem {
	sensitive := dictmask.SensitiveDictType(item.DictType)
	if sensitive {
		item.Value = dictmask.Preview(item.Value)
	}
	return DictEntryListItem{
		DictEntry: item,
		Sensitive: sensitive,
		Builtin:   isBuiltinDictEntry(item.DictType, item.Label),
	}
}

// resolveDictEntrySubmittedValue 前端原样回传脱敏预览时视为“值未修改”，保留库中明文。
func resolveDictEntrySubmittedValue(rawVal, oldValue string, oldSensitive, newSensitive bool) string {
	trimmed := strings.TrimSpace(rawVal)
	if trimmed != "" && (oldSensitive || newSensitive) && trimmed == dictmask.Preview(oldValue) {
		return oldValue
	}
	return trimmed
}
