package kafkamgmt

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"

	"github.com/segmentio/kafka-go"
)

// TopicInfo Topic 概要。
type TopicInfo struct {
	Name              string `json:"name"`
	Partitions        int    `json:"partitions"`
	ReplicationFactor int    `json:"replication_factor"`
	Internal          bool   `json:"internal"`
}

// PartitionInfo 分区详情。
type PartitionInfo struct {
	ID       int   `json:"id"`
	Leader   int   `json:"leader"`
	Replicas []int `json:"replicas"`
	Isr      []int `json:"isr"`
}

// TopicDetail Topic 详情。
type TopicDetail struct {
	Name       string          `json:"name"`
	Partitions []PartitionInfo `json:"partitions"`
}

// BrokerInfo Broker 节点。
type BrokerInfo struct {
	ID   int    `json:"id"`
	Host string `json:"host"`
	Port int    `json:"port"`
	Rack string `json:"rack,omitempty"`
}

// CreateTopicRequest 创建 Topic。
type CreateTopicRequest struct {
	ConnectionID      uint   `json:"connection_id"`
	Name              string `json:"name"`
	NumPartitions     int    `json:"num_partitions"`
	ReplicationFactor int    `json:"replication_factor"`
}

// DeleteTopicRequest 删除 Topic。
type DeleteTopicRequest struct {
	ConnectionID uint   `json:"connection_id"`
	Topic        string `json:"topic"`
}

// CreatePartitionsRequest 扩容分区（目标总分区数）。
type CreatePartitionsRequest struct {
	ConnectionID uint   `json:"connection_id"`
	Topic        string `json:"topic"`
	TotalCount   int    `json:"total_count"`
}

// ProduceRequest 受控生产。
type ProduceRequest struct {
	ConnectionID uint   `json:"connection_id"`
	Topic        string `json:"topic"`
	Key          string `json:"key"`
	Value        string `json:"value"`
	Partition    *int   `json:"partition"`
}

// ConsumeRequest 抽样消费。
type ConsumeRequest struct {
	ConnectionID uint   `json:"connection_id"`
	Topic        string `json:"topic"`
	MaxMessages  int    `json:"max_messages"`
	From         string `json:"from"` // latest|earliest
	TimeoutMs    int    `json:"timeout_ms"`
	Partition    *int   `json:"partition"`
}

