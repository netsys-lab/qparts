package qpscion

import (
	"fmt"

	"github.com/scionproto/scion/pkg/snet"
)

type PathCache struct {
	Paths map[string][]QPartsPath
}

func NewPathCache() *PathCache {
	return &PathCache{
		Paths: make(map[string][]QPartsPath),
	}
}

func (pc *PathCache) Refresh(remote *snet.UDPAddr) error {
	paths, err := QueryPaths(remote.IA)
	if err != nil {
		return nil
	}

	pc.Paths[remote.String()] = paths
	return nil
}

// TODO: Add expiration time
func (pc *PathCache) Get(remote *snet.UDPAddr) ([]QPartsPath, error) {
	fmt.Println("Getting paths for remote: ", remote)
	if err := pc.Refresh(remote); err != nil {
		return nil, err
	}

	return pc.Paths[remote.String()], nil

}

var Paths *PathCache

func init() {
	Paths = NewPathCache()
}
