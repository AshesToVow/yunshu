package dbmgmt

import (
	"strings"

	"github.com/xwb1989/sqlparser"
)

// extractTableRefsAST 用 AST 提取表引用；解析失败返回 ok=false，由调用方回退正则。
func extractTableRefsAST(sqlText, defaultDB string) ([]queryTableRef, bool) {
	text := strings.TrimRight(strings.TrimSpace(sqlText), "; \t\r\n")
	if text == "" {
		return nil, false
	}
	stmt, err := sqlparser.Parse(text)
	if err != nil {
		return nil, false
	}
	seen := map[string]struct{}{}
	var refs []queryTableRef
	add := func(schema, table string) {
		table = strings.TrimSpace(table)
		if table == "" {
			return
		}
		schema = strings.TrimSpace(schema)
		if schema == "" {
			schema = defaultDB
		}
		key := strings.ToLower(schema) + "." + strings.ToLower(table)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		refs = append(refs, queryTableRef{Schema: schema, Table: table})
	}
	walkTableExprs := func(expr sqlparser.TableExpr) {
		var walk func(sqlparser.TableExpr)
		walk = func(e sqlparser.TableExpr) {
			if e == nil {
				return
			}
			switch t := e.(type) {
			case *sqlparser.AliasedTableExpr:
				switch expr := t.Expr.(type) {
				case sqlparser.TableName:
					add(expr.Qualifier.String(), expr.Name.String())
				case *sqlparser.Subquery:
					if sel, ok := expr.Select.(*sqlparser.Select); ok {
						for _, fe := range sel.From {
							walk(fe)
						}
					}
				}
			case *sqlparser.JoinTableExpr:
				walk(t.LeftExpr)
				walk(t.RightExpr)
			case *sqlparser.ParenTableExpr:
				for _, ie := range t.Exprs {
					walk(ie)
				}
			}
		}
		walk(expr)
	}
	switch s := stmt.(type) {
	case *sqlparser.Select:
		for _, fe := range s.From {
			walkTableExprs(fe)
		}
	case *sqlparser.Union:
		left, ok1 := s.Left.(*sqlparser.Select)
		right, ok2 := s.Right.(*sqlparser.Select)
		if ok1 {
			for _, fe := range left.From {
				walkTableExprs(fe)
			}
		}
		if ok2 {
			for _, fe := range right.From {
				walkTableExprs(fe)
			}
		}
	case *sqlparser.Insert:
		add(s.Table.Qualifier.String(), s.Table.Name.String())
		if sel, ok := s.Rows.(*sqlparser.Select); ok {
			for _, fe := range sel.From {
				walkTableExprs(fe)
			}
		}
	case *sqlparser.Update:
		for _, te := range s.TableExprs {
			walkTableExprs(te)
		}
	case *sqlparser.Delete:
		for _, te := range s.TableExprs {
			walkTableExprs(te)
		}
		if s.Targets != nil {
			for _, tn := range s.Targets {
				add(tn.Qualifier.String(), tn.Name.String())
			}
		}
	case *sqlparser.DDL:
		if s.Table.Name.String() != "" {
			add(s.Table.Qualifier.String(), s.Table.Name.String())
		}
		if s.NewName.Name.String() != "" {
			add(s.NewName.Qualifier.String(), s.NewName.Name.String())
		}
	default:
		return nil, false
	}
	return refs, true
}
