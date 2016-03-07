package install

import (
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
)

const (
	RepositoryRoot = "mantl-install/repository"
	AppsRoot       = "mantl-install/apps/"
)

type Repository struct {
	Name  string
	Index int
}

type RepositoryCollection []*Repository

func (r Repository) PackagesKey() string {
	return path.Join(
		RepositoryRoot,
		fmt.Sprintf("%d", r.Index),
		"repo/packages",
	)
}

func (install *Install) getRepositories() (RepositoryCollection, error) {
	idxs, err := install.repositoryIndexes()
	if err != nil {
		return nil, err
	}

	var repositories RepositoryCollection
	for _, idx := range idxs {
		name, err := install.repositoryName(idx)
		if err != nil {
			log.Warnf("Could not find name for repository %d: %v", idx, err)
			continue
		}

		repositories = append(repositories, &Repository{
			Index: idx,
			Name:  name,
		})
	}

	return repositories, nil
}

func (install *Install) repositoryName(idx int) (string, error) {
	key := path.Join(RepositoryRoot, fmt.Sprintf("%d", idx), "name")
	kp, _, err := install.kv.Get(key, nil)
	if err != nil || kp == nil {
		log.Errorf("Could not retrieve repository name from %s: %v", key, err)
		return "", err
	}

	return string(kp.Value), nil
}

func (install *Install) repositoryIndexes() ([]int, error) {
	// retrieves repository indexes like [0, 1, ...] from mantl-install/repository/
	indexes, _, err := install.kv.Keys(RepositoryRoot+"/", "/", nil)
	if err != nil {
		return nil, err
	}

	var idxs []int
	for _, key := range indexes {
		parts := strings.Split(strings.TrimSuffix(key, "/"), "/")
		sidx := parts[len(parts)-1]
		idx, err := strconv.Atoi(sidx)
		if err != nil {
			log.Warnf("Unexpected repository index at %s: %v", key, err)
			continue
		}
		idxs = append(idxs, idx)
	}

	sort.Sort(sort.IntSlice(idxs))
	return idxs, nil
}
