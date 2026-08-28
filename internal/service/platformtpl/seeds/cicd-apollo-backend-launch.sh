#!/bin/bash
# 同步到共享库 backend-launch-template.sh 的启动段。
# 多 Meta 时必须给 -Dapollo.meta=... 整体加引号。

# 占位符由 deploy.groovy applyLaunchPlaceholders 替换：
#   {{JAVA_BIN}} {{JVM_OPTS}} {{APOLLO_ENV}} {{APOLLO_META}} {{APOLLO_NAMESPACES}} {{JARNAME}}
#
# APOLLO_META 示例：
#   http://10.241.243.21:8080,http://10.241.243.20:8080,http://10.241.243.19:8080

nohup "{{JAVA_BIN}}" {{JVM_OPTS}} \
  -Denv="{{APOLLO_ENV}}" \
  -Dapollo.meta="{{APOLLO_META}}" \
  -Dapollo.bootstrap.namespaces="{{APOLLO_NAMESPACES}}" \
  -jar "{{JARNAME}}" >> "${app_dir}/${log_path}" 2>&1 &
