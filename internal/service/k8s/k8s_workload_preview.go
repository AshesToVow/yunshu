package k8s

import (
	"context"
	"fmt"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/k8sutil"
	"yunshu/internal/pkg/pagination"
	bizerrors "yunshu/internal/pkg/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/yaml"
)

type WorkloadPreviewResult struct {
	DryRunOK  bool                  `json:"dry_run_ok"`
	Message   string                `json:"message,omitempty"`
	Diffs     []WorkloadYAMLDiff    `json:"diffs"`
	Impact    WorkloadImpactSummary `json:"impact"`
	Refs      []workloadRef         `json:"refs"`
}

type WorkloadYAMLDiff struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Exists    bool   `json:"exists"`
	Before    string `json:"before,omitempty"`
	After     string `json:"after,omitempty"`
	Unified   string `json:"unified,omitempty"`
}

type WorkloadImpactSummary struct {
	Secrets       []string `json:"secrets,omitempty"`
	ConfigMaps    []string `json:"config_maps,omitempty"`
	PVCs          []string `json:"pvcs,omitempty"`
	ServiceAccounts []string `json:"service_accounts,omitempty"`
	ReplicaHint   string   `json:"replica_hint,omitempty"`
}

type SnapshotListQuery struct {
	ProjectID uint   `form:"project_id"`
	ClusterID uint   `form:"cluster_id"`
	Namespace string `form:"namespace"`
	Kind      string `form:"kind"`
	Name      string `form:"name"`
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
}

type SnapshotRollbackRequest struct {
	SnapshotID uint `json:"snapshot_id" binding:"required"`
	ClusterID  uint `json:"cluster_id" binding:"required"`
}

// PreviewApply server-side 预检：拉取现网 YAML、做文本 diff、解析引用影响；不落库。
func (s *K8sWorkloadService) PreviewApply(ctx context.Context, req NamespacedApplyRequest) (*WorkloadPreviewResult, error) {
	cluster, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Manifest) == "" {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg01433598170d)
	}
	if err := s.dyn.ensureManifestNamespacesAllowed(ctx, req.Manifest); err != nil {
		return nil, err
	}
	refs := extractWorkloadRefsForApply(req.Manifest)
	out := &WorkloadPreviewResult{DryRunOK: true, Refs: refs, Impact: extractImpactFromManifest(req.Manifest)}
	docs := mapDocsByRef(req.Manifest)
	for _, r := range refs {
		gvk, ok := s.dyn.GVKByKind(r.Kind)
		if !ok {
			continue
		}
		diff := WorkloadYAMLDiff{Kind: r.Kind, Namespace: r.Namespace, Name: r.Name, After: docs[refKey(r)]}
		u, err := s.dyn.GetByGVK(ctx, k, gvk, r.Namespace, r.Name)
		if err != nil {
			if apierrors.IsNotFound(err) {
				diff.Exists = false
				diff.Unified = unifiedDiff("", diff.After, r.Name+" (create)")
				out.Diffs = append(out.Diffs, diff)
				continue
			}
			out.DryRunOK = false
			out.Message = err.Error()
			return out, nil
		}
		diff.Exists = true
		y, _ := yaml.Marshal(u.Object)
		diff.Before = string(y)
		diff.Unified = unifiedDiff(diff.Before, diff.After, r.Kind+"/"+r.Namespace+"/"+r.Name)
		out.Diffs = append(out.Diffs, diff)
	}
	_ = cluster
	return out, nil
}

// Apply 覆盖：门禁 + 变更前快照 + diff 写入 change_events。
func (s *K8sWorkloadService) Apply(ctx context.Context, req NamespacedApplyRequest) error {
	cluster, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(req.Manifest) == "" {
		return constants.ErrBadRequestWithMsg(constants.ErrMsg01433598170d)
	}
	refs := extractWorkloadRefsForApply(req.Manifest)
	ns := ""
	if len(refs) > 0 {
		ns = refs[0].Namespace
	}
	if err := assertK8sWritable(ctx, cluster, "apply", ns); err != nil {
		return err
	}

	var snapIDs []uint
	docs := mapDocsByRef(req.Manifest)
	pid := projectIDOf(cluster)
	var actorID *uint
	if u, ok := auth.RequestUserFromContext(ctx); ok && u != nil {
		id := u.ID
		actorID = &id
	}
	for _, r := range refs {
		gvk, ok := s.dyn.GVKByKind(r.Kind)
		if !ok || pid == 0 {
			continue
		}
		u, err := s.dyn.GetByGVK(ctx, k, gvk, r.Namespace, r.Name)
		if err != nil {
			continue
		}
		y, _ := yaml.Marshal(u.Object)
		row := model.K8sWorkloadSnapshot{
			ProjectID: pid,
			ClusterID: req.ClusterID,
			Namespace: r.Namespace,
			Kind:      r.Kind,
			Name:      r.Name,
			YAML:      string(y),
			ActorID:   actorID,
			Reason:    "before_apply",
		}
		if s.db != nil {
			if err := s.db.WithContext(ctx).Create(&row).Error; err == nil {
				snapIDs = append(snapIDs, row.ID)
			}
		}
	}

	err = s.dyn.ApplyManifest(ctx, k, req.Manifest, func(c context.Context) bool {
		if len(refs) == 0 {
			return false
		}
		for _, r := range refs {
			if strings.TrimSpace(r.Name) == "" {
				continue
			}
			if !s.dyn.ExistsByKind(c, k, r.Kind, r.Namespace, r.Name) {
				return false
			}
		}
		return true
	})
	if err != nil {
		return k8sFail(ctx, "k8s.workload", "api", err)
	}

	diffs := make([]map[string]any, 0, len(refs))
	for _, r := range refs {
		diffs = append(diffs, map[string]any{
			"kind": r.Kind, "namespace": r.Namespace, "name": r.Name,
			"after_len": len(docs[refKey(r)]),
		})
	}
	payload := map[string]any{
		"refs":         refs,
		"impact":       extractImpactFromManifest(req.Manifest),
		"snapshot_ids": snapIDs,
		"diff_summary": diffs,
	}
	if len(refs) > 0 {
		r := refs[0]
		recordK8sChange(ctx, cluster, "apply", r.Kind, r.Namespace, r.Name, payload)
	} else {
		recordK8sChange(ctx, cluster, "apply", "Workload", "", "", payload)
	}
	return nil
}

