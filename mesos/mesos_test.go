package mesos

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
)

var stateJson = `
{
    "activated_slaves": 1,
    "build_date": "2015-07-24 10:07:54",
    "build_time": 1437732474,
    "build_user": "root",
    "cluster": "vagrant",
    "completed_frameworks": [],
    "deactivated_slaves": 0,
    "elected_time": 1444139988.01358,
    "flags": {
        "allocation_interval": "1secs",
        "allocator": "HierarchicalDRF",
        "authenticate": "true",
        "authenticate_slaves": "true",
        "authenticators": "crammd5",
        "cluster": "vagrant",
        "credentials": "/etc/mesos/credentials",
        "framework_sorter": "drf",
        "help": "false",
        "hostname": "default.node.consul",
        "initialize_driver_logging": "true",
        "ip": "192.168.242.55",
        "log_auto_initialize": "true",
        "log_dir": "/var/log/mesos",
        "logbufsecs": "0",
        "logging_level": "INFO",
        "max_slave_ping_timeouts": "5",
        "port": "15050",
        "quiet": "false",
        "quorum": "1",
        "recovery_slave_removal_limit": "100%",
        "registry": "replicated_log",
        "registry_fetch_timeout": "1mins",
        "registry_store_timeout": "5secs",
        "registry_strict": "false",
        "root_submissions": "true",
        "slave_ping_timeout": "15secs",
        "slave_reregister_timeout": "10mins",
        "user_sorter": "drf",
        "version": "false",
        "webui_dir": "/usr/share/mesos/webui",
        "work_dir": "/var/lib/mesos",
        "zk": "zk://mesos:7gSUd9gU8AvqNO6h@zookeeper.service.consul:2181/mesos",
        "zk_session_timeout": "10secs"
    },
    "frameworks": [
        {
            "active": true,
            "checkpoint": true,
            "completed_tasks": [],
            "executors": [],
            "failover_timeout": 604800,
            "hostname": "default",
            "id": "20151006-135938-938649792-15050-8041-0000",
            "name": "chronos",
            "offered_resources": {
                "cpus": 0,
                "disk": 0,
                "mem": 0
            },
            "offers": [],
            "pid": "scheduler-291cd194-01fb-47f8-9951-b16538b8181a@192.168.242.55:56393",
            "registered_time": 1444140078.11997,
            "resources": {
                "cpus": 0,
                "disk": 0,
                "mem": 0
            },
            "role": "*",
            "tasks": [],
            "unregistered_time": 0,
            "used_resources": {
                "cpus": 0,
                "disk": 0,
                "mem": 0
            },
            "user": "root",
            "webui_url": "http://default.node.consul:14400"
        },
        {
            "active": true,
            "checkpoint": true,
            "completed_tasks": [],
            "executors": [],
            "failover_timeout": 604800,
            "hostname": "default",
            "id": "20151006-135423-16777343-5050-782-0000",
            "name": "marathon",
            "offered_resources": {
                "cpus": 0,
                "disk": 0,
                "mem": 0
            },
            "offers": [],
            "pid": "scheduler-c7615f0c-3b44-4fd1-89c5-688d9a78de75@192.168.242.55:53193",
            "registered_time": 1444139990.67918,
            "resources": {
                "cpus": 0.2,
                "disk": 0,
                "mem": 256,
                "ports": "[4680-4680, 4000-4000]"
            },
            "role": "*",
            "tasks": [
                {
                    "executor_id": "",
                    "framework_id": "20151006-135423-16777343-5050-782-0000",
                    "id": "marathon-consul.9224fbb6-6c32-11e5-a890-ce10b690c57f",
                    "labels": [],
                    "name": "marathon-consul",
                    "resources": {
                        "cpus": 0.1,
                        "disk": 0,
                        "mem": 128,
                        "ports": "[4000-4000]"
                    },
                    "slave_id": "20151006-135938-938649792-15050-8041-S0",
                    "state": "TASK_RUNNING",
                    "statuses": [
                        {
                            "state": "TASK_RUNNING",
                            "timestamp": 1444140016.90111
                        }
                    ]
                },
                {
                    "executor_id": "",
                    "framework_id": "20151006-135423-16777343-5050-782-0000",
                    "id": "mesos-consul.8e9da9b5-6c32-11e5-a890-ce10b690c57f",
                    "labels": [],
                    "name": "mesos-consul",
                    "resources": {
                        "cpus": 0.1,
                        "disk": 0,
                        "mem": 128,
                        "ports": "[4680-4680]"
                    },
                    "slave_id": "20151006-135938-938649792-15050-8041-S0",
                    "state": "TASK_RUNNING",
                    "statuses": [
                        {
                            "state": "TASK_RUNNING",
                            "timestamp": 1444140011.5227
                        }
                    ]
                }
            ],
            "unregistered_time": 0,
            "used_resources": {
                "cpus": 0.2,
                "disk": 0,
                "mem": 256,
                "ports": "[4680-4680, 4000-4000]"
            },
            "user": "root",
            "webui_url": "http://default.node.consul:18080"
        }
    ],
    "git_sha": "4ce5475346a0abb7ef4b7ffc9836c5836d7c7a66",
    "git_tag": "0.23.0",
    "hostname": "default.node.consul",
    "id": "20151006-135938-938649792-15050-8041",
    "leader": "master@192.168.242.55:15050",
    "log_dir": "/var/log/mesos",
    "orphan_tasks": [],
    "pid": "master@192.168.242.55:15050",
    "slaves": [
        {
            "active": true,
            "attributes": {
                "node_id": "default"
            },
            "hostname": "default",
            "id": "20151006-135938-938649792-15050-8041-S0",
            "offered_resources": {
                "cpus": 0,
                "disk": 0,
                "mem": 0
            },
            "pid": "slave(1)@192.168.242.55:5051",
            "registered_time": 1444139993.04084,
            "reserved_resources": {},
            "resources": {
                "cpus": 1,
                "disk": 13778,
                "mem": 748,
                "ports": "[4000-5000, 31000-32000]"
            },
            "unreserved_resources": {
                "cpus": 1,
                "disk": 13778,
                "mem": 748,
                "ports": "[4000-5000, 31000-32000]"
            },
            "used_resources": {
                "cpus": 0.2,
                "disk": 0,
                "mem": 256,
                "ports": "[4680-4680, 4000-4000]"
            }
        }
    ],
    "start_time": 1444139978.91345,
    "unregistered_frameworks": [],
    "version": "0.23.0"
}
`

