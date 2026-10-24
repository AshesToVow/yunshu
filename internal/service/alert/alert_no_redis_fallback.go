package alert

import (
	"encoding/json"
	"strings"
	"sync"
	"time"
)

type localGroupSnap struct {
	count      int64
	firstSeen  time.Time
	lastSent   time.Time
	lastDigest string
}

type localEscPending struct {
	raw string
	due int64
}

// processLocalAlertState 单进程降级：Redis 不可用时仍做分组节流与升级排队。
type processLocalAlertState struct {
	mu       sync.Mutex
	groups   map[string]*localGroupSnap
	esc      map[string]*localEscPending
	escLevel map[string]int
	escLock  map[string]time.Time
}

func (s *AlertService) processLocal() *processLocalAlertState {
	s.localMu.Lock()
	defer s.localMu.Unlock()
	if s.local == nil {
		s.local = &processLocalAlertState{
			groups:   map[string]*localGroupSnap{},
			esc:      map[string]*localEscPending{},
			escLevel: map[string]int{},
			escLock:  map[string]time.Time{},
		}
	}
	return s.local
}

func (s *AlertService) peekLocalFiringGroupTiming(groupKey, labelsDigest string) (bool, string, int64, string, string) {
	st := s.processLocal()
	st.mu.Lock()
	defer st.mu.Unlock()
	now := time.Now().UTC()
	g := st.groups[groupKey]
	if g == nil {
		g = &localGroupSnap{firstSeen: now}
		st.groups[groupKey] = g
	}
	g.count++
	first := g.firstSeen.UTC().Format(time.RFC3339)
	lastSent := ""
	if !g.lastSent.IsZero() {
		lastSent = g.lastSent.UTC().Format(time.RFC3339)
	}
	should, reason := evaluateFiringGroupTiming(s.cfg, now, first, lastSent, g.lastDigest, labelsDigest)
	return should, reason, g.count, first, now.Format(time.RFC3339)
}

func (s *AlertService) commitLocalFiringGroupTiming(groupKey, labelsDigest string) {
	st := s.processLocal()
	st.mu.Lock()
	defer st.mu.Unlock()
	g := st.groups[groupKey]
	if g == nil {
		g = &localGroupSnap{firstSeen: time.Now().UTC()}
		st.groups[groupKey] = g
	}
	g.lastSent = time.Now().UTC()
	if d := strings.TrimSpace(labelsDigest); d != "" {
		g.lastDigest = d
	}
}

func (s *AlertService) scheduleLocalEscalation(fp string, raw []byte, due int64) {
	st := s.processLocal()
	st.mu.Lock()
	defer st.mu.Unlock()
	if cur, ok := st.esc[fp]; ok && cur.due > 0 && cur.due <= due {
		return
	}
	st.esc[fp] = &localEscPending{raw: string(raw), due: due}
}

func (s *AlertService) loadLocalEscalation(fp string) *escalationPendingEnvelope {
	st := s.processLocal()
	st.mu.Lock()
	raw := ""
	if cur, ok := st.esc[fp]; ok {
		raw = cur.raw
	}
	st.mu.Unlock()
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var env escalationPendingEnvelope
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		return nil
	}
	if env.Labels == nil {
		env.Labels = map[string]string{}
	}
	if env.Outgoing == nil {
		env.Outgoing = map[string]any{}
	}
	return &env
}

func (s *AlertService) clearLocalEscalation(fp string, clearLevel bool) {
	st := s.processLocal()
	st.mu.Lock()
	defer st.mu.Unlock()
	delete(st.esc, fp)
	delete(st.escLock, fp)
	if clearLevel {
		delete(st.escLevel, fp)
	}
}

func (s *AlertService) listLocalDueEscalation(now time.Time) []string {
	st := s.processLocal()
	st.mu.Lock()
	defer st.mu.Unlock()
	out := make([]string, 0)
	limit := now.Unix()
	for fp, item := range st.esc {
		if item == nil || item.due <= 0 || item.due > limit {
			continue
		}
		out = append(out, fp)
		if len(out) >= 50 {
			break
		}
	}
	return out
}

func (s *AlertService) localEscalationLevel(fp string) int {
	st := s.processLocal()
	st.mu.Lock()
	defer st.mu.Unlock()
	return normalizeEscalationLevel(st.escLevel[fp])
}

func (s *AlertService) setLocalEscalationLevel(fp string, level int) {
	st := s.processLocal()
	st.mu.Lock()
	defer st.mu.Unlock()
	st.escLevel[fp] = normalizeEscalationLevel(level)
}

func (s *AlertService) tryLocalEscalationLock(fp string) bool {
	st := s.processLocal()
	st.mu.Lock()
	defer st.mu.Unlock()
	now := time.Now()
	if until, ok := st.escLock[fp]; ok && until.After(now) {
		return false
	}
	st.escLock[fp] = now.Add(30 * time.Second)
	return true
}
