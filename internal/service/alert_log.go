package service

import "yunshu/internal/pkg/logutil"

func alertLog() *logutil.Component {
	return logutil.Worker("alert")
}
