package alert

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"sync"
	"time"

	"yunshu/internal/interfaces"
	bizerrors "yunshu/internal/pkg/errors"
)

// ReceiverGroupCache 接收组缓存
type ReceiverGroupCache struct {
	repo     interfaces.AlertReceiverGroupRepository
	mu       sync.RWMutex
	groups   map[uint]*CachedReceiverGroup
	lastLoad time.Time
}

// CachedReceiverGroup 缓存的接收组
type CachedReceiverGroup struct {
	ID                     uint
	ProjectID              uint
	Name                   string
	ChannelIDs             []uint
	EmailRecipients        []string
	ActiveTimeStart        *string
	ActiveTimeEnd          *string
	Weekdays               []int
	EscalationLevel        int
	EscalationDelaySeconds int
}

// IsActiveNow 检查接收组当前是否生效
func (g *CachedReceiverGroup) IsActiveNow() bool {
	now := time.Now()

	if len(g.Weekdays) > 0 {
		weekday := int(now.Weekday())
		found := slices.Contains(g.Weekdays, weekday)
		if !found {
			return false
		}
	}

	if g.ActiveTimeStart != nil && g.ActiveTimeEnd != nil {
		currentTime := now.Format("15:04")
		if currentTime < *g.ActiveTimeStart || currentTime > *g.ActiveTimeEnd {
			return false
		}
	}

	return true
}

// NewReceiverGroupCache 创建接收组缓存
func NewReceiverGroupCache(repo interfaces.AlertReceiverGroupRepository) *ReceiverGroupCache {
	cache := &ReceiverGroupCache{
		repo:   repo,
		groups: make(map[uint]*CachedReceiverGroup),
	}
	_ = cache.Refresh()
	return cache
}

// Get 获取接收组
func (c *ReceiverGroupCache) Get(id uint) (*CachedReceiverGroup, error) {
	c.mu.RLock()
	if group, ok := c.groups[id]; ok {
		c.mu.RUnlock()
		return group, nil
	}
	c.mu.RUnlock()

	if err := c.Refresh(); err != nil {
		return nil, bizerrors.Pass(context.Background(), "alert.receiver", "Get", err)
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	if group, ok := c.groups[id]; ok {
		return group, nil
	}
	return nil, nil
}

// Refresh 刷新缓存
func (c *ReceiverGroupCache) Refresh() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if time.Since(c.lastLoad) < 30*time.Second && len(c.groups) > 0 {
		return nil
	}

	groups, err := c.repo.ListEnabled(context.Background())
	if err != nil {
		return bizerrors.Pass(context.Background(), "alert.receiver", "Refresh", err)
	}

	newGroups := make(map[uint]*CachedReceiverGroup, len(groups))
	for _, g := range groups {
		cached := &CachedReceiverGroup{
			ID:                     g.ID,
			ProjectID:              g.ProjectID,
			Name:                   g.Name,
			ChannelIDs:             parseUintSliceJSON(g.ChannelIDsJSON),
			EmailRecipients:        parseStringSliceJSON(g.EmailRecipientsJSON),
			ActiveTimeStart:        g.ActiveTimeStart,
			ActiveTimeEnd:          g.ActiveTimeEnd,
			Weekdays:               parseIntSliceJSON(g.WeekdaysJSON),
			EscalationLevel:        g.EscalationLevel,
			EscalationDelaySeconds: g.EscalationDelaySeconds,
		}
		newGroups[g.ID] = cached
	}

	c.groups = newGroups
	c.lastLoad = time.Now()
	return nil
}

// Invalidate 使缓存失效
func (c *ReceiverGroupCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.groups = make(map[uint]*CachedReceiverGroup)
	c.lastLoad = time.Time{}
}

func parseStringSliceJSON(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}
	var out []string
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func parseIntSliceJSON(raw string) []int {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}
	var out []int
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}
