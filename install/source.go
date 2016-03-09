package install

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"

	log "github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
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

func (s Source) IsValid() bool {
	return (s.Name != "" && s.Path != "")
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
		var err error
		var relkey string
		if isSourceArtifact(filePath) {
			var data []byte
			relkey, err = filepath.Rel(sourcePath, filePath)
			if err == nil {
				data, err = ioutil.ReadFile(filePath)
				if err == nil {
					key := path.Join(source.rootKey(), relkey)
					err = install.addSourceArtifact(key, data)
				} else {
					log.Errorf("Could not read file %v: %v", filePath, err)
				}
			}
		}
		return err
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
	// only clone a single commit from master, rather than all branches
	gitArgs := []string{
		"clone", source.Path, dest,
		"--single-branch", "--depth", "1",
	}

	if source.Branch != "" {
		gitArgs = append(gitArgs, "--branch", source.Branch)
	}

	log.Debugf("Running git with args: %v", gitArgs)
	err = exec.Command("git", gitArgs...).Run()
	if err != nil {
		return err
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

func (install *Install) addSourceArtifact(key string, data []byte) error {
	kp := &consul.KVPair{Key: key, Value: data}
	_, err := install.kv.Put(kp, nil)
	if err == nil {
		log.Debugf("Wrote %v", key)
	}
	return err
}

func isSourceArtifact(filePath string) bool {
	b := path.Ext(filePath) == ".json"
	if !b {
		log.Debugf("Skipping %v", filePath)
	}
	return b
}
