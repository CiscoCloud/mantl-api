package install

import (
	"github.com/CiscoCloud/mantl-api/marathon"
	"github.com/stretchr/testify/assert"
	"testing"
)

var apps = []*marathon.App{
	&marathon.App{
		ID: "/example",
		Labels: map[string]string{
			packageNameKey: "example",
		},
	},
	&marathon.App{
		ID: "/example2",
		Labels: map[string]string{
			packageNameKey: "example",
		},
	},
	&marathon.App{
		ID: "/kafka",
		Labels: map[string]string{
			packageNameKey: "kafka",
		},
	},
	&marathon.App{
		ID: "/notapackage",
	},
}

func assertFiltered(t *testing.T, expected []string, apps []*marathon.App) {
	if !assert.Equal(t, len(expected), len(apps)) {
		return
	}

	ids := make([]string, len(apps))
	for i, app := range apps {
		ids[i] = app.ID
	}

	assert.Equal(t, expected, ids)
}

func TestFilterPackages(t *testing.T) {
	t.Parallel()
	results := filterPackages(apps)
	assertFiltered(t, []string{"/example", "/example2", "/kafka"}, results)
}

func TestFilterByPackageName(t *testing.T) {
	t.Parallel()
	results := filterByPackageName("example", apps)
	assertFiltered(t, []string{"/example", "/example2"}, results)
}

func TestFilterByID(t *testing.T) {
	t.Parallel()
	results := filterByID("/example", apps)
	assertFiltered(t, []string{"/example"}, results)
}

func TestFilterByIDNoSlash(t *testing.T) {
	t.Parallel()
	results := filterByID("example", apps)
	assertFiltered(t, []string{"/example"}, results)
}
