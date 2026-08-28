package platformtpl

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/objectstore"
	"yunshu/internal/pkg/pagination"

	"gorm.io/gorm"
)

// Service 平台模板中心：目录 CRUD、版本、发布、解析。
type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

type ListQuery struct {
	Category string `form:"category"`
	Keyword  string `form:"keyword"`
	Status   *int   `form:"status"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type TemplateItem struct {
	model.PlatformTemplate
	PublishedChecksum string `json:"published_checksum,omitempty"`
	HasMinIOMirror    bool   `json:"has_minio_mirror"`
}

type UpsertRequest struct {
	TemplateKey string `json:"template_key"`
	Category    string `json:"category" binding:"required"`
	Name        string `json:"name" binding:"required,max=128"`
	Format      string `json:"format"`
	Description string `json:"description" binding:"omitempty,max=512"`
	Status      *int   `json:"status"`
}

type SaveDraftRequest struct {
	Content string `json:"content" binding:"required"`
	Remark  string `json:"remark" binding:"omitempty,max=512"`
}

type VersionItem struct {
	model.PlatformTemplateVersion
	ContentPreview string `json:"content_preview,omitempty"`
}

type ResolveResult struct {
	TemplateKey string `json:"template_key"`
	Version     int    `json:"version"`
	Format      string `json:"format"`
	Content     string `json:"content"`
	Source      string `json:"source"` // published | builtin | draft
}

func (s *Service) List(ctx context.Context, q ListQuery) (*pagination.Result[TemplateItem], error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	tx := s.db.WithContext(ctx).Model(&model.PlatformTemplate{})
	if c := strings.TrimSpace(q.Category); c != "" {
		tx = tx.Where("category = ?", c)
	}
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		tx = tx.Where("template_key LIKE ? OR name LIKE ? OR description LIKE ?", like, like, like)
	}
	if q.Status != nil {
		tx = tx.Where("status = ?", *q.Status)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, bizerrors.Pass(ctx, "platformtpl", "List", err)
	}
	var rows []model.PlatformTemplate
	if err := tx.Order("category ASC, template_key ASC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, bizerrors.Pass(ctx, "platformtpl", "List", err)
	}
	items := make([]TemplateItem, 0, len(rows))
	for _, row := range rows {
		item := TemplateItem{PlatformTemplate: row}
		if row.PublishedVersion > 0 {
			var ver model.PlatformTemplateVersion
			if err := s.db.WithContext(ctx).
				Where("template_id = ? AND version = ?", row.ID, row.PublishedVersion).
				First(&ver).Error; err == nil {
				item.PublishedChecksum = ver.Checksum
				item.HasMinIOMirror = strings.TrimSpace(ver.StorageKey) != ""
			}
		}
		items = append(items, item)
	}
	return &pagination.Result[TemplateItem]{List: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *Service) Detail(ctx context.Context, id uint) (*TemplateItem, error) {
	var row model.PlatformTemplate
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, bizerrors.Pass(ctx, "platformtpl", "Detail", err)
	}
	item := &TemplateItem{PlatformTemplate: row}
	if row.PublishedVersion > 0 {
		var ver model.PlatformTemplateVersion
		if err := s.db.WithContext(ctx).
			Where("template_id = ? AND version = ?", row.ID, row.PublishedVersion).
			First(&ver).Error; err == nil {
			item.PublishedChecksum = ver.Checksum
			item.HasMinIOMirror = strings.TrimSpace(ver.StorageKey) != ""
		}
	}
	return item, nil
}

func (s *Service) Create(ctx context.Context, req UpsertRequest, actorID uint) (*TemplateItem, error) {
	key := strings.TrimSpace(req.TemplateKey)
	if key == "" {
		return nil, constants.ErrBadRequestWithMsg("template_key 必填")
	}
	cat := normalizeCategory(req.Category)
	if cat == "" {
		return nil, constants.ErrBadRequestWithMsg("不支持的 category")
	}
	format := strings.TrimSpace(req.Format)
	if format == "" {
		format = model.PlatformTemplateFormatText
	}
	status := model.PlatformTemplateStatusEnabled
	if req.Status != nil {
		status = *req.Status
	}
	row := model.PlatformTemplate{
		TemplateKey: key,
		Category:    cat,
		Name:        strings.TrimSpace(req.Name),
		Format:      format,
		Description: strings.TrimSpace(req.Description),
		Status:      status,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, bizerrors.Pass(ctx, "platformtpl", "Create", err)
	}
	_ = actorID
	return s.Detail(ctx, row.ID)
}

func (s *Service) Update(ctx context.Context, id uint, req UpsertRequest) (*TemplateItem, error) {
	var row model.PlatformTemplate
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, bizerrors.Pass(ctx, "platformtpl", "Update", err)
	}
	if cat := normalizeCategory(req.Category); cat != "" {
		row.Category = cat
	}
	if name := strings.TrimSpace(req.Name); name != "" {
		row.Name = name
	}
	if f := strings.TrimSpace(req.Format); f != "" {
		row.Format = f
	}
	row.Description = strings.TrimSpace(req.Description)
	if req.Status != nil {
		row.Status = *req.Status
	}
	// template_key 创建后不可改，避免破坏引用
	if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
		return nil, bizerrors.Pass(ctx, "platformtpl", "Update", err)
	}
	return s.Detail(ctx, row.ID)
}

func (s *Service) Delete(ctx context.Context, id uint) error {
	var row model.PlatformTemplate
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrNotFound
		}
		return bizerrors.Pass(ctx, "platformtpl", "Delete", err)
	}
	if row.IsBuiltin {
		return constants.ErrBadRequestWithMsg("内置模板不可删除，可停用或发布新版本覆盖")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("template_id = ?", id).Delete(&model.PlatformTemplateVersion{}).Error; err != nil {
			return err
		}
		return tx.Delete(&row).Error
	})
}

// SaveDraft 新增一版草稿（未自动发布）。
func (s *Service) SaveDraft(ctx context.Context, id uint, req SaveDraftRequest, actorID uint) (*VersionItem, error) {
	var row model.PlatformTemplate
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, bizerrors.Pass(ctx, "platformtpl", "SaveDraft", err)
	}
	content := req.Content
	sum := checksum(content)
	var maxVer int
	_ = s.db.WithContext(ctx).Model(&model.PlatformTemplateVersion{}).
		Where("template_id = ?", id).Select("COALESCE(MAX(version),0)").Scan(&maxVer).Error
	ver := model.PlatformTemplateVersion{
		TemplateID:    id,
		Version:       maxVer + 1,
		ContentInline: content,
		Checksum:      sum,
		Remark:        strings.TrimSpace(req.Remark),
		CreatedBy:     actorID,
		CreatedAt:     time.Now(),
	}
	ver.StorageKey = s.tryMirrorMinIO(ctx, row.TemplateKey, ver.Version, content, row.Format)
	if err := s.db.WithContext(ctx).Create(&ver).Error; err != nil {
		return nil, bizerrors.Pass(ctx, "platformtpl", "SaveDraft", err)
	}
	return &VersionItem{PlatformTemplateVersion: ver, ContentPreview: preview(content)}, nil
}

// Publish 将指定版本（或最新草稿）设为发布态。
func (s *Service) Publish(ctx context.Context, id uint, version int) (*TemplateItem, error) {
	var row model.PlatformTemplate
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, bizerrors.Pass(ctx, "platformtpl", "Publish", err)
	}
	var ver model.PlatformTemplateVersion
	q := s.db.WithContext(ctx).Where("template_id = ?", id)
	if version > 0 {
		q = q.Where("version = ?", version)
	} else {
		q = q.Order("version DESC")
	}
	if err := q.First(&ver).Error; err != nil {
		return nil, constants.ErrBadRequestWithMsg("无可发布版本，请先保存草稿")
	}
	if strings.TrimSpace(ver.StorageKey) == "" {
		ver.StorageKey = s.tryMirrorMinIO(ctx, row.TemplateKey, ver.Version, ver.ContentInline, row.Format)
		if ver.StorageKey != "" {
			_ = s.db.WithContext(ctx).Model(&ver).Update("storage_key", ver.StorageKey).Error
		}
	}
	if err := s.db.WithContext(ctx).Model(&row).Update("published_version", ver.Version).Error; err != nil {
		return nil, bizerrors.Pass(ctx, "platformtpl", "Publish", err)
	}
	return s.Detail(ctx, id)
}

func (s *Service) ListVersions(ctx context.Context, id uint) ([]VersionItem, error) {
	var rows []model.PlatformTemplateVersion
	if err := s.db.WithContext(ctx).Where("template_id = ?", id).
		Order("version DESC").Find(&rows).Error; err != nil {
		return nil, bizerrors.Pass(ctx, "platformtpl", "ListVersions", err)
	}
	out := make([]VersionItem, 0, len(rows))
	for _, r := range rows {
		item := VersionItem{PlatformTemplateVersion: r, ContentPreview: preview(r.ContentInline)}
		item.ContentInline = "" // 列表不回全文
		out = append(out, item)
	}
	return out, nil
}

func (s *Service) GetVersionContent(ctx context.Context, id uint, version int) (*VersionItem, error) {
	var ver model.PlatformTemplateVersion
	if err := s.db.WithContext(ctx).
		Where("template_id = ? AND version = ?", id, version).First(&ver).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, bizerrors.Pass(ctx, "platformtpl", "GetVersionContent", err)
	}
	if strings.TrimSpace(ver.ContentInline) == "" && ver.StorageKey != "" {
		if body, err := s.loadMinIO(ctx, ver.StorageKey); err == nil {
			ver.ContentInline = string(body)
		}
	}
	return &VersionItem{PlatformTemplateVersion: ver}, nil
}

// ResolvePublished 业务侧解析：已发布正文；无则 builtin 种子内容。
func (s *Service) ResolvePublished(ctx context.Context, templateKey string) (*ResolveResult, error) {
	key := strings.TrimSpace(templateKey)
	if key == "" {
		return nil, constants.ErrBadRequestWithMsg("template_key 必填")
	}
	var row model.PlatformTemplate
	err := s.db.WithContext(ctx).
		Where("template_key = ? AND status = ?", key, model.PlatformTemplateStatusEnabled).
		First(&row).Error
	if err == nil && row.PublishedVersion > 0 {
		ver, err := s.GetVersionContent(ctx, row.ID, row.PublishedVersion)
		if err == nil && strings.TrimSpace(ver.ContentInline) != "" {
			return &ResolveResult{
				TemplateKey: key, Version: ver.Version, Format: row.Format,
				Content: ver.ContentInline, Source: "published",
			}, nil
		}
	}
	if body, format, ok := BuiltinContent(key); ok {
		return &ResolveResult{
			TemplateKey: key, Version: 0, Format: format,
			Content: body, Source: "builtin",
		}, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, bizerrors.Pass(ctx, "platformtpl", "ResolvePublished", err)
	}
	return nil, constants.ErrNotFoundWithMsg("模板未发布且无内置正文: " + key)
}

func (s *Service) tryMirrorMinIO(ctx context.Context, templateKey string, version int, content, format string) string {
	cli, err := objectstore.NewFromDB(ctx, s.db)
	if err != nil || cli == nil {
		return ""
	}
	key := fmt.Sprintf("platform-templates/%s/v%d", sanitizeKey(templateKey), version)
	ct := contentTypeOf(format)
	if err := cli.PutBytes(ctx, key, []byte(content), ct); err != nil {
		return ""
	}
	return key
}

func (s *Service) loadMinIO(ctx context.Context, storageKey string) ([]byte, error) {
	cli, err := objectstore.NewFromDB(ctx, s.db)
	if err != nil {
		return nil, err
	}
	return cli.GetBytes(ctx, storageKey)
}

func normalizeCategory(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case model.PlatformTemplateCategoryCicdSnippet, "cicd", "cicd-snippet":
		return model.PlatformTemplateCategoryCicdSnippet
	case model.PlatformTemplateCategoryAlert:
		return model.PlatformTemplateCategoryAlert
	case model.PlatformTemplateCategoryInspect:
		return model.PlatformTemplateCategoryInspect
	case model.PlatformTemplateCategoryLoggie:
		return model.PlatformTemplateCategoryLoggie
	default:
		return ""
	}
}

func checksum(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func preview(s string) string {
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) > 200 {
		return string(runes[:200]) + "…"
	}
	return s
}

func sanitizeKey(k string) string {
	k = strings.ReplaceAll(k, "..", "")
	k = strings.ReplaceAll(k, "\\", "/")
	return strings.Trim(k, "/")
}

func contentTypeOf(format string) string {
	switch format {
	case model.PlatformTemplateFormatYAML:
		return "application/x-yaml"
	case model.PlatformTemplateFormatHTML:
		return "text/html; charset=utf-8"
	case model.PlatformTemplateFormatShell:
		return "text/x-shellscript"
	default:
		return "text/plain; charset=utf-8"
	}
}
