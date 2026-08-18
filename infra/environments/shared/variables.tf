variable "aws_region" {
  type    = string
  default = "us-east-1"
}
variable "project" {
  type    = string
  default = "middleware-boletos"
}
variable "repository" {
  type    = string
  default = "kaiorocha/middleware-boletos"
}
variable "environment" {
  type = string
  validation {
    condition     = contains(["develop", "production"], var.environment)
    error_message = "environment must be develop or production"
  }
}
variable "vpc_cidr" {
  type = string
}
variable "container_port" {
  type    = number
  default = 8080
}
variable "initial_image_tag" {
  type        = string
  default     = "bootstrap"
  description = "An immutable image tag that already exists in ECR before the first service creation."
}
variable "ecs_cpu" {
  type    = number
  default = 256
}
variable "ecs_memory" {
  type    = number
  default = 512
}
variable "cpu_architecture" {
  type    = string
  default = "X86_64"
  validation {
    condition     = contains(["X86_64", "ARM64"], var.cpu_architecture)
    error_message = "Use X86_64 or ARM64."
  }
}
variable "ecs_desired_count" {
  type    = number
  default = 1
}
variable "ecs_min_capacity" {
  type    = number
  default = 1
}
variable "ecs_max_capacity" {
  type = number
}
variable "rds_instance_class" {
  type    = string
  default = "db.t4g.micro"
}
variable "rds_allocated_storage" {
  type    = number
  default = 20
}
variable "rds_max_allocated_storage" {
  type    = number
  default = 100
}
variable "rds_multi_az" {
  type    = bool
  default = false
}
variable "rds_backup_retention_period" {
  type = number
}
variable "rds_deletion_protection" {
  type = bool
}
variable "rds_skip_final_snapshot" {
  type = bool
}
variable "database_username" {
  type    = string
  default = "middleware"
}
variable "database_name" {
  type    = string
  default = "middleware_boletos"
}
variable "jwt_issuer" {
  type    = string
  default = "middleware-boletos"
}
variable "jwt_audience" {
  type    = string
  default = "middleware-boletos-api"
}
variable "cors_allowed_origins" {
  type = string
}
variable "log_retention_days" {
  type = number
}
variable "enable_basic_alarms" {
  type    = bool
  default = false
}
variable "enable_custom_domain" {
  type    = bool
  default = false
}
variable "domain_name" {
  type    = string
  default = ""
}
variable "api_subdomain" {
  type    = string
  default = "api"
}
variable "app_subdomain" {
  type    = string
  default = "app"
}
variable "route53_zone_id" {
  type    = string
  default = ""
}
variable "enable_amplify" {
  type        = bool
  default     = false
  description = "Reserved: Amplify is intentionally not provisioned in this phase."
}
