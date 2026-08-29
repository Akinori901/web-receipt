# 規約検証・開発環境のエントリポイント（web-receipt）。
#
# CI（.github/workflows/verify.yml）と同じコマンドをローカルでも流せるようにする。
# 「CI でだけ落ちる」状態を作らないため。
.DEFAULT_GOAL := help
.PHONY: help setup build up down logs seed migrate verify verify-go verify-front fmt clean

DC := docker compose

help: ## このヘルプを表示
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
	  | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

## ── 開発環境 ─────────────────────────────────────────────
setup: ## 初回セットアップ（build → up → seed → migrate を順に実行）
	$(MAKE) build
	$(MAKE) up
	$(MAKE) seed
	$(MAKE) migrate
	@echo ""
	@echo "  セットアップ完了。seed で出力された値を .env に書き写してから使ってください。"
	@echo "  Go API   : http://localhost:8002/api/health"
	@echo "  Frontend : http://localhost:5173"

build: ## Docker イメージをビルド
	$(DC) build

up: ## 全サービスを起動（初回は 'make setup' を使うこと）
	$(DC) up -d db storage cognito go frontend
	# バケット作成は冪等。ボリュームを消した後の起動でも確実に作られるようにする。
	$(DC) up storage-init
	@echo "  Go API   : http://localhost:8002/api/health"
	@echo "  Frontend : http://localhost:5173"

down: ## 停止（ボリュームは残す）
	$(DC) down

clean: ## 停止してボリュームも削除（DBを初期化する）
	$(DC) down -v --remove-orphans

logs: ## ログを追う
	$(DC) logs -f

seed: ## ローカル Cognito にプールとテストユーザーを作る
	$(DC) up -d cognito storage
	$(DC) run --rm --entrypoint bash \
	  -e COGNITO_ENDPOINT_URL=http://cognito:9229 \
	  -v $(PWD)/scripts:/s storage-init /s/seed-cognito.sh

migrate: ## DB マイグレーション（golang-migrate・冪等）
	$(DC) run --rm migrate

## ── 規約検証（CI と同じ内容）──────────────────────────────
verify: verify-go verify-front ## 層検証 + 静的解析 + テスト

verify-go: ## Go: 層検証(go-arch-lint) + vet + test
	@echo "==> Go クリーンアーキ層検証"
	$(DC) run --rm go sh -c "go install github.com/fe3dback/go-arch-lint@latest && \
	  $$(go env GOPATH)/bin/go-arch-lint check"
	$(DC) run --rm go go vet ./...
	$(DC) run --rm go go test ./...

verify-front: ## Frontend: 境界検証(eslint-boundaries) + 型検査
	@echo "==> フロント境界検証"
	$(DC) run --rm frontend npm run lint
	$(DC) run --rm frontend npm run typecheck

fmt: ## フォーマット
	$(DC) run --rm go gofmt -w .
