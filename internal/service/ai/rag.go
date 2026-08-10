package ai

import (
	"context"
	"strings"
	"unicode"

	"yunshu/internal/ai/runbooks"
	"yunshu/internal/ai/prompts"
)

type ragHit struct {
	Source  string  `json:"source"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

// retrieveKnowledge：优先 ES BM25（yunshu-ai-kb-*），失败则回退内嵌文档关键词匹配。
func (s *Service) retrieveKnowledge(ctx context.Context, query string, topK int) []ragHit {
	query = strings.TrimSpace(query)
	if query == "" || topK <= 0 {
		return nil
	}
	if hits := s.retrieveFromES(ctx, query, topK); len(hits) > 0 {
		return hits
	}
	return retrieveFromEmbed(query, topK)
}

func (s *Service) retrieveFromES(ctx context.Context, query string, topK int) []ragHit {
	if s.esProvider == nil {
		return nil
	}
	cli, _, err := s.esProvider.Client(ctx)
	if err != nil || cli == nil {
		return nil
	}
	body := map[string]any{
		"size": topK,
		"query": map[string]any{
			"multi_match": map[string]any{
				"query":  query,
				"fields": []string{"title^2", "content", "source"},
			},
		},
	}
	res, err := cli.Search(ctx, "yunshu-ai-kb-*", body)
	if err != nil || res == nil {
		return nil
	}
	hitsRaw, _ := res["hits"].(map[string]any)
	if hitsRaw == nil {
		return nil
	}
	arr, _ := hitsRaw["hits"].([]any)
	out := make([]ragHit, 0, len(arr))
	for _, it := range arr {
		m, _ := it.(map[string]any)
		if m == nil {
			continue
		}
		src, _ := m["_source"].(map[string]any)
		if src == nil {
			continue
		}
		score, _ := m["_score"].(float64)
		out = append(out, ragHit{
			Source:  strAny(src["source"]),
			Content: strAny(src["content"]),
			Score:   score,
		})
	}
	return out
}

func strAny(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func retrieveFromEmbed(query string, topK int) []ragHit {
	docs := embeddedKBDocs()
	tokens := tokenize(query)
	type scored struct {
		hit   ragHit
		score float64
	}
	var ranked []scored
	for _, d := range docs {
		sc := float64(0)
		lower := strings.ToLower(d.Content + " " + d.Source)
		for _, t := range tokens {
			if strings.Contains(lower, t) {
				sc++
			}
		}
		if sc > 0 {
			ranked = append(ranked, scored{hit: ragHit{Source: d.Source, Content: d.Content, Score: sc}, score: sc})
		}
	}
	// 简单选择排序取 topK
	for i := 0; i < len(ranked); i++ {
		for j := i + 1; j < len(ranked); j++ {
			if ranked[j].score > ranked[i].score {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
	}
	if len(ranked) > topK {
		ranked = ranked[:topK]
	}
	out := make([]ragHit, 0, len(ranked))
	for _, r := range ranked {
		out = append(out, r.hit)
	}
	return out
}

type kbDoc struct {
	Source  string
	Content string
}

func embeddedKBDocs() []kbDoc {
	var docs []kbDoc
	for _, name := range runbooks.Names() {
		body, err := runbooks.Load(name)
		if err != nil {
			continue
		}
		docs = append(docs, kbDoc{Source: "runbook:" + name, Content: body})
	}
	for _, name := range []string{"system_ops_assistant", "k8s_pod_diagnose", "cicd_build_fail", "alert_explain"} {
		body, err := prompts.Load(name, map[string]string{"context_json": "{}"})
		if err != nil {
			continue
		}
		docs = append(docs, kbDoc{Source: "prompt:" + name, Content: truncateStr(body, 4000)})
	}
	docs = append(docs, kbDoc{
		Source:  "doc:ai.md",
		Content: "Yunshu AI 模块：字典 ai_* 配置 Provider；API status/ping/chat/k8s pod-diagnose/cicd build-fail/alert explain；只分析建议不自动执行写操作；助手支持只读工具调用与排障剧本。",
	})
	docs = append(docs, kbDoc{
		Source:  "doc:esmgmt",
		Content: "esmgmt 插件用于 Elasticsearch 集群管理：连接管理、cluster health、索引列表、受限 REST 代理；与日志平台 project-logs / log-platform 职责分离。",
	})
	return docs
}

func tokenize(q string) []string {
	q = strings.ToLower(q)
	var cur strings.Builder
	var out []string
	flush := func() {
		t := cur.String()
		cur.Reset()
		if len(t) >= 2 {
			out = append(out, t)
		}
	}
	for _, r := range q {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r > 127 {
			cur.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return out
}

// SyncKnowledgeBase 将内嵌文档写入 ES 索引 yunshu-ai-kb-v1（可选）。
func (s *Service) SyncKnowledgeBase(ctx context.Context) (int, error) {
	if s.esProvider == nil {
		return 0, nil
	}
	cli, _, err := s.esProvider.Client(ctx)
	if err != nil {
		return 0, err
	}
	docs := embeddedKBDocs()
	n := 0
	for i, d := range docs {
		id := strings.ReplaceAll(d.Source, "/", "_")
		body := map[string]any{
			"source":  d.Source,
			"title":   d.Source,
			"content": d.Content,
			"seq":     i,
		}
		if err := cli.IndexDoc(ctx, "yunshu-ai-kb-v1", id, body); err != nil {
			continue
		}
		n++
	}
	return n, nil
}
