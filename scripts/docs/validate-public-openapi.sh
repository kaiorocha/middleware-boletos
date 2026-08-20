#!/usr/bin/env bash
set -euo pipefail

SPEC="${1:-docs/openapi/middleware-boletos-public.openapi.json}"
node scripts/docs/validate-public-openapi.mjs "$SPEC"
npx --yes @redocly/cli@1.34.5 lint "$SPEC"
