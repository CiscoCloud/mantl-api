package zookeeper

import (
	log "github.com/Sirupsen/logrus"
	"github.com/samuel/go-zookeeper/zk"
	"path"
	"sort"
	"strings"
	"time"
)

type Zookeeper struct {
	Servers []string
}

func NewZookeeper(servers []string) *Zookeeper {
	return &Zookeeper{servers}
}

func (z *Zookeeper) Delete(keyPath string) error {
	conn, err := z.connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	keyPath = strings.TrimPrefix(keyPath, "zk:")

	return z.deleteTree(conn, keyPath)
}

func (z *Zookeeper) connect() (*zk.Conn, error) {
	conn, _, err := zk.Connect(z.Servers, time.Second*10)
	return conn, err
}

func (z *Zookeeper) deleteTree(conn *zk.Conn, keyPath string) error {
	result, err := z.znodeTree(conn, keyPath, "")
	if err != nil {
		log.Errorf("Could not retrieve znode tree: %v", err)
		return err
	}

	for i := len(result) - 1; i >= 0; i-- {
		znode := keyPath + "/" + result[i]
		if err = z.deleteNode(conn, znode); err != nil {
			log.Warnf("Could not delete %s from zookeeper: %v", znode, err)
		}
	}

	err = z.deleteNode(conn, keyPath)
	if err != nil {
		log.Warnf("Could not delete %s from zookeeper: %v", keyPath, err)
	}
	return err
}

func (z *Zookeeper) znodeTree(conn *zk.Conn, keyPath string, nextKeyPath string) ([]string, error) {
	children, _, err := conn.Children(keyPath)
	if err != nil {
		return children, err
	}
	sort.Sort(sort.StringSlice(children))
	nested := []string{}
	for _, child := range children {
		nextChild := path.Join(nextKeyPath, child)
		nested = append(nested, nextChild)
		nestedChildren, err := z.znodeTree(conn, path.Join(keyPath, child), nextChild)
		if err != nil {
			return children, err
		}
		nested = append(nested, nestedChildren...)
	}
	return nested, err
}

func (z *Zookeeper) deleteNode(conn *zk.Conn, keyPath string) error {
	log.Debugf("Deleting zk://%s", keyPath)
	return conn.Delete(keyPath, -1)
}
