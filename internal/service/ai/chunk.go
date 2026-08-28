package ai

import (
	"context"
	"strings"
	"unicode"

	"yunshu/internal/model"
)

// rechunkDocument 按 Markdown 标题切片。
func (s *Service) rechunkDocument(ctx context.Context, doc *model.AiKbDocument) error {
	_ = s.db.WithContext(ctx).Where("document_id = ?", doc.ID).Delete(&model.AiKbChunk{}).Error
	parts := splitMarkdownChunks(doc.Content)
	for i, p := range parts {
		ch := model.AiKbChunk{
			DocumentID:  doc.ID,
			KBID:        doc.KBID,
			Seq:         i,
			HeadingPath: p.heading,
			Content:     p.body,
		}
		if err := s.db.WithContext(ctx).Create(&ch).Error; err != nil {
			return err
		}
	}
	return nil
}

type mdChunk struct {
	heading string
	body    string
}

func splitMarkdownChunks(content string) []mdChunk {
	lines := strings.Split(content, "\n")
	var out []mdChunk
	var curHeading string
	var buf strings.Builder
	flush := func() {
		body := strings.TrimSpace(buf.String())
		buf.Reset()
		if body == "" && curHeading == "" {
			return
		}
		out = append(out, mdChunk{heading: curHeading, body: body})
	}
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "#") {
			flush()
			curHeading = strings.TrimLeft(trim, "#")
			curHeading = strings.TrimSpace(curHeading)
			continue
		}
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	flush()
	if len(out) == 0 && strings.TrimSpace(content) != "" {
		out = append(out, mdChunk{heading: "", body: strings.TrimSpace(content)})
	}
	// 合并过短块
	merged := make([]mdChunk, 0, len(out))
	for _, c := range out {
		if len(merged) > 0 && runeLen(c.body) < 80 {
			merged[len(merged)-1].body += "\n" + c.body
			continue
		}
		merged = append(merged, c)
	}
	return merged
}

func runeLen(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}

func tokenizeQuery(q string) []string {
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
		add(cur.String())
		cur.Reset()
	}
	var cjk []rune
	flushCJK := func() {
		if len(cjk) == 0 {
			return
		}
		for i := range cjk {
			add(string(cjk[i]))
			if i+1 < len(cjk) {
				add(string(cjk[i : i+2]))
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
