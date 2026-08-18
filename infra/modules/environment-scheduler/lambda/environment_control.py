import json
import logging
import os
import time

import boto3

LOG = logging.getLogger()
LOG.setLevel(logging.INFO)


def _clients():
    return boto3.client("ecs"), boto3.client("application-autoscaling"), boto3.client("rds")


def _configuration():
    return {
        "environment": os.environ.get("ALLOWED_ENVIRONMENT", "develop"),
        "cluster": os.environ["ECS_CLUSTER"],
        "service": os.environ["ECS_SERVICE"],
        "resource_id": os.environ["SCALABLE_RESOURCE_ID"],
        "db_id": os.environ["RDS_INSTANCE_ID"],
    }


def _status(ecs, scaling, rds, cfg):
    service = ecs.describe_services(cluster=cfg["cluster"], services=[cfg["service"]])["services"][0]
    target = scaling.describe_scalable_targets(
        ServiceNamespace="ecs", ResourceIds=[cfg["resource_id"]],
        ScalableDimension="ecs:service:DesiredCount",
    )["ScalableTargets"][0]
    database = rds.describe_db_instances(DBInstanceIdentifier=cfg["db_id"])["DBInstances"][0]
    return {
        "environment": cfg["environment"], "rds_status": database["DBInstanceStatus"],
        "ecs_desired": service["desiredCount"], "ecs_running": service["runningCount"],
        "autoscaling_min": target["MinCapacity"], "autoscaling_max": target["MaxCapacity"],
    }


def _wait_for_rds(rds, db_id, terminal, transitional, timeout=840):
    deadline = time.time() + timeout
    while time.time() < deadline:
        state = rds.describe_db_instances(DBInstanceIdentifier=db_id)["DBInstances"][0]["DBInstanceStatus"]
        if state in terminal:
            return state
        if state not in transitional:
            return state
        time.sleep(20)
    raise TimeoutError(f"RDS did not reach {terminal} before timeout")


def _set_min(scaling, cfg, minimum, maximum):
    scaling.register_scalable_target(
        ServiceNamespace="ecs", ResourceId=cfg["resource_id"],
        ScalableDimension="ecs:service:DesiredCount", MinCapacity=minimum, MaxCapacity=maximum,
    )


def _stop(ecs, scaling, rds, cfg, before):
    _set_min(scaling, cfg, 0, before["autoscaling_max"])
    ecs.update_service(cluster=cfg["cluster"], service=cfg["service"], desiredCount=0)
    ecs.get_waiter("services_stable").wait(
        cluster=cfg["cluster"], services=[cfg["service"]],
        WaiterConfig={"Delay": 15, "MaxAttempts": 40},
    )
    state = _wait_for_rds(rds, cfg["db_id"], {"available", "stopped"}, {"starting", "stopping", "backing-up", "modifying"})
    if state == "available":
        rds.stop_db_instance(DBInstanceIdentifier=cfg["db_id"])
    elif state != "stopped":
        raise RuntimeError(f"RDS cannot be stopped safely from state {state}")


def _start(ecs, scaling, rds, cfg, before):
    state = _wait_for_rds(rds, cfg["db_id"], {"available", "stopped"}, {"starting", "stopping", "backing-up", "modifying"})
    if state == "stopped":
        rds.start_db_instance(DBInstanceIdentifier=cfg["db_id"])
        _wait_for_rds(rds, cfg["db_id"], {"available"}, {"starting", "backing-up", "modifying"})
    elif state != "available":
        raise RuntimeError(f"RDS cannot be started safely from state {state}")
    _set_min(scaling, cfg, 1, before["autoscaling_max"])
    ecs.update_service(cluster=cfg["cluster"], service=cfg["service"], desiredCount=1)


def handler(event, _context):
    cfg = _configuration()
    action = str(event.get("action", "")).lower()
    requested_environment = event.get("environment")
    if cfg["environment"] != "develop" or requested_environment != "develop":
        raise ValueError("environment control is restricted to develop")
    if action not in {"start", "stop", "status"}:
        raise ValueError("action must be start, stop, or status")
    ecs, scaling, rds = _clients()
    before = _status(ecs, scaling, rds, cfg)
    if action == "stop":
        _stop(ecs, scaling, rds, cfg, before)
    elif action == "start":
        _start(ecs, scaling, rds, cfg, before)
    after = _status(ecs, scaling, rds, cfg)
    result = {"action": action, "previous": before, "final": after, "result": "success"}
    LOG.info(json.dumps(result, sort_keys=True))
    return result
