package logplatform

import (
	"fmt"
	"strings"

	"yunshu/internal/config"
)

// renderLoggieSinkYAML 主机/集群共用的 Loggie sink 块（Kafka 或 Elasticsearch）。
func renderLoggieSinkYAML(topicOrIndex string, viaKafka bool, esCfg config.ElasticsearchConfig, kafkaCfg config.KafkaConfig) string {
	esCfg = esCfg.Normalized()
	kafkaCfg = kafkaCfg.Normalized()
	topicOrIndex = strings.TrimSpace(topicOrIndex)
	if viaKafka {
		var brokers strings.Builder
		for _, b := range kafkaCfg.Brokers {
			b = strings.TrimSpace(b)
			if b == "" {
				continue
			}
			brokers.WriteString("        - ")
			brokers.WriteString(quoteYAML(b))
			brokers.WriteByte('\n')
		}
		if brokers.Len() == 0 {
			brokers.WriteString("        - \"127.0.0.1:9092\"\n")
		}
		sasl := ""
		if u := strings.TrimSpace(kafkaCfg.Username); u != "" {
			mech := kafkaCfg.SASLMechanism
			if mech == "" || mech == "none" {
				mech = "plain"
			}
			sasl = fmt.Sprintf(`
      sasl:
        type: %s
        userName: %s
        password: %s`, quoteYAML(mech), quoteYAML(u), quoteYAML(kafkaCfg.Password))
		}
		if topicOrIndex == "" {
			topicOrIndex = "yunshu-logs"
		}
		return fmt.Sprintf(`    sink:
      type: kafka
      brokers:
%s      topic: %s
      codec:
        type: json%s
`, brokers.String(), quoteYAML(topicOrIndex), sasl)
	}

	var hostsLines strings.Builder
	for _, h := range esCfg.Addresses {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		hostsLines.WriteString("        - ")
		hostsLines.WriteString(quoteYAML(h))
		hostsLines.WriteByte('\n')
	}
	if hostsLines.Len() == 0 {
		hostsLines.WriteString("        - \"http://127.0.0.1:9200\"\n")
	}
	sinkAuth := ""
	if u := strings.TrimSpace(esCfg.Username); u != "" {
		sinkAuth += fmt.Sprintf("\n      username: %s", quoteYAML(u))
	}
	if p := strings.TrimSpace(esCfg.Password); p != "" {
		sinkAuth += fmt.Sprintf("\n      password: %s", quoteYAML(p))
	}
	if topicOrIndex == "" {
		topicOrIndex = "yunshu-agent-${+YYYY.MM.DD}"
	}
	return fmt.Sprintf(`    sink:
      type: elasticsearch
      hosts:
%s      index: %s
      codec:
        type: json
        beatsFormat: true%s
`, hostsLines.String(), quoteYAML(topicOrIndex), sinkAuth)
}

// renderSinkYAML 主机 Agent sink。
func renderSinkYAML(serverID uint, serverHost string, esCfg config.ElasticsearchConfig, kafkaCfg config.KafkaConfig) string {
	esCfg = esCfg.Normalized()
	kafkaCfg = kafkaCfg.Normalized()
	host := strings.TrimSpace(serverHost)
	if host == "" {
		host = fmt.Sprintf("server-%d", serverID)
	}
	if kafkaCfg.SinkViaKafka() {
		return renderLoggieSinkYAML(AgentKafkaTopicTemplate(host, kafkaCfg.TopicPrefix), true, esCfg, kafkaCfg)
	}
	return renderLoggieSinkYAML(AgentIndexSink(host), false, esCfg, kafkaCfg)
}

// renderClusterSinkBlock 集群 DaemonSet sink（与主机共用渲染逻辑；Kafka 条件对齐 SinkViaKafka）。
func renderClusterSinkBlock(projectID, clusterID uint, esCfg config.ElasticsearchConfig, kafkaCfg config.KafkaConfig) string {
	esCfg = esCfg.Normalized()
	kafkaCfg = kafkaCfg.Normalized()
	if kafkaCfg.SinkViaKafka() {
		return renderLoggieSinkYAML(
			K8sKafkaTopicTemplate(clusterID, projectID, kafkaCfg.K8sTopicPrefix),
			true,
			esCfg,
			kafkaCfg,
		)
	}
	return renderLoggieSinkYAML(K8sIndexSink(clusterID, projectID, esCfg.K8sIndexPrefix), false, esCfg, kafkaCfg)
}
