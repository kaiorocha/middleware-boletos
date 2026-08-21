Deployment - middleware-boletos

Sequence (minimal):
1. Build backend binary and frontend static assets.
2. Push images to registry (tag by commit/sha).
3. Backup production database before schema changes.
4. Run database migrations (controlled, single-run per deploy). Example:

   docker run --rm \
     --env-file ./prod.env \
     <registry>/middleware-boletos-backend:<tag> \
     migrate

5. If migrations fail, abort deploy and investigate. Do not promote the new image.
6. Deploy backend image (start new instance but ensure readiness checks pass).
7. Verify /health and /ready on new instances.
8. Deploy frontend image/host (point to new backend API URL if changed).
9. Run smoke tests (login, list tenants, create proposal boleto with Mock/staging, do not emit real boleto in production).

Frontend deploys use an independent ECS `web` service and ECR repository in the same cluster as the API. The shared ALB sends `/api/*`, `/health`, and `/ready` to the API and sends application traffic to `web`. With custom domains, `api.<domain>` and `app.<domain>` share the same certificate and ALB. Frontend images use immutable commit SHA tags and are deployed by the reusable frontend workflow.

Notes:
- Migrations must run before new app receives traffic. The application no longer runs migrations automatically during startup; run `migrate` explicitly.
- If migrations fail, abort deploy and rollback to previous image.
- Use image rollback for quick revert.
- Ensure secrets (DB, JWT, provider credentials) are set in environment or secret manager for each environment (staging/production).
- Do not share staging and production databases or credentials.

Develop/HML deploys include environment startup after the immutable image is pushed and before the candidate task definition and migration are run. The reusable control script starts a stopped RDS instance, waits up to roughly 20 minutes for `available`, restores API and web ECS autoscaling minimum and desired count to 1, and only then allows deployment to continue. Backend smoke tests require both `/health` and `/ready` to return HTTP 200; frontend smoke tests require the public static endpoint `/web-health` to return HTTP 200.

Production does not execute this startup step and has no shutdown scheduler; it remains continuously active. The environment guard in the control script rejects any value other than `develop` before issuing AWS mutation calls.
