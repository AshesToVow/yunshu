package router

import "yunshu/internal/bootstrap"

func (d *RouteDeps) App() *bootstrap.App {
	if d == nil {
		return nil
	}
	return d.app
}
