# Infraestrutura da documentação pública

Root Terraform isolado para o portal público. Provisiona bucket S3 privado com Block Public Access, versionamento e SSE-S3; CloudFront com OAC, HTTPS, cache e headers de segurança; e uma role OIDC de publicação com acesso somente ao bucket/distribuição. Não há WAF, logs, banco, Lambda, NAT nem bucket público.

O state remoto usa a chave exclusiva `middleware-boletos/public-docs/terraform.tfstate` no bucket já existente. O provider OIDC também deve existir e é recebido por ARN, nunca duplicado.

## Domínio

Com `enable_custom_domain=true`, `domain_name`, `docs_subdomain` e `route53_zone_id`, o Terraform solicita o certificado ACM em `us-east-1`, valida-o na hosted zone existente e cria o alias `docs.<domínio>` para CloudFront. A hosted zone não é criada.

## Execução

Use `.github/workflows/public-docs-infra.yml` com o environment protegido `public-docs`. Pull requests executam somente validação/plan; `apply` existe apenas por despacho manual. Variáveis de infra: `AWS_ROLE_ARN`, `AWS_ACCOUNT_ID`, `TERRAFORM_STATE_BUCKET`, `GITHUB_OIDC_PROVIDER_ARN`, `PUBLIC_DOCS_DOMAIN` e `ROUTE53_ZONE_ID`. Após o primeiro apply, copie os outputs para `PUBLIC_DOCS_PUBLISH_ROLE_ARN`, `PUBLIC_DOCS_BUCKET`, `PUBLIC_DOCS_DISTRIBUTION_ID` e `PUBLIC_DOCS_URL`; o workflow de publicação também requer `PUBLIC_API_PRODUCTION_URL` e aceita `PUBLIC_API_HML_URL` opcional.

Não execute `terraform apply` localmente sem aprovação. Para rollback de conteúdo, republique um commit/tag anterior pelo workflow. O versionamento do S3 preserva versões adicionais. A arquitetura S3 + CloudFront + Route53 tende a ser de baixo custo em baixo volume; logs e WAF podem ser avaliados futuramente.
