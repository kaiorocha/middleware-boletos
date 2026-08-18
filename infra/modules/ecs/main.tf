variable "name" {
  type = string
}
variable "region" {
  type = string
}
variable "subnet_ids" {
  type = list(string)
}
variable "security_group_id" {
  type = string
}
variable "target_group_arn" {
  type = string
}
variable "execution_role_arn" {
  type = string
}
variable "task_role_arn" {
  type = string
}
variable "ecr_repository_url" {
  type = string
}
variable "image_tag" {
  type = string
}
variable "secret_arn" {
  type = string
}
variable "log_group_name" {
  type = string
}
variable "cpu" {
  type = number
}
variable "memory" {
  type = number
}
variable "cpu_architecture" {
  type = string
}
variable "desired_count" {
  type = number
}
variable "min_capacity" {
  type = number
}
variable "max_capacity" {
  type = number
}
variable "container_port" {
  type = number
}
variable "environment" {
  type = string
}
variable "enable_admin_bootstrap" {
  type = bool
}
variable "tags" {
  type = map(string)
}

resource "aws_ecs_cluster" "this" {
  name = var.name
  setting {
    name  = "containerInsights"
    value = "enabled"
  }
  tags = var.tags
}
resource "aws_ecs_task_definition" "api" {
  family                   = "${var.name}-api"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = var.cpu
  memory                   = var.memory
  execution_role_arn       = var.execution_role_arn
  task_role_arn            = var.task_role_arn
  runtime_platform {
    operating_system_family = "LINUX"
    cpu_architecture        = var.cpu_architecture
  }
  container_definitions = jsonencode([{
    name = "api", image = "${var.ecr_repository_url}:${var.image_tag}", essential = true
    portMappings = [{
      containerPort = var.container_port, hostPort = var.container_port, protocol = "tcp"
    }]
    environment = [{
      name = "APP_ENV", value = var.environment
      }, {
      name = "PORT", value = tostring(var.container_port)
      }, {
      name = "ENABLE_ADMIN_BOOTSTRAP", value = tostring(var.enable_admin_bootstrap)
    }]
    secrets = concat(
      [for key in ["DATABASE_URL", "JWT_SECRET", "JWT_ISSUER", "JWT_AUDIENCE", "CORS_ALLOWED_ORIGINS"] : {
        name = key, valueFrom = "${var.secret_arn}:${key}::"
      }],
      var.enable_admin_bootstrap ? [for key in ["BOOTSTRAP_ADMIN_EMAIL", "BOOTSTRAP_ADMIN_NAME", "BOOTSTRAP_ADMIN_PASSWORD"] : {
        name = key, valueFrom = "${var.secret_arn}:${key}::"
      }] : []
    )
    logConfiguration = {
      logDriver = "awslogs", options = {
        awslogs-group = var.log_group_name, awslogs-region = var.region, awslogs-stream-prefix = "api"
      }
    }
    healthCheck = {
      command = ["CMD-SHELL", "wget -qO- --tries=1 http://localhost:${var.container_port}/health || exit 1"], interval = 30, timeout = 5, retries = 3, startPeriod = 30
    }

  }])
  tags = var.tags
}
resource "aws_ecs_service" "api" {
  name                               = "api"
  cluster                            = aws_ecs_cluster.this.id
  task_definition                    = aws_ecs_task_definition.api.arn
  desired_count                      = var.desired_count
  launch_type                        = "FARGATE"
  platform_version                   = "LATEST"
  health_check_grace_period_seconds  = 60
  deployment_minimum_healthy_percent = 100
  deployment_maximum_percent         = 200
  deployment_circuit_breaker {
    enable   = true
    rollback = true
  }
  network_configuration {
    subnets          = var.subnet_ids
    security_groups  = [var.security_group_id]
    assign_public_ip = true
  }
  load_balancer {
    target_group_arn = var.target_group_arn
    container_name   = "api"
    container_port   = var.container_port
  }
  tags = var.tags

  # Deployment ownership is intentionally split:
  # - Terraform owns the service topology and its deployment safeguards.
  # - GitHub Actions registers and deploys immutable task definition revisions.
  # - Application Auto Scaling owns the running desired count.
  # Without these exceptions, a later infrastructure apply could roll the
  # service back to Terraform's bootstrap revision or reset a scaled service.
  lifecycle {
    ignore_changes = [
      task_definition,
      desired_count,
    ]
  }
}
resource "aws_appautoscaling_target" "api" {
  max_capacity       = var.max_capacity
  min_capacity       = var.min_capacity
  resource_id        = "service/${aws_ecs_cluster.this.name}/${aws_ecs_service.api.name}"
  scalable_dimension = "ecs:service:DesiredCount"
  service_namespace  = "ecs"

  # Runtime ownership is deliberate: the scheduler and deployment workflow
  # toggle only min_capacity. Terraform continues to own max_capacity and all
  # other target attributes.
  lifecycle {
    ignore_changes = [min_capacity]
  }
}
resource "aws_appautoscaling_policy" "cpu" {
  name               = "${var.name}-cpu"
  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.api.resource_id
  scalable_dimension = aws_appautoscaling_target.api.scalable_dimension
  service_namespace  = aws_appautoscaling_target.api.service_namespace
  target_tracking_scaling_policy_configuration {
    target_value       = 60
    scale_out_cooldown = 60
    scale_in_cooldown  = 300
    predefined_metric_specification {
      predefined_metric_type = "ECSServiceAverageCPUUtilization"
    }
  }
}
resource "aws_appautoscaling_policy" "memory" {
  name               = "${var.name}-memory"
  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.api.resource_id
  scalable_dimension = aws_appautoscaling_target.api.scalable_dimension
  service_namespace  = aws_appautoscaling_target.api.service_namespace
  target_tracking_scaling_policy_configuration {
    target_value       = 70
    scale_out_cooldown = 60
    scale_in_cooldown  = 300
    predefined_metric_specification {
      predefined_metric_type = "ECSServiceAverageMemoryUtilization"
    }
  }
}
output "cluster_name" {
  value = aws_ecs_cluster.this.name
}
output "service_name" {
  value = aws_ecs_service.api.name
}
output "task_definition_arn" {
  value = aws_ecs_task_definition.api.arn
}
output "service_arn" {
  value = aws_ecs_service.api.id
}
output "scalable_resource_id" {
  value = aws_appautoscaling_target.api.resource_id
}
