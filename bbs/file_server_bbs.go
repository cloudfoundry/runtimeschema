package bbs

import (
	"errors"
	"github.com/cloudfoundry/storeadapter"
	"math/rand"
	"path"
)

const FileServerSchemaRoot = SchemaRoot + "file_server"

type fileServerBBS struct {
	store storeadapter.StoreAdapter
}

func fileServerSchemaPath(segments ...string) string {
	return path.Join(append([]string{FileServerSchemaRoot}, segments...)...)
}

func (self *fileServerBBS) MaintainFileServerPresence(heartbeatIntervalInSeconds uint64, fileServerURL string, fileServerId string) (chan bool, chan error, error) {
	key := fileServerSchemaPath(fileServerId)
	return maintainPresence(self.store, key, []byte(fileServerURL), heartbeatIntervalInSeconds)
}

func (self *stagerBBS) GetAvailableFileServer() (string, error) {
	node, err := self.store.ListRecursively(FileServerSchemaRoot)
	if err != nil {
		return "", err
	}

	if len(node.ChildNodes) == 0 {
		return "", errors.New("No file servers are currently available")
	}

	randomServerIndex := rand.Intn(len(node.ChildNodes))
	return string(node.ChildNodes[randomServerIndex].Value), nil
}
