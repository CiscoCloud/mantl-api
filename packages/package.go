package packages

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
	"github.com/ryane/mantl-api/repository"
	"path"
	"sort"
	"strings"
)

type PackageVersion struct {
	Version   string
	Index     string
	Supported bool
}

type PackageVersionByMostRecent []PackageVersion

func (p PackageVersionByMostRecent) Len() int           { return len(p) }
func (p PackageVersionByMostRecent) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p PackageVersionByMostRecent) Less(i, j int) bool { return p[j].Index < p[i].Index }

type Package struct {
	Name           string
	Description    string
	Framework      bool
	CurrentVersion string
	Supported      bool
	Tags           []string
	Versions       map[string]PackageVersion
}

type PackageCollection []*Package

type packageIndex struct {
	Packages []packageIndexEntry
}

type packageIndexEntry struct {
	CurrentVersion string
	Description    string
	Framework      bool
	Name           string
	Tags           []string
	Versions       map[string]string
}

func (p Package) ContainerId() string {
	return strings.ToUpper(string([]rune(p.Name)[0]))
}

func (p Package) PackageKey() string {
	return path.Join(
		p.ContainerId(),
		p.Name,
	)
}

func (p Package) PackageVersionKey(index string) string {
	return path.Join(
		p.PackageKey(),
		index,
	)
}

func (p Package) PackageVersions() []PackageVersion {
	versions := make([]PackageVersion, len(p.Versions))
	for _, pv := range p.Versions {
		versions = append(versions, pv)
	}
	return versions
}

func (p Package) SupportedVersions() []PackageVersion {
	var versions []PackageVersion
	for _, pv := range p.PackageVersions() {
		if pv.Supported {
			versions = append(versions, pv)
		}
	}
	return versions
}

func (p Package) HasSupportedVersion() bool {
	return len(p.SupportedVersions()) > 0
}

func (p packageIndexEntry) ToPackage() *Package {
	pkg := &Package{
		Name:           p.Name,
		Description:    p.Description,
		Framework:      p.Framework,
		CurrentVersion: p.CurrentVersion,
		Tags:           p.Tags,
	}

	pkg.Versions = make(map[string]PackageVersion)
	for version, index := range p.Versions {
		pkg.Versions[version] = PackageVersion{
			Version:   version,
			Index:     index,
			Supported: false,
		}
	}
	return pkg
}

func Packages(client *consul.Client) ([]*Package, error) {
	packageIndexEntries, err := packageIndexEntries(client)
	if err != nil {
		log.Errorf("Could not retrieve base package index: %v", err)
		return nil, err
	}

	packages := make(PackageCollection, len(packageIndexEntries))
	for i, entry := range packageIndexEntries {
		pkg := entry.ToPackage()
		setSupportedVersions(client, pkg)
		setCurrentVersion(pkg)
		packages[i] = pkg
	}

	return packages, nil
}

func setSupportedVersions(client *consul.Client, pkg *Package) {
	kv := client.KV()
	layers, err := repository.Layers(client)
	if err != nil {
		log.Errorf("Could not read layer repositories: %v", err)
		return
	}

	for version, pkgVersion := range pkg.Versions {
		for _, layer := range layers {
			versionKey := path.Join(
				layer.PackagesKey(),
				pkg.PackageVersionKey(pkgVersion.Index),
				"mantl.json",
			)

			kp, _, err := kv.Get(versionKey, nil)
			if err != nil {
				log.Warnf("Could not read %s: %v", versionKey, err)
			}

			pkgVersion.Supported = kp != nil
			pkg.Versions[version] = pkgVersion
		}
	}

	pkg.Supported = pkg.HasSupportedVersion()
}

func setCurrentVersion(pkg *Package) {
	if !pkg.Supported || !pkg.HasSupportedVersion() {
		// we don't support any version so defer to the base package
		return
	}

	if cv, ok := pkg.Versions[pkg.CurrentVersion]; ok {
		if cv.Supported {
			// CurrentVersion is supported so we leave it alone
			return
		}
	}

	// CurrentVersion is not supported so we want to set it to the highest supported version
	versions := pkg.SupportedVersions()
	sort.Sort(PackageVersionByMostRecent(versions))
	for _, pv := range versions {
		if pv.Supported {
			pkg.CurrentVersion = pv.Version
			break
		}
	}
}

func packageIndexEntries(client *consul.Client) ([]packageIndexEntry, error) {
	baseRepo, err := repository.BaseRepository(client)
	if err != nil || baseRepo == nil {
		log.Errorf("Could not retrieve base repository: %v", err)
		return nil, err
	}

	baseIndex := baseRepo.PackageIndexKey()

	kv := client.KV()
	kp, _, err := kv.Get(baseIndex, nil)
	if err != nil || kp == nil {
		log.Errorf("Could not read %s: %v", baseIndex, err)
		return nil, err
	}

	var packageIndex packageIndex
	err = json.Unmarshal(kp.Value, &packageIndex)
	if err != nil {
		log.Errorf("Could not unmarshal index from %s: %v", baseIndex, err)
		return nil, err
	}

	return packageIndex.Packages, nil
}
