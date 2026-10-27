package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

// K8sResourceWatchQuery 将 client-go Watch 以 SSE 推送给浏览器（按需，非常驻 Informer 聚合）。
//
// 两种模式（二选一语义）：
// 1) 短名：不传 version，resource 为内置别名（如 pods、deployments）。
// 2) 任意 GVR：传 version（及可选 group，核心组留空）、resource 为 API 复数名（如 pods、horizontalpodautoscalers、customresources），由 RESTMapper 按当前集群 Discovery 校验。
type K8sResourceWatchQuery struct {
	ClusterID        uint   `form:"cluster_id" binding:"required"`
	Namespace        string `form:"namespace"`
	Group            string `form:"group"`
	Version          string `form:"version"`
	Resource         string `form:"resource"`
	LabelSelector    string `form:"label_selector"`
	FieldSelector    string `form:"field_selector"`
	ResourceVersion  string `form:"resource_version"`
	TimeoutSeconds   int    `form:"timeout_seconds"`
	HeartbeatSeconds int    `form:"heartbeat_seconds"`
}

type watchGVR struct {
	GVR        schema.GroupVersionResource
	Namespaced bool
}

// ResolveWatchTarget 解析 Watch 目标（短名表或 GVR+RESTMapper）；cfg 为已生效的集群 kubeconfig。
func ResolveWatchTarget(cfg *rest.Config, q *K8sResourceWatchQuery) (watchGVR, error) {
	ver := strings.TrimSpace(q.Version)
	if ver != "" {
		group := strings.TrimSpace(q.Group)
		res := strings.TrimSpace(q.Resource)
		if res == "" {
			return watchGVR{}, constants.ErrBadRequestWithMsg("GVR 模式需传入 resource（API 复数名，如 pods、jobs）")
		}
		return resolveGVRWithRESTMapper(cfg, group, ver, res)
	}
	slug := strings.TrimSpace(q.Resource)
	if slug == "" {
		return watchGVR{}, constants.ErrBadRequestWithMsg("请传入 resource 短名（如 pods），或同时传入 version+resource 使用 GVR 模式")
	}
	return resolveWatchGVR(slug)
}

func resolveGVRWithRESTMapper(cfg *rest.Config, group, version, resource string) (watchGVR, error) {
	disc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return watchGVR{}, bizerrors.Internalf(context.Background(), "k8s.watch", "discovery_client", err, "discovery client: %v")
	}
	gr, err := restmapper.GetAPIGroupResources(disc)
	if err != nil {
		return watchGVR{}, bizerrors.Internalf(context.Background(), "k8s.watch", "api_groups", err, "discovery: %v")
	}
	mapper := restmapper.NewDiscoveryRESTMapper(gr)

	gvrIn := schema.GroupVersionResource{Group: group, Version: version, Resource: resource}
	resolved, err := mapper.ResourceFor(gvrIn)
	if err != nil {
		return watchGVR{}, constants.ErrBadRequestWithMsg(fmt.Sprintf("RESTMapper 无法解析 GVR %q/%q/%q: %v", group, version, resource, err))
	}
	gvks, err := mapper.KindsFor(resolved)
	if err != nil || len(gvks) == 0 {
		msg := ""
		if err != nil {
			msg = err.Error()
		}
		return watchGVR{}, constants.ErrBadRequestWithMsg(fmt.Sprintf("无法解析资源 Kind（%s）：%s", resolved.String(), msg))
	}
	gvk := gvks[0]
	mapList, err := mapper.RESTMappings(schema.GroupKind{Group: gvk.Group, Kind: gvk.Kind}, gvk.Version)
	if err != nil || len(mapList) == 0 {
		msg := ""
		if err != nil {
			msg = err.Error()
		}
		return watchGVR{}, constants.ErrBadRequestWithMsg(fmt.Sprintf("无法取得 RESTMapping（%s %s）：%s", gvk.Group, gvk.Kind, msg))
	}
	var chosen *apimeta.RESTMapping
	for _, m := range mapList {
		if m != nil && m.Resource == resolved {
			chosen = m
			break
		}
	}
	if chosen == nil {
		chosen = mapList[0]
	}
	namespaced := chosen.Scope.Name() == apimeta.RESTScopeNameNamespace
	return watchGVR{GVR: resolved, Namespaced: namespaced}, nil
}

