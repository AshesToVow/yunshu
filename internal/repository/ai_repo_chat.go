package repository

import (
	"context"
	"time"

	"yunshu/internal/model"
)

// --- KB Chunk ---

func (r *AiRepository) DeleteChunksByDocumentID(ctx context.Context, documentID uint) error {
	return r.dbq(ctx).Where("document_id = ?", documentID).Delete(&model.AiKbChunk{}).Error
}

func (r *AiRepository) CreateChunk(ctx context.Context, ch *model.AiKbChunk) error {
	return r.dbq(ctx).Create(ch).Error
}

func (r *AiRepository) ListRecentChunks(ctx context.Context, limit int) ([]model.AiKbChunk, error) {
	var chunks []model.AiKbChunk
	q := r.dbq(ctx).Order("id DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&chunks).Error
	return chunks, err
}

func (r *AiRepository) ListChunksMissingEmbedding(ctx context.Context, limit int) ([]model.AiKbChunk, error) {
	var chunks []model.AiKbChunk
	q := r.dbq(ctx).
		Where("embedding IS NULL OR LENGTH(embedding) = 0").
		Order("id ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&chunks).Error
	return chunks, err
}

func (r *AiRepository) UpdateChunkEmbedding(ctx context.Context, chunkID uint, embedding []byte) error {
	return r.dbq(ctx).Model(&model.AiKbChunk{}).Where("id = ?", chunkID).
		Update("embedding", embedding).Error
}

// --- Approval ---

func (r *AiRepository) CreateApproval(ctx context.Context, row *model.AiToolApproval) error {
	return r.dbq(ctx).Create(row).Error
}

func (r *AiRepository) DeleteApproval(ctx context.Context, id uint) error {
	return r.dbq(ctx).Delete(&model.AiToolApproval{}, id).Error
}

func (r *AiRepository) ListApprovals(ctx context.Context, p AiApprovalListParams) ([]model.AiToolApproval, int64, error) {
	q := r.dbq(ctx).Model(&model.AiToolApproval{})
	if p.Status != "" {
		q = q.Where("status = ?", p.Status)
	}
	if p.RestrictUser {
		q = q.Where("user_id = ?", p.UserID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.AiToolApproval
	err := q.Order("id desc").Offset(p.Offset).Limit(p.Limit).Find(&list).Error
	return list, total, err
}

func (r *AiRepository) GetApprovalByID(ctx context.Context, id uint) (*model.AiToolApproval, error) {
	var row model.AiToolApproval
	if err := r.firstByID(ctx, &row, id); err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AiRepository) SaveApproval(ctx context.Context, row *model.AiToolApproval) error {
	return r.dbq(ctx).Save(row).Error
}

func (r *AiRepository) ClaimApprovalExecution(ctx context.Context, id uint, fromStatuses []string) (int64, error) {
	res := r.dbq(ctx).Model(&model.AiToolApproval{}).
		Where("id = ? AND status IN ?", id, fromStatuses).
		Updates(map[string]any{"status": "executing", "updated_at": time.Now()})
	return res.RowsAffected, res.Error
}

func (r *AiRepository) UpdateApprovalFields(ctx context.Context, id uint, fields map[string]any) error {
	return r.dbq(ctx).Model(&model.AiToolApproval{}).Where("id = ?", id).Updates(fields).Error
}

// --- Chat Session / Message ---

func (r *AiRepository) CreateSession(ctx context.Context, row *model.AiChatSession) error {
	return r.dbq(ctx).Create(row).Error
}

func (r *AiRepository) ListSessions(ctx context.Context, p AiSessionListParams) ([]model.AiChatSession, int64, error) {
	q := r.dbq(ctx).Model(&model.AiChatSession{}).Where("user_id = ?", p.UserID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.AiChatSession
	err := q.Order("COALESCE(last_message_at, updated_at) DESC, id DESC").
		Offset(p.Offset).Limit(p.Limit).Find(&rows).Error
	return rows, total, err
}

func (r *AiRepository) GetSessionByUser(ctx context.Context, userID, sessionID uint) (*model.AiChatSession, error) {
	var sess model.AiChatSession
	err := r.dbq(ctx).Where("id = ? AND user_id = ?", sessionID, userID).First(&sess).Error
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func (r *AiRepository) UpdateSession(ctx context.Context, id uint, updates map[string]any) error {
	return r.dbq(ctx).Model(&model.AiChatSession{}).Where("id = ?", id).Updates(updates).Error
}

func (r *AiRepository) DeleteSessionCascade(ctx context.Context, sessionID uint) error {
	return r.Transaction(ctx, func(tx AiRepo) error {
		repo := tx.(*AiRepository)
		if err := repo.dbq(ctx).Where("session_id = ?", sessionID).Delete(&model.AiChatMessage{}).Error; err != nil {
			return err
		}
		return repo.dbq(ctx).Delete(&model.AiChatSession{}, sessionID).Error
	})
}

func (r *AiRepository) ClearSessionMessages(ctx context.Context, sessionID uint, sessionUpdates map[string]any) error {
	return r.Transaction(ctx, func(tx AiRepo) error {
		repo := tx.(*AiRepository)
		if err := repo.dbq(ctx).Where("session_id = ?", sessionID).Delete(&model.AiChatMessage{}).Error; err != nil {
			return err
		}
		return repo.dbq(ctx).Model(&model.AiChatSession{}).Where("id = ?", sessionID).Updates(sessionUpdates).Error
	})
}

func (r *AiRepository) CountSessionMessages(ctx context.Context, sessionIDs []uint) (map[uint]int64, error) {
	out := map[uint]int64{}
	if len(sessionIDs) == 0 {
		return out, nil
	}
	type cntRow struct {
		SessionID uint
		Cnt       int64
	}
	var counts []cntRow
	err := r.dbq(ctx).Model(&model.AiChatMessage{}).
		Select("session_id, COUNT(*) AS cnt").
		Where("session_id IN ?", sessionIDs).
		Group("session_id").
		Scan(&counts).Error
	if err != nil {
		return nil, err
	}
	for _, c := range counts {
		out[c.SessionID] = c.Cnt
	}
	return out, nil
}

func (r *AiRepository) ListSessionMessages(ctx context.Context, sessionID uint, limit int) ([]model.AiChatMessage, error) {
	var msgs []model.AiChatMessage
	q := r.dbq(ctx).Where("session_id = ?", sessionID).Order("id ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&msgs).Error
	return msgs, err
}

func (r *AiRepository) CreateMessages(ctx context.Context, msgs []model.AiChatMessage) error {
	if len(msgs) == 0 {
		return nil
	}
	return r.dbq(ctx).Create(&msgs).Error
}

func (r *AiRepository) CountChatSessions(ctx context.Context) (int64, error) {
	return r.countModel(ctx, &model.AiChatSession{})
}

// --- Investigation ---

func (r *AiRepository) CreateInvestigation(ctx context.Context, row *model.AiInvestigation) error {
	return r.dbq(ctx).Create(row).Error
}

func (r *AiRepository) SaveInvestigation(ctx context.Context, row *model.AiInvestigation) error {
	return r.dbq(ctx).Save(row).Error
}

func (r *AiRepository) ListInvestigations(ctx context.Context, p AiInvestigationListParams) ([]model.AiInvestigation, int64, error) {
	q := r.dbq(ctx).Model(&model.AiInvestigation{}).Where("user_id = ?", p.UserID)
	if p.Kind != "" {
		q = q.Where("kind = ?", p.Kind)
	}
	if p.Status != "" {
		q = q.Where("status = ?", p.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.AiInvestigation
	err := q.Order("id desc").Offset(p.Offset).Limit(p.Limit).Find(&list).Error
	return list, total, err
}

func (r *AiRepository) GetInvestigationByUser(ctx context.Context, userID, id uint) (*model.AiInvestigation, error) {
	var row model.AiInvestigation
	err := r.dbq(ctx).Where("id = ? AND user_id = ?", id, userID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}
