# Terraform（土台のみ）

SAM（`infra/sam/`）のデプロイより前に必要なものだけをここで作ります。

- 作るもの: Go の Lambda コンテナイメージを置く ECR リポジトリ（`web-receipt-go`）と、古いイメージを消すライフサイクル
- 作らないもの: Cognito・API・S3・CloudFront（SAM 側）、VPC・RDS PostgreSQL（既存のものを SAM のパラメータで渡す）

同じリソースを 2 つの IaC で管理しないため、境界はここで固定しています。

```bash
terraform init
terraform plan     # 必ず内容を確認する
terraform apply    # plan を見てから
```
