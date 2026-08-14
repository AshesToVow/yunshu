package logplatform

import (
	"strings"
	"testing"

	"yunshu/internal/config"
)

func TestRenderLoggieSinkYAML_KafkaAndES(t *testing.T) {
	kafka := renderLoggieSinkYAML("yunshu-k8s-1-p2-${+YYYY.MM.DD}", true, config.ElasticsearchConfig{}, config.KafkaConfig{
		Enabled:       true,
		Brokers:       []string{"10.0.0.1:9092"},
		ConsumerGroup: "yunshu-log-es",
	})
	if !strings.Contains(kafka, "type: kafka") || !strings.Contains(kafka, "10.0.0.1:9092") {
		t.Fatalf("kafka sink unexpected:\n%s", kafka)
	}
	es := renderLoggieSinkYAML("yunshu-agent-host-${+YYYY.MM.DD}", false, config.ElasticsearchConfig{
		Addresses: []string{"http://es:9200"},
	}, config.KafkaConfig{})
	if !strings.Contains(es, "type: elasticsearch") || !strings.Contains(es, "http://es:9200") {
		t.Fatalf("es sink unexpected:\n%s", es)
	}
}

func TestRenderClusterSinkUsesSinkViaKafka(t *testing.T) {
	// Normalized 会补默认 consumer_group；关闭 Enabled 才应直写 ES
	yml := renderClusterSinkBlock(1, 2, config.ElasticsearchConfig{Addresses: []string{"http://es:9200"}}, config.KafkaConfig{
		Enabled: false,
		Brokers: []string{"kafka:9092"},
	})
	if strings.Contains(yml, "type: kafka") {
		t.Fatal("expected ES sink when SinkViaKafka=false")
	}
	if !strings.Contains(yml, "yunshu-k8s-2-p1-") {
		t.Fatalf("expected project-scoped index, got:\n%s", yml)
	}
	via := renderClusterSinkBlock(1, 2, config.ElasticsearchConfig{}, config.KafkaConfig{
		Enabled: true,
		Brokers: []string{"kafka:9092"},
	})
	if !strings.Contains(via, "type: kafka") {
		t.Fatal("expected kafka sink when SinkViaKafka=true")
	}
}
