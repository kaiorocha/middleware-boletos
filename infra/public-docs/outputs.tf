output "docs_bucket_name" { value = aws_s3_bucket.docs.id }
output "cloudfront_distribution_id" { value = aws_cloudfront_distribution.docs.id }
output "cloudfront_domain_name" { value = aws_cloudfront_distribution.docs.domain_name }
output "docs_url" { value = var.enable_custom_domain ? "https://${local.fqdn}" : "https://${aws_cloudfront_distribution.docs.domain_name}" }
output "publisher_role_arn" { value = aws_iam_role.publisher.arn }
