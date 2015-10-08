package http

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseUrl(t *testing.T) {
	tests := [][]string{
		{"http://localhost:8500", "http", "localhost:8500"},
		{"localhost:8500", "http", "localhost:8500"},
		{"https://localhost:8500", "https", "localhost:8500"},
		{"http://localhost", "http", "localhost"},
	}

	for _, testCase := range tests {
		url := testCase[0]
		expectedScheme := testCase[1]
		expectedHost := testCase[2]

		scheme, host, _ := ParseUrl(url)
		assert.Equal(t, expectedScheme, scheme)
		assert.Equal(t, expectedHost, host)
	}
}
