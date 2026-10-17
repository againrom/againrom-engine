package game

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mod"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseMapEditorCompleteScenarioPackageSAV(t *testing.T) {
	x := installedInspector(t)
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Key("new"); err != nil {
		t.Fatal(err)
	}
	_ = e.Key("clear")
	e.TextInput("32x32")
	for range 2 { // the size, then single-player
		if err := e.Key("enter"); err != nil {
			t.Fatal(err)
		}
	}
	e.Document().Viewer.SetFlat(true)
	closeMapPanel(t, e)
	if err := workspaceField(e, "metadata", 0, "Group mission"); err != nil {
		t.Fatal(err)
	}
	_ = e.Key("drop")
	clickCell(e, image.Pt(16, 12))
	class := int64(creatureClass(t, x))
	for i, cell := range []image.Point{{22, 22}, {20, 22}, {18, 22}} {
		recordAddInput(t, e, "unit", cell, class)
		group := int64(71)
		if i == 2 {
			group = 72
		}
		recordFieldInput(t, e, "group", group)
	}
	recordAddInput(t, e, "structure", image.Pt(10, 18), int64(authoredBlockingKind(t, x)))
	_ = e.Key("clone")
	clickCell(e, image.Pt(15, 18))
	e.Select(recordIndex(t, e, "Structure", 0), true)
	recordFieldInput(t, e, "owner", 2)
	e.Select(recordIndex(t, e, "Unit", 0), true)
	if err := e.Key("stock"); err != nil {
		t.Fatal(err)
	}
	const itemCode = 0x1106
	recordFieldInput(t, e, "additem", itemCode)
	recordRecipeInput(t, e, 5)
	recordAddInput(t, e, "sack", image.Pt(17, 12), itemCode)
	recordRecipeInput(t, e, 9)
	const body = "Group 71 is alive.\nSave, reload and finish the group."
	const page = "<part=1>\n" + body
	workspacePage(t, e, "event01.txt", page)

	group := scriptWindowAdd(t, e, "Condition")
	scriptWindowSet(t, e, "Condition", group, "type0", "2")
	scriptWindowSet(t, e, "Condition", group, "value0", "71")
	scriptWindowSet(t, e, "Condition", group, "opcode", "1")
	zero := scriptWindowAdd(t, e, "Condition")
	variable := scriptWindowAdd(t, e, "Condition")
	scriptWindowSet(t, e, "Condition", variable, "value0", "99")
	scriptWindowSet(t, e, "Condition", variable, "opcode", "19")
	one := scriptWindowAdd(t, e, "Condition")
	scriptWindowSet(t, e, "Condition", one, "value0", "1")
	set := scriptWindowAdd(t, e, "Action")
	scriptWindowSet(t, e, "Action", set, "value0", "99")
	scriptWindowSet(t, e, "Action", set, "opcode", "3")
	scriptWindowSet(t, e, "Action", set, "type1", "1")
	scriptWindowSet(t, e, "Action", set, "value1", "1")
	message := scriptWindowAdd(t, e, "Action")
	win := scriptWindowAdd(t, e, "Action")
	scriptWindowSet(t, e, "Action", win, "opcode", "4")
	nodeID := func(kind string, index int) string {
		for _, field := range e.Document().Edits.Fields(kind, index) {
			if field.Key == "id" {
				return strconv.FormatInt(field.Value, 10)
			}
		}
		t.Fatalf("%s %d has no node ID", kind, index)
		return ""
	}
	stage := scriptWindowAdd(t, e, "Trigger")
	result := scriptWindowAdd(t, e, "Trigger")
	for _, row := range []struct {
		trigger    int
		key, value string
	}{
		{stage, "left0", nodeID("Condition", group)}, {stage, "right0", nodeID("Condition", zero)}, {stage, "cmp0", "2"},
		{stage, "left1", nodeID("Condition", variable)}, {stage, "right1", nodeID("Condition", zero)},
		{stage, "action0", nodeID("Action", set)}, {stage, "action1", nodeID("Action", message)},
		{result, "left0", nodeID("Condition", group)}, {result, "right0", nodeID("Condition", zero)},
		{result, "left1", nodeID("Condition", variable)}, {result, "right1", nodeID("Condition", one)},
		{result, "action0", nodeID("Action", win)},
	} {
		scriptWindowSet(t, e, "Trigger", row.trigger, row.key, row.value)
	}

	wantMap, err := alm.SaveForm(e.Document().SourceBytes)
	if err != nil {
		t.Fatal(err)
	}
	parent := t.TempDir()
	const id = "complete-map"
	folder := filepath.Join(parent, id)
	for _, field := range []struct {
		index int
		value string
	}{{1, id}, {2, "Complete map"}, {3, mod.MissionReplace}, {4, "20"}, {0, folder}} {
		if err := workspaceField(e, "package", field.index, field.value); err != nil {
			t.Fatal("W package", field.index, err)
		}
	}
	if err := pastNotice(e, "saveas"); err != nil || e.Dirty() {
		t.Fatalf("Ctrl+S did not checkpoint the complete package: %v", err)
	}
	mission := loadWorkspaceMod(t, x.archives.Root, folder)
	if mission.Kind != mod.MissionReplace || mission.Number != 20 || !bytes.Equal(mission.MapBytes, wantMap) {
		t.Fatal("normal mod loader changed the authored route or map")
	}
	coldEditor := installedInspector(t).Editor()
	if err := coldEditor.Open(newMapSource(32, 32)); err != nil {
		t.Fatal(err)
	}
	if err := workspaceField(coldEditor, "package", 0, folder); err != nil {
		t.Fatal("W cold reopen", err)
	}
	if coldEditor.Dirty() || !bytes.Equal(coldEditor.Document().SourceBytes, wantMap) {
		t.Fatal("cold editor did not restore the authored map checkpoint")
	}
	if got := coldEditor.Document().Edits.(*mapInspectionEdits).workspace.Pages["event01.txt"]; string(got) != page {
		t.Fatalf("cold editor page %q", got)
	}
	if _, err := os.Stat(filepath.Join(folder, editorPackageRecord)); err != nil {
		t.Fatal("W did not write an owned package", err)
	}

	f := guestFront(t, parent, id)
	f.SetTipsOff(true)
	app := f.App("complete authored map")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	if err := app.OpenChargen(ui.NewChargen(f.ChargenSetup()), f.NewGameBegin(20)); err != nil {
		t.Fatal(err)
	}
	if err := headlessCreateCharacter(app, HeadlessCharacter{Name: "Author"}); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenMap || f.liveMission != 20 {
		t.Fatalf("production entry screen=%s mission=%d", app.Screen(), f.liveMission)
	}
	check := func(f *FrontEnd) string {
		t.Helper()
		items := authoredRecordItems(t, f, itemCode)
		ss := f.live.world.Structures()
		if len(ss) != 2 || ss[0].ID == ss[1].ID || ss[0].Field42 <= 0 || ss[1].Field42 <= 0 || ss[0].Blocking == 0 {
			t.Fatal("authored structures lost their unique identities or standing obstacle", ss)
		}
		return fmt.Sprintf("%s structures %v/%v", items, ss[0].ID, ss[1].ID)
	}
	wantRecords := check(f)
	wantBody := string(EncodeInstallTextFile([]byte(body), x.font.Selector))
	sawPage := false
	for frame := 0; frame < 1000; frame++ {
		text, kind, up := f.LiveNotice()
		if up && kind == ui.NoticeDialogue && text == wantBody && f.live.mission.part == 1 {
			sawPage = bytes.Equal(f.live.mission.payload, EncodeInstallTextFile([]byte(page), x.font.Selector))
			break
		}
		if up {
			if err := app.HeadlessActivate("notice"); err != nil {
				t.Fatal(err)
			}
		}
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if !sawPage || f.live.world.ScriptRegister(99) != 1 || f.live.world.Outcome() != sim.OutcomeUndecided {
		t.Fatalf("authored stage page=%v register99=%d outcome=%v", sawPage, f.live.world.ScriptRegister(99), f.live.world.Outcome())
	}
	store, name, _ := menuSAVE(t, f, app, OriginalStore{})
	cold := guestFront(t, parent, id)
	cold.SetTipsOff(true)
	loaded := cold.App("complete authored map cold LOAD")
	t.Cleanup(loaded.StopAudio)
	loaded.Layout(1024, 768)
	save, list, load := cold.SaveSeams(store, OriginalStore{}, nil)
	loaded.SetSaveSeams(save, list, load)
	groundAppLoad(t, loaded, list, localOriginalSaveToken(name))
	if got := check(cold); got != wantRecords || cold.live.world.ScriptRegister(99) != 1 || cold.live.world.Outcome() != sim.OutcomeUndecided {
		t.Fatalf("cold LOAD records=%s want=%s register99=%d outcome=%v", got, wantRecords, cold.live.world.ScriptRegister(99), cold.live.world.Outcome())
	}
	for _, step := range []struct {
		id   uint32
		want sim.Outcome
	}{{3, sim.OutcomeUndecided}, {1, sim.OutcomeUndecided}, {2, sim.OutcomeWon}} {
		killScriptUnit(t, cold, step.id)
		if out := runScript(t, cold, loaded, 120); out != step.want {
			t.Fatalf("after map unit %d died: outcome=%v want=%v", step.id, out, step.want)
		}
	}
	t.Log("N/D/U/J/C/K/I/X/F2/A/E/W/Ctrl+S -> cold package reopen -> production chargen -> authored event -> menu SAV -> cold LOAD -> unrelated/first/last group deaths; only last wins")
}
