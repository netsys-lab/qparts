package qparts

/*
import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"time"
)

type SchedulerSbd struct {
	paths          []int
	congestedPaths []int
	strategy       string
	numPaths       int
	events         []CongestionEvent
	lastSchedTime  time.Time
}

func NewSchedulerSbd() *SchedulerSbd {
	numPathsEnv := os.Getenv("NUM_PATHS")
	res, _ := strconv.Atoi(numPathsEnv)
	return &SchedulerSbd{
		paths:    make([]int, 0),
		strategy: os.Getenv("SCHEDULER_STRATEGY"),
		numPaths: res,
		events:   make([]CongestionEvent, 0),
	}
}

func isInEpsilonRange(x, y, epsilon uint32) bool {
	return y >= x-epsilon && y <= x+epsilon
}

func removeDuplicate[T comparable](sliceList []T) []T {
	allKeys := make(map[T]bool)
	list := []T{}
	for _, item := range sliceList {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

func (sr *SchedulerSbd) OnCongestionEvent(event *CongestionEvent) error {
	Log.Info("Congestion event")

	notDo := os.Getenv("NO_SBD")
	if notDo != "" {
		Log.Info("Skipping SBD")
		return nil
	}

	sr.events = append(sr.events, *event)
	if len(sr.events) > 8 {
		sr.events = sr.events[1:]
	}

	congestedPaths := make([]uint32, 0)

	for i, e := range sr.events {
		if e.PacketLoss > 0 && i < len(sr.events)-1 {
			Log.Info("COMPARING ", e.PacketLoss, " with ", sr.events[i+1].PacketLoss)
			if e.PacketLoss < 90 && sr.events[i+1].PacketLoss < 90 {
				Log.Info("Detected congestion event")
				congestedPaths = append(congestedPaths, e.PathId)
				congestedPaths = append(congestedPaths, sr.events[i+1].PathId)
			}
			// if isInEpsilonRange(e.PacketLoss, sr.events[i+1].PacketLoss, 100) || isInEpsilonRange(sr.events[i+1].PacketLoss, e.PacketLoss, 100) {
			//	Log.Info("Detected congestion event")
			//	congestedPaths = append(congestedPaths, e.PathId)
			//}
		}
	}

	congestedPaths = removeDuplicate(congestedPaths)

	if len(congestedPaths) > 0 {

		diff := time.Since(sr.lastSchedTime)
		Log.Info("Congested paths", congestedPaths)
		s := event.Remote.String()
		paths := State.Remotes[s].Paths

		usedPaths := make([]*PartsPath, 0)
		for _, v := range sr.paths {
			usedPaths = append(usedPaths, paths[v])
		}

		//realCongestedPaths := make([]*PartsPath, 0)
		//for _, v := range congestedPaths {
		//	realCongestedPaths = append(realCongestedPaths, paths[v])
		//}

		affectedInterfaces := sharedBottleneckDetection(usedPaths, congestedPaths)
		Log.Info("Affected interfaces", affectedInterfaces)
		logSBDResult(*event, affectedInterfaces, diff)

		unbottleneckedPaths := filterPathsByBottlenecks(paths, affectedInterfaces)
		Log.Info("Unbottlenecked paths", unbottleneckedPaths)

		//for _, v := range unbottleneckedPaths {
		//	Log.Info(v.Sorter)
		//}

		// Copy the path selection strategy stuff here and continue...
		pathIds := selectPaths(unbottleneckedPaths, sr.numPaths, sr.strategy, State, s)

		sr.paths = make([]int, 0)
		for _, v := range pathIds {
			if v >= len(unbottleneckedPaths) {
				Log.Info("Invalid path index")
				continue
			}
			path := unbottleneckedPaths[v]
			Log.Info("Selected path ", path.Sorter)
			sr.paths = append(sr.paths, int(path.Id))
			sr.events = make([]CongestionEvent, 0)
		}

		// TODO: Fix this later...
		// sr.events = make([]CongestionEvent, 0)

	}

	return nil
}

func logSBDResult(event CongestionEvent, affectedInterfaces []string, dur time.Duration) {

	f, err := os.OpenFile("/opt/sbd.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}

	defer f.Close()
	bottleneck := os.Getenv("BOTTLENECK")
	strategy := os.Getenv("SCHEDULER_STRATEGY")
	numPathsEnv := os.Getenv("NUM_PATHS")
	result := len(affectedInterfaces)
	durationStr := fmt.Sprintf("%dms", dur.Milliseconds())
	text := fmt.Sprintf("Shared Bottleneck Detection Result: %d;Bottleneck: %s;Strategy: %s; NumPaths: %s; Duration: %s\n", result, bottleneck, strategy, numPathsEnv, durationStr)

	if _, err = f.WriteString(text); err != nil {
		panic(err)
	}
}

// filterPathsByBottlenecks filters paths to include only those that contain all interfaces from bottlenecks.
func filterPathsByBottlenecks(paths []*PartsPath, bottlenecks []string) []*PartsPath {
	filteredPaths := make([]*PartsPath, 0)
	// bottleneckedPathAdded := false

	for _, path := range paths {
		if containsAny(path.Interfaces, bottlenecks) {
			//if !bottleneckedPathAdded {
			//	filteredPaths = append(filteredPaths, path)
			//}
			// bottleneckedPathAdded = true
		} else {
			// Keep paths that do not contain all bottleneck interfaces unchanged
			filteredPaths = append(filteredPaths, path)
		}
	}

	return filteredPaths
}

// containsAll checks if slice 'a' contains all elements of slice 'b'
func containsAll(a, b []string) bool {
	set := make(map[string]bool)
	for _, item := range a {
		set[item] = true
	}
	for _, item := range b {
		if !set[item] {
			return false
		}
	}
	return true
}

// containsAll checks if slice 'a' contains all elements of slice 'b'
func containsAny(a, b []string) bool {
	for _, item := range a {
		for _, item2 := range b {
			if item == item2 {
				return true
			}
		}
	}

	return false
}

func selectPaths(allPaths []*PartsPath, count int, strategy string, state *NetworkState, s string) []int {
	newPaths := make([]*PartsPath, len(allPaths))
	copy(newPaths, allPaths)

	numPaths := min(len(newPaths), count)
	// sr.numPaths = numPaths
	retPaths := make([]int, 0)
	switch strategy {
	case "random":
		paths, err := selectRandomPaths(newPaths, numPaths)
		if err != nil {
			log.Fatal("Error selecting random paths")
		}

		for i, v := range state.Remotes[s].Paths {
			for _, p := range paths {
				if v.Id == p.Id {
					retPaths = append(retPaths, i)
				}
			}
		}

	case "shortest":
		paths, err := selectShortestPaths(newPaths, numPaths)
		if err != nil {
			log.Fatal("Error selecting random paths")
		}

		for i, v := range state.Remotes[s].Paths {
			for _, p := range paths {
				if v.Id == p.Id {
					retPaths = append(retPaths, i)
				}
			}
		}
	case "least":
		paths, err := selectLeastDisjointPaths(newPaths, numPaths)
		if err != nil {
			log.Fatal("Error selecting random paths")
		}

		for i, v := range state.Remotes[s].Paths {
			for _, p := range paths {
				if v.Id == p.Id {
					retPaths = append(retPaths, i)
				}
			}
		}
	case "most":
		paths, err := selectMostDisjointPaths(newPaths, numPaths)
		if err != nil {
			log.Fatal("Error selecting random paths")
		}

		for i, v := range state.Remotes[s].Paths {
			for _, p := range paths {
				if v.Id == p.Id {
					retPaths = append(retPaths, i)
				}
			}
		}
	}

	return retPaths
}

func (sr *SchedulerSbd) ScheduleWrite(data []byte, stream *PartsStream, state *NetworkState) SchedulingDecision {

	s := stream.conn.remote.String()

	if len(sr.paths) == 0 {
		sr.paths = selectPaths(state.Remotes[s].Paths, sr.numPaths, sr.strategy, state, s)
	}

	assignments := make([]DataAssignment, len(sr.paths))
	for i, index := range sr.paths {
		Log.Info("Scheduling write for index", index)
		da := DataAssignment{
			Path: state.Remotes[s].Paths[index],
			Data: data,
		}
		assignments[i] = da
	}
	sr.lastSchedTime = time.Now()

	//Log.Info("Got paths available")
	//Log.Info(state.Remotes[s].Paths)

	//for i, p := range state.Remotes[s].Paths {
	//	fmt.Printf("Path %d has id %d\n", i, p.Id)
	//}

	//sr.index++

	//if sr.index >= len(state.Remotes[s].Paths) {
	//		Log.Info("Resetting index")/
	//		sr.index = 0/
	//	}

	return SchedulingDecision{
		Assignments: assignments, // []DataAssignment{da},
	}
}

// sharedBottleneckDetection identifies shared bottlenecks among a subset of paths and checks them against other paths.
func sharedBottleneckDetection(paths []*PartsPath, affectedIds []uint32) []string {
	affected := make(map[uint32]bool)
	for _, id := range affectedIds {
		affected[id] = true
	}

	// Step 1: Build the intersection of interfaces from affected paths
	var intersection []string
	first := true
	for _, path := range paths {
		if affected[path.Id] {
			if first {
				intersection = append(intersection, path.Interfaces...)
				first = false
			} else {
				intersection = intersect(intersection, path.Interfaces)
			}
		}
	}
	Log.Info("INTERSECTION", intersection)

	// Step 2: Remove interfaces from the intersection if they appear in unaffected paths
	for _, path := range paths {
		if !affected[path.Id] {
			Log.Info("BEFORE DIFFERENCE ", path.Id, intersection)
			intersection = difference(intersection, path.Interfaces)
			Log.Info("AFTER DIFFERENCE", intersection)
			Log.Info("--------------")

			if len(intersection) == 2 {
				break
			}
		}
	}

	return intersection
}

// intersect returns the intersection of two slices of strings
func intersect(a, b []string) []string {
	m := make(map[string]bool)
	var intersection []string
	for _, item := range a {
		m[item] = true
	}
	for _, item := range b {
		if m[item] {
			intersection = append(intersection, item)
		}
	}
	return intersection
}

// difference returns the elements in `a` that aren't in `b`
func difference(a, b []string) []string {
	m := make(map[string]bool)
	var diff []string
	for _, item := range b {
		m[item] = true
	}
	for _, item := range a {
		if !m[item] {
			diff = append(diff, item)
		}
	}
	return diff
}

func selectRandomPaths(paths []*PartsPath, n int) ([]*PartsPath, error) {
	if n > len(paths) {
		return nil, errors.New("not enough paths to select from")
	}

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(paths), func(i, j int) { paths[i], paths[j] = paths[j], paths[i] })

	return paths[:n], nil
}

func selectShortestPaths(paths []*PartsPath, n int) ([]*PartsPath, error) {
	if n > len(paths) {
		return nil, errors.New("not enough paths to select from")
	}

	sort.Slice(paths, func(i, j int) bool {
		return len(paths[i].Interfaces) < len(paths[j].Interfaces)
	})

	return paths[:n], nil
}

func selectLeastDisjointPaths(paths []*PartsPath, n int) ([]*PartsPath, error) {
	if n > len(paths) {
		return nil, errors.New("not enough paths to select from")
	}

	// Count intersections with all other paths and sum them up
	disjointScore := make(map[int]int)
	for i := range paths {
		for j := range paths {
			if i != j {
				disjointScore[i] += countIntersect(paths[i].Interfaces, paths[j].Interfaces)
			}
		}
	}

	sort.Slice(paths, func(i, j int) bool {
		return disjointScore[i] < disjointScore[j]
	})

	return paths[:n], nil
}

func selectMostDisjointPaths(paths []*PartsPath, n int) ([]*PartsPath, error) {
	if n > len(paths) {
		return nil, errors.New("not enough paths to select from")
	}

	disjointScore := make(map[int]int)
	for i := range paths {
		for j := range paths {
			if i != j {
				disjointScore[i] += countIntersect(paths[i].Interfaces, paths[j].Interfaces)
			}
		}
	}

	sort.Slice(paths, func(i, j int) bool {
		return disjointScore[i] > disjointScore[j]
	})

	return paths[:n], nil
}

func countIntersect(a, b []string) int {
	intersect := make(map[string]bool)
	for _, v := range a {
		intersect[v] = true
	}
	count := 0
	for _, v := range b {
		if intersect[v] {
			count++
		}
	}
	return count
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
*/
