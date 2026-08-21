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
services=("$SERVICE")
if [[ -n "${WEB_SERVICE:-}" && "$WEB_SERVICE" != "$SERVICE" ]]; then
  services+=("$WEB_SERVICE")
fi

rds_status() {
  aws rds describe-db-instances --db-instance-identifier "$RDS_INSTANCE_ID" \
    --query 'DBInstances[0].DBInstanceStatus' --output text
}

scaling_values() {
  local service="$1" resource_id="service/${CLUSTER}/$1"
  aws application-autoscaling describe-scalable-targets --service-namespace ecs \
    --resource-ids "$resource_id" --scalable-dimension ecs:service:DesiredCount \
    --query 'ScalableTargets[0].[MinCapacity,MaxCapacity]' --output text
}

show_status() {
  local rds service ecs_values scaling
  rds="$(rds_status)"
  for service in "${services[@]}"; do
    ecs_values="$(aws ecs describe-services --cluster "$CLUSTER" --services "$service" \
      --query 'services[0].[desiredCount,runningCount]' --output text)"
    scaling="$(scaling_values "$service")"
    printf 'environment=develop service=%s rds_status=%s ecs_desired=%s ecs_running=%s autoscaling_min=%s autoscaling_max=%s\n' \
      "$service" "$rds" "$(awk '{print $1}' <<<"$ecs_values")" "$(awk '{print $2}' <<<"$ecs_values")" \
      "$(awk '{print $1}' <<<"$scaling")" "$(awk '{print $2}' <<<"$scaling")"
  done
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

if [[ "$action" == "stop" ]]; then
  for service in "${services[@]}"; do
    read -r min_capacity max_capacity <<<"$(scaling_values "$service")"
    aws application-autoscaling register-scalable-target --service-namespace ecs \
      --resource-id "service/${CLUSTER}/${service}" --scalable-dimension ecs:service:DesiredCount \
      --min-capacity 0 --max-capacity "$max_capacity" >/dev/null
    aws ecs update-service --cluster "$CLUSTER" --service "$service" --desired-count 0 >/dev/null
  done
  aws ecs wait services-stable --cluster "$CLUSTER" --services "${services[@]}"
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
  for service in "${services[@]}"; do
    read -r min_capacity max_capacity <<<"$(scaling_values "$service")"
    aws application-autoscaling register-scalable-target --service-namespace ecs \
      --resource-id "service/${CLUSTER}/${service}" --scalable-dimension ecs:service:DesiredCount \
      --min-capacity 1 --max-capacity "$max_capacity" >/dev/null
    aws ecs update-service --cluster "$CLUSTER" --service "$service" --desired-count 1 >/dev/null
  done
fi

show_status
