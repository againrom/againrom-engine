// Command worldmapcheck drives the production town-to-world-map-to-mission
// path against a lawful install. It writes no asset and opens no window.
package main

import (
	"flag"
	"fmt"
	"os"

	"againrom/pkg/game"
)

func main() {
	assets := flag.String("assets", os.Getenv("AGAINROM_ASSETS"), "lawful game install root (or AGAINROM_ASSETS)")
	flag.Parse()
	if *assets == "" {
		fmt.Fprintln(os.Stderr, "worldmapcheck: -assets or AGAINROM_ASSETS is required")
		os.Exit(2)
	}
	f, err := game.NewFrontEnd(*assets)
	if err != nil {
		fmt.Fprintln(os.Stderr, "worldmapcheck:", err)
		os.Exit(1)
	}
	line, err := f.WorldMapWitness()
	if err != nil {
		fmt.Fprintln(os.Stderr, "worldmapcheck:", err)
		os.Exit(1)
	}
	fmt.Println(line)
	digest, err := f.WorldMapGraphWitness()
	if err != nil {
		fmt.Fprintln(os.Stderr, "worldmapcheck:", err)
		os.Exit(1)
	}
	fmt.Println(digest)
	criterion, err := f.WorldMapCriterionWitness()
	if err != nil {
		fmt.Fprintln(os.Stderr, "worldmapcheck:", err)
		os.Exit(1)
	}
	fmt.Println(criterion)
	sweep, err := f.WorldMapSweep()
	if err != nil {
		fmt.Fprintln(os.Stderr, "worldmapcheck:", err)
		os.Exit(1)
	}
	fmt.Println(sweep)
}
