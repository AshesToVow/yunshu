package apperror

import (
	"errors"
	"net/http"
	"testing"
)

func TestMarkLoggedSkipsBoundaryDuplicate(t *testing.T) {
	root := NewBiz(http.StatusInternalServerError, 50001, "InternalError", "boom")
	wrapped := MarkLogged(root)
	if !AlreadyLogged(wrapped) {
		t.Fatal("expected AlreadyLogged")
	}
	app, ok := IsAppError(wrapped)
	if !ok || app.HTTPStatus() != http.StatusInternalServerError {
		t.Fatalf("IsAppError: ok=%v status=%d", ok, app.HTTPStatus())
	}
}

func TestIsAppErrorUnwraps(t *testing.T) {
	root := NewBiz(http.StatusBadRequest, 11020, "BadRequest", "bad")
	wrapped := errors.Join(MarkLogged(root), errors.New("noise"))
	app, ok := IsAppError(wrapped)
	if !ok || app.HTTPStatus() != http.StatusBadRequest {
		t.Fatalf("got ok=%v status=%d", ok, app.HTTPStatus())
	}
}
