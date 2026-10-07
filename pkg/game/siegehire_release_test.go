package game

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// siegeTypes returns the TypeIDs of the installed Catapult and Ballista rows,
// found by name as the tavern's buildSiegeSquad finds them.
func siegeTypes(t *testing.T, f *FrontEnd) map[int32]bool {
	t.Helper()
	types := map[int32]bool{}
	for _, name := range []string{"Catapult", "Ballista"} {
		for i := 1; i < f.Table.Units.Len(); i++ {
			if f.Table.Units.EntryName(i) != name {
				continue
			}
			d, err := data.NewUnitDef(name, f.Table.Units.EntryParams(i))
			if err != nil {
				t.Fatal(err)
			}
			types[d.TypeID] = true
			break
		}
	}
	if len(types) != 2 {
		t.Fatalf("installed siege types %v", types)
	}
	return types
}

// siegeSAVRows returns the token row of each placement-free siege actor in a
// SAV, keyed by TypeID, and requires exactly one actor of each type.
func siegeSAVRows(t *testing.T, raw []byte, types map[int32]bool) map[int32]uint8 {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := file.ActorGraph()
	if err != nil {
		t.Fatal(err)
	}
	rows := map[int32]uint8{}
	for _, a := range graph.Actors {
		if a.Class != "Unit" || !types[int32(a.TypeID)] || a.MapUnitID != 0 {
			continue
		}
		if _, repeated := rows[int32(a.TypeID)]; repeated {
			t.Fatalf("SAV holds more than one hired siege actor of type %d", a.TypeID)
		}
		rows[int32(a.TypeID)] = a.DefRow
	}
	if len(rows) != len(types) {
		t.Fatalf("SAV hired siege rows %v, want one per type %v", rows, types)
	}
	return rows
}

// liveSiegeHires returns the player's placement-free siege actors.
func liveSiegeHires(t *testing.T, f *FrontEnd, types map[int32]bool) []sim.Entity {
	t.Helper()
	var out []sim.Entity
	for _, e := range f.live.world.Entities() {
		if types[e.TypeID] && e.MapUnitID == 0 && e.Owner == sim.SelfSlot {
			out = append(out, e)
		}
	}
	if len(out) != len(types) {
		t.Fatalf("live world holds %d hired siege actors, want %d", len(out), len(types))
	}
	return out
}

// moveSiegeHires orders every hired siege actor dx cells along its row and
// requires each to stand on its target cell within 256 ticks.
func moveSiegeHires(t *testing.T, f *FrontEnd, types map[int32]bool, dx int32) {
	t.Helper()
	before := liveSiegeHires(t, f, types)
	for _, e := range before {
		f.live.pending = append(f.live.pending, sim.MoveTo(e.ID, sim.CellPoint{X: e.X + dx, Y: e.Y}))
	}
	for range 256 {
		f.live.tick()
	}
	for i, e := range liveSiegeHires(t, f, types) {
		if e.X != before[i].X+dx || e.Y != before[i].Y {
			t.Fatalf("hired siege actor %d stands at (%d,%d), ordered to (%d,%d)", e.ID, e.X, e.Y, before[i].X+dx, before[i].Y)
		}
	}
}

// writeOrdinarySAV writes SAV through the ordinary producer and returns its path
// and bytes.
func writeOrdinarySAV(t *testing.T, f *FrontEnd, name string) (string, []byte) {
	t.Helper()
	s, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal("ordinary SAVE", err)
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return path, raw
}

// TestReleaseHiredSiegeCreaturesSurviveMissionSAV hires one Catapult and one
// Ballista with the tavern's builder, enters mission 10 and writes SAV through
// the ordinary producer. Two cycles load the file in a fresh FrontEnd, move
// both creatures and write SAV again with the loaded token rows unchanged.
func TestReleaseHiredSiegeCreaturesSurviveMissionSAV(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	types := siegeTypes(t, f)
	party := f.ChargenParty(ui.ChargenResult{Name: "Siege witness", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	for typ := 1; typ <= 2; typ++ {
		members, ok := buildSiegeSquad(f.Table, typ, 1)
		if !ok || len(members) != 1 {
			t.Fatalf("siege type %d did not resolve", typ)
		}
		party = append(party, members...)
	}
	f.Carried = party
	if err := f.App("siege hire SAV").OpenMission(f.MissionOpenerWith(10, f.Carried)); err != nil {
		t.Fatal(err)
	}
	liveSiegeHires(t, f, types)
	path, raw := writeOrdinarySAV(t, f, "siege.sav")
	want := siegeSAVRows(t, raw, types)
	for cycle := range 2 {
		cold := loadAreaContinuation(t, path)
		moveSiegeHires(t, cold, types, int32(3-6*cycle))
		path, raw = writeOrdinarySAV(t, cold, fmt.Sprintf("siege%d.sav", cycle))
		if got := siegeSAVRows(t, raw, types); !reflect.DeepEqual(got, want) {
			t.Fatalf("cycle %d SAV siege rows %v, want %v", cycle, got, want)
		}
	}
}

// TestReleaseOwnerSiegeHireSAVLoads loads the owner's mission-150 SAV, whose
// hired Catapult and Ballista carry Units row 0, and the original EN resave
// of that file, which keeps row 0. Each loads, moves both creatures, writes
// SAV with the rows unchanged and loads that SAV again.
func TestReleaseOwnerSiegeHireSAVLoads(t *testing.T) {
	dir := os.Getenv("AGAINROM_SIEGE_HIRE_DIR")
	if dir == "" {
		t.Skip("set AGAINROM_SIEGE_HIRE_DIR to the owner mission-150 siege-hire saves")
	}
	for _, tc := range []struct {
		name, sha string
		dx        int32
	}{
		{"savegoo.sav", "3d747808aa4fcd724dfe5a00ed49a53b8cef9bda237764a27b29625d263e47a1", -3},
		{"game0009-original-resave.sav", "a88f9a0f195014a67a2685937f88678d210dda697dcd037daebf5f4089109da6", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, tc.name)
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if fmt.Sprintf("%x", sha256.Sum256(raw)) != tc.sha {
				t.Fatal("owner siege-hire SAV SHA differs")
			}
			f := loadAreaContinuation(t, path)
			types := siegeTypes(t, f)
			want := siegeSAVRows(t, raw, types)
			moveSiegeHires(t, f, types, tc.dx)
			next, resaved := writeOrdinarySAV(t, f, "resave.sav")
			if got := siegeSAVRows(t, resaved, types); !reflect.DeepEqual(got, want) {
				t.Fatalf("resave siege rows %v, want %v", got, want)
			}
			liveSiegeHires(t, loadAreaContinuation(t, next), types)
		})
	}
}
