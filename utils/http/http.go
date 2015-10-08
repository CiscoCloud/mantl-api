package http

import (
	"bytes"
	"crypto/tls"
	log "github.com/Sirupsen/logrus"
	"io"
	"io/ioutil"
	h "net/http"
	"net/url"
)

type HttpClient struct {
	Location    string
	Protocol    string
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

func NewHttpClient(location string, protocol string, user string, pw string, noVerifySsl bool) *HttpClient {
	return &HttpClient{location, protocol, user, pw, noVerifySsl}
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

	response, err := client.Do(request)
	logHTTP(response, method, url, err)

	httpReq := &HttpRequest{
		Request:  request,
		Response: response,
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
	u := url.URL{
		Scheme: c.Protocol,
		Host:   c.Location,
		Path:   path,
	}

	return u.String()
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
