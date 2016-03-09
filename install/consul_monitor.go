package install

import (
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/hashicorp/consul/api"
)

func (inst *Install) Watch(intervalSeconds time.Duration) {
	ticker := time.NewTicker(intervalSeconds * time.Second)
	kv := inst.consul.KV()
	for _ = range ticker.C {
		kvps, _, err := kv.List(AppsRoot, nil)
		if err != nil {
			log.Warnf("Could not retrieve %s keys: %v", AppsRoot, err)
			continue
		}

		for _, kvp := range kvps {
			appName := strings.TrimPrefix(kvp.Key, AppsRoot)
			if appName != "" {
				inst.installPackageFromKVPair(kvp, kv)
			}
		}
	}
}

func (inst *Install) installPackageFromKVPair(kvp *api.KVPair, kv *api.KV) {
	defer func() {
		_, err := kv.Delete(kvp.Key, nil)
		if err != nil {
			log.Errorf("Could not delete %s key: %v", kvp.Key, err)
		}
	}()

	pkgReq, err := NewPackageRequest(kvp.Value)
	if err != nil {
		log.Warnf("Failed to parse package request from %s", kvp.Key)
		return
	}

	_, err = inst.InstallPackage(pkgReq)
	if err != nil {
		log.Errorf("Failed to install app from %s: %v", kvp.Key, err)
	}
}
