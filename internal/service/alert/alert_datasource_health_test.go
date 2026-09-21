package alert

import (
	"testing"
	"time"

	"yunshu/internal/model"
)

func TestIsDatasourceHealthBlocking(t *testing.T) {
	t.Parallel()
	now := time.Now()
	ago := now.Add(-5 * time.Minute)
	stale := now.Add(-30 * time.Minute)

	if IsDatasourceHealthBlocking(nil, now) {
		t.Fatal("nil should not block")
	}
	if IsDatasourceHealthBlocking(&model.AlertDatasource{
		LastHealthStatus: model.DatasourceHealthOK,
		LastHealthAt:     &ago,
	}, now) {
		t.Fatal("ok should not block")
	}
	if !IsDatasourceHealthBlocking(&model.AlertDatasource{
		LastHealthStatus: model.DatasourceHealthDown,
		LastHealthAt:     &ago,
	}, now) {
		t.Fatal("recent down should block")
	}
	if IsDatasourceHealthBlocking(&model.AlertDatasource{
		LastHealthStatus: model.DatasourceHealthDown,
		LastHealthAt:     &stale,
	}, now) {
		t.Fatal("stale down should not block")
	}
}
