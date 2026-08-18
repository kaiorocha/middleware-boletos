data "aws_caller_identity" "current" {}
resource "aws_s3_bucket" "state" {
  bucket = var.state_bucket_name
  lifecycle {
    prevent_destroy = true
  }
  tags = {
    Project = "middleware-boletos", Environment = "shared", ManagedBy = "terraform", Repository = var.github_repository, Component = "terraform-state"
  }
}
resource "aws_s3_bucket_versioning" "state" {
  bucket = aws_s3_bucket.state.id
  versioning_configuration {
    status = "Enabled"
  }
}
resource "aws_s3_bucket_server_side_encryption_configuration" "state" {
  bucket = aws_s3_bucket.state.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
    bucket_key_enabled = true
  }
}
resource "aws_s3_bucket_public_access_block" "state" {
  bucket                  = aws_s3_bucket.state.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}
resource "aws_s3_bucket_lifecycle_configuration" "state" {
  bucket = aws_s3_bucket.state.id
  rule {
    id     = "old-versions"
    status = "Enabled"
    filter {}
    noncurrent_version_expiration {
      noncurrent_days           = 90
      newer_noncurrent_versions = 20
    }
  }
}
resource "aws_iam_openid_connect_provider" "github" {
  count           = var.github_oidc_provider_arn == "" ? 1 : 0
  url             = "https://token.actions.githubusercontent.com"
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = ["6938fd4d98bab03faadb97b34396831e3780aea1"]
  tags = {
    Project = "middleware-boletos", Environment = "shared", ManagedBy = "terraform", Repository = var.github_repository
  }
}
locals {
  oidc_arn = var.github_oidc_provider_arn != "" ? var.github_oidc_provider_arn : aws_iam_openid_connect_provider.github[0].arn
}
data "aws_iam_policy_document" "github_assume" {
  statement {
    actions = ["sts:AssumeRoleWithWebIdentity"]
    principals {
      type        = "Federated"
      identifiers = [local.oidc_arn]
    }
    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:aud"
      values   = ["sts.amazonaws.com"]
    }
    condition {
      test     = "StringLike"
      variable = "token.actions.githubusercontent.com:sub"
      values   = ["repo:${var.github_repository}:environment:develop", "repo:${var.github_repository}:environment:production", "repo:${var.github_repository}:pull_request"]
    }

  }
}
resource "aws_iam_role" "github" {
  name               = "github-actions-terraform-middleware-boletos"
  assume_role_policy = data.aws_iam_policy_document.github_assume.json
}
resource "aws_iam_role_policy" "github" {
  name = "middleware-boletos-infrastructure"
  role = aws_iam_role.github.id
  policy = jsonencode({
    Version = "2012-10-17", Statement = [
      {
        Effect = "Allow", Action = ["s3:ListBucket"], Resource = [aws_s3_bucket.state.arn]
      },
      {
        Effect = "Allow", Action = ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"], Resource = ["${aws_s3_bucket.state.arn}/middleware-boletos/*"]
      },
      {
        Effect = "Allow", Action = ["ec2:*", "ecs:*", "ecr:*", "elasticloadbalancing:*", "application-autoscaling:*", "rds:*", "logs:*", "cloudwatch:*", "secretsmanager:*", "acm:*", "route53:*", "iam:Get*", "iam:List*", "iam:CreateRole", "iam:DeleteRole", "iam:TagRole", "iam:UntagRole", "iam:PassRole", "iam:PutRolePolicy", "iam:DeleteRolePolicy", "iam:AttachRolePolicy", "iam:DetachRolePolicy"], Resource = "*"
      }
    ]
  })
}
output "state_bucket" {
  value = aws_s3_bucket.state.id
}
output "github_actions_role_arn" {
  value = aws_iam_role.github.arn
}
