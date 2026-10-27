package logplatform

import (
	"fmt"
	"net/url"
	"strings"

	"yunshu/internal/config"
	"yunshu/internal/pkg/validateutil"

	"gopkg.in/yaml.v3"
)

type loggiePipelinesDoc struct {
	Pipelines []loggiePipelineSpec `yaml:"pipelines"`
}

type loggiePipelineSpec struct {
	Name    string             `yaml:"name"`
	Sources []loggieSourceSpec `yaml:"sources"`
	Sink    loggieSinkSpec     `yaml:"sink"`
}

type loggieSourceSpec struct {
	Type  string   `yaml:"type"`
	Paths []string `yaml:"paths"`
}

type loggieSinkSpec struct {
	Type    string   `yaml:"type"`
	Hosts   []string `yaml:"hosts"`
	Brokers []string `yaml:"brokers"`
}

func parseLoggiePipelines(raw string) (*loggiePipelinesDoc, error) {
	var doc loggiePipelinesDoc
	if err := yaml.Unmarshal([]byte(raw), &doc); err != nil {
		return nil, fmt.Errorf("pipelines.yml 解析失败: %v", err)
	}
	if len(doc.Pipelines) == 0 {
		return nil, fmt.Errorf("pipelines.yml 至少需要一条 pipeline")
	}
	return &doc, nil
}

func loggiePipelineLabel(p loggiePipelineSpec, idx int) string {
	if name := strings.TrimSpace(p.Name); name != "" {
		return name
	}
	return fmt.Sprintf("#%d", idx+1)
}

// validateLoggiePipelinesStructure 限制自定义 pipeline 的 source/sink 类型与采集路径。
// 自定义 YAML 若允许任意 source 类型，可绕过日志源白名单读取主机任意文件。
func validateLoggiePipelinesStructure(raw string) error {
	doc, err := parseLoggiePipelines(raw)
	if err != nil {
		return err
	}
	for i, p := range doc.Pipelines {
		label := loggiePipelineLabel(p, i)
		if len(p.Sources) == 0 {
			return fmt.Errorf("pipeline %s 缺少 sources", label)
		}
		for _, src := range p.Sources {
			switch strings.ToLower(strings.TrimSpace(src.Type)) {
			case "file":
				if len(src.Paths) == 0 {
					return fmt.Errorf("pipeline %s 的 file source 缺少 paths", label)
				}
				for _, pth := range src.Paths {
					if err := validateutil.ValidateLogCollectPath(pth); err != nil {
						return fmt.Errorf("pipeline %s 日志路径不合法 %q: %v", label, pth, err)
					}
				}
			case "journal", "kubeevent":
			default:
				return fmt.Errorf("pipeline %s 不支持的 source 类型: %s", label, src.Type)
			}
		}
		switch strings.ToLower(strings.TrimSpace(p.Sink.Type)) {
		case "elasticsearch", "kafka", "devnull", "stdout":
		default:
			return fmt.Errorf("pipeline %s 不支持的 sink 类型: %s", label, p.Sink.Type)
		}
	}
	return nil
}

// validateLoggiePipelinesSinks 要求自定义 sink 只能指向平台已配置的 ES/Kafka，避免凭据/日志外带。
func validateLoggiePipelinesSinks(raw string, esCfg config.ElasticsearchConfig, kafkaCfg config.KafkaConfig) error {
	doc, err := parseLoggiePipelines(raw)
	if err != nil {
		return err
	}
	esAllowed := map[string]bool{}
	for _, a := range esCfg.Addresses {
		if h := endpointHost(a); h != "" {
			esAllowed[h] = true
		}
	}
	kafkaAllowed := map[string]bool{}
	for _, b := range kafkaCfg.Brokers {
		if h := endpointHost(b); h != "" {
			kafkaAllowed[h] = true
		}
	}
	for i, p := range doc.Pipelines {
		label := loggiePipelineLabel(p, i)
		switch strings.ToLower(strings.TrimSpace(p.Sink.Type)) {
		case "elasticsearch":
			if len(esAllowed) == 0 {
				return fmt.Errorf("pipeline %s 使用 Elasticsearch sink，但平台未配置 Elasticsearch", label)
			}
			if len(p.Sink.Hosts) == 0 {
				return fmt.Errorf("pipeline %s 的 Elasticsearch sink 缺少 hosts", label)
			}
			for _, h := range p.Sink.Hosts {
				if !esAllowed[endpointHost(h)] {
					return fmt.Errorf("pipeline %s 的 Elasticsearch 地址不在平台配置中: %s", label, h)
				}
			}
		case "kafka":
			if len(kafkaAllowed) == 0 {
				return fmt.Errorf("pipeline %s 使用 Kafka sink，但平台未配置 Kafka", label)
			}
			if len(p.Sink.Brokers) == 0 {
				return fmt.Errorf("pipeline %s 的 Kafka sink 缺少 brokers", label)
			}
			for _, b := range p.Sink.Brokers {
				if !kafkaAllowed[endpointHost(b)] {
					return fmt.Errorf("pipeline %s 的 Kafka 地址不在平台配置中: %s", label, b)
				}
			}
		}
	}
	return nil
}

func endpointHost(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if u, err := url.Parse(s); err == nil && u.Host != "" {
		return strings.ToLower(u.Host)
	}
	return strings.ToLower(s)
}
