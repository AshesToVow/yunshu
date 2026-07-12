package dbmgmt

import (
	"context"
	"fmt"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
)

func (s *Service) provisionDatabaseAfterGrant(ctx context.Context, req *model.DbAccessRequest) error {
	if !isNewDatabaseRequest(*req) {
		return nil
	}
	dbName := strings.TrimSpace(req.DatabaseName)
	if dbName == "" || !isValidDbIdentifier(dbName) {
		return nil
	}
	inst, err := s.repo.GetInstanceInProject(ctx, req.ProjectID, req.InstanceID)
	if err != nil {
		return err
	}
	var sqlText string
	switch strings.ToLower(strings.TrimSpace(inst.Driver)) {
	case model.DbDriverPostgres:
		sqlText = fmt.Sprintf(`CREATE DATABASE "%s"`, strings.ReplaceAll(dbName, `"`, `""`))
	default:
		if strings.TrimSpace(req.DatabaseName) == "" {
			return nil
		}
		meta := parseAccessRequestMeta(req.MetaJSON)
		charset := strings.TrimSpace(meta.Charset)
		collation := strings.TrimSpace(meta.Collation)
		if charset == "" {
			charset = "utf8mb4"
		}
		if collation == "" {
			collation = "utf8mb4_general_ci"
		}
		if isAllowedDbCharset(charset) && isAllowedDbCollation(collation) {
			sqlText = fmt.Sprintf(
				"CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET %s COLLATE %s",
				escapeMySQLIdent(dbName), charset, collation,
			)
		} else {
			sqlText = fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", escapeMySQLIdent(dbName))
		}
	}
	systemActor := &auth.CurrentUser{Username: "system"}
	_, err = s.runWriteSQL(ctx, inst, "", sqlText, systemActor, nil)
	return err
}
