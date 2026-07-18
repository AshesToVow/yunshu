package logplatform

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"yunshu/internal/config"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/esclient"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
)

// KafkaToESService 以消费者组订阅各 Agent Topic，bulk 写入 Elasticsearch。
type KafkaToESService struct {
	kafka *KafkaProvider
	es    *ElasticsearchProvider

	mu             sync.Mutex
	cancel         context.CancelFunc
	running        bool
	cfgFingerprint string

	consumedTotal atomic.Int64
	writtenTotal  atomic.Int64
	errorTotal    atomic.Int64
	lastConsumeAt atomic.Int64
	lastError     atomic.Value
	runningFlag   atomic.Bool
	activeTopics  atomic.Value // []string
}

func NewKafkaToESService(kafka *KafkaProvider, es *ElasticsearchProvider) *KafkaToESService {
	s := &KafkaToESService{kafka: kafka, es: es}
	s.lastError.Store("")
	s.activeTopics.Store([]string(nil))
	return s
}

type KafkaPartitionLag struct {
	Topic          string `json:"topic"`
	Partition      int    `json:"partition"`
	HighWaterMark  int64  `json:"high_water_mark"`
	ConsumerOffset int64  `json:"consumer_offset"`
	Lag            int64  `json:"lag"`
}

type KafkaQueueStats struct {
	Enabled         bool                `json:"enabled"`
	SinkViaKafka    bool                `json:"sink_via_kafka"`
	Brokers         []string            `json:"brokers"`
	TopicPrefix     string              `json:"topic_prefix"`
	Topics          []string            `json:"topics,omitempty"`
	ConsumerGroup   string              `json:"consumer_group"`
	ConsumerRunning bool                `json:"consumer_running"`
	LagTotal        int64               `json:"lag_total"`
	Partitions      []KafkaPartitionLag `json:"partitions,omitempty"`
	ConsumedTotal   int64               `json:"consumed_total"`
	WrittenTotal    int64               `json:"written_total"`
	ErrorTotal      int64               `json:"error_total"`
	LastConsumeAt   string              `json:"last_consume_at,omitempty"`
	LastError       string              `json:"last_error,omitempty"`
	Message         string              `json:"message,omitempty"`
	HasSASL         bool                `json:"has_sasl"`
}

type KafkaConfigPreviewItem struct {
	Enabled         bool     `json:"enabled"`
	Brokers         []string `json:"brokers"`
	TopicPrefix     string   `json:"topic_prefix"`
	TopicExample    string   `json:"topic_example"`
	ConsumerGroup   string   `json:"consumer_group"`
	Username        string   `json:"username,omitempty"`
	HasPassword     bool     `json:"has_password"`
	SASLMechanism   string   `json:"sasl_mechanism,omitempty"`
	BatchSize       int      `json:"batch_size"`
	TopicPartitions int      `json:"topic_partitions"`
	Workers         int      `json:"workers"`
	SinkViaKafka    bool     `json:"sink_via_kafka"`
}

func (s *KafkaToESService) Run(ctx context.Context) {
	if s == nil {
		return
	}
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	s.reconcile(ctx)
	for {
		select {
		case <-ctx.Done():
			s.stopConsumer()
			return
		case <-ticker.C:
			s.reconcile(ctx)
		}
	}
}

func (s *KafkaToESService) reconcile(parent context.Context) {
	if s.kafka == nil {
		s.stopConsumer()
		return
	}
	cfg, err := s.kafka.Resolve(parent)
	if err != nil {
		s.setLastError(err.Error())
		return
	}
	cfg = cfg.Normalized()
	if !cfg.SinkViaKafka() {
		s.stopConsumer()
		s.mu.Lock()
		s.cfgFingerprint = kafkaFingerprint(cfg, nil)
		s.mu.Unlock()
		return
	}
	if s.es == nil {
		s.setLastError("elasticsearch provider nil; cannot consume kafka to es")
		s.stopConsumer()
		return
	}
	esCfg, err := s.es.Resolve(parent)
	if err != nil || !esCfg.Enabled {
		s.setLastError("elasticsearch disabled; kafka consumer paused")
		s.stopConsumer()
		return
	}

	topics, err := listAgentKafkaTopics(parent, cfg)
	if err != nil {
		s.setLastError("list topics: " + err.Error())
		// 仍尝试按已有 fingerprint 继续；无 topic 则停
	}
	fp := kafkaFingerprint(cfg, topics)
	s.mu.Lock()
	same := s.running && s.cfgFingerprint == fp
	s.mu.Unlock()
	if same {
		return
	}
	s.stopConsumer()
	if len(topics) == 0 {
		s.setLastError("暂无 Agent Topic（yunshu-agent-{ip}-YYYY.MM.DD），请先引导/热更 Agent")
		s.activeTopics.Store([]string(nil))
		s.mu.Lock()
		s.cfgFingerprint = fp
		s.mu.Unlock()
		return
	}
	s.setLastError("")
	s.startConsumer(parent, cfg, topics)
}

