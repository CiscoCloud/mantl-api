package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
)

var domain string

type discoveryRecord struct {
	name          string
	tag           string
	port          int
	scheme        string
	defaultUrl    string
	discoveredUrl string
	domain        string
}

func NewDiscovery(client *consul.Client, name string, tag string, scheme string, defaultUrl string) *discoveryRecord {
	if domain == "" {
		log.Debug("Looking up consul domain")
		domain = consulDomain(client)
	}

	rec := &discoveryRecord{
		name:       name,
		tag:        tag,
		defaultUrl: defaultUrl,
		domain:     domain,
		scheme:     scheme,
	}

	discover(client, rec)

	return rec
}

func discover(client *consul.Client, r *discoveryRecord) {
	services, _, err := client.Catalog().Service(r.name, r.tag, nil)
	if err != nil {
		log.Warnf("Couldn't get %s services from consul: %v", r.name, err)
	}

	if len(services) > 0 {
		serviceName := services[0].ServiceName
		r.port = services[0].ServicePort
		location := fmt.Sprintf("%s.service.%s:%d", serviceName, r.domain, r.port)
		if r.tag != "" {
			location = fmt.Sprintf("%s.%s", r.tag, location)
		}

		if r.scheme != "" {
			r.discoveredUrl = fmt.Sprintf("%s://%s", r.scheme, location)
		} else {
			r.discoveredUrl = location
		}
	} else {
		r.discoveredUrl = r.defaultUrl
	}
	log.Debugf("Discovered %s service url: %s", r.name, r.discoveredUrl)
}

func consulDomain(client *consul.Client) string {
	domain := "consul"

	agentConfig, err := client.Agent().Self()
	if err != nil {
		log.Warnf("Could not get consul agent info: %v", err)
		return domain
	}

	if config, ok := agentConfig["Config"]; ok {
		if consulDomain, ok := config["Domain"]; ok {
			if consulDomain, ok := consulDomain.(string); ok {
				domain = consulDomain
			}
		}
	}

	return domain
}
