package dictconfig

import (
	"context"
	"testing"
)

func TestResolvePodDebugImageFallback(t *testing.T) {
	t.Parallel()
	got := ResolvePodDebugImage(context.Background(), nil)
	if got != DefaultPodDebugImage {
		t.Fatalf("got %q want %q", got, DefaultPodDebugImage)
	}
}
