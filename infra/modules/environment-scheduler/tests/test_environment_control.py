import importlib.util
import os
import pathlib
import sys
import types
import unittest
from unittest.mock import Mock

sys.modules.setdefault("boto3", types.SimpleNamespace(client=Mock()))
MODULE_PATH = pathlib.Path(__file__).parents[1] / "lambda" / "environment_control.py"
SPEC = importlib.util.spec_from_file_location("environment_control", MODULE_PATH)
control = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(control)


class FakeWaiter:
    def wait(self, **_kwargs):
        return None


class EnvironmentControlTest(unittest.TestCase):
    def setUp(self):
        os.environ.update({
            "ALLOWED_ENVIRONMENT": "develop", "ECS_CLUSTER": "middleware-boletos-develop",
            "ECS_SERVICE": "api", "SCALABLE_RESOURCE_ID": "service/middleware-boletos-develop/api",
            "WEB_ECS_SERVICE": "web", "WEB_SCALABLE_RESOURCE_ID": "service/middleware-boletos-develop/web",
            "RDS_INSTANCE_ID": "middleware-boletos-develop-db",
        })
        self.ecs, self.scaling, self.rds = Mock(), Mock(), Mock()
        self.ecs.get_waiter.return_value = FakeWaiter()
        self.ecs.describe_services.return_value = {"services": [
            {"serviceName": "api", "desiredCount": 1, "runningCount": 1},
            {"serviceName": "web", "desiredCount": 1, "runningCount": 1}
        ]}
        self.scaling.describe_scalable_targets.return_value = {
            "ScalableTargets": [
                {"ResourceId": "service/middleware-boletos-develop/api", "MinCapacity": 1, "MaxCapacity": 2},
                {"ResourceId": "service/middleware-boletos-develop/web", "MinCapacity": 1, "MaxCapacity": 2},
            ]
        }
        self.rds.describe_db_instances.return_value = {
            "DBInstances": [{"DBInstanceStatus": "available"}]
        }
        control._clients = lambda: (self.ecs, self.scaling, self.rds)

    def test_production_rejected_without_aws_calls(self):
        with self.assertRaises(ValueError):
            control.handler({"action": "stop", "environment": "production"}, None)
        self.ecs.describe_services.assert_not_called()
        self.rds.describe_db_instances.assert_not_called()

    def test_invalid_environment_rejected(self):
        os.environ["ALLOWED_ENVIRONMENT"] = "production"
        with self.assertRaises(ValueError):
            control.handler({"action": "stop", "environment": "develop"}, None)

    def test_stop_active(self):
        control.handler({"action": "stop", "environment": "develop"}, None)
        self.assertEqual(self.ecs.update_service.call_count, 2)
        self.ecs.update_service.assert_any_call(cluster="middleware-boletos-develop", service="api", desiredCount=0)
        self.ecs.update_service.assert_any_call(cluster="middleware-boletos-develop", service="web", desiredCount=0)
        self.rds.stop_db_instance.assert_called_once()

    def test_stop_already_stopped(self):
        self.rds.describe_db_instances.return_value = {"DBInstances": [{"DBInstanceStatus": "stopped"}]}
        control.handler({"action": "stop", "environment": "develop"}, None)
        self.rds.stop_db_instance.assert_not_called()

    def test_start_stopped(self):
        states = iter(["stopped", "stopped", "available", "available"])
        self.rds.describe_db_instances.side_effect = lambda **_kwargs: {"DBInstances": [{"DBInstanceStatus": next(states)}]}
        control.time.sleep = lambda _seconds: None
        control.handler({"action": "start", "environment": "develop"}, None)
        self.rds.start_db_instance.assert_called_once()
        self.assertEqual(self.ecs.update_service.call_count, 2)

    def test_start_available(self):
        control.handler({"action": "start", "environment": "develop"}, None)
        self.rds.start_db_instance.assert_not_called()
        self.assertEqual(self.ecs.update_service.call_count, 2)


if __name__ == "__main__":
    unittest.main()
