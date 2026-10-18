# web-receipt

**組織単位で領収書を作り、PDF で発行し、相手に受け取ってもらうまでを扱う Web アプリ。**

<p>
  <img alt="Go" src="https://img.shields.io/badge/Go-clean_architecture-00ADD8?logo=go&logoColor=white">
  <img alt="React" src="https://img.shields.io/badge/React-feature_boundaries-61DAFB?logo=react&logoColor=black">
  <img alt="PostgreSQL" src="https://img.shields.io/badge/PostgreSQL-pgx-4169E1?logo=postgresql&logoColor=white">
  <img alt="Cognito" src="https://img.shields.io/badge/Auth-Cognito_(OIDC_+_PKCE)-DD344C?logo=amazonaws&logoColor=white">
  <img alt="License" src="https://img.shields.io/badge/License-MIT-green">
</p>

[clean-arch-starter](https://github.com/Akinori901/clean-arch-starter) の Go + React の構成から切り出して、
実際に使うアプリとして育てているものです。層の規約（`.claude/rules/`）と、それを機械検証する CI もそのまま引き継いでいます。

## できること

| 機能 | 内容 |
|---|---|
| 組織（テナント） | 発行者情報（法人名・住所・電話・メール・インボイス登録番号）を組織ごとに持つ。領収書は組織に属し、認可も組織で判定する |
| メンバー | 組織の管理者がメンバーを Cognito に登録する。管理者 / メンバーの役割 |
| 領収書 | 宛名・金額・税区分・但し書き・発行日で作成。一覧（宛名で絞り込み）・再発行 |
| PDF 発行 | 発行者欄と押印用の四角枠つきで出力（日本語フォントを同梱） |
| 受け渡し | 相手に「閲覧 ID ＋ 閲覧パスワード」を渡し、ログインなしで受け取ってもらう。宛名は相手が確定でき、確定後は変更できない |

## 構成

```
web-receipt/
├── go/                   バックエンド（Go・クリーンアーキテクチャ）
│   ├── cmd/app/          HTTP サーバと Lambda の切り替えだけ
│   ├── internal/
│   │   ├── entity/       最内層（標準ライブラリのみ）。値オブジェクトとドメインのエラー
│   │   ├── usecase/      ユースケースと、外側への契約（interface）
│   │   ├── repo/         PostgreSQL（pgx）・Cognito・S3・PDF（fpdf）の実装
│   │   ├── controller/   HTTP（chi）
│   │   └── app/          依存の結線（composition root）
│   └── migrations/       golang-migrate
├── web/                  フロントエンド（React・機能単位の feature 構成）
│   └── src/features/     auth / organization / receipts / handover / receipt-view / health
└── .claude/rules/        層の規約（AI にコードを書かせる前提）
```

| 対象 | ローカル |
|---|---|
| DB | PostgreSQL（Docker、ホスト側 15433） |
| オブジェクトストレージ | SeaweedFS（S3 API 互換、8333） |
| 認証 | cognito-local（9229） |
| バックエンド | Go（8002） |
| フロントエンド | Vite（5173） |

ローカルと本番で同じ SDK・同じコードパスを通し、違いは環境変数（エンドポイント）だけで吸収します。

## クイックスタート

```bash
cp .env.example .env
make setup        # build → up → seed（ローカル Cognito にプールとテストユーザー）→ migrate
```

seed が出力したユーザープール ID とクライアント ID を `.env` に書き写し、`make up` で起動し直します。
そのうえで http://localhost:5173 を開き、`scripts/seed-cognito.sh` が作るテストユーザーでサインインします
（メールアドレスとパスワードは環境変数 `TEST_EMAIL` / `TEST_PASSWORD` で変えられます）。

## 規約の検証

```bash
make verify       # Go: go-arch-lint（層の依存方向）+ vet + test / Frontend: 境界検証 + 型検査
```

層の依存方向は文章ではなく CI で落ちる形にしています（詳しくは `.claude/rules/`）。

## デプロイ（AWS）

Go は Lambda（コンテナイメージ・arm64）、React は S3 + CloudFront、認証は Cognito で、SAM（`infra/sam/`）が管理します。

1. `infra/terraform` で Go のイメージを置く ECR（`web-receipt-go`）を作る
2. VPC と RDS PostgreSQL を用意する（既存のものでよい。Lambda から 5432 で届くこと）
3. GitHub Secrets に `AWS_DEPLOY_ROLE_ARN`（OIDC）/ `DB_HOST` / `DB_SECRET_ARN` / `VPC_SUBNET_IDS` / `LAMBDA_SG_ID` を入れる
4. `main` への push で dev 環境へ自動デプロイ。stg / prod は Actions の `workflow_dispatch` で環境を選んで実行する

DB の認証情報は Secrets Manager から動的参照で Lambda に渡すので、値はリポジトリにも GitHub Secrets にも置きません。
Secrets が未設定のあいだ、デプロイ系のジョブはスキップされます（検証は常に走ります）。

## ライセンス

[MIT](LICENSE)。同梱の日本語フォント（Noto Sans JP）は [SIL Open Font License](go/internal/repo/assets/fonts/OFL.txt) です。
