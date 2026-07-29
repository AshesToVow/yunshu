package cicd

import (
	"context"
	"fmt"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/service/cicd/helmscaffold"
)

// HelmScaffoldQuery 下载脚手架时可覆盖服务默认值。
type HelmScaffoldQuery struct {
	ChartName       string `form:"chart_name"`
	ImageRepository string `form:"image_repository"`
	ReplicaCount    int    `form:"replica_count"`
	ContainerPort   int    `form:"container_port"`
	ServicePort     int    `form:"service_port"`
}

// BuildHelmScaffoldZip 按服务标识与发布配置生成 helm/ 目录 zip。
func (s *Service) BuildHelmScaffoldZip(ctx context.Context, projectID, serviceID uint, q HelmScaffoldQuery) (filename string, data []byte, err error) {
	svc, err := s.GetService(ctx, projectID, serviceID)
	if err != nil {
		return "", nil, err
	}
	opts := helmscaffold.Options{
		ChartName:       firstNonEmpty(q.ChartName, svc.Identifier, svc.Name),
		ImageRepository: strings.TrimSpace(q.ImageRepository),
		ReplicaCount:    q.ReplicaCount,
		ContainerPort:   q.ContainerPort,
		ServicePort:     q.ServicePort,
	}
	if opts.ImageRepository == "" || opts.ReplicaCount <= 0 || opts.ContainerPort <= 0 {
		var cfg model.CicdDeployConfig
		_ = s.db.WithContext(ctx).
			Where("service_id = ? AND deploy_kind = ?", serviceID, "container").
			Order("id DESC").
			First(&cfg).Error
		if opts.ImageRepository == "" && strings.TrimSpace(cfg.ImageName) != "" {
			opts.ImageRepository = strings.TrimSpace(cfg.ImageName)
			if !strings.Contains(opts.ImageRepository, "/") {
				opts.ImageRepository = fmt.Sprintf("harbor.example/registry/%s", opts.ImageRepository)
			}
		}
		if opts.ReplicaCount <= 0 && cfg.Replicas > 0 {
			opts.ReplicaCount = cfg.Replicas
		}
		if opts.ContainerPort <= 0 && cfg.ContainerPort > 0 {
			opts.ContainerPort = cfg.ContainerPort
		}
	}
	filename, data, err = helmscaffold.BuildZip(opts)
	if err != nil {
		return "", nil, constants.ErrInternalWithMsg("生成 Helm 脚手架失败: " + err.Error())
	}
	return filename, data, nil
}

// BuildHelmScaffoldZipPreview 未绑定服务时按表单参数生成 zip。
func (s *Service) BuildHelmScaffoldZipPreview(_ context.Context, q HelmScaffoldQuery) (filename string, data []byte, err error) {
	if strings.TrimSpace(q.ChartName) == "" {
		return "", nil, constants.ErrBadRequestWithMsg("chart_name 不能为空")
	}
	filename, data, err = helmscaffold.BuildZip(helmscaffold.Options{
		ChartName:       q.ChartName,
		ImageRepository: q.ImageRepository,
		ReplicaCount:    q.ReplicaCount,
		ContainerPort:   q.ContainerPort,
		ServicePort:     q.ServicePort,
	})
	if err != nil {
		return "", nil, constants.ErrInternalWithMsg("生成 Helm 脚手架失败: " + err.Error())
	}
	return filename, data, nil
}
