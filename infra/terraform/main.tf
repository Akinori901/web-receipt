# SAM のデプロイより前に必要な「土台」だけを Terraform で持つ。
# アプリ層（Lambda / API Gateway / Cognito / S3 / CloudFront）は infra/sam/ が管理する。
#
# 同じリソースを 2 つの IaC で管理しないこと（Cognito は SAM 側にある）。
# VPC と RDS（PostgreSQL）は既存のものを使い、SAM のパラメータで渡す。

terraform {
  required_version = ">= 1.5"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
}

provider "aws" {
  region = var.region
}

locals {
  tags = {
    Project   = var.project
    ManagedBy = "terraform"
  }
}

# ── Go の Lambda コンテナイメージの置き場 ─────────────────────
# 環境（dev / stg / prod）で共有し、イメージはコミット SHA のタグで区別する。
# 名前は .github/workflows/deploy.yml の ECR_REPOSITORY と揃える。
resource "aws_ecr_repository" "go" {
  name                 = "${var.project}-go"
  image_tag_mutability = "IMMUTABLE" # 同じタグの上書きを禁止（デプロイしたものを後から差し替えさせない）

  image_scanning_configuration {
    scan_on_push = true
  }

  encryption_configuration {
    encryption_type = "AES256"
  }

  tags = local.tags
}

# 古いイメージを溜め込まない（ストレージ課金を抑える）
resource "aws_ecr_lifecycle_policy" "go" {
  repository = aws_ecr_repository.go.name
  policy = jsonencode({
    rules = [{
      rulePriority = 1
      description  = "直近 ${var.keep_images} 個だけ残す"
      selection = {
        tagStatus   = "any"
        countType   = "imageCountMoreThan"
        countNumber = var.keep_images
      }
      action = { type = "expire" }
    }]
  })
}
