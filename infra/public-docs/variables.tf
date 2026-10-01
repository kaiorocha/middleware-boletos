variable "aws_region" {
  description = "AWS region. CloudFront ACM certificates must be in us-east-1."
  type        = string
  default     = "us-east-1"

  validation {
    condition     = var.aws_region == "us-east-1"
    error_message = "public-docs must run in us-east-1 because CloudFront uses ACM there."
  }
}

variable "environment" {
  description = "Environment tag and bucket suffix."
  type        = string
  default     = "production"
}

variable "enable_custom_domain" {
  description = "Provision ACM, validation records and the Route53 alias."
  type        = bool
  default     = false
}

variable "domain_name" {
  description = "Apex domain, for example example.com."
  type        = string
  default     = ""
}

variable "docs_subdomain" {
  description = "Documentation subdomain label."
  type        = string
  default     = "docs"
}

variable "route53_zone_id" {
  description = "Existing Route53 hosted zone ID; no zone is created."
  type        = string
  default     = ""
}

variable "github_oidc_provider_arn" {
  description = "ARN of the existing GitHub Actions OIDC provider."
  type        = string
}

variable "github_repository" {
  description = "GitHub owner/repository allowed to assume the publication role."
  type        = string
  default     = "kaiorocha/middleware-boletos"
}