func (s *K8sWorkloadService) deleteWorkloadByKind(ctx context.Context, req NamespacedDeleteRequest, kind string) error {
	cluster, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if err := assertK8sWritable(ctx, cluster, "delete", req.Namespace); err != nil {
		return err
	}
	gvk, ok := s.dyn.GVKByKind(kind)
	if !ok {
		return constants.ErrBadRequestWithMsg(constants.ErrMsgd5692b195622)
	}
	pid := projectIDOf(cluster)
	var actorID *uint
	if u, ok := auth.RequestUserFromContext(ctx); ok && u != nil {
		id := u.ID
		actorID = &id
	}
	var snapID uint
	if u, err := s.dyn.GetByGVK(ctx, k, gvk, req.Namespace, req.Name); err == nil && pid > 0 {
		y, _ := yaml.Marshal(u.Object)
		row := model.K8sWorkloadSnapshot{
			ProjectID: pid, ClusterID: req.ClusterID,
			Namespace: req.Namespace, Kind: kind, Name: req.Name,
			YAML: string(y), ActorID: actorID, Reason: "before_delete",
		}
		if s.db != nil {
		if s.db != nil {
			if err := s.db.WithContext(ctx).Create(&row).Error; err == nil {
				snapID = row.ID
			}
		}
		}
	}
	if err := s.dyn.DeleteByGVK(ctx, k, gvk, req.Namespace, req.Name, req.K8sDeleteOptions); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return bizerrors.Internalf(ctx, "k8s.workload", "delete", err, constants.ErrFmt32b88f9cc2e5, kind)
	}
	recordK8sChange(ctx, cluster, "delete", kind, req.Namespace, req.Name, map[string]any{
		"impact":      "resource_removed",
		"snapshot_id": snapID,
	})
	return nil
}

func (s *K8sWorkloadService) ListSnapshots(ctx context.Context, q SnapshotListQuery) (*pagination.Result[model.K8sWorkloadSnapshot], error) {
	if q.ProjectID == 0 && q.ClusterID == 0 {
		return nil, constants.ErrBadRequestWithMsg("project_id or cluster_id required")
	}
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	dbq := s.db.WithContext(ctx).Model(&model.K8sWorkloadSnapshot{})
	if q.ProjectID > 0 {
		dbq = dbq.Where("project_id = ?", q.ProjectID)
	}
	if q.ClusterID > 0 {
		dbq = dbq.Where("cluster_id = ?", q.ClusterID)
	}
	if ns := strings.TrimSpace(q.Namespace); ns != "" {
		dbq = dbq.Where("namespace = ?", ns)
	}
	if kind := strings.TrimSpace(q.Kind); kind != "" {
		dbq = dbq.Where("kind = ?", kind)
	}
	if name := strings.TrimSpace(q.Name); name != "" {
		dbq = dbq.Where("name = ?", name)
	}
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, err
	}
	var list []model.K8sWorkloadSnapshot
	if err := dbq.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, err
	}
	return &pagination.Result[model.K8sWorkloadSnapshot]{List: list, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *K8sWorkloadService) RollbackSnapshot(ctx context.Context, req SnapshotRollbackRequest) error {
	var snap model.K8sWorkloadSnapshot
	if err := s.db.WithContext(ctx).Where("id = ?", req.SnapshotID).First(&snap).Error; err != nil {
		return constants.ErrNotFound
	}
	cluster, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if snap.ClusterID != 0 && snap.ClusterID != req.ClusterID {
		return constants.ErrBadRequestWithMsg("快照与集群不匹配")
	}
	if err := assertK8sWritable(ctx, cluster, "rollback", snap.Namespace); err != nil {
		return err
	}
	if err := s.dyn.ApplyManifest(ctx, k, snap.YAML, nil); err != nil {
		return k8sFail(ctx, "k8s.workload", "rollback", err)
	}
	recordK8sChange(ctx, cluster, "rollback", snap.Kind, snap.Namespace, snap.Name, map[string]any{
		"snapshot_id": snap.ID,
		"impact":      "restored_from_snapshot",
	})
	return nil
}

