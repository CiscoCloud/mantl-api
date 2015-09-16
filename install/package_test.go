package install

import (
	"github.com/stretchr/testify/assert"
	"sort"
	"strings"
	"testing"
)

var configJson = `
{
  "type": "object",
  "properties": {
    "mesos": {
      "description": "Mesos specific configuration properties",
      "type": "object",
      "properties": {
        "master": {
          "default": "zk://master.mesos:2181/mesos",
          "description": "The URL of the Mesos master. The format is a comma-delimited list of hosts like zk://host1:port,host2:port/mesos. If using ZooKeeper, pay particular attention to the leading zk:// and trailing /mesos! If not using ZooKeeper, standard host:port patterns, like localhost:5050 or 10.0.0.5:5050,10.0.0.6:5050, are also acceptable.",
          "type": "string"
        }
      },
      "required": [
        "master"
      ]
    },
    "cassandra": {
      "description": "Cassandra Framework Configuration Properties",
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "framework": {
          "description": "Framework Scheduler specific Configuration Properties",
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "failover-timeout-seconds": {
              "description": "The failover_timeout for Mesos in seconds. If the framework instance has not re-registered with Mesos this long after a failover, Mesos will shut down all running tasks started by the framework.",
              "type": "integer",
              "minimum": 0,
              "default": 604800
            },
            "role": {
              "description": "Mesos role for this framework.",
              "type": "string",
              "default": "*"
            },
            "authentication": {
              "description": "Framework Scheduler Authentication Configuration Properties",
              "type": "object",
              "additionalProperties": false,
              "properties": {
                "enabled": {
                  "description": "Whether framework authentication should be used",
                  "type": "boolean",
                  "default": false
                },
                "principal": {
                  "description": "The Mesos principal used for authentication.",
                  "type": "string"
                },
                "secret": {
                  "description": "The path to the Mesos secret file containing the authentication secret.",
                  "type": "string"
                }
              },
              "required": [
                "enabled"
              ]
            }
          },
          "required": [
            "failover-timeout-seconds",
            "role",
            "authentication"
          ]
        },
        "cluster-name": {
          "description": "The name of the framework to register with mesos. Will also be used as the cluster name in Cassandra",
          "type": "string",
          "default": "dcos"
        },
        "zk": {
          "description": "ZooKeeper URL for storing state. Format: zk://host1:port1,host2:port2,.../path (can have nested directories)",
          "type": "string"
        },
        "zk-timeout-ms": {
          "description": "Timeout for ZooKeeper in milliseconds.",
          "type": "integer",
          "minimum": 0,
          "default": 10000
        },
        "node-count": {
          "description": "The number of nodes in the ring for the framework to run.",
          "type": "integer",
          "minimum": 1,
          "default": 3
        },
        "seed-count": {
          "description": "The number of seed nodes in the ring for the framework to run.",
          "type": "integer",
          "minimum": 1,
          "default": 2
        },
        "health-check-interval-seconds": {
          "description": "The interval in seconds that the framework should check the health of each Cassandra Server instance.",
          "type": "integer",
          "minimum": 15,
          "default": 60
        },
        "bootstrap-grace-time-seconds": {
          "description": "The minimum number of seconds to wait between starting each node. Setting this too low could result in the ring not bootstrapping correctly.",
          "type": "integer",
          "minimum": 15,
          "default": 120
        },
        "data-directory": {
          "description": "The location on disk where Cassandra will be configured to write it's data.",
          "type": "string",
          "default": "."
        },
        "resources": {
          "description": "Cassandra Server Resources Configuration Properties",
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "cpus": {
              "description": "CPU shares to allocate to each Cassandra Server Instance.",
              "type": "number",
              "minimum": 0.0,
              "default": 0.1
            },
            "mem": {
              "description": "Memory (MB) to allocate to each Cassandra Server instance.",
              "type": "integer",
              "minimum": 0,
              "default": 768
            },
            "disk": {
              "description": "Disk (MB) to allocate to each Cassandra Server instance.",
              "type": "integer",
              "minimum": 0,
              "default": 16
            },
            "heap-mb": {
              "description": "The amount of memory in MB that are allocated to each Cassandra Server Instance. This value should be smaller than 'cassandra.resources.mem'. The remaining difference will be used for memory mapped files and other off-heap memory requirements.",
              "type": "integer",
              "minimum": 0
            }
          },
          "required": [
            "cpus",
            "mem",
            "disk"
          ]
        },
        "dc": {
          "description": "Cassandra multi Datacenter Configuration Properties",
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "default-dc": {
              "description": "Default value to be set for dc name in the GossipingPropertyFileSnitch",
              "type": "string",
              "default": "DC1"
            },
            "default-rack": {
              "description": "Default value to be set for rack name in the GossipingPropertyFileSnitch",
              "type": "string",
              "default": "RAC1"
            },
            "external-dcs": {
              "description": "Name and URL for another instance of Cassandra DCOS Service",
              "type": "array",
              "additionalProperties": false,
              "items": {
                "type": "object",
                "additionalProperties": false,
                "properties": {
                  "name": {
                    "type": "string"
                  },
                  "url": {
                    "type": "string"
                  }
                },
                "required": [
                  "name",
                  "url"
                ]
              }
            }
          },
          "required": [
            "default-dc",
            "default-rack"
          ]
        }
      },
      "required": [
        "framework",
        "cluster-name",
        "zk-timeout-ms",
        "node-count",
        "seed-count",
        "health-check-interval-seconds",
        "bootstrap-grace-time-seconds",
        "data-directory",
        "resources"
      ]
    }
  },
  "required": [
    "mesos",
    "cassandra"
  ]

}
`

