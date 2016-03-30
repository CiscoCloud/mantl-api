package install

import (
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/Wuvist/mustache"
	"path"
	"sort"
	"strings"
)

type PackageVersion struct {
	Version   string `json:"version"`
	Index     string `json:"index"`
	Supported bool   `json:"supported"`
}

type packageVersionByMostRecent []*PackageVersion

func (p packageVersionByMostRecent) Len() int           { return len(p) }
func (p packageVersionByMostRecent) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p packageVersionByMostRecent) Less(i, j int) bool { return p[j].Index < p[i].Index }

type PackageRequest struct {
	Name             string                 `json:"name"`
	Version          string                 `json:"version"`
	AppID            string                 `json:"id"`
	Config           map[string]interface{} `json:"config"`
	UninstallOptions map[string]interface{} `json:"uninstallOptions"`
}

func NewPackageRequest(data []byte) (request *PackageRequest, err error) {
	request = &PackageRequest{}
	if len(data) > 0 {
		err = json.Unmarshal(data, &request)
	}

	if err == nil {
		if request.Name == "" {
			err = errors.New("Name is required")
		}
	}

	return request, err
}

type Package struct {
	Name           string                     `json:"name"`
	Description    string                     `json:"description"`
	Framework      bool                       `json:"framework"`
	CurrentVersion string                     `json:"currentVersion"`
	Supported      bool                       `json:"supported"`
	Tags           []string                   `json:"tags"`
	Versions       map[string]*PackageVersion `json:"versions"`
}

