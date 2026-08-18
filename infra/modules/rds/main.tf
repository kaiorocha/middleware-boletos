variable "name" {
  type = string
}
variable "subnet_ids" {
  type = list(string)
}
variable "security_group_id" {
  type = string
}
variable "instance_class" {
  type = string
}
variable "allocated_storage" {
  type = number
}
variable "max_allocated_storage" {
  type = number
}
variable "multi_az" {
  type = bool
}
variable "backup_retention_period" {
  type = number
}
variable "deletion_protection" {
  type = bool
}
variable "skip_final_snapshot" {
  type = bool
}
variable "username" {
  type = string
}
variable "password" {
  type      = string
  sensitive = true
}
variable "database_name" {
  type = string
}
variable "engine_major_version" {
  type        = string
  default     = "16"
  description = "PostgreSQL major version. The latest available minor version is selected in the target AWS region."
}
variable "tags" {
  type = map(string)
}

data "aws_rds_engine_version" "postgres" {
  engine  = "postgres"
  version = var.engine_major_version
  latest  = true
}

resource "aws_db_subnet_group" "this" {
  name       = var.name
  subnet_ids = var.subnet_ids
  tags = merge(var.tags, {
    Component = "database"
  })
}
resource "aws_db_instance" "this" {
  identifier                   = var.name
  engine                       = "postgres"
  engine_version               = data.aws_rds_engine_version.postgres.version_actual
  instance_class               = var.instance_class
  allocated_storage            = var.allocated_storage
  max_allocated_storage        = var.max_allocated_storage
  storage_type                 = "gp3"
  storage_encrypted            = true
  db_name                      = var.database_name
  username                     = var.username
  password                     = var.password
  port                         = 5432
  multi_az                     = var.multi_az
  publicly_accessible          = false
  db_subnet_group_name         = aws_db_subnet_group.this.name
  vpc_security_group_ids       = [var.security_group_id]
  backup_retention_period      = var.backup_retention_period
  deletion_protection          = var.deletion_protection
  skip_final_snapshot          = var.skip_final_snapshot
  final_snapshot_identifier    = var.skip_final_snapshot ? null : "${var.name}-final"
  auto_minor_version_upgrade   = true
  apply_immediately            = false
  copy_tags_to_snapshot        = true
  performance_insights_enabled = false
  tags = merge(var.tags, {
    Name = var.name, Component = "database"
  })
}
output "address" {
  value = aws_db_instance.this.address
}
output "endpoint" {
  value = aws_db_instance.this.endpoint
}
output "port" {
  value = aws_db_instance.this.port
}
output "arn" {
  value = aws_db_instance.this.arn
}
output "identifier" {
  value = aws_db_instance.this.identifier
}
