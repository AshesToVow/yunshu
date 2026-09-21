package esmgmt

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
)

// IndexTemplateView 索引模板。
type IndexTemplateView struct {
	Kind string          `json:"kind"` // legacy | composable
	Name string          `json:"name"`
	Body json.RawMessage `json:"body"`
}

// PutTemplateRequest 创建或覆盖模板。
type PutTemplateRequest struct {
	ConnectionID uint            `json:"connection_id"`
	Kind         string          `json:"kind"`
	Name         string          `json:"name"`
	Body         json.RawMessage `json:"body"`
}

func (s *Service) ListTemplates(ctx context.Context, connectionID uint, kind string) ([]IndexTemplateView, error) {
	cli, err := s.resolveClient(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	kind = strings.ToLower(strings.TrimSpace(kind))
	out := make([]IndexTemplateView, 0)
	if kind == "" || kind == "legacy" {
		legacy, err := cli.ListLegacyTemplates(ctx)
		if err != nil {
			return nil, constants.ErrBadRequestWithMsg("列出旧版模板失败: " + err.Error())
		}
		for name, body := range legacy {
			if strings.HasPrefix(name, ".") {
				continue
			}
			out = append(out, IndexTemplateView{Kind: "legacy", Name: name, Body: body})
		}
	}
	if kind == "" || kind == "composable" {
		rows, err := cli.ListComposableTemplates(ctx)
		if err != nil {
			return nil, constants.ErrBadRequestWithMsg("列出可组合模板失败: " + err.Error())
		}
		for _, row := range rows {
			if strings.HasPrefix(row.Name, ".") {
				continue
			}
			body := row.Body
			if len(body) == 0 {
				body = json.RawMessage("null")
			}
			out = append(out, IndexTemplateView{Kind: "composable", Name: row.Name, Body: body})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind == out[j].Kind {
			return out[i].Name < out[j].Name
		}
		return out[i].Kind < out[j].Kind
	})
	return out, nil
}

func (s *Service) GetTemplate(ctx context.Context, connectionID uint, kind, name string) (*IndexTemplateView, error) {
	kind, name, err := validateTemplateRef(kind, name)
	if err != nil {
		return nil, err
	}
	cli, err := s.resolveClient(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	var body json.RawMessage
	if kind == "legacy" {
		body, err = cli.GetLegacyTemplate(ctx, name)
	} else {
		body, err = cli.GetComposableTemplate(ctx, name)
	}
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("读取模板失败: " + err.Error())
	}
	return &IndexTemplateView{Kind: kind, Name: name, Body: body}, nil
}

func (s *Service) PutTemplate(ctx context.Context, req PutTemplateRequest, actor *auth.CurrentUser) error {
	if err := s.assertConnectionWrite(ctx, req.ConnectionID, actor); err != nil {
		return err
	}
	kind, name, err := validateTemplateRef(req.Kind, req.Name)
	if err != nil {
		return err
	}
	body := bytesTrim(req.Body)
	if len(body) == 0 || !json.Valid(body) || body[0] != '{' {
		return constants.ErrBadRequestWithMsg("模板内容须为 JSON 对象")
	}
	if len(body) > maxTemplateBytes {
		return constants.ErrBadRequestWithMsg("模板过大")
	}
	if strings.Contains(strings.ToLower(string(body)), "painless") {
		return constants.ErrBadRequestWithMsg("不允许在模板中写入脚本")
	}
	cli, err := s.resolveClient(ctx, req.ConnectionID)
	if err != nil {
		return err
	}
	if kind == "legacy" {
		err = cli.PutLegacyTemplate(ctx, name, body)
	} else {
		err = cli.PutComposableTemplate(ctx, name, body)
	}
	if err != nil {
		return constants.ErrBadRequestWithMsg("保存模板失败: " + err.Error())
	}
	return nil
}

func (s *Service) DeleteTemplate(ctx context.Context, connectionID uint, kind, name string, actor *auth.CurrentUser) error {
	if err := s.assertConnectionWrite(ctx, connectionID, actor); err != nil {
		return err
	}
	kind, name, err := validateTemplateRef(kind, name)
	if err != nil {
		return err
	}
	cli, err := s.resolveClient(ctx, connectionID)
	if err != nil {
		return err
	}
	if kind == "legacy" {
		err = cli.DeleteLegacyTemplate(ctx, name)
	} else {
		err = cli.DeleteComposableTemplate(ctx, name)
	}
	if err != nil {
		return constants.ErrBadRequestWithMsg("删除模板失败: " + err.Error())
	}
	return nil
}

func validateTemplateRef(kind, name string) (string, string, error) {
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind != "legacy" && kind != "composable" {
		return "", "", constants.ErrBadRequestWithMsg("kind 须为 legacy 或 composable")
	}
	name = strings.TrimSpace(name)
	if name == "" || strings.HasPrefix(name, ".") {
		return "", "", constants.ErrBadRequestWithMsg("模板名非法")
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '.', r == '_', r == '-':
		default:
			return "", "", constants.ErrBadRequestWithMsg("模板名含非法字符")
		}
	}
	return kind, name, nil
}
