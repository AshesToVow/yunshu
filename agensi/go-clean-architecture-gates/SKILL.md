---
name: go-clean-architecture-gates
description: >-
  Hard gates and conventions for Go + Gin + GORM services built with Clean
  Architecture: Handler→Service→Repository layering, no *gorm.DB in Service
  structs, soft-delete conventions, transaction patterns, Wire DI rules for
  cross-domain dependencies, and handler file organization. Use when writing
  or reviewing Go service/repository/handler/router code or Wire providers.
---

# Go Clean Architecture Gates

Companion skills: modern Go guidelines, Go error handling, Go code style.

## Layering (hard gates)

```
Handler → Service → Repository(interface) → GORM
```

| Layer | Allowed | Forbidden |
|-------|---------|-----------|
| Handler | Parse/auth context, call Service, respond | Hold or call `*gorm.DB` |
| Service | Business orchestration, depend on **Repo interfaces** | **New** `db *gorm.DB` fields; hardcode concrete Repo implementations across domains |
| Repository | GORM / SQL | Business rules (ACL decisions etc. belong in Service) |

## Service structs must not hold *gorm.DB

- **Ban**: adding a `db *gorm.DB` (or equivalent direct DB handle) field to a Service struct.
- **Exceptions** (must be commented with the reason):
  - Dictionary / very short utility services
  - Existing legacy files: bug fixes only — do **not** add new direct table queries; new capabilities go through Repo
- Multi-table writes: prefer Repo-level `Transaction` / `WithTx`, or a Service-boundary `db.WithContext(ctx).Transaction` that only uses Repos rebuilt from `tx`.

## Cross-domain dependency injection (Wire)

- Inject AI / CI-CD / alerting-style cross-domain dependencies through **one provider per domain**.
- **Ban** adding `SetXxxDeps`-style setters in the route-assembly function; never register two providers for the same type.
- Plugin/module HTTP: define narrow interfaces in a dedicated deps package; the route-assembly struct implements the bundle interface. The plugin runtime must **not** hold a `Deps` field (avoids plugin↔handler import cycles); registration goes through a binder closure.
- Workers: fetch slots via typed assertion on the runtime (`plugin.As[*Concrete](rt.Xxx)`).
- New plugin route: add a narrow interface to the bundle, implement the getter in the router, dispatch in the plugin-route binder.

## Soft-delete

| Entity type | Rule |
|-------------|------|
| Configurable business entities (users, projects, alert silences, …) | Default `DeletedAt gorm.DeletedAt` + `json:"-"` + `gorm:"index"` |
| Business unique keys must be reusable | Composite unique index **includes** `deleted_at` |
| Global-unique seed rows (permissions, …) | Unique index may omit `deleted_at`; revive via upsert that clears `deleted_at` |
| Audit / append-only (operation logs, SQL audit logs, acks) | **No** soft-delete |
| DB-admin managed tables without `DeletedAt` | Hard delete — never assume GORM soft-delete |
| Cross-table joins | Soft-deleted tables need explicit `deleted_at IS NULL` (raw SQL / `Joins`) |
| Physical cleanup | Only `Unscoped()` / DBA; never in business paths |

## Transactions (Tx)

1. Handlers never open transactions.
2. Multi-table writes: `db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { ... })` — always carry `ctx`.
3. Prefer Repo-level wrapping: `Transaction(ctx, fn)` rebuilds same-interface Repos from `tx`; avoids bare `tx` scattered in Service code.
4. Approval chains: prefer the workflow engine's existing Tx; new multi-table writes must be an explicit Transaction or join the workflow.
5. Two-step creation may use compensating soft-delete rollback (must be commented); prefer a real Tx when possible.

## Handler file organization

Fat handlers: split into multiple files, **same struct** (e.g. `OrderHandler` + `order_export_handler.go`).

- Don't split the type or change DI/router signatures unless the domain is truly independent.
- New capabilities land in the matching `*_handler.go` instead of piling into one file.

## Self-check checklist

- [ ] New Service has no `*gorm.DB` field (or exception is commented)
- [ ] New persistence goes through a Repo interface
- [ ] Soft-delete choice matches the table above
- [ ] Multi-table writes have a Tx or an explicit compensating strategy
- [ ] No new `SetXxxDeps` setters; plugin runtime's `any` surface not widened
