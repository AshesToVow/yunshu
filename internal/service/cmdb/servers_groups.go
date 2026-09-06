package cmdb

import (
	"context"
	"errors"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"

	"gorm.io/gorm"
)

// ListServerGroupTree 返回项目下的服务器分组树。
func (s *Service) ListServerGroupTree(ctx context.Context, q ServerGroupTreeQuery) ([]ServerGroupItem, error) {
	if err := s.ensureDefaultServerGroups(ctx, q.ProjectID); err != nil {
		return nil, bizerrors.Pass(ctx, "cmdb", "ListServerGroupTree", err)
	}
	list, err := s.serverGroupRepo.ListByProject(ctx, q.ProjectID)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "cmdb", "ListServerGroupTree", err)
	}
	items := make([]ServerGroupItem, 0, len(list))
	for _, it := range list {
		items = append(items, ServerGroupItem{
			ID: it.ID, ProjectID: it.ProjectID, ParentID: it.ParentID, Name: it.Name,
			Category: it.Category, Provider: it.Provider, Sort: it.Sort, Status: it.Status,
		})
	}
	byParent := map[uint][]ServerGroupItem{}
	roots := make([]ServerGroupItem, 0)
	for _, it := range items {
		if it.ParentID == nil {
			roots = append(roots, it)
			continue
		}
		byParent[*it.ParentID] = append(byParent[*it.ParentID], it)
	}
	var attach func(*ServerGroupItem)
	attach = func(node *ServerGroupItem) {
		children := byParent[node.ID]
		for i := range children {
			child := children[i]
			attach(&child)
			node.Children = append(node.Children, child)
		}
	}
	for i := range roots {
		attach(&roots[i])
	}
	return roots, nil
}

// UpsertServerGroup 创建或更新服务器分组。
// category 除内置 self_hosted / cloud 外，允许自定义标识（与数据字典 server_group_category 对齐）；
// 自定义类型按自建语义处理（可挂服务器，不走云账号同步）。
func (s *Service) UpsertServerGroup(ctx context.Context, req ServerGroupUpsertRequest) (*ServerGroupItem, error) {
	if err := s.ensureDefaultServerGroups(ctx, req.ProjectID); err != nil {
		return nil, bizerrors.Pass(ctx, "cmdb", "UpsertServerGroup", err)
	}
	category := strings.TrimSpace(req.Category)
	if category == "" {
		category = model.ServerGroupCategorySelfHosted
	}
	if err := validateServerGroupCategory(category); err != nil {
		return nil, err
	}
	provider := strings.TrimSpace(req.Provider)
	if category != model.ServerGroupCategoryCloud && provider == "" {
		provider = "custom"
	}
	status := req.Status
	if status != model.StatusDisabled {
		status = model.StatusEnabled
	}

	var item *model.ServerGroup
	var err error
	if req.ID != nil && *req.ID > 0 {
		item, err = s.serverGroupRepo.GetByID(ctx, *req.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, constants.ErrNotFoundWithMsg(constants.ErrMsg97c4c24a4cdf)
			}
			return nil, bizerrors.Pass(ctx, "cmdb", "UpsertServerGroup", err)
		}
	} else {
		item = &model.ServerGroup{}
	}
	item.ProjectID = req.ProjectID
	item.ParentID = req.ParentID
	item.Name = strings.TrimSpace(req.Name)
	item.Category = category
	item.Provider = provider
	item.Sort = req.Sort
	item.Status = status
	if item.ID == 0 {
		err = s.serverGroupRepo.Create(ctx, item)
	} else {
		err = s.serverGroupRepo.Save(ctx, item)
	}
	if err != nil {
		return nil, bizerrors.Pass(ctx, "cmdb", "UpsertServerGroup", err)
	}
	return &ServerGroupItem{
		ID: item.ID, ProjectID: item.ProjectID, ParentID: item.ParentID, Name: item.Name,
		Category: item.Category, Provider: item.Provider, Sort: item.Sort, Status: item.Status,
	}, nil
}

