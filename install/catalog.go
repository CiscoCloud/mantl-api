package install

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
	"sort"
	"strings"
)

type packageCatalog struct {
	catalog map[string]map[string]map[string]string
	kv      *consul.KV
}

func (c packageCatalog) names() []string {
	names := make([]string, 0, len(c.catalog))
	for name, _ := range c.catalog {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (c packageCatalog) packageKeys() map[string][]string {
	keys := make(map[string][]string)

	for _, name := range c.names() {
		keySlice := keys[name]

		repos := c.catalog[name]
		repoIdxs := make([]string, 0, len(repos))
		for repoIdx, _ := range repos {
			repoIdxs = append(repoIdxs, repoIdx)
		}
		sort.Strings(repoIdxs)

		for _, repoIdx := range repoIdxs {
			versions := c.catalog[name][repoIdx]
			verIdxs := make([]string, 0, len(versions))
			for verIdx, _ := range versions {
				verIdxs = append(verIdxs, verIdx)
			}
			sort.Strings(verIdxs)

			for _, verIdx := range verIdxs {
				pkgKey := c.catalog[name][repoIdx][verIdx]
				keySlice = append(keySlice, pkgKey)
			}
		}

		keys[name] = keySlice
	}

	return keys
}

func (c packageCatalog) packagesIndex() (PackageCollection, error) {
	packages := PackageCollection{}

	keyMap := c.packageKeys()

	for _, name := range c.names() {
		pkg := NewPackage(name)
		keys := keyMap[name]

		supported := false
		supportedVersions := make([]string, 0, len(keys))

		for _, key := range keys {
			versionIndex := packageVersionIndex(key)

			supportedVersion := keyExists(key+"mantl.json", c.kv)
			if supportedVersion {
				supported = true
				supportedVersions = append(supportedVersions, versionIndex)
			}

			meta := c.packageMeta(key)
			if meta != nil {
				if desc, ok := meta["description"]; ok {
					pkg.Description = desc.(string)
				}
				if isFramework, ok := meta["framework"]; ok {
					pkg.Framework = isFramework.(bool)
				}

				if tagList, ok := meta["tags"].([]interface{}); ok {
					tags := make([]string, len(tagList))
					for i, tag := range tagList {
						tags[i] = tag.(string)
					}
					pkg.Tags = tags
				}

				version := meta["version"].(string)

				pkgVersion := &PackageVersion{
					Version: version,
					Index:   versionIndex,
				}
				pkg.Versions[version] = pkgVersion
			}
		}

		pkg.Supported = supported

		for _, sv := range supportedVersions {
			for _, pv := range pkg.Versions {
				if sv == pv.Index {
					pv.Supported = true
					pkg.CurrentVersion = pv.Version
				}
			}
		}

		if pkg.CurrentVersion == "" {
			latest := pkg.FindLatestPackageVersion()
			if latest != nil {
				pkg.CurrentVersion = latest.Version
			}
		}

		packages = append(packages, pkg)
	}

	return packages, nil
}

func (c packageCatalog) packageMeta(key string) (meta map[string]interface{}) {
	kp, _, err := c.kv.Get(key+"package.json", nil)
	if err != nil {
		log.Warnf("Could not get package key %s: %v", key, err)
		return nil
	}

	if kp != nil {
		err = json.Unmarshal(kp.Value, &meta)
		if err != nil {
			log.Warnf("Could not get unmarshal package.json from %s: %v", key, err)
			return nil
		}
	}

	return meta
}

func packageVersionIndex(key string) string {
	parts := strings.Split(strings.TrimSuffix(key, "/"), "/")
	return parts[len(parts)-1]
}

func keyExists(key string, kv *consul.KV) bool {
	kp, _, err := kv.Get(key, nil)
	if err != nil {
		log.Warnf("Could not get key path %s: %v", key, err)
		return false
	}

	return kp != nil
}

func NewPackageCatalog(kv *consul.KV, repositoryRoot string) (*packageCatalog, error) {
	catalog := &packageCatalog{kv: kv}
	pkgIndex := make(map[string]map[string]map[string]string)

	keys, _, err := kv.Keys(repositoryRoot, "", nil)
	if err != nil {
		return nil, err
	}
	sort.Strings(keys)

	// package key example: mantl-install/repository/0/repo/packages/S/spark/3/config.json
	for _, key := range keys {
		parts := strings.Split(key, "/")
		if len(parts) == 9 {
			repoIdx := parts[2]
			name := parts[6]
			verIdx := parts[7]
			_, ok := pkgIndex[name]
			if !ok {
				pkgIndex[name] = make(map[string]map[string]string)
			}

			_, ok = pkgIndex[name][repoIdx]
			if !ok {
				pkgIndex[name][repoIdx] = make(map[string]string)
			}

			_, ok = pkgIndex[name][repoIdx][verIdx]
			if !ok {
				pkgKey := key[0 : strings.LastIndex(key, "/")+1]
				pkgIndex[name][repoIdx][verIdx] = pkgKey
			}
		}
	}

	catalog.catalog = pkgIndex
	return catalog, nil
}
