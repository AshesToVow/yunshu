package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type CicdRepository struct {
	db *gorm.DB
}

func NewCicdRepository(db *gorm.DB) CicdRepo {
	if db == nil {
		return &CicdRepository{}
	}
	return &CicdRepository{db: db}
}

func (r *CicdRepository) Transaction(ctx context.Context, fn func(CicdRepo) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&CicdRepository{db: tx})
	})
}

func (r *CicdRepository) dbq(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

// --- Service ---

func (r *CicdRepository) ListServices(ctx context.Context, p CicdServiceListParams) ([]model.CicdService, int64, error) {
	q := r.dbq(ctx).Model(&model.CicdService{}).Where("project_id = ?", p.ProjectID)
	if p.RestrictIDs {
		if len(p.IDs) == 0 {
			return nil, 0, nil
		}
		q = q.Where("id IN ?", p.IDs)
	} else if len(p.IDs) > 0 {
		q = q.Where("id IN ?", p.IDs)
	}
	if kw := strings.TrimSpace(p.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("name LIKE ? OR identifier LIKE ?", like, like)
	}
	if st := strings.TrimSpace(p.ServiceType); st != "" {
		q = q.Where("service_type = ?", st)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.CicdService
	err := q.Order("id DESC").Offset(p.Offset).Limit(p.Limit).Find(&rows).Error
	return rows, total, err
}

func (r *CicdRepository) GetService(ctx context.Context, projectID, serviceID uint) (*model.CicdService, error) {
	var row model.CicdService
	err := r.dbq(ctx).Where("id = ? AND project_id = ?", serviceID, projectID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *CicdRepository) GetServiceByID(ctx context.Context, serviceID uint) (*model.CicdService, error) {
	var row model.CicdService
	if err := r.dbq(ctx).Where("id = ?", serviceID).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *CicdRepository) GetServiceByProjectIdentifier(ctx context.Context, projectID uint, identifier string) (*model.CicdService, error) {
	var row model.CicdService
	err := r.dbq(ctx).Where("project_id = ? AND identifier = ?", projectID, identifier).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *CicdRepository) CreateService(ctx context.Context, row *model.CicdService) error {
	return r.dbq(ctx).Create(row).Error
}

func (r *CicdRepository) SaveService(ctx context.Context, row *model.CicdService) error {
	return r.dbq(ctx).Save(row).Error
}

func (r *CicdRepository) DeleteServiceCascade(ctx context.Context, projectID, serviceID uint) error {
	return r.Transaction(ctx, func(tx CicdRepo) error {
		tr := tx.(*CicdRepository)
		if err := tr.dbq(ctx).Where("service_id = ?", serviceID).Delete(&model.CicdCiConfig{}).Error; err != nil {
			return err
		}
		if err := tr.dbq(ctx).Where("service_id = ?", serviceID).Delete(&model.CicdDeployConfig{}).Error; err != nil {
			return err
		}
		if err := tr.dbq(ctx).Where("service_id = ?", serviceID).Delete(&model.CicdBuildRun{}).Error; err != nil {
			return err
		}
		if err := tr.dbq(ctx).Where("service_id = ?", serviceID).Delete(&model.CicdReleaseRun{}).Error; err != nil {
			return err
		}
		return tr.dbq(ctx).Where("id = ? AND project_id = ?", serviceID, projectID).Delete(&model.CicdService{}).Error
	})
}

func (r *CicdRepository) UpdateServiceJenkinsJob(ctx context.Context, serviceID uint, jobName string) error {
	return r.dbq(ctx).Model(&model.CicdService{}).Where("id = ?", serviceID).Update("jenkins_job", jobName).Error
}

func (r *CicdRepository) CountDuplicateServiceIdentifier(ctx context.Context, projectID, excludeID uint, identifier string) (int64, error) {
	var n int64
	q := r.dbq(ctx).Model(&model.CicdService{}).Where("project_id = ? AND identifier = ?", projectID, identifier)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	err := q.Count(&n).Error
	return n, err
}

func (r *CicdRepository) ListServicesByProject(ctx context.Context, projectID uint) ([]model.CicdService, error) {
	var rows []model.CicdService
	err := r.dbq(ctx).Where("project_id = ?", projectID).Find(&rows).Error
	return rows, err
}

func (r *CicdRepository) ListServicesByIDs(ctx context.Context, ids []uint) ([]model.CicdService, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []model.CicdService
	err := r.dbq(ctx).Where("id IN ?", ids).Find(&rows).Error
	return rows, err
}

func (r *CicdRepository) PluckServiceIDsWithCI(ctx context.Context, serviceIDs []uint) ([]uint, error) {
	if len(serviceIDs) == 0 {
		return nil, nil
	}
	var ids []uint
	err := r.dbq(ctx).Model(&model.CicdCiConfig{}).
		Where("service_id IN ?", serviceIDs).Distinct("service_id").Pluck("service_id", &ids).Error
	return ids, err
}

func (r *CicdRepository) CountDeployConfigsByServiceIDs(ctx context.Context, serviceIDs []uint) ([]CicdServiceDeployCount, error) {
	if len(serviceIDs) == 0 {
		return nil, nil
	}
	var rows []CicdServiceDeployCount
	err := r.dbq(ctx).Model(&model.CicdDeployConfig{}).
		Select("service_id, COUNT(*) AS cnt").
		Where("service_id IN ? AND status = 1", serviceIDs).
		Group("service_id").Scan(&rows).Error
	return rows, err
}

func (r *CicdRepository) ListLatestBuildsByServiceIDs(ctx context.Context, serviceIDs []uint) ([]model.CicdBuildRun, error) {
	if len(serviceIDs) == 0 {
		return nil, nil
	}
	var builds []model.CicdBuildRun
	err := r.dbq(ctx).Where("service_id IN ?", serviceIDs).Order("id DESC").Find(&builds).Error
	return builds, err
}

func (r *CicdRepository) CountCIByService(ctx context.Context, serviceID uint) (int64, error) {
	var n int64
	err := r.dbq(ctx).Model(&model.CicdCiConfig{}).Where("service_id = ?", serviceID).Count(&n).Error
	return n, err
}

// --- CI / Deploy config ---

func (r *CicdRepository) GetCIConfig(ctx context.Context, serviceID uint) (*model.CicdCiConfig, error) {
	var row model.CicdCiConfig
	err := r.dbq(ctx).Where("service_id = ?", serviceID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *CicdRepository) CreateCIConfig(ctx context.Context, row *model.CicdCiConfig) error {
	return r.dbq(ctx).Create(row).Error
}

func (r *CicdRepository) SaveCIConfig(ctx context.Context, row *model.CicdCiConfig) error {
	return r.dbq(ctx).Save(row).Error
}

func (r *CicdRepository) ListDeployConfigs(ctx context.Context, serviceID uint) ([]model.CicdDeployConfig, error) {
	var rows []model.CicdDeployConfig
	err := r.dbq(ctx).Where("service_id = ?", serviceID).Order("id ASC").Find(&rows).Error
	return rows, err
}

func (r *CicdRepository) GetDeployConfig(ctx context.Context, serviceID, configID uint) (*model.CicdDeployConfig, error) {
	var row model.CicdDeployConfig
	err := r.dbq(ctx).Where("id = ? AND service_id = ?", configID, serviceID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *CicdRepository) GetDeployConfigByID(ctx context.Context, configID uint) (*model.CicdDeployConfig, error) {
	var row model.CicdDeployConfig
	if err := r.dbq(ctx).Where("id = ?", configID).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *CicdRepository) CreateDeployConfig(ctx context.Context, row *model.CicdDeployConfig) error {
	return r.dbq(ctx).Create(row).Error
}

func (r *CicdRepository) SaveDeployConfig(ctx context.Context, row *model.CicdDeployConfig) error {
	return r.dbq(ctx).Save(row).Error
}

func (r *CicdRepository) DeleteDeployConfig(ctx context.Context, serviceID, configID uint) error {
	return r.dbq(ctx).Where("id = ? AND service_id = ?", configID, serviceID).Delete(&model.CicdDeployConfig{}).Error
}

func (r *CicdRepository) CountDeployConfigDupName(ctx context.Context, serviceID, excludeID uint, name string) (int64, error) {
	var n int64
	q := r.dbq(ctx).Model(&model.CicdDeployConfig{}).Where("service_id = ? AND name = ?", serviceID, name)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	err := q.Count(&n).Error
	return n, err
}

func (r *CicdRepository) CountDeployConfigDupKindTenv(ctx context.Context, serviceID, excludeID uint, deployKind, tenv string) (int64, error) {
	var n int64
	q := r.dbq(ctx).Model(&model.CicdDeployConfig{}).
		Where("service_id = ? AND deploy_kind = ? AND tenv = ?", serviceID, strings.TrimSpace(deployKind), strings.TrimSpace(tenv))
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	err := q.Count(&n).Error
	return n, err
}

func (r *CicdRepository) CountContainerDeploys(ctx context.Context, serviceID uint) (int64, error) {
	var n int64
	err := r.dbq(ctx).Model(&model.CicdDeployConfig{}).
		Where("service_id = ? AND deploy_kind = ?", serviceID, model.CicdDeployKindContainer).
		Count(&n).Error
	return n, err
}

func (r *CicdRepository) GetFirstContainerDeploy(ctx context.Context, serviceID uint) (*model.CicdDeployConfig, error) {
	var row model.CicdDeployConfig
	err := r.dbq(ctx).
		Where("service_id = ? AND deploy_kind = ?", serviceID, model.CicdDeployKindContainer).
		Order("id ASC").First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *CicdRepository) GetFirstDeployConfig(ctx context.Context, serviceID uint) (*model.CicdDeployConfig, error) {
	var row model.CicdDeployConfig
	err := r.dbq(ctx).Where("service_id = ?", serviceID).Order("id ASC").First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *CicdRepository) GetServiceByJenkinsJob(ctx context.Context, jobName string) (*model.CicdService, error) {
	job := strings.TrimSpace(jobName)
	var svc model.CicdService
	err := r.dbq(ctx).
		Where("jenkins_job = ? OR identifier = ?", job, job).
		Order("id DESC").
		First(&svc).Error
	if err != nil {
		return nil, err
	}
	return &svc, nil
}

// --- Enrichment helpers ---

func (r *CicdRepository) ListUserGroupsByIDs(ctx context.Context, ids []uint) ([]CicdNameID, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var groups []model.UserGroup
	if err := r.dbq(ctx).Select("id, name").Where("id IN ?", ids).Find(&groups).Error; err != nil {
		return nil, err
	}
	out := make([]CicdNameID, len(groups))
	for i, g := range groups {
		out[i] = CicdNameID{ID: g.ID, Name: g.Name}
	}
	return out, nil
}

func (r *CicdRepository) ListUsersByIDs(ctx context.Context, ids []uint) ([]CicdUserBrief, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var users []model.User
	if err := r.dbq(ctx).Select("id, username, nickname").Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	out := make([]CicdUserBrief, len(users))
	for i, u := range users {
		out[i] = CicdUserBrief{ID: u.ID, Username: u.Username, Nickname: u.Nickname}
	}
	return out, nil
}

func (r *CicdRepository) GetProjectName(ctx context.Context, projectID uint) (string, error) {
	var row model.Project
	err := r.dbq(ctx).Select("name").Where("id = ?", projectID).First(&row).Error
	if err != nil {
		return "", err
	}
	return row.Name, nil
}

func (r *CicdRepository) GetServiceBrief(ctx context.Context, serviceID uint) (*CicdServiceBrief, error) {
	var row model.CicdService
	err := r.dbq(ctx).Select("id, name, identifier").Where("id = ?", serviceID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &CicdServiceBrief{ID: row.ID, Name: row.Name, Identifier: row.Identifier}, nil
}

func (r *CicdRepository) ListServersByProject(ctx context.Context, projectID uint) ([]model.Server, error) {
	var rows []model.Server
	err := r.dbq(ctx).Where("project_id = ?", projectID).Find(&rows).Error
	return rows, err
}

func (r *CicdRepository) LookupLinkedK8sWorkload(ctx context.Context, cicdServiceID uint) (clusterID uint, namespace, kind, name string, err error) {
	var catalogLink model.ServiceLink
	if err = r.dbq(ctx).
		Where("link_type = ? AND ref_id = ?", model.ServiceLinkCicdService, cicdServiceID).
		First(&catalogLink).Error; err != nil {
		return 0, "", "", "", err
	}
	var wl model.ServiceLink
	if err = r.dbq(ctx).
		Where("service_id = ? AND link_type = ?", catalogLink.ServiceID, model.ServiceLinkK8sWorkload).
		First(&wl).Error; err != nil {
		return 0, "", "", "", err
	}
	parts := strings.Split(strings.TrimSpace(wl.RefKey), "/")
	if len(parts) < 4 {
		return 0, "", "", "", errors.New("invalid k8s workload ref_key")
	}
	var cid uint
	_, _ = fmt.Sscanf(parts[0], "%d", &cid)
	return cid, parts[1], parts[2], parts[3], nil
}

// --- Cross-domain ---

func (r *CicdRepository) CountFiringAlertsSince(ctx context.Context, projectID uint, since time.Time) (int64, error) {
	var n int64
	q := r.dbq(ctx).Model(&model.AlertEvent{}).
		Where("status = ? AND severity IN ? AND created_at >= ?", "firing", []string{"critical", "warning"}, since)
	if projectID > 0 {
		q = q.Where("project_id = ?", projectID)
	}
	err := q.Count(&n).Error
	return n, err
}

func (r *CicdRepository) CountChangeEventsByRelease(ctx context.Context, releaseRunID uint) (int64, error) {
	var n int64
	like := fmt.Sprintf(`%%"release_id":%d%%`, releaseRunID)
	err := r.dbq(ctx).Model(&model.ChangeEvent{}).
		Where("source = ? AND payload_json LIKE ?", model.ChangeSourceCicd, like).
		Count(&n).Error
	return n, err
}

func (r *CicdRepository) UpdateWorkflowTicketStepFields(ctx context.Context, stepID uint, fields map[string]any) error {
	return r.dbq(ctx).Model(&model.WorkflowTicketStep{}).Where("id = ?", stepID).Updates(fields).Error
}

func isNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
