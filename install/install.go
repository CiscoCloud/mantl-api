package install

import (
	log "github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
)

type Install struct {
	consul *consul.Client
	kv     *consul.KV
}

func NewInstall(consulClient *consul.Client) *Install {
	return &Install{consulClient, consulClient.KV()}
}

func (install *Install) Packages() (PackageCollection, error) {
	return install.getPackages()
}

func (install *Install) Repositories() (RepositoryCollection, error) {
	return install.getRepositories()
}

func (install *Install) BaseRepository() (*Repository, error) {
	return install.getBaseRepository()
}

func (install *Install) LayerRepositories() (RepositoryCollection, error) {
	return install.getLayerRepositories()
}

func (install *Install) SyncSources(sources []*Source) error {
	// sync repositories if they don't exist
	for _, source := range sources {
		ts, err := install.sourceLastUpdated(source)
		log.Debugf("%s source last updated at %v", source.Name, ts)
		if err != nil || ts.IsZero() {
			log.Debugf("Syncing %v source", source.Name)
			err := install.syncSource(source)
			if err != nil {
				log.Errorf("Could not sync %s source from %s: %v", source.Name, source.Path, err)
				return err
			}
		}
	}
	return nil
}
