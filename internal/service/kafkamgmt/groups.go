package kafkamgmt

import (
	"context"
	"sort"
	"strings"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"

	"github.com/segmentio/kafka-go"
)

// GroupInfo 消费组概要。
type GroupInfo struct {
	GroupID  string `json:"group_id"`
	State    string `json:"state,omitempty"`
	Protocol string `json:"protocol,omitempty"`
}

// GroupMember 消费组成员。
type GroupMember struct {
	MemberID   string   `json:"member_id"`
	ClientID   string   `json:"client_id"`
	ClientHost string   `json:"client_host"`
	Topics     []string `json:"topics,omitempty"`
}

// GroupLagPartition 分区 lag。
type GroupLagPartition struct {
	Topic     string `json:"topic"`
	Partition int    `json:"partition"`
	Committed int64  `json:"committed"`
	End       int64  `json:"end"`
	Lag       int64  `json:"lag"`
}

// GroupLag 消费组 lag。
type GroupLag struct {
	GroupID    string              `json:"group_id"`
	LagTotal   int64               `json:"lag_total"`
	Partitions []GroupLagPartition `json:"partitions"`
}

// DeleteGroupRequest 删除消费组。
type DeleteGroupRequest struct {
	ConnectionID uint   `json:"connection_id"`
	GroupID      string `json:"group_id"`
}

func (s *Service) ListGroups(ctx context.Context, connectionID uint) ([]GroupInfo, error) {
	ep, err := s.resolveEndpoint(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	client, err := ep.client()
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}
	resp, err := client.ListGroups(ctx, &kafka.ListGroupsRequest{})
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("列出消费组失败: " + err.Error())
	}
	out := make([]GroupInfo, 0)
	if resp != nil {
		for _, g := range resp.Groups {
			id := strings.TrimSpace(g.GroupID)
			if id == "" {
				continue
			}
			out = append(out, GroupInfo{GroupID: id})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].GroupID < out[j].GroupID })
	return out, nil
}

func (s *Service) GetGroupMembers(ctx context.Context, connectionID uint, groupID string) ([]GroupMember, error) {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return nil, constants.ErrBadRequestWithMsg("group_id 不能为空")
	}
	ep, err := s.resolveEndpoint(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	client, err := ep.client()
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}
	resp, err := client.DescribeGroups(ctx, &kafka.DescribeGroupsRequest{GroupIDs: []string{groupID}})
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("描述消费组失败: " + err.Error())
	}
	out := make([]GroupMember, 0)
	if resp == nil {
		return out, nil
	}
	for _, g := range resp.Groups {
		if g.Error != nil {
			return nil, constants.ErrBadRequestWithMsg(g.Error.Error())
		}
		for _, m := range g.Members {
			topics := make([]string, 0)
			for _, t := range m.MemberAssignments.Topics {
				topics = append(topics, t.Topic)
			}
			sort.Strings(topics)
			out = append(out, GroupMember{
				MemberID:   m.MemberID,
				ClientID:   m.ClientID,
				ClientHost: m.ClientHost,
				Topics:     topics,
			})
		}
	}
	return out, nil
}

func (s *Service) GetGroupLag(ctx context.Context, connectionID uint, groupID, topicFilter string) (*GroupLag, error) {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return nil, constants.ErrBadRequestWithMsg("group_id 不能为空")
	}
	ep, err := s.resolveEndpoint(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	client, err := ep.client()
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}

	topicFilter = strings.TrimSpace(topicFilter)
	topics := []string{}
	if topicFilter != "" {
		topics = []string{topicFilter}
	} else if members, merr := s.GetGroupMembers(ctx, connectionID, groupID); merr == nil {
		seen := map[string]struct{}{}
		for _, m := range members {
			for _, t := range m.Topics {
				seen[t] = struct{}{}
			}
		}
		for t := range seen {
			topics = append(topics, t)
		}
		sort.Strings(topics)
	}

	out := &GroupLag{GroupID: groupID, Partitions: make([]GroupLagPartition, 0)}
	if len(topics) == 0 {
		return out, nil
	}

	topicParts := map[string][]int{}
	for _, topic := range topics {
		if strings.HasPrefix(topic, "__") {
			continue
		}
		conn, derr := ep.dialLeader(ctx)
		if derr != nil {
			return nil, constants.ErrBadRequestWithMsg("连接 Kafka 失败: " + derr.Error())
		}
		ps, perr := conn.ReadPartitions(topic)
		_ = conn.Close()
		if perr != nil {
			continue
		}
		ids := make([]int, 0, len(ps))
		for _, p := range ps {
			ids = append(ids, p.ID)
		}
		topicParts[topic] = ids
	}
	if len(topicParts) == 0 {
		return out, nil
	}

	offResp, err := client.OffsetFetch(ctx, &kafka.OffsetFetchRequest{
		GroupID: groupID,
		Topics:  topicParts,
	})
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("获取消费位移失败: " + err.Error())
	}

	dialer, err := ep.dialer()
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}

	if offResp != nil {
		for topic, partitions := range offResp.Topics {
			for _, p := range partitions {
				if p.Error != nil {
					continue
				}
				end := int64(0)
				pc, derr := dialer.DialLeader(ctx, "tcp", ep.brokers[0], topic, p.Partition)
				if derr == nil {
					if last, oerr := pc.ReadLastOffset(); oerr == nil {
						end = last
					}
					_ = pc.Close()
				}
				committed := p.CommittedOffset
				lag := end - committed
				if lag < 0 || committed < 0 {
					lag = 0
				}
				out.Partitions = append(out.Partitions, GroupLagPartition{
					Topic:     topic,
					Partition: p.Partition,
					Committed: committed,
					End:       end,
					Lag:       lag,
				})
				out.LagTotal += lag
			}
		}
	}
	sort.Slice(out.Partitions, func(i, j int) bool {
		if out.Partitions[i].Topic == out.Partitions[j].Topic {
			return out.Partitions[i].Partition < out.Partitions[j].Partition
		}
		return out.Partitions[i].Topic < out.Partitions[j].Topic
	})
	return out, nil
}

func (s *Service) DeleteGroup(ctx context.Context, req DeleteGroupRequest, actor *auth.CurrentUser) error {
	if err := s.assertConnectionWrite(ctx, req.ConnectionID, actor); err != nil {
		return err
	}
	groupID := strings.TrimSpace(req.GroupID)
	if groupID == "" {
		return constants.ErrBadRequestWithMsg("group_id 不能为空")
	}
	if groupID == "yunshu-log-es" || strings.HasPrefix(groupID, "yunshu-kafkamgmt-sample-") {
		return constants.ErrBadRequestWithMsg("禁止删除平台保留消费组: " + groupID)
	}
	ep, err := s.resolveEndpoint(ctx, req.ConnectionID)
	if err != nil {
		return err
	}
	client, err := ep.client()
	if err != nil {
		return constants.ErrBadRequestWithMsg(err.Error())
	}
	resp, err := client.DeleteGroups(ctx, &kafka.DeleteGroupsRequest{GroupIDs: []string{groupID}})
	if err != nil {
		return constants.ErrBadRequestWithMsg("删除消费组失败: " + err.Error())
	}
	if resp != nil {
		if terr, ok := resp.Errors[groupID]; ok && terr != nil {
			return constants.ErrBadRequestWithMsg(terr.Error())
		}
	}
	return nil
}
