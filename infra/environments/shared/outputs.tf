output "vpc_id" {
  value = module.network.vpc_id
}
output "public_subnet_ids" {
  value = module.network.public_subnet_ids
}
output "database_subnet_ids" {
  value = module.network.database_subnet_ids
}
output "alb_dns_name" {
  value = module.alb.dns_name
}
output "ecs_cluster_name" {
  value = module.ecs.cluster_name
}
output "ecs_service_name" {
  value = module.ecs.service_name
}
output "web_ecs_service_name" {
  value = module.ecs.web_service_name
}
output "ecs_task_definition_arn" {
  value = module.ecs.task_definition_arn
}
output "web_ecs_task_definition_arn" {
  value = module.ecs.web_task_definition_arn
}
output "api_security_group_id" {
  value = module.security.api_security_group_id
}
output "web_security_group_id" {
  value = module.security.web_security_group_id
}
output "ecr_repository_url" {
  value = module.ecr.repository_url
}
output "web_ecr_repository_url" {
  value = module.web_ecr.repository_url
}
output "rds_endpoint" {
  value = module.rds.endpoint
}
output "cloudwatch_log_group" {
  value = module.logs.log_group_name
}
output "api_url" {
  value = var.enable_custom_domain ? "https://${local.api_fqdn}" : "http://${module.alb.dns_name}"
}
output "app_url" {
  value = var.enable_custom_domain ? "https://${local.app_fqdn}" : "http://${module.alb.dns_name}"
}
output "environment_scheduler_name" {
  value = try(module.environment_scheduler[0].schedule_name, null)
}
output "shutdown_schedule" {
  value = var.enable_scheduled_shutdown ? var.shutdown_time : null
}
output "shutdown_timezone" {
  value = var.enable_scheduled_shutdown ? var.shutdown_timezone : null
}
output "environment_control_lambda_name" {
  value = try(module.environment_scheduler[0].lambda_name, null)
}
output "rds_instance_id" {
  value = module.rds.identifier
}