func (s *KafkaToESService) startConsumer(parent context.Context, cfg config.KafkaConfig, topics []string) {
	cfg = cfg.Normalized()
	runCtx, cancel := context.WithCancel(parent)
	s.mu.Lock()
	s.cancel = cancel
	s.running = true
	s.cfgFingerprint = kafkaFingerprint(cfg, topics)
	s.mu.Unlock()
	s.runningFlag.Store(true)
	s.activeTopics.Store(append([]string(nil), topics...))
	s.setLastError("")
	slog.Default().With("component", "kafka-to-es").Info("kafka consumer starting",
		"group", cfg.ConsumerGroup, "topics", len(topics), "brokers", len(cfg.Brokers))

	go func() {
		defer func() {
			s.runningFlag.Store(false)
			s.mu.Lock()
			s.running = false
			s.mu.Unlock()
		}()
		s.consumeLoop(runCtx, cfg, topics)
	}()
}

func (s *KafkaToESService) stopConsumer() {
	s.mu.Lock()
	cancel := s.cancel
	s.cancel = nil
	s.running = false
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	s.runningFlag.Store(false)
}

func (s *KafkaToESService) consumeLoop(ctx context.Context, cfg config.KafkaConfig, topics []string) {
	dialer, err := kafkaDialer(cfg)
	if err != nil {
		s.setLastError(err.Error())
		slog.Default().With("component", "kafka-to-es").Error("kafka dialer", "err", err)
		return
	}
	// 必须使用消费者组 + 多 Topic（每 Agent 一个）
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		GroupID:        cfg.ConsumerGroup,
		GroupTopics:    topics,
		MinBytes:       1,
		MaxBytes:       10e6,
		MaxWait:        time.Second,
		CommitInterval: 0,
		Dialer:         dialer,
		StartOffset:    kafka.LastOffset,
	})
	defer reader.Close()

	batchSize := cfg.BatchSize
	batch := make([]kafka.Message, 0, batchSize)
	flushInterval := 2 * time.Second
	lastFlush := time.Now()
	prefix := cfg.TopicPrefix

	flush := func() {
		if len(batch) == 0 {
			return
		}
		toWrite := append([]kafka.Message(nil), batch...)
		res, err := s.writeBatch(ctx, toWrite, prefix)
		if err != nil {
			// 传输级失败（ES 不可达 / 5xx）：不提交、不清空 batch，下轮重试形成背压。
			s.errorTotal.Add(1)
			s.setLastError(err.Error())
			slog.Default().With("component", "kafka-to-es").Warn("bulk write failed", "err", err, "n", len(toWrite))
			return
		}
		// 写入成功（可能含被 ES 拒绝的坏文档）：坏文档重试也无用，好文档已落库；
		// 必须提交 offset 并清空 batch，否则单条毒消息会让 batch 无限增长（OOM）并永久阻塞该消费组。
		if res != nil && res.Failed > 0 {
			s.errorTotal.Add(int64(res.Failed))
			s.setLastError(fmt.Sprintf("bulk 拒绝 %d 条文档（已跳过）：%s", res.Failed, res.FirstError))
			slog.Default().With("component", "kafka-to-es").Warn("bulk items rejected; skipping poison docs",
				"failed", res.Failed, "n", len(toWrite), "sample", res.FirstError)
		}
		if err := reader.CommitMessages(ctx, toWrite...); err != nil {
			// 提交失败：不清空 batch，下轮重写。因文档无固定 _id，重写好文档会产生少量重复，属可接受的 at-least-once。
			s.errorTotal.Add(1)
			s.setLastError("commit: " + err.Error())
			return
		}
		if written := len(toWrite) - failedCount(res); written > 0 {
			s.writtenTotal.Add(int64(written))
		}
		batch = batch[:0]
		lastFlush = time.Now()
	}

	for {
		if ctx.Err() != nil {
			flush()
			return
		}
		if time.Since(lastFlush) >= flushInterval {
			flush()
		}
		msgCtx, cancel := context.WithTimeout(ctx, time.Second)
		msg, err := reader.FetchMessage(msgCtx)
		cancel()
		if err != nil {
			if ctx.Err() != nil {
				flush()
				return
			}
			continue
		}
		s.consumedTotal.Add(1)
		s.lastConsumeAt.Store(time.Now().UnixMilli())
		batch = append(batch, msg)
		if len(batch) >= batchSize {
			flush()
		}
	}
}

