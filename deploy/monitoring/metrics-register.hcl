# 统一注册 Policy：给 consul_targets_sync.py / consul_k8s_pods_sync.py / 启停脚本用
service "telegraf" {
  policy = "write"
}
service "icmp" {
  policy = "write"
}
service "http" {
  policy = "write"
}
service "tcp" {
  policy = "write"
}
service "pushgateway" {
  policy = "write"
}
service "blackbox-target" {
  policy = "write"
}
service "k8s-pod" {
  policy = "write"
}
service "k8s-pod-metrics" {
  policy = "write"
}
node_prefix "" {
  policy = "read"
}
