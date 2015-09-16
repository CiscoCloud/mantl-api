package mesos

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
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
	Location         string
	Protocol         string
	Username         string
	Password         string
	NoVerifySsl      bool
	Credentials      string
	Principal        string
	credentialsCache map[string]string
}

func DefaultConfig() *Mesos {
	return &Mesos{
		Location:    "localhost:5050",
		Protocol:    "http",
		NoVerifySsl: false,
		Credentials: "/etc/mesos/credentials",
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

func NewMesos(location, protocol string, username string, password string, verifySsl bool, credentialsPath string, principal string) (*Mesos, error) {
	creds, err := parseCredentials(credentialsPath)
	if err != nil {
		log.Warnf("Could not read credentials file %s: %v", credentialsPath, err)
	}

	return &Mesos{location, protocol, username, password, verifySsl, credentialsPath, principal, creds}, nil
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
func (m Mesos) GetCredential(principal string) string {
	return m.credentialsCache[principal]
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
