//go:build release

package game

import (
	"slices"
	"strconv"
	"testing"

	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

// TestReleaseSecondTestPlayBuildsTheEditedMapHeadless runs test play on an
// installed second-game map edited in the window, then builds what the
// launched game builds from the test mod: the second-game session opens the
// mission the map replaces and its world holds the edited map's units.
func TestReleaseSecondTestPlayBuildsTheEditedMapHeadless(t *testing.T) {
	x, started := f5Inspector(t)
	sources, _ := installedSecondMaps(t, x)
	source := sources[0]
	for _, s := range sources {
		if sourceMission(s) > 0 {
			source = s
			break
		}
	}
	d, err := x.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	host := d.Edits.(*mapInspectionEdits)
	if _, ok := ui.Seam[ui.InspectionTestPlay](d.Edits); !ok {
		t.Fatal("the second-game host offers no test play")
	}
	m, err := host.model.Map()
	if err != nil {
		t.Fatal(err)
	}
	before := len(m.Units)
	if before == 0 {
		t.Fatalf("%s holds no unit to edit", source)
	}
	for _, u := range m.Units {
		if u.X>>8 == 40 && u.Y>>8 == 40 {
			t.Skipf("%s has a unit at the chosen cell", source)
		}
	}
	at, err := host.model.CloneRecord(mapedit.KindUnit, 0, 40<<8|0x80, 40<<8|0x80)
	if err != nil {
		t.Fatal(err)
	}
	if line, err := host.TestPlay(); err != nil || len(*started) != 1 {
		t.Fatalf("test play: %q %v, launches %d", line, err, len(*started))
	}
	args := (*started)[0]
	number, err := strconv.Atoi(startedArg(t, args, "mission"))
	if err != nil {
		t.Fatal(err)
	}
	f := guestFront(t, startedArg(t, args, "mods-dir"), TestPlayModID)
	app := f.App("second test play")
	res := ui.ChargenResult{Name: "Test", Choices: []int{0, 0, 0}, Stats: []int{31, 27, 24, 29}}
	open, err := f.MissionBegin(number)(res)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if f.live == nil || f.live.mission == nil || f.live.mission.number != number || f.Town.second == nil {
		t.Fatalf("mission %d did not open in the second-game session", number)
	}
	found := 0
	for _, e := range f.live.world.Entities() {
		if e.X == 40 && e.Y == 40 {
			found++
		}
	}
	if found != 1 {
		t.Fatalf("the world holds %d entities at the edited unit's cell, want 1", found)
	}
	t.Logf("PASS: %s plays as mission %d in the second-game session; the edited unit (record %d) stands in the world; launch %v", source, number, at, slices.Clone(args[1:]))
}
