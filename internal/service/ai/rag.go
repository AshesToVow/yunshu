package ai

import (
	"context"
	"strings"
	"unicode"

	"yunshu/internal/ai/knowledge"
	"yunshu/internal/ai/prompts"
	"yunshu/internal/ai/runbooks"
)

type ragHit struct {
	Source  string  `json:"source"`
	Module  string  `json:"module,omitempty"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

// retrieveKnowledge：优先 ES BM25（yunshu-ai-kb-*），失败则回退内嵌文档关键词匹配；按功能模块加权过滤。
func (s *Service) retrieveKnowledge(ctx context.Context, query string, topK int) []ragHit {
	query = strings.TrimSpace(query)
	if query == "" || topK <= 0 {
		return nil
	}
	modules := knowledge.InferModules(query)
	if hits := s.retrieveFromES(ctx, query, modules, topK); len(hits) > 0 {
		return hits
	}
	return retrieveFromEmbed(query, modules, topK)
}

func (s *Service) retrieveFromES(ctx context.Context, query string, modules []string, topK int) []ragHit {
	if s.esProvider == nil {
		return nil
	}
	cli, _, err := s.esProvider.Client(ctx)
	if err != nil || cli == nil {
		return nil
	}
	must := []any{
		map[string]any{
			"multi_match": map[string]any{
				"query":  query,
				"fields": []string{"title^2", "content", "source", "module"},
			},
		},
	}
	boolQ := map[string]any{"must": must}
	if len(modules) > 0 {
		boolQ["should"] = []any{
			map[string]any{"terms": map[string]any{"module": modules}},
		}
		boolQ["minimum_should_match"] = 0
	}
	body := map[string]any{
		"size":  topK,
		"query": map[string]any{"bool": boolQ},
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
		mod := strAny(src["module"])
		// 模块命中时略加权，便于同题多文档时优先相关模块
		if len(modules) > 0 && mod != "" {
			for _, want := range modules {
				if want == mod {
					score += 2
					break
				}
			}
		}
		out = append(out, ragHit{
			Source:  strAny(src["source"]),
			Module:  mod,
			Content: strAny(src["content"]),
			Score:   score,
		})
	}
	sortRagHits(out)
	if len(out) > topK {
		out = out[:topK]
	}
	return out
}

func strAny(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func retrieveFromEmbed(query string, modules []string, topK int) []ragHit {
	docs := embeddedKBDocs()
	tokens := tokenize(query)
	modSet := map[string]struct{}{}
	for _, m := range modules {
		modSet[m] = struct{}{}
	}
	type scored struct {
		hit   ragHit
		score float64
	}
	var ranked []scored
	for _, d := range docs {
		sc := float64(0)
		lower := strings.ToLower(d.Content + " " + d.Source + " " + d.Module)
		for _, t := range tokens {
			if strings.Contains(lower, t) {
				sc++
			}
		}
		if len(modSet) > 0 {
			if _, ok := modSet[d.Module]; ok {
				sc += 3
			}
		}
		if sc > 0 {
			ranked = append(ranked, scored{
				hit:   ragHit{Source: d.Source, Module: d.Module, Content: d.Content, Score: sc},
				score: sc,
			})
		}
	}
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

func sortRagHits(hits []ragHit) {
	for i := 0; i < len(hits); i++ {
		for j := i + 1; j < len(hits); j++ {
			if hits[j].Score > hits[i].Score {
				hits[i], hits[j] = hits[j], hits[i]
			}
		}
	}
}

type kbDoc struct {
	Module  string
	Source  string
	Content string
}

func embeddedKBDocs() []kbDoc {
	var docs []kbDoc
	for _, d := range knowledge.ModuleDocs() {
		docs = append(docs, kbDoc{Module: d.Module, Source: d.Source, Content: d.Content})
	}
	for _, name := range runbooks.Names() {
		body, err := runbooks.Load(name)
		if err != nil {
			continue
		}
		docs = append(docs, kbDoc{Module: knowledge.ModuleK8s, Source: "runbook:" + name, Content: body})
	}
	for _, name := range []string{"system_ops_assistant", "k8s_pod_diagnose", "cicd_build_fail", "alert_explain"} {
		body, err := prompts.Load(name, map[string]string{"context_json": "{}"})
		if err != nil {
			continue
		}
		mod := knowledge.ModuleAI
		switch name {
		case "k8s_pod_diagnose":
			mod = knowledge.ModuleK8s
		case "cicd_build_fail":
			mod = knowledge.ModuleCICD
		case "alert_explain":
			mod = knowledge.ModuleAlert
		}
		docs = append(docs, kbDoc{Module: mod, Source: "prompt:" + name, Content: truncateStr(body, 4000)})
	}
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

// SyncKnowledgeBase 将内嵌文档写入 ES 索引 yunshu-ai-kb-v1（含 module 字段）。
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
		title := d.Source
		for _, md := range knowledge.ModuleDocs() {
			if md.Source == d.Source {
				title = md.Title
				break
			}
		}
		body := map[string]any{
			"source":  d.Source,
			"module":  d.Module,
			"title":   title,
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
