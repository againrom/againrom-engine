package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// pointerSaveMission is mission 10 with one Pointer (map id 42, kind 0x4a)
// at cell 16/21 whose type-11c record names action node 1, so its slot is 1:
// node 0 raises event 7, node 1 event 19.
func pointerSaveMission() secondMissionContent {
	msg := func(id, event uint32) alm.ScriptNode {
		n := alm.ScriptNode{Opcode: 2, ID: id}
		n.Type[0], n.Value[0] = 1, event
		return n
	}
	text := "#event7\r\n<part=1>\r\nwrong event\r\n#event19\r\n<part=1>\r\n" + pointerEventLine + "\r\n"
	return secondMissionContent{
		objects:  []alm.Object{{X: 16<<8 | 0x80, Y: 21<<8 | 0x80, Kind: 0x4a, Field0C: 1, Field12: 42}},
		actions:  []alm.ScriptNode{msg(1, 7), msg(2, 19)},
		pointers: []alm.StructureAction{{ID: 42, Gate: 1, Position: 1}},
		main:     []synth.File{{Path: "text/mission10.txt", Data: []byte(text)}},
		font:     true,
	}
}

// pointerSaveFile is the fixture mission file with the hero at 11/21 and the
// Pointer in its Buildings.
func pointerSaveFile() []byte {
	hero := secondMissionFixtureHero()
	body := secondSavedWorldOf(10, 44, &hero, true, 0, []secondSavedPointer{{col: 16, row: 21, kind: 0x4a, id: 42, key: 0x0b000001}})
	return labelledContainer(sav.SecondGame.Magic, "mission", body,
		append(secondMissionStore(4, 0, -1), secondSavedTail(0, [2]uint32{10, placeMission})...))
}

func pointerSaveSign(t *testing.T, w *sim.World) sim.Structure {
	t.Helper()
	for _, s := range w.Structures() {
		if sim.PointerKind(s.Kind) {
			return s
		}
	}
	t.Fatal("no Pointer in the World")
	return sim.Structure{}
}

// A SAVE taken while the hero walks to a Pointer cold-LOADs, and the loaded
// run equals the uninterrupted one: the hero arrives, the same event window
// opens on the same step, and the World hash after it closes is equal.
func TestPointerUseSavedOnTheWayColdLoadsAndRunsAsTheUninterruptedRun(t *testing.T) {
	c := pointerSaveMission()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0001.sav"), pointerSaveFile(), 0o600); err != nil {
		t.Fatal(err)
	}
	f, app := secondMissionLoadOf(t, c, dir, "game0001.sav")
	sign := pointerSaveSign(t, f.live.world)
	if !sign.Usable() || sign.UseAmount != 1 {
		t.Fatalf("loaded Pointer %+v, want slot 1 and usable", sign)
	}
	hero := secondMissionActor(t, f.live.world)
	f.live.useStructure(uint32(hero.ID), uint32(sign.ID))
	for range 4 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if len(f.live.world.StructureUses()) != 1 || f.live.mission.open {
		t.Fatalf("uses %v open %v, want one use pending on the way", f.live.world.StructureUses(), f.live.mission.open)
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := SaveStore{Dir: dir}.List()
	if err != nil || len(entries) != 1 || entries[0].Name == "game0001.sav" {
		t.Fatalf("our SAVE: %+v %v: %s", entries, err, app.HeadlessMessage())
	}
	name := entries[0].Name
	if err := os.Remove(filepath.Join(dir, "game0001.sav")); err != nil {
		t.Fatal(err)
	}
	cold, coldApp := secondMissionLoadOf(t, c, dir, name)
	if msg := coldApp.HeadlessMessage(); cold.live == nil || cold.live.world == nil {
		t.Fatalf("cold LOAD refused: %s", msg)
	}
	if cold.live.world.Hash() != f.live.world.Hash() || len(cold.live.world.StructureUses()) != 1 {
		t.Fatalf("cold LOAD: hash equal %v, uses %v", cold.live.world.Hash() == f.live.world.Hash(), cold.live.world.StructureUses())
	}
	opened := [2]int{-1, -1}
	runs := [2]*FrontEnd{f, cold}
	apps := [2]*ui.App{app, coldApp}
	for _, a := range apps {
		for tries := 0; a.Screen() != ui.ScreenMap; tries++ {
			if err := a.HeadlessKey("escape"); err != nil || tries > 4 {
				t.Fatalf("screen %v: %v", a.Screen(), err)
			}
		}
	}
	for step := 0; step < 400 && (opened[0] < 0 || opened[1] < 0); step++ {
		for i, a := range apps {
			if err := a.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
			if opened[i] < 0 && runs[i].live.mission.open {
				opened[i] = step
			}
		}
		if f.live.world.Hash() != cold.live.world.Hash() {
			t.Fatalf("the loaded run diverged at step %d", step)
		}
	}
	if opened[0] < 0 || opened[0] != opened[1] {
		t.Fatalf("event window opened at steps %v, want the same step", opened)
	}
	for _, r := range runs {
		if e := secondMissionActor(t, r.live.world); e.X < 15 || e.X > 17 || e.Y < 20 || e.Y > 22 {
			t.Fatalf("the event window opened with the hero at %d,%d, not beside the Pointer", e.X, e.Y)
		}
	}
	held := f.live.world.Hash()
	for range 48 {
		for _, a := range apps {
			if err := a.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
		}
	}
	for i := range apps {
		body, kind, open := runs[i].live.view.NoticeState()
		if !open || kind != ui.NoticeDialogue || body != pointerEventLine+"\r\n" {
			t.Fatalf("run %d notice %q/%v/%v, want event 19's line", i, body, kind, open)
		}
		if err := apps[i].HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
		if _, _, open := runs[i].live.view.NoticeState(); open {
			t.Fatalf("run %d: Enter left the event window open", i)
		}
	}
	for range 8 {
		for _, a := range apps {
			if err := a.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
		}
	}
	if f.live.mission.open || cold.live.mission.open || f.live.world.Hash() != cold.live.world.Hash() || f.live.world.Hash() == held {
		t.Fatalf("after the close: open %v/%v, hash equal %v, mission ran on %v", f.live.mission.open, cold.live.mission.open,
			f.live.world.Hash() == cold.live.world.Hash(), f.live.world.Hash() != held)
	}
}

// A SAV program whose instant carries a negative action slot is refused on
// LOAD, as the World byte form refuses it; slot 0 and above compile.
func TestCurrentScriptProgramRefusesANegativeActionSlot(t *testing.T) {
	for _, slot := range []int32{-1, 0, 1} {
		p := currentScriptProgram{Dialect: sim.ScriptROM2, Checks: missionChecks(),
			Instants: []sim.ScriptInstant{{Slot: slot, Op: sim.ScriptInstantMessage, Args: [10]int32{19}}}}
		if _, err := p.compile(); (err != nil) != (slot < 0) {
			t.Fatalf("slot %d: %v", slot, err)
		}
	}
}
