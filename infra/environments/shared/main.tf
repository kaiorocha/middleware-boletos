terraform {
  required_version = ">= 1.10.0, < 2.0.0"
  required_providers {
    aws = {
      source = "hashicorp/aws", version = "~> 6.0"
    }
    random = {
      source = "hashicorp/random", version = "~> 3.7"
    }
    archive = {
      source = "hashicorp/archive", version = "~> 2.7"
    }

  }
  backend "s3" {
    use_lockfile = true
  }
}

provider "aws" {
  region = var.aws_region
  default_tags {
    tags = local.tags
  }
}

data "aws_availability_zones" "available" {
  state = "available"
}
locals {
  name = "${var.project}-${var.environment}"
  tags = {
    Project = var.project, Environment = var.environment, ManagedBy = "terraform", Repository = var.repository
  }
  api_fqdn = var.enable_custom_domain ? "${var.api_subdomain}.${var.domain_name}" : ""
  app_fqdn = var.enable_custom_domain ? "${var.app_subdomain}.${var.domain_name}" : ""
}

resource "random_password" "database" {
  length           = 32
  special          = true
  override_special = "!#$%&*()-_=+[]{}<>:?"
}

module "network" {
  source             = "../../modules/network"
  name               = local.name
  vpc_cidr           = var.vpc_cidr
  availability_zones = slice(data.aws_availability_zones.available.names, 0, 2)
  tags               = local.tags
}
module "security" {
  source             = "../../modules/security"
  name               = local.name
  vpc_id             = module.network.vpc_id
  container_port     = var.container_port
  web_container_port = var.web_container_port
  tags               = local.tags
}
module "ecr" {
  source          = "../../modules/ecr"
  repository_name = "middleware-boletos-backend-${var.environment}"
  tags            = local.tags
}
module "web_ecr" {
  source          = "../../modules/ecr"
  repository_name = "middleware-boletos-frontend-${var.environment}"
  component       = "web"
  tags            = local.tags
}
module "logs" {
  source         = "../../modules/cloudwatch"
  name           = var.environment
  retention_days = var.log_retention_days
  tags           = local.tags
}
module "rds" {
  source                  = "../../modules/rds"
  name                    = "${local.name}-db"
  subnet_ids              = module.network.database_subnet_ids
  security_group_id       = module.security.database_security_group_id
  instance_class          = var.rds_instance_class
  allocated_storage       = var.rds_allocated_storage
  max_allocated_storage   = var.rds_max_allocated_storage
  multi_az                = var.rds_multi_az
  backup_retention_period = var.rds_backup_retention_period
  deletion_protection     = var.rds_deletion_protection
  skip_final_snapshot     = var.rds_skip_final_snapshot
  username                = var.database_username
  password                = random_password.database.result
  database_name           = var.database_name
  tags                    = local.tags
}
module "secrets" {
  source                = "../../modules/secrets"
  name                  = local.name
  db_username           = var.database_username
  db_password           = random_password.database.result
  db_host               = module.rds.address
  db_port               = module.rds.port
  db_name               = var.database_name
  jwt_issuer            = var.jwt_issuer
  jwt_audience          = var.jwt_audience
  cors_allowed_origins  = var.cors_allowed_origins
  bootstrap_admin_email = var.bootstrap_admin_email
  bootstrap_admin_name  = var.bootstrap_admin_name
  tags                  = local.tags
}
module "iam" {
  source     = "../../modules/iam"
  name       = local.name
  secret_arn = module.secrets.secret_arn
  tags       = local.tags
}
module "dns" {
  source                    = "../../modules/dns"
  enabled                   = var.enable_custom_domain
  zone_id                   = var.route53_zone_id
  fqdn                      = local.api_fqdn
  subject_alternative_names = var.enable_custom_domain ? [local.app_fqdn] : []
  tags                      = local.tags
}
module "alb" {
  source             = "../../modules/alb"
  name               = local.name
  vpc_id             = module.network.vpc_id
  subnet_ids         = module.network.public_subnet_ids
  security_group_id  = module.security.alb_security_group_id
  container_port     = var.container_port
  web_container_port = var.web_container_port
  app_fqdn           = local.app_fqdn
  enable_https       = var.enable_custom_domain
  certificate_arn    = module.dns.certificate_arn
  tags               = local.tags
}
resource "aws_route53_record" "api" {
  count   = var.enable_custom_domain ? 1 : 0
  zone_id = var.route53_zone_id
  name    = local.api_fqdn
  type    = "A"
  alias {
    name                   = module.alb.dns_name
    zone_id                = module.alb.zone_id
    evaluate_target_health = true
  }
}
resource "aws_route53_record" "app" {
  count   = var.enable_custom_domain ? 1 : 0
  zone_id = var.route53_zone_id
  name    = local.app_fqdn
  type    = "A"
  alias {
    name                   = module.alb.dns_name
    zone_id                = module.alb.zone_id
    evaluate_target_health = true
  }
}
module "ecs" {
  source                 = "../../modules/ecs"
  name                   = local.name
  region                 = var.aws_region
  subnet_ids             = module.network.public_subnet_ids
  security_group_id      = module.security.api_security_group_id
  target_group_arn       = module.alb.target_group_arn
  execution_role_arn     = module.iam.execution_role_arn
  task_role_arn          = module.iam.task_role_arn
  ecr_repository_url     = module.ecr.repository_url
  image_tag              = var.initial_image_tag
  secret_arn             = module.secrets.secret_arn
  log_group_name         = module.logs.log_group_name
  cpu                    = var.ecs_cpu
  memory                 = var.ecs_memory
  cpu_architecture       = var.cpu_architecture
  desired_count          = var.ecs_desired_count
  min_capacity           = var.ecs_min_capacity
  max_capacity           = var.ecs_max_capacity
  container_port         = var.container_port
  environment            = var.environment
  enable_admin_bootstrap = var.enable_admin_bootstrap
  tags                   = local.tags
  web_security_group_id  = module.security.web_security_group_id
  web_target_group_arn   = module.alb.web_target_group_arn
  web_ecr_repository_url = module.web_ecr.repository_url
  web_log_group_name     = module.logs.web_log_group_name
  web_container_port     = var.web_container_port
  web_cpu                = var.web_ecs_cpu
  web_memory             = var.web_ecs_memory
  web_desired_count      = var.web_ecs_desired_count
  web_min_capacity       = var.web_ecs_min_capacity
  web_max_capacity       = var.web_ecs_max_capacity

