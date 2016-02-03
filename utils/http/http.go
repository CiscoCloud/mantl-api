package http

import (
	"bytes"
	"crypto/tls"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io"
	"io/ioutil"
	h "net/http"
	"net/url"
	gpath "path"
	"strings"
)

type HttpClient struct {
	Location    string
	Protocol    string
	Path        string
	Username    string
	Password    string
	NoVerifySsl bool
}

type HttpRequest struct {
	Request      *h.Request
	Response     *h.Response
	ResponseText string
	ResponseBody []byte
}

func ParseUrl(u string) (scheme string, host string, path string, err error) {
	if !strings.HasPrefix(u, "http") {
		u = fmt.Sprintf("http://%s", u)
	}

	url, err := url.Parse(u)
	if err != nil {
		return "", "", "", err
	}

	return url.Scheme, url.Host, url.Path, nil
}

func NewHttpClient(url string, user string, pw string, noVerifySsl bool) (*HttpClient, error) {
	protocol, location, path, err := ParseUrl(url)
	if err != nil {
		return nil, err
	}
	return &HttpClient{location, protocol, path, user, pw, noVerifySsl}, nil
}

func (c HttpClient) Get(url string) (*HttpRequest, error) {
	return c.doRequest("GET", url, nil)
}

func (c HttpClient) Delete(url string) (*HttpRequest, error) {
	return c.doRequest("DELETE", url, nil)
}

func (c HttpClient) Post(url string, data []byte) (*HttpRequest, error) {
	return c.doRequest("POST", url, data)
}

func (c HttpClient) doRequest(method string, path string, data []byte) (*HttpRequest, error) {
	url := c.url(path)
	client := c.getClient()

	var buf io.Reader
	if len(data) > 0 {
		buf = bytes.NewBuffer(data)
	}

	log.Debugf("%s %s", method, url)
	request, err := h.NewRequest(method, url, buf)
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Accept", "application/json")

	if c.Username != "" && c.Password != "" {
		request.SetBasicAuth(c.Username, c.Password)
	}

	if err != nil {
		log.WithFields(log.Fields{
			"method": method,
			"url":    url,
		}).Error(err)
		return nil, err
	}

	httpReq := &HttpRequest{
		Request:      request,
		ResponseBody: []byte{},
	}

	response, err := client.Do(request)
	httpReq.Response = response

	if err != nil {
		return httpReq, err
	}

	responseBody, err := responseBody(response)
	httpReq.ResponseBody = responseBody
	if len(responseBody) > 0 {
		httpReq.ResponseText = string(responseBody)
	}

	return httpReq, err
}

func (c HttpClient) getClient() *h.Client {
	client := &h.Client{}
	client.Transport = &h.Transport{
		Proxy: h.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: c.NoVerifySsl,
		},
	}

	return client
}

func (c HttpClient) url(path string) string {
	urlPath := joinPaths(c.Path, path)
	u := url.URL{
		Scheme: c.Protocol,
		Host:   c.Location,
		Path:   urlPath,
	}

	return u.String()
}

func joinPaths(paths ...string) string {
	last := paths[len(paths)-1]
	joined := gpath.Join(paths...)

	if strings.HasSuffix(last, "/") && !strings.HasSuffix(joined, "/") {
		joined += "/"
	}

	return joined
}

func responseBody(response *h.Response) ([]byte, error) {
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

func logHTTP(resp *h.Response, method string, url string, err error) {
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
