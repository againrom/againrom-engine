// Command mapunitcensus counts the authored map ids of every unit placement in
// the shipped campaign, on one lawful install.
//
// It reads an install and therefore lives here rather than in a test: golden
// rule 2 keeps a game install out of go test. Run it as
//
//	go run ./cmd/mapunitcensus -assets <root>
//	AGAINROM_ASSETS=<root> go run ./cmd/mapunitcensus
//
// Both forms are accepted, and the flag wins where both are given. A tool that
// reads only one of the two is how a shipped load list came out empty once:
// the caller's habit and the tool's reader have to be the same channel.
//
// and it prints one line per finding plus a summary.
package main

import (
	"flag"
	"fmt"
	"os"

	"againrom/pkg/formats/alm"
	"againrom/pkg/game"
)

// missions is the campaign scenario.res ships: fifteen tens and thirteen side
// maps. It is the same list pipeline/check-milestone.sh censuses, spelt here
// because this tool answers for the same population.
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
		fmt.Fprintln(os.Stderr, "mapunitcensus: pass -assets or set AGAINROM_ASSETS to a lawful install root")
		os.Exit(2)
	}
	ar, err := game.OpenArchives(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mapunitcensus:", err)
		os.Exit(2)
	}

	var total, zeros, dups int
	var maxID uint16
	status := 0
	for _, n := range missions {
		addr, ok := game.MissionMap(n)
		if !ok {
			fmt.Printf("m%d is not a campaign mission number\n", n)
			status = 1
			continue
		}
		b, err := ar.Containers.ReadFile(addr)
		if err != nil {
			fmt.Printf("m%d read: %v\n", n, err)
			status = 1
			continue
		}
		m, err := alm.Open(b)
		if err != nil {
			fmt.Printf("m%d open: %v\n", n, err)
			status = 1
			continue
		}
		// Duplicates are counted PER MAP, because a unit id names a placement
		// within its own map and two maps reusing one id is not a collision.
		seen := map[uint16]int{}
		for _, u := range m.Units {
			total++
			if u.UnitID == 0 {
				zeros++
				fmt.Printf("m%d places a unit at authored id 0 at tile (%d,%d)\n", n, u.X>>8, u.Y>>8)
			}
			if u.UnitID > maxID {
				maxID = u.UnitID
			}
			seen[u.UnitID]++
		}
		for id, k := range seen {
			if k > 1 {
				dups++
				fmt.Printf("m%d authors unit id %d %d times\n", n, id, k)
			}
		}
	}
	fmt.Printf("units=%d zeros=%d duplicate-ids=%d max-id=%d\n", total, zeros, dups, maxID)
	os.Exit(status)
}
