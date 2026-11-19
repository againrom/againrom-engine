package game

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

// A map saved into a mod folder keeps its dialogue file in the folder the mod
// loader reads.
func TestMapSavedIntoAModFolderKeepsItsDialogueFile(t *testing.T) {
	e, _ := openGoalMap(t, millMap(t))
	s := formHost(e)
	if _, err := s.ExtendScript(); err != nil {
		t.Fatal(err)
	}
	buildMill(t, e)
	dlgApply(t, e)
	dlgCancel(t, e)
	_ = e.Key("escape")
	dir := writeEditorMod(t, t.TempDir(), "[[mission]]\nkey='choose'\nnumber=21\nmap='maps/choose.almx'\nafter=20\nparty=[{character='npc22'}]\n")
	if err := os.MkdirAll(filepath.Join(dir, "maps"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "maps", "choose.almx"), millMap(t), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveAs(filepath.Join(dir, "maps", "saved.almx")); err != nil {
		t.Fatal(err)
	}
	row := mod.Mission{Map: "maps/saved.almx"}
	if err := row.ReadFiles(dir); err != nil || row.DialogueFiles["dialogue"] == nil {
		t.Fatalf("the mod folder holds no dialogue beside the map: %v %v", err, row.DialogueFiles)
	}
}

// A loose map with a trigger that starts a dialogue and then pays gold is
// saved, opened again in a fresh editor and played with the test play key.
// The test mod the key writes carries the dialogue file, the game loads it
// as it loads a mod, the hero entering the area pays the gold and opens the
// dialogue with its replies, and a reply runs its action.
func TestTestPlayOfASavedLooseMapOpensItsDialogue(t *testing.T) {
	_, e := workspaceFixture(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "choose.almx")
	if err := os.WriteFile(path, millMap(t), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := e.Open(path); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
	closeMapPanel(t, e)
	e.Fit()
	s := formHost(e)
	if _, err := s.ExtendScript(); err != nil {
		t.Fatal(err)
	}
	wizNewTrigger(t, e, "Choose")
	wizPress(t, e, "", 0)
	wizPress(t, e, "c0.p1", 0)
	dragCell(e, image.Pt(12, 12), image.Pt(20, 20))
	wizPress(t, e, "+action", 0)
	trigChoose(t, e, wizField(t, e, "a0.item"), "Start dialog")
	wizPress(t, e, "a0.item", 0)
	fillMill(t, e)
	dlgApply(t, e)
	dlgCancel(t, e)
	wizPress(t, e, "+action", 0)
	trigChoose(t, e, wizField(t, e, "a1.item"), "Give gold")
	wizApply(t, e)
	if _, err := s.SaveInPlace(); err != nil {
		t.Fatal(err)
	}

	f, again := workspaceFixture(t)
	if err := again.Open(path); err != nil {
		t.Fatal(err)
	}
	again.Layout(1280, 800)
	if _, err := formHost(again).TestPlay(); err != nil || len(f.started) != 1 {
		t.Fatalf("the test play did not start: %v (started %d)", err, len(f.started))
	}
	modDir := filepath.Join(f.profile, "testplay", "testplay")
	src, _, err := mod.ReadInside(modDir, mod.MissionsFile)
	if err != nil {
		t.Fatal(err)
	}
	d, err := mod.ParseMissions("m", mod.MissionsFile, src, nil, "en")
	if err != nil || len(d.Missions) != 1 {
		t.Fatalf("the test mod declares %d missions: %v", len(d.Missions), err)
	}
	if err := d.Missions[0].ReadFiles(modDir); err != nil {
		t.Fatal(err)
	}
	if d.Missions[0].DialogueFiles["dialogue"] == nil {
		t.Fatalf("the test mod carries no dialogue file: %v", tree(t, modDir))
	}
	front := extendedScriptFront(t, d.Missions[0].MapBytes)
	front.Campaign = resolved(guestCampaign(), nil)
	if _, err := front.SetModMissions(d); err != nil {
		t.Fatal(err)
	}
	if err := front.App("test play").OpenMission(front.MissionOpener(d.Missions[0].Number)); err != nil {
		t.Fatal(err)
	}
	for range 80 {
		front.live.tick()
	}
	mw := front.live
	if got := mw.world.Purse(1); got != 500 {
		t.Fatalf("the trigger paid %d gold", got)
	}
	if mw.mission.dialogue == nil || len(mw.mission.replies) != 4 {
		t.Fatalf("the trigger opened no dialogue with four replies: %+v", mw.mission.replies)
	}
	mw.advanceNotice(ui.NoticeReply1)
	missionSteps(mw, 1)
	if got := mw.world.ScriptRegister(10); got != 1 {
		t.Fatalf("the reply left variable 10 at %d", got)
	}
}
