Runbook - middleware-boletos

Startup
- Ensure DATABASE_URL, JWT_SECRET, JWT_ISSUER, JWT_AUDIENCE, CORS_ALLOWED_ORIGINS and PORT are set in production.
- Build and run container or start backend binary.
- Check `GET /health` -> 200
- Check `GET /ready` -> 200 when DB reachable

Health
- /health: liveness only, returns {"data":{"status":"ok"}}
- /ready: readiness, checks DB via PingContext with 2s timeout; returns 200 when ready, 503 when not.

Database unavailable
- If /ready returns 503, database is unavailable. Investigate DB, network, credentials.
- Do not route traffic to instances failing readiness.

Provider unavailable
- Provider outages do not affect /ready. Providers are external and logged separately; follow provider runbook.

Migration failure
- Migrations are NOT executed during storage.Connect. They must be run explicitly using the `migrate` command before routing traffic to a new release.
- If migrations fail when running `migrate`, do not deploy the new image; investigate and rollback as needed.

Rollback
- Use previous image to rollback.
- Prefer backward-compatible migrations; do not rely on DOWN migrations automatically.

Logs
- Uses structured logs (slog). Key events:
  - application_starting
  - application_started
  - application_shutdown_started
  - application_shutdown_completed
- Sensitive values are not logged (secrets removed from code and .env.example).

Secrets
- Do not store secrets in repo. Use environment or secret manager.

Common errors
- "auth_config_invalid": missing or invalid JWT config in production
- "db_connect_error": DB connection failed
- "server_failed": server runtime error

Develop / HML on-demand

Start HML
- Run the **Develop Environment Control** workflow with `action=start`.
- It starts/waits for RDS, restores autoscaling min=1 and ECS desired=1, waits for ECS stability, then checks `/health` and `/ready`.

Stop HML
- Run the same workflow with `action=stop`, or wait for the daily schedule (20:00 `America/Sao_Paulo` by default).
- The operation sets autoscaling min=0, ECS desired=0, waits for tasks to stop, and then stops RDS. Repeating it is safe.

HML did not start
- Run `action=status` and inspect the workflow and `/aws/lambda/middleware-boletos-develop-environment-control` logs.
- Confirm the GitHub develop environment has `AWS_ROLE_ARN` and `TERRAFORM_STATE_BUCKET`, and that the OIDC role can control the develop ECS scalable target/service and RDS instance.

RDS stuck in starting
- Check RDS events and storage/maintenance status in AWS. Do not issue another start while it is transitional. The workflow allows about 20 minutes and fails safely if the instance never becomes available.

ECS has no tasks
- Run `action=status`. If RDS is available but min/desired remain 0, run `action=start`; otherwise inspect ECS service events, task execution-role access to Secrets Manager, image availability, and target health.

Schedule did not run
- Check that `enable_scheduled_shutdown=true` in develop, inspect the EventBridge Scheduler execution metrics and the Lambda log group, and verify the scheduler invoke role. The schedule uses the configured IANA timezone, not a manually converted UTC offset.
- RDS may be automatically restarted by AWS after up to 7 days stopped. The following daily schedule will stop it again.
