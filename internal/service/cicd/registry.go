package cicd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"yunshu/internal/config"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/pkg/platformhttp"

	"gorm.io/gorm"
)

// ResolvedRegistry 统一解析后的仓库连接信息。
type ResolvedRegistry struct {
	RegistryID   uint
	Type         string
	URL          string
	HostIP       string
	Username     string
	Password     string
	ProjectGroup string
}

func (r ResolvedRegistry) ToHarborConfig() config.HarborConfig {
	return config.HarborConfig{
		URL:          r.URL,
		HostIP:       r.HostIP,
		ProjectGroup: r.ProjectGroup,
		Username:     r.Username,
		Password:     r.Password,
	}
}

// ResolveRegistryForProject 解析优先级：绑定 → projects.harbor_* → 默认注册中心 → 字典/YAML。
func (s *Service) ResolveRegistryForProject(ctx context.Context, projectID uint) ResolvedRegistry {
	cfg := s.resolvedConfig(ctx)
	out := ResolvedRegistry{
		Type:         model.ImageRegistryTypeHarbor,
		URL:          cfg.Harbor.URL,
		HostIP:       cfg.Harbor.HostIP,
		Username:     cfg.Harbor.Username,
		Password:     cfg.Harbor.Password,
		ProjectGroup: cfg.Harbor.ProjectGroup,
	}
	if projectID > 0 && s.db != nil {
		var bind model.ProjectRegistryBinding
		if err := s.db.WithContext(ctx).Where("project_id = ?", projectID).First(&bind).Error; err == nil && bind.RegistryID > 0 {
			if reg, err := s.getRegistry(ctx, bind.RegistryID); err == nil {
				out = registryToResolved(reg)
				if v := strings.TrimSpace(bind.HarborProject); v != "" {
					out.ProjectGroup = v
				}
				return out
			}
		}
		ov := s.loadProjectCicdOverrides(ctx, projectID)
		if ov.HarborURL != "" || ov.HarborProject != "" {
			if ov.HarborURL != "" {
				out.URL = stripHarborHost(ov.HarborURL)
			}
			if ov.HarborProject != "" {
				out.ProjectGroup = ov.HarborProject
			}
			return out
		}
	}
	if s.db != nil {
		var def model.ImageRegistry
		if err := s.db.WithContext(ctx).
			Where("is_default = ? AND status = 1", true).
			Order("id ASC").
			First(&def).Error; err == nil {
			out = registryToResolved(&def)
		}
	}
	if out.ProjectGroup == "" {
		out.ProjectGroup = "registry"
	}
	return out
}

func registryToResolved(reg *model.ImageRegistry) ResolvedRegistry {
	if reg == nil {
		return ResolvedRegistry{}
	}
	proj := strings.TrimSpace(reg.DefaultProject)
	if proj == "" {
		proj = "registry"
	}
	return ResolvedRegistry{
		RegistryID:   reg.ID,
		Type:         strings.TrimSpace(reg.Type),
		URL:          stripHarborHost(reg.URL),
		HostIP:       strings.TrimSpace(reg.HostIP),
		Username:     strings.TrimSpace(reg.Username),
		Password:     reg.Password,
		ProjectGroup: proj,
	}
}

func (s *Service) getRegistry(ctx context.Context, id uint) (*model.ImageRegistry, error) {
	var reg model.ImageRegistry
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&reg).Error; err != nil {
		return nil, constants.ErrNotFound
	}
	return &reg, nil
}

// --- CRUD ---

type RegistryUpsertRequest struct {
	Name           string `json:"name" binding:"required,max=128"`
	Type           string `json:"type" binding:"omitempty,max=32"`
	URL            string `json:"url" binding:"required,max=256"`
	HostIP         string `json:"host_ip" binding:"omitempty,max=64"`
	Username       string `json:"username" binding:"omitempty,max=128"`
	Password       string `json:"password" binding:"omitempty,max=256"`
	DefaultProject string `json:"default_project" binding:"omitempty,max=128"`
	IsDefault      bool   `json:"is_default"`
	Status         *int   `json:"status"`
	Remark         string `json:"remark" binding:"omitempty,max=512"`
}