var optionsJson = `
{
  "mesos": {
    "master": "zk://zookeeper.service.consul:2181/mesos",
    "added-config": 0
  },
  "cassandra": {
    "cluster-name": "dcos-test",
    "zk": "zk://zookeeper.service.consul:2181/cassandra-mesos/dcos-test",
    "framework": {
      "authentication": { "enabled": true }
    }
	}
}
`

var marathonJson = `
{
  "id": "/cassandra/{{cassandra.cluster-name}}",
  "cmd": "$(pwd)/jre*/bin/java $JAVA_OPTS -classpath cassandra-mesos-framework.jar io.mesosphere.mesos.frameworks.cassandra.framework.Main",
  "instances": 1,
  "cpus": 0.5,
  "mem": 512,
  "ports": [
    0
  ],
  "uris": [
    "https://downloads.mesosphere.io/cassandra-mesos/artifacts/0.2.0-1/cassandra-mesos-0.2.0-1.tar.gz",
    "https://downloads.mesosphere.io/java/jre-7u76-linux-x64.tar.gz"
  ],
  "healthChecks": [
    {
      "gracePeriodSeconds": 120,
      "intervalSeconds": 15,
      "maxConsecutiveFailures": 0,
      "path": "/health/cluster",
      "portIndex": 0,
      "protocol": "HTTP",
      "timeoutSeconds": 5
    },
    {
      "gracePeriodSeconds": 120,
      "intervalSeconds": 30,
      "maxConsecutiveFailures": 3,
      "path": "/health/process",
      "portIndex": 0,
      "protocol": "HTTP",
      "timeoutSeconds": 5
    }
  ],
  "labels": {
    "DCOS_PACKAGE_FRAMEWORK_NAME": "cassandra.{{cassandra.cluster-name}}"
  },
  "env": {
    "MESOS_ZK": "{{mesos.master}}"
    ,"JAVA_OPTS": "-Xms256m -Xmx256m"
    ,"CASSANDRA_CLUSTER_NAME": "{{cassandra.cluster-name}}"
    ,"CASSANDRA_NODE_COUNT": "{{cassandra.node-count}}"
    ,"CASSANDRA_SEED_COUNT": "{{cassandra.seed-count}}"
    ,"CASSANDRA_RESOURCE_CPU_CORES": "{{cassandra.resources.cpus}}"
    ,"CASSANDRA_RESOURCE_MEM_MB": "{{cassandra.resources.mem}}"
    ,"CASSANDRA_RESOURCE_DISK_MB": "{{cassandra.resources.disk}}"
    ,"CASSANDRA_FAILOVER_TIMEOUT_SECONDS": "{{cassandra.framework.failover-timeout-seconds}}"
    ,"CASSANDRA_DATA_DIRECTORY": "{{cassandra.data-directory}}"
    ,"CASSANDRA_HEALTH_CHECK_INTERVAL_SECONDS": "{{cassandra.health-check-interval-seconds}}"
    ,"CASSANDRA_ZK_TIMEOUT_MS": "{{cassandra.zk-timeout-ms}}"
    ,"CASSANDRA_BOOTSTRAP_GRACE_TIME_SECONDS": "{{cassandra.bootstrap-grace-time-seconds}}"
    ,"CASSANDRA_FRAMEWORK_MESOS_ROLE": "{{cassandra.framework.role}}"
    ,"CASSANDRA_DEFAULT_DC": "{{cassandra.dc.default-dc}}"
    ,"CASSANDRA_DEFAULT_RACK": "{{cassandra.dc.default-rack}}"
    ,"MESOS_AUTHENTICATE": "{{cassandra.framework.authentication.enabled}}"
{{#cassandra.dc.external-dcs}}
    ,"CASSANDRA_EXTERNAL_DC_{{name}}": "{{url}}"
{{/cassandra.dc.external-dcs}}
{{#cassandra.zk}} {{! if the full cassandra zk url has been specified use it }}
    ,"CASSANDRA_ZK": "{{cassandra.zk}}"
{{/cassandra.zk}}
{{^cassandra.zk}} {{! else, create a url based on convention and cluster name }}
    ,"CASSANDRA_ZK": "zk://master.mesos:2181/cassandra-mesos/{{cassandra.cluster-name}}"
{{/cassandra.zk}}
{{#cassandra.resource.heap-mb}}
,"CASSANDRA_RESOURCE_HEAP_MB": "{{cassandra.resource.heap-mb}}"
{{/cassandra.resource.heap-mb}}
{{#framework.authentication.principal}}
    ,"DEFAULT_PRINCIPAL": "{{cassandra.framework.authentication.principal}}"
{{/framework.authentication.principal}}
{{#framework.authentication.secret}}
    ,"DEFAULT_SECRET": "{{cassandra.framework.authentication.secret}}"
{{/framework.authentication.secret}}
  }
}
`

