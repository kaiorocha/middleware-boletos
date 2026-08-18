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

Notes:
- Migrations must run before new app receives traffic. The application no longer runs migrations automatically during startup; run `migrate` explicitly.
- If migrations fail, abort deploy and rollback to previous image.
- Use image rollback for quick revert.
- Ensure secrets (DB, JWT, provider credentials) are set in environment or secret manager for each environment (staging/production).
- Do not share staging and production databases or credentials.
