package cmd

import "yunshu/internal/model"

func seedPermissionsKafkamgmt() []model.Permission {
	return []model.Permission{
		{Name: "Kafka 连接列表", Resource: "/api/v1/kafkamgmt/connections", Action: "GET", Description: "List Kafka connections"},
		{Name: "Kafka 创建连接", Resource: "/api/v1/kafkamgmt/connections", Action: "POST", Description: "Create Kafka connection"},
		{Name: "Kafka 从字典导入连接", Resource: "/api/v1/kafkamgmt/connections/import-from-dict", Action: "POST", Description: "Import Kafka connection from data dictionary"},
		{Name: "Kafka 更新连接", Resource: "/api/v1/kafkamgmt/connections/:id", Action: "PUT", Description: "Update Kafka connection"},
		{Name: "Kafka 删除连接", Resource: "/api/v1/kafkamgmt/connections/:id", Action: "DELETE", Description: "Delete Kafka connection"},
		{Name: "Kafka 连接探测", Resource: "/api/v1/kafkamgmt/connections/:id/ping", Action: "POST", Description: "Ping Kafka connection"},
		{Name: "Kafka 连通测试", Resource: "/api/v1/kafkamgmt/connections/test", Action: "POST", Description: "Test Kafka credentials without saving"},
		{Name: "Kafka Broker 列表", Resource: "/api/v1/kafkamgmt/brokers", Action: "GET", Description: "List Kafka brokers"},
		{Name: "Kafka Topic 列表", Resource: "/api/v1/kafkamgmt/topics", Action: "GET", Description: "List Kafka topics"},
		{Name: "Kafka Topic 详情", Resource: "/api/v1/kafkamgmt/topics/detail", Action: "GET", Description: "Get Kafka topic partitions"},
		{Name: "Kafka Topic 配置", Resource: "/api/v1/kafkamgmt/topics/config", Action: "GET", Description: "Get Kafka topic configs"},
		{Name: "Kafka 创建 Topic", Resource: "/api/v1/kafkamgmt/topics", Action: "POST", Description: "Create Kafka topic"},
		{Name: "Kafka 删除 Topic", Resource: "/api/v1/kafkamgmt/topics", Action: "DELETE", Description: "Delete Kafka topic"},
		{Name: "Kafka 扩容分区", Resource: "/api/v1/kafkamgmt/topics/partitions", Action: "POST", Description: "Increase Kafka topic partitions"},
		{Name: "Kafka 生产消息", Resource: "/api/v1/kafkamgmt/topics/produce", Action: "POST", Description: "Produce sample Kafka message"},
		{Name: "Kafka 抽样消费", Resource: "/api/v1/kafkamgmt/topics/consume", Action: "POST", Description: "Consume sample Kafka messages"},
		{Name: "Kafka 消费组列表", Resource: "/api/v1/kafkamgmt/groups", Action: "GET", Description: "List Kafka consumer groups"},
		{Name: "Kafka 消费组成员", Resource: "/api/v1/kafkamgmt/groups/members", Action: "GET", Description: "List Kafka consumer group members"},
		{Name: "Kafka 消费组 Lag", Resource: "/api/v1/kafkamgmt/groups/lag", Action: "GET", Description: "Get Kafka consumer group lag"},
		{Name: "Kafka 删除消费组", Resource: "/api/v1/kafkamgmt/groups", Action: "DELETE", Description: "Delete Kafka consumer group"},
	}
}
