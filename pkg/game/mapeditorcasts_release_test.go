package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

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

// testPlayLaunch presses F5 and returns the mods folder and mission the
// recording host was asked to start.
func testPlayLaunch(t *testing.T, e *ui.MapEditor, started *[][]string) (string, int) {
	t.Helper()
	*started = nil
	if err := pastNotice(e, "testplay"); err != nil {
		t.Fatal(err)
	}
	if len(*started) != 1 {
		t.Fatal("F5 did not request exactly one launch", *started)
	}
	dir, number := "", 0
	args := (*started)[0]
	for i := 0; i+1 < len(args); i++ {
		switch args[i] {
		case "-mods-dir":
			dir = args[i+1]
		case "-mission":
			n, err := strconv.Atoi(args[i+1])
			if err != nil {
				t.Fatal(err)
			}
			number = n
		}
	}
	if dir == "" || number == 0 {
		t.Fatal("F5 arguments omitted the test mission", args)
	}
	return dir, number
}

// A trap added without a template on a new map, which holds no type-9
// record, fires its spell when the party steps on its cell in test play.
func TestReleaseEditorNewTrapOnANewMapCastsInTestPlay(t *testing.T) {
	x, started := f5Inspector(t)
	c := &editorClicker{t: t, now: time.Unix(1, 0)}
	e := x.Editor()
	e.Layout(1280, 800)
	_ = e.Key("new")
	_ = e.Key("enter")
	r, ok := e.PromptButtonRect(0)
	c.click(e, r, ok, "the mode button")
	s := e.Document().Edits.(*mapInspectionEdits)
	if n := s.model.Count(mapedit.KindEnchantment); n != 0 {
		t.Fatalf("the new map holds %d type-9 records", n)
	}
	dir, number := testPlayLaunch(t, e, started)
	f := guestFrontWorld(t, dir, mod.Mission{Mod: TestPlayModID, Number: number})
	_, start := leaderOf(t, f)
	route := openRoute(t, f.live.mission.state.Map, start)
	cell := route.at(4, 0)
	if err := e.Key("casts"); err != nil {
		t.Fatal(err)
	}
	addSteps(t, e, map[string]string{"template": strconv.Itoa(mapedit.SpellConstructor(0)), "link": "0",
		"cell": strconv.Itoa(cell.X) + "," + strconv.Itoa(cell.Y), "spell": "13", "power": "1"})
	want := cellSpellBytes(0, uint32(cell.X), uint32(cell.Y), 0, 0, 13, 1,
		[3]uint16{uint16(cell.X), uint16(cell.Y), 0}, [3]uint16{uint16(cell.X), uint16(cell.Y), 0})
	if got, _ := s.model.Record(mapedit.KindEnchantment, 0); !bytes.Equal(got, want) {
		t.Fatalf("new trap %x want %x", got, want)
	}
	dir, number = testPlayLaunch(t, e, started)
	f = guestFrontWorld(t, dir, mod.Mission{Mod: TestPlayModID, Number: number})
	id, _ := leaderOf(t, f)
	w := f.live.world
	if trail := walk(f, id, route.start); len(trail) == 0 || trail[len(trail)-1] != route.start {
		t.Fatal("approach failed")
	}
	sim.Step(w, []sim.Command{sim.MoveTo(id, sim.CellPoint{X: int32(cell.X), Y: int32(cell.Y)})})
	for tick := 0; tick < 1500; tick++ {
		for _, cast := range sim.StepReported(w, nil).ScriptCasts {
			if cast.Spell == 13 && cast.FromX == int32(cell.X) && cast.FromY == int32(cell.Y) && cast.ToX == int32(cell.X) && cast.ToY == int32(cell.Y) {
				t.Logf("new trap at %v casts spell 13 at tick %d", cell, tick)
				return
			}
		}
	}
	t.Fatalf("new trap at %v did not cast; tails=%v", cell, w.CellTails())
}

// A shipped casting structure record off its structure's anchor cell edits
// and serves as a template; only the edited and the copied bytes change.
func TestReleaseEditorOffAnchorCastingRecordEditsAndCopies(t *testing.T) {
	x := installedInspector(t)
	for _, key := range installedMapKeys(x) {
		e := x.Editor()
		e.Layout(1280, 800)
		if err := e.Open(key); err != nil {
			t.Fatal(key, err)
		}
		s := e.Document().Edits.(*mapInspectionEdits)
		m, err := s.model.Map()
		if err != nil {
			t.Fatal(err)
		}
		for i, rec := range m.Enchantments {
			if rec.A != 4 || rec.X == 0 && rec.Y == 0 || !spellBound(m, rec) {
				continue
			}
			id := uint32(rec.B) | uint32(rec.C)<<16
			var anchor image.Point
			for _, o := range m.Objects {
				if uint32(int32(int16(o.Field12))) == id {
					anchor = image.Pt(int(o.X>>8), int(o.Y>>8))
				}
			}
			if anchor == image.Pt(int(rec.X), int(rec.Y)) {
				continue
			}
			closeMapPanel(t, e)
			before, _ := s.model.Record(mapedit.KindEnchantment, i)
			e.Select(recordIndex(t, e, "Enchantment", i), false)
			power := int64(rec.SpellRaw>>16) ^ 1
			recordFieldInput(t, e, "power", power)
			want := bytes.Clone(before)
			binary.LittleEndian.PutUint16(want[20:], uint16(power))
			if got, _ := s.model.Record(mapedit.KindEnchantment, i); !bytes.Equal(got, want) {
				t.Fatalf("%s record %d off anchor %v: edit %x want %x", key, i, anchor, got, want)
			}
			cell := image.Pt(int(rec.X)+1, int(rec.Y))
			n := s.model.Count(mapedit.KindEnchantment)
			if err := e.Key("casts"); err != nil {
				t.Fatal(err)
			}
			addSteps(t, e, map[string]string{"template": strconv.Itoa(i), "cell": strconv.Itoa(cell.X) + "," + strconv.Itoa(cell.Y)})
			copied := bytes.Clone(want)
			binary.LittleEndian.PutUint32(copied[4:], uint32(cell.X))
			binary.LittleEndian.PutUint32(copied[8:], uint32(cell.Y))
			if got, _ := s.model.Record(mapedit.KindEnchantment, n); s.model.Count(mapedit.KindEnchantment) != n+1 || !bytes.Equal(got, copied) {
				t.Fatalf("%s record %d: copy %x want %x", key, i, got, copied)
			}
			t.Logf("%s record %d at %d,%d, structure %d anchored at %v: edited and copied to %v", key, i, rec.X, rec.Y, id, anchor, cell)
			return
		}
	}
	t.Fatal("no installed casting structure record sits off its structure's anchor cell")
}