// WatchResourceLabel 返回 SSE 与 eventbus 使用的资源标识（短名或规范化 GVR 路径）。
func WatchResourceLabel(def watchGVR, q *K8sResourceWatchQuery) string {
	if strings.TrimSpace(q.Version) != "" {
		return def.GVR.Group + "/" + def.GVR.Version + "/" + def.GVR.Resource
	}
	return strings.TrimSpace(q.Resource)
}

func resolveWatchGVR(name string) (watchGVR, error) {
	return builtinWatchGVRBySlug(name)
}

// StreamResourceWatch 将 Watch 结果以 SSE 写入 out（需已设置 Content-Type: text/event-stream）。
// def 须由 ResolveWatchTarget 与 q 使用同一集群 cfg 事先解析。
func (s *K8sRuntimeService) StreamResourceWatch(ctx context.Context, cfg *rest.Config, q K8sResourceWatchQuery, def watchGVR, out io.Writer, flush func()) error {
	ns := strings.TrimSpace(q.Namespace)
	if def.Namespaced && ns == "" {
		return constants.ErrBadRequestWithMsg("该资源为命名空间级，请传入 namespace")
	}
	if !def.Namespaced {
		ns = ""
	}

	timeout := q.TimeoutSeconds
	if timeout <= 0 {
		timeout = 3600
	}
	if timeout > 7200 {
		timeout = 7200
	}
	wctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return bizerrors.Internalf(ctx, "k8s.watch", "dynamic_client", err, "dynamic client: %v")
	}

	nri := dyn.Resource(def.GVR)
	var ri dynamic.ResourceInterface = nri
	if def.Namespaced {
		ri = nri.Namespace(ns)
	}
	selectors := metav1.ListOptions{
		LabelSelector: strings.TrimSpace(q.LabelSelector),
		FieldSelector: strings.TrimSpace(q.FieldSelector),
	}

	hb := q.HeartbeatSeconds
	if hb <= 0 {
		hb = 25
	}
	tick := time.NewTicker(time.Duration(hb) * time.Second)
	defer tick.Stop()

	writeFrame := func(evName string, payload map[string]any) error {
		b, jerr := json.Marshal(payload)
		if jerr != nil {
			return jerr
		}
		line := fmt.Sprintf("event: %s\ndata: %s\n\n", evName, string(b))
		if _, err := io.WriteString(out, line); err != nil {
			return err
		}
		if flush != nil {
			flush()
		}
		return nil
	}

	// 断线续传：LIST 取最新 resourceVersion → 发送 resync（前端据此重新拉取列表）→ 从该版本 Watch。
	// Watch 关闭 / 410 过期 / 短暂网络抖动时自动重建，直到超时或客户端断开。
	const maxConsecutiveFailures = 5
	rv := strings.TrimSpace(q.ResourceVersion)
	failures := 0
	for {
		if wctx.Err() != nil {
			return nil
		}
		if rv == "" {
			listOpts := selectors
			listOpts.Limit = 1
			list, lerr := ri.List(wctx, listOpts)
			if lerr != nil {
				if wctx.Err() != nil {
					return nil
				}
				if !watchFailureFatal(lerr) {
					failures++
				}
				_ = writeFrame("error", map[string]any{
					"cluster_id": q.ClusterID,
					"reason":     "watch_list_failed",
					"message":    lerr.Error(),
					"terminal":   watchFailureFatal(lerr) || failures >= maxConsecutiveFailures,
				})
				if watchFailureFatal(lerr) || failures >= maxConsecutiveFailures {
					return nil
				}
				if !sleepCtx(wctx, watchBackoff(failures)) {
					return nil
				}
				continue
			}
			rv = strings.TrimSpace(list.GetResourceVersion())
			failures = 0
			if err := writeFrame("resync", map[string]any{
				"cluster_id":       q.ClusterID,
				"resource":         WatchResourceLabel(def, &q),
				"resource_version": rv,
			}); err != nil {
				return nil
			}
		}

		watchOpts := selectors
		watchOpts.ResourceVersion = rv
		watchOpts.AllowWatchBookmarks = true
		watcher, werr := ri.Watch(wctx, watchOpts)
		if werr != nil {
			if wctx.Err() != nil {
				return nil
			}
			if !watchFailureFatal(werr) {
				failures++
			}
			_ = writeFrame("error", map[string]any{
				"cluster_id": q.ClusterID,
				"reason":     "watch_failed",
				"message":    werr.Error(),
				"terminal":   watchFailureFatal(werr) || failures >= maxConsecutiveFailures,
			})
			if watchFailureFatal(werr) || failures >= maxConsecutiveFailures {
				return nil
			}
			rv = ""
			if !sleepCtx(wctx, watchBackoff(failures)) {
				return nil
			}
			continue
		}
		failures = 0

		closedNormally := false
	pump:
		for {
			select {
			case <-wctx.Done():
				watcher.Stop()
				return nil
			case <-tick.C:
				if _, err := io.WriteString(out, ":heartbeat\n\n"); err != nil {
					watcher.Stop()
					return nil
				}
				if flush != nil {
					flush()
				}
			case ev, ok := <-watcher.ResultChan():
				if !ok {
					closedNormally = true
					break pump
				}
				if ev.Type == watch.Error {
					st := statusFromObject(ev.Object)
					if watchStatusFatal(ev.Object) {
						_ = writeFrame("error", map[string]any{
							"cluster_id": q.ClusterID,
							"reason":     "watch_error",
							"status":     st,
							"terminal":   true,
						})
						watcher.Stop()
						return nil
					}
					break pump
				}
				u, convErr := runtime.DefaultUnstructuredConverter.ToUnstructured(ev.Object)
				if convErr != nil {
					_ = writeFrame("error", map[string]any{"cluster_id": q.ClusterID, "reason": convErr.Error()})
					continue
				}
				if next := objectResourceVersion(ev.Object); next != "" {
					rv = next
				}
				if err := writeFrame("watch", map[string]any{
					"cluster_id":    q.ClusterID,
					"resource":      WatchResourceLabel(def, &q),
					"group_version": def.GVR.Group + "/" + def.GVR.Version,
					"gvr":           def.GVR.Group + "/" + def.GVR.Version + "/" + def.GVR.Resource,
					"type":          string(ev.Type),
					"object":        u,
				}); err != nil {
					watcher.Stop()
					return nil
				}
			}
		}
		watcher.Stop()
		if wctx.Err() != nil {
			return nil
		}
		if closedNormally {
			// apiserver 正常关闭长连接（常见于超时），立即从上次 RV 续传即可。
			_ = writeFrame("error", map[string]any{
				"cluster_id": q.ClusterID,
				"reason":     "watch_closed",
				"terminal":   false,
			})
			rv = ""
			if !sleepCtx(wctx, time.Second) {
				return nil
			}
			continue
		}
		// Watch 返回错误（如 410 resourceVersion 过期）：重新 LIST 取新版本后续传。
		_ = writeFrame("error", map[string]any{
			"cluster_id": q.ClusterID,
			"reason":     "watch_expired",
			"terminal":   false,
		})
		rv = ""
		if !sleepCtx(wctx, time.Second) {
			return nil
		}
	}
}

