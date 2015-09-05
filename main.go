package main

import (
	log "github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
	"github.com/ryane/mantl-api/packages"
	"github.com/ryane/mantl-api/repository"
	"github.com/ryane/mantl-api/source"
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

	sources := []*source.Source{
		&source.Source{
			Name:       "mantl",
			Path:       "/Users/ryan/Downloads/universe",
			SourceType: source.FileSystem,
			Index:      1,
		},
		&source.Source{
			Name:       "mesosphere",
			Path:       "https://github.com/mesosphere/universe.git",
			SourceType: source.Git,
			Index:      0,
		},
	}

	// sync repositories if they don't exist
	for _, source := range sources {
		ts, err := source.LastUpdated(client)
		log.Debugf("%s source last updated at %v", source.Name, ts)
		if err != nil || ts.IsZero() {
			log.Debugf("Syncing %v source", source.Name)
			err := source.Sync(client)
			if err != nil {
				log.Errorf("Could not sync %s source from %s: %v", source.Name, source.Path, err)
			}
		}
	}

	// start listener

	// list packages
	packages, err := packages.Packages(client)
	if err != nil {
		log.Errorf("Could not retrieve packages: %v", err)
	}

	for _, p := range packages {
		log.Debugf("%s (%s)", p.Name, p.CurrentVersion)
		log.Debugf("%+v", p)
		log.Debug("")
	}

	repos, err := repository.Repositories(client)
	for _, r := range repos {
		log.Debugf("%v (%d)", r.Name, r.Index)
	}
}

func testConsul(client *consul.Client) error {
	kv := client.KV()
	_, _, err := kv.Get("mantl-install", nil)
	return err
}
