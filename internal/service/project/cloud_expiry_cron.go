package project

import (
	"strings"
	"time"

	"github.com/robfig/cron/v3"

	"yunshu/internal/pkg/constants"
)

// cloudExpiryCronParser 支持五段/六段（可选秒）、以及 @every 等描述符（与 robfig/cron v3 一致）。
var cloudExpiryCronParser = cron.NewParser(
	cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
)

func parseCloudExpiryCronSchedule(spec string) (cron.Schedule, error) {
	return cloudExpiryCronParser.Parse(strings.TrimSpace(spec))
}

// ValidateCloudExpiryCronSpec 校验云到期规则的 Cron 表达式语法；空串合法（启用定时时由业务层要求必填）。
func ValidateCloudExpiryCronSpec(spec string) error {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil
	}
	if _, err := parseCloudExpiryCronSchedule(spec); err != nil {
		return constants.ErrBadRequestWithMsg("无效的 eval_cron_spec：" + err.Error())
	}
	return nil
}

// ShouldEvalCloudExpiryByCron 判断距上次评估是否已到 cron 下次触发时刻。
func ShouldEvalCloudExpiryByCron(spec string, last time.Time, hasLast bool, now time.Time) bool {
	sched, err := parseCloudExpiryCronSchedule(spec)
	if err != nil {
		return false
	}
	if !hasLast || last.IsZero() {
		return true
	}
	next := sched.Next(last)
	return !now.Before(next)
}
