# Mantl API

[![Build Status](http://drone04.shipped-cisco.com/api/badges/CiscoCloud/mantl-api/status.svg)](http://drone04.shipped-cisco.com/CiscoCloud/mantl-api)

An API interface to [Mantl](https://mantl.io).

Currently, Mantl API allows you to install and uninstall [DCOS packages](https://github.com/mesosphere/universe) on Mantl. More capabilities are planned for the future.

<!-- markdown-toc start - Don't edit this section. Run M-x markdown-toc-generate-toc again -->
**Table of Contents**

- [Mantl API](#mantl-api)
    - [Comparison to Other Software](#comparison-to-other-software)
    - [Building](#building)
    - [Deploying](#deploying)
        - [Options](#options)
    - [Package Repository](#package-repository)
        - [Synchronizing Repository Sources](#synchronizing-repository-sources)
    - [Usage](#usage)
        - [Installing a Package](#installing-a-package)
        - [Uninstalling a Package](#uninstalling-a-package)
    - [Endpoints](#endpoints)
        - [GET /health](#get-health)
        - [GET /1/packages](#get-1packages)
        - [GET /1/packages/<package>](#get-1packagespackage)
        - [POST /1/packages](#post-1packages)
        - [DELETE /1/packages/<package>](#delete-1packagespackage)
    - [License](#license)

<!-- markdown-toc end -->

## Comparison to Other Software

Mantl API leverages the [Mesosphere DCOS package repository](https://github.com/mesosphere/universe) and provides capabilities similar to the [`package`](https://docs.mesosphere.com/using/cli/packagesyntax/) command in the [DCOS CLI](https://github.com/mesosphere/dcos-cli). The goal is to provide a simple, API-driven way to install and uninstall pre-built packages on Mantl clusters. In the future, mantl-api will contain additional functionality for maintaining and operating Mantl clusters.

## Building

```shell
docker build -t mantl-api .
```

## Deploying

You can run Mantl API on your cluster via Marathon. An example `mantl-api.json` is included below:

```json
{
  "id": "/mantl-api",
  "container": {
    "type": "DOCKER",
    "docker": {
      "image": "ciscocloud/mantl-api:0.1.0",
      "network": "BRIDGE",
      "portMappings": [
        { "containerPort": 4001, "hostPort": 0 }
      ]
    }
  },
  "instances": 1,
  "cpus": 1.0,
  "mem": 512,
  "constraints": [["hostname", "UNIQUE"]],
  "env": {
    "CONSUL_HTTP_SSL_VERIFY": "false",
    "MANTL_API_LOG_LEVEL": "debug",
    "MANTL_API_CONSUL": "https://consul.service.consul:8500",
    "MANTL_API_MESOS_PRINCIPAL": "mantl-api",
    "MANTL_API_MESOS_SECRET": "secret"
  },
  "healthChecks": [
    {
      "protocol": "HTTP",
      "path": "/health",
      "gracePeriodSeconds": 3,
      "intervalSeconds": 10,
      "portIndex": 0,
      "timeoutSeconds": 10,
      "maxConsecutiveFailures": 3
    }
  ]
}
```

You will need to replace the `MANTL_API_MESOS_PRINCIPAL` and `MANTL_API_MESOS_SECRET` variables with valid Mesos credentials. These can be found in the `security.yml` file that was generated when you ran [security-setup](http://microservices-infrastructure.readthedocs.org/en/latest/security/security_setup.html) for your Mantl cluster.

All Mantl API configuration can be set with environment variables. See below for the additional configuration options that are available.

To install Mantl API, you can submit the above json to Marathon.

```shell
curl -k -u admin:mantlpw -X POST -d @mantl-api.json -H "Content-type: application/json" https://mantl-control-01/marathon/v2/apps
```

You will need to replace `admin:mantlpw` with valid Marathon credentials (see `marathon_http_credentials` in your `security.yml`). You will also replace `mantl-control-01` with the host of one your Mantl control nodes.

After a few moments, Mantl API should be running on your cluster.

### Options

 Argument                 | Default                                                                   | Description
--------------------------|---------------------------------------------------------------------------|---------------------------------------------------------------
 `log-level`              | info                                                                      | one of debug, info, warn, error, or fatal
 `log-format`             | text                                                                      | specify output (text or json)
 `consul`                 | http://localhost:8500                                                     | Consul API address
 `marathon`               | Discovered via a `marathon` service registered in Consul                  | Marathon API address
 `marathon-user`          | None                                                                      | Marathon API user
 `marathon-password`      | None                                                                      | Marathon API password
 `marathon-no-verify-ssl` | False                                                                     | When True, disables SSL verification for the Marathon API
 `mesos`                  | Discovered via a `mesos` service with a `leader` tag registered in Consul | Mesos API address
 `mesos-principal`        | None                                                                      | Mesos principal for framework authentication
 `mesos-secret`           | None                                                                      | Mesos secret for framework authentication
 `mesos-no-verify-ssl`    | False                                                                     | When True, disables SSL verification for the Mesos API
 `listen`                 | :4001                                                                     | Listen for connections at this address
 `zookeeper`              | Discovered via a `marathon` service registered in Consul                  | Comma-delimited list of zookeeper servers
 `force-sync`             | False                                                                     | Forces a synchronization of all repository sources at startup

Every option can be set via environment variables prefixed with `MANTL_API`. For example, you can use `MANTL_API_LOG_LEVEL` for `log-level`, `MANTL_API_CONSUL` for `consul`, and so on.

## Package Repository

[mantl-universe](https://github.com/ciscocloud/mantl-universe) contains the list of packages that work out-of-the-box on Mantl today. You can install any of the [DCOS packages](https://github.com/mesosphere/universe) but you will likely have to customize some of the configuration to work on Mantl. Most of the packages assume that service discovery is provided by [Mesos-DNS](https://github.com/mesosphere/mesos-dns) and need to be converted to work with the [Consul DNS](https://www.consul.io/docs/agent/dns.html) interface. Contributions are welcome!


### Synchronizing Repository Sources

The package repositories are synchronized to the Consul K/V backend. If you want to refresh your repositories, you can run the following command:

```shell
mantl-api sync --consul http://consul.service.consul:8500
```

## Usage

### Installing a Package

It is a single API call to install a package. In the example below, we are going to run [Cassandra](http://cassandra.apache.org) on our Mantl cluster.

```shell
curl -X POST -d "{\"name\": \"cassandra\"}" http://mantl-worker-003:4001/1/packages
```

You will need to replace `mantl-worker-003` and `4001` with the host and port where Mantl API is running. You can use the Marathon API or UI to find this. You'll also need to make sure that the port is accessible to the machine where you are calling the API from. Adjust the security groups or firewall rules for your platform accordingly.

After about 5 minutes, you should have Cassandra up and running on your Mantl cluster.

### Uninstalling a Package

Uninstalling is just as easy. Run the command below to uninstall Cassandra:

```shell
curl -X DELETE http://mantl-worker-003.jossware.org:4001/1/packages/cassandra
```

After a moment, Cassandra will have been removed from your cluster. This will also remove the [Zookeeper](https://zookeeper.apache.org) state for the Cassandra framework. In the future, we will add more flexibility in being able to control what is uninstalled.

## Endpoints

 Endpoint            | Method | Description
---------------------|--------|-----------------------------------------------------
 `/health`           | GET    | health check - returns `OK` with an HTTP 200 status
 `/1/packages`       | GET    | list available packages
 `/1/packages`       | POST   | install a package
 `/1/packages/:name` | GET    | provides information about a specific package
 `/1/packages/:name` | DELETE | uninstalls a specific package

### GET /health

`GET /health`: returns `OK`

```shell
curl http://mantl-worker-003:4001/health
```

```
OK
```

### GET /1/packages

`GET /1/packages`: returns a JSON representation of packages available to install.

```shell
curl http://mantl-control-01:4001/1/packages | jq .
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
curl http://mantl-worker-001:4001/1/packages/cassandra | jq .
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

### POST /1/packages

`POST /1/packages`: post a JSON representation of a package to install.

```shell
curl -X POST -d "{\"name\": \"cassandra\"}" http://mantl-worker-001:4001/1/packages | jq .
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

### DELETE /1/packages/<package>

`DELETE /1/packages/<package>`: post a JSON representation of package specific uninstall options.

```shell
curl -X DELETE -d "{\"name\": \"cassandra\"}" http://mantl-worker-001:4001/1/packages/cassandra
```

## License

mantl-api is released under the Apache 2.0 license (see [LICENSE](LICENSE))
