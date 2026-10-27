package auth

import (
	"context"
	"errors"
	"testing"
)

func TestWSRevalidatorContextRoundTrip(t *testing.T) {
	if _, ok := WSRevalidatorFromContext(context.Background()); ok {
		t.Fatal("expected no revalidator on empty context")
	}
	if _, ok := WSRevalidatorFromContext(nil); ok {
		t.Fatal("expected no revalidator on nil context")
	}

	sentinel := errors.New("revoked")
	ctx := WithWSRevalidator(context.Background(), func(context.Context) error { return sentinel })
	fn, ok := WSRevalidatorFromContext(ctx)
	if !ok {
		t.Fatal("expected revalidator")
	}
	if err := fn(context.Background()); !errors.Is(err, sentinel) {
		t.Fatalf("err=%v, want sentinel", err)
	}

	if got := WithWSRevalidator(ctx, nil); got != ctx {
		t.Fatal("nil revalidator should leave context unchanged")
	}
}