// ConsumedMessage 抽样消息。
type ConsumedMessage struct {
	Partition int    `json:"partition"`
	Offset    int64  `json:"offset"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	Time      string `json:"time,omitempty"`
}

const maxProduceBytes = 100 * 1024
const maxConsumeMessages = 50

func (s *Service) ListBrokers(ctx context.Context, connectionID uint) ([]BrokerInfo, error) {
	ep, err := s.resolveEndpoint(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	conn, err := ep.dialLeader(ctx)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("连接 Kafka 失败: " + err.Error())
	}
	defer conn.Close()
	brokers, err := conn.Brokers()
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("获取 Broker 失败: " + err.Error())
	}
	out := make([]BrokerInfo, 0, len(brokers))
	for _, b := range brokers {
		out = append(out, BrokerInfo{
			ID:   b.ID,
			Host: b.Host,
			Port: b.Port,
			Rack: b.Rack,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *Service) ListTopics(ctx context.Context, connectionID uint) ([]TopicInfo, error) {
	ep, err := s.resolveEndpoint(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	conn, err := ep.dialLeader(ctx)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("连接 Kafka 失败: " + err.Error())
	}
	defer conn.Close()
	parts, err := conn.ReadPartitions()
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("列出 Topic 失败: " + err.Error())
	}
	type agg struct {
		parts int
		rf    int
	}
	m := map[string]*agg{}
	for _, p := range parts {
		name := strings.TrimSpace(p.Topic)
		if name == "" {
			continue
		}
		a := m[name]
		if a == nil {
			a = &agg{}
			m[name] = a
		}
		a.parts++
		if rf := len(p.Replicas); rf > a.rf {
			a.rf = rf
		}
	}
	out := make([]TopicInfo, 0, len(m))
	for name, a := range m {
		out = append(out, TopicInfo{
			Name:              name,
			Partitions:        a.parts,
			ReplicationFactor: a.rf,
			Internal:          strings.HasPrefix(name, "__"),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *Service) GetTopicDetail(ctx context.Context, connectionID uint, topic string) (*TopicDetail, error) {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return nil, constants.ErrBadRequestWithMsg("topic 不能为空")
	}
	ep, err := s.resolveEndpoint(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	conn, err := ep.dialLeader(ctx)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("连接 Kafka 失败: " + err.Error())
	}
	defer conn.Close()
	parts, err := conn.ReadPartitions(topic)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("读取分区失败: " + err.Error())
	}
	out := &TopicDetail{Name: topic, Partitions: make([]PartitionInfo, 0, len(parts))}
	for _, p := range parts {
		if p.Topic != topic {
			continue
		}
		replicas := make([]int, 0, len(p.Replicas))
		for _, r := range p.Replicas {
			replicas = append(replicas, r.ID)
		}
		isr := make([]int, 0, len(p.Isr))
		for _, r := range p.Isr {
			isr = append(isr, r.ID)
		}
		out.Partitions = append(out.Partitions, PartitionInfo{
			ID:       p.ID,
			Leader:   p.Leader.ID,
			Replicas: replicas,
			Isr:      isr,
		})
	}
	sort.Slice(out.Partitions, func(i, j int) bool { return out.Partitions[i].ID < out.Partitions[j].ID })
	return out, nil
}

func (s *Service) GetTopicConfig(ctx context.Context, connectionID uint, topic string) (map[string]string, error) {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return nil, constants.ErrBadRequestWithMsg("topic 不能为空")
	}
	ep, err := s.resolveEndpoint(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	client, err := ep.client()
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}
	resp, err := client.DescribeConfigs(ctx, &kafka.DescribeConfigsRequest{
		Resources: []kafka.DescribeConfigRequestResource{{
			ResourceType: kafka.ResourceTypeTopic,
			ResourceName: topic,
		}},
	})
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("读取 Topic 配置失败: " + err.Error())
	}
	out := map[string]string{}
	if resp == nil {
		return out, nil
	}
	for _, res := range resp.Resources {
		if res.Error != nil {
			return nil, constants.ErrBadRequestWithMsg(res.Error.Error())
		}
		for _, e := range res.ConfigEntries {
			out[e.ConfigName] = e.ConfigValue
		}
	}
	return out, nil
}

func (s *Service) CreateTopic(ctx context.Context, req CreateTopicRequest, actor *auth.CurrentUser) error {
	if err := s.assertConnectionWrite(ctx, req.ConnectionID, actor); err != nil {
		return err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return constants.ErrBadRequestWithMsg("topic 名称不能为空")
	}
	if strings.HasPrefix(name, "__") {
		return constants.ErrBadRequestWithMsg("禁止创建内部 Topic")
	}
	parts := req.NumPartitions
	if parts <= 0 {
		parts = 1
	}
	if parts > 128 {
		return constants.ErrBadRequestWithMsg("分区数过大（最多 128）")
	}
	rf := req.ReplicationFactor
	if rf <= 0 {
		rf = 1
	}
	ep, err := s.resolveEndpoint(ctx, req.ConnectionID)
	if err != nil {
		return err
	}
	client, err := ep.client()
	if err != nil {
		return constants.ErrBadRequestWithMsg(err.Error())
	}
	resp, err := client.CreateTopics(ctx, &kafka.CreateTopicsRequest{
		Topics: []kafka.TopicConfig{{
			Topic:             name,
			NumPartitions:     parts,
			ReplicationFactor: rf,
		}},
	})
	if err != nil {
		return constants.ErrBadRequestWithMsg("创建 Topic 失败: " + err.Error())
	}
	if resp != nil {
		if terr, ok := resp.Errors[name]; ok && terr != nil {
			msg := strings.ToLower(terr.Error())
			if terr == kafka.TopicAlreadyExists || strings.Contains(msg, "already exists") {
				return constants.ErrBadRequestWithMsg("Topic 已存在")
			}
			return constants.ErrBadRequestWithMsg(terr.Error())
		}
	}
	return nil
}

func (s *Service) DeleteTopic(ctx context.Context, req DeleteTopicRequest, actor *auth.CurrentUser) error {
	if err := s.assertConnectionWrite(ctx, req.ConnectionID, actor); err != nil {
		return err
	}
	topic := strings.TrimSpace(req.Topic)
	if topic == "" {
		return constants.ErrBadRequestWithMsg("topic 不能为空")
	}
	if strings.HasPrefix(topic, "__") {
		return constants.ErrBadRequestWithMsg("禁止删除内部 Topic")
	}
	ep, err := s.resolveEndpoint(ctx, req.ConnectionID)
	if err != nil {
		return err
	}
	client, err := ep.client()
	if err != nil {
		return constants.ErrBadRequestWithMsg(err.Error())
	}
	resp, err := client.DeleteTopics(ctx, &kafka.DeleteTopicsRequest{Topics: []string{topic}})
	if err != nil {
		return constants.ErrBadRequestWithMsg("删除 Topic 失败: " + err.Error())
	}
	if resp != nil {
		if terr, ok := resp.Errors[topic]; ok && terr != nil {
			msg := strings.ToLower(terr.Error())
			if strings.Contains(msg, "unknown topic") || strings.Contains(msg, "does not exist") {
				return nil
			}
			return constants.ErrBadRequestWithMsg(terr.Error())
		}
	}
	return nil
}

func (s *Service) CreatePartitions(ctx context.Context, req CreatePartitionsRequest, actor *auth.CurrentUser) error {
	if err := s.assertConnectionWrite(ctx, req.ConnectionID, actor); err != nil {
		return err
	}
	topic := strings.TrimSpace(req.Topic)
	if topic == "" {
		return constants.ErrBadRequestWithMsg("topic 不能为空")
	}
	if req.TotalCount <= 0 {
		return constants.ErrBadRequestWithMsg("total_count 必须大于 0")
	}
	if req.TotalCount > 256 {
		return constants.ErrBadRequestWithMsg("分区数过大（最多 256）")
	}
	ep, err := s.resolveEndpoint(ctx, req.ConnectionID)
	if err != nil {
		return err
	}
	client, err := ep.client()
	if err != nil {
		return constants.ErrBadRequestWithMsg(err.Error())
	}
	resp, err := client.CreatePartitions(ctx, &kafka.CreatePartitionsRequest{
		Topics: []kafka.TopicPartitionsConfig{{
			Name:  topic,
			Count: int32(req.TotalCount),
		}},
	})
	if err != nil {
		return constants.ErrBadRequestWithMsg("扩容分区失败: " + err.Error())
	}
	if resp != nil {
		if terr, ok := resp.Errors[topic]; ok && terr != nil {
			return constants.ErrBadRequestWithMsg(terr.Error())
		}
	}
	return nil
}

func (s *Service) Produce(ctx context.Context, req ProduceRequest, actor *auth.CurrentUser) error {
	if err := s.assertConnectionWrite(ctx, req.ConnectionID, actor); err != nil {
		return err
	}
	topic := strings.TrimSpace(req.Topic)
	if topic == "" {
		return constants.ErrBadRequestWithMsg("topic 不能为空")
	}
	if len(req.Value) == 0 {
		return constants.ErrBadRequestWithMsg("消息内容不能为空")
	}
	if len(req.Value) > maxProduceBytes {
		return constants.ErrBadRequestWithMsg(fmt.Sprintf("消息过大（最多 %d 字节）", maxProduceBytes))
	}
	ep, err := s.resolveEndpoint(ctx, req.ConnectionID)
	if err != nil {
		return err
	}
	dialer, err := ep.dialer()
	if err != nil {
		return constants.ErrBadRequestWithMsg(err.Error())
	}
	w := &kafka.Writer{
		Addr:         kafka.TCP(ep.brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        false,
		Transport:    &kafka.Transport{SASL: dialer.SASLMechanism},
	}
	defer w.Close()
	msg := kafka.Message{
		Key:   []byte(req.Key),
		Value: []byte(req.Value),
		Time:  time.Now(),
	}
	if req.Partition != nil {
		// 指定分区：直连该分区 leader
		conn, derr := dialer.DialLeader(ctx, "tcp", ep.brokers[0], topic, *req.Partition)
		if derr != nil {
			return constants.ErrBadRequestWithMsg("连接分区失败: " + derr.Error())
		}
		defer conn.Close()
		_, err := conn.WriteMessages(msg)
		if err != nil {
			return constants.ErrBadRequestWithMsg("生产失败: " + err.Error())
		}
		return nil
	}
	if err := w.WriteMessages(ctx, msg); err != nil {
		return constants.ErrBadRequestWithMsg("生产失败: " + err.Error())
	}
	return nil
}

func (s *Service) Consume(ctx context.Context, req ConsumeRequest) ([]ConsumedMessage, error) {
	topic := strings.TrimSpace(req.Topic)
	if topic == "" {
		return nil, constants.ErrBadRequestWithMsg("topic 不能为空")
	}
	maxN := req.MaxMessages
	if maxN <= 0 {
		maxN = 10
	}
	if maxN > maxConsumeMessages {
		maxN = maxConsumeMessages
	}
	timeoutMs := req.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = 3000
	}
	if timeoutMs > 15000 {
		timeoutMs = 15000
	}
	from := strings.ToLower(strings.TrimSpace(req.From))
	startOffset := kafka.LastOffset
	if from == "earliest" {
		startOffset = kafka.FirstOffset
	}
	ep, err := s.resolveEndpoint(ctx, req.ConnectionID)
	if err != nil {
		return nil, err
	}
	dialer, err := ep.dialer()
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}
	cfg := kafka.ReaderConfig{
		Brokers:     ep.brokers,
		Topic:       topic,
		Dialer:      dialer,
		MinBytes:    1,
		MaxBytes:    1e6,
		MaxWait:     time.Duration(timeoutMs) * time.Millisecond,
		StartOffset: startOffset,
	}
	if req.Partition != nil {
		cfg.Partition = *req.Partition
	} else {
		// 无固定 partition 时用临时 group，避免影响业务消费组
		cfg.GroupID = fmt.Sprintf("yunshu-kafkamgmt-sample-%d", time.Now().UnixNano())
		cfg.StartOffset = startOffset
	}
	r := kafka.NewReader(cfg)
	defer r.Close()

	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	out := make([]ConsumedMessage, 0, maxN)
	for len(out) < maxN {
		remain := time.Until(deadline)
		if remain <= 0 {
			break
		}
		rctx, cancel := context.WithTimeout(ctx, remain)
		msg, err := r.ReadMessage(rctx)
		cancel()
		if err != nil {
			break
		}
		item := ConsumedMessage{
			Partition: msg.Partition,
			Offset:    msg.Offset,
			Key:       string(msg.Key),
			Value:     string(msg.Value),
		}
		if !msg.Time.IsZero() {
			item.Time = msg.Time.UTC().Format(time.RFC3339)
		}
		out = append(out, item)
	}
	return out, nil
}
