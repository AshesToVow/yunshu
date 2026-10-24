package router

import (
	"yunshu/internal/handler"
	"yunshu/internal/routedeps"
)

func (d *RouteDeps) KafkamgmtHandler() *handler.KafkamgmtHandler {
	if d == nil {
		return nil
	}
	return d.kafkamgmtHandler
}

var _ KafkamgmtRouteDeps = (*RouteDeps)(nil)

var _ routedeps.KafkamgmtRouteDeps = (*RouteDeps)(nil)
