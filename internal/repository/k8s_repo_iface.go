package repository

import (
	"context"

	"yunshu/internal/model"
	"yunshu/internal/pkg/k8sauth"
	"yunshu/internal/pkg/pagination"
)

// K8sClusterRepo is implemented by *K8sClusterRepository.
type K8sClusterRepo interface {
	Create(ctx context.Context, cluster *model.K8sCluster) (error)
	Update(ctx context.Context, cluster *model.K8sCluster) (error)
	Delete(ctx context.Context, id uint) (error)
	GetByID(ctx context.Context, id uint) (*model.K8sCluster, error)
	List(ctx context.Context, params K8sClusterListParams) ([]model.K8sCluster, int64, error)
	ListAllBrief(ctx context.Context) ([]model.K8sCluster, error)
}

var _ K8sClusterRepo = (*K8sClusterRepository)(nil)

// K8sClusterAccessRepo is implemented by *K8sClusterAccessRepository.
type K8sClusterAccessRepo interface {
	Upsert(ctx context.Context, it *model.K8sClusterAccessGrant) (error)
	ListGrantsApplyingToCluster(ctx context.Context, clusterID uint) ([]model.K8sClusterAccessGrant, error)
	ListByPrincipal(ctx context.Context, kind string, ref string) ([]model.K8sClusterAccessGrant, error)
	ListByRoleCode(ctx context.Context, roleCode string) ([]model.K8sClusterAccessGrant, error)
	GetByID(ctx context.Context, id uint) (*model.K8sClusterAccessGrant, error)
	DeleteByID(ctx context.Context, id uint) (error)
	EffectiveTier(ctx context.Context, pack k8sauth.PrincipalPack, clusterID uint) (int)
	EffectiveCapabilities(ctx context.Context, pack k8sauth.PrincipalPack, clusterID uint) []string
	BuildEffectiveTierIndex(ctx context.Context, pack k8sauth.PrincipalPack) (EffectiveTierIndex, error)
	HasAnyK8sGrant(ctx context.Context, pack k8sauth.PrincipalPack) (bool)
}

var _ K8sClusterAccessRepo = (*K8sClusterAccessRepository)(nil)

// K8sNamespaceAllowRepo is implemented by *K8sNamespaceAllowRepository.
type K8sNamespaceAllowRepo interface {
	WhitelistActiveForCluster(ctx context.Context, pack k8sauth.PrincipalPack, clusterID uint) (bool, error)
	NamespaceAllowed(ctx context.Context, pack k8sauth.PrincipalPack, clusterID uint, namespace string) (bool, error)
	WhitelistUnionNamespaces(ctx context.Context, pack k8sauth.PrincipalPack, clusterID uint) ([]string, error)
	DistinctNamespacesForPrincipalCluster(ctx context.Context, principalKind string, principalRef string, clusterID uint) ([]string, error)
	List(ctx context.Context, principalKind string, principalRef string, clusterID uint) ([]model.K8sNamespaceAllowRule, error)
	Create(ctx context.Context, it *model.K8sNamespaceAllowRule) (error)
	DeleteByID(ctx context.Context, id uint) (error)
	DeleteByPrincipalCluster(ctx context.Context, principalKind string, principalRef string, clusterID uint) (error)
}

var _ K8sNamespaceAllowRepo = (*K8sNamespaceAllowRepository)(nil)

// K8sNamespaceDenyRepo is implemented by *K8sNamespaceDenyRepository.
type K8sNamespaceDenyRepo interface {
	IsDenied(ctx context.Context, pack k8sauth.PrincipalPack, clusterID uint, namespace string) (bool, error)
	DeniedNamespaceNames(ctx context.Context, pack k8sauth.PrincipalPack, clusterID uint) ([]string, error)
	List(ctx context.Context, principalKind string, principalRef string, clusterID uint) ([]model.K8sNamespaceDenyRule, error)
	Create(ctx context.Context, it *model.K8sNamespaceDenyRule) (error)
	DeleteByID(ctx context.Context, id uint) (error)
	DeleteByPrincipalCluster(ctx context.Context, principalKind string, principalRef string, clusterID uint) (error)
}

var _ K8sNamespaceDenyRepo = (*K8sNamespaceDenyRepository)(nil)

// K8sCrTemplateListFilter filters CR/YAML templates.
type K8sCrTemplateListFilter struct {
	ProjectID uint
	Kind      string
}

// K8sCrTemplateRepo is implemented by *K8sCrTemplateRepository.
type K8sCrTemplateRepo interface {
	List(ctx context.Context, f K8sCrTemplateListFilter) ([]model.K8sCrTemplate, error)
	GetByID(ctx context.Context, id uint) (*model.K8sCrTemplate, error)
	Create(ctx context.Context, row *model.K8sCrTemplate) error
	Save(ctx context.Context, row *model.K8sCrTemplate) error
	DeleteByID(ctx context.Context, id uint) (rowsAffected int64, err error)
}

var _ K8sCrTemplateRepo = (*K8sCrTemplateRepository)(nil)

// K8sWorkloadSnapshotListParams filters workload snapshots with pagination.
type K8sWorkloadSnapshotListParams struct {
	ProjectID uint
	ClusterID uint
	Namespace string
	Kind      string
	Name      string
	Page      int
	PageSize  int
}

// K8sWorkloadSnapshotRepo is implemented by *K8sWorkloadSnapshotRepository.
type K8sWorkloadSnapshotRepo interface {
	Create(ctx context.Context, row *model.K8sWorkloadSnapshot) error
	GetByID(ctx context.Context, id uint) (*model.K8sWorkloadSnapshot, error)
	List(ctx context.Context, p K8sWorkloadSnapshotListParams) (*pagination.Result[model.K8sWorkloadSnapshot], error)
}

var _ K8sWorkloadSnapshotRepo = (*K8sWorkloadSnapshotRepository)(nil)

// HarborMergeRepo reads image registry / project Harbor fields for Helm Harbor resolution.
type HarborMergeRepo interface {
	GetDefaultEnabledHarborRegistry(ctx context.Context) (*model.ImageRegistry, error)
	GetProjectRegistryBinding(ctx context.Context, projectID uint) (*model.ProjectRegistryBinding, error)
	GetImageRegistryByID(ctx context.Context, id uint) (*model.ImageRegistry, error)
	GetProjectHarborFields(ctx context.Context, projectID uint) (harborURL, harborProject string, err error)
}

var _ HarborMergeRepo = (*HarborMergeRepository)(nil)

