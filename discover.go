package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
)

type discoveryRecord struct {
	name            string
	tag             string
	discoveredHosts []string
}

func NewDiscovery(client *consul.Client, name string, tag string) *discoveryRecord {
	rec := &discoveryRecord{
		name:            name,
		tag:             tag,
		discoveredHosts: []string{},
	}

	discover(client, rec)

	return rec
}

func discover(client *consul.Client, r *discoveryRecord) {
	services, _, err := client.Catalog().Service(r.name, r.tag, nil)
	if err != nil {
		log.Warnf("Couldn't get %s services from consul: %v", r.name, err)
	}

	var hosts []string
	for _, svc := range services {
		host := svc.Node
		port := svc.ServicePort
		location := fmt.Sprintf("%s:%d", host, port)
		hosts = append(hosts, location)
	}
	r.discoveredHosts = hosts
	log.Debugf("Discovered %s service hosts: %v", r.name, r.discoveredHosts)
}
