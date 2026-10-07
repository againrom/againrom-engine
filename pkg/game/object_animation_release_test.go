package game

import (
	"strings"
	"testing"
)

// TestReleaseForesterStaticObjectsExposeEveryDrawableCycle opens Forester
// through the production map-list route over a lawful install. The exact
// population distinguishes the live game route from the old tile-bit gate,
// which admitted none of these cycles.
func TestReleaseForesterStaticObjectsExposeEveryDrawableCycle(t *testing.T) {
	f := releaseFront(t)
	row := -1
	for i, entry := range f.Maps {
		if entry.Mission == 0 && !entry.FromArchive && strings.EqualFold(entry.Source, "Forester.alm") {
			row = i
			break
		}
	}
	if row < 0 {
		t.Fatal("map list has no loose Forester.alm row")
	}

	v, _, _, _, _, _, _, _, _, _, err := f.loadMap(row)
	if err != nil {
		t.Fatalf("open Forester through production row %d: %v", row, err)
	}
	placements, counts := v.Statics()
	// The 49 raw bit13 objects stay alive after the light parser clears that
	// collision flag. Runtime fire, rather than file impassability, selects death.
	if placements != 8604 || counts.Placed != 8604 || counts.Animated != 8349 {
		t.Fatalf("Forester statics: placements=%d counts=%+v, want 8604 placed and 8349 live cycle candidates",
			placements, counts)
	}
	if _, dead := v.ScorchedScenery(); dead != 0 {
		t.Fatalf("Forester false burned objects = %d, want 0", dead)
	}
	if counts.NoClass != 0 || counts.NoFrame != 0 {
		t.Errorf("Forester statics skipped installed content: %+v", counts)
	}
}
