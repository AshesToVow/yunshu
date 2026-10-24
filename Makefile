# Yunshu 统一构建入口。本地与 CI 共用；全部目标可 help 查看。
SHELL := /bin/bash
.DEFAULT_GOAL := help

BIN        ?= yunshu
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
IMAGE      ?= yunshu:$(VERSION)
GOFLAGS    ?=
OPENAPI    ?= docs/apipost/permission-system.openapi.yaml
API_DOC_MD ?= docs/API接口设计说明书.md

.PHONY: help
help: ## 显示所有目标
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## 编译后端二进制（BINS 预留：cmd/ 单入口即根 main）
	go build $(GOFLAGS) -ldflags "-X main.version=$(VERSION)" -o $(BIN) .

.PHONY: test
test: ## 后端单元测试（与 CI 一致）
	go test ./... -short -count=1

.PHONY: vet
vet: ## go vet 全仓
	go vet ./...

.PHONY: fmt
fmt: ## gofmt 全仓（cmd/internal/tools）
	gofmt -w $$(find cmd internal tools -name '*.go' -not -path '*/wire_gen.go') 2>/dev/null || true

.PHONY: lint
lint: ## golangci-lint（本地需先安装，见 CONTRIBUTING）
	golangci-lint run ./...

.PHONY: genopenapi
genopenapi: ## 重新生成 OpenAPI 与接口设计文档（改路由后必须执行）
	go run ./tools/genopenapi -out $(OPENAPI)
	python tools/gen_api_design_md.py

.PHONY: generrno
generrno: ## 重新生成错误码清单（改手写业务码后必须执行）
	go run ./tools/generrno

.PHONY: docs-drift
docs-drift: genopenapi generrno ## 检查 API 文档/错误码清单是否有未提交漂移
	@if ! git diff --quiet -- $(OPENAPI) $(API_DOC_MD) docs/错误码清单.md; then \
		echo "文档与代码漂移，请提交 genopenapi/generrno 重新生成结果"; \
		git diff --stat -- $(OPENAPI) $(API_DOC_MD) docs/错误码清单.md; \
		exit 1; \
	fi; \
	echo "文档与代码一致"

.PHONY: web-install
web-install: ## 前端依赖安装
	cd web && npm ci

.PHONY: web-build
web-build: ## 前端构建（tsc -b + vite）
	cd web && npm run build

.PHONY: web-lint
web-lint: ## 前端 lint（全 src）
	cd web && npm run lint

.PHONY: web-test
web-test: ## 前端单测（vitest）
	cd web && npm test

.PHONY: ci
ci: vet test docs-drift web-lint web-test web-build ## CI 本地预演（等价流水线）

.PHONY: hooks
hooks: ## 安装仓库 Git 钩子
	sh scripts/install-hooks.sh

.PHONY: changelog
changelog: ## 生成 CHANGELOG（需 git-chglog）
	git-chglog -o CHANGELOG.md

.PHONY: image
image: ## 构建 Docker 镜像
	docker build -f Dockerfile.backend -t $(IMAGE) .

.PHONY: run
run: ## 本地启动（需先完成 config 与 migrate/seed）
	go run . server

.PHONY: migrate
migrate: ## 执行数据库迁移
	go run . migrate

.PHONY: seed
seed: ## 写入种子数据（权限/字典等）
	go run . seed
