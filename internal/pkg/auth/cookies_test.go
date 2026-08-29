package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestExtractAccessTokenBearerThenCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("bearer wins", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Request.Header.Set("Authorization", "Bearer from-header")
		c.Request.AddCookie(&http.Cookie{Name: CookieAccess, Value: "from-cookie"})
		if got := ExtractAccessToken(c); got != "from-header" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("cookie fallback", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Request.AddCookie(&http.Cookie{Name: CookieAccess, Value: " from-cookie "})
		if got := ExtractAccessToken(c); got != "from-cookie" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("empty", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		if got := ExtractAccessToken(c); got != "" {
			t.Fatalf("got %q", got)
		}
	})
}

func TestExtractRefreshToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	c.Request.AddCookie(&http.Cookie{Name: CookieRefresh, Value: "rt-1"})
	if got := ExtractRefreshToken(c); got != "rt-1" {
		t.Fatalf("got %q", got)
	}
}

func TestWriteAndClearSessionCookiesHttpOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/login", nil)

	accessExp := time.Now().Add(15 * time.Minute)
	refreshExp := time.Now().Add(7 * 24 * time.Hour)
	WriteSessionCookies(c, SessionCookies{
		AccessToken:      "at",
		RefreshToken:     "rt",
		AccessExpiresAt:  accessExp,
		RefreshExpiresAt: refreshExp,
		Secure:           false,
	})
	cookies := w.Result().Cookies()
	if len(cookies) < 2 {
		t.Fatalf("expected 2 cookies, got %d", len(cookies))
	}
	for _, ck := range cookies {
		if !ck.HttpOnly {
			t.Fatalf("cookie %s must be HttpOnly", ck.Name)
		}
		if ck.Name != CookieAccess && ck.Name != CookieRefresh {
			t.Fatalf("unexpected cookie %s", ck.Name)
		}
	}

	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	ClearSessionCookies(c2, false, 0, "")
	cleared := w2.Result().Cookies()
	if len(cleared) < 2 {
		t.Fatalf("expected clear cookies, got %d", len(cleared))
	}
}
