variable "enabled" {
  type = bool
}
variable "zone_id" {
  type = string
}
variable "fqdn" {
  type = string
}
variable "tags" {
  type = map(string)
}

resource "aws_acm_certificate" "api" {
  count             = var.enabled ? 1 : 0
  domain_name       = var.fqdn
  validation_method = "DNS"
  tags = merge(var.tags, {
    Component = "api"
  })
  lifecycle {
    create_before_destroy = true
  }
}
resource "aws_route53_record" "validation" {
  for_each = var.enabled ? {
    for dvo in aws_acm_certificate.api[0].domain_validation_options : dvo.domain_name => {
      name = dvo.resource_record_name, record = dvo.resource_record_value, type = dvo.resource_record_type
    }
  } : {}
  zone_id = var.zone_id
  name    = each.value.name
  type    = each.value.type
  records = [each.value.record]
  ttl     = 60
}
resource "aws_acm_certificate_validation" "api" {
  count                   = var.enabled ? 1 : 0
  certificate_arn         = aws_acm_certificate.api[0].arn
  validation_record_fqdns = [for record in aws_route53_record.validation : record.fqdn]
}
output "certificate_arn" {
  value = var.enabled ? aws_acm_certificate_validation.api[0].certificate_arn : null
}
