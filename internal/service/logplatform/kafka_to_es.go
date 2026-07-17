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

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
)

// KafkaToESService 消费 Kafka 日志并 bulk 写入 Elasticsearch；同时提供积压观测。
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
}

func NewKafkaToESService(kafka *KafkaProvider, es *ElasticsearchProvider) *KafkaToESService {
	s := &KafkaToESService{kafka: kafka, es: es}
	s.lastError.Store("")
	return s
}

type KafkaPartitionLag struct {
	Partition      int   `json:"partition"`
	HighWaterMark  int64 `json:"high_water_mark"`
	ConsumerOffset int64 `json:"consumer_offset"`
	Lag            int64 `json:"lag"`
}

type KafkaQueueStats struct {
	Enabled         bool                `json:"enabled"`
	SinkViaKafka    bool                `json:"sink_via_kafka"`
	Brokers         []string            `json:"brokers"`
	Topic           string              `json:"topic"`
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
	Enabled       bool     `json:"enabled"`
	Brokers       []string `json:"brokers"`
	Topic         string   `json:"topic"`
	ConsumerGroup string   `json:"consumer_group"`
	Username      string   `json:"username,omitempty"`
	HasPassword   bool     `json:"has_password"`
	SASLMechanism string   `json:"sasl_mechanism,omitempty"`
	BatchSize     int      `json:"batch_size"`
	Workers       int      `json:"workers"`
	SinkViaKafka  bool     `json:"sink_via_kafka"`
}

// Run 阻塞直至 ctx 取消：按字典配置启停消费者。
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
	fp := kafkaFingerprint(cfg)
	if !cfg.SinkViaKafka() {
		s.stopConsumer()
		s.mu.Lock()
		s.cfgFingerprint = fp
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

	s.mu.Lock()
	same := s.running && s.cfgFingerprint == fp
	s.mu.Unlock()
	if same {
		return
	}
	s.stopConsumer()
	s.startConsumer(parent, cfg)
}