  # Do not start tasks before the secret has an AWSCURRENT version and the
  # execution role has all policies required to retrieve it.
  depends_on = [module.secrets, module.iam]
}

module "environment_scheduler" {
  source = "../../modules/environment-scheduler"
  count  = var.enable_scheduled_shutdown ? 1 : 0

  name                     = local.name
  environment              = var.environment
  aws_region               = var.aws_region
  shutdown_time            = var.shutdown_time
  shutdown_timezone        = var.shutdown_timezone
  ecs_cluster_name         = module.ecs.cluster_name
  ecs_service_name         = module.ecs.service_name
  ecs_service_arn          = module.ecs.service_arn
  scalable_resource_id     = module.ecs.scalable_resource_id
  web_ecs_service_name     = module.ecs.web_service_name
  web_ecs_service_arn      = module.ecs.web_service_arn
  web_scalable_resource_id = module.ecs.web_scalable_resource_id
  rds_instance_arn         = module.rds.arn
  rds_instance_id          = module.rds.identifier
  log_retention_days       = 7
  tags                     = local.tags

  depends_on = [module.ecs, module.rds]
}

resource "aws_cloudwatch_metric_alarm" "ecs_cpu" {
  count               = var.enable_basic_alarms ? 1 : 0
  alarm_name          = "${local.name}-ecs-cpu-high"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  metric_name         = "CPUUtilization"
  namespace           = "AWS/ECS"
  period              = 300
  statistic           = "Average"
  threshold           = 80
  treat_missing_data  = "notBreaching"
  dimensions = {
    ClusterName = module.ecs.cluster_name, ServiceName = module.ecs.service_name
  }
  tags = local.tags
}
resource "aws_cloudwatch_metric_alarm" "ecs_memory" {
  count               = var.enable_basic_alarms ? 1 : 0
  alarm_name          = "${local.name}-ecs-memory-high"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  metric_name         = "MemoryUtilization"
  namespace           = "AWS/ECS"
  period              = 300
  statistic           = "Average"
  threshold           = 85
  treat_missing_data  = "notBreaching"
  dimensions = {
    ClusterName = module.ecs.cluster_name, ServiceName = module.ecs.service_name
  }
  tags = local.tags
}
resource "aws_cloudwatch_metric_alarm" "web_ecs_cpu" {
  count               = var.enable_basic_alarms ? 1 : 0
  alarm_name          = "${local.name}-web-ecs-cpu-high"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  metric_name         = "CPUUtilization"
  namespace           = "AWS/ECS"
  period              = 300
  statistic           = "Average"
  threshold           = 80
  treat_missing_data  = "notBreaching"
  dimensions = {
    ClusterName = module.ecs.cluster_name, ServiceName = module.ecs.web_service_name
  }
  tags = local.tags
}
resource "aws_cloudwatch_metric_alarm" "web_ecs_memory" {
  count               = var.enable_basic_alarms ? 1 : 0
  alarm_name          = "${local.name}-web-ecs-memory-high"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  metric_name         = "MemoryUtilization"
  namespace           = "AWS/ECS"
  period              = 300
  statistic           = "Average"
  threshold           = 85
  treat_missing_data  = "notBreaching"
  dimensions = {
    ClusterName = module.ecs.cluster_name, ServiceName = module.ecs.web_service_name
  }
  tags = local.tags
}
resource "aws_cloudwatch_metric_alarm" "rds_cpu" {
  count               = var.enable_basic_alarms ? 1 : 0
  alarm_name          = "${local.name}-rds-cpu-high"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  metric_name         = "CPUUtilization"
  namespace           = "AWS/RDS"
  period              = 300
  statistic           = "Average"
  threshold           = 80
  treat_missing_data  = "notBreaching"
  dimensions = {
    DBInstanceIdentifier = module.rds.identifier
  }
  tags = local.tags
}

resource "aws_cloudwatch_metric_alarm" "alb_5xx" {
  count               = var.enable_basic_alarms ? 1 : 0
  alarm_name          = "${local.name}-alb-5xx-high"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "HTTPCode_ELB_5XX_Count"
  namespace           = "AWS/ApplicationELB"
  period              = 300
  statistic           = "Sum"
  threshold           = 10
  treat_missing_data  = "notBreaching"
  dimensions          = { LoadBalancer = module.alb.arn_suffix }
  tags                = local.tags
}

resource "aws_cloudwatch_metric_alarm" "rds_storage" {
  count               = var.enable_basic_alarms ? 1 : 0
  alarm_name          = "${local.name}-rds-storage-low"
  comparison_operator = "LessThanThreshold"
  evaluation_periods  = 2
  metric_name         = "FreeStorageSpace"
  namespace           = "AWS/RDS"
  period              = 300
  statistic           = "Average"
  threshold           = 2147483648
  treat_missing_data  = "notBreaching"
  dimensions          = { DBInstanceIdentifier = module.rds.identifier }
  tags                = local.tags
}
