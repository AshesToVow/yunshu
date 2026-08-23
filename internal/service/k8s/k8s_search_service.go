package k8s

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/k8sauth"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/repository"

	corev1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

type K8sSearchQuery struct {
	Q     string `form:"q" binding:"required"`
	Types string `form:"types"`
	Limit int    `form:"limit"`
}

type K8sSearchItem struct {
	Type        string `json:"type"`
	ClusterID   uint   `json:"cluster_id"`
	ClusterName string `json:"cluster_name"`
	Namespace   string `json:"namespace"`
	Name        string `json:"name"`
	Extra       string `json:"extra,omitempty"`
	Status      string `json:"status,omitempty"`
	LinkPath    string `json:"link_path,omitempty"`
}

type K8sSearchService struct {
	runtime     *K8sRuntimeService
	clusterRepo interfaces.K8sClusterRepository
	memberRepo  interfaces.ProjectMemberRepository
	accessRepo  interfaces.K8sClusterAccessRepository
	nsDenyRepo  interfaces.K8sNamespaceDenyRepository
	nsAllowRepo interfaces.K8sNamespaceAllowRepository
}

func NewK8sSearchService(
	runtime *K8sRuntimeService,
	clusterRepo interfaces.K8sClusterRepository,
	memberRepo interfaces.ProjectMemberRepository,
	accessRepo interfaces.K8sClusterAccessRepository,
	nsDeny interfaces.K8sNamespaceDenyRepository,
	nsAllow interfaces.K8sNamespaceAllowRepository,
) *K8sSearchService {
	return &K8sSearchService{
		runtime: runtime, clusterRepo: clusterRepo, memberRepo: memberRepo, accessRepo: accessRepo,
		nsDenyRepo: nsDeny, nsAllowRepo: nsAllow,
	}
}

func (s *K8sSearchService) Search(ctx context.Context, q K8sSearchQuery) ([]K8sSearchItem, error) {
	q.Q = strings.TrimSpace(q.Q)
	if q.Q == "" {
		return nil, nil
	}
	limit := q.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	types := parseSearchTypes(q.Types)
	clusters, err := s.accessibleClusters(ctx)
	if err != nil {
		return nil, err
	}
	var (
		mu    sync.Mutex
		wg    sync.WaitGroup
		out   []K8sSearchItem
		sem   = make(chan struct{}, 5)
	)
	for _, cl := range clusters {
		cl := cl
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			items := s.searchCluster(cctx, cl, q.Q, types, limit)
			if len(items) == 0 {
				return
			}
			mu.Lock()
			out = append(out, items...)
			mu.Unlock()
		}()
	}
	wg.Wait()
	if len(out) > limit*4 {
		out = out[:limit*4]
	}
	return out, nil
}

func parseSearchTypes(raw string) map[string]bool {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return map[string]bool{"pod": true, "service": true, "ingress": true, "event": true, "deployment": true, "configmap": true, "namespace": true}
	}
	out := map[string]bool{}
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out[p] = true
		}
	}
	return out
}

func (s *K8sSearchService) accessibleClusters(ctx context.Context) ([]model.K8sCluster, error) {
	params := repository.K8sClusterListParams{Page: 1, PageSize: 500}
	if u, ok := auth.RequestUserFromContext(ctx); ok && u != nil && !auth.IsSuperAdminRole(u.RoleCodes) && s.memberRepo != nil {
		ids, _ := s.memberRepo.ListProjectIDsByUser(ctx, u.ID)
		params.ProjectMemberFilter = true
		params.ProjectMemberIDs = ids
	}
	clusters, _, err := s.clusterRepo.List(ctx, params)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "k8s.search", "ListClusters", err)
	}
	enabled := make([]model.K8sCluster, 0, len(clusters))
	for _, c := range clusters {
		if c.Status == model.StatusEnabled {
			enabled = append(enabled, c)
		}
	}
	u, ok := auth.RequestUserFromContext(ctx)
	if !ok || u == nil || auth.IsSuperAdminRole(u.RoleCodes) || s.accessRepo == nil {
		return enabled, nil
	}
	pack := k8sauth.PackFromCurrentUser(u)
	idx, err := s.accessRepo.BuildEffectiveTierIndex(ctx, pack)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "k8s.search", "BuildEffectiveTierIndex", err)
	}
	out := make([]model.K8sCluster, 0, len(enabled))
	for _, c := range enabled {
		if idx.ClusterAccessible(c.ID, K8sAccessRankReadonly) {
			out = append(out, c)
		}
	}
	return out, nil
}

