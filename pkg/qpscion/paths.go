package qpscion

import (
	"context"
	"sort"
	"strings"

	"github.com/scionproto/scion/pkg/addr"
	"github.com/scionproto/scion/pkg/snet"
)

type PartsPath struct {
	Internal    snet.Path
	Interfaces  []string
	Id          uint32
	Sorter      string
	MTU         int
	PayloadSize int
}

func IdFromSnetPath(path snet.Path) string {
	s := ""
	for i, p := range path.Metadata().Interfaces {
		s += p.String()
		if i < len(path.Metadata().Interfaces)-1 {
			s += "-"
		}
	}
	return s
}

func QueryRawPaths(remote addr.IA) ([]snet.Path, error) {
	hc := host()
	return hc.queryPaths(context.Background(), remote)
}

func QueryPaths(remote addr.IA) ([]PartsPath, error) {
	hc := host()
	paths, err := hc.queryPaths(context.Background(), remote)
	if err != nil {
		return nil, err
	}

	PartsPaths := make([]PartsPath, 0)
	for i, path := range paths {

		// Print interfaces
		// Log.Info("Interfaces:")
		//for _, intf := range path.Metadata().Interfaces {
		//	Log.Info(intf)
		//}

		PartsPath := PartsPath{
			Internal:    path,
			Interfaces:  InterfacesToIds(path.Metadata().Interfaces),
			Id:          uint32(i), // IdFromSnetPath(path),
			MTU:         1472,
			PayloadSize: 1400,
		}
		PartsPath.Sorter = strings.Join(PartsPath.Interfaces, ",")
		PartsPaths = append(PartsPaths, PartsPath)
	}

	SortPartsPaths(PartsPaths)
	return PartsPaths, nil
}

func InterfacesToIds(intfs []snet.PathInterface) []string {
	ids := make([]string, 0)
	for _, intf := range intfs {
		it := strings.ReplaceAll(intf.String(), "#", "-")
		ids = append(ids, it)
	}
	return ids
}

// SortPartsPaths sorts a slice of PartsPath structs based on the length of the Sorter string,
// and then alphabetically by the Sorter string.
func SortPartsPaths(paths []PartsPath) {
	sort.Slice(paths, func(i, j int) bool {
		// First compare the length of the Sorter strings
		if len(paths[i].Sorter) != len(paths[j].Sorter) {
			return len(paths[i].Sorter) < len(paths[j].Sorter)
		}
		// If lengths are equal, sort alphabetically
		return paths[i].Sorter < paths[j].Sorter
	})
}

// SortPartsPaths sorts a slice of PartsPath structs based on the length of the Sorter string,
// and then alphabetically by the Sorter string.
func SortPartsPathsP(paths []*PartsPath) {
	sort.Slice(paths, func(i, j int) bool {
		// First compare the length of the Sorter strings
		if len(paths[i].Sorter) != len(paths[j].Sorter) {
			return len(paths[i].Sorter) < len(paths[j].Sorter)
		}
		// If lengths are equal, sort alphabetically
		return paths[i].Sorter < paths[j].Sorter
	})
}
