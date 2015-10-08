package marathon

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

var marathonAppJson = `
{
  "id": "/example",
  "container": {
    "type": "DOCKER",
    "docker": {
      "image": "repo/image-name:0.1.0",
      "forcePullImage": true
    }
  },
  "instances": 1,
  "cpus": 1.0,
  "mem": 512,
  "ports": [
    0
  ],
  "constraints": [["hostname", "UNIQUE"]],
  "env": {
    "SSL_VERIFY": "false",
    "LOG_LEVEL": "debug"
  }
}
`

var marathonAppsJson = `
{
  "apps": [
    {
      "acceptedResourceRoles": null,
      "args": [
        "--verbose"
      ],
      "backoffFactor": 1.15,
      "backoffSeconds": 1,
      "cmd": null,
      "constraints": [
        [
          "hostname",
          "UNIQUE"
        ]
      ],
      "container": {
        "docker": {
          "forcePullImage": false,
          "image": "ciscocloud/marathon-consul:0.1",
          "network": "BRIDGE",
          "parameters": [],
          "portMappings": [
            {
              "containerPort": 4000,
              "hostPort": 4000,
              "protocol": "tcp",
              "servicePort": 10001
            }
          ],
          "privileged": false
        },
        "type": "DOCKER",
        "volumes": [
          {
            "containerPath": "/usr/local/share/ca-certificates/",
            "hostPath": "/etc/pki/ca-trust/source/anchors/",
            "mode": "RO"
          }
        ]
      },
      "cpus": 0.1,
      "dependencies": [],
      "deployments": [],
      "disk": 0.0,
      "env": {},
      "executor": "",
      "healthChecks": [
        {
          "gracePeriodSeconds": 300,
          "ignoreHttp1xx": false,
          "intervalSeconds": 60,
          "maxConsecutiveFailures": 3,
          "path": "/health",
          "portIndex": 0,
          "protocol": "HTTP",
          "timeoutSeconds": 20
        }
      ],
      "id": "/marathon-consul",
      "instances": 1,
      "labels": {},
      "maxLaunchDelaySeconds": 3600,
      "mem": 128.0,
      "ports": [
        10001
      ],
      "requirePorts": false,
      "storeUrls": [],
      "tasksHealthy": 1,
      "tasksRunning": 1,
      "tasksStaged": 0,
      "tasksUnhealthy": 0,
      "upgradeStrategy": {
        "maximumOverCapacity": 1.0,
        "minimumHealthCapacity": 1.0
      },
      "uris": [],
      "user": null,
      "version": "2015-10-07T12:56:03.577Z"
    },
    {
      "acceptedResourceRoles": null,
      "args": [
        "--zk=zk://zookeeper.service.consul:2181/mesos",
        "--refresh=7s"
      ],
      "backoffFactor": 1.15,
      "backoffSeconds": 1,
      "cmd": null,
      "constraints": [],
      "container": {
        "docker": {
          "forcePullImage": false,
          "image": "ciscocloud/mesos-consul:v0.2.1",
          "network": "BRIDGE",
          "parameters": [],
          "privileged": false
        },
        "type": "DOCKER",
        "volumes": []
      },
      "cpus": 0.1,
      "dependencies": [],
      "deployments": [],
      "disk": 0.0,
      "env": {},
      "executor": "",
      "healthChecks": [],
      "id": "/mesos-consul",
      "instances": 1,
      "labels": {},
      "maxLaunchDelaySeconds": 3600,
      "mem": 128.0,
      "ports": [
        10000
      ],
      "requirePorts": false,
      "storeUrls": [],
      "tasksHealthy": 0,
      "tasksRunning": 1,
      "tasksStaged": 0,
      "tasksUnhealthy": 0,
      "upgradeStrategy": {
        "maximumOverCapacity": 1.0,
        "minimumHealthCapacity": 1.0
      },
      "uris": [],
      "user": null,
      "version": "2015-10-07T12:56:01.940Z"
    }
  ]
}
`

func TestToApp(t *testing.T) {
	t.Parallel()
	ts, marathon := fakeMarathon(successHandler)
	defer ts.Close()
	app, err := marathon.ToApp(marathonAppJson)
	assert.Nil(t, err)
	assert.Equal(t, "/example", app.ID)
}

func TestApps(t *testing.T) {
	t.Parallel()
	ts, marathon := fakeMarathon(appsResponseHandler)
	defer ts.Close()

	apps, err := marathon.Apps()

	assert.Nil(t, err)
	assert.Equal(t, 2, len(apps))
}

func TestAppsNoApps(t *testing.T) {
	t.Parallel()
	ts, marathon := fakeMarathon(emptyJsonHandler)
	defer ts.Close()

	apps, err := marathon.Apps()

	assert.Nil(t, err)
	assert.Equal(t, 0, len(apps))
}

func TestCreateApp(t *testing.T) {
	t.Parallel()
	ts, marathon := fakeMarathon(appResponseHandler)
	defer ts.Close()

	app, _ := marathon.ToApp(marathonAppJson)
	response, err := marathon.CreateApp(app)

	assert.Nil(t, err)
	assert.True(t, len(response) > 0)
}

func TestCreateAppConflict(t *testing.T) {
	t.Parallel()
	ts, marathon := fakeMarathon(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(409)
	})
	defer ts.Close()

	app, _ := marathon.ToApp(marathonAppJson)
	_, err := marathon.CreateApp(app)

	assert.NotNil(t, err)
	assert.Equal(t, "409 Conflict - application already exists", err.Error())
}

func TestDestroyApp(t *testing.T) {
	t.Parallel()
	ts, marathon := fakeMarathon(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	})
	defer ts.Close()

	_, err := marathon.DestroyApp("/123")

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed deleting /123 from marathon")
}

func TestDestroyAppError(t *testing.T) {
	t.Parallel()
	ts, marathon := fakeMarathon(emptyJsonHandler)
	defer ts.Close()

	response, err := marathon.DestroyApp("/123")

	assert.Nil(t, err)
	assert.True(t, len(response) > 0)
}

func fakeMarathon(handler func(w http.ResponseWriter, r *http.Request)) (*httptest.Server, *Marathon) {
	ts := httptest.NewServer(http.HandlerFunc(handler))
	marathon, _ := NewMarathon(ts.URL, "", "", false)
	return ts, marathon
}

func successHandler(w http.ResponseWriter, r *http.Request) {}
func emptyJsonHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "{}")
}
func appResponseHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, marathonAppJson)
}
func appsResponseHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, marathonAppsJson)
}
