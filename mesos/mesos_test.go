package mesos

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

var mesosCredentials = `
marathon marathons$cr$T!
  chronos chronos+123-secret
follower follow3rS3cr$t    
mantl-install   mantl-install(secret)
`

func TestParseCredentials(t *testing.T) {
	t.Parallel()

	temp, _ := ioutil.TempFile(os.TempDir(), "mantl-api-mesos-test")
	defer os.Remove(temp.Name())

	m := DefaultConfig()
	m.Credentials = temp.Name()

	temp.WriteString(mesosCredentials)
	temp.Close()
	t.Logf(temp.Name())

	creds, _ := m.ParseCredentials()
	assert.Equal(t, "marathons$cr$T!", creds["marathon"])
	assert.Equal(t, "chronos+123-secret", creds["chronos"])
	assert.Equal(t, "follow3rS3cr$t", creds["follower"])
	assert.Equal(t, "mantl-install(secret)", creds["mantl-install"])
}
