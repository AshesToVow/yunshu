package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"yunshu/internal/handler"
	"yunshu/internal/middleware"
	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/repository"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type silenceRepoFake struct {
	rows []model.AlertSilence
	next uint
}

func (f *silenceRepoFake) DisableExpired(ctx context.Context, now time.Time) error { return nil }

func (f *silenceRepoFake) ListActiveAt(ctx context.Context, at time.Time) ([]model.AlertSilence, error) {
	return nil, nil
}

func (f *silenceRepoFake) ListEnabledUnexpiredByProject(ctx context.Context, projectID uint, now time.Time) ([]model.AlertSilence, error) {
	return nil, nil
}

func (f *silenceRepoFake) Create(ctx context.Context, row *model.AlertSilence) error {
	f.next++
	row.ID = f.next
	f.rows = append(f.rows, *row)
	return nil
}

func (f *silenceRepoFake) CreateBatch(ctx context.Context, rows []*model.AlertSilence) error {
	for _, row := range rows {
		if err := f.Create(ctx, row); err != nil {
			return err
		}
	}
	return nil
}

func (f *silenceRepoFake) GetByID(ctx context.Context, id uint) (*model.AlertSilence, error) {
	for i := range f.rows {
		if f.rows[i].ID == id {
			cp := f.rows[i]
			return &cp, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *silenceRepoFake) Save(ctx context.Context, row *model.AlertSilence) error { return nil }

func (f *silenceRepoFake) Delete(ctx context.Context, id uint) error {
	for i := range f.rows {
		if f.rows[i].ID == id {
			f.rows = append(f.rows[:i], f.rows[i+1:]...)
			return nil
		}
	}
	return nil
}

func (f *silenceRepoFake) ListPaged(ctx context.Context, projectID uint, projectIDs []uint, keyword string, offset, limit int) ([]model.AlertSilence, int64, error) {
	return append([]model.AlertSilence{}, f.rows...), int64(len(f.rows)), nil
}

var _ repository.AlertSilenceRepo = (*silenceRepoFake)(nil)

type memberRepoFake struct{}

func (m *memberRepoFake) ListByProject(ctx context.Context, projectID uint) ([]model.ProjectMember, error) {
	return nil, nil
}

func (m *memberRepoFake) ListUserIDsByProject(ctx context.Context, projectID uint) ([]uint, error) {
	return nil, nil
}

func (m *memberRepoFake) DeleteByUserID(ctx context.Context, userID uint) error { return nil }

func (m *memberRepoFake) ListRolesByUserAndProjectIDs(ctx context.Context, userID uint, projectIDs []uint) (map[uint]string, error) {
	return nil, nil
}

func (m *memberRepoFake) ListProjectIDsByUser(ctx context.Context, userID uint) ([]uint, error) {
	return nil, nil
}

func (m *memberRepoFake) GetByID(ctx context.Context, id uint) (*model.ProjectMember, error) {
	return nil, gorm.ErrRecordNotFound
}

func (m *memberRepoFake) GetByProjectAndUser(ctx context.Context, projectID uint, userID uint) (*model.ProjectMember, error) {
	if projectID == 1 && userID == 11 {
		return &model.ProjectMember{ProjectID: projectID, UserID: userID}, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *memberRepoFake) Create(ctx context.Context, row *model.ProjectMember) error { return nil }
func (m *memberRepoFake) Save(ctx context.Context, row *model.ProjectMember) error   { return nil }
func (m *memberRepoFake) DeleteByID(ctx context.Context, id uint) error              { return nil }
func (m *memberRepoFake) DeleteByProject(ctx context.Context, projectID uint) error  { return nil }

func (m *memberRepoFake) ListDisplayByProject(ctx context.Context, projectID uint) ([]repository.ProjectMemberListRow, error) {
	return nil, nil
}

var _ repository.ProjectMemberRepo = (*memberRepoFake)(nil)

func newSilenceHandlerEngine(t *testing.T, repo *silenceRepoFake) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	silenceSvc := service.NewAlertSilenceService(repo, &memberRepoFake{})
	h := handler.NewAlertPlatformHandler(nil, silenceSvc, nil, nil, nil, nil, nil)
	eng := gin.New()
	eng.Use(middleware.ErrorHandler())
	eng.GET("/silences", h.ListSilences)
	eng.POST("/silences", func(c *gin.Context) {
		c.Set(auth.ContextUserKey, &auth.CurrentUser{ID: 11})
		h.CreateSilence(c)
	})
	eng.DELETE("/silences/:id", func(c *gin.Context) {
		c.Set(auth.ContextUserKey, &auth.CurrentUser{ID: 11})
		h.DeleteSilence(c)
	})
	return eng
}

func TestAlertPlatformSilence_ListCreateDelete(t *testing.T) {
	t.Parallel()
	repo := &silenceRepoFake{}
	eng := newSilenceHandlerEngine(t, repo)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/silences?page=1&page_size=10", nil)
	eng.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", w.Code, w.Body.String())
	}
	var listBody struct {
		Code int `json:"code"`
		Data struct {
			Total int64 `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &listBody); err != nil {
		t.Fatal(err)
	}
	if listBody.Code != http.StatusOK || listBody.Data.Total != 0 {
		t.Fatalf("unexpected list: %+v body=%s", listBody, w.Body.String())
	}

	now := time.Now().UTC()
	payload := map[string]any{
		"project_id":    1,
		"name":          "handler-silence",
		"matchers_json": `[{"name":"alertname","value":"X","is_regex":false}]`,
		"starts_at":     now.Format(time.RFC3339),
		"ends_at":       now.Add(2 * time.Hour).Format(time.RFC3339),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/silences", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	eng.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create status=%d body=%s", w.Code, w.Body.String())
	}
	if len(repo.rows) != 1 || repo.rows[0].CreatedBy != 11 {
		t.Fatalf("repo after create: %+v", repo.rows)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/silences/1", nil)
	eng.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", w.Code, w.Body.String())
	}
	if len(repo.rows) != 0 {
		t.Fatalf("expected deleted, got %+v", repo.rows)
	}
}
