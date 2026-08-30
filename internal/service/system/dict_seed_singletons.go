package system

// dictSingletonTypes 单值型字典类型集合：这类 dict_type 全局只允许一条记录，
// seed 幂等按「类型」判断（ExistsByType），而非「类型 + 标签」，避免人工改标签后重复插入。
func dictSingletonTypes() map[string]struct{} {
	return map[string]struct{}{
		// Mail
		"mail_host":       {},
		"mail_port":       {},
		"mail_use_tls":    {},
		"mail_username":   {},
		"mail_password":   {},
		"mail_from_email": {},
		"mail_from_name":  {},
		// 密码策略
		"password_min_length":      {},
		"password_max_length":      {},
		"password_require_upper":   {},
		"password_require_lower":   {},
		"password_require_digit":   {},
		"password_require_special": {},
		"password_expiry_days":     {},
		"password_forbid_username": {},
		// Alert 运行配置
		"alert_webhook_token":           {},
		"alert_enrich_prometheus_url":   {},
		"alert_enrich_prometheus_token": {},
		// K8s Event 转发
		"k8s_event_forward_enabled":                 {},
		"k8s_event_forward_watcher_buffer_size":     {},
		"k8s_event_forward_worker_interval_seconds": {},
		"k8s_event_forward_worker_batch_size":       {},
		"k8s_event_forward_worker_max_retries":      {},
		// MinIO
		"minio_endpoint":      {},
		"minio_access_key":    {},
		"minio_secret_key":    {},
		"minio_bucket":        {},
		"minio_use_ssl":       {},
		"minio_region":        {},
		"minio_backup_prefix": {},
		// CI/CD 平台配置
		"cicd_enabled":                          {},
		"cicd_jenkins_base_url":                 {},
		"cicd_jenkins_username":                 {},
		"cicd_jenkins_api_token":                {},
		"cicd_jenkinsfile_repo":                 {},
		"cicd_jenkinsfile_branch":               {},
		"cicd_jenkinsfile_front":                {},
		"cicd_jenkinsfile_backend":              {},
		"cicd_git_credential_id":                {},
		"cicd_ssh_deploy_credential_id":         {},
		"cicd_minio_credential_id":              {},
		"cicd_minio_access_key":                 {},
		"cicd_minio_secret_key":                 {},
		"cicd_minio_endpoint":                   {},
		"cicd_minio_bucket_frontend":            {},
		"cicd_minio_bucket_backend":             {},
		"cicd_jenkinsfile_k8s":                  {},
		"cicd_harbor_url":                       {},
		"cicd_harbor_host_ip":                   {},
		"cicd_harbor_credential_id":             {},
		"cicd_harbor_project_group":             {},
		"cicd_harbor_username":                  {},
		"cicd_harbor_password":                  {},
		"cicd_default_wait_mins":                {},
		"cicd_default_artifact_retain_count":    {},
		"cicd_run_sync_interval_seconds":        {},
		"cicd_approval_sla_hours":               {},
		"cicd_approval_reminder_interval_hours": {},
		"cicd_forbid_self_approve":              {},
		"cicd_prod_force_audit":                 {},
		"cicd_sonar_enabled":                    {},
		"cicd_sonar_url":                        {},
		"cicd_sonar_token":                      {},
		"cicd_sonar_gate_block":                 {},
		"cicd_jenkins_callback_hmac_secret":     {},
		"cicd_jenkins_callback_url":             {},
		// AI
		"ai_enabled":            {},
		"ai_default_provider":   {},
		"ai_timeout_sec":        {},
		"ai_max_tokens":         {},
		"ai_openai_base_url":    {},
		"ai_openai_api_key":     {},
		"ai_openai_model":       {},
		"ai_deepseek_base_url":  {},
		"ai_deepseek_api_key":   {},
		"ai_deepseek_model":     {},
		"ai_anthropic_base_url": {},
		"ai_anthropic_api_key":  {},
		"ai_anthropic_model":    {},
		// dbmgmt / cmdb
		"dbmgmt_query_timeout_seconds":            {},
		"dbmgmt_max_rows":                         {},
		"dbmgmt_max_import_file_mb":               {},
		"cmdb_max_transfer_file_mb":               {},
		"dbmgmt_prod_force_approval":              {},
		"dbmgmt_forbid_self_approve":              {},
		"dbmgmt_approval_sla_hours":               {},
		"dbmgmt_approval_reminder_interval_hours": {},
		"dbmgmt_ping_interval_seconds":            {},
		"dbmgmt_max_concurrent_per_instance":      {},
		"dbmgmt_goinception_enabled":              {},
		"dbmgmt_goinception_host":                 {},
		"dbmgmt_goinception_port":                 {},
		"dbmgmt_goinception_backup":               {},
		// Elasticsearch
		"elasticsearch_enabled":                {},
		"elasticsearch_addresses":              {},
		"elasticsearch_username":               {},
		"elasticsearch_password":               {},
		"elasticsearch_index_pattern":          {},
		"elasticsearch_k8s_index_prefix":       {},
		"elasticsearch_default_retention_days": {},
		"elasticsearch_cleanup_cron_spec":      {},
		"esmgmt_backup_scheduler_enabled":      {},
		"esmgmt_backup_scheduler_tick_spec":    {},
		// Kafka
		"kafka_enabled":          {},
		"kafka_brokers":          {},
		"kafka_topic_prefix":     {},
		"kafka_k8s_topic_prefix": {},
		"kafka_topic":            {},
		"kafka_consumer_group":   {},
		"kafka_username":         {},
		"kafka_password":         {},
		"kafka_sasl_mechanism":   {},
		"kafka_batch_size":       {},
		"kafka_workers":          {},
	}
}

// builtinDictSeeds 汇总各域内置字典种子；新增域只需新增 dict_seed_<domain>.go 并在此登记。
func builtinDictSeeds() []DictEntryCreateRequest {
	groups := [][]DictEntryCreateRequest{
		alertDictSeeds(),
		k8sDictSeeds(),
		notifyDictSeeds(),
		storageDictSeeds(),
		serverDictSeeds(),
		cicdDictSeeds(),
		aiDictSeeds(),
		dbmgmtDictSeeds(),
		securityDictSeeds(),
	}

	total := 0
	for _, g := range groups {
		total += len(g)
	}

	seeds := make([]DictEntryCreateRequest, 0, total)
	for _, g := range groups {
		seeds = append(seeds, g...)
	}
	return seeds
}