func (s *KafkaToESService) writeBatch(ctx context.Context, msgs []kafka.Message, topicPrefix string) (*esclient.BulkResult, error) {
	cli, _, err := s.es.Client(ctx)
	if err != nil {
		return nil, err
	}
	var ndjson strings.Builder
	for _, m := range msgs {
		doc, index, err := parseKafkaLogMessage(m.Value, m.Topic, topicPrefix)
		if err != nil {
			s.errorTotal.Add(1)
			continue
		}
		meta, _ := json.Marshal(map[string]any{"index": map[string]any{"_index": index}})
		body, _ := json.Marshal(doc)
		ndjson.Write(meta)
		ndjson.WriteByte('\n')
		ndjson.Write(body)
		ndjson.WriteByte('\n')
	}
	if ndjson.Len() == 0 {
		return &esclient.BulkResult{}, nil
	}
	return cli.Bulk(ctx, []byte(ndjson.String()))
}

func failedCount(res *esclient.BulkResult) int {
	if res == nil {
		return 0
	}
	return res.Failed
}

func parseKafkaLogMessage(raw []byte, topic, topicPrefix string) (map[string]any, string, error) {
	s := strings.TrimSpace(string(raw))
	if s == "" {
		return nil, "", fmt.Errorf("empty message")
	}
	var root map[string]any
	if err := json.Unmarshal([]byte(s), &root); err != nil {
		now := time.Now().UTC()
		host := resolveHostForIndex(nil, topic, topicPrefix, 0)
		return map[string]any{
			"@timestamp": now.Format(time.RFC3339Nano),
			"message":    s,
		}, resolveIndexName(host, 0, topic, topicPrefix, now), nil
	}

	doc := map[string]any{}
	for k, v := range root {
		doc[k] = v
	}
	if _, ok := doc["message"]; !ok {
		if b, ok := doc["body"]; ok {
			switch t := b.(type) {
			case string:
				doc["message"] = t
			default:
				doc["message"] = fmt.Sprint(t)
			}
		}
	}

	fields := map[string]any{}
	if f, ok := doc["fields"].(map[string]any); ok {
		fields = f
	}
	serverHost := strings.TrimSpace(fmt.Sprint(fields["server_host"]))
	if serverHost == "" || serverHost == "<nil>" {
		serverHost = strings.TrimSpace(fmt.Sprint(doc["server_host"]))
		if serverHost == "<nil>" {
			serverHost = ""
		}
	}
	serverID := parseUintAny(fields["server_id"])
	if serverID == 0 {
		serverID = parseUintAny(doc["server_id"])
	}
	if serverID == 0 {
		if id, ok := ParseServerIDFromAgentKafkaTopic(topic, topicPrefix); ok {
			serverID = id
		}
	}
	host := resolveHostForIndex(fields, topic, topicPrefix, serverID)
	if serverHost != "" {
		host = serverHost
	}
	ts := parseTimestampAny(doc["@timestamp"])
	if ts.IsZero() {
		ts = parseTimestampAny(doc["timestamp"])
	}
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	if _, ok := doc["@timestamp"]; !ok {
		doc["@timestamp"] = ts.Format(time.RFC3339Nano)
	}
	return doc, resolveIndexName(host, serverID, topic, topicPrefix, ts), nil
}

func resolveHostForIndex(fields map[string]any, topic, topicPrefix string, serverID uint) string {
	if fields != nil {
		if h := strings.TrimSpace(fmt.Sprint(fields["server_host"])); h != "" && h != "<nil>" {
			return h
		}
	}
	if key, ok := ParseHostKeyFromAgentName(topic, topicPrefix); ok {
		return key
	}
	if serverID > 0 {
		return fmt.Sprintf("server-%d", serverID)
	}
	return "unknown"
}

