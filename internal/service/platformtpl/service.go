package platformtpl

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/objectstore"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"

	"gorm.io/gorm"
)

// ObjectStoreFactory resolves MinIO from dict (injected at Wire; nil skips mirror).
type ObjectStoreFactory func(ctx context.Context) (*objectstore.Client, error)

// Service 平台模板中心：目录 CRUD、版本、发布、解析。
type Service struct {
	repo           interfaces.PlatformTemplateRepository
	newObjectStore ObjectStoreFactory
}

func NewService(repo interfaces.PlatformTemplateRepository, newObjectStore ObjectStoreFactory) *Service {
	return &Service{repo: repo, newObjectStore: newObjectStore}
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
	rows, total, err := s.repo.List(ctx, repository.PlatformTemplateListParams{
		Category: strings.TrimSpace(q.Category),
		Keyword:  strings.TrimSpace(q.Keyword),
		Status:   q.Status,
		Offset:   (page - 1) * pageSize,
		Limit:    pageSize,
	})
	if err != nil {
		return nil, bizerrors.Pass(ctx, "platformtpl", "List", err)
	}
	items := make([]TemplateItem, 0, len(rows))
	for _, row := range rows {
		item := TemplateItem{PlatformTemplate: row}
		if row.PublishedVersion > 0 {
			if ver, err := s.repo.GetVersion(ctx, row.ID, row.PublishedVersion); err == nil {
				item.PublishedChecksum = ver.Checksum
				item.HasMinIOMirror = strings.TrimSpace(ver.StorageKey) != ""
			}
		}
		items = append(items, item)
	}
	return &pagination.Result[TemplateItem]{List: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *Service) Detail(ctx context.Context, id uint) (*TemplateItem, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, bizerrors.Pass(ctx, "platformtpl", "Detail", err)
	}
	item := &TemplateItem{PlatformTemplate: *row}
	if row.PublishedVersion > 0 {
		if ver, err := s.repo.GetVersion(ctx, row.ID, row.PublishedVersion); err == nil {
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
	if err := s.repo.Create(ctx, &row); err != nil {
		return nil, bizerrors.Pass(ctx, "platformtpl", "Create", err)
	}
	_ = actorID
	return s.Detail(ctx, row.ID)
}

func (s *Service) Update(ctx context.Context, id uint, req UpsertRequest) (*TemplateItem, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
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
	if err := s.repo.Save(ctx, row); err != nil {
		return nil, bizerrors.Pass(ctx, "platformtpl", "Update", err)
	}
	return s.Detail(ctx, row.ID)
}

func (s *Service) Delete(ctx context.Context, id uint) error {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrNotFound
		}
		return bizerrors.Pass(ctx, "platformtpl", "Delete", err)
	}
	if row.IsBuiltin {
		return constants.ErrBadRequestWithMsg("内置模板不可删除，可停用或发布新版本覆盖")
	}
	return s.repo.Transaction(ctx, func(tx interfaces.PlatformTemplateRepository) error {
		if err := tx.DeleteVersionsByTemplateID(ctx, id); err != nil {
			return err
		}
		return tx.Delete(ctx, row)
	})
}

// SaveDraft 新增一版草稿（未自动发布）。
func (s *Service) SaveDraft(ctx context.Context, id uint, req SaveDraftRequest, actorID uint) (*VersionItem, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, bizerrors.Pass(ctx, "platformtpl", "SaveDraft", err)
	}
	content := req.Content
	sum := checksum(content)
	maxVer, _ := s.repo.MaxVersion(ctx, id)
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
	if err := s.repo.CreateVersion(ctx, &ver); err != nil {
		return nil, bizerrors.Pass(ctx, "platformtpl", "SaveDraft", err)
	}
	return &VersionItem{PlatformTemplateVersion: ver, ContentPreview: preview(content)}, nil
}

// Publish 将指定版本（或最新草稿）设为发布态。
func (s *Service) Publish(ctx context.Context, id uint, version int) (*TemplateItem, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, bizerrors.Pass(ctx, "platformtpl", "Publish", err)
	}
	var ver *model.PlatformTemplateVersion
	if version > 0 {
		ver, err = s.repo.GetVersion(ctx, id, version)
	} else {
		ver, err = s.repo.GetLatestVersion(ctx, id)
	}
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("无可发布版本，请先保存草稿")
	}
	if strings.TrimSpace(ver.StorageKey) == "" {
		ver.StorageKey = s.tryMirrorMinIO(ctx, row.TemplateKey, ver.Version, ver.ContentInline, row.Format)
		if ver.StorageKey != "" {
			_ = s.repo.UpdateVersionStorageKey(ctx, ver.ID, ver.StorageKey)
		}
	}
	if err := s.repo.UpdatePublishedVersion(ctx, id, ver.Version); err != nil {
		return nil, bizerrors.Pass(ctx, "platformtpl", "Publish", err)
	}
	return s.Detail(ctx, id)
}

func (s *Service) ListVersions(ctx context.Context, id uint) ([]VersionItem, error) {
	rows, err := s.repo.ListVersions(ctx, id)
	if err != nil {
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
	ver, err := s.repo.GetVersion(ctx, id, version)
	if err != nil {
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
	return &VersionItem{PlatformTemplateVersion: *ver}, nil
}

// ResolvePublished 业务侧解析：已发布正文；无则 builtin 种子内容。
func (s *Service) ResolvePublished(ctx context.Context, templateKey string) (*ResolveResult, error) {
	key := strings.TrimSpace(templateKey)
	if key == "" {
		return nil, constants.ErrBadRequestWithMsg("template_key 必填")
	}
	row, err := s.repo.GetByKeyEnabled(ctx, key)
	if err == nil && row.PublishedVersion > 0 {
		ver, verr := s.GetVersionContent(ctx, row.ID, row.PublishedVersion)
		if verr == nil && strings.TrimSpace(ver.ContentInline) != "" {
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
	if s.newObjectStore == nil {
		return ""
	}
	cli, err := s.newObjectStore(ctx)
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
	if s.newObjectStore == nil {
		return nil, errors.New("object store not configured")
	}
	cli, err := s.newObjectStore(ctx)
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
