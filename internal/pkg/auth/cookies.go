package auth

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Cookie names for browser session (HttpOnly). JS must not read these.
const (
	CookieAccess  = "ys_at"
	CookieRefresh = "ys_rt"
)

// SessionCookies carries values to write on login/refresh.
type SessionCookies struct {
	AccessToken        string
	RefreshToken       string
	AccessExpiresAt    time.Time
	RefreshExpiresAt   time.Time
	Secure             bool
	SameSite           http.SameSite
	Domain             string // empty = host-only
}

func sameSiteOrLax(v http.SameSite) http.SameSite {
	if v == 0 {
		return http.SameSiteLaxMode
	}
	return v
}

// WriteSessionCookies sets access + refresh HttpOnly cookies.
func WriteSessionCookies(c *gin.Context, s SessionCookies) {
	if c == nil {
		return
	}
	ss := sameSiteOrLax(s.SameSite)
	writeCookie(c, CookieAccess, s.AccessToken, s.AccessExpiresAt, s.Secure, ss, s.Domain)
	writeCookie(c, CookieRefresh, s.RefreshToken, s.RefreshExpiresAt, s.Secure, ss, s.Domain)
}

// ClearSessionCookies expires auth cookies.
func ClearSessionCookies(c *gin.Context, secure bool, sameSite http.SameSite, domain string) {
	if c == nil {
		return
	}
	ss := sameSiteOrLax(sameSite)
	expired := time.Unix(0, 0)
	writeCookie(c, CookieAccess, "", expired, secure, ss, domain)
	writeCookie(c, CookieRefresh, "", expired, secure, ss, domain)
}

func writeCookie(c *gin.Context, name, value string, expires time.Time, secure bool, sameSite http.SameSite, domain string) {
	maxAge := int(time.Until(expires).Seconds())
	if value == "" || maxAge <= 0 {
		maxAge = -1
		value = ""
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Domain:   strings.TrimSpace(domain),
		MaxAge:   maxAge,
		Expires:  expires,
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
	})
}

// ExtractAccessToken prefers Authorization Bearer, then ys_at cookie.
func ExtractAccessToken(c *gin.Context) string {
	if c == nil {
		return ""
	}
	header := c.GetHeader("Authorization")
	if strings.HasPrefix(header, "Bearer ") {
		if t := strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")); t != "" {
			return t
		}
	}
	if t, err := c.Cookie(CookieAccess); err == nil {
		return strings.TrimSpace(t)
	}
	return ""
}

// ExtractRefreshToken reads ys_rt cookie.
func ExtractRefreshToken(c *gin.Context) string {
	if c == nil {
		return ""
	}
	t, err := c.Cookie(CookieRefresh)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(t)
}
