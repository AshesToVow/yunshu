// Package all 通过 blank import 注册全部内置业务插件（GVA 风格插件表）。
package all

import (
	_ "yunshu/internal/plugins/alert"
	_ "yunshu/internal/plugins/backup"
	_ "yunshu/internal/plugins/cmdb"
	_ "yunshu/internal/plugins/core"
	_ "yunshu/internal/plugins/k8s"
	_ "yunshu/internal/plugins/project"
)
