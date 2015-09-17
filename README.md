# mantl-api (experimental)

## GET /1/packages

`GET /1/packages`: returns a JSON representation of packages available to install.

```shell
curl http://gce-control-01:4001/1/packages | jq .
```

```json
[
  {
    "name": "cassandra",
    "description": "Apache Cassandra running on Apache Mesos",
    "framework": true,
    "currentVersion": "0.2.0-1",
    "supported": true,
    "tags": [
      "mesosphere",
      "framework",
      "data",
      "database"
    ],
    "versions": {
      "0.1.0-1": {
        "version": "0.1.0-1",
        "index": "0",
        "supported": true
      },
      "0.2.0-1": {
        "version": "0.2.0-1",
        "index": "1",
        "supported": true
      }
    }
  },
  ...
  {
    "name": "swarm",
    "description": "Swarm is a Docker-native clustering system.",
    "framework": true,
    "currentVersion": "0.4.0",
    "supported": false,
    "tags": [
      "mesosphere",
      "framework",
      "docker"
    ],
    "versions": {
      "0.4.0": {
        "version": "0.4.0",
        "index": "0",
        "supported": false
      }
    }
  }
]
```

### GET /1/packages/<package>

`GET /1/packages/<package>`: returns a JSON representation of a package.

```shell
curl http://gce-control-01:4001/1/packages/cassandra | jq .
```

```json
{
  "name": "cassandra",
  "description": "Apache Cassandra running on Apache Mesos",
  "framework": true,
  "currentVersion": "0.2.0-1",
  "supported": true,
  "tags": [
    "mesosphere",
    "framework",
    "data",
    "database"
  ],
  "versions": {
    "0.1.0-1": {
      "version": "0.1.0-1",
      "index": "0",
      "supported": true
    },
    "0.2.0-1": {
      "version": "0.2.0-1",
      "index": "1",
      "supported": true
    }
  }
}
```

## POST /1/packages

`POST /1/packages`: post a JSON representation of a package to install.

```shell
curl -X POST -d "{\"name\": \"cassandra\"}" http://gce-control-01:4001/1/packages | jq .
```

```json
{
  "id": "/cassandra/dcos-test",
  "cmd": "$(pwd)/jre*/bin/java $JAVA_OPTS -classpath cassandra-mesos-framework.jar io.mesosphere.mesos.frameworks.cassandra.framework.Main",
  "args": null,
  "user": null,
  "env": {
    "CASSANDRA_ZK_TIMEOUT_MS": "10000",
    "JAVA_OPTS": "-Xms256m -Xmx256m",
    "MESOS_AUTHENTICATE": "true",
    "CASSANDRA_RESOURCE_CPU_CORES": "0.1",
    "CASSANDRA_FAILOVER_TIMEOUT_SECONDS": "604800",
    "DEFAULT_SECRET": "secret",
    "CASSANDRA_BOOTSTRAP_GRACE_TIME_SECONDS": "120",
    "CASSANDRA_DEFAULT_DC": "DC1",
    "CASSANDRA_RESOURCE_MEM_MB": "768",
    "MESOS_ZK": "zk://zookeeper.service.consul:2181/mesos",
    "CASSANDRA_DATA_DIRECTORY": ".",
    "DEFAULT_PRINCIPAL": "mantl-install",
    "CASSANDRA_FRAMEWORK_MESOS_ROLE": "*",
    "CASSANDRA_CLUSTER_NAME": "dcos-test",
    "CASSANDRA_RESOURCE_DISK_MB": "16",
    "CASSANDRA_SEED_COUNT": "2",
    "CASSANDRA_ZK": "zk://zookeeper.service.consul:2181/cassandra-mesos/dcos-test",
    "CASSANDRA_NODE_COUNT": "3",
    "CASSANDRA_DEFAULT_RACK": "RAC1",
    "CASSANDRA_HEALTH_CHECK_INTERVAL_SECONDS": "60"
  },
  "instances": 1,
  "cpus": 0.5,
  "mem": 512,
  "disk": 0,
  "executor": "",
  "constraints": [],
  "uris": [
    "https://downloads.mesosphere.io/cassandra-mesos/artifacts/0.2.0-1/cassandra-mesos-0.2.0-1.tar.gz",
    "https://downloads.mesosphere.io/java/jre-7u76-linux-x64.tar.gz"
  ],
  "storeUrls": [],
  "ports": [
    0
  ],
  "requirePorts": false,
  "backoffFactor": 0,
  "container": null,
  "healthChecks": [
    {
      "path": "/health/cluster",
      "protocol": "HTTP",
      "portIndex": 0,
      "command": null,
      "gracePeriodSeconds": 120,
      "intervalSeconds": 15,
      "timeoutSeconds": 5,
      "maxConsecutiveFailures": 0,
      "ignoreHttp1xx": false
    },
    {
      "path": "/health/process",
      "protocol": "HTTP",
      "portIndex": 0,
      "command": null,
      "gracePeriodSeconds": 120,
      "intervalSeconds": 30,
      "timeoutSeconds": 5,
      "maxConsecutiveFailures": 3,
      "ignoreHttp1xx": false
    }
  ],
  "dependencies": [],
  "upgradeStrategy": {
    "minimumHealthCapacity": 0,
    "maximumOverCapacity": 0
  },
  "labels": {
    "MANTL_PACKAGE_NAME": "cassandra",
    "MANTL_PACKAGE_IS_FRAMEWORK": "true",
    "DCOS_PACKAGE_FRAMEWORK_NAME": "cassandra.dcos-test",
    "MANTL_PACKAGE_FRAMEWORK_NAME": "cassandra.dcos-test",
    "MANTL_PACKAGE_INDEX": "1",
    "MANTL_PACKAGE_VERSION": "0.2.0-1"
  },
  "acceptedResourceRoles": null,
  "version": "2015-09-17T22:19:24.576Z",
  "tasks": [],
  "deployments": [
    {
      "id": "dd113528-388c-4438-a524-9eba45aa2908"
    }
  ],
  "tasksStaged": 0,
  "tasksRunning": 0,
  "tasksHealthy": 0,
  "tasksUnhealthy": 0,
  "backoffSeconds": 0,
  "maxLaunchDelaySeconds": 3600
}
```

## DELETE /1/packages/<package>

`DELETE /1/packages/<package>`: post a JSON representation of package specific uninstall options.

```shell
curl -X DELETE -d "{\"name\": \"cassandra\"}" http://gce-control-01:4001/1/packages/cassandra
```
