variable "repository_name" {
  type = string
}
variable "tags" {
  type = map(string)
}
variable "image_retention_count" {
  type    = number
  default = 25
}
variable "component" {
  type    = string
  default = "api"
}

resource "aws_ecr_repository" "this" {
  name                 = var.repository_name
  image_tag_mutability = "IMMUTABLE"
  encryption_configuration {
    encryption_type = "AES256"
  }
  image_scanning_configuration {
    scan_on_push = true
  }
  tags = merge(var.tags, {
    Component = var.component
  })
}

resource "aws_ecr_lifecycle_policy" "this" {
  repository = aws_ecr_repository.this.name
  policy = jsonencode({
    rules = [{
      rulePriority = 1, description = "Keep recent images", selection = {
        tagStatus = "any", countType = "imageCountMoreThan", countNumber = var.image_retention_count
        }, action = {
        type = "expire"
      }
    }]
  })
}

output "repository_url" {
  value = aws_ecr_repository.this.repository_url
}
output "repository_arn" {
  value = aws_ecr_repository.this.arn
}
