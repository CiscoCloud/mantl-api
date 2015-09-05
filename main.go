package main

import (
	log "github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
	"github.com/ryane/mantl-api/api"
	"github.com/ryane/mantl-api/install"
)

func main() {
	log.SetLevel(log.DebugLevel)

	// TODO: configurable consul client
	client, err := consul.NewClient(consul.DefaultConfig())
	if err != nil {
		log.Fatalf("Could not create consul client: %v", err)
	}

	// abort if we cannot connect to consul
	err = testConsul(client)
	if err != nil {
		log.Fatalf("Could not connect to consul: %v", err)
	}

	inst := install.NewInstall(client)

	// sync sources to consul
	sources := []*install.Source{
		&install.Source{
			Name:       "mantl",
			Path:       "/Users/ryan/Downloads/universe",
			SourceType: install.FileSystem,
			Index:      1,
		},
		&install.Source{
			Name:       "mesosphere",
			Path:       "https://github.com/mesosphere/universe.git",
			SourceType: install.Git,
			Index:      0,
		},
	}
	inst.SyncSources(sources)

	// start listener
	// TODO: configurable api (port, address, etc.)
	api.NewApi(":4001", inst).Start()
}

func testConsul(client *consul.Client) error {
	kv := client.KV()
	_, _, err := kv.Get("mantl-install", nil)
	return err
}
