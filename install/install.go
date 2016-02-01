package install

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/CiscoCloud/mantl-api/marathon"
	"github.com/CiscoCloud/mantl-api/mesos"
	"github.com/CiscoCloud/mantl-api/zookeeper"
	log "github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
	"strconv"
	"strings"
)

const packageNameKey = "MANTL_PACKAGE_NAME"
const packageVersionKey = "MANTL_PACKAGE_VERSION"
const packageIndexKey = "MANTL_PACKAGE_INDEX"
const packageIsFrameworkKey = "MANTL_PACKAGE_IS_FRAMEWORK"
const packageFrameworkNameKey = "MANTL_PACKAGE_FRAMEWORK_NAME"
const packageUninstallKey = "MANTL_PACKAGE_UNINSTALL"
const dcosPackageFrameworkNameKey = "DCOS_PACKAGE_FRAMEWORK_NAME"
const traefikEnableKey = "traefik.enable"

var apiConfig map[string]interface{}

type Install struct {
	consul    *consul.Client
	kv        *consul.KV
	marathon  *marathon.Marathon
	mesos     *mesos.Mesos
	zookeeper *zookeeper.Zookeeper
}

func NewInstall(consulClient *consul.Client, marathon *marathon.Marathon, mesos *mesos.Mesos, zookeeper *zookeeper.Zookeeper) (*Install, error) {
	if mesos != nil {
		mesosAuthRequired, err := mesos.RequiresAuthentication()
		if err != nil {
			return nil, err
		}

		apiConfig = map[string]interface{}{
			"mantl": map[string]interface{}{
				"mesos": map[string]interface{}{
					"principal":              mesos.Principal,
					"secret":                 mesos.Secret,
					"secret-path":            mesos.SecretPath,
					"authentication-enabled": mesosAuthRequired,
				},
			},
		}
	}

	return &Install{consulClient, consulClient.KV(), marathon, mesos, zookeeper}, nil
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

func (install *Install) InstallPackage(pkgReq *PackageRequest) (string, error) {
	pkgDef, err := install.GetPackageDefinition(pkgReq.Name, pkgReq.Version, pkgReq.Config, apiConfig)

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

	err = addMantlLabels(app, pkgDef)
	if err != nil {
		log.Errorf("Could not add labels to marathon json: %v", err)
		return "", err
	}

	log.Debugf("Submitting application to marathon: %+v", app)

	response, err := install.marathon.CreateApp(app)

	if err != nil {
		log.Errorf("Could not create app in Marathon: %v", err)
		return "", err
	}

	return response, nil
}

func (install *Install) FindInstalled(pkgReq *PackageRequest) ([]*marathon.App, error) {
	installedApps, err := install.installedApps()

	if err != nil {
		log.Errorf("Could not retrieve applications from marathon: %v", err)
		return []*marathon.App{}, err
	}

	matching := filterByPackageName(pkgReq.Name, installedApps)
	if pkgReq.AppID != "" {
		matching = filterByID(pkgReq.AppID, matching)
	}

	return matching, nil
}

func (install *Install) UninstallPackage(app *marathon.App) error {
	if app == nil {
		return errors.New("App cannot be nil when uninstalling a package")
	}

	// remove app from marathon
	_, err := install.marathon.DestroyApp(app.ID)

	if err != nil {
		log.Errorf("Could not destroy app in Marathon: %v", err)
		return err
	}

	// get framework name
	fwName := app.Labels[packageFrameworkNameKey]
	if fwName == "" {
		fwName = app.Labels[dcosPackageFrameworkNameKey]
	}

	if fwName != "" {
		// shutdown mesos framework
		err = install.mesos.ShutdownFrameworkByName(fwName)
		if err != nil {
			log.Errorf("Could not shutdown framework from Mesos: %v", err)
			return err
		}
	}

	// run post-uninstall
	err = install.postUninstall(app)
	if err != nil {
		log.Errorf("Failed to run post-uninstall for %s: %v", app.ID, err)
		return nil
	}

	return nil
}

func (install *Install) SyncSources(sources []*Source, force bool) error {
	// sync repositories if they don't exist
	for _, source := range sources {
		ts, err := install.sourceLastUpdated(source)
		log.Debugf("%s source last updated at %v", source.Name, ts)
		if err != nil || ts.IsZero() || force {
			if force {
				log.Debugf("Forcing sync")
			}
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

func (install *Install) postUninstall(app *marathon.App) error {
	name := app.Labels[packageNameKey]
	encoded := app.Labels[packageUninstallKey]
	if encoded != "" {
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			log.Errorf("Could not perform post-install for %s. Could not decode uninstall json: %v", name, err)
			return err
		}

		uninstall := &packageUninstall{}
		err = json.Unmarshal(decoded, &uninstall)

		if err != nil {
			log.Errorf("Could not perform post-install for %s. Could not decode unmarshal uninstall json: %v", name, err)
			return err
		}

		// run zookeeper delete commands
		if uninstall.Zookeeper != nil && len(uninstall.Zookeeper.Delete) > 0 {
			for _, deleteNode := range uninstall.Zookeeper.Delete {
				if deleteNode.Always {
					install.zookeeper.Delete(deleteNode.Path)
				}
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

func filterByID(id string, apps []*marathon.App) []*marathon.App {
	packages := []*marathon.App{}

	if !strings.HasPrefix(id, "/") {
		id = "/" + id
	}

	for _, app := range apps {
		if app.ID == id {
			packages = append(packages, app)
		}
	}
	return packages
}
func filterPackages(apps []*marathon.App) []*marathon.App {
	packages := []*marathon.App{}

	for _, app := range apps {
		if _, ok := app.Labels[packageNameKey]; ok {
			packages = append(packages, app)
		}
	}

	return packages
}

func filterByPackageName(name string, apps []*marathon.App) []*marathon.App {
	packages := []*marathon.App{}

	for _, app := range apps {
		if app.Labels[packageNameKey] == name {
			packages = append(packages, app)
		}
	}

	return packages
}

func addMantlLabels(app *marathon.App, pkgDef *packageDefinition) error {
	if app.Labels == nil {
		app.Labels = make(map[string]string)
	}
	app.Labels[packageNameKey] = pkgDef.name
	app.Labels[packageVersionKey] = pkgDef.version
	app.Labels[packageIndexKey] = pkgDef.release
	app.Labels[packageIsFrameworkKey] = strconv.FormatBool(pkgDef.framework)

	uninstallJson, err := pkgDef.UninstallJson()
	if err != nil {
		return err
	}
	app.Labels[packageUninstallKey] = base64.StdEncoding.EncodeToString([]byte(uninstallJson))

	if pkgDef.frameworkName != "" {
		app.Labels[packageFrameworkNameKey] = pkgDef.frameworkName
	}

	// copy DCOS_PACKAGE_FRAMEWORK_NAME if it exists
	if fwName, ok := app.Labels[dcosPackageFrameworkNameKey]; ok {
		app.Labels[packageFrameworkNameKey] = fwName
	}

	lb, err := pkgDef.LoadBalancer()
	if err == nil {
		app.Labels[traefikEnableKey] = strconv.FormatBool(lb == "external")
	} else {
		log.Warnf("Unable to retrieve load balancer configuration: %s", err.Error())
	}

	return nil
}
