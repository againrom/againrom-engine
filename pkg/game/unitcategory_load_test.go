package game_test

import (
	"os"
	"reflect"
	"sort"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/game"
	"againrom/pkg/render/terrain"
)

func TestLoadUnitsCarriesInheritedZWithoutChangingTheCanvas(t *testing.T) {
	node := func(name string, value int32) synth.RegNode { return synth.RegNode{Name: name, Kind: 2, Int: value} }
	raw := synth.UnitsReg(unitFiles,
		[]synth.RegNode{node("ID", 3), node("Z", 96), node("Width", 48), node("Height", 56)},
		[]synth.RegNode{node("ID", 7), node("Parent", 3)},
		[]synth.RegNode{node("ID", 9)})
	set, err := game.LoadUnits(openContainers(t, synth.Archive([]synth.File{{Path: graphicsEntry(t, game.UnitRegistry), Data: raw}})))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []int32{3, 7} {
		if c := set.Classes[id]; c.Z != 96 || c.Width != 48 || c.Height != 56 {
			t.Fatalf("class%d did not retain Z separately from geometry: %+v", id, c)
		}
	}
	if got := set.Classes[9].Z; got != 0 {
		t.Fatalf("absent Z=%d want0", got)
	}
}

func TestReleaseUnitDrawCategoriesRetainInstalledZ(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: installed unit registry required")
	}
	a, err := game.OpenArchives(root)
	if err != nil {
		t.Fatal(err)
	}
	set, err := game.LoadUnits(a.Containers)
	if err != nil {
		t.Fatal(err)
	}
	var air []int32
	for id, class := range set.Classes {
		if class.Z == 0 {
			continue
		}
		air = append(air, id)
		if class.Z != 96 || terrain.UnitCategoryFor(class, 0) != terrain.UnitAir || terrain.UnitCategoryFor(class, 4) != terrain.UnitAir {
			t.Fatalf("installed class%d: Z/category lost", id)
		}
	}
	sort.Slice(air, func(i, j int) bool { return air[i] < air[j] })
	if !reflect.DeepEqual(air, []int32{70, 71}) {
		t.Fatalf("installed nonzero Z classes=%v want70,71 (REG-UNITS-061)", air)
	}
	t.Logf("installed classes=%d; air class IDs=%v; Z=96 preserved as category, no pixel lift", len(set.Classes), air)
}
