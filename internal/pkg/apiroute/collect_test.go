package apiroute

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCollectAndSkip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	api := engine.Group("/api/v1")
	api.GET("/users", func(c *gin.Context) {})
	api.GET("/users/:id", func(c *gin.Context) {})
	api.GET("/health", func(c *gin.Context) {})
	api.POST("/auth/login", func(c *gin.Context) {})

	entries := Collect(engine)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d: %+v", len(entries), entries)
	}
	if entries[0].Path != "/api/v1/users" || entries[0].Method != "GET" {
		t.Fatalf("unexpected first entry: %+v", entries[0])
	}
	if !ShouldSkip("GET", "/api/v1/health") {
		t.Fatal("health should be skipped")
	}
	if !ShouldSkip("POST", "/api/v1/auth/login") {
		t.Fatal("auth login should be skipped")
	}
}

func TestDefaultName(t *testing.T) {
	name := DefaultName("GET", "/api/v1/projects/:id/dbmgmt/sql-tickets")
	if name == "" {
		t.Fatal("expected non-empty name")
	}
}