func refKey(r workloadRef) string {
	return fmt.Sprintf("%s/%s/%s", r.Kind, r.Namespace, r.Name)
}

func mapDocsByRef(manifest string) map[string]string {
	out := map[string]string{}
	for _, doc := range k8sutil.SplitYAMLDocs(manifest) {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}
		var m map[string]any
		if err := yaml.Unmarshal([]byte(doc), &m); err != nil {
			continue
		}
		kind, _ := m["kind"].(string)
		meta, _ := m["metadata"].(map[string]any)
		if meta == nil {
			continue
		}
		name, _ := meta["name"].(string)
		ns, _ := meta["namespace"].(string)
		if ns == "" {
			ns = "default"
		}
		out[fmt.Sprintf("%s/%s/%s", strings.TrimSpace(kind), ns, strings.TrimSpace(name))] = doc
	}
	return out
}

func extractImpactFromManifest(manifest string) WorkloadImpactSummary {
	var impact WorkloadImpactSummary
	seenS, seenC, seenP, seenSA := map[string]struct{}{}, map[string]struct{}{}, map[string]struct{}{}, map[string]struct{}{}
	add := func(set map[string]struct{}, list *[]string, v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		if _, ok := set[v]; ok {
			return
		}
		set[v] = struct{}{}
		*list = append(*list, v)
	}
	for _, doc := range k8sutil.SplitYAMLDocs(manifest) {
		var m map[string]any
		if err := yaml.Unmarshal([]byte(doc), &m); err != nil {
			continue
		}
		spec, _ := m["spec"].(map[string]any)
		if spec == nil {
			continue
		}
		if replicas, ok := spec["replicas"]; ok {
			impact.ReplicaHint = fmt.Sprintf("%v", replicas)
		}
		tpl, _ := spec["template"].(map[string]any)
		if tpl == nil {
			continue
		}
		podSpec, _ := tpl["spec"].(map[string]any)
		if podSpec == nil {
			continue
		}
		if sa, _ := podSpec["serviceAccountName"].(string); sa != "" {
			add(seenSA, &impact.ServiceAccounts, sa)
		}
		for _, key := range []string{"volumes"} {
			arr, _ := podSpec[key].([]any)
			for _, item := range arr {
				vm, _ := item.(map[string]any)
				if vm == nil {
					continue
				}
				if sec, _ := vm["secret"].(map[string]any); sec != nil {
					if n, _ := sec["secretName"].(string); n != "" {
						add(seenS, &impact.Secrets, n)
					}
				}
				if cm, _ := vm["configMap"].(map[string]any); cm != nil {
					if n, _ := cm["name"].(string); n != "" {
						add(seenC, &impact.ConfigMaps, n)
					}
				}
				if pvc, _ := vm["persistentVolumeClaim"].(map[string]any); pvc != nil {
					if n, _ := pvc["claimName"].(string); n != "" {
						add(seenP, &impact.PVCs, n)
					}
				}
			}
		}
		containers, _ := podSpec["containers"].([]any)
		for _, c := range containers {
			cm, _ := c.(map[string]any)
			if cm == nil {
				continue
			}
			for _, envFrom := range []string{"envFrom"} {
				arr, _ := cm[envFrom].([]any)
				for _, item := range arr {
					em, _ := item.(map[string]any)
					if em == nil {
						continue
					}
					if sec, _ := em["secretRef"].(map[string]any); sec != nil {
						if n, _ := sec["name"].(string); n != "" {
							add(seenS, &impact.Secrets, n)
						}
					}
					if cref, _ := em["configMapRef"].(map[string]any); cref != nil {
						if n, _ := cref["name"].(string); n != "" {
							add(seenC, &impact.ConfigMaps, n)
						}
					}
				}
			}
		}
	}
	return impact
}

func unifiedDiff(before, after, title string) string {
	bLines := strings.Split(before, "\n")
	aLines := strings.Split(after, "\n")
	var b strings.Builder
	b.WriteString("--- before " + title + "\n")
	b.WriteString("+++ after " + title + "\n")
	max := len(bLines)
	if len(aLines) > max {
		max = len(aLines)
	}
	limit := max
	if limit > 200 {
		limit = 200
	}
	for i := 0; i < limit; i++ {
		var bl, al string
		if i < len(bLines) {
			bl = bLines[i]
		}
		if i < len(aLines) {
			al = aLines[i]
		}
		if bl == al {
			continue
		}
		if bl != "" {
			b.WriteString("- " + bl + "\n")
		}
		if al != "" {
			b.WriteString("+ " + al + "\n")
		}
	}
	if max > 200 {
		b.WriteString(fmt.Sprintf("... truncated (%d lines)\n", max))
	}
	return b.String()
}
