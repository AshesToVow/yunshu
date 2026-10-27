package handler

import (
	"errors"
	"sync/atomic"
	"testing"

	"yunshu/internal/pkg/auth"
)

func TestWSReauthReason(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{name: "account disabled", err: auth.ErrWSAccountDisabled, want: "账号已被禁用，连接已关闭"},
		{name: "session expired", err: auth.ErrWSSessionExpired, want: "登录状态已失效，请重新登录"},
		{name: "grant or backend failure", err: errors.New("forbidden"), want: "会话或集群授权已失效，请重新连接"},
		{name: "wrapped account disabled", err: errors.Join(errors.New("ctx"), auth.ErrWSAccountDisabled), want: "账号已被禁用，连接已关闭"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := wsReauthReason(tc.err); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestWSStopReason(t *testing.T) {
	if got := wsStopReason(nil); got != "" {
		t.Fatalf("nil value got %q", got)
	}
	var v atomic.Value
	if got := wsStopReason(&v); got != "" {
		t.Fatalf("empty value got %q", got)
	}
	v.Store("会话已达到最长时长限制，请重新连接")
	if got := wsStopReason(&v); got != "会话已达到最长时长限制，请重新连接" {
		t.Fatalf("got %q", got)
	}
}
