package project

import (
	"context"
	"errors"
	"fmt"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/repository"
	bizerrors "yunshu/internal/pkg/errors"

	"gorm.io/gorm"
)

// TopologyNode 应用拓扑节点（与 K8s 资源拓扑结构一致，便于前端复用）。
type TopologyNode struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Kind       string `json:"kind"`
	State      string `json:"state,omitempty"`
	StateLevel string `json:"state_level,omitempty"`
}

type TopologyEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind,omitempty"`
}

type ApplicationTopologyGraph struct {
	Nodes []TopologyNode `json:"nodes"`
	Edges []TopologyEdge `json:"edges"`
}

type appGraphBuilder struct {
	nodes   []TopologyNode
	edges   []TopologyEdge
	nodeSet map[string]struct{}
	edgeSet map[string]struct{}
}

func newAppGraphBuilder() *appGraphBuilder {
	return &appGraphBuilder{nodeSet: map[string]struct{}{}, edgeSet: map[string]struct{}{}}
}

func (b *appGraphBuilder) addNode(id, label, kind, state, stateLevel string) {
	if _, ok := b.nodeSet[id]; ok {
		return
	}
	b.nodeSet[id] = struct{}{}
	b.nodes = append(b.nodes, TopologyNode{ID: id, Label: label, Kind: kind, State: state, StateLevel: stateLevel})
}

func (b *appGraphBuilder) addEdge(from, to, kind string) {
	key := from + "->" + to
	if _, ok := b.edgeSet[key]; ok {
		return
	}
	b.edgeSet[key] = struct{}{}
	b.edges = append(b.edges, TopologyEdge{From: from, To: to, Kind: kind})
}

func (b *appGraphBuilder) build() *ApplicationTopologyGraph {
	return &ApplicationTopologyGraph{Nodes: b.nodes, Edges: b.edges}
}

func appNodeID(kind string, id uint) string {
	return fmt.Sprintf("%s/%d", kind, id)
}

func lifecycleStateLevel(s string) string {
	switch s {
	case model.ProjectLifecyclePlanning:
		return "progressing"
	case model.ProjectLifecycleSuspended, model.ProjectLifecycleArchived:
		return "abnormal"
	default:
		return "normal"
	}
}

// ApplicationTopology 构建项目应用拓扑（参考 KubeVela App Topology：应用 → 组件 → 底层资源）。
func (s *ProjectMgmtService) ApplicationTopology(ctx context.Context, projectID uint) (*ApplicationTopologyGraph, error) {
	p, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrProjectNotFound
		}
		return nil, bizerrors.Pass(ctx, "project", "ApplicationTopology", err)
	}

	b := newAppGraphBuilder()
	rootID := appNodeID("project", p.ID)
	b.addNode(rootID, p.Name, "Project", p.LifecycleStatus, lifecycleStateLevel(p.LifecycleStatus))

	groups, _ := s.serverGroupRepo.ListByProject(ctx, projectID)
	for _, g := range groups {
		gid := appNodeID("server_group", g.ID)
		state := "enabled"
		level := "normal"
		if g.Status != model.StatusEnabled {
			state = "disabled"
			level = "abnormal"
		}
		b.addNode(gid, g.Name, "ServerGroup", state, level)
		b.addEdge(rootID, gid, "contains")
	}

	servers, _, err := s.serverRepo.List(ctx, repository.ServerListParams{ProjectID: projectID, Page: 1, PageSize: 500})
	if err != nil {
		return nil, bizerrors.Pass(ctx, "project", "ApplicationTopology.servers", err)
	}
	for _, sv := range servers {
		sid := appNodeID("server", sv.ID)
		state := sv.Host
		level := "normal"
		if sv.Status != model.StatusEnabled {
			level = "abnormal"
			state = "disabled"
		}
		b.addNode(sid, sv.Name, "Server", state, level)
		if sv.GroupID != nil && *sv.GroupID > 0 {
			b.addEdge(appNodeID("server_group", *sv.GroupID), sid, "hosts")
		} else {
			b.addEdge(rootID, sid, "hosts")
		}
	}

	services, _, err := s.serviceRepo.List(ctx, repository.ServiceListParams{ProjectID: projectID, Page: 1, PageSize: 1000})
	if err != nil {
		return nil, bizerrors.Pass(ctx, "project", "ApplicationTopology.services", err)
	}
	for _, svc := range services {
		svcID := appNodeID("service", svc.ID)
		env := ""
		if svc.Env != nil {
			env = *svc.Env
		}
		level := "normal"
		if svc.Status != model.StatusEnabled {
			level = "abnormal"
		}
		b.addNode(svcID, svc.Name, "AppService", env, level)
		b.addEdge(appNodeID("server", svc.ServerID), svcID, "runs")
	}

	logs, _, err := s.logRepo.List(ctx, repository.LogSourceListParams{ProjectID: projectID, Page: 1, PageSize: 2000})
	if err != nil {
		return nil, bizerrors.Pass(ctx, "project", "ApplicationTopology.logs", err)
	}
	for _, ls := range logs {
		lid := appNodeID("log_source", ls.ID)
		level := "normal"
		if ls.Status != model.StatusEnabled {
			level = "abnormal"
		}
		b.addNode(lid, ls.Path, "LogSource", ls.LogType, level)
		b.addEdge(appNodeID("service", ls.ServiceID), lid, "observes")
	}

	return b.build(), nil
}
