package cmdb

import (
	"context"
	"errors"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	cryptox "yunshu/internal/pkg/crypto"
	bizerrors "yunshu/internal/pkg/errors"

	"gorm.io/gorm"
)

func toCloudAccountItem(it model.CloudAccount) CloudAccountItem {
	var lastSyncAt *string
	if it.LastSyncAt != nil {
		v := it.LastSyncAt.Format(time.RFC3339)
		lastSyncAt = &v
	}
	return CloudAccountItem{
		ID: it.ID, ProjectID: it.ProjectID, GroupID: it.GroupID, Provider: it.Provider,
		AccountName: it.AccountName, RegionScope: it.RegionScope, Status: it.Status,
		LastSyncAt: lastSyncAt, LastSyncError: it.LastSyncError, CreatedAt: it.CreatedAt.Format(time.RFC3339),
	}
}

// ListCloudAccounts 列出项目/分组下的云账号。
func (s *Service) ListCloudAccounts(ctx context.Context, q CloudAccountListQuery) ([]CloudAccountItem, error) {
	list, err := s.cloudAccountRepo.ListByProjectAndGroup(ctx, q.ProjectID, q.GroupID)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "cmdb", "ListCloudAccounts", err)
	}
	out := make([]CloudAccountItem, 0, len(list))
	for _, it := range list {
		out = append(out, toCloudAccountItem(it))
	}
	return out, nil
}

// UpsertCloudAccount 创建或更新云账号。
func (s *Service) UpsertCloudAccount(ctx context.Context, req CloudAccountUpsertRequest) (*CloudAccountItem, error) {
	var item *model.CloudAccount
	var err error
	if req.ID != nil && *req.ID > 0 {
		item, err = s.cloudAccountRepo.GetByID(ctx, *req.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, constants.ErrNotFoundWithMsg(constants.ErrMsgd19fc495559f)
			}
			return nil, bizerrors.Pass(ctx, "cmdb", "UpsertCloudAccount", err)
		}
	} else {
		item = &model.CloudAccount{}
	}

	item.ProjectID = req.ProjectID
	item.GroupID = req.GroupID
	item.Provider = strings.TrimSpace(req.Provider)
	item.AccountName = strings.TrimSpace(req.AccountName)
	item.RegionScope = strings.TrimSpace(req.RegionScope)
	status := req.Status
	if status != model.StatusDisabled {
		status = model.StatusEnabled
	}
	item.Status = status

	// 同一项目内 AK 不可重复绑定到多个云账号
	if strings.TrimSpace(req.AK) != "" {
		accounts, err := s.cloudAccountRepo.ListByProjectAndGroup(ctx, req.ProjectID, nil)
		if err != nil {
			return nil, bizerrors.Pass(ctx, "cmdb", "UpsertCloudAccount", err)
		}
		for _, ex := range accounts {
			if req.ID != nil && ex.ID == *req.ID {
				continue
			}
			if ex.EncAK == nil {
				continue
			}
			dec, derr := cryptox.DecryptString(s.aead, *ex.EncAK)
			if derr != nil {
				continue
			}
			if strings.TrimSpace(dec) == strings.TrimSpace(req.AK) {
				return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg74ea54455bb4)
			}
		}
	}

	if strings.TrimSpace(req.AK) != "" {
		encAK, err := cryptox.EncryptString(s.aead, req.AK)
		if err != nil {
			return nil, bizerrors.Pass(ctx, "cmdb", "UpsertCloudAccount", err)
		}
		item.EncAK = &encAK
	}
	if strings.TrimSpace(req.SK) != "" {
		encSK, err := cryptox.EncryptString(s.aead, req.SK)
		if err != nil {
			return nil, bizerrors.Pass(ctx, "cmdb", "UpsertCloudAccount", err)
		}
		item.EncSK = &encSK
	}
	if item.ID == 0 {
		err = s.cloudAccountRepo.Create(ctx, item)
	} else {
		err = s.cloudAccountRepo.Save(ctx, item)
	}
	if err != nil {
		return nil, bizerrors.Pass(ctx, "cmdb", "UpsertCloudAccount", err)
	}
	out := toCloudAccountItem(*item)
	return &out, nil
}

func (s *Service) providerFor(name string) (CloudProvider, error) {
	return CloudProviderByName(name)
}

// CloudProviderByName 按名称返回云 Provider 实现。
func CloudProviderByName(name string) (CloudProvider, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "alibaba", "aliyun":
		return &AlibabaCloudProvider{}, nil
	case "tencent", "qcloud":
		return &TencentCloudProvider{}, nil
	case "jd", "jingdong":
		return &JdCloudProvider{}, nil
	default:
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg4e7f045ccd87)
	}
}

