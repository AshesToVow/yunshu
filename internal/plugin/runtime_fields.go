package plugin

import (
	"yunshu/internal/service"
	"yunshu/internal/service/cicd"
	dbmgmtsvc "yunshu/internal/service/dbmgmt"
	inspectsvc "yunshu/internal/service/inspect"
)

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

// DbmgmtSvc 返回数据库管理服务（由 router 注入）。
func (rt *Runtime) DbmgmtSvc() *dbmgmtsvc.Service {
	if rt == nil {
		return nil
	}
	svc, _ := rt.Dbmgmt.(*dbmgmtsvc.Service)
	return svc
}

// CicdSvc 返回 CI/CD 服务（由 router 注入）。
func (rt *Runtime) CicdSvc() *cicd.Service {
	if rt == nil {
		return nil
	}
	svc, _ := rt.Cicd.(*cicd.Service)
	return svc
}

// AlertSvc 返回告警服务（由 router 注入）。
func (rt *Runtime) AlertSvc() *service.AlertService {
	if rt == nil {
		return nil
	}
	svc, _ := rt.Alert.(*service.AlertService)
	return svc
}

// InspectSvc 返回巡检服务（由 router 注入）。
func (rt *Runtime) InspectSvc() *inspectsvc.Service {
	if rt == nil {
		return nil
	}
	svc, _ := rt.Inspect.(*inspectsvc.Service)
	return svc
}

// LogRetentionSvc 返回日志保留策略服务。
func (rt *Runtime) LogRetentionSvc() *service.LogRetentionService {
	if rt == nil {
		return nil
	}
	svc, _ := rt.LogRetention.(*service.LogRetentionService)
	return svc
}

// KafkaToESSvc 返回 Kafka→ES 消费服务。
func (rt *Runtime) KafkaToESSvc() *service.KafkaToESService {
	if rt == nil {
		return nil
	}
	svc, _ := rt.KafkaToES.(*service.KafkaToESService)
	return svc
}
