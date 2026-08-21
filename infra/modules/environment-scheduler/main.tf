variable "name" { type = string }
variable "environment" {
  type = string
  validation {
    condition     = var.environment == "develop"
    error_message = "The environment scheduler is restricted to develop."
  }
}
variable "aws_region" { type = string }
variable "shutdown_time" { type = string }
variable "shutdown_timezone" { type = string }
variable "ecs_cluster_name" { type = string }
variable "ecs_service_name" { type = string }
variable "ecs_service_arn" { type = string }
variable "scalable_resource_id" { type = string }
variable "web_ecs_service_name" { type = string }
variable "web_ecs_service_arn" { type = string }
variable "web_scalable_resource_id" { type = string }
variable "rds_instance_arn" { type = string }
variable "rds_instance_id" { type = string }
variable "log_retention_days" { type = number }
variable "tags" { type = map(string) }

locals {
  lambda_name         = "${var.name}-environment-control"
  schedule_name       = "${var.name}-daily-shutdown"
  shutdown_hour       = tonumber(split(":", var.shutdown_time)[0])
  shutdown_minute     = tonumber(split(":", var.shutdown_time)[1])
  schedule_expression = "cron(${local.shutdown_minute} ${local.shutdown_hour} * * ? *)"
}

data "archive_file" "lambda" {
  type        = "zip"
  source_file = "${path.module}/lambda/environment_control.py"
  output_path = "${path.root}/.terraform/${local.lambda_name}.zip"
}

data "aws_iam_policy_document" "lambda_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "lambda" {
  name               = "${local.lambda_name}-role"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume.json
  tags               = var.tags
}

data "aws_iam_policy_document" "lambda" {
  statement {
    sid       = "DevelopEcsServiceOnly"
    actions   = ["ecs:DescribeServices", "ecs:UpdateService"]
    resources = [var.ecs_service_arn, var.web_ecs_service_arn]
  }
  # Application Auto Scaling does not support resource-level permissions for
  # these API calls. The Lambda also validates the exact develop resource ID.
  statement {
    sid       = "DevelopScalingTargetOnly"
    actions   = ["application-autoscaling:DescribeScalableTargets", "application-autoscaling:RegisterScalableTarget"]
    resources = ["*"]
  }
  statement {
    sid       = "DevelopRdsOnly"
    actions   = ["rds:DescribeDBInstances", "rds:StartDBInstance", "rds:StopDBInstance"]
    resources = [var.rds_instance_arn]
  }
  statement {
    sid       = "LambdaLogs"
    actions   = ["logs:CreateLogStream", "logs:PutLogEvents"]
    resources = ["${aws_cloudwatch_log_group.lambda.arn}:*"]
  }
}

resource "aws_iam_role_policy" "lambda" {
  name   = "develop-environment-control"
  role   = aws_iam_role.lambda.id
  policy = data.aws_iam_policy_document.lambda.json
}

resource "aws_cloudwatch_log_group" "lambda" {
  name              = "/aws/lambda/${local.lambda_name}"
  retention_in_days = var.log_retention_days
  tags              = var.tags
}

resource "aws_lambda_function" "control" {
  function_name    = local.lambda_name
  role             = aws_iam_role.lambda.arn
  handler          = "environment_control.handler"
  runtime          = "python3.13"
  filename         = data.archive_file.lambda.output_path
  source_code_hash = data.archive_file.lambda.output_base64sha256
  timeout          = 900
  memory_size      = 128
  environment {
    variables = {
      ALLOWED_ENVIRONMENT      = "develop"
      ECS_CLUSTER              = var.ecs_cluster_name
      ECS_SERVICE              = var.ecs_service_name
      SCALABLE_RESOURCE_ID     = var.scalable_resource_id
      WEB_ECS_SERVICE          = var.web_ecs_service_name
      WEB_SCALABLE_RESOURCE_ID = var.web_scalable_resource_id
      RDS_INSTANCE_ID          = var.rds_instance_id
    }
  }
  depends_on = [aws_cloudwatch_log_group.lambda, aws_iam_role_policy.lambda]
  tags       = var.tags
}

data "aws_iam_policy_document" "scheduler_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["scheduler.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "scheduler" {
  name               = "${local.schedule_name}-role"
  assume_role_policy = data.aws_iam_policy_document.scheduler_assume.json
  tags               = var.tags
}

resource "aws_iam_role_policy" "scheduler" {
  name = "invoke-develop-control"
  role = aws_iam_role.scheduler.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow", Action = ["lambda:InvokeFunction"], Resource = [aws_lambda_function.control.arn]
    }]
  })
}

resource "aws_scheduler_schedule" "shutdown" {
  name                         = local.schedule_name
  schedule_expression          = local.schedule_expression
  schedule_expression_timezone = var.shutdown_timezone
  flexible_time_window { mode = "OFF" }
  target {
    arn      = aws_lambda_function.control.arn
    role_arn = aws_iam_role.scheduler.arn
    input    = jsonencode({ action = "stop", environment = "develop" })
    retry_policy {
      maximum_event_age_in_seconds = 3600
      maximum_retry_attempts       = 2
    }
  }
}

output "schedule_name" { value = aws_scheduler_schedule.shutdown.name }
output "lambda_name" { value = aws_lambda_function.control.function_name }
