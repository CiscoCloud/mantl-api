package marathon

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/CiscoCloud/mantl-api/utils/http"
	log "github.com/Sirupsen/logrus"
)

type Marathon struct {
	httpClient *http.HttpClient
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

func NewMarathon(location, protocol string, username string, password string, noVerifySsl bool) *Marathon {
	return &Marathon{
		httpClient: &http.HttpClient{
			Location:    location,
			Protocol:    protocol,
			Username:    username,
			Password:    password,
			NoVerifySsl: noVerifySsl,
		},
	}
}

func (m Marathon) ToApp(appJson string) (*App, error) {
	app := &App{}
	err := json.Unmarshal([]byte(appJson), app)
	return app, err
}

func (m Marathon) Apps() ([]*App, error) {
	httpReq, err := m.httpClient.Get("/v2/apps")

	if err != nil || (httpReq.Response.StatusCode != 200) {
		return nil, err
	}

	body := httpReq.ResponseBody
	apps := &AppResponse{}
	err = json.Unmarshal(body, apps)
	return apps.Apps, err
}

func (m Marathon) CreateApp(app *App) (string, error) {
	jsonBlob, err := json.Marshal(app)
	if err != nil {
		return "", err
	}

	log.Debugf("app json: %s", string(jsonBlob))

	httpReq, err := m.httpClient.Post("/v2/apps", jsonBlob)

	if err != nil {
		return "", err
	}

	switch httpReq.Response.StatusCode {
	case 409:
		return "", errors.New("409 Conflict - application already exists")
	default:
		return httpReq.ResponseText, nil
	}
}

func (m Marathon) DestroyApp(appId string) (string, error) {
	httpReq, err := m.httpClient.Delete("/v2/apps" + appId)
	if err != nil {
		return "", err
	}

	responseText := httpReq.ResponseText
	if httpReq.Response.StatusCode != 200 {
		return responseText, errors.New(fmt.Sprintf("Failed deleting %s from marathon: %s", appId, responseText))
	}

	return responseText, nil
}
