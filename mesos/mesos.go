package mesos

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

type Mesos struct {
	Location    string
	Protocol    string
	Principal   string
	Secret      string
	NoVerifySsl bool
}

func DefaultConfig() *Mesos {
	return &Mesos{
		Location:    "localhost:5050",
		Protocol:    "http",
		NoVerifySsl: false,
		Principal:   "mantl-install",
	}
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
}

func NewMesos(location, protocol string, principal string, secret string, verifySsl bool) *Mesos {
	return &Mesos{location, protocol, principal, secret, verifySsl}
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
	response, err := m.post("/master/teardown", []byte(data))
	if err != nil {
		return err
	}
	if response.StatusCode == 200 {
		return nil
	} else {
		responseText, _ := m.responseText(response)
		return errors.New(fmt.Sprintf("Could not shutdown framework %s: %d %s", frameworkId, response.StatusCode, responseText))
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

func (m Mesos) state() (*State, error) {
	response, err := m.get("/master/state.json")
	if err != nil {
		return nil, err
	}

	body, err := m.responseBody(response)
	if err != nil {
		return nil, err
	}

	state := &State{}
	err = json.Unmarshal(body, state)
	return state, err
}

func (m Mesos) get(url string) (*http.Response, error) {
	return m.doRequest("GET", url, nil)
}

func (m Mesos) delete(url string) (*http.Response, error) {
	return m.doRequest("DELETE", url, nil)
}

func (m Mesos) post(url string, data []byte) (*http.Response, error) {
	return m.doRequest("POST", url, data)
}

func (m Mesos) doRequest(method string, path string, data []byte) (*http.Response, error) {
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

	if m.Principal != "" && m.Secret != "" {
		request.SetBasicAuth(m.Principal, m.Secret)
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

func (m Mesos) getClient() *http.Client {
	client := &http.Client{}
	client.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: m.NoVerifySsl,
		},
	}

	return client
}

func (m Mesos) url(path string) string {
	u := url.URL{
		Scheme: m.Protocol,
		Host:   m.Location,
		Path:   path,
	}

	return u.String()
}

func (m Mesos) responseBody(response *http.Response) ([]byte, error) {
	if response == nil {
		return nil, nil
	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Errorf("Could not read response body: %v", err)
		return nil, err
	}

	return body, nil
}

func (m Mesos) responseText(response *http.Response) (string, error) {
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

func (m Mesos) logHTTP(resp *http.Response, method string, url string, err error) {
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
