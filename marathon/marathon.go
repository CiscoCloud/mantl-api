package marathon

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	log "github.com/Sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

type Marathon struct {
	Location    string
	Protocol    string
	Username    string
	Password    string
	NoVerifySsl bool
}

type PortMapping struct {
	ContainerPort int    `json:"containerPort,omitempty"`
	HostPort      int    `json:"hostPort,omitempty"`
	ServicePort   int    `json:"servicePort,omitempty"`
	Protocol      string `json:"protocol,omitempty"`
}

type Parameter struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type Docker struct {
	Image          string        `json:"image,omitempty"`
	Parameters     []Parameter   `json:"parameters,omitempty"`
	Privileged     bool          `json:"privileged,omitempty"`
	Network        string        `json:"network,omitempty"`
	PortMappings   []PortMapping `json:"portMappings,omitempty"`
	ForcePullImage bool          `json:"forcePullImage,omitempty"`
}

type Volume struct {
	ContainerPath string `json:"containerPath,omitempty"`
	HostPath      string `json:"hostPath,omitempty"`
	Mode          string `json:"mode,omitempty"`
}

type Container struct {
	Docker  *Docker  `json:"docker,omitempty"`
	Type    string   `json:"type,omitempty"`
	Volumes []Volume `json:"volumes,omitempty"`
}

type HealthCheck struct {
	Path                   string              `json:"path,omitempty"`
	PortIndex              int                 `json:"portIndex,omitempty"`
	Protocol               string              `json:"protocol,omitempty"`
	GracePeriodSeconds     int                 `json:"gracePeriodSeconds,omitempty"`
	IntervalSeconds        int                 `json:"intervalSeconds,omitempty"`
	TimeoutSeconds         int                 `json:"timeoutSeconds,omitempty"`
	MaxConsecutiveFailures int                 `json:"maxConsecutiveFailures,omitempty"`
	Command                *HealthCheckCommand `json:"command,omitempty"`
}

type HealthCheckCommand struct {
	Value string `json:"value,omitempty"`
}

type UpgradeStrategy struct {
	MinimumHealthCapacity float64 `json:"minimumHealthCapacity,omitempty"`
	MaximumOverCapacity   float64 `json:"maximumOverCapacity,omitempty"`
}

type App struct {
	Args            []string          `json:"args,omitempty"`
	BackoffFactor   float64           `json:"backoffFactor,omitempty"`
	BackoffSeconds  int               `json:"backoffSeconds,omitempty"`
	Cmd             string            `json:"cmd,omitempty"`
	Constraints     [][]string        `json:"constraints,omitempty"`
	Container       *Container        `json:"container,omitempty"`
	CPUs            float64           `json:"cpus,omitempty"`
	Dependencies    []string          `json:"dependencies,omitempty"`
	Disk            float64           `json:"disk,omitempty"`
	Env             map[string]string `json:"env,omitempty"`
	Executor        string            `json:"executor,omitempty"`
	Labels          map[string]string `json:"labels,omitempty"`
	HealthChecks    []HealthCheck     `json:"healthChecks,omitempty"`
	ID              string            `json:"id,omitempty"`
	Instances       int               `json:"instances,omitempty"`
	Mem             float64           `json:"mem,omitempty"`
	Ports           []int             `json:"ports,omitempty"`
	RequirePorts    bool              `json:"requirePorts,omitempty"`
	StoreUrls       []string          `json:"storeUrls,omitempty"`
	UpgradeStrategy UpgradeStrategy   `json:"upgradeStrategy,omitempty"`
	Uris            []string          `json:"uris,omitempty"`
	User            string            `json:"user,omitempty"`
	Version         string            `json:"version,omitempty"`
}

type AppResponse struct {
	Apps []*App `json:"apps"`
}

func NewMarathon(location, protocol string, username string, password string, verifySsl bool) (*Marathon, error) {
	return &Marathon{location, protocol, username, password, verifySsl}, nil
}

func (m Marathon) ToApp(appJson string) (*App, error) {
	app := &App{}
	err := json.Unmarshal([]byte(appJson), app)
	return app, err
}

func (m Marathon) Apps() ([]*App, error) {
	response, err := m.get("/v2/apps")

	if err != nil || (response.StatusCode != 200) {
		return nil, err
	}

	body, err := m.responseBody(response)
	if err != nil {
		return nil, err
	}

	apps := &AppResponse{}
	err = json.Unmarshal(body, apps)
	return apps.Apps, err
}

func (m Marathon) CreateApp(app *App) (string, error) {
	jsonBlob, err := json.Marshal(app)
	if err != nil {
		return "", nil
	}

	log.Debugf("app json: %s", string(jsonBlob))

	response, err := m.post("/v2/apps", jsonBlob)

	if err != nil {
		return "", err
	}

	switch response.StatusCode {
	case 409:
		return "", errors.New("409 Conflict - application already exists")
	default:
		return m.responseText(response)
	}
}

func (m Marathon) DestroyApp(appId string) (string, error) {
	response, err := m.delete("/v2/apps" + appId)
	if err != nil || (response.StatusCode != 200) {
		return "", err
	}

	return m.responseText(response)
}

func (m Marathon) get(url string) (*http.Response, error) {
	return m.doRequest("GET", url, nil)
}

func (m Marathon) delete(url string) (*http.Response, error) {
	return m.doRequest("DELETE", url, nil)
}

func (m Marathon) post(url string, data []byte) (*http.Response, error) {
	return m.doRequest("POST", url, data)
}

func (m Marathon) doRequest(method string, path string, data []byte) (*http.Response, error) {
	url := m.url(path)
	client := m.getClient()

	var buf io.Reader
	if len(data) > 0 {
		buf = bytes.NewBuffer(data)
	}

	log.Debugf("%s %s", method, url)
	request, err := http.NewRequest(method, url, buf)
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Accept", "application/json")

	if m.Username != "" && m.Password != "" {
		request.SetBasicAuth(m.Username, m.Password)
	}

	if err != nil {
		log.WithFields(log.Fields{
			"method": method,
			"url":    url,
		}).Error(err)
		return nil, err
	}

	response, err := client.Do(request)
	m.logHTTP(response, method, url, err)

	return response, err
}

func (m Marathon) getClient() *http.Client {
	client := &http.Client{}
	client.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: m.NoVerifySsl,
		},
	}

	return client
}

func (m Marathon) url(path string) string {
	marathon := url.URL{
		Scheme: m.Protocol,
		Host:   m.Location,
		Path:   path,
	}

	return marathon.String()
}

func (m Marathon) responseBody(response *http.Response) ([]byte, error) {
	if response == nil {
		return nil, nil
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Errorf("Could not read response body: %v", err)
		return nil, err
	}

	return body, nil
}

func (m Marathon) responseText(response *http.Response) (string, error) {
	responseText := ""

	body, err := m.responseBody(response)
	if err != nil {
		return responseText, err
	}

	if len(body) > 0 {
		responseText = string(body)
	}

	return responseText, nil
}

func (m Marathon) logHTTP(resp *http.Response, method string, url string, err error) {
	fields := log.Fields{
		"url":    url,
		"method": method,
	}

	if resp != nil {
		fields["status"] = resp.Status
		fields["statusCode"] = resp.StatusCode
	}

	if err != nil {
		log.WithFields(fields).Error(err.Error())
	} else {
		log.WithFields(fields).Debugf("%s %s", method, url)
	}
}
