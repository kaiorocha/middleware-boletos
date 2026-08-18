#!/usr/bin/env bash
set -euo pipefail

action="${1:-}"
environment="${2:-develop}"

if [[ "$environment" != "develop" ]]; then
  echo "REFUSED: environment control is restricted to develop" >&2
  exit 64
fi
if [[ ! "$action" =~ ^(start|stop|status)$ ]]; then
  echo "Usage: $0 {start|stop|status} develop" >&2
  exit 64
fi

: "${CLUSTER:?CLUSTER is required}"
: "${SERVICE:?SERVICE is required}"
: "${RDS_INSTANCE_ID:?RDS_INSTANCE_ID is required}"
resource_id="service/${CLUSTER}/${SERVICE}"

rds_status() {
  aws rds describe-db-instances --db-instance-identifier "$RDS_INSTANCE_ID" \
    --query 'DBInstances[0].DBInstanceStatus' --output text
}

scaling_values() {
  aws application-autoscaling describe-scalable-targets --service-namespace ecs \
    --resource-ids "$resource_id" --scalable-dimension ecs:service:DesiredCount \
    --query 'ScalableTargets[0].[MinCapacity,MaxCapacity]' --output text
}

show_status() {
  local rds ecs_values scaling
  rds="$(rds_status)"
  ecs_values="$(aws ecs describe-services --cluster "$CLUSTER" --services "$SERVICE" \
    --query 'services[0].[desiredCount,runningCount]' --output text)"
  scaling="$(scaling_values)"
  printf 'environment=develop rds_status=%s ecs_desired=%s ecs_running=%s autoscaling_min=%s autoscaling_max=%s\n' \
    "$rds" "$(awk '{print $1}' <<<"$ecs_values")" "$(awk '{print $2}' <<<"$ecs_values")" \
    "$(awk '{print $1}' <<<"$scaling")" "$(awk '{print $2}' <<<"$scaling")"
}

wait_rds_stable() {
  local state
  for _ in {1..60}; do
    state="$(rds_status)"
    case "$state" in
      available|stopped) printf '%s\n' "$state"; return 0 ;;
      starting|stopping|backing-up|modifying) sleep 20 ;;
      *) echo "RDS is in unsupported state: $state" >&2; return 1 ;;
    esac
  done
  echo "Timed out waiting for RDS transition" >&2
  return 1
}

if [[ "$action" == "status" ]]; then
  show_status
  exit 0
fi

read -r min_capacity max_capacity <<<"$(scaling_values)"
if [[ "$action" == "stop" ]]; then
  aws application-autoscaling register-scalable-target --service-namespace ecs \
    --resource-id "$resource_id" --scalable-dimension ecs:service:DesiredCount \
    --min-capacity 0 --max-capacity "$max_capacity" >/dev/null
  aws ecs update-service --cluster "$CLUSTER" --service "$SERVICE" --desired-count 0 >/dev/null
  aws ecs wait services-stable --cluster "$CLUSTER" --services "$SERVICE"
  state="$(wait_rds_stable)"
  if [[ "$state" == "available" ]]; then
    aws rds stop-db-instance --db-instance-identifier "$RDS_INSTANCE_ID" >/dev/null
  fi
else
  state="$(wait_rds_stable)"
  if [[ "$state" == "stopped" ]]; then
    aws rds start-db-instance --db-instance-identifier "$RDS_INSTANCE_ID" >/dev/null
    aws rds wait db-instance-available --db-instance-identifier "$RDS_INSTANCE_ID"
  fi
  aws application-autoscaling register-scalable-target --service-namespace ecs \
    --resource-id "$resource_id" --scalable-dimension ecs:service:DesiredCount \
    --min-capacity 1 --max-capacity "$max_capacity" >/dev/null
  aws ecs update-service --cluster "$CLUSTER" --service "$SERVICE" --desired-count 1 >/dev/null
fi

show_status
