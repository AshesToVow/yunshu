package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// RegisterOpsEndpoints 注册平台运维探针与 Prometheus 抓取端点（无登录；依赖网络边界）。
func RegisterOpsEndpoints(engine *gin.Engine, db *gorm.DB, rdb *redis.Client, startedAt time.Time) {
	if engine == nil {
		return
	}
	if startedAt.IsZero() {
		startedAt = time.Now()
	}

	engine.GET("/metrics", gin.WrapH(promhttp.Handler()))

	engine.GET("/livez", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	engine.GET("/readyz", func(c *gin.Context) {
		checks := gin.H{}
		ready := true

		if db != nil {
			sqlDB, err := db.DB()
			if err != nil {
				ready = false
				checks["mysql"] = err.Error()
			} else if err := sqlDB.Ping(); err != nil {
				ready = false
				checks["mysql"] = err.Error()
			} else {
				checks["mysql"] = "ok"
			}
		} else {
			ready = false
			checks["mysql"] = "nil"
		}

		if rdb != nil {
			if err := rdb.Ping(c.Request.Context()).Err(); err != nil {
				ready = false
				checks["redis"] = err.Error()
			} else {
				checks["redis"] = "ok"
			}
		} else {
			checks["redis"] = "skipped"
		}

		status := http.StatusOK
		state := "ok"
		if !ready {
			status = http.StatusServiceUnavailable
			state = "not_ready"
		}
		c.JSON(status, gin.H{
			"status":  state,
			"uptime":  int(time.Since(startedAt).Seconds()),
			"checks":  checks,
		})
	})
}