func NewPackage(name string) *Package {
	return &Package{
		Name:     name,
		Versions: make(map[string]*PackageVersion),
	}
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

func (p Package) PackageVersions() []*PackageVersion {
	var versions []*PackageVersion
	for _, pv := range p.Versions {
		versions = append(versions, pv)
	}
	return versions
}

func (p Package) SupportedVersions() []*PackageVersion {
	var versions []*PackageVersion
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

func (p Package) GetPackageVersion(version string) *PackageVersion {
	for _, v := range p.PackageVersions() {
		if strings.EqualFold(v.Version, strings.TrimSpace(version)) {
			return v
		}
	}
	return nil
}

func (p Package) FindLatestPackageVersion() *PackageVersion {
	versions := p.PackageVersions()
	sort.Sort(packageVersionByMostRecent(versions))
	if len(versions) > 0 {
		return versions[0]
	} else {
		return nil
	}
}

func (p Package) FindLatestSupportedPackageVersion() *PackageVersion {
	versions := p.PackageVersions()
	sort.Sort(packageVersionByMostRecent(versions))

	for _, v := range versions {
		if v.Supported {
			return v
		}
	}

	return nil
}

func (p Package) FindPackageVersion(version string) *PackageVersion {
	pkgVer := p.GetPackageVersion(version)
	if pkgVer == nil {
		pkgVer = p.GetPackageVersion(p.CurrentVersion)
		if pkgVer == nil {
			pkgVer = p.FindLatestSupportedPackageVersion()
			if pkgVer == nil {
				pkgVer = p.FindLatestPackageVersion()
			}
		}
	}

	return pkgVer
}

type PackageCollection []*Package

type packageConfigGroup struct {
	Description          string                        `json:"description"`
	Type                 string                        `json:"type"`
	AdditionalProperties bool                          `json:"additionalProperties"`
	Properties           map[string]packageConfigGroup `json:"properties"`
	Required             []string                      `json:"required"`
	Minimum              interface{}                   `json:"minimum"`
	Default              interface{}                   `json:"default"`
}

func (g packageConfigGroup) defaultConfig() map[string]interface{} {
	defaults := make(map[string]interface{})

	for groupName, group := range g.Properties {
		if group.Default != nil {
			defaults[groupName] = transformedConfigValue(group.Default, group.Type)
		} else if group.Type == "object" {
			defaults[groupName] = group.defaultConfig()
		}
	}

	return defaults
}

type packageDefinition struct {
	commandJson   []byte
	configJson    []byte
	marathonJson  []byte
	packageJson   []byte
	optionsJson   []byte
	uninstallJson []byte
	apiConfig     map[string]interface{}
	userConfig    map[string]interface{}
	name          string
	version       string
	release       string
	framework     bool
	frameworkName string
}

func (d packageDefinition) IsValid() bool {
	return len(d.configJson) > 0 &&
		len(d.marathonJson) > 0 &&
		len(d.packageJson) > 0
}

func (d packageDefinition) ConfigSchema() (packageConfigGroup, error) {
	config := packageConfigGroup{}
	if len(d.configJson) > 0 {
		err := json.Unmarshal(d.configJson, &config)
		if err != nil {
			log.Errorf("Could not unmarshal configuration schema: %v", err)
			return config, err
		}
	}
	return config, nil
}

func (d packageDefinition) Options() (map[string]interface{}, error) {
	var options map[string]interface{}
	if len(d.optionsJson) > 0 {
		// Render options
		optionsTemplate := string(d.optionsJson)
		tmpl, err := mustache.ParseString(optionsTemplate)
		if err != nil {
			log.Errorf("Could not parse options template: %v", err)
			return nil, err
		}

		renderedOptions := tmpl.Render(d.apiConfig)

		err = json.Unmarshal([]byte(renderedOptions), &options)
		if err != nil {
			log.Errorf("Could not unmarshal options json: %v", err)
			return nil, err
		}

		// merge user config
		mergeConfig(options, d.userConfig)

		// add api config to options
		mergeConfig(options, d.apiConfig)
	}

	return options, nil
}

func (d packageDefinition) MergedConfig() (map[string]interface{}, error) {
	schema, err := d.ConfigSchema()

	if err != nil {
		log.Errorf("Could not retrieve configuration schema")
		return nil, err
	}

	options, err := d.Options()

	if err != nil {
		log.Errorf("Could not retrieve options")
		return nil, err
	}

	config := schema.defaultConfig()

	return mergeConfig(config, options), nil
}

func (d packageDefinition) MarathonAppJson() (string, error) {
	return d.renderJsonMustacheTemplate(d.marathonJson)
}

func (d packageDefinition) UninstallJson() (string, error) {
	return d.renderJsonMustacheTemplate(d.uninstallJson)
}

func (d packageDefinition) LoadBalancer() (string, error) {
	config, err := d.MergedConfig()
	if err != nil {
		log.Errorf("Unable to retrieve package definition configuration: %v", err)
		return "off", err
	}

	lb := "off"
	v, ok := getConfig(config, "mantl.load-balancer").(string)
	if ok {
		lb = strings.ToLower(v)
	}
	return lb, nil
}

func (d packageDefinition) renderJsonMustacheTemplate(jsonBlob []byte) (string, error) {
	template := string(jsonBlob)
	config, err := d.MergedConfig()
	if err != nil {
		log.Errorf("Unable to retrieve package definition configuration: %v", err)
		return "", err
	}

	// Render template with config
	tmpl, err := mustache.ParseString(template)
	if err != nil {
		log.Errorf("Could not parse template: %v", err)
		return "", err
	}

	json := tmpl.Render(config)

	return json, nil
}

type packageUninstall struct {
	Zookeeper *zookeeperCommands `json:"zookeeper"`
}

type zookeeperCommands struct {
	Delete []*zookeeperNode `json:"delete"`
}

type zookeeperNode struct {
	Path   string `json:"path"`
	Always bool   `json:"always"`
}

func (install *Install) getPackages() (PackageCollection, error) {
	catalog, err := NewPackageCatalog(install.kv, RepositoryRoot)
	if err != nil {
		return nil, err
	}

	return catalog.packagesIndex()
}

func (install *Install) getPackageByName(name string) (*Package, error) {
	packages, err := install.getPackages()

	if err != nil {
		return nil, err
	}

	n := strings.TrimSpace(name)
	for _, p := range packages {
		if strings.EqualFold(n, p.Name) {
			return p, nil
		}
	}

	return nil, nil
}

func (install *Install) GetPackageDefinition(name string, version string, userConfig map[string]interface{}, apiConfig map[string]interface{}) (*packageDefinition, error) {
	pkg, err := install.getPackageByName(name)
	if err != nil {
		return nil, err
	}

	if pkg == nil {
		return nil, errors.New(fmt.Sprintf("Could not find %s package", name))
	}

	pkgVersion := pkg.FindPackageVersion(version)
	if pkgVersion == nil {
		return nil, errors.New(fmt.Sprintf("Could not find installable version for %s", name))
	}

	repositories, err := install.Repositories()
	if err != nil {
		return nil, err
	}

	pkgDef := &packageDefinition{
		name:       pkg.Name,
		version:    pkgVersion.Version,
		release:    pkgVersion.Index,
		framework:  pkg.Framework,
		apiConfig:  apiConfig,
		userConfig: userConfig,
	}

	for _, repo := range repositories {
		pkgKey := path.Join(
			repo.PackagesKey(),
			pkg.PackageVersionKey(pkgVersion.Index),
		)

		data := install.getPackageDefinitionFile("command.json", pkgKey)
		if len(data) > 0 {
			pkgDef.commandJson = data
		}
		data = install.getPackageDefinitionFile("config.json", pkgKey)
		if len(data) > 0 {
			pkgDef.configJson = data
		}
		data = install.getPackageDefinitionFile("marathon.json", pkgKey)
		if len(data) > 0 {
			pkgDef.marathonJson = data
		}
		data = install.getPackageDefinitionFile("package.json", pkgKey)
		if len(data) > 0 {
			pkgDef.packageJson = data
		}
		data = install.getPackageDefinitionFile("mantl.json", pkgKey)
		if len(data) > 0 {
			pkgDef.optionsJson = data
		}
		data = install.getPackageDefinitionFile("uninstall.json", pkgKey)
		if len(data) > 0 {
			pkgDef.uninstallJson = data
		}
	}

	config, err := pkgDef.MergedConfig()
	if err != nil {
		log.Errorf("Unable to retrieve package definition configuration: %v", err)
		return nil, err
	}

	fwName := getConfigVal(config, pkgDef.name, "framework-name")
	if fwName != nil {
		if fwName, ok := fwName.(string); ok {
			pkgDef.frameworkName = fwName
		}
	}

	return pkgDef, nil
}

func (install *Install) getPackageDefinitionFile(name string, key string) []byte {
	kp, _, err := install.kv.Get(path.Join(key, name), nil)
	if err != nil {
		log.Errorf("Could not retrieve %s from %s: %v", name, key, err)
		return []byte{}
	}

	if kp != nil {
		return kp.Value
	}

	return []byte{}
}

func transformedConfigValue(val interface{}, typ string) interface{} {
	// TODO: probably should use the config schema for this
	if slice, ok := val.([]interface{}); ok {
		// if the config val is an array, convert it to a json representation
		blob, err := json.Marshal(slice)
		if err == nil {
			return string(blob)
		} else {
			log.Warnf("Could not marshal %+v config value to json: %v", val, err)
			return val
		}
	} else {
		switch typ {
		case "integer":
			return fmt.Sprintf("%d", int(val.(float64)))
		case "number":
			return fmt.Sprintf("%0.2f", val.(float64))
		default:
			return val
		}
	}
}

func mergeConfig(config map[string]interface{}, override map[string]interface{}) map[string]interface{} {
	for k, v := range override {
		_, configExists := config[k]
		configVal, configValIsMap := config[k].(map[string]interface{})
		overrideVal, overrideValIsMap := v.(map[string]interface{})
		if configExists && configValIsMap && overrideValIsMap {
			config[k] = mergeConfig(configVal, overrideVal)
		} else {
			config[k] = transformedConfigValue(v, "")
		}
	}

	return config
}

func getConfig(m map[string]interface{}, node string) interface{} {
	keys := strings.Split(node, ".")
	return getConfigVal(m, keys...)
}

func getConfigVal(m map[string]interface{}, keys ...string) interface{} {
	nested := m
	for _, key := range keys {
		if nm, ok := nested[key].(map[string]interface{}); ok {
			nested = nm
		} else if val, ok := nested[key]; ok {
			return val
		}
	}
	return nil
}
