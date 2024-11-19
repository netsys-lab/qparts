package qpscion

import (
	"context"
	"hash/fnv"
	"sort"
	"strings"

	"github.com/scionproto/scion/pkg/addr"
	"github.com/scionproto/scion/pkg/snet"
)

type QPartsPath struct {
	Internal    snet.Path
	Interfaces  []string
	Id          uint64
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

func stringToUint64(input string) uint64 {
	// Create a new FNV-1a 64-bit hash function
	hasher := fnv.New64a()

	// Write the input string to the hash function
	hasher.Write([]byte(input))

	// Get the uint64 hash value
	return hasher.Sum64()
}

func QueryPaths(remote addr.IA) ([]QPartsPath, error) {
	hc := host()
	paths, err := hc.queryPaths(context.Background(), remote)
	if err != nil {
		return nil, err
	}

	PartsPaths := make([]QPartsPath, 0)
	for _, path := range paths {

		// Print interfaces
		// Log.Info("Interfaces:")
		//for _, intf := range path.Metadata().Interfaces {
		//	Log.Info(intf)
		//}

		pathSorter := IdFromSnetPath(path)
		id := stringToUint64(pathSorter)

		PartsPath := QPartsPath{
			Internal:    path,
			Interfaces:  InterfacesToIds(path.Metadata().Interfaces),
			Id:          id,   // IdFromSnetPath(path),
			MTU:         1472, // TODO: MTU Discovery
			Sorter:      pathSorter,
			PayloadSize: 1200, // TODO: Payloadsize
		}
		PartsPath.Sorter = strings.Join(PartsPath.Interfaces, ",")
		PartsPaths = append(PartsPaths, PartsPath)
	}

	SortPartsPaths(PartsPaths)

	if len(PartsPaths) == 1 {
		PartsPaths = append(PartsPaths, PartsPaths[0])
		PartsPaths = append(PartsPaths, PartsPaths[0])
	}

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
func SortPartsPaths(paths []QPartsPath) {
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
func SortPartsPathsP(paths []*QPartsPath) {
	sort.Slice(paths, func(i, j int) bool {
		// First compare the length of the Sorter strings
		if len(paths[i].Sorter) != len(paths[j].Sorter) {
			return len(paths[i].Sorter) < len(paths[j].Sorter)
		}
		// If lengths are equal, sort alphabetically
		return paths[i].Sorter < paths[j].Sorter
	})
}
