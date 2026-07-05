package mysqlbackup

import (
	"context"
	"encoding/json"
	"strings"

	"yunshu/internal/model"
)

type MysqlBackupNotifyUserItem struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Email    string `json:"email,omitempty"`
}

func parseNotifyUserIDs(raw string) []uint {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}
	var ids []uint
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil
	}
	return dedupeNotifyUserIDs(ids)
}

func marshalNotifyUserIDs(ids []uint) string {
	ids = dedupeNotifyUserIDs(ids)
	if len(ids) == 0 {
		return "[]"
	}
	bs, err := json.Marshal(ids)
	if err != nil {
		return "[]"
	}
	return string(bs)
}

func dedupeNotifyUserIDs(ids []uint) []uint {
	seen := make(map[uint]struct{}, len(ids))
	out := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func (s *MysqlBackupService) resolveNotifyUserBriefs(ctx context.Context, ids []uint) []MysqlBackupNotifyUserItem {
	if s == nil || s.userRepo == nil || len(ids) == 0 {
		return nil
	}
	users, err := s.userRepo.ListByIDs(ctx, ids)
	if err != nil || len(users) == 0 {
		return nil
	}
	byID := make(map[uint]model.User, len(users))
	for _, u := range users {
		byID[u.ID] = u
	}
	out := make([]MysqlBackupNotifyUserItem, 0, len(ids))
	for _, id := range ids {
		u, ok := byID[id]
		if !ok {
			continue
		}
		item := MysqlBackupNotifyUserItem{
			ID:       u.ID,
			Username: strings.TrimSpace(u.Username),
			Nickname: strings.TrimSpace(u.Nickname),
		}
		if u.Email != nil {
			item.Email = strings.TrimSpace(*u.Email)
		}
		out = append(out, item)
	}
	return out
}

func (s *MysqlBackupService) resolveNotifyEmails(ctx context.Context, ids []uint) []string {
	briefs := s.resolveNotifyUserBriefs(ctx, ids)
	if len(briefs) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	out := make([]string, 0, len(briefs))
	for _, b := range briefs {
		e := strings.ToLower(strings.TrimSpace(b.Email))
		if e == "" || !strings.Contains(e, "@") {
			continue
		}
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		out = append(out, e)
	}
	return out
}