func resolveIndexName(host string, serverID uint, topic, topicPrefix string, ts time.Time) string {
	host = strings.TrimSpace(host)
	if host != "" && host != "unknown" && !strings.HasPrefix(host, "server-") {
		return AgentIndexForDay(host, ts)
	}
	if key, ok := ParseHostKeyFromAgentName(topic, topicPrefix); ok {
		return fmt.Sprintf("%s-%s-%s", defaultAgentIndexPrefix, key, ts.UTC().Format("2006.01.02"))
	}
	if serverID > 0 {
		return AgentIndexForDayByServerID(serverID, ts)
	}
	if id, ok := ParseServerIDFromAgentKafkaTopic(topic, topicPrefix); ok {
		return AgentIndexForDayByServerID(id, ts)
	}
	return AgentIndexForDay(host, ts)
}

func parseUintAny(v any) uint {
	switch t := v.(type) {
	case float64:
		if t > 0 {
			return uint(t)
		}
	case string:
		n, _ := strconv.ParseUint(strings.TrimSpace(t), 10, 64)
		return uint(n)
	case json.Number:
		n, _ := t.Int64()
		if n > 0 {
			return uint(n)
		}
	}
	return 0
}

func parseTimestampAny(v any) time.Time {
	switch t := v.(type) {
	case string:
		s := strings.TrimSpace(t)
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.000Z07:00", "2006-01-02 15:04:05"} {
			if ts, err := time.Parse(layout, s); err == nil {
				return ts
			}
		}
	case float64:
		if t > 1e12 {
			return time.UnixMilli(int64(t)).UTC()
		}
		if t > 1e9 {
			return time.Unix(int64(t), 0).UTC()
		}
	}
	return time.Time{}
}

func kafkaDialer(cfg config.KafkaConfig) (*kafka.Dialer, error) {
	d := &kafka.Dialer{Timeout: 10 * time.Second, DualStack: true}
	user := strings.TrimSpace(cfg.Username)
	if user == "" {
		return d, nil
	}
	mech := strings.ToLower(strings.TrimSpace(cfg.SASLMechanism))
	if mech == "" || mech == "none" {
		mech = "plain"
	}
	var mechanism sasl.Mechanism
	var err error
	switch mech {
	case "plain":
		mechanism = plain.Mechanism{Username: user, Password: cfg.Password}
	case "scram-sha-256":
		mechanism, err = scram.Mechanism(scram.SHA256, user, cfg.Password)
	case "scram-sha-512":
		mechanism, err = scram.Mechanism(scram.SHA512, user, cfg.Password)
	default:
		mechanism = plain.Mechanism{Username: user, Password: cfg.Password}
	}
	if err != nil {
		return nil, err
	}
	d.SASLMechanism = mechanism
	return d, nil
}

func kafkaFingerprint(cfg config.KafkaConfig, topics []string) string {
	cfg = cfg.Normalized()
	return fmt.Sprintf("%v|%s|%s|%s|%s|%d|%v|%s",
		cfg.Enabled, strings.Join(cfg.Brokers, ","), cfg.TopicPrefix, cfg.ConsumerGroup,
		cfg.Username, cfg.BatchSize, cfg.SASLMechanism, strings.Join(topics, ","))
}

func (s *KafkaToESService) setLastError(msg string) {
	s.lastError.Store(strings.TrimSpace(msg))
}

func (s *KafkaToESService) ConfigPreview(ctx context.Context) (*KafkaConfigPreviewItem, error) {
	if s == nil || s.kafka == nil {
		return &KafkaConfigPreviewItem{}, nil
	}
	cfg, err := s.kafka.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	cfg = cfg.Normalized()
	return &KafkaConfigPreviewItem{
		Enabled:         cfg.Enabled,
		Brokers:         cfg.Brokers,
		TopicPrefix:     cfg.TopicPrefix,
		TopicExample:    AgentKafkaTopicForDay("10.10.10.1", cfg.TopicPrefix, time.Now().UTC()),
		ConsumerGroup:   cfg.ConsumerGroup,
		Username:        cfg.Username,
		HasPassword:     strings.TrimSpace(cfg.Password) != "",
		SASLMechanism:   cfg.SASLMechanism,
		BatchSize:       cfg.BatchSize,
		TopicPartitions: cfg.TopicPartitions,
		Workers:         cfg.Workers,
		SinkViaKafka:    cfg.SinkViaKafka(),
	}, nil
}

