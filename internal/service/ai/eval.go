package ai

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
)

// RunEvalSuite 对启用的评估用例做轻量回归（关键词/禁止项，不强制调 LLM 以降低成本；可选真实 Chat）。
func (s *Service) RunEvalSuite(ctx context.Context, user *auth.CurrentUser, liveChat bool) (*model.AiEvalRun, error) {
	s.ensureSeed()
	var uid uint
	if user != nil {
		uid = user.ID
	}
	run := model.AiEvalRun{Suite: "default", Status: "running", CreatedBy: uid, CreatedAt: time.Now()}
	if err := s.db.WithContext(ctx).Create(&run).Error; err != nil {
		return nil, err
	}
	var cases []model.AiEvalCase
	_ = s.db.WithContext(ctx).Where("enabled = ?", true).Find(&cases).Error
	total, max := 0.0, 0.0
	for _, c := range cases {
		max += float64(c.ScoreWeight)
		reply := ""
		toolsUsed := []string{}
		if liveChat {
			enableTools := true
			res, err := s.Chat(ctx, uid, user, ChatRequest{
				Messages:    []ChatMessage{{Role: "user", Content: c.InputQuestion}},
				EnableTools: &enableTools,
				DisableRAG:  false,
			})
			if err == nil && res != nil {
				reply = res.Reply
				for _, st := range res.ToolSteps {
					toolsUsed = append(toolsUsed, st.Name)
				}
			} else if err != nil {
				reply = "ERROR: " + err.Error()
			}
		} else {
			// 离线：用 RAG + Prompt 存在性做冒烟
			hits := s.retrieveKnowledge(ctx, c.InputQuestion, 4)
			parts := make([]string, 0, len(hits))
			for _, h := range hits {
				parts = append(parts, h.Content)
			}
			reply = strings.Join(parts, "\n")
			if reply == "" {
				reply = "（无 RAG 命中，离线评估仅检查知识库覆盖）"
			}
		}
		score, detail := scoreEvalCase(c, reply, toolsUsed)
		total += score
		_ = s.db.WithContext(ctx).Create(&model.AiEvalResult{
			RunID: run.ID, CaseID: c.ID, CaseCode: c.CaseCode,
			Passed: score >= float64(c.ScoreWeight)*0.6, Score: score, MaxScore: float64(c.ScoreWeight),
			Detail: detail, Reply: truncateStr(reply, 4000),
		}).Error
	}
	now := time.Now()
	run.Status = "done"
	run.TotalScore = total
	run.MaxScore = max
	run.FinishedAt = &now
	if max > 0 {
		run.Summary = "score=" + formatFloat(total) + "/" + formatFloat(max)
	}
	_ = s.db.WithContext(ctx).Save(&run).Error
	return &run, nil
}

func formatFloat(f float64) string {
	b, _ := json.Marshal(f)
	return string(b)
}

func scoreEvalCase(c model.AiEvalCase, reply string, tools []string) (float64, string) {
	lower := strings.ToLower(reply)
	weight := float64(c.ScoreWeight)
	if weight <= 0 {
		weight = 10
	}
	got := weight
	var notes []string
	var expect []string
	_ = json.Unmarshal([]byte(c.ExpectKeywords), &expect)
	for _, kw := range expect {
		kw = strings.TrimSpace(strings.ToLower(kw))
		if kw == "" {
			continue
		}
		if !strings.Contains(lower, kw) {
			got -= weight * 0.15
			notes = append(notes, "missing keyword:"+kw)
		}
	}
	var forbid []string
	_ = json.Unmarshal([]byte(c.ForbidKeywords), &forbid)
	for _, kw := range forbid {
		kw = strings.TrimSpace(strings.ToLower(kw))
		if kw == "" {
			continue
		}
		if strings.Contains(lower, kw) {
			got -= weight * 0.3
			notes = append(notes, "forbid:"+kw)
		}
	}
	var expectTools []string
	_ = json.Unmarshal([]byte(c.ExpectTools), &expectTools)
	if len(expectTools) > 0 && len(tools) > 0 {
		set := map[string]struct{}{}
		for _, t := range tools {
			set[t] = struct{}{}
		}
		for _, t := range expectTools {
			if _, ok := set[t]; !ok {
				got -= weight * 0.1
				notes = append(notes, "missing tool:"+t)
			}
		}
	}
	if got < 0 {
		got = 0
	}
	return got, strings.Join(notes, "; ")
}
