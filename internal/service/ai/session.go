package ai

import (
	"context"
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/pagination"
)

type SessionCreateRequest struct {
	Title       string `json:"title"`
	ProjectID   uint   `json:"project_id"`
	ClusterID   uint   `json:"cluster_id"`
	Provider    string `json:"provider"`
	EnableTools *bool  `json:"enable_tools"`
	EnableWrite bool   `json:"enable_write"`
}

type SessionUpdateRequest struct {
	Title       *string `json:"title"`
	ProjectID   *uint   `json:"project_id"`
	ClusterID   *uint   `json:"cluster_id"`
	Provider    *string `json:"provider"`
	EnableTools *bool   `json:"enable_tools"`
	EnableWrite *bool   `json:"enable_write"`
}

type SessionListQuery struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

type SessionListItem struct {
	model.AiChatSession
	MessageCount int64 `json:"message_count"`
}

type SessionDetail struct {
	Session  model.AiChatSession  `json:"session"`
	Messages []model.AiChatMessage `json:"messages"`
}

func (s *Service) CreateSession(ctx context.Context, userID uint, req SessionCreateRequest) (*model.AiChatSession, error) {
	if userID == 0 {
		return nil, constants.ErrUnauthorized
	}
	actor := resolveActor(ctx, nil)
	if actor == nil {
		actor = &auth.CurrentUser{ID: userID}
	}
	if req.ProjectID > 0 {
		if err := s.assertProjectMember(ctx, actor, req.ProjectID); err != nil {
			return nil, err
		}
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "新对话"
	}
	enableTools := true
	if req.EnableTools != nil {
		enableTools = *req.EnableTools
	}
	row := model.AiChatSession{
		UserID:      userID,
		Title:       truncateRunes(title, 64),
		ProjectID:   req.ProjectID,
		ClusterID:   req.ClusterID,
		Provider:    strings.TrimSpace(req.Provider),
		EnableTools: enableTools,
		EnableWrite: req.EnableWrite,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) ListSessions(ctx context.Context, userID uint, q SessionListQuery) (*pagination.Result[SessionListItem], error) {
	if userID == 0 {
		return nil, constants.ErrUnauthorized
	}
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	dbq := s.db.WithContext(ctx).Model(&model.AiChatSession{}).Where("user_id = ?", userID)
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []model.AiChatSession
	if err := dbq.Order("COALESCE(last_message_at, updated_at) DESC, id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]SessionListItem, 0, len(rows))
	for _, row := range rows {
		var cnt int64
		_ = s.db.WithContext(ctx).Model(&model.AiChatMessage{}).Where("session_id = ?", row.ID).Count(&cnt).Error
		items = append(items, SessionListItem{AiChatSession: row, MessageCount: cnt})
	}
	return &pagination.Result[SessionListItem]{List: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *Service) GetSession(ctx context.Context, userID, sessionID uint) (*SessionDetail, error) {
	sess, err := s.getOwnedSession(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	var msgs []model.AiChatMessage
	if err := s.db.WithContext(ctx).Where("session_id = ?", sessionID).
		Order("id ASC").Limit(500).Find(&msgs).Error; err != nil {
		return nil, err
	}
	return &SessionDetail{Session: *sess, Messages: msgs}, nil
}

func (s *Service) UpdateSession(ctx context.Context, userID, sessionID uint, req SessionUpdateRequest) (*model.AiChatSession, error) {
	sess, err := s.getOwnedSession(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{}
	if req.Title != nil {
		t := truncateRunes(strings.TrimSpace(*req.Title), 64)
		if t == "" {
			t = "新对话"
		}
		updates["title"] = t
	}
	if req.ProjectID != nil {
		if *req.ProjectID > 0 {
			actor := resolveActor(ctx, nil)
			if actor == nil {
				actor = &auth.CurrentUser{ID: userID}
			}
			if err := s.assertProjectMember(ctx, actor, *req.ProjectID); err != nil {
				return nil, err
			}
		}
		updates["project_id"] = *req.ProjectID
	}
	if req.ClusterID != nil {
		updates["cluster_id"] = *req.ClusterID
	}
	if req.Provider != nil {
		updates["provider"] = strings.TrimSpace(*req.Provider)
	}
	if req.EnableTools != nil {
		updates["enable_tools"] = *req.EnableTools
	}
	if req.EnableWrite != nil {
		updates["enable_write"] = *req.EnableWrite
	}
	if len(updates) == 0 {
		return sess, nil
	}
	if err := s.db.WithContext(ctx).Model(sess).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.getOwnedSession(ctx, userID, sessionID)
}

func (s *Service) DeleteSession(ctx context.Context, userID, sessionID uint) error {
	sess, err := s.getOwnedSession(ctx, userID, sessionID)
	if err != nil {
		return err
	}
	tx := s.db.WithContext(ctx).Begin()
	if err := tx.Where("session_id = ?", sess.ID).Delete(&model.AiChatMessage{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Delete(sess).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (s *Service) ClearSessionMessages(ctx context.Context, userID, sessionID uint) error {
	sess, err := s.getOwnedSession(ctx, userID, sessionID)
	if err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Where("session_id = ?", sess.ID).Delete(&model.AiChatMessage{}).Error; err != nil {
		return err
	}
	now := time.Now()
	return s.db.WithContext(ctx).Model(sess).Updates(map[string]any{
		"title":           "新对话",
		"last_message_at": nil,
		"updated_at":      now,
	}).Error
}

func (s *Service) getOwnedSession(ctx context.Context, userID, sessionID uint) (*model.AiChatSession, error) {
	if userID == 0 {
		return nil, constants.ErrUnauthorized
	}
	if sessionID == 0 {
		return nil, constants.ErrBadRequestWithMsg("会话 ID 无效")
	}
	var sess model.AiChatSession
	err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", sessionID, userID).First(&sess).Error
	if err != nil {
		return nil, constants.ErrNotFoundWithMsg("会话不存在")
	}
	return &sess, nil
}

type chatTurnMeta struct {
	ToolSteps []toolStep `json:"tool_steps,omitempty"`
	RAGHits   []ragHit   `json:"rag_hits,omitempty"`
	Provider  string     `json:"provider,omitempty"`
	Model     string     `json:"model,omitempty"`
}

// persistChatTurn 将本轮 user/assistant 消息写入 MySQL，并返回会话 ID。
func (s *Service) persistChatTurn(
	ctx context.Context,
	userID uint,
	req ChatRequest,
	userContent, assistantContent string,
	steps []toolStep,
	ragHits []ragHit,
	provider, modelName string,
) (uint, error) {
	if userID == 0 {
		return 0, nil
	}
	userContent = strings.TrimSpace(userContent)
	assistantContent = strings.TrimSpace(assistantContent)
	if userContent == "" && assistantContent == "" {
		return req.SessionID, nil
	}

	var sess *model.AiChatSession
	var err error
	if req.SessionID > 0 {
		sess, err = s.getOwnedSession(ctx, userID, req.SessionID)
		if err != nil {
			return 0, err
		}
	} else {
		title := "新对话"
		if userContent != "" {
			title = truncateRunes(userContent, 40)
		}
		enableTools := true
		if req.EnableTools != nil {
			enableTools = *req.EnableTools
		}
		sess, err = s.CreateSession(ctx, userID, SessionCreateRequest{
			Title:       title,
			ProjectID:   req.ProjectID,
			ClusterID:   req.ClusterID,
			Provider:    req.Provider,
			EnableTools: &enableTools,
			EnableWrite: req.EnableWrite,
		})
		if err != nil {
			return 0, err
		}
	}

	now := time.Now()
	msgs := make([]model.AiChatMessage, 0, 2)
	if userContent != "" {
		msgs = append(msgs, model.AiChatMessage{
			SessionID: sess.ID,
			Role:      "user",
			Content:   userContent,
			CreatedAt: now,
		})
	}
	if assistantContent != "" {
		meta, _ := json.Marshal(chatTurnMeta{
			ToolSteps: steps,
			RAGHits:   ragHits,
			Provider:  provider,
			Model:     modelName,
		})
		msgs = append(msgs, model.AiChatMessage{
			SessionID: sess.ID,
			Role:      "assistant",
			Content:   assistantContent,
			MetaJSON:  string(meta),
			CreatedAt: now.Add(time.Millisecond),
		})
	}
	if len(msgs) > 0 {
		if err := s.db.WithContext(ctx).Create(&msgs).Error; err != nil {
			return sess.ID, err
		}
	}

	updates := map[string]any{
		"last_message_at": now,
		"project_id":      req.ProjectID,
		"cluster_id":      req.ClusterID,
		"provider":        strings.TrimSpace(req.Provider),
		"enable_write":    req.EnableWrite,
		"updated_at":      now,
	}
	if req.EnableTools != nil {
		updates["enable_tools"] = *req.EnableTools
	}
	if sess.Title == "新对话" && userContent != "" {
		updates["title"] = truncateRunes(userContent, 40)
	}
	_ = s.db.WithContext(ctx).Model(sess).Updates(updates).Error
	return sess.ID, nil
}

func truncateRunes(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max]) + "…"
}
