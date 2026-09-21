package esmgmt

import (
	"context"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
)

const maxSearchSize = 100
const maxDocBytes = 512 * 1024
const maxTemplateBytes = 256 * 1024

// DocHit 检索命中。
type DocHit struct {
	Index  string          `json:"index"`
	ID     string          `json:"id"`
	Score  any             `json:"score,omitempty"`
	Source json.RawMessage `json:"source"`
}

// SearchDocsRequest 文档检索。
type SearchDocsRequest struct {
	ConnectionID uint   `json:"connection_id"`
	Index        string `json:"index"`
	Query        string `json:"query"`
	Size         int    `json:"size"`
	From         int    `json:"from"`
}

// SearchDocsResult 检索结果。
type SearchDocsResult struct {
	Total int      `json:"total"`
	Hits  []DocHit `json:"hits"`
}

// UpsertDocRequest 写入文档。
type UpsertDocRequest struct {
	ConnectionID uint            `json:"connection_id"`
	Index        string          `json:"index"`
	ID           string          `json:"id"`
	Source       json.RawMessage `json:"source"`
}

func (s *Service) SearchDocs(ctx context.Context, req SearchDocsRequest) (*SearchDocsResult, error) {
	index, err := validateIndexRef(req.Index, true)
	if err != nil {
		return nil, err
	}
	size := req.Size
	if size <= 0 {
		size = 20
	}
	if size > maxSearchSize {
		size = maxSearchSize
	}
	from := req.From
	if from < 0 {
		from = 0
	}
	if from > 10000 {
		return nil, constants.ErrBadRequestWithMsg("from 过大")
	}
	body := map[string]any{
		"size": size,
		"from": from,
		"sort": []any{map[string]any{"_doc": "asc"}},
	}
	q := strings.TrimSpace(req.Query)
	if q == "" {
		body["query"] = map[string]any{"match_all": map[string]any{}}
	} else {
		if strings.Contains(strings.ToLower(q), "painless") {
			return nil, constants.ErrBadRequestWithMsg("不允许脚本查询")
		}
		body["query"] = map[string]any{
			"query_string": map[string]any{
				"query":            q,
				"default_operator": "AND",
			},
		}
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	cli, err := s.resolveClient(ctx, req.ConnectionID)
	if err != nil {
		return nil, err
	}
	raw, err := cli.SearchRaw(ctx, index, payload)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("检索失败: " + err.Error())
	}
	return parseSearchHits(raw)
}

func parseSearchHits(raw []byte) (*SearchDocsResult, error) {
	var wrap struct {
		Hits struct {
			Total any `json:"total"`
			Hits  []struct {
				Index  string          `json:"_index"`
				ID     string          `json:"_id"`
				Score  any             `json:"_score"`
				Source json.RawMessage `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil, constants.ErrBadRequestWithMsg("解析检索结果失败")
	}
	out := &SearchDocsResult{Hits: make([]DocHit, 0, len(wrap.Hits.Hits))}
	switch t := wrap.Hits.Total.(type) {
	case float64:
		out.Total = int(t)
	case map[string]any:
		if v, ok := t["value"].(float64); ok {
			out.Total = int(v)
		}
	}
	for _, h := range wrap.Hits.Hits {
		src := h.Source
		if len(src) == 0 {
			src = json.RawMessage("null")
		}
		out.Hits = append(out.Hits, DocHit{Index: h.Index, ID: h.ID, Score: h.Score, Source: src})
	}
	if out.Total == 0 {
		out.Total = len(out.Hits)
	}
	return out, nil
}

func (s *Service) GetDoc(ctx context.Context, connectionID uint, index, id string) (json.RawMessage, error) {
	index, id, err := validateDocRef(index, id)
	if err != nil {
		return nil, err
	}
	cli, err := s.resolveClient(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	src, found, err := cli.GetDoc(ctx, index, id)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("读取文档失败: " + err.Error())
	}
	if !found {
		return nil, constants.ErrNotFoundWithMsg("文档不存在")
	}
	return src, nil
}

func (s *Service) UpsertDoc(ctx context.Context, req UpsertDocRequest, actor *auth.CurrentUser) error {
	if err := s.assertConnectionWrite(ctx, req.ConnectionID, actor); err != nil {
		return err
	}
	index, id, err := validateDocRef(req.Index, req.ID)
	if err != nil {
		return err
	}
	if err := guardPlatformIndex(index, actor); err != nil {
		return err
	}
	body := bytesTrim(req.Source)
	if len(body) == 0 || !json.Valid(body) {
		return constants.ErrBadRequestWithMsg("source 须为 JSON")
	}
	if body[0] != '{' {
		return constants.ErrBadRequestWithMsg("source 须为 JSON 对象")
	}
	if len(body) > maxDocBytes {
		return constants.ErrBadRequestWithMsg("文档过大")
	}
	if strings.Contains(strings.ToLower(string(body)), "painless") {
		return constants.ErrBadRequestWithMsg("不允许写入脚本")
	}
	cli, err := s.resolveClient(ctx, req.ConnectionID)
	if err != nil {
		return err
	}
	if err := cli.PutDoc(ctx, index, id, body); err != nil {
		return constants.ErrBadRequestWithMsg("写入文档失败: " + err.Error())
	}
	return nil
}

func (s *Service) DeleteDoc(ctx context.Context, connectionID uint, index, id string, actor *auth.CurrentUser) error {
	if err := s.assertConnectionWrite(ctx, connectionID, actor); err != nil {
		return err
	}
	index, id, err := validateDocRef(index, id)
	if err != nil {
		return err
	}
	if err := guardPlatformIndex(index, actor); err != nil {
		return err
	}
	cli, err := s.resolveClient(ctx, connectionID)
	if err != nil {
		return err
	}
	if err := cli.DeleteDoc(ctx, index, id); err != nil {
		return constants.ErrBadRequestWithMsg("删除文档失败: " + err.Error())
	}
	return nil
}

func bytesTrim(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}

func validateIndexRef(name string, allowWildcard bool) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", constants.ErrBadRequestWithMsg("索引名不能为空")
	}
	if strings.HasPrefix(name, ".") || strings.Contains(name, "/.") {
		return "", constants.ErrBadRequestWithMsg("禁止操作系统索引")
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '.', r == '_', r == '-':
		case allowWildcard && r == '*':
		default:
			return "", constants.ErrBadRequestWithMsg("索引名含非法字符")
		}
	}
	if strings.Contains(name, "..") {
		return "", constants.ErrBadRequestWithMsg("索引名含非法字符")
	}
	return name, nil
}

func validateDocRef(index, id string) (string, string, error) {
	index, err := validateIndexRef(index, false)
	if err != nil {
		return "", "", err
	}
	id = strings.TrimSpace(id)
	if id == "" || strings.Contains(id, "/") || strings.Contains(id, "..") {
		return "", "", constants.ErrBadRequestWithMsg("文档 ID 非法")
	}
	if utf8.RuneCountInString(id) > 512 {
		return "", "", constants.ErrBadRequestWithMsg("文档 ID 过长")
	}
	return index, id, nil
}

func guardPlatformIndex(index string, actor *auth.CurrentUser) error {
	lower := strings.ToLower(index)
	if strings.Contains(lower, "yunshu-") && !isSuperAdmin(actor) {
		return constants.ErrForbiddenWithMsg("写入 yunshu-* 索引须超级管理员")
	}
	return nil
}
