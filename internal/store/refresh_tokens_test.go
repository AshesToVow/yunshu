package store

import (
	"context"
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

	if _, err := ConsumeRefreshToken(ctx, rdb, token); err == nil {
		t.Fatal("expected second consume to fail")
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

	_ = SaveRefreshToken(ctx, rdb, "a", RefreshSession{UserID: 1, AccessTokenID: "x", FamilyID: "f"}, time.Hour)
	_ = SaveRefreshToken(ctx, rdb, "b", RefreshSession{UserID: 1, AccessTokenID: "y", FamilyID: "f"}, time.Hour)
	if err := InvalidateAllUserRefreshTokens(ctx, rdb, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := ConsumeRefreshToken(ctx, rdb, "a"); err == nil {
		t.Fatal("expected revoked")
	}
}
