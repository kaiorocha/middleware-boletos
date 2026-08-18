variable "aws_region" {
  type    = string
  default = "us-east-1"
}
variable "state_bucket_name" {
  type = string
}
variable "github_repository" {
  type    = string
  default = "kaiorocha/middleware-boletos"
}
variable "github_oidc_provider_arn" {
  type        = string
  default     = ""
  description = "Existing provider ARN. Leave empty to create it."
}
