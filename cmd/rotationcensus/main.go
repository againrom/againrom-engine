// Command rotationcensus prints the live sim.Entity.RotationSpeed distribution
// over the shipped campaign, read through the production mission door.
//
// cmd/mapunitcensus is the structural precedent: flag/env asset root, a fixed
// mission list, one line per finding, a summary line, non-zero exit on a read
// failure. Its own missions list is copied here unchanged, and its header's
// own words -- "fifteen tens and thirteen side maps" -- are the count this
// tool answers for: 28 missions, not 24. A different number seen in a report
// is a report about a different population than the one this tool reads.
//
//	go run ./cmd/rotationcensus -assets <root>
//	AGAINROM_ASSETS=<root> go run ./cmd/rotationcensus
//
// Both forms are accepted; the flag wins where both are given.
//
// This tool prints a distribution and totals. It renders no verdict: whether
// a given RotationSpeed value is correct is a question for the story's own
// research citations, not for this tool.
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"againrom/pkg/game"
)

// missions is againrom/cmd/mapunitcensus's own list, copied unchanged: the
// campaign scenario.res ships fifteen tens and thirteen side maps, 28 in all.
var missions = []int{
	10, 20, 30, 31, 40, 41, 50, 51, 60, 61, 70, 71, 80, 81, 90, 91,
	100, 101, 110, 111, 120, 121, 130, 131, 140, 141, 150, 151,
}

func main() {
	assets := flag.String("assets", "", "game asset root (default $AGAINROM_ASSETS)")
	flag.Parse()
	root := *assets
	if root == "" {
		root = os.Getenv("AGAINROM_ASSETS")
	}
	if root == "" {
		fmt.Fprintln(os.Stderr, "rotationcensus: pass -assets or set AGAINROM_ASSETS to a lawful install root")
		os.Exit(2)
	}

	front, err := game.NewFrontEnd(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "rotationcensus:", err)
		os.Exit(2)
	}

	dist := map[int32]int{}
	var total, missionsRead, missionsFailed int
	status := 0
	for _, n := range missions {
		open := front.MissionOpenerWith(n, front.NextParty())
		_, _, _, _, _, _, _, _, _, _, err := open()
		if err != nil {
			fmt.Printf("m%d open: %v\n", n, err)
			missionsFailed++
			status = 1
			continue
		}
		w, ok := front.LiveWorld()
		if !ok {
			fmt.Printf("m%d: opener left no live world\n", n)
			missionsFailed++
			status = 1
			continue
		}
		missionsRead++
		ents := w.Entities()
		perMission := map[int32]int{}
		for _, e := range ents {
			dist[e.RotationSpeed]++
			perMission[e.RotationSpeed]++
			total++
		}
		var keys []int32
		for k := range perMission {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
		for _, k := range keys {
			fmt.Printf("m%d: RotationSpeed=%d count=%d\n", n, k, perMission[k])
		}
	}

	var keys []int32
	for k := range dist {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	fmt.Println("---")
	for _, k := range keys {
		fmt.Printf("RotationSpeed=%d total=%d\n", k, dist[k])
	}
	fmt.Printf("missions-read=%d missions-failed=%d entities=%d distinct-values=%d\n",
		missionsRead, missionsFailed, total, len(keys))
	os.Exit(status)
}
