package game

import (
	"bytes"
	"image"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func newScriptAuthorMod(t *testing.T) (string, *ui.MapEditor, *MapInspector) {
	t.Helper()
	root := newMapMod(t)
	folder := filepath.Join(root, newMapModID)
	declaration := filepath.Join(folder, "data", "missions.toml")
	b, err := os.ReadFile(declaration)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(declaration, append(b, []byte("texts = \"mission/text\"\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
	writeTestPlayInput(t, filepath.Join(folder, "mission", "text", "event01.txt"), []byte("Stage one"))
	x := installedInspector(t)
	if err := x.SetMod(folder); err != nil {
		t.Fatal(err)
	}
	e := x.Editor()
	if err := e.Open(newMapSource(32, 32)); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
	scriptWindowScenario(t, e)
	if _, err := scriptOf(t, e).SetDropCell(image.Pt(16, 16)); err != nil {
		t.Fatal(err)
	}
	_ = e.Key("undo")
	_ = e.Key("redo")
	if err := e.Key("saveas"); err != nil || e.Dirty() {
		t.Fatalf("save authored mission: %v", err)
	}
	return root, e, x
}

func TestReleaseScriptWindowScenarioContinuesFromSAVColdLOAD(t *testing.T) {
	root, e, x := newScriptAuthorMod(t)
	reopened := x.Editor()
	if err := reopened.Open(filepath.Join(root, newMapModID, "mission", "map.alm")); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(reopened.Document().SourceBytes, e.Document().SourceBytes) {
		t.Fatal("saved/reopened ALM changed")
	}
	f := guestFront(t, root, newMapModID)
	f.SetTipsOff(true)
	a := f.App("authored scenario")
	t.Cleanup(a.StopAudio)
	a.Layout(1024, 768)
	if err := a.OpenChargen(ui.NewChargen(f.ChargenSetup()), f.NewGameBegin(20)); err != nil {
		t.Fatal(err)
	}
	if err := headlessCreateCharacter(a, HeadlessCharacter{Name: "Author"}); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ui.ScreenMap || f.liveMission != 20 {
		t.Fatalf("entry screen=%s mission=%d", a.Screen(), f.liveMission)
	}
	for i := 0; i < 1000 && f.live.world.ScriptRegister(99) == 0; i++ {
		runScript(t, f, a, 1)
	}
	if f.live.world.ScriptRegister(99) != 3 || f.live.world.Outcome() != sim.OutcomeUndecided {
		t.Fatalf("mid-scenario register=%d outcome=%v", f.live.world.ScriptRegister(99), f.live.world.Outcome())
	}
	before, err := f.live.world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	name, _ := deadPatrolSave(t, f, store)
	cold := guestFront(t, root, newMapModID)
	cold.SetTipsOff(true)
	_, _, load := cold.SaveSeams(store, OriginalStore{}, nil)
	open, town, err := load(localOriginalSaveToken(name))
	if err != nil || town {
		t.Fatalf("cold LOAD town=%v error=%v", town, err)
	}
	ca := cold.App("authored scenario cold LOAD")
	t.Cleanup(ca.StopAudio)
	ca.Layout(1024, 768)
	if err := ca.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	after, err := cold.live.world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("cold LOAD changed mid-scenario World")
	}
	if out := runScript(t, cold, ca, 1000); out != sim.OutcomeWon {
		t.Fatalf("next action after cold LOAD outcome=%v", out)
	}
	if cold.live.world.ScriptRegister(99) != 3 {
		t.Fatal("once/action order lost across SAV")
	}
	t.Logf("authored ALM reopened; production chargen -> mission20; register99=3; SAV/cold LOAD exact World; next script pass won")
}
