package qparts

import (
	"context"
	"strings"

	"github.com/scionproto/scion/pkg/snet"
)

type NetworkState struct {
	Remotes    map[string]*PathsMetrics
	Interfaces []*InterfaceMetrics
}

type PathsMetrics struct {
	Paths []*PartsPath
}

type InterfaceMetrics struct {
	InterfaceId   string
	MaxThroughput int
}

var State *NetworkState

func init() {
	State = &NetworkState{
		Remotes: make(map[string]*PathsMetrics),
	}
}

func (s *NetworkState) AddRemote(remote *snet.UDPAddr) error {

	str := remote.String()
	// Check if remote is in state
	if _, ok := s.Remotes[str]; !ok {
		pm := &PathsMetrics{}
		s.Remotes[str] = pm
		hc := host()
		paths, err := hc.queryPaths(context.Background(), remote.IA)
		if err != nil {
			return err
		}

		s.Remotes[str].Paths = make([]*PartsPath, 0)
		for id, path := range paths {
			// TODO: Generate path id
			p := &PartsPath{Internal: path, Id: uint32(id), Interfaces: InterfacesToIds(path.Metadata().Interfaces), MTU: 1472, PayloadSize: 1200}
			p.Sorter = strings.Join(p.Interfaces, ",")
			s.Remotes[str].Paths = append(s.Remotes[str].Paths, p)
		}

		SortPartsPathsP(s.Remotes[str].Paths)
	}

	return nil
}