func objectResourceVersion(obj any) string {
	if acc, err := apimeta.Accessor(obj); err == nil {
		return strings.TrimSpace(acc.GetResourceVersion())
	}
	return ""
}

func watchStatusFatal(obj any) bool {
	st, ok := obj.(*metav1.Status)
	if !ok || st == nil {
		return false
	}
	return st.Code == 401 || st.Code == 403
}

func watchFailureFatal(err error) bool {
	return apierrors.IsUnauthorized(err) || apierrors.IsForbidden(err)
}

func watchBackoff(failures int) time.Duration {
	if failures <= 0 {
		return time.Second
	}
	d := time.Duration(1<<min(failures-1, 4)) * time.Second
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	return d
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func statusFromObject(obj any) map[string]any {
	if obj == nil {
		return nil
	}
	switch t := obj.(type) {
	case *unstructured.Unstructured:
		if t == nil {
			return nil
		}
		return t.UnstructuredContent()
	case *metav1.Status:
		if t == nil {
			return nil
		}
		out, err := runtime.DefaultUnstructuredConverter.ToUnstructured(t)
		if err != nil {
			return map[string]any{"message": t.Message, "code": t.Code}
		}
		return out
	default:
		out, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return map[string]any{"message": fmt.Sprintf("%v", obj)}
		}
		return out
	}
}