func validateServerGroupCategory(category string) error {
	// 标识：小写字母开头，仅 a-z 0-9 _，最长 32
	if len(category) == 0 || len(category) > 32 {
		return constants.ErrBadRequestWithMsg("分组类型标识长度须为 1～32")
	}
	for i, r := range category {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_'
		if i == 0 && (r < 'a' || r > 'z') {
			return constants.ErrBadRequestWithMsg("分组类型标识须以小写字母开头（如 idc、colo）")
		}
		if !ok {
			return constants.ErrBadRequestWithMsg("分组类型标识仅允许小写字母、数字和下划线")
		}
	}
	return nil
}

// DeleteServerGroup 删除服务器分组。
func (s *Service) DeleteServerGroup(ctx context.Context, projectID, groupID uint) error {
	group, err := s.serverGroupRepo.GetByID(ctx, groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrNotFoundWithMsg(constants.ErrMsg97c4c24a4cdf)
		}
		return bizerrors.Pass(ctx, "cmdb", "DeleteServerGroup", err)
	}
	if group.ProjectID != projectID {
		return constants.ErrBadRequestWithMsg(constants.ErrMsg757ed9cbc3d5)
	}
	return s.serverGroupRepo.DeleteByID(ctx, groupID)
}

const (
	defaultSelfHostedGroupName = "自建服务器"
	defaultCloudRootGroupName  = "云服务器"
	defaultAlibabaGroupName    = "阿里云"
	defaultTencentGroupName    = "腾讯云"
	defaultJDGroupName         = "京东云"
)

func defaultServerGroupName(g model.ServerGroup) string {
	switch {
	case g.Category == model.ServerGroupCategorySelfHosted && g.ParentID == nil:
		return defaultSelfHostedGroupName
	case g.Category == model.ServerGroupCategoryCloud && g.ParentID == nil:
		return defaultCloudRootGroupName
	case g.Category == model.ServerGroupCategoryCloud && g.Provider == "alibaba":
		return defaultAlibabaGroupName
	case g.Category == model.ServerGroupCategoryCloud && g.Provider == "tencent":
		return defaultTencentGroupName
	case g.Category == model.ServerGroupCategoryCloud && g.Provider == "jd":
		return defaultJDGroupName
	default:
		return ""
	}
}

func isCorruptedGroupName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return true
	}
	for _, r := range name {
		if r != '?' && r != '？' {
			return false
		}
	}
	return true
}

