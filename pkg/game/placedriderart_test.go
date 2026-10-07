package game

import (
	"testing"

	"againrom/pkg/render/terrain"
)

func TestRosterBodyArtKeepsARiderOnHisMountedClass(t *testing.T) {
	foot, mounted, hero := &terrain.UnitClass{}, &terrain.UnitClass{}, &terrain.UnitClass{}
	units := &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{4: foot, 21: mounted, 3: hero}}
	if got := rosterBodyArt(units, nil, 21, "heroes", "swordsman_", 4); got != mounted {
		t.Fatalf("a type 21 rider draws the equipment class, want his own class record")
	}
	if got := rosterBodyArt(units, nil, 19, "heroes", "swordsman_", 4); got != foot {
		t.Fatalf("a rider whose class record is absent falls back to the equipment class")
	}
	if got := rosterBodyArt(units, nil, 3, "heroes", "swordsman_", 4); got != foot {
		t.Fatalf("a footman keeps the equipment class")
	}
}
