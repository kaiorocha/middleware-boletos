variable "name" {
  type = string
}
variable "retention_days" {
  type = number
}
variable "tags" {
  type = map(string)
}

resource "aws_cloudwatch_log_group" "api" {
  name              = "/aws/ecs/middleware-boletos/${var.name}/api"
  retention_in_days = var.retention_days
  tags = merge(var.tags, {
    Component = "api"
  })
}
resource "aws_cloudwatch_log_group" "web" {
  name              = "/aws/ecs/middleware-boletos/${var.name}/web"
  retention_in_days = var.retention_days
  tags = merge(var.tags, {
    Component = "web"
  })
}
output "log_group_name" {
  value = aws_cloudwatch_log_group.api.name
}
output "web_log_group_name" {
  value = aws_cloudwatch_log_group.web.name
}