func (s *Service) repairCorruptedDefaultServerGroups(ctx context.Context, list []model.ServerGroup) error {
	for i := range list {
		g := list[i]
		if !isCorruptedGroupName(g.Name) {
			continue
		}
		want := defaultServerGroupName(g)
		if want == "" || want == g.Name {
			continue
		}
		g.Name = want
		if err := s.serverGroupRepo.Save(ctx, &g); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) ensureDefaultServerGroups(ctx context.Context, projectID uint) error {
	list, err := s.serverGroupRepo.ListByProject(ctx, projectID)
	if err != nil {
		return bizerrors.Pass(ctx, "cmdb", "ensureDefaultServerGroups", err)
	}
	if err := s.repairCorruptedDefaultServerGroups(ctx, list); err != nil {
		return bizerrors.Pass(ctx, "cmdb", "ensureDefaultServerGroups", err)
	}

	// Avoid repeated backfill scans on hot list endpoints (repair above always runs).
	s.ensureMu.Lock()
	if ts, ok := s.ensuredProjectAt[projectID]; ok && time.Since(ts) < 30*time.Second {
		s.ensureMu.Unlock()
		return nil
	}
	s.ensureMu.Unlock()

	if len(list) == 0 {
		selfHosted := model.ServerGroup{
			ProjectID: projectID,
			Name:      defaultSelfHostedGroupName,
			Category:  model.ServerGroupCategorySelfHosted,
			Provider:  "custom",
			Sort:      1,
			Status:    model.StatusEnabled,
		}
		cloudRoot := model.ServerGroup{
			ProjectID: projectID,
			Name:      defaultCloudRootGroupName,
			Category:  model.ServerGroupCategoryCloud,
			Provider:  "custom",
			Sort:      2,
			Status:    model.StatusEnabled,
		}
		if err := s.serverGroupRepo.Create(ctx, &selfHosted); err != nil {
			return bizerrors.Pass(ctx, "cmdb", "ensureDefaultServerGroups", err)
		}
		if err := s.serverGroupRepo.Create(ctx, &cloudRoot); err != nil {
			return bizerrors.Pass(ctx, "cmdb", "ensureDefaultServerGroups", err)
		}
		alibaba := model.ServerGroup{
			ProjectID: projectID,
			ParentID:  &cloudRoot.ID,
			Name:      defaultAlibabaGroupName,
			Category:  model.ServerGroupCategoryCloud,
			Provider:  "alibaba",
			Sort:      10,
			Status:    model.StatusEnabled,
		}
		tencent := model.ServerGroup{
			ProjectID: projectID,
			ParentID:  &cloudRoot.ID,
			Name:      defaultTencentGroupName,
			Category:  model.ServerGroupCategoryCloud,
			Provider:  "tencent",
			Sort:      11,
			Status:    model.StatusEnabled,
		}
		jd := model.ServerGroup{
			ProjectID: projectID,
			ParentID:  &cloudRoot.ID,
			Name:      defaultJDGroupName,
			Category:  model.ServerGroupCategoryCloud,
			Provider:  "jd",
			Sort:      12,
			Status:    model.StatusEnabled,
		}
		_ = s.serverGroupRepo.Create(ctx, &alibaba)
		_ = s.serverGroupRepo.Create(ctx, &tencent)
		_ = s.serverGroupRepo.Create(ctx, &jd)

	}

	// Backfill ungrouped servers every time (historical data compatibility).
	// This cannot rely on "len(list) == 0" only, because old rows may be inserted
	// later without group_id while default groups already exist.
	list, err = s.serverGroupRepo.ListByProject(ctx, projectID)
	if err != nil {
		return bizerrors.Pass(ctx, "cmdb", "ensureDefaultServerGroups", err)
	}
	var selfHostedID uint
	for _, g := range list {
		if g.ParentID != nil {
			continue
		}
		if strings.TrimSpace(g.Category) != model.ServerGroupCategorySelfHosted {
			continue
		}
		selfHostedID = g.ID
		break
	}
	if selfHostedID == 0 {
		// Fallback: create one self-hosted root if missing unexpectedly.
		selfHosted := model.ServerGroup{
			ProjectID: projectID,
			Name:      defaultSelfHostedGroupName,
			Category:  model.ServerGroupCategorySelfHosted,
			Provider:  "custom",
			Sort:      1,
			Status:    model.StatusEnabled,
		}
		if err := s.serverGroupRepo.Create(ctx, &selfHosted); err != nil {
			return bizerrors.Pass(ctx, "cmdb", "ensureDefaultServerGroups", err)
		}
		selfHostedID = selfHosted.ID
	}
	if servers, err := s.serverRepo.ListByProjectWithoutGroup(ctx, projectID); err == nil {
		for i := range servers {
			sv := servers[i]
			sv.GroupID = &selfHostedID
			if strings.TrimSpace(sv.SourceType) == "" {
				sv.SourceType = model.ServerGroupCategorySelfHosted
			}
			_ = s.serverRepo.Save(ctx, &sv)
		}
	}

	s.ensureMu.Lock()
	s.ensuredProjectAt[projectID] = time.Now()
	s.ensureMu.Unlock()
	return nil
}

