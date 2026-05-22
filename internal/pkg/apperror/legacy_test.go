package apperror

import (
	"errors"
	"net/http"
	"testing"
)

func TestLegacyReasonOnNewBiz(t *testing.T) {
	err := NewBiz(http.StatusNotFound, 40401, "NotFound", "missing")
	var app *AppError
	if !errors.As(err, &app) || app.Reason != "NotFound" {
		t.Fatalf("reason=%q", app.Reason)
	}
}
