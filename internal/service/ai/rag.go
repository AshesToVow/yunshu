package ai

import (
	"context"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"yunshu/internal/ai/knowledge"
	"yunshu/internal/ai/runbooks"
	"yunshu/internal/model"
)

type ragHit struct {
	Source  string  `json:"source"`
	Module  string  `json:"module,omitempty"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

// retrieveKnowledge：案例/DB chunks 优先，其次 ES，再回退内嵌文档。
func (s *Service) retrieveKnowledge(ctx context.Context, query string, topK int) []ragHit {
	query = strings.TrimSpace(query)
	if query == "" || topK <= 0 {
		return nil
	}
	s.ensureSeed()
	modules := knowledge.InferModules(query)
	if hits := s.retrieveFromDB(ctx, query, modules, topK); len(hits) > 0 {
		return filterWeakHits(hits, 1)
	}
	if hits := s.retrieveFromES(ctx, query, modules, topK); len(hits) > 0 {
		return filterWeakHits(hits, 0.5)
	}
	return filterWeakHits(retrieveFromEmbed(query, modules, topK), 1)
}

func (s *Service) retrieveFromDB(ctx context.Context, query string, modules []string, topK int) []ragHit {
	tokens := tokenizeQuery(query)
	if len(tokens) == 0 {
		return nil
	}
	var out []ragHit
	// 故障案例优先
	var cases []model.AiIncidentCase
	_ = s.db.WithContext(ctx).Where("enabled = ?", true).Limit(200).Find(&cases).Error
	for _, c := range cases {
		blob := strings.ToLower(c.Title + " " + c.Symptom + " " + c.RootCause + " " + c.Solution + " " + c.Category + " " + c.Technology)
		sc := scoreTokens(blob, tokens)
		if sc <= 0 {
			continue
		}
		if len(modules) > 0 {
			for _, m := range modules {
				if strings.Contains(blob, m) || (m == "k8s" && strings.Contains(blob, "pod")) {
					sc += 4
					break
				}
			}
		}
		content := "【案例】" + c.Title + "\n现象：" + truncateRunesStr(c.Symptom, 400) +
			"\n根因：" + truncateRunesStr(c.RootCause, 400) +
			"\n方案：" + truncateRunesStr(c.Solution, 400)
		out = append(out, ragHit{Source: "case:" + c.CaseID, Module: "case", Content: content, Score: sc + c.Confidence})
	}
	// KB chunks
	var chunks []model.AiKbChunk
	_ = s.db.WithContext(ctx).Order("id DESC").Limit(500).Find(&chunks).Error
	for _, ch := range chunks {
		blob := strings.ToLower(ch.HeadingPath + " " + ch.Content)
		sc := scoreTokens(blob, tokens)
		if sc <= 0 {
			continue
		}
		out = append(out, ragHit{
			Source:  "chunk:" + strconv.FormatUint(uint64(ch.ID), 10),
			Module:  "kb",
			Content: truncateRunesStr(ch.HeadingPath+"\n"+ch.Content, 2200),
			Score:   sc,
		})
	}
	// SOP
	var sops []model.AiSOP
	_ = s.db.WithContext(ctx).Where("enabled = ?", true).Limit(100).Find(&sops).Error
	for _, sp := range sops {
		blob := strings.ToLower(sp.Title + " " + sp.Scenario + " " + sp.CheckSteps + " " + sp.ExecSteps)
		sc := scoreTokens(blob, tokens)
		if sc <= 0 {
			continue
		}
		content := "【SOP】" + sp.Title + "\n场景：" + truncateRunesStr(sp.Scenario, 300) +
			"\n检查：" + truncateRunesStr(sp.CheckSteps, 500) +
			"\n执行：" + truncateRunesStr(sp.ExecSteps, 500)
		out = append(out, ragHit{Source: "sop:" + sp.Code, Module: "sop", Content: content, Score: sc})
	}
	sortRagHits(out)
	if len(out) > topK {
		out = out[:topK]
	}
	return out
}

func scoreTokens(blob string, tokens []string) float64 {
	sc := float64(0)
	for _, t := range tokens {
		if strings.Contains(blob, t) {
			sc++
		}
	}
	return sc
}

func filterWeakHits(hits []ragHit, minScore float64) []ragHit {
	if len(hits) == 0 {
		return nil
	}
	out := make([]ragHit, 0, len(hits))
	for _, h := range hits {
		if h.Score >= minScore {
			out = append(out, h)
		}
	}
	if len(out) == 0 && len(hits) > 0 {
		// 保底取最高分 1～2 条，避免完全无 RAG
		sortRagHits(hits)
		n := 2
		if len(hits) < n {
			n = len(hits)
		}
		return hits[:n]
	}
	return out
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
				"fields": []string{"title^3", "content", "source^2", "module"},
			},
		},
	}
	boolQ := map[string]any{"must": must}
	if len(modules) > 0 {
		boolQ["should"] = []any{
			map[string]any{"terms": map[string]any{"module.keyword": modules}},
			map[string]any{"terms": map[string]any{"module": modules}},
		}
		// 有模块推断时提高相关模块权重，但不硬过滤（避免字段类型差异导致空结果）
		boolQ["boost"] = 1.0
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
		if len(modules) > 0 && mod != "" {
			for _, want := range modules {
				if want == mod {
					score += 5
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
		lower := strings.ToLower(d.Title + " " + d.Content + " " + d.Source + " " + d.Module)
		for _, t := range tokens {
			if strings.Contains(lower, t) {
				sc++
			}
		}
		if len(modSet) > 0 {
			if _, ok := modSet[d.Module]; ok {
				sc += 4
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
	Title   string
	Content string
}

func embeddedKBDocs() []kbDoc {
	var docs []kbDoc
	for _, d := range knowledge.ModuleDocs() {
		docs = append(docs, kbDoc{Module: d.Module, Source: d.Source, Title: d.Title, Content: d.Content})
	}
	for _, name := range runbooks.Names() {
		body, err := runbooks.Load(name)
		if err != nil {
			continue
		}
		docs = append(docs, kbDoc{
			Module:  knowledge.ModuleK8s,
			Source:  "runbook:" + name,
			Title:   "排障剧本 " + name,
			Content: body,
		})
	}
	// 注意：不再把 prompt 模板塞进知识库，避免「教模型如何说话」的元指令污染检索
	return docs
}

func tokenize(q string) []string {
	q = strings.ToLower(q)
	var cur strings.Builder
	var out []string
	seen := map[string]struct{}{}
	add := func(t string) {
		t = strings.TrimSpace(t)
		if len(t) < 2 {
			return
		}
		if _, ok := seen[t]; ok {
			return
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	flush := func() {
		t := cur.String()
		cur.Reset()
		add(t)
	}
	var cjk []rune
	flushCJK := func() {
		if len(cjk) == 0 {
			return
		}
		// 中文连续串：单字（长度≥2 时）+ 二字/三字 gram，改善关键词召回
		if len(cjk) == 1 {
			add(string(cjk[0]))
		} else {
			for i := 0; i < len(cjk); i++ {
				add(string(cjk[i]))
				if i+1 < len(cjk) {
					add(string(cjk[i : i+2]))
				}
				if i+2 < len(cjk) {
					add(string(cjk[i : i+3]))
				}
			}
		}
		cjk = cjk[:0]
	}
	for _, r := range q {
		switch {
		case unicode.Is(unicode.Han, r):
			flush()
			cjk = append(cjk, r)
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			flushCJK()
			cur.WriteRune(r)
		default:
			flush()
			flushCJK()
		}
	}
	flush()
	flushCJK()
	return out
}

// SyncKnowledgeBase 将 DB 文档/案例/SOP 同步到 ES（含 module 字段）。
func (s *Service) SyncKnowledgeBase(ctx context.Context) (int, error) {
	s.ensureSeed()
	if s.esProvider == nil {
		return 0, nil
	}
	cli, _, err := s.esProvider.Client(ctx)
	if err != nil {
		return 0, err
	}
	n := 0
	var docs []model.AiKbDocument
	_ = s.db.WithContext(ctx).Where("enabled = ?", true).Find(&docs).Error
	for i, d := range docs {
		id := "doc-" + strconv.FormatUint(uint64(d.ID), 10)
		body := map[string]any{
			"source": d.Source, "module": "kb", "title": d.Title, "content": d.Content, "seq": i,
		}
		if err := cli.IndexDoc(ctx, "yunshu-ai-kb-v1", id, body); err == nil {
			n++
		}
	}
	var cases []model.AiIncidentCase
	_ = s.db.WithContext(ctx).Where("enabled = ?", true).Find(&cases).Error
	for i, c := range cases {
		content := c.Title + "\n" + c.Symptom + "\n" + c.RootCause + "\n" + c.Solution
		body := map[string]any{
			"source": "case:" + c.CaseID, "module": "case", "title": c.Title, "content": content, "seq": i,
		}
		if err := cli.IndexDoc(ctx, "yunshu-ai-kb-v1", "case-"+c.CaseID, body); err == nil {
			n++
		}
	}
	var sops []model.AiSOP
	_ = s.db.WithContext(ctx).Where("enabled = ?", true).Find(&sops).Error
	for i, sp := range sops {
		content := sp.Title + "\n" + sp.Scenario + "\n" + sp.CheckSteps + "\n" + sp.ExecSteps
		body := map[string]any{
			"source": "sop:" + sp.Code, "module": "sop", "title": sp.Title, "content": content, "seq": i,
		}
		if err := cli.IndexDoc(ctx, "yunshu-ai-kb-v1", "sop-"+sp.Code, body); err == nil {
			n++
		}
	}
	return n, nil
}

func truncateRunesStr(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	if maxRunes <= 0 || utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxRunes]) + "…"
}