func (s *KafkaToESService) startConsumer(parent context.Context, cfg config.KafkaConfig) {
	cfg = cfg.Normalized()
	runCtx, cancel := context.WithCancel(parent)
	s.mu.Lock()
	s.cancel = cancel
	s.running = true
	s.cfgFingerprint = kafkaFingerprint(cfg)
	s.mu.Unlock()
	s.runningFlag.Store(true)
	slog.Default().With("component", "kafka-to-es").Info("kafka consumer starting",
		"topic", cfg.Topic, "group", cfg.ConsumerGroup, "brokers", len(cfg.Brokers))

	go func() {
		defer func() {
			s.runningFlag.Store(false)
			s.mu.Lock()
			s.running = false
			s.mu.Unlock()
		}()
		s.consumeLoop(runCtx, cfg)
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

func (s *KafkaToESService) consumeLoop(ctx context.Context, cfg config.KafkaConfig) {
	dialer, err := kafkaDialer(cfg)
	if err != nil {
		s.setLastError(err.Error())
		slog.Default().With("component", "kafka-to-es").Error("kafka dialer", "err", err)
		return
	}
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		GroupID:        cfg.ConsumerGroup,
		Topic:          cfg.Topic,
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

	flush := func() {
		if len(batch) == 0 {
			return
		}
		toWrite := append([]kafka.Message(nil), batch...)
		if err := s.writeBatch(ctx, toWrite); err != nil {
			s.errorTotal.Add(1)
			s.setLastError(err.Error())
			slog.Default().With("component", "kafka-to-es").Warn("bulk write failed", "err", err, "n", len(toWrite))
			return
		}
		if err := reader.CommitMessages(ctx, toWrite...); err != nil {
			s.errorTotal.Add(1)
			s.setLastError("commit: " + err.Error())
			return
		}
		s.writtenTotal.Add(int64(len(toWrite)))
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

func (s *KafkaToESService) writeBatch(ctx context.Context, msgs []kafka.Message) error {
	cli, _, err := s.es.Client(ctx)
	if err != nil {
		return err
	}
	var ndjson strings.Builder
	for _, m := range msgs {
		doc, index, err := parseKafkaLogMessage(m.Value)
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
		return nil
	}
	return cli.Bulk(ctx, []byte(ndjson.String()))
}

func parseKafkaLogMessage(raw []byte) (map[string]any, string, error) {
	s := strings.TrimSpace(string(raw))
	if s == "" {
		return nil, "", fmt.Errorf("empty message")
	}
	var root map[string]any
	if err := json.Unmarshal([]byte(s), &root); err != nil {
		now := time.Now().UTC()
		return map[string]any{
			"@timestamp": now.Format(time.RFC3339Nano),
			"message":    s,
		}, AgentIndexForDay(0, now), nil
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
	serverID := parseUintAny(fields["server_id"])
	if serverID == 0 {
		serverID = parseUintAny(doc["server_id"])
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
	return doc, AgentIndexForDay(serverID, ts), nil
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

func kafkaFingerprint(cfg config.KafkaConfig) string {
	cfg = cfg.Normalized()
	return fmt.Sprintf("%v|%s|%s|%s|%s|%d|%v",
		cfg.Enabled, strings.Join(cfg.Brokers, ","), cfg.Topic, cfg.ConsumerGroup,
		cfg.Username, cfg.BatchSize, cfg.SASLMechanism)
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
		Enabled:       cfg.Enabled,
		Brokers:       cfg.Brokers,
		Topic:         cfg.Topic,
		ConsumerGroup: cfg.ConsumerGroup,
		Username:      cfg.Username,
		HasPassword:   strings.TrimSpace(cfg.Password) != "",
		SASLMechanism: cfg.SASLMechanism,
		BatchSize:     cfg.BatchSize,
		Workers:       cfg.Workers,
		SinkViaKafka:  cfg.SinkViaKafka(),
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
	out.Topic = cfg.Topic
	out.ConsumerGroup = cfg.ConsumerGroup
	out.HasSASL = strings.TrimSpace(cfg.Username) != ""

	if !cfg.SinkViaKafka() {
		out.Message = "Kafka 中转未启用，Loggie 直写 Elasticsearch"
		return out, nil
	}

	lags, lagTotal, lagErr := fetchKafkaLag(ctx, cfg)
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
		out.Message = "消费运行中"
	} else {
		out.Message = "消费者未运行（请确认 ES 已启用）"
	}
	return out, nil
}

func fetchKafkaLag(ctx context.Context, cfg config.KafkaConfig) ([]KafkaPartitionLag, int64, error) {
	dialer, err := kafkaDialer(cfg)
	if err != nil {
		return nil, 0, err
	}
	conn, err := dialer.DialContext(ctx, "tcp", cfg.Brokers[0])
	if err != nil {
		return nil, 0, fmt.Errorf("dial kafka: %w", err)
	}
	defer conn.Close()

	parts, err := conn.ReadPartitions(cfg.Topic)
	if err != nil {
		return nil, 0, fmt.Errorf("read partitions: %w", err)
	}

	base := make([]KafkaPartitionLag, 0, len(parts))
	seen := map[int]struct{}{}
	for _, p := range parts {
		if p.Topic != cfg.Topic {
			continue
		}
		if _, ok := seen[p.ID]; ok {
			continue
		}
		seen[p.ID] = struct{}{}
		pc, err := dialer.DialLeader(ctx, "tcp", cfg.Brokers[0], cfg.Topic, p.ID)
		if err != nil {
			continue
		}
		hw, err := pc.ReadLastOffset()
		_ = pc.Close()
		if err != nil {
			continue
		}
		base = append(base, KafkaPartitionLag{
			Partition:      p.ID,
			HighWaterMark:  hw,
			ConsumerOffset: -1,
			Lag:            -1,
		})
	}

	if lags, err := fetchGroupOffsets(ctx, dialer, cfg, base); err == nil {
		return lags, sumLag(lags), nil
	}
	return base, 0, nil
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

func fetchGroupOffsets(ctx context.Context, dialer *kafka.Dialer, cfg config.KafkaConfig, base []KafkaPartitionLag) ([]KafkaPartitionLag, error) {
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
		Topics: map[string][]kafka.OffsetRequest{cfg.Topic: offsetReqs},
	})
	if err != nil {
		return nil, err
	}

	offResp, err := client.OffsetFetch(ctx, &kafka.OffsetFetchRequest{
		GroupID: cfg.ConsumerGroup,
		Topics:  map[string][]int{cfg.Topic: partIDs},
	})
	if err != nil {
		return nil, err
	}

	hwByPart := map[int]int64{}
	if list != nil {
		if topicParts, ok := list.Topics[cfg.Topic]; ok {
			for _, p := range topicParts {
				hwByPart[p.Partition] = p.LastOffset
			}
		}
	}
	cgByPart := map[int]int64{}
	if offResp != nil {
		if topicParts, ok := offResp.Topics[cfg.Topic]; ok {
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
			Partition:      p.Partition,
			HighWaterMark:  hw,
			ConsumerOffset: cg,
			Lag:            lag,
		})
	}
	return out, nil
}
