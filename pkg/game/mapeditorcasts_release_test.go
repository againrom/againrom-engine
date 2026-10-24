package game

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"againrom/pkg/mapedit"
	"againrom/pkg/mod"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseEditorCellSpellFieldsKeepEveryMap(t *testing.T) {
	x := installedInspector(t)
	count, records, unknown := 0, 0, 0
	for _, key := range installedMapKeys(x) {
		e := x.Editor()
		if err := e.Open(key); err != nil {
			t.Fatal(key, err)
		}
		s := e.Document().Edits.(*mapInspectionEdits)
		before := s.model.Bytes()
		m, err := s.model.Map()
		if err != nil {
			t.Fatal(err)
		}
		for i, rec := range m.Enchantments {
			records++
			fields := s.Fields("Enchantment", i)
			if rec.X != 0 || rec.Y != 0 {
				if len(fields) == 0 {
					unknown++
					continue
				}
				for _, f := range fields {
					if _, err := s.SetField("Enchantment", i, f.Key, f.Value); err != nil {
						t.Fatal(key, i, f.Key, err)
					}
				}
			}
		}
		out, err := s.model.SaveBytes()
		if err != nil || !bytes.Equal(before, out) || !bytes.Equal(before, s.model.Bytes()) {
			t.Fatal(key, "current fields changed map bytes", err)
		}
		count++
	}
	t.Logf("maps=%d type-9=%d unknown spatial=%d", count, records, unknown)
}

func TestReleaseEditorCellSpellFrames(t *testing.T) {
	x := installedInspector(t)
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	closeMapPanel(t, e)
	dir := os.Getenv("AGAINROM_CAST_FRAMES")
	if dir == "" {
		dir = t.TempDir()
	}
	e.Select(recordIndex(t, e, "Enchantment", 0), false)
	writeEditorFrame(t, e, filepath.Join(dir, "trap.png"), true, image.Rectangle{})
	if err := e.Key("casts"); err != nil {
		t.Fatal(err)
	}
	writeEditorFrame(t, e, filepath.Join(dir, "cast-add.png"), true, image.Rectangle{})
}

func TestReleaseEditorPlacedTrapCastsInTestPlay(t *testing.T) {
	x, started := f5Inspector(t)
	plain := guestMods(t, guestModID)
	saveEditedGuestMap(t, x, plain, func(*ui.MapEditor) {})
	f := guestFrontWorld(t, plain)
	_, start := leaderOf(t, f)
	route := openRoute(t, f.live.mission.state.Map, start)
	cell := route.at(4, 0)
	edited := guestMods(t, guestModID)
	modsDir, number := "", 0
	saveEditedGuestMap(t, x, edited, func(e *ui.MapEditor) {
		closeMapPanel(t, e)
		addCellSpell(t, e, 0, cell, "0")
		index := e.Document().Edits.(*mapInspectionEdits).model.Count(mapedit.KindEnchantment) - 1
		e.Select(recordIndex(t, e, "Enchantment", index), false)
		recordFieldInput(t, e, "power", 1)
		recordFieldInput(t, e, "sourcex", int64(cell.X))
		recordFieldInput(t, e, "sourcey", int64(cell.Y))
		if err := pastNotice(e, "testplay"); err != nil {
			t.Fatal(err)
		}
		if len(*started) != 1 {
			t.Fatal("F5 did not request exactly one launch", *started)
		}
		args := (*started)[0]
		for i := 0; i+1 < len(args); i++ {
			if args[i] == "-mods-dir" {
				modsDir = args[i+1]
			}
			if args[i] == "-mission" {
				var err error
				number, err = strconv.Atoi(args[i+1])
				if err != nil {
					t.Fatal(err)
				}
			}
		}
		out, err := os.ReadFile(filepath.Join(modsDir, TestPlayModID, testPlayMap))
		if err != nil {
			t.Fatal(err)
		}
		want, err := e.Document().Edits.(*mapInspectionEdits).model.SaveBytes()
		if err != nil || !bytes.Equal(out, want) {
			t.Fatal("F5 changed the edited trap map", err)
		}
	})
	if modsDir == "" || number == 0 {
		t.Fatal("F5 arguments omitted the test mission")
	}
	f = guestFrontWorld(t, modsDir, mod.Mission{Mod: TestPlayModID, Number: number})
	id, _ := leaderOf(t, f)
	w := f.live.world
	if trail := walk(f, id, route.start); len(trail) == 0 || trail[len(trail)-1] != route.start {
		t.Fatal("approach failed")
	}
	sim.Step(w, []sim.Command{sim.MoveTo(id, sim.CellPoint{X: int32(cell.X), Y: int32(cell.Y)})})
	for tick := 0; tick < 1500; tick++ {
		r := sim.StepReported(w, nil)
		for _, cast := range r.ScriptCasts {
			if cast.Spell == 13 && cast.FromX == int32(cell.X) && cast.FromY == int32(cell.Y) && cast.ToX == int32(cell.X) && cast.ToY == int32(cell.Y) {
				t.Logf("editor trap at %v casts spell 13 at tick %d through the game loader", cell, tick)
				return
			}
		}
	}
	t.Fatal(fmt.Sprintf("placed trap at %v did not cast; tails=%v", cell, w.CellTails()))
}
