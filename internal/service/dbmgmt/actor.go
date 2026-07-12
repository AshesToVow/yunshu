package dbmgmt

import (
	"fmt"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
)

// actorUserID 从当前用户取 ID。
func actorUserID(u *auth.CurrentUser) uint {
	if u == nil {
		return 0
	}
	return u.ID
}

func actorUsername(u *auth.CurrentUser) string {
	if u == nil {
		return ""
	}
	if u.Nickname != "" {
		return u.Nickname
	}
	return u.Username
}

func principalRefs(u *auth.CurrentUser) []struct{ kind, ref string } {
	if u == nil {
		return nil
	}
	out := []struct{ kind, ref string }{
		{model.DbPrincipalUser, fmt.Sprintf("%d", u.ID)},
	}
	for _, code := range u.RoleCodes {
		if code != "" {
			out = append(out, struct{ kind, ref string }{model.DbPrincipalRole, code})
		}
	}
	for _, code := range u.GroupCodes {
		if code != "" {
			out = append(out, struct{ kind, ref string }{model.DbPrincipalGroup, code})
		}
	}
	return out
}