func (s *K8sSearchService) searchCluster(ctx context.Context, cl model.K8sCluster, q string, types map[string]bool, limit int) []K8sSearchItem {
	_, k, err := s.runtime.GetClusterKubectl(ctx, cl.ID)
	if err != nil {
		return nil
	}
	kw := strings.ToLower(q)
	var items []K8sSearchItem
	add := func(typ, ns, name, extra, status, link string) {
		if len(items) >= limit {
			return
		}
		if !s.namespaceAllowed(ctx, cl.ID, ns) {
			return
		}
		if !strings.Contains(strings.ToLower(name), kw) && !strings.Contains(strings.ToLower(extra), kw) {
			return
		}
		items = append(items, K8sSearchItem{
			Type: typ, ClusterID: cl.ID, ClusterName: cl.Name, Namespace: ns, Name: name,
			Extra: extra, Status: status, LinkPath: link,
		})
	}
	if types["pod"] {
		var pods []corev1.Pod
		if err := k.WithContext(ctx).Resource(&corev1.Pod{}).AllNamespace().List(&pods).Error; err == nil {
			for _, p := range pods {
				add("pod", p.Namespace, p.Name, string(p.Status.Phase), string(p.Status.Phase),
					"/pods?cluster_id="+itoa(cl.ID)+"&namespace="+p.Namespace+"&keyword="+p.Name)
			}
		}
	}
	if types["service"] {
		var svcs []corev1.Service
		if err := k.WithContext(ctx).Resource(&corev1.Service{}).AllNamespace().List(&svcs).Error; err == nil {
			for _, svc := range svcs {
				add("service", svc.Namespace, svc.Name, svc.Spec.ClusterIP, string(svc.Spec.Type),
					"/k8s-services?cluster_id="+itoa(cl.ID)+"&namespace="+svc.Namespace+"&keyword="+svc.Name)
			}
		}
	}
	if types["ingress"] {
		var ings []networkingv1.Ingress
		if err := k.WithContext(ctx).Resource(&networkingv1.Ingress{}).AllNamespace().List(&ings).Error; err == nil {
			for _, ing := range ings {
				hosts := ingressHosts(&ing)
				add("ingress", ing.Namespace, ing.Name, hosts, "",
					"/ingresses?cluster_id="+itoa(cl.ID)+"&namespace="+ing.Namespace+"&keyword="+ing.Name)
			}
		}
	}
	if types["event"] {
		var events []corev1.Event
		if err := k.WithContext(ctx).Resource(&corev1.Event{}).AllNamespace().Limit(300).List(&events).Error; err == nil {
			for _, e := range events {
				hay := strings.ToLower(e.Reason + " " + e.Message + " " + e.InvolvedObject.Name)
				if !strings.Contains(hay, kw) {
					continue
				}
				add("event", e.Namespace, e.InvolvedObject.Name, e.Reason+" | "+e.Message, e.Type,
					"/events?cluster_id="+itoa(cl.ID)+"&namespace="+e.Namespace+"&keyword="+e.InvolvedObject.Name)
			}
		}
	}
	if types["deployment"] {
		var deps []appsv1.Deployment
		if err := k.WithContext(ctx).Resource(&appsv1.Deployment{}).AllNamespace().List(&deps).Error; err == nil {
			for _, d := range deps {
				add("deployment", d.Namespace, d.Name, fmt.Sprintf("%d/%d", d.Status.ReadyReplicas, d.Status.Replicas), "",
					"/deployments?cluster_id="+itoa(cl.ID)+"&namespace="+d.Namespace+"&keyword="+d.Name)
			}
		}
	}
	if types["configmap"] {
		var cms []corev1.ConfigMap
		if err := k.WithContext(ctx).Resource(&corev1.ConfigMap{}).AllNamespace().List(&cms).Error; err == nil {
			for _, cm := range cms {
				add("configmap", cm.Namespace, cm.Name, "", "",
					"/configmaps?cluster_id="+itoa(cl.ID)+"&namespace="+cm.Namespace+"&keyword="+cm.Name)
			}
		}
	}
	if types["namespace"] {
		var nss []corev1.Namespace
		if err := k.WithContext(ctx).Resource(&corev1.Namespace{}).List(&nss).Error; err == nil {
			for _, ns := range nss {
				add("namespace", ns.Name, ns.Name, string(ns.Status.Phase), string(ns.Status.Phase),
					"/namespaces?cluster_id="+itoa(cl.ID)+"&keyword="+ns.Name)
			}
		}
	}
	return items
}

func (s *K8sSearchService) namespaceAllowed(ctx context.Context, clusterID uint, namespace string) bool {
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		return true
	}
	u, ok := auth.RequestUserFromContext(ctx)
	if !ok || u == nil {
		return true
	}
	if s.nsDenyRepo == nil && s.nsAllowRepo == nil {
		return true
	}
	pack := k8sauth.PackFromCurrentUser(u)
	allowed, err := NamespaceAllowedByPolicy(ctx, s.nsDenyRepo, s.nsAllowRepo, pack, clusterID, ns)
	return err == nil && allowed
}

func ingressHosts(ing *networkingv1.Ingress) string {
	var hosts []string
	for _, r := range ing.Spec.Rules {
		if r.Host != "" {
			hosts = append(hosts, r.Host)
		}
	}
	return strings.Join(hosts, ",")
}

func itoa(n uint) string {
	return strconv.FormatUint(uint64(n), 10)
}
