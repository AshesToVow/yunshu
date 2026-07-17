package cronutil

import (
	"context"
	"strings"
	"time"

	"github.com/robfig/cron/v3"

	"yunshu/internal/pkg/constants"
)

var parser = cron.NewParser(
	cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
)

func parseSchedule(spec string) (cron.Schedule, error) {
	return parser.Parse(strings.TrimSpace(spec))
}

// ParseSchedule 解析 Cron 表达式（测试与高级场景）。
func ParseSchedule(spec string) (cron.Schedule, error) {
	return parseSchedule(spec)
}

// ValidateSpec 校验 Cron 表达式；空串合法。
func ValidateSpec(spec, fieldLabel string) error {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil
	}
	if _, err := parseSchedule(spec); err != nil {
		if fieldLabel == "" {
			fieldLabel = "cron_spec"
		}
		return constants.ErrBadRequestWithMsg("无效的 " + fieldLabel + "：" + err.Error())
	}
	return nil
}

// ShouldRunAfterLast 距上次执行是否已到下次 Cron 触发时刻（云到期等场景）。
func ShouldRunAfterLast(spec string, last time.Time, hasLast bool, now time.Time) bool {
	sched, err := parseSchedule(spec)
	if err != nil {
		return false
	}
	if !hasLast || last.IsZero() {
		return true
	}
	next := sched.Next(last)
	return !now.Before(next)
}

// ShouldRunWithDayAnchor 定时任务到点判断：首次启用等到下一次 Cron 触发，同窗口不重复跑。
func ShouldRunWithDayAnchor(spec string, last *time.Time, now time.Time) bool {
	sched, err := parseSchedule(spec)
	if err != nil {
		return false
	}
	if last == nil || last.IsZero() {
		dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		dueToday := sched.Next(dayStart.Add(-time.Second))
		if now.Before(dueToday) {
			return false
		}
		// 首次启用且处于当天计划时刻窗口内：允许执行
		if now.Sub(dueToday) <= 3*time.Minute {
			return true
		}
		// 首次启用但已错过当天计划时刻：不补跑，等下一次 cron
		nextDue := sched.Next(dueToday)
		if now.Before(nextDue) {
			return false
		}
		last = &dueToday
	}
	var anchor time.Time
	anchor = *last
	dueAt := sched.Next(anchor)
	if now.Before(dueAt) {
		return false
	}
	if !last.Before(dueAt) {
		return false
	}
	return true
}

// RunWorker 按 spec 周期性执行 job，阻塞至 ctx 取消。spec 无效且 fallbackSpec 非空时尝试 fallback。
func RunWorker(ctx context.Context, spec string, job func(), fallbackSpec string) {
	spec = strings.TrimSpace(spec)
	c := cron.New(cron.WithSeconds())
	wrapped := func() {
		if ctx.Err() != nil {
			return
		}
		job()
	}
	if _, err := c.AddFunc(spec, wrapped); err != nil {
		fallbackSpec = strings.TrimSpace(fallbackSpec)
		if fallbackSpec == "" || fallbackSpec == spec {
			return
		}
		if _, err2 := c.AddFunc(fallbackSpec, wrapped); err2 != nil {
			return
		}
	}
	c.Start()
	<-ctx.Done()
	stopCtx := c.Stop()
	<-stopCtx.Done()
}
