package cronutil

import (
	"testing"

	"github.com/robfig/cron/v3"
)

func TestNormalizeCronSpecForSeconds(t *testing.T) {
	t.Parallel()
	if got := NormalizeCronSpecForSeconds("*/15 * * * *"); got != "0 */15 * * * *" {
		t.Fatalf("got %q", got)
	}
	if got := NormalizeCronSpecForSeconds("0 */5 * * * *"); got != "0 */5 * * * *" {
		t.Fatalf("6-field unchanged: %q", got)
	}
}

func TestCronWithSecondsAcceptsNormalizedFiveField(t *testing.T) {
	t.Parallel()
	c := cron.New(cron.WithSeconds())
	_, err := c.AddFunc("*/15 * * * *", func() {})
	if err == nil {
		t.Fatal("expected raw 5-field spec to fail under WithSeconds()")
	}
	_, err = c.AddFunc(NormalizeCronSpecForSeconds("*/15 * * * *"), func() {})
	if err != nil {
		t.Fatalf("normalized 5-field should work: %v", err)
	}
}