type RegistryItem struct {
	model.ImageRegistry
	HasPassword bool `json:"has_password"`
}

type RegistryListQuery struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

func (s *Service) ListRegistries(ctx context.Context, q RegistryListQuery) (*pagination.Result[RegistryItem], error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	db := s.db.WithContext(ctx).Model(&model.ImageRegistry{})
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []model.ImageRegistry
	if err := db.Order("is_default DESC, id ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]RegistryItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, toRegistryItem(r))
	}
	return &pagination.Result[RegistryItem]{List: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *Service) GetRegistry(ctx context.Context, id uint) (*RegistryItem, error) {
	reg, err := s.getRegistry(ctx, id)
	if err != nil {
		return nil, err
	}
	item := toRegistryItem(*reg)
	return &item, nil
}

func toRegistryItem(r model.ImageRegistry) RegistryItem {
	has := strings.TrimSpace(r.Password) != ""
	r.Password = ""
	return RegistryItem{ImageRegistry: r, HasPassword: has}
}

func (s *Service) UpsertRegistry(ctx context.Context, id uint, req RegistryUpsertRequest) (*RegistryItem, error) {
	typ := strings.TrimSpace(req.Type)
	if typ == "" {
		typ = model.ImageRegistryTypeHarbor
	}
	if typ != model.ImageRegistryTypeHarbor && typ != model.ImageRegistryTypeDockerRegistry {
		return nil, constants.ErrBadRequestWithMsg("type 须为 harbor 或 docker_registry")
	}
	status := 1
	if req.Status != nil {
		status = *req.Status
	}
	var reg model.ImageRegistry
	if id > 0 {
		if err := s.db.WithContext(ctx).Where("id = ?", id).First(&reg).Error; err != nil {
			return nil, constants.ErrNotFound
		}
	}
	reg.Name = strings.TrimSpace(req.Name)
	reg.Type = typ
	reg.URL = stripHarborHost(req.URL)
	reg.HostIP = strings.TrimSpace(req.HostIP)
	reg.Username = strings.TrimSpace(req.Username)
	if strings.TrimSpace(req.Password) != "" {
		reg.Password = strings.TrimSpace(req.Password)
	}
	reg.DefaultProject = strings.TrimSpace(req.DefaultProject)
	reg.IsDefault = req.IsDefault
	reg.Status = status
	reg.Remark = strings.TrimSpace(req.Remark)

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if reg.IsDefault {
			if err := tx.Model(&model.ImageRegistry{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		if id == 0 {
			return tx.Create(&reg).Error
		}
		return tx.Save(&reg).Error
	})
	if err != nil {
		return nil, err
	}
	item := toRegistryItem(reg)
	return &item, nil
}

func (s *Service) DeleteRegistry(ctx context.Context, id uint) error {
	res := s.db.WithContext(ctx).Delete(&model.ImageRegistry{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return constants.ErrNotFound
	}
	_ = s.db.WithContext(ctx).Where("registry_id = ?", id).Delete(&model.ProjectRegistryBinding{}).Error
	_ = s.db.WithContext(ctx).Where("registry_id = ?", id).Delete(&model.ImageCleanupPolicy{}).Error
	return nil
}

func (s *Service) PingRegistry(ctx context.Context, id uint) (map[string]any, error) {
	reg, err := s.getRegistry(ctx, id)
	if err != nil {
		return nil, err
	}
	resolved := registryToResolved(reg)
	client := registryHTTPClient(resolved.URL)
	switch resolved.Type {
	case model.ImageRegistryTypeDockerRegistry:
		code, body, err := s.registryDo(ctx, client, resolved, http.MethodGet, "/v2/")
		if err != nil {
			return nil, constants.ErrBadRequestWithMsg("连通失败: " + err.Error())
		}
		if code >= 200 && code < 300 {
			return map[string]any{"ok": true, "type": resolved.Type, "http_status": code}, nil
		}
		return nil, constants.ErrBadRequestWithMsg(fmt.Sprintf("连通失败 HTTP %d: %s", code, truncate(body, 200)))
	default:
		code, body, err := s.registryDo(ctx, client, resolved, http.MethodGet, "/api/v2.0/systeminfo")
		if err != nil {
			return nil, constants.ErrBadRequestWithMsg("连通失败: " + err.Error())
		}
		if code >= 200 && code < 300 {
			return map[string]any{"ok": true, "type": "harbor", "http_status": code, "body_preview": truncate(body, 120)}, nil
		}
		// 部分 Harbor 需登录才可看 systeminfo，再试 projects
		code2, _, err2 := s.registryDo(ctx, client, resolved, http.MethodGet, "/api/v2.0/projects?page_size=1")
		if err2 == nil && code2 >= 200 && code2 < 300 {
			return map[string]any{"ok": true, "type": "harbor", "http_status": code2}, nil
		}
		return nil, constants.ErrBadRequestWithMsg(fmt.Sprintf("连通失败 HTTP %d: %s", code, truncate(body, 200)))
	}
}

type ProjectRegistryBindingRequest struct {
	RegistryID    uint   `json:"registry_id" binding:"required"`
	HarborProject string `json:"harbor_project" binding:"omitempty,max=128"`
}

func (s *Service) GetProjectRegistryBinding(ctx context.Context, projectID uint) (*model.ProjectRegistryBinding, error) {
	var bind model.ProjectRegistryBinding
	if err := s.db.WithContext(ctx).Where("project_id = ?", projectID).First(&bind).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &model.ProjectRegistryBinding{ProjectID: projectID}, nil
		}
		return nil, err
	}
	return &bind, nil
}

func (s *Service) UpsertProjectRegistryBinding(ctx context.Context, projectID uint, req ProjectRegistryBindingRequest) (*model.ProjectRegistryBinding, error) {
	if _, err := s.getRegistry(ctx, req.RegistryID); err != nil {
		return nil, err
	}
	var bind model.ProjectRegistryBinding
	err := s.db.WithContext(ctx).Where("project_id = ?", projectID).First(&bind).Error
	if err != nil {
		bind = model.ProjectRegistryBinding{
			ProjectID:     projectID,
			RegistryID:    req.RegistryID,
			HarborProject: strings.TrimSpace(req.HarborProject),
		}
		if err := s.db.WithContext(ctx).Create(&bind).Error; err != nil {
			return nil, err
		}
		return &bind, nil
	}
	bind.RegistryID = req.RegistryID
	bind.HarborProject = strings.TrimSpace(req.HarborProject)
	if err := s.db.WithContext(ctx).Save(&bind).Error; err != nil {
		return nil, err
	}
	return &bind, nil
}

func (s *Service) DeleteProjectRegistryBinding(ctx context.Context, projectID uint) error {
	res := s.db.WithContext(ctx).Where("project_id = ?", projectID).Delete(&model.ProjectRegistryBinding{})
	if res.Error != nil {
		return res.Error
	}
	return nil
}

// --- HTTP helpers ---

func registryHTTPClient(serverName string) *http.Client {
	return platformhttp.NewInsecureTLSClient(30*time.Second, serverName)
}

func (s *Service) registryDo(ctx context.Context, client *http.Client, r ResolvedRegistry, method, path string) (int, string, error) {
	host := stripHarborHost(r.URL)
	connect := host
	if ip := strings.TrimSpace(r.HostIP); ip != "" {
		connect = ip
	}
	scheme := "https"
	fullURL := fmt.Sprintf("%s://%s%s", scheme, connect, path)
	req, err := http.NewRequestWithContext(ctx, method, fullURL, nil)
	if err != nil {
		return 0, "", err
	}
	req.Host = host
	if u := strings.TrimSpace(r.Username); u != "" {
		token := base64.StdEncoding.EncodeToString([]byte(u + ":" + r.Password))
		req.Header.Set("Authorization", "Basic "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	return resp.StatusCode, string(b), nil
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func encodeHarborRepo(name string) string {
	// Harbor API 要求 repository 名中的 / 编码为 %2F
	return url.PathEscape(strings.TrimPrefix(name, "/"))
}

func decodeJSONList[T any](raw string) ([]T, error) {
	var out []T
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}