func (s *KafkaToESService) Stats(ctx context.Context) (*KafkaQueueStats, error) {
	out := &KafkaQueueStats{
		ConsumedTotal:   s.consumedTotal.Load(),
		WrittenTotal:    s.writtenTotal.Load(),
		ErrorTotal:      s.errorTotal.Load(),
		ConsumerRunning: s.runningFlag.Load(),
	}
	if ms := s.lastConsumeAt.Load(); ms > 0 {
		out.LastConsumeAt = time.UnixMilli(ms).Format(time.RFC3339)
	}
	if v, ok := s.lastError.Load().(string); ok {
		out.LastError = v
	}
	if topics, ok := s.activeTopics.Load().([]string); ok {
		out.Topics = append([]string(nil), topics...)
	}
	if s == nil || s.kafka == nil {
		out.Message = "Kafka 未配置"
		return out, nil
	}
	cfg, err := s.kafka.Resolve(ctx)
	if err != nil {
		out.Message = err.Error()
		return out, nil
	}
	cfg = cfg.Normalized()
	out.Enabled = cfg.Enabled
	out.SinkViaKafka = cfg.SinkViaKafka()
	out.Brokers = cfg.Brokers
	out.TopicPrefix = cfg.TopicPrefix
	out.ConsumerGroup = cfg.ConsumerGroup
	out.HasSASL = strings.TrimSpace(cfg.Username) != ""

	if !cfg.SinkViaKafka() {
		out.Message = "Kafka 中转未启用，Loggie 直写 Elasticsearch"
		return out, nil
	}

	// 始终以 broker 实况为准，避免删除后仍展示内存中的 activeTopics
	topics := out.Topics
	if listed, err := listAgentKafkaTopics(ctx, cfg); err == nil {
		topics = listed
		out.Topics = listed
		s.activeTopics.Store(append([]string(nil), listed...))
	}

	lags, lagTotal, lagErr := fetchKafkaLagMulti(ctx, cfg, topics)
	if lagErr != nil {
		out.Message = lagErr.Error()
		if out.LastError == "" {
			out.LastError = lagErr.Error()
		}
		return out, nil
	}
	out.Partitions = lags
	out.LagTotal = lagTotal
	if out.ConsumerRunning {
		out.Message = fmt.Sprintf("消费组 %s 运行中，订阅 %d 个 Agent Topic", cfg.ConsumerGroup, len(topics))
		if strings.Contains(out.LastError, "暂无 Agent Topic") {
			out.LastError = ""
		}
	} else if len(topics) == 0 {
		out.Message = "暂无 Agent Topic，请先引导/热更 Agent（将自动创建 yunshu-agent-{ip}-YYYY.MM.DD）"
	} else {
		out.Message = "消费者未运行（请确认 ES 已启用）"
		if strings.Contains(out.LastError, "暂无 Agent Topic") {
			out.LastError = ""
		}
	}
	return out, nil
}

// DeleteTopic 删除 Agent Topic。
func (s *KafkaToESService) DeleteTopic(ctx context.Context, topic string) error {
	if s == nil || s.kafka == nil {
		return constants.ErrBadRequestWithMsg("Kafka 未配置")
	}
	cfg, err := s.kafka.Resolve(ctx)
	if err != nil {
		return err
	}
	if !cfg.SinkViaKafka() {
		return constants.ErrBadRequestWithMsg("Kafka 中转未启用")
	}
	topic = strings.TrimSpace(topic)
	if err := DeleteAgentKafkaTopic(ctx, cfg, topic); err != nil {
		return constants.ErrBadRequestWithMsg(err.Error())
	}
	// 立即从内存订阅列表移除，并强制下次 reconcile 重建消费者
	if cur, ok := s.activeTopics.Load().([]string); ok {
		next := make([]string, 0, len(cur))
		for _, t := range cur {
			if t != topic {
				next = append(next, t)
			}
		}
		s.activeTopics.Store(next)
	}
	s.mu.Lock()
	s.cfgFingerprint = ""
	s.mu.Unlock()
	s.kafka.InvalidateCache()
	return nil
}

