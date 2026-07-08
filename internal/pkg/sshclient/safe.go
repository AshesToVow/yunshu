package sshclient

import "log/slog"

func safeGo(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Default().With("component", "sshclient").Error("goroutine panic", "recover", r)
			}
		}()
		fn()
	}()
}