func TestPackageVersionByMostRecent(t *testing.T) {
	t.Parallel()

	versions := []*PackageVersion{
		{Version: "1.0", Index: "1"},
		{Version: "0.9", Index: "0"},
		{Version: "1.2", Index: "3"},
		{Version: "1.1", Index: "2"},
	}

	sort.Sort(packageVersionByMostRecent(versions))

	idxs := make([]string, len(versions))
	for i, v := range versions {
		idxs[i] = v.Index
	}

	assert.Equal(t, []string{"3", "2", "1", "0"}, idxs)
}

func TestValidPackageDefinition(t *testing.T) {
	t.Parallel()
	pkgDef := &packageDefinition{
		configJson:   []byte("test"),
		marathonJson: []byte("test"),
		packageJson:  []byte("test"),
	}
	assert.True(t, pkgDef.IsValid())
}

func TestInvalidPackageDefinition(t *testing.T) {
	t.Parallel()
	pkgDef := &packageDefinition{}
	assert.False(t, pkgDef.IsValid())
}

func TestConfigSchemaType(t *testing.T) {
	t.Parallel()

	pkgDef := &packageDefinition{configJson: []byte(configJson)}
	schema, err := pkgDef.ConfigSchema()
	if assert.NotNil(t, schema, "schema should not be nil: %v", err) {
		assert.Equal(t, "object", schema.Type, "schema should have an object type: %v", err)
	}
}

func TestExtractDefaultValues(t *testing.T) {
	t.Parallel()

	pkgDef := &packageDefinition{configJson: []byte(configJson)}
	schema, _ := pkgDef.ConfigSchema()
	defaults := schema.defaultConfig()

	assert.Equal(t, "zk://master.mesos:2181/mesos", getConfigVal(defaults, "mesos", "master"))
	assert.Equal(t, ".", getConfigVal(defaults, "cassandra", "data-directory"))
	assert.Equal(t, "dcos", getConfigVal(defaults, "cassandra", "cluster-name"))
	assert.Equal(t, false, getConfigVal(defaults, "cassandra", "framework", "authentication", "enabled"))
}

func TestMergeOptions(t *testing.T) {
	t.Parallel()

	pkgDef := &packageDefinition{
		configJson:  []byte(configJson),
		optionsJson: []byte(optionsJson),
	}

	merged, _ := pkgDef.MergedConfig()

	assert.Equal(t, "zk://zookeeper.service.consul:2181/mesos", getConfigVal(merged, "mesos", "master"))
	assert.Equal(t, 0.0, getConfigVal(merged, "mesos", "added-config"))
	assert.Equal(t, "dcos-test", getConfigVal(merged, "cassandra", "cluster-name"))
	assert.Equal(t, "zk://zookeeper.service.consul:2181/cassandra-mesos/dcos-test", getConfigVal(merged, "cassandra", "zk"))
	assert.Equal(t, true, getConfigVal(merged, "cassandra", "framework", "authentication", "enabled"))
}

func TestMarathon(t *testing.T) {
	t.Parallel()

	pkgDef := &packageDefinition{
		configJson:   []byte(configJson),
		optionsJson:  []byte(optionsJson),
		marathonJson: []byte(marathonJson),
	}

	marathon, _ := pkgDef.MarathonAppJson()
	assert.True(t, strings.Contains(marathon, "\"id\": \"/cassandra/dcos-test\","))
	assert.True(t, strings.Contains(marathon, "\"MESOS_ZK\": \"zk://zookeeper.service.consul:2181/mesos\""))
}

func getConfigVal(m map[string]interface{}, keys ...string) interface{} {
	nested := m
	for _, key := range keys {
		if nm, ok := nested[key].(map[string]interface{}); ok {
			nested = nm
		} else if val, ok := nested[key]; ok {
			return val
		}
	}
	return nil
}
