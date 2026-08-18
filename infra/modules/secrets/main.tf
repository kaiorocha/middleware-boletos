variable "name" {
  type = string
}
variable "db_username" {
  type = string
}
variable "db_password" {
  type      = string
  sensitive = true
}
variable "db_host" {
  type = string
}
variable "db_port" {
  type = number
}
variable "db_name" {
  type = string
}
variable "jwt_issuer" {
  type = string
}
variable "jwt_audience" {
  type = string
}
variable "cors_allowed_origins" {
  type = string
}
variable "tags" {
  type = map(string)
}

resource "random_password" "jwt" {
  length  = 64
  special = false
}

resource "aws_secretsmanager_secret" "app" {
  name                    = "${var.name}/api"
  recovery_window_in_days = 7
  tags = merge(var.tags, {
    Component = "api"
  })
}
resource "aws_secretsmanager_secret_version" "app" {
  secret_id = aws_secretsmanager_secret.app.id
  secret_string = jsonencode({
    DATABASE_URL         = "postgres://${urlencode(var.db_username)}:${urlencode(var.db_password)}@${var.db_host}:${var.db_port}/${var.db_name}?sslmode=require"
    JWT_SECRET           = random_password.jwt.result
    JWT_ISSUER           = var.jwt_issuer
    JWT_AUDIENCE         = var.jwt_audience
    CORS_ALLOWED_ORIGINS = var.cors_allowed_origins

  })
}
output "secret_arn" {
  value = aws_secretsmanager_secret.app.arn
}
