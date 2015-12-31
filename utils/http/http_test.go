package http

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseUrl(t *testing.T) {
	tests := [][]string{
		{"http://localhost:8500", "http", "localhost:8500", ""},
		{"localhost:8500", "http", "localhost:8500", ""},
		{"https://localhost:8500", "https", "localhost:8500", ""},
		{"http://localhost", "http", "localhost", ""},
		{"http://localhost/mesos", "http", "localhost", "/mesos"},
	}

	for _, testCase := range tests {
		url := testCase[0]
		expectedScheme := testCase[1]
		expectedHost := testCase[2]
		expectedPath := testCase[3]

		scheme, host, path, _ := ParseUrl(url)
		assert.Equal(t, expectedScheme, scheme)
		assert.Equal(t, expectedHost, host)
		assert.Equal(t, expectedPath, path)
	}
}

func TestJoinPath(t *testing.T) {
	tests := [][]string{
		{"/mesos/master/state.json", "/mesos/", "master/state.json"},
		{"/mesos/master/state.json", "/mesos/", "/master/state.json"},
		{"/mesos/master/teardown/", "/mesos/", "/master/teardown/"},
		{"/mesos/master/teardown/", "/mesos/", "master/teardown/"},
		{"/mesos/master/teardown", "/mesos/", "master/teardown"},
	}

	for _, testCase := range tests {
		joinedPath := joinPaths(testCase[1], testCase[2])
		assert.Equal(t, testCase[0], joinedPath)
	}
}
