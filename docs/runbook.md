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
