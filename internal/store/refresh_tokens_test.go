package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRefreshTokenRotateOnce(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()

	token := "refresh-opaque-1"
	if err := SaveRefreshToken(ctx, rdb, token, RefreshSession{
		UserID:        7,
		AccessTokenID: "aid-1",
		FamilyID:      "fam-1",
	}, time.Hour); err != nil {
		t.Fatal(err)
	}

	sess, err := ConsumeRefreshToken(ctx, rdb, token)
	if err != nil {
		t.Fatal(err)
	}
	if sess.UserID != 7 || sess.AccessTokenID != "aid-1" || sess.FamilyID != "fam-1" {
		t.Fatalf("unexpected session: %+v", sess)
	}

	_, err = ConsumeRefreshToken(ctx, rdb, token)
	if err == nil {
		t.Fatal("expected second consume to fail")
	}
	var reuse *RefreshReuseError
	if !errors.As(err, &reuse) {
		t.Fatalf("expected RefreshReuseError, got %v", err)
	}
	if reuse.FamilyID != "fam-1" || reuse.UserID != 7 {
		t.Fatalf("unexpected reuse meta: %+v", reuse)
	}
}

func TestInvalidateRefreshFamilyOnReuse(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()

	oldTok := "rt-old"
	newTok := "rt-new"
	if err := SaveRefreshToken(ctx, rdb, oldTok, RefreshSession{
		UserID: 1, AccessTokenID: "a1", FamilyID: "fam-x",
	}, time.Hour); err != nil {
		t.Fatal(err)
	}
	// victim rotates
	if _, err := ConsumeRefreshToken(ctx, rdb, oldTok); err != nil {
		t.Fatal(err)
	}
	if err := SaveRefreshToken(ctx, rdb, newTok, RefreshSession{
		UserID: 1, AccessTokenID: "a2", FamilyID: "fam-x",
	}, time.Hour); err != nil {
		t.Fatal(err)
	}
	_ = rdb.Set(ctx, AccessTokenKey("a2"), "1", time.Hour).Err()

	// attacker replays old
	_, err = ConsumeRefreshToken(ctx, rdb, oldTok)
	var reuse *RefreshReuseError
	if !errors.As(err, &reuse) {
		t.Fatalf("expected reuse, got %v", err)
	}
	if err := InvalidateRefreshFamily(ctx, rdb, reuse.FamilyID); err != nil {
		t.Fatal(err)
	}
	if n, _ := rdb.Exists(ctx, RefreshTokenKey(newTok)).Result(); n != 0 {
		t.Fatal("expected new refresh revoked")
	}
	if n, _ := rdb.Exists(ctx, AccessTokenKey("a2")).Result(); n != 0 {
		t.Fatal("expected access revoked")
	}
}

func TestInvalidateAllUserRefreshTokens(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()

	_ = SaveRefreshToken(ctx, rdb, "a", RefreshSession{UserID: 1, AccessTokenID: "x", FamilyID: "f1"}, time.Hour)
	_ = SaveRefreshToken(ctx, rdb, "b", RefreshSession{UserID: 1, AccessTokenID: "y", FamilyID: "f2"}, time.Hour)
	if err := InvalidateAllUserRefreshTokens(ctx, rdb, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := ConsumeRefreshToken(ctx, rdb, "a"); err == nil {
		t.Fatal("expected revoked")
	}
}