func fetchKafkaLagMulti(ctx context.Context, cfg config.KafkaConfig, topics []string) ([]KafkaPartitionLag, int64, error) {
	if len(topics) == 0 {
		return nil, 0, nil
	}
	var all []KafkaPartitionLag
	var total int64
	for _, topic := range topics {
		lags, err := fetchTopicLag(ctx, cfg, topic)
		if err != nil {
			continue
		}
		all = append(all, lags...)
		total += sumLag(lags)
	}
	return all, total, nil
}

func fetchTopicLag(ctx context.Context, cfg config.KafkaConfig, topic string) ([]KafkaPartitionLag, error) {
	dialer, err := kafkaDialer(cfg)
	if err != nil {
		return nil, err
	}
	conn, err := dialer.DialContext(ctx, "tcp", cfg.Brokers[0])
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	parts, err := conn.ReadPartitions(topic)
	if err != nil {
		return nil, err
	}
	base := make([]KafkaPartitionLag, 0, len(parts))
	seen := map[int]struct{}{}
	for _, p := range parts {
		if p.Topic != topic {
			continue
		}
		if _, ok := seen[p.ID]; ok {
			continue
		}
		seen[p.ID] = struct{}{}
		pc, err := dialer.DialLeader(ctx, "tcp", cfg.Brokers[0], topic, p.ID)
		if err != nil {
			continue
		}
		hw, err := pc.ReadLastOffset()
		_ = pc.Close()
		if err != nil {
			continue
		}
		base = append(base, KafkaPartitionLag{
			Topic:          topic,
			Partition:      p.ID,
			HighWaterMark:  hw,
			ConsumerOffset: -1,
			Lag:            -1,
		})
	}
	if lags, err := fetchGroupOffsets(ctx, dialer, cfg, topic, base); err == nil {
		return lags, nil
	}
	return base, nil
}

func sumLag(parts []KafkaPartitionLag) int64 {
	var n int64
	for _, p := range parts {
		if p.Lag > 0 {
			n += p.Lag
		}
	}
	return n
}

func fetchGroupOffsets(ctx context.Context, dialer *kafka.Dialer, cfg config.KafkaConfig, topic string, base []KafkaPartitionLag) ([]KafkaPartitionLag, error) {
	transport := &kafka.Transport{}
	if dialer != nil && dialer.SASLMechanism != nil {
		transport.SASL = dialer.SASLMechanism
	}
	client := &kafka.Client{
		Addr:      kafka.TCP(cfg.Brokers...),
		Timeout:   10 * time.Second,
		Transport: transport,
	}

	partIDs := make([]int, 0, len(base))
	offsetReqs := make([]kafka.OffsetRequest, 0, len(base))
	for _, p := range base {
		partIDs = append(partIDs, p.Partition)
		offsetReqs = append(offsetReqs, kafka.OffsetRequest{
			Partition: p.Partition,
			Timestamp: kafka.LastOffset,
		})
	}

	list, err := client.ListOffsets(ctx, &kafka.ListOffsetsRequest{
		Topics: map[string][]kafka.OffsetRequest{topic: offsetReqs},
	})
	if err != nil {
		return nil, err
	}

	offResp, err := client.OffsetFetch(ctx, &kafka.OffsetFetchRequest{
		GroupID: cfg.ConsumerGroup,
		Topics:  map[string][]int{topic: partIDs},
	})
	if err != nil {
		return nil, err
	}

	hwByPart := map[int]int64{}
	if list != nil {
		if topicParts, ok := list.Topics[topic]; ok {
			for _, p := range topicParts {
				hwByPart[p.Partition] = p.LastOffset
			}
		}
	}
	cgByPart := map[int]int64{}
	if offResp != nil {
		if topicParts, ok := offResp.Topics[topic]; ok {
			for _, p := range topicParts {
				cgByPart[p.Partition] = p.CommittedOffset
			}
		}
	}

	out := make([]KafkaPartitionLag, 0, len(base))
	for _, p := range base {
		hw := hwByPart[p.Partition]
		if hw == 0 {
			hw = p.HighWaterMark
		}
		cg, hasCG := cgByPart[p.Partition]
		if !hasCG {
			cg = -1
		}
		lag := int64(0)
		if cg >= 0 && hw >= cg {
			lag = hw - cg
		} else if cg < 0 {
			lag = hw
		}
		out = append(out, KafkaPartitionLag{
			Topic:          topic,
			Partition:      p.Partition,
			HighWaterMark:  hw,
			ConsumerOffset: cg,
			Lag:            lag,
		})
	}
	return out, nil
}
