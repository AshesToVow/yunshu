import { getData, http } from "./http";

export interface KafkamgmtConnection {
  id: number;
  name: string;
  brokers: string;
  username?: string;
  has_password?: boolean;
  sasl_mechanism?: string;
  timeout_sec?: number;
  is_default?: boolean;
  remark?: string;
  created_at?: string;
  updated_at?: string;
}

export interface KafkaTopicInfo {
  name: string;
  partitions: number;
  replication_factor: number;
  internal?: boolean;
}

export interface KafkaBrokerInfo {
  id: number;
  host: string;
  port: number;
  rack?: string;
}

export interface KafkaGroupInfo {
  group_id: string;
  state?: string;
  protocol?: string;
}

export interface KafkaGroupMember {
  member_id: string;
  client_id: string;
  client_host: string;
  topics?: string[];
}

export interface KafkaGroupLag {
  group_id: string;
  lag_total: number;
  partitions: Array<{
    topic: string;
    partition: number;
    committed: number;
    end: number;
    lag: number;
  }>;
}

export interface KafkaConsumedMessage {
  partition: number;
  offset: number;
  key?: string;
  value?: string;
  time?: string;
}

function connParams(connectionId?: number) {
  return typeof connectionId === "number" ? { connection_id: connectionId } : undefined;
}

export function listKafkamgmtConnections() {
  return getData<KafkamgmtConnection[]>(http.get("/kafkamgmt/connections"));
}

export function createKafkamgmtConnection(payload: Record<string, unknown>) {
  return getData<KafkamgmtConnection>(http.post("/kafkamgmt/connections", payload));
}

export function importKafkamgmtConnectionFromDict() {
  return getData<KafkamgmtConnection>(http.post("/kafkamgmt/connections/import-from-dict", {}));
}

export function updateKafkamgmtConnection(id: number, payload: Record<string, unknown>) {
  return getData<KafkamgmtConnection>(http.put(`/kafkamgmt/connections/${id}`, payload));
}

export function deleteKafkamgmtConnection(id: number) {
  return getData<{ ok: boolean }>(http.delete(`/kafkamgmt/connections/${id}`));
}

export function pingKafkamgmtConnection(id: number) {
  return getData<{ ok: boolean; message?: string }>(http.post(`/kafkamgmt/connections/${id}/ping`, {}));
}

export function testKafkamgmtConnection(payload: {
  brokers?: string;
  username?: string;
  password?: string;
  sasl_mechanism?: string;
  timeout_sec?: number;
  connection_id?: number;
}) {
  return getData<{ ok: boolean; message?: string }>(http.post("/kafkamgmt/connections/test", payload));
}

export function listKafkamgmtBrokers(connectionId?: number) {
  return getData<KafkaBrokerInfo[]>(http.get("/kafkamgmt/brokers", { params: connParams(connectionId) }));
}

export function listKafkamgmtTopics(connectionId?: number) {
  return getData<KafkaTopicInfo[]>(http.get("/kafkamgmt/topics", { params: connParams(connectionId) }));
}

export function getKafkamgmtTopicDetail(topic: string, connectionId?: number) {
  return getData<{
    name: string;
    partitions: Array<{ id: number; leader: number; replicas: number[]; isr: number[] }>;
  }>(http.get("/kafkamgmt/topics/detail", { params: { topic, ...connParams(connectionId) } }));
}

export function getKafkamgmtTopicConfig(topic: string, connectionId?: number) {
  return getData<Record<string, string>>(
    http.get("/kafkamgmt/topics/config", { params: { topic, ...connParams(connectionId) } }),
  );
}

export function createKafkamgmtTopic(payload: {
  connection_id?: number;
  name: string;
  num_partitions?: number;
  replication_factor?: number;
}) {
  return getData<{ ok: boolean }>(http.post("/kafkamgmt/topics", payload));
}

export function deleteKafkamgmtTopic(payload: { connection_id?: number; topic: string }) {
  return getData<{ ok: boolean }>(http.delete("/kafkamgmt/topics", { data: payload }));
}

export function createKafkamgmtPartitions(payload: {
  connection_id?: number;
  topic: string;
  total_count: number;
}) {
  return getData<{ ok: boolean }>(http.post("/kafkamgmt/topics/partitions", payload));
}

export function produceKafkamgmtMessage(payload: {
  connection_id?: number;
  topic: string;
  key?: string;
  value: string;
  partition?: number;
}) {
  return getData<{ ok: boolean }>(http.post("/kafkamgmt/topics/produce", payload));
}

export function consumeKafkamgmtMessages(payload: {
  connection_id?: number;
  topic: string;
  max_messages?: number;
  from?: string;
  timeout_ms?: number;
  partition?: number;
}) {
  return getData<KafkaConsumedMessage[]>(http.post("/kafkamgmt/topics/consume", payload));
}

export function listKafkamgmtGroups(connectionId?: number) {
  return getData<KafkaGroupInfo[]>(http.get("/kafkamgmt/groups", { params: connParams(connectionId) }));
}

export function getKafkamgmtGroupMembers(groupId: string, connectionId?: number) {
  return getData<KafkaGroupMember[]>(
    http.get("/kafkamgmt/groups/members", { params: { group_id: groupId, ...connParams(connectionId) } }),
  );
}

export function getKafkamgmtGroupLag(groupId: string, connectionId?: number, topic?: string) {
  return getData<KafkaGroupLag>(
    http.get("/kafkamgmt/groups/lag", {
      params: { group_id: groupId, topic, ...connParams(connectionId) },
    }),
  );
}

export function deleteKafkamgmtGroup(payload: { connection_id?: number; group_id: string }) {
  return getData<{ ok: boolean }>(http.delete("/kafkamgmt/groups", { data: payload }));
}
