package mesos

import (
	"bufio"
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
	"os"
	"regexp"
	"strings"
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

type StateResponse struct {
	CompletedFrameworks    []*Framework `json:"completed_frameworks"`
	Frameworks             []*Framework `json:"frameworks"`
	UnregisteredFrameworks []*Framework `json:"unregistered_frameworks"`
}

func (s *StateResponse) AllFrameworks() []*Framework {
	return append(s.Frameworks, append(s.CompletedFrameworks, s.UnregisteredFrameworks...)...)
}

func NewMesos(location, protocol string, principal string, secret string, verifySsl bool) (*Mesos, error) {
	return &Mesos{location, protocol, principal, secret, verifySsl}, nil
}

func (m Mesos) State() (*StateResponse, error) {
	response, err := m.get("/master/state.json")
	if err != nil {
		return nil, err
	}

	body, err := m.responseBody(response)
	if err != nil {
		return nil, err
	}

	state := &StateResponse{}
	err = json.Unmarshal(body, state)
	return state, err
}

func (m Mesos) Shutdown(frameworkId string) (string, error) {
	data := fmt.Sprintf("frameworkId=%s", frameworkId)
	response, err := m.post("/master/teardown", []byte(data))
	if err != nil {
		return "", err
	}
	return m.responseText(response)
}

func (m Mesos) ShutdownFrameworkByName(name string) (string, error) {
	// find mesos framework
	state, _ := m.State()
	matchingFrameworks := make(map[string]*Framework)
	for _, fw := range state.Frameworks {
		if fw.Name == name && fw.Active {
			matchingFrameworks[fw.ID] = fw
		}
	}

	if fwCount := len(matchingFrameworks); fwCount > 1 {
		return "", errors.New(fmt.Sprintf("There are %d %s frameworks.", fwCount, name))
	}

	var frameworkId string
	for fwid, _ := range matchingFrameworks {
		frameworkId = fwid
		break
	}

	// shutdown mesos framework
	return m.Shutdown(frameworkId)
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

func parseCredentials(filePath string) (map[string]string, error) {
	credentials := make(map[string]string)

	file, err := os.Open(filePath)
	if err != nil {
		log.Errorf("Could not open credentials file: %v", err)
		return nil, err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		parts := regexp.MustCompile("\\s+").Split(line, -1)
		if len(parts) == 2 {
			principal := strings.TrimSpace(parts[0])
			secret := strings.TrimSpace(parts[1])
			credentials[principal] = secret
		}

	}

	return credentials, nil
}
