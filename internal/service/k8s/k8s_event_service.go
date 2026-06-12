package k8s

import (
	"context"
	"sort"
	"strings"
	"yunshu/internal/interfaces"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/k8sauth"
	bizerrors "yunshu/internal/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EventListQuery struct {
	ClusterID uint   `form:"cluster_id" binding:"required"`
	Namespace string `form:"namespace"`
	Kind      string `form:"kind"`
	Name      string `form:"name"`
	Keyword   string `form:"keyword"`
	Limit     int64  `form:"limit"`
}

type EventItem struct {
	Namespace    string `json:"namespace"`
	Type         string `json:"type"`
	Reason       string `json:"reason"`
	Message      string `json:"message"`
	Count        int32  `json:"count"`
	FirstTime    string `json:"first_time,omitempty"`
	LastTime     string `json:"last_time,omitempty"`
	CreationTime string `json:"creation_time,omitempty"`
	InvolvedKind string `json:"involved_kind,omitempty"`
	InvolvedName string `json:"involved_name,omitempty"`
}

// EventGroupItem 按 involvedObject + reason 聚合后的 Event 摘要。
type EventGroupItem struct {
	Namespace    string      `json:"namespace"`
	InvolvedKind string      `json:"involved_kind"`
	InvolvedName string      `json:"involved_name"`
	Reason       string      `json:"reason"`
	Type         string      `json:"type"`
	TotalCount   int32       `json:"total_count"`
	EventCount   int         `json:"event_count"`
	LastTime     string      `json:"last_time"`
	FirstTime    string      `json:"first_time,omitempty"`
	Message      string      `json:"message"`
	Events       []EventItem `json:"events,omitempty"`
}

type K8sEventService struct {
	runtime     *K8sRuntimeService
	nsDenyRepo  interfaces.K8sNamespaceDenyRepository
	nsAllowRepo interfaces.K8sNamespaceAllowRepository
}

// NewK8sEventService 创建相关逻辑。
func NewK8sEventService(
	runtime *K8sRuntimeService,
	nsDeny interfaces.K8sNamespaceDenyRepository,
	nsAllow interfaces.K8sNamespaceAllowRepository,
) *K8sEventService {
	return &K8sEventService{runtime: runtime, nsDenyRepo: nsDeny, nsAllowRepo: nsAllow}
}

// List 查询列表相关的业务逻辑。
func (s *K8sEventService) List(ctx context.Context, q EventListQuery) ([]EventItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	ns := strings.TrimSpace(q.Namespace)
	if ns == "" {
		ns = metav1.NamespaceAll
	}
	limit := q.Limit
	if limit <= 0 || limit > 5000 {
		limit = 500
	}
	var list []corev1.Event
	query := k.WithContext(ctx).Resource(&corev1.Event{}).Limit(int(limit))
	if ns == metav1.NamespaceAll {
		query = query.AllNamespace()
	} else {
		query = query.Namespace(ns)
	}
	if err := query.List(&list).Error; err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.event", "api", err, constants.ErrFmtd678ffdd4e0f)
	}
	kw := strings.ToLower(strings.TrimSpace(q.Keyword))
	kind := strings.TrimSpace(q.Kind)
	name := strings.TrimSpace(q.Name)
	clusterWide := ns == metav1.NamespaceAll
	out := make([]EventItem, 0, len(list))
	for _, e := range list {
		if clusterWide && !s.namespaceAllowed(ctx, q.ClusterID, e.Namespace) {
			continue
		}
		if kind != "" && e.InvolvedObject.Kind != kind {
			continue
		}
		if name != "" && e.InvolvedObject.Name != name {
			continue
		}
		if kw != "" {
			hay := strings.ToLower(e.Reason + " " + e.Message + " " + e.InvolvedObject.Name)
			if !strings.Contains(hay, kw) {
				continue
			}
		}
		first := ""
		last := ""
		if !e.FirstTimestamp.IsZero() {
			first = e.FirstTimestamp.Time.Format("2006-01-02 15:04:05")
		}
		if !e.LastTimestamp.IsZero() {
			last = e.LastTimestamp.Time.Format("2006-01-02 15:04:05")
		}
		creation := ""
		if !e.CreationTimestamp.IsZero() {
			creation = e.CreationTimestamp.Time.Format("2006-01-02 15:04:05")
		}
		out = append(out, EventItem{
			Namespace:    e.Namespace,
			Type:         e.Type,
			Reason:       e.Reason,
			Message:      e.Message,
			Count:        e.Count,
			FirstTime:    first,
			LastTime:     last,
			CreationTime: creation,
			InvolvedKind: e.InvolvedObject.Kind,
			InvolvedName: e.InvolvedObject.Name,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].LastTime > out[j].LastTime
	})
	return out, nil
}

func (s *K8sEventService) namespaceAllowed(ctx context.Context, clusterID uint, namespace string) bool {
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		return true
	}
	u, ok := auth.RequestUserFromContext(ctx)
	if !ok || u == nil || auth.IsSuperAdminRole(u.RoleCodes) {
		return true
	}
	if s.nsDenyRepo == nil && s.nsAllowRepo == nil {
		return true
	}
	pack := k8sauth.PackFromCurrentUser(u)
	allowed, err := NamespaceAllowedByPolicy(ctx, s.nsDenyRepo, s.nsAllowRepo, pack, clusterID, ns)
	return err == nil && allowed
}

// ListGrouped 按 involved_kind + involved_name + reason + namespace 聚合 Event。
func (s *K8sEventService) ListGrouped(ctx context.Context, q EventListQuery) ([]EventGroupItem, error) {
	items, err := s.List(ctx, q)
	if err != nil {
		return nil, err
	}
	type groupKey struct {
		ns, kind, name, reason string
	}
	groups := map[groupKey]*EventGroupItem{}
	order := make([]groupKey, 0)
	for _, e := range items {
		k := groupKey{
			ns:     e.Namespace,
			kind:   e.InvolvedKind,
			name:   e.InvolvedName,
			reason: e.Reason,
		}
		g, ok := groups[k]
		if !ok {
			g = &EventGroupItem{
				Namespace:    e.Namespace,
				InvolvedKind: e.InvolvedKind,
				InvolvedName: e.InvolvedName,
				Reason:       e.Reason,
				Type:         e.Type,
				FirstTime:    e.FirstTime,
				LastTime:     e.LastTime,
				Message:      e.Message,
			}
			groups[k] = g
			order = append(order, k)
		}
		g.TotalCount += e.Count
		if g.TotalCount == 0 {
			g.TotalCount = 1
		}
		g.EventCount++
		if e.LastTime > g.LastTime {
			g.LastTime = e.LastTime
			g.Message = e.Message
			g.Type = e.Type
		}
		if g.FirstTime == "" || (e.FirstTime != "" && e.FirstTime < g.FirstTime) {
			g.FirstTime = e.FirstTime
		}
		if len(g.Events) < 20 {
			g.Events = append(g.Events, e)
		}
	}
	out := make([]EventGroupItem, 0, len(order))
	for _, k := range order {
		if g := groups[k]; g != nil {
			out = append(out, *g)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastTime > out[j].LastTime })
	return out, nil
}
