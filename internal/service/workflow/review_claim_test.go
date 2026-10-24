package workflow

import (
	"errors"
	"testing"

	bizerrors "yunshu/internal/pkg/errors"
)

func TestErrStepConflictIsBizError(t *testing.T) {
	t.Parallel()
	if errStepConflict == nil {
		t.Fatal("errStepConflict nil")
	}
	var be *bizerrors.BizError
	if !errors.As(errStepConflict, &be) {
		t.Fatalf("want BizError, got %T: %v", errStepConflict, errStepConflict)
	}
	if be.Message == "" {
		t.Fatal("empty conflict message")
	}
}
