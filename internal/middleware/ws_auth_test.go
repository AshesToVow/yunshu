package middleware

import (
	"context"
	"errors"
	"testing"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/store"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

type fakeWSUserLoader struct {
	user *model.User
	err  error
}

func (f *fakeWSUserLoader) GetByID(context.Context, uint) (*model.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.user, nil
}

func TestWSSessionRevalidator(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	if err := rdb.Set(context.Background(), store.AccessTokenKey("tok-1"), "1", 0).Err(); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	enabled := &model.User{ID: 1, Status: model.StatusEnabled}
	disabled := &model.User{ID: 1, Status: model.StatusDisabled}
	forceChange := &model.User{ID: 1, Status: model.StatusEnabled, MustChangePassword: true}

	cases := []struct {
		name    string
		rdb     *redis.Client
		users   wsUserLoader
		token   string
		wantErr error
		wantAny bool
	}{
		{name: "enabled session", rdb: rdb, users: &fakeWSUserLoader{user: enabled}, token: "tok-1"},
		{name: "nil users skips account check", rdb: rdb, users: nil, token: "tok-1"},
		{name: "session missing", rdb: rdb, users: &fakeWSUserLoader{user: enabled}, token: "tok-missing", wantErr: auth.ErrWSSessionExpired},
		{name: "account disabled", rdb: rdb, users: &fakeWSUserLoader{user: disabled}, token: "tok-1", wantErr: auth.ErrWSAccountDisabled},
		{name: "account missing", rdb: rdb, users: &fakeWSUserLoader{}, token: "tok-1", wantErr: auth.ErrWSSessionExpired},
		{name: "password change required", rdb: rdb, users: &fakeWSUserLoader{user: forceChange}, token: "tok-1", wantErr: auth.ErrWSPasswordExpired},
		{name: "loader error fails closed", rdb: rdb, users: &fakeWSUserLoader{err: errors.New("db down")}, token: "tok-1", wantAny: true},
		{name: "redis missing fails closed", rdb: nil, users: &fakeWSUserLoader{user: enabled}, token: "tok-1", wantAny: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := wsSessionRevalidator(tc.rdb, tc.users, 1, tc.token, nil)(context.Background())
			switch {
			case tc.wantErr != nil:
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err=%v, want %v", err, tc.wantErr)
				}
			case tc.wantAny:
				if err == nil {
					t.Fatal("expected error")
				}
				if errors.Is(err, auth.ErrWSSessionExpired) || errors.Is(err, auth.ErrWSAccountDisabled) || errors.Is(err, auth.ErrWSPasswordExpired) {
					t.Fatalf("unexpected classified error: %v", err)
				}
			default:
				if err != nil {
					t.Fatalf("unexpected err: %v", err)
				}
			}
		})
	}
}
