package archtest

import (
	"os"
	"strings"
	"testing"
)

// placementOccupancyFile holds placementOpen, the fit test placement asks.
const placementOccupancyFile = "pkg/sim/route.go"

// TestPlacementAsksOneOccupancyPredicate: placementOpen holds no occupancy loop.
func TestPlacementAsksOneOccupancyPredicate(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadRandomnessSources(root)
	if err != nil {
		t.Fatal(err)
	}
	src, ok := files[placementOccupancyFile]
	if !ok {
		t.Fatalf("%s is missing; the guard checks nothing", placementOccupancyFile)
	}
	start := strings.Index(src, "func (w *World) placementOpen(")
	if start < 0 {
		t.Fatal("placementOpen is missing; the guard checks nothing")
	}
	body := src[start:]
	if end := strings.Index(body, "\n}\n"); end >= 0 {
		body = body[:end]
	}
	if !strings.Contains(body, "w.occupancyOpenFootprint(") {
		t.Errorf("placementOpen no longer asks occupancyOpenFootprint")
	}
	for _, banned := range []string{"footprintsOverlap(", "range w.entities", "heldCell()"} {
		if strings.Contains(body, banned) {
			t.Errorf("placementOpen spells its own occupancy rule: %s", banned)
		}
	}
}
