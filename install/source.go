package install

import (
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	FileSystem = iota
	Git
)

type SourceType uint8

type Source struct {
	Name       string
	Path       string
	SourceType SourceType
	Branch     string
	Index      int
}

func (install *Install) syncSource(source *Source) error {
	switch source.SourceType {
	case FileSystem:
		return install.sync(source, source.Path)
	case Git:
		return install.syncGitSource(source)
	}
	return errors.New("Unknown source type")
}

func (install *Install) sourceLastUpdated(source *Source) (time.Time, error) {
	kp, _, err := install.kv.Get(sourceTimestampKey(source), nil)
	if err != nil || kp == nil {
		return time.Time{}, err
	}

	ts, err := time.Parse(time.UnixDate, string(kp.Value))
	if err != nil {
		return time.Time{}, err
	}

	return ts, nil
}

func (install *Install) sync(source *Source, sourcePath string) error {
	// TODO: lock or something to prevent simultaneous syncs?
	err := filepath.Walk(sourcePath, func(filePath string, f os.FileInfo, e error) error {
		if isSourceArtifact(filePath) {
			relkey, err := filepath.Rel(sourcePath, filePath)
			if err == nil {
				data, err := ioutil.ReadFile(filePath)
				if err == nil {
					key := path.Join(source.rootKey(), relkey)
					install.addSourceArtifact(key, data)
				} else {
					log.Errorf("Could not read file %v: %v", filePath, err)
				}
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	err = install.setName(source)
	if err != nil {
		log.Errorf("Could not write %s name: %v", source.Name, err)
		return err
	}

	err = install.setTimestamp(source)
	if err != nil {
		log.Errorf("Could not write %s timestamp: %v", source.Name, err)
		return err
	}

	return nil
}

func (install *Install) syncGitSource(source *Source) error {
	temp, err := ioutil.TempDir(os.TempDir(), "mantl-install")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)

	dest := path.Join(temp, source.Name)
	log.Debugf("Cloning %s into %s", source.Path, dest)
	err = exec.Command("git", "clone", source.Path, dest).Run()
	if err != nil {
		return err
	}

	if source.Branch != "" {
		var branch, remote string

		parts := strings.Split(source.Branch, "/")
		if len(parts) > 1 {
			remote = parts[0]
			branch = parts[1]
		} else {
			remote = "origin"
			branch = parts[0]
		}

		remoteBranch := fmt.Sprintf("%s/%s", remote, branch)

		log.Debugf("Checking out branch %s", remoteBranch)
		cmd := exec.Command("git", "checkout", "-b", branch, remoteBranch)
		cmd.Dir = dest
		err = cmd.Run()
		if err != nil {
			return err
		}
	}

	os.RemoveAll(path.Join(dest, ".git"))

	return install.sync(source, dest)
}

func (source *Source) rootKey() string {
	return path.Join(RepositoryRoot, fmt.Sprintf("%d", source.Index))
}

func sourceTimestampKey(source *Source) string {
	return path.Join(source.rootKey(), "updated")
}

func (install *Install) setName(source *Source) error {
	key := path.Join(source.rootKey(), "name")
	_, err := install.kv.Put(&consul.KVPair{Key: key, Value: []byte(source.Name)}, nil)
	return err
}

func (install *Install) setTimestamp(source *Source) error {
	ts := time.Now().UTC().Format(time.UnixDate)
	_, err := install.kv.Put(&consul.KVPair{Key: sourceTimestampKey(source), Value: []byte(ts)}, nil)
	return err
}

func (install *Install) addSourceArtifact(key string, data []byte) {
	kp := &consul.KVPair{Key: key, Value: data}
	_, err := install.kv.Put(kp, nil)
	if err == nil {
		log.Debugf("Wrote %v", key)
	} else {
		log.Errorf("Could not write %v to KV: %v", key, err)
	}
}

func isSourceArtifact(filePath string) bool {
	b := path.Ext(filePath) == ".json"
	if !b {
		log.Debugf("Skipping %v", filePath)
	}
	return b
}
