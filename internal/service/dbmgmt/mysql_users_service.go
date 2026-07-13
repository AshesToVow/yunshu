package dbmgmt

import (
	"context"
	"fmt"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
)

type InstanceMySQLUserItem struct {
	ID            uint     `json:"id,omitempty"`
	Username      string   `json:"username"`
	Host          string   `json:"host"`
	GrantLines    []string `json:"grant_lines,omitempty"`
	HasPassword   bool     `json:"has_password"`
	FromPlatform  bool     `json:"from_platform"`
	Remark        string   `json:"remark,omitempty"`
}

func (s *Service) ListInstanceMySQLUsers(ctx context.Context, projectID, instanceID uint, actor *auth.CurrentUser) ([]InstanceMySQLUserItem, error) {
	inst, err := s.repo.GetInstanceInProject(ctx, projectID, instanceID)
	if err != nil {
		return nil, err
	}
	if inst.Driver != model.DbDriverMySQL {
		return nil, constants.ErrBadRequestWithMsg("仅 MySQL 实例支持用户管理")
	}
	perm, err := s.GetEffectivePermission(ctx, projectID, instanceID, actor)
	if err != nil {
		return nil, err
	}
	if !perm.CanManage && !auth.IsSuperAdminRole(actor.RoleCodes) {
		return nil, constants.ErrForbidden
	}

	accounts, _ := s.repo.ListInstanceAccounts(ctx, projectID, instanceID)
	byKey := map[string]model.DbInstanceAccount{}
	for _, a := range accounts {
		byKey[a.Username+"@"+a.Host] = a
	}

	release := s.acquireInstance(instanceID)
	defer release()
	sess, err := s.openSession(ctx, inst)
	if err != nil {
		return nil, err
	}
	defer sess.Close()

	rows, err := sess.DB.QueryContext(ctx, `SELECT user, host FROM mysql.user WHERE user NOT IN ('mysql.sys','mysql.session','mysql.infoschema','mysql') ORDER BY user, host`)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("查询 mysql.user 失败: " + err.Error())
	}
	defer rows.Close()

	var out []InstanceMySQLUserItem
	for rows.Next() {
		var user, host string
		if err := rows.Scan(&user, &host); err != nil {
			return nil, err
		}
		item := InstanceMySQLUserItem{Username: user, Host: host}
		if acc, ok := byKey[user+"@"+host]; ok {
			item.ID = acc.ID
			item.FromPlatform = true
			item.Remark = acc.Remark
			item.HasPassword = acc.EncPassword != ""
			if acc.GrantsSummary != "" {
				item.GrantLines = strings.Split(acc.GrantsSummary, "\n")
			}
		}
		grantRows, gerr := sess.DB.QueryContext(ctx, fmt.Sprintf("SHOW GRANTS FOR `%s`@`%s`", escapeMySQLIdent(user), escapeMySQLIdent(host)))
		if gerr == nil {
			var lines []string
			for grantRows.Next() {
				var line string
				if scanErr := grantRows.Scan(&line); scanErr == nil {
					lines = append(lines, line)
				}
			}
			grantRows.Close()
			if len(lines) > 0 {
				item.GrantLines = lines
			}
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Service) ListInstanceAccounts(ctx context.Context, projectID, instanceID uint, actor *auth.CurrentUser) ([]InstanceMySQLUserItem, error) {
	perm, err := s.GetEffectivePermission(ctx, projectID, instanceID, actor)
	if err != nil {
		return nil, err
	}
	if !perm.CanManage && !auth.IsSuperAdminRole(actor.RoleCodes) {
		return nil, constants.ErrForbidden
	}
	list, err := s.repo.ListInstanceAccounts(ctx, projectID, instanceID)
	if err != nil {
		return nil, err
	}
	out := make([]InstanceMySQLUserItem, 0, len(list))
	for _, a := range list {
		item := InstanceMySQLUserItem{
			ID: a.ID, Username: a.Username, Host: a.Host,
			FromPlatform: true, HasPassword: a.EncPassword != "", Remark: a.Remark,
		}
		if a.GrantsSummary != "" {
			item.GrantLines = strings.Split(a.GrantsSummary, "\n")
		}
		out = append(out, item)
	}
	return out, nil
}

type InstanceMySQLUserPrivilegesResponse struct {
	Privileges []string `json:"privileges"`
}

func (s *Service) GetInstanceMySQLUserPrivileges(ctx context.Context, projectID, instanceID uint, mysqlUser, mysqlHost, privLevel, database string, actor *auth.CurrentUser) (*InstanceMySQLUserPrivilegesResponse, error) {
	if err := s.requireInstanceManage(ctx, projectID, instanceID, actor); err != nil {
		return nil, err
	}
	inst, err := s.repo.GetInstanceInProject(ctx, projectID, instanceID)
	if err != nil {
		return nil, err
	}
	if inst.Driver != model.DbDriverMySQL {
		return nil, constants.ErrBadRequestWithMsg("仅 MySQL 实例支持用户权限查询")
	}
	user := strings.TrimSpace(mysqlUser)
	host := strings.TrimSpace(mysqlHost)
	if user == "" {
		return nil, constants.ErrBadRequestWithMsg("请指定 MySQL 用户")
	}
	if host == "" {
		host = "%"
	}
	if strings.EqualFold(strings.TrimSpace(privLevel), model.DbAppUserPrivDatabase) && strings.TrimSpace(database) == "" {
		return &InstanceMySQLUserPrivilegesResponse{Privileges: []string{}}, nil
	}

	release := s.acquireInstance(instanceID)
	defer release()
	sess, err := s.openSession(ctx, inst)
	if err != nil {
		return nil, err
	}
	defer sess.Close()

	grantRows, err := sess.DB.QueryContext(ctx, fmt.Sprintf("SHOW GRANTS FOR `%s`@`%s`", escapeMySQLIdent(user), escapeMySQLIdent(host)))
	var lines []string
	if err == nil {
		defer grantRows.Close()
		for grantRows.Next() {
			var line string
			if scanErr := grantRows.Scan(&line); scanErr == nil {
				lines = append(lines, line)
			}
		}
		if err := grantRows.Err(); err != nil {
			return nil, constants.ErrBadRequestWithMsg("读取用户授权失败: " + err.Error())
		}
	}
	if len(lines) == 0 {
		accounts, _ := s.repo.ListInstanceAccounts(ctx, projectID, instanceID)
		for _, acc := range accounts {
			if acc.Username == user && acc.Host == host && acc.GrantsSummary != "" {
				for _, ln := range strings.Split(acc.GrantsSummary, "\n") {
					if t := strings.TrimSpace(ln); t != "" {
						lines = append(lines, t)
					}
				}
				break
			}
		}
	}
	return &InstanceMySQLUserPrivilegesResponse{
		Privileges: parseMySQLGrantPrivileges(lines, privLevel, database),
	}, nil
}

type InstanceAccountPasswordResponse struct {
	Password string `json:"password"`
}

func (s *Service) GetInstanceAccountPassword(ctx context.Context, projectID, instanceID, accountID uint, actor *auth.CurrentUser) (*InstanceAccountPasswordResponse, error) {
	acc, err := s.repo.GetInstanceAccount(ctx, projectID, accountID)
	if err != nil {
		return nil, err
	}
	if acc.InstanceID != instanceID {
		return nil, constants.ErrNotFound
	}
	pw, err := s.RevealInstanceAccountPassword(ctx, projectID, accountID, actor)
	if err != nil {
		return nil, err
	}
	return &InstanceAccountPasswordResponse{Password: pw}, nil
}