// TODO: test when multiple active frameworks
func TestFindFrameworks(t *testing.T) {
	t.Parallel()
	ts, mesos := fakeMesos(mesosStateHandler)
	defer ts.Close()

	fws, _ := mesos.FindFrameworks("chronos")
	assert.Equal(t, 1, len(fws))
	assert.Equal(t, "chronos", fws[0].Name)
}

func TestFindFramework(t *testing.T) {
	t.Parallel()
	ts, mesos := fakeMesos(mesosStateHandler)
	defer ts.Close()
	fw, _ := mesos.FindFramework("chronos")
	assert.Equal(t, "chronos", fw.Name)
}

func TestFindFrameworkNoMatching(t *testing.T) {
	t.Parallel()
	ts, mesos := fakeMesos(mesosStateHandler)
	defer ts.Close()
	fw, _ := mesos.FindFramework("fake")
	assert.Nil(t, fw)
}

// TODO: test when multiple active frameworks

func TestFrameworks(t *testing.T) {
	t.Parallel()
	ts, mesos := fakeMesos(mesosStateHandler)
	defer ts.Close()

	fws, _ := mesos.Frameworks()

	fwNames := make([]string, len(fws))
	for i, fw := range fws {
		fwNames[i] = fw.Name
	}

	sort.Strings(fwNames)
	assert.Equal(t, []string{"chronos", "marathon"}, fwNames)
}

func TestShutdown(t *testing.T) {
	t.Parallel()
	ts, mesos := fakeMesos(func(w http.ResponseWriter, r *http.Request) {})
	defer ts.Close()

	fwId := "20151006-135938-938649792-15050-8041-0000"
	err := mesos.Shutdown(fwId)
	assert.Nil(t, err)
}

func TestShutdownError(t *testing.T) {
	t.Parallel()

	errMsg := "No framework found with specified ID"
	statusCode := 400
	fwId := "123"
	ts, mesos := fakeMesos(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		fmt.Fprintf(w, errMsg)
	})
	defer ts.Close()

	err := mesos.Shutdown(fwId)
	assert.NotNil(t, err)
	expected := fmt.Sprintf("Could not shutdown framework %s: %d %s", fwId, statusCode, errMsg)
	assert.Equal(t, expected, err.Error())
}

func TestShutdownFrameworkByName(t *testing.T) {
	t.Parallel()
	var payload string
	ts, mesos := fakeMesos(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/master/state.json" {
			fmt.Fprint(w, stateJson)
		} else {
			reqData, _ := ioutil.ReadAll(r.Body)
			payload = string(reqData)
		}
	})
	defer ts.Close()

	fwName := "chronos"
	err := mesos.ShutdownFrameworkByName(fwName)
	assert.Nil(t, err)
	assert.Equal(t, "frameworkId=20151006-135938-938649792-15050-8041-0000", payload)
}

func TestShutdownFrameworkByNameNotFound(t *testing.T) {
	t.Parallel()
	ts, mesos := fakeMesos(mesosStateHandler)
	defer ts.Close()

	fwName := "fakefw"
	err := mesos.ShutdownFrameworkByName(fwName)
	assert.Nil(t, err)
}

func fakeMesos(handler func(w http.ResponseWriter, r *http.Request)) (*httptest.Server, *Mesos) {
	ts := httptest.NewServer(http.HandlerFunc(handler))

	mesos, _ := NewMesos(ts.URL, "", "", false)
	return ts, mesos
}

func mesosStateHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, stateJson)
}
