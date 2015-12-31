package mesos

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/CiscoCloud/mantl-api/utils/http"
	log "github.com/Sirupsen/logrus"
	"strconv"
)

type Mesos struct {
	Principal  string
	Secret     string
	httpClient *http.HttpClient
}

type Framework struct {
	Name   string `json:"name"`
	ID     string `json:"id"`
	Active bool   `json:"active"`
}

type State struct {
	CompletedFrameworks    []*Framework `json:"completed_frameworks"`
	Frameworks             []*Framework `json:"frameworks"`
	UnregisteredFrameworks []*Framework `json:"unregistered_frameworks"`
	Flags                  Flags        `json:"flags"`
}

type Flags struct {
	Authenticate       string `json:"authenticate"`
	AuthenticateSlaves string `json:"authenticate_slaves"`
}

func NewMesos(url string, principal string, secret string, noVerifySsl bool) (*Mesos, error) {
	httpClient, err := http.NewHttpClient(url, principal, secret, noVerifySsl)

	if err != nil {
		return nil, err
	}

	return &Mesos{
		Principal:  principal,
		Secret:     secret,
		httpClient: httpClient,
	}, nil
}

func (m Mesos) Frameworks() ([]*Framework, error) {
	state, err := m.state()
	if err != nil {
		return []*Framework{}, err
	}

	return state.Frameworks, nil
}

func (m Mesos) Shutdown(frameworkId string) error {
	log.Debugf("Shutting down framework: %s", frameworkId)
	data := fmt.Sprintf("frameworkId=%s", frameworkId)
	httpReq, err := m.httpClient.Post("/master/teardown/", []byte(data))
	if err != nil {
		return err
	}
	if httpReq.Response.StatusCode == 200 {
		return nil
	} else {
		responseText := httpReq.ResponseText
		return errors.New(fmt.Sprintf("Could not shutdown framework %s: %d %s", frameworkId, httpReq.Response.StatusCode, responseText))
	}
}

func (m Mesos) ShutdownFrameworkByName(name string) error {
	log.Debugf("Looking for %s framework", name)

	// find mesos framework
	fw, err := m.FindFramework(name)
	if err != nil {
		return err
	}

	if fw == nil {
		log.Debugf("Framework %s not active", name)
		return nil
	}

	// shutdown mesos framework
	return m.Shutdown(fw.ID)
}

func (m Mesos) FindFrameworks(name string) ([]*Framework, error) {
	state, err := m.state()
	if err != nil {
		return []*Framework{}, err
	}

	matching := make(map[string]*Framework)
	for _, fw := range state.Frameworks {
		if fw.Name == name && fw.Active {
			matching[fw.ID] = fw
		}
	}

	var uniqueFws []*Framework
	for _, fw := range matching {
		uniqueFws = append(uniqueFws, fw)
	}

	return uniqueFws, nil
}

func (m Mesos) FindFramework(name string) (*Framework, error) {
	fws, err := m.FindFrameworks(name)
	if err != nil {
		return nil, err
	}

	fwCount := len(fws)
	if fwCount == 0 {
		return nil, nil
	} else if fwCount > 1 {
		return nil, errors.New(fmt.Sprintf("There are %d %s frameworks.", fwCount, name))
	}

	return fws[0], nil
}

func (m Mesos) RequiresAuthentication() (bool, error) {
	state, err := m.state()
	if err != nil {
		return false, err
	}

	b, err := strconv.ParseBool(state.Flags.Authenticate)
	if err != nil {
		return false, err
	}

	return b, nil
}

func (m Mesos) state() (*State, error) {
	httpReq, err := m.httpClient.Get("/master/state.json")
	if err != nil {
		return nil, err
	}

	body := httpReq.ResponseBody
	if err != nil {
		return nil, err
	}

	state := &State{}
	err = json.Unmarshal(body, state)
	return state, err
}
