package alert_test

import (
	"context"
	"testing"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/repository"
	"yunshu/internal/service/alert"
)

type fakeSilenceRepo struct {
	rows []model.AlertSilence
	next uint
}

func (f *fakeSilenceRepo) DisableExpired(ctx context.Context, now time.Time) error {
	for i := range f.rows {
		if f.rows[i].Enabled && f.rows[i].EndsAt.Before(now) {
			f.rows[i].Enabled = false
		}
	}
	return nil
}

func (f *fakeSilenceRepo) ListActiveAt(ctx context.Context, at time.Time) ([]model.AlertSilence, error) {
	out := make([]model.AlertSilence, 0)
	for _, r := range f.rows {
		if r.Enabled && !r.StartsAt.After(at) && r.EndsAt.After(at) {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeSilenceRepo) ListEnabledUnexpiredByProject(ctx context.Context, projectID uint, now time.Time) ([]model.AlertSilence, error) {
	out := make([]model.AlertSilence, 0)
	for _, r := range f.rows {
		if r.ProjectID == projectID && r.Enabled && r.EndsAt.After(now) {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeSilenceRepo) Create(ctx context.Context, row *model.AlertSilence) error {
	f.next++
	row.ID = f.next
	f.rows = append(f.rows, *row)
	return nil
}

func (f *fakeSilenceRepo) GetByID(ctx context.Context, id uint) (*model.AlertSilence, error) {
	for i := range f.rows {
		if f.rows[i].ID == id {
			cp := f.rows[i]
			return &cp, nil
		}
	}
	return nil, constants.ErrAlertSilenceNotFound
}

func (f *fakeSilenceRepo) Save(ctx context.Context, row *model.AlertSilence) error {
	for i := range f.rows {
		if f.rows[i].ID == row.ID {
			f.rows[i] = *row
			return nil
		}
	}
	return constants.ErrAlertSilenceNotFound
}

func (f *fakeSilenceRepo) Delete(ctx context.Context, id uint) error {
	for i := range f.rows {
		if f.rows[i].ID == id {
			f.rows = append(f.rows[:i], f.rows[i+1:]...)
			return nil
		}
	}
	return nil
}

func (f *fakeSilenceRepo) ListPaged(ctx context.Context, projectID uint, keyword string, offset, limit int) ([]model.AlertSilence, int64, error) {
	var filtered []model.AlertSilence
	for _, r := range f.rows {
		if projectID > 0 && r.ProjectID != projectID {
			continue
		}
		filtered = append(filtered, r)
	}
	total := int64(len(filtered))
	if offset >= len(filtered) {
		return []model.AlertSilence{}, total, nil
	}
	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[offset:end], total, nil
}

var _ repository.AlertSilenceRepo = (*fakeSilenceRepo)(nil)

func TestParseSilenceMatchersJSON(t *testing.T) {
	t.Parallel()
	ms, err := alert.ParseSilenceMatchersJSON(`[{"name":"alertname","value":"DiskFull","is_regex":false}]`)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 1 || ms[0].Name != "alertname" || ms[0].Value != "DiskFull" {
		t.Fatalf("unexpected matchers: %+v", ms)
	}
	if _, err := alert.ParseSilenceMatchersJSON(`[{"name":"","value":"x"}]`); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestLabelsMatchSilenceMatchers(t *testing.T) {
	t.Parallel()
	ms := []alert.SilenceMatcher{{Name: "severity", Value: "prod"}, {Name: "alertname", Value: "Disk.*", IsRegex: true}}
	if !alert.LabelsMatchSilenceMatchers(ms, map[string]string{"severity": "prod", "alertname": "DiskFull"}) {
		t.Fatal("expected match")
	}
	if alert.LabelsMatchSilenceMatchers(ms, map[string]string{"severity": "prod", "alertname": "CPUHigh"}) {
		t.Fatal("expected no match")
	}
	if !alert.LabelsMatchSilenceMatchers(nil, map[string]string{"a": "b"}) {
		t.Fatal("empty matchers should match all")
	}
}

func TestAlertSilenceService_CreateRejectsDuplicate(t *testing.T) {
	t.Parallel()
	now := time.Now()
	repo := &fakeSilenceRepo{rows: []model.AlertSilence{{
		ID: 1, ProjectID: 7, Name: "existing", Enabled: true,
		MatchersJSON: `[{"name":"alertname","value":"DiskFull","is_regex":false}]`,
		StartsAt:     now.Add(-time.Hour),
		EndsAt:       now.Add(2 * time.Hour),
	}}, next: 1}
	svc := alert.NewAlertSilenceService(repo)
	_, err := svc.Create(context.Background(), 1, alert.AlertSilenceUpsertRequest{
		ProjectID:    7,
		Name:         "dup",
		MatchersJSON: `[{"name":"alertname","value":"DiskFull","is_regex":false}]`,
		StartsAt:     now,
		EndsAt:       now.Add(time.Hour),
	})
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestAlertSilenceService_CreateAndList(t *testing.T) {
	t.Parallel()
	now := time.Now()
	repo := &fakeSilenceRepo{}
	svc := alert.NewAlertSilenceService(repo)
	row, err := svc.Create(context.Background(), 9, alert.AlertSilenceUpsertRequest{
		ProjectID:    3,
		Name:         "nightly",
		MatchersJSON: `[{"name":"severity","value":"staging","is_regex":false}]`,
		StartsAt:     now,
		EndsAt:       now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if row.ID == 0 || row.CreatedBy != 9 {
		t.Fatalf("bad row: %+v", row)
	}
	list, total, _, _, err := svc.List(context.Background(), alert.AlertSilenceListQuery{ProjectID: 3, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("list total=%d len=%d", total, len(list))
	}
}

func TestAlertSilenceService_FirstMatchingSilenceID(t *testing.T) {
	t.Parallel()
	now := time.Now()
	repo := &fakeSilenceRepo{rows: []model.AlertSilence{{
		ID: 42, Enabled: true,
		MatchersJSON: `[{"name":"alertname","value":"HighCPU","is_regex":false}]`,
		StartsAt:     now.Add(-time.Minute),
		EndsAt:       now.Add(time.Hour),
	}}, next: 42}
	svc := alert.NewAlertSilenceService(repo)
	id, ok, err := svc.FirstMatchingSilenceID(context.Background(), map[string]string{"alertname": "HighCPU"}, now)
	if err != nil || !ok || id != 42 {
		t.Fatalf("id=%d ok=%v err=%v", id, ok, err)
	}
}

func TestAlertSilenceService_Delete(t *testing.T) {
	t.Parallel()
	repo := &fakeSilenceRepo{rows: []model.AlertSilence{{ID: 5, Name: "x"}}, next: 5}
	svc := alert.NewAlertSilenceService(repo)
	if err := svc.Delete(context.Background(), 5); err != nil {
		t.Fatal(err)
	}
	if len(repo.rows) != 0 {
		t.Fatalf("expected empty repo, got %d", len(repo.rows))
	}
}
