package system

import (
	"context"
	"strings"

	"yunshu/internal/dictcategory"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/dictmask"
	"yunshu/internal/pkg/pagination"

	"gorm.io/gorm"
)

func (s *DictEntryService) List(ctx context.Context, query DictEntryListQuery) (*pagination.Result[model.DictEntry], error) {
	s.ensureBuiltins(ctx)
	query.DictType = canonicalDictType(query.DictType)
	page, pageSize := pagination.Normalize(query.Page, query.PageSize)
	list, total, err := s.repo.List(ctx, query.DictType, query.Keyword, dictcategory.NormalizeID(query.Category), query.Status, page, pageSize)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "dict", "List", err)
	}
	return &pagination.Result[model.DictEntry]{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *DictEntryService) Create(ctx context.Context, req DictEntryCreateRequest) (*model.DictEntry, error) {
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
	item.DictType = strings.TrimSpace(req.DictType)
	item.DictType = canonicalDictType(item.DictType)
	item.Label = strings.TrimSpace(req.Label)
	item.Value = strings.TrimSpace(rawVal)
	item.Sort = dictEntrySort(req.Sort)
	item.Status = req.Status
	item.Remark = strings.TrimSpace(req.Remark)
	if item.DictType == "" || item.Label == "" || item.Value == "" {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg0adab830348f)
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
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return constants.ErrNotFoundWithMsg(constants.ErrMsg094b285159a4)
		}
		return bizerrors.Pass(ctx, "dict", "Delete", err)
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
