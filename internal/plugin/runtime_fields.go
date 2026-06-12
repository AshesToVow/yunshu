package plugin

import "yunshu/internal/service"

// K8sRuntimeSvc 返回 K8s 运行时服务（由 router 注入）。
func (rt *Runtime) K8sRuntimeSvc() *service.K8sRuntimeService {
	if rt == nil {
		return nil
	}
	svc, _ := rt.K8sRuntime.(*service.K8sRuntimeService)
	return svc
}

// MysqlBackupSvc 返回 MySQL 备份服务（由 router 注入）。
func (rt *Runtime) MysqlBackupSvc() *service.MysqlBackupService {
	if rt == nil {
		return nil
	}
	svc, _ := rt.MysqlBackup.(*service.MysqlBackupService)
	return svc
}
