package source

import (
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
	"github.com/libgit2/git2go"
	"github.com/ryane/mantl-api/repository"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
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
	Index      int
}

func (source *Source) Sync(client *consul.Client) error {
	switch source.SourceType {
	case FileSystem:
		return sync(source, source.Path, client)
	case Git:
		return syncGitSource(source, client)
	}
	return errors.New("Unknown source type")
}

func (source *Source) LastUpdated(client *consul.Client) (time.Time, error) {
	kv := client.KV()
	kp, _, err := kv.Get(sourceTimestampKey(source), nil)
	if err != nil || kp == nil {
		return time.Time{}, err
	}

	ts, err := time.Parse(time.UnixDate, string(kp.Value))
	if err != nil {
		return time.Time{}, err
	}

	return ts, nil
}

func sync(source *Source, sourcePath string, client *consul.Client) error {
	kv := client.KV()
	// TODO: lock or something to prevent simultaneous syncs?

	err := filepath.Walk(sourcePath, func(filePath string, f os.FileInfo, e error) error {
		if isSourceArtifact(filePath) {
			relkey, err := filepath.Rel(sourcePath, filePath)
			if err == nil {
				data, err := ioutil.ReadFile(filePath)
				if err == nil {
					key := path.Join(source.rootKey(), relkey)
					addSourceArtifact(kv, key, data)
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

	err = setName(source, kv)
	if err != nil {
		log.Errorf("Could not write %s name: %v", source.Name, err)
		return err
	}

	err = setTimestamp(source, kv)
	if err != nil {
		log.Errorf("Could not write %s timestamp: %v", source.Name, err)
		return err
	}

	return nil
}

func syncGitSource(source *Source, client *consul.Client) error {
	temp, err := ioutil.TempDir(os.TempDir(), "mantl-install")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)

	dest := path.Join(temp, source.Name)
	log.Debugf("Cloning %s into %s", source.Path, dest)
	_, err = git.Clone(source.Path, dest, &git.CloneOptions{})
	if err != nil {
		return err
	}

	os.RemoveAll(path.Join(dest, ".git"))

	return sync(source, dest, client)
}

func (source *Source) rootKey() string {
	return path.Join(repository.RepositoryRoot, fmt.Sprintf("%d", source.Index))
}

func sourceTimestampKey(source *Source) string {
	return path.Join(source.rootKey(), "updated")
}

func setName(source *Source, kv *consul.KV) error {
	key := path.Join(source.rootKey(), "name")
	_, err := kv.Put(&consul.KVPair{Key: key, Value: []byte(source.Name)}, nil)
	return err
}

func setTimestamp(source *Source, kv *consul.KV) error {
	ts := time.Now().UTC().Format(time.UnixDate)
	_, err := kv.Put(&consul.KVPair{Key: sourceTimestampKey(source), Value: []byte(ts)}, nil)
	return err
}

func addSourceArtifact(kv *consul.KV, key string, data []byte) {
	kp := &consul.KVPair{Key: key, Value: data}
	_, err := kv.Put(kp, nil)
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
