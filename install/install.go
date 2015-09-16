package install

import (
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
	"github.com/ryane/mantl-api/marathon"
	"github.com/ryane/mantl-api/mesos"
	"strconv"
)

const packageNameKey = "MANTL_PACKAGE_NAME"
const packageVersionKey = "MANTL_PACKAGE_VERSION"
const packageIndexKey = "MANTL_PACKAGE_INDEX"
const packageIsFrameworkKey = "MANTL_PACKAGE_IS_FRAMEWORK"
const packageFrameworkNameKey = "MANTL_PACKAGE_FRAMEWORK_NAME"
const dcosPackageFrameworkNameKey = "DCOS_PACKAGE_FRAMEWORK_NAME"

type Install struct {
	consul   *consul.Client
	kv       *consul.KV
	marathon *marathon.Marathon
	mesos    *mesos.Mesos
}

func NewInstall(consulClient *consul.Client, marathon *marathon.Marathon, mesos *mesos.Mesos) *Install {
	return &Install{consulClient, consulClient.KV(), marathon, mesos}
}

func (install *Install) Packages() (PackageCollection, error) {
	return install.getPackages()
}

func (install *Install) Package(name string) (*Package, error) {
	return install.getPackageByName(name)
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

func (install *Install) InstallPackage(pkgReq *PackageRequest) (string, error) {
	internalConfig := map[string]string{
		"mantl-install-mesos-principal": install.mesos.Principal,
		"mantl-install-mesos-secret":    install.mesos.GetCredential(install.mesos.Principal),
	}

	pkgDef, err := install.GetPackageDefinition(pkgReq.Name, pkgReq.Version, internalConfig)
	if err != nil {
		log.Errorf("Could not find package definition: %v", err)
		return "", err
	}

	marathonJson, err := pkgDef.MarathonAppJson()
	if err != nil {
		log.Errorf("Could not generate marathon json: %v", err)
		return "", err
	}

	app, err := install.marathon.ToApp(marathonJson)
	if err != nil {
		log.Errorf("Could not unmarshal marathon json: %v", err)
		return "", err
	}

	addMantlLabels(app, pkgDef)

	log.Debugf("Submitting application to marathon: %+v", app)

	response, err := install.marathon.CreateApp(app)

	if err != nil {
		log.Errorf("Could not create app in Marathon: %v", err)
		return "", err
	}

	return response, nil
}

func (install *Install) UninstallPackage(pkgReq *PackageRequest) (string, error) {
	// find marathon app by id
	matching := install.findInstalledApp(pkgReq)

	if matching == nil {
		log.Warnf("Could not find matching package for %s %s", pkgReq.Name, pkgReq.Version)
		return "", nil
	}

	log.Debugf("Found matching app at %s", matching.ID)

	// get framework name
	fwName := matching.Labels[packageFrameworkNameKey]
	if fwName == "" {
		fwName = matching.Labels[dcosPackageFrameworkNameKey]
	}

	// remove app from marathon
	_, err := install.marathon.DestroyApp(matching.ID)

	if err != nil {
		log.Errorf("Could not destroy app in Marathon: %v", err)
		return "", err
	}

	// find mesos framework
	state, _ := install.mesos.State()
	matchingFrameworks := make(map[string]*mesos.Framework)
	for _, fw := range state.AllFrameworks() {
		if fw.Name == fwName {
			matchingFrameworks[fw.ID] = fw
		}
	}

	log.Debug(matchingFrameworks)

	if fwCount := len(matchingFrameworks); fwCount > 1 {
		return "", errors.New(fmt.Sprintf("There are %d %s frameworks.", fwCount, fwName))
	}

	var frameworkId string
	for fwid, _ := range matchingFrameworks {
		frameworkId = fwid
		break
	}

	// shutdown mesos framework
	install.mesos.Shutdown(frameworkId)

	return "", nil
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

func (install *Install) installedApps() ([]*marathon.App, error) {
	apps, err := install.marathon.Apps()
	if err != nil {
		log.Errorf("Could not retrieve installed apps from Marathon: %v", err)
		return nil, err
	}

	return apps, err
}

func (install *Install) findInstalledApp(pkgReq *PackageRequest) *marathon.App {
	apps, err := install.installedApps()
	if err != nil {
		return nil
	}

	// TODO: this needs to be more sophisticated
	// TODO: take version into account
	// TODO: check and prompt if more than 1 matching instance
	var matching *marathon.App
	for _, app := range apps {
		if app.Labels[packageNameKey] == pkgReq.Name {
			matching = app
			break
		}
	}
	return matching
}

func addMantlLabels(app *marathon.App, pkgDef *packageDefinition) {
	app.Labels[packageNameKey] = pkgDef.name
	app.Labels[packageVersionKey] = pkgDef.version
	app.Labels[packageIndexKey] = pkgDef.release
	app.Labels[packageIsFrameworkKey] = strconv.FormatBool(pkgDef.framework)

	if pkgDef.frameworkName != "" {
		app.Labels[packageFrameworkNameKey] = pkgDef.frameworkName
	}

	// copy DCOS_PACKAGE_FRAMEWORK_NAME if it exists
	if fwName, ok := app.Labels[dcosPackageFrameworkNameKey]; ok {
		app.Labels[packageFrameworkNameKey] = fwName
	}
}
