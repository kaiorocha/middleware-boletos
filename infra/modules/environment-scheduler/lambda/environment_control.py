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
        "services": [os.environ["ECS_SERVICE"], os.environ["WEB_ECS_SERVICE"]],
        "resource_ids": [os.environ["SCALABLE_RESOURCE_ID"], os.environ["WEB_SCALABLE_RESOURCE_ID"]],
        "db_id": os.environ["RDS_INSTANCE_ID"],
    }


def _status(ecs, scaling, rds, cfg):
    services = ecs.describe_services(cluster=cfg["cluster"], services=cfg["services"])["services"]
    targets = scaling.describe_scalable_targets(
        ServiceNamespace="ecs", ResourceIds=cfg["resource_ids"],
        ScalableDimension="ecs:service:DesiredCount",
    )["ScalableTargets"]
    database = rds.describe_db_instances(DBInstanceIdentifier=cfg["db_id"])["DBInstances"][0]
    target_by_id = {target["ResourceId"]: target for target in targets}
    service_by_name = {service["serviceName"]: service for service in services}
    return {
        "environment": cfg["environment"], "rds_status": database["DBInstanceStatus"],
        "services": {
            name: {
                "ecs_desired": service_by_name[name]["desiredCount"],
                "ecs_running": service_by_name[name]["runningCount"],
                "autoscaling_min": target_by_id[resource_id]["MinCapacity"],
                "autoscaling_max": target_by_id[resource_id]["MaxCapacity"],
            }
            for name, resource_id in zip(cfg["services"], cfg["resource_ids"])
        },
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


def _set_min(scaling, resource_id, minimum, maximum):
    scaling.register_scalable_target(
        ServiceNamespace="ecs", ResourceId=resource_id,
        ScalableDimension="ecs:service:DesiredCount", MinCapacity=minimum, MaxCapacity=maximum,
    )


def _stop(ecs, scaling, rds, cfg, before):
    for service, resource_id in zip(cfg["services"], cfg["resource_ids"]):
        _set_min(scaling, resource_id, 0, before["services"][service]["autoscaling_max"])
        ecs.update_service(cluster=cfg["cluster"], service=service, desiredCount=0)
    ecs.get_waiter("services_stable").wait(
        cluster=cfg["cluster"], services=cfg["services"],
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
    for service, resource_id in zip(cfg["services"], cfg["resource_ids"]):
        _set_min(scaling, resource_id, 1, before["services"][service]["autoscaling_max"])
        ecs.update_service(cluster=cfg["cluster"], service=service, desiredCount=1)


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