// SyncCloudAccount 从云厂商同步实例到 CMDB。
func (s *Service) SyncCloudAccount(ctx context.Context, req CloudSyncRequest) (*CloudSyncResult, error) {
	acc, err := s.cloudAccountRepo.GetByID(ctx, req.AccountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFoundWithMsg(constants.ErrMsgd19fc495559f)
		}
		return nil, bizerrors.Pass(ctx, "cmdb", "SyncCloudAccount", err)
	}
	if acc.ProjectID != req.ProjectID {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg053a6a395b16)
	}
	if acc.EncAK == nil || acc.EncSK == nil {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg2a88bfc17d34)
	}
	ak, err := cryptox.DecryptString(s.aead, *acc.EncAK)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "cmdb", "SyncCloudAccount", err)
	}
	sk, err := cryptox.DecryptString(s.aead, *acc.EncSK)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "cmdb", "SyncCloudAccount", err)
	}
	provider, err := s.providerFor(acc.Provider)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "cmdb", "SyncCloudAccount", err)
	}
	instances, err := provider.ListInstances(ctx, ak, sk, acc.RegionScope)
	now := time.Now()
	acc.LastSyncAt = &now
	if err != nil {
		msg := err.Error()
		acc.LastSyncError = &msg
		_ = s.cloudAccountRepo.Save(ctx, acc)
		return nil, bizerrors.Pass(ctx, "cmdb", "SyncCloudAccount", err)
	}
	acc.LastSyncError = nil
	_ = s.cloudAccountRepo.Save(ctx, acc)

	currentServers, err := s.serverRepo.ListByProjectGroupProvider(ctx, acc.ProjectID, acc.GroupID, acc.Provider)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "cmdb", "SyncCloudAccount.listCurrent", err)
	}
	allCloudServers, err := s.serverRepo.ListByProjectProviderCloud(ctx, acc.ProjectID, acc.Provider)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "cmdb", "SyncCloudAccount.listCloud", err)
	}
	existedByInstanceID := make(map[string]*model.Server, len(allCloudServers))
	for i := range allCloudServers {
		instanceID := strings.TrimSpace(allCloudServers[i].CloudInstanceID)
		if instanceID == "" {
			continue
		}
		existedByInstanceID[instanceID] = &allCloudServers[i]
	}
	seen := make(map[string]struct{}, len(instances))
	added := 0
	updated := 0
	unchanged := 0
	for _, ins := range instances {
		instanceID := strings.TrimSpace(ins.InstanceID)
		if instanceID != "" {
			seen[instanceID] = struct{}{}
		}
		existed := existedByInstanceID[instanceID]
		var reqID *uint
		changed := true
		if existed != nil {
			reqID = &existed.ID
			changed = strings.TrimSpace(existed.Name) != strings.TrimSpace(ins.Name) ||
				strings.TrimSpace(existed.Host) != strings.TrimSpace(ins.Host) ||
				strings.TrimSpace(existed.CloudRegion) != strings.TrimSpace(ins.Region) ||
				strings.TrimSpace(existed.OSType) != strings.TrimSpace(ins.OSType) ||
				existed.Status != ins.Status ||
				existed.GroupID == nil || *existed.GroupID != acc.GroupID
		}
		groupID := acc.GroupID
		_, upErr := s.UpsertServer(ctx, ServerUpsertRequest{
			ID:              reqID,
			ProjectID:       acc.ProjectID,
			GroupID:         &groupID,
			Name:            ins.Name,
			Host:            ins.Host,
			Port:            22,
			OSType:          ins.OSType,
			Tags:            "cloud-sync",
			Status:          ins.Status,
			SourceType:      model.ServerGroupCategoryCloud,
			Provider:        acc.Provider,
			CloudInstanceID: instanceID,
			CloudRegion:     ins.Region,
		})
		if upErr == nil {
			if existed == nil {
				added++
			} else if changed {
				updated++
			} else {
				unchanged++
			}
		}
	}

	disabled := 0
	for _, sv := range currentServers {
		if strings.TrimSpace(sv.CloudInstanceID) == "" {
			continue
		}
		if _, ok := seen[sv.CloudInstanceID]; ok {
			continue
		}
		if sv.Status != model.StatusDisabled {
			sv.Status = model.StatusDisabled
			if err := s.serverRepo.Save(ctx, &sv); err == nil {
				disabled++
			}
		}
	}
	return &CloudSyncResult{
		Total: len(instances), Added: added, Updated: updated, Disabled: disabled, Unchanged: unchanged, Message: "sync completed",
	}, nil
}

// DeleteCloudAccount 删除云账号。
func (s *Service) DeleteCloudAccount(ctx context.Context, projectID, accountID uint) error {
	acc, err := s.cloudAccountRepo.GetByID(ctx, accountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrNotFoundWithMsg(constants.ErrMsgd19fc495559f)
		}
		return bizerrors.Pass(ctx, "cmdb", "DeleteCloudAccount", err)
	}
	if acc.ProjectID != projectID {
		return constants.ErrBadRequestWithMsg(constants.ErrMsg053a6a395b16)
	}
	return s.cloudAccountRepo.DeleteByID(ctx, accountID)
}

