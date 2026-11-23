package game

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapedit"
	"againrom/pkg/mod"
	"againrom/pkg/rules"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// The globals the fixture's mill mission leaves once its setup and first
// reply ran, and the initials the mod declares.
var (
	millWonGlobals  = []sim.ScriptGlobal{{ID: 1, Value: 1}, {ID: 2, Value: 10}, {ID: 3, Value: math.MinInt32}}
	millInitials    = []sim.ScriptGlobal{{ID: 1}, {ID: 2, Value: 5}, {ID: 3, Value: math.MaxInt32 - 1}}
	yardTipGlobals  = []sim.ScriptGlobal{{ID: 1, Value: 1}, {ID: 2, Value: 11}, {ID: 3, Value: math.MinInt32}}
	millGlobalsName = "millglobals"
)

// globalsYardMap is the fixture's second map, mission 20. Trigger "Remember"
// sets variable 50 once global 1 is 1 and global 2 is 10, as the mill's win
// leaves them; trigger "Setup" opens the yard dialogue, whose first reply is
// offered only while global 2 is 10 and adds 1 to it and sets variable 51.
func globalsYardMap(t *testing.T) []byte {
	t.Helper()
	conditions := []scriptNode{
		{label: "Quest", opcode: opReadGlobal, id: 1, values: []uint32{1}, tags: []uint32{1}},
		{label: "One", opcode: 0x10002, id: 2, values: []uint32{1}, tags: []uint32{1}},
		{label: "Coins", opcode: opReadGlobal, id: 3, values: []uint32{2}, tags: []uint32{1}},
		{label: "Ten", opcode: 0x10002, id: 4, values: []uint32{10}, tags: []uint32{1}},
	}
	actions := []scriptNode{
		{label: "Drop location", opcode: dropActionOpcode, id: 99, values: []uint32{16, 16}, tags: []uint32{5, 6}},
		{label: "Tip", opcode: opAddGlobal, id: 21, values: []uint32{2, 1}, tags: []uint32{1, 1}},
		{label: "Remembered", opcode: 3, id: 31, values: []uint32{50, 1}, tags: []uint32{1, 1}},
		{label: "Thanked", opcode: 3, id: 32, values: []uint32{51, 1}, tags: []uint32{1, 1}},
		{label: "yard", opcode: 2, id: 41, values: []uint32{1, 1}, tags: []uint32{1, 1}},
	}
	o := synth.ALMOptions{Width: 40, Height: 40, Units: extendedUnits, Type7Payload: scriptPayload(actions, conditions)}
	b := built(t, o, func(ed *mapedit.Editor) {
		if err := ed.SetUnitField(0, "owner", 1); err != nil {
			t.Fatal(err)
		}
	}, nil).Bytes()
	out, err := alm.AppendScriptRecord(b, alm.ScriptRecord{Variables: 100, Triggers: []alm.ScriptTrigger{
		{Name: "Remember", Once: 1, Left: []uint32{1, 3}, Right: []uint32{2, 4}, Cmp: []uint32{0, 0}, Acts: []uint32{31}},
		{Name: "Setup", Once: 1, Acts: []uint32{41}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// globalsMillBase is the two-map fixture front: missions 10 and 20 of the
// install's campaign, mill the map at 10.
func globalsMillBase(t *testing.T, mill []byte) *FrontEnd {
	t.Helper()
	f := currentPoolFixtureFront(t)
	path := filepath.Join(t.TempDir(), ScenarioArchive)
	b := synth.Archive([]synth.File{{Path: "10.alm", Data: mill}, {Path: "20.alm", Data: globalsYardMap(t)},
		{Path: "npc.reg", Data: synth.NPCReg(nil)}})
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	fs, err := vfs.Open([]string{path}, nil)
	if err != nil {
		t.Fatal(err)
	}
	f.Archives.Containers = fs
	c := Campaign{Main: []int{10, 20}, Offered: []int{10}, Chapters: map[int]Chapter{10: {Mission: 10}}}
	f.Campaign, f.Town = resolved(c, nil), NewTown(c)
	return f
}

// playMill opens mission 10 and plays its setup and first reply.
func playMill(t *testing.T, f *FrontEnd) {
	t.Helper()
	openExtendedMission(t, f, 40)
	f.live.advanceNotice(ui.NoticeReply1)
	missionSteps(f.live, 2)
	if got := f.live.world.ScriptGlobals(); !reflect.DeepEqual(got, millWonGlobals) {
		t.Fatalf("the mill left %v, want %v", got, millWonGlobals)
	}
}

// winIntoYard wins the live mission through its acknowledged advance and
// opens the mission the win leads to.
func winIntoYard(t *testing.T, f *FrontEnd) {
	t.Helper()
	f.live.mission.outcome = sim.OutcomeWon
	list := func(...ui.NoticeAction) (ui.NoticeDest, string, ui.MapOpener) { return ui.NoticeToMapList, "", nil }
	dest, line, open := continueMission(frontTransitions{f}, f.liveMission, f.live.mission.state, list)()
	if dest != ui.NoticeToMission || open == nil {
		t.Fatalf("the win went to %v: %s", dest, line)
	}
	if err := f.App("globals yard").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if f.liveMission != 20 {
		t.Fatalf("the win opened mission %d", f.liveMission)
	}
}

// playYard plays mission 20's triggers and its first reply.
func playYard(t *testing.T, f *FrontEnd) {
	t.Helper()
	for range 40 {
		f.live.tick()
	}
	if f.live.mission.dialogue == nil || len(f.live.mission.replies) != 2 {
		t.Fatalf("the yard dialogue offers %d replies", len(f.live.mission.replies))
	}
	f.live.advanceNotice(ui.NoticeReply1)
	missionSteps(f.live, 2)
	if got := f.live.world.ScriptGlobals(); !reflect.DeepEqual(got, yardTipGlobals) {
		t.Fatalf("the yard left %v, want %v", got, yardTipGlobals)
	}
}

// globalsSave is the SAV of f as it stands: a mission save with a live
// mission, else a town save.
func globalsSave(t *testing.T, f *FrontEnd, onMap bool) []byte {
	t.Helper()
	s, label, err := f.Snapshot(onMap)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// savedMark is the mod mark a SAV holds.
func savedMark(t *testing.T, raw []byte) (sav.DocumentData, modMark) {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	mark, ok, err := readModMark(doc)
	if err != nil || !ok {
		t.Fatal("the save holds no readable mod mark", err)
	}
	return doc, mark
}

// millWonTown wins the mill through the public finisher and answers the
// town SAV of that moment, before mission 20 opens.
func millWonTown(t *testing.T, f *FrontEnd) []byte {
	t.Helper()
	playMill(t, f)
	ms := f.live.mission.state
	if next, line := f.FinishMissionWithRoster(10, ms.Party, ms.World, ms.Start.IDs, ms.Start.Roster); next != 20 {
		t.Fatalf("the mill's win answered %d: %s", next, line)
	}
	f.endLive()
	// The fixture's town has not opened; a town SAVE needs one.
	f.Town.open = true
	return globalsSave(t, f, false)
}

// townLoad loads raw into a new front over the mod at dir and opens mission
// 20 from its town.
func townLoad(t *testing.T, dir string, mill, raw []byte) *FrontEnd {
	t.Helper()
	cold := globalsMillBase(t, mill)
	if _, err := launchMods(t, cold, dir); err != nil {
		t.Fatal(err)
	}
	if _, town, err := cold.RestoreOriginal(raw); err != nil || !town {
		t.Fatal("town LOAD", town, err)
	}
	if err := cold.App("globals town").OpenMission(cold.MissionOpener(20)); err != nil {
		t.Fatal(err)
	}
	return cold
}

func TestGlobalsYardMapIsTheBuiltMap(t *testing.T) {
	if !bytes.Equal(readGlobalsMill(t, "maps/yard.almx"), globalsYardMap(t)) {
		t.Fatal("testdata/globals-mill/maps/yard.almx is not the map globalsYardMap builds")
	}
}

// The mill's win carries its globals to mission 20, which AutoGetMission
// opens: the yard's trigger and reply compare them and act. A fresh campaign
// entering mission 20 sees the initials, so neither holds.
func TestGlobalsCampaignWinCarriesToTheNextMap(t *testing.T) {
	f := globalsMillFront(t, globalsMillFolder)
	playMill(t, f)
	winIntoYard(t, f)
	if got := f.Town.modGlobals()[millGlobalsName]; !reflect.DeepEqual(got, millWonGlobals) {
		t.Fatalf("the campaign store holds %v", got)
	}
	if got := f.live.world.ScriptGlobals(); !reflect.DeepEqual(got, millWonGlobals) {
		t.Fatalf("mission 20 seated %v, want %v", got, millWonGlobals)
	}
	playYard(t, f)
	if w := f.live.world; w.ScriptRegister(50) != 1 || w.ScriptRegister(51) != 1 {
		t.Fatalf("the yard's trigger and reply gave variables 50=%d 51=%d", w.ScriptRegister(50), w.ScriptRegister(51))
	}
	fresh := globalsMillFront(t, globalsMillFolder)
	if err := fresh.App("globals fresh").OpenMission(fresh.MissionOpener(20)); err != nil {
		t.Fatal(err)
	}
	if got := fresh.live.world.ScriptGlobals(); !reflect.DeepEqual(got, millInitials) {
		t.Fatalf("a fresh mission 20 seated %v", got)
	}
	for range 40 {
		fresh.live.tick()
	}
	if fresh.live.world.ScriptRegister(50) != 0 || len(fresh.live.mission.replies) != 1 {
		t.Fatalf("at the initials the trigger held or the reply was offered: %d replies", len(fresh.live.mission.replies))
	}
}

// A lost mission takes no win: its writes stay behind. A restart, which the
// mission menu offers once the town has opened, seats the values the mission
// entered with, and the campaign store keeps the last win's.
func TestGlobalsCampaignLossAndRestartDiscard(t *testing.T) {
	lost := globalsMillFront(t, globalsMillFolder)
	playMill(t, lost)
	lost.live.mission.outcome = sim.OutcomeLost
	menu := func(...ui.NoticeAction) (ui.NoticeDest, string, ui.MapOpener) { return ui.NoticeToMenu, "", nil }
	if dest, _, _ := lost.continuity(10, lost.live.mission.state, menu)(ui.NoticeExitMain); dest != ui.NoticeToMenu {
		t.Fatalf("the loss went to %v", dest)
	}
	if lost.Town.modGlobals() != nil {
		t.Fatalf("a loss wrote the store %v", lost.Town.modGlobals())
	}
	f := globalsMillFront(t, globalsMillFolder)
	f.Town.open = true
	playMill(t, f)
	dest, line, open := f.continuity(10, f.live.mission.state, f.live.advanceNotice)(ui.NoticeRestart)
	if dest != ui.NoticeToMission || open == nil {
		t.Fatalf("restart: %v %s", dest, line)
	}
	if err := f.App("globals restart").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if got := f.live.world.ScriptGlobals(); !reflect.DeepEqual(got, millInitials) {
		t.Fatalf("the restarted mill seated %v", got)
	}
	playMill(t, f)
	winIntoYard(t, f)
	playYard(t, f)
	dest, line, open = f.continuity(20, f.live.mission.state, f.live.advanceNotice)(ui.NoticeRestart)
	if dest != ui.NoticeToMission || open == nil {
		t.Fatalf("restart: %v %s", dest, line)
	}
	if err := f.App("globals restart").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if got := f.live.world.ScriptGlobals(); !reflect.DeepEqual(got, millWonGlobals) {
		t.Fatalf("the restarted yard seated %v, want the entry values", got)
	}
	if got := f.Town.modGlobals()[millGlobalsName]; !reflect.DeepEqual(got, millWonGlobals) {
		t.Fatalf("the restart left the store %v", got)
	}
}

// A town SAVE writes the campaign store into the mod mark, and a cold LOAD
// gives it back to the next mission.
func TestGlobalsCampaignTownSaveAndColdLoad(t *testing.T) {
	f := globalsMillFront(t, globalsMillFolder)
	raw := millWonTown(t, f)
	_, mark := savedMark(t, raw)
	want := map[string][]modMarkGlobal{millGlobalsName: {{ID: 1, Value: 1}, {ID: 2, Value: 10}, {ID: 3, Value: math.MinInt32}}}
	if !reflect.DeepEqual(mark.Globals, want) {
		t.Fatalf("the town mark holds %v", mark.Globals)
	}
	cold := townLoad(t, globalsMillFolder, readGlobalsMill(t, "maps/mill.almx"), raw)
	if got := cold.Town.modGlobals()[millGlobalsName]; !reflect.DeepEqual(got, millWonGlobals) {
		t.Fatalf("the loaded store is %v", got)
	}
	if got := cold.live.world.ScriptGlobals(); !reflect.DeepEqual(got, millWonGlobals) {
		t.Fatalf("mission 20 after LOAD seated %v", got)
	}
}

// A mission SAVE writes the entry values into the mod mark and the live ones
// beside the script state: LOAD gives the live values to the World and the
// entry values to the store, and a restart seats the entry values.
func TestGlobalsCampaignMissionSaveLoadAndRestart(t *testing.T) {
	f := globalsMillFront(t, globalsMillFolder)
	playMill(t, f)
	winIntoYard(t, f)
	playYard(t, f)
	raw := globalsSave(t, f, true)
	doc, mark := savedMark(t, raw)
	if got := mark.Globals[millGlobalsName]; len(got) != 3 || got[1].Value != 10 {
		t.Fatalf("the mission mark holds %v, want the entry values", mark.Globals)
	}
	actions, err := readCurrentActions(&doc)
	if err != nil || actions.Program == nil || !reflect.DeepEqual(actions.Program.Globals, yardTipGlobals) {
		t.Fatal("the actions leaf lacks the live values", err)
	}
	cold := globalsMillFront(t, globalsMillFolder)
	open, town, err := cold.RestoreOriginal(raw)
	if err == nil && !town {
		err = cold.App("globals reload").OpenMission(open)
	}
	if err != nil || town {
		t.Fatal("mission LOAD", town, err)
	}
	if got := cold.live.world.ScriptGlobals(); !reflect.DeepEqual(got, yardTipGlobals) || cold.live.world.Hash() != f.live.world.Hash() {
		t.Fatalf("after LOAD the World holds %v", got)
	}
	if got := cold.Town.modGlobals()[millGlobalsName]; !reflect.DeepEqual(got, millWonGlobals) {
		t.Fatalf("after LOAD the store holds %v", got)
	}
	dest, line, reopen := cold.continuity(20, cold.live.mission.state, cold.live.advanceNotice)(ui.NoticeRestart)
	if dest != ui.NoticeToMission || reopen == nil {
		t.Fatalf("restart: %v %s", dest, line)
	}
	if err := cold.App("globals restart").OpenMission(reopen); err != nil {
		t.Fatal(err)
	}
	if got := cold.live.world.ScriptGlobals(); !reflect.DeepEqual(got, millWonGlobals) {
		t.Fatalf("the restart after LOAD seated %v, want the entry values", got)
	}
}

// A save whose mod mark holds no globals, as an older engine save, and a
// save with no mod mark, as an original one admitted under the mods, give a
// store of none: the next mission seats the initials.
func TestGlobalsCampaignOlderAndOriginalSavesTakeInitials(t *testing.T) {
	raw := millWonTown(t, globalsMillFront(t, globalsMillFolder))
	doc, mark := savedMark(t, raw)
	mark.Globals = nil
	payload, err := json.Marshal(mark)
	if err == nil {
		err = sav.SetNativeMods(&doc.State, payload)
	}
	older, err2 := sav.EncodeDocumentData(doc)
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	sav.ClearNativeMods(&doc.State)
	original, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	for name, b := range map[string][]byte{"older": older, "original": original} {
		cold := globalsMillBase(t, readGlobalsMill(t, "maps/mill.almx"))
		if _, err := launchMods(t, cold, globalsMillFolder); err != nil {
			t.Fatal(err)
		}
		cold.Table.Mods.AcceptUnmarked = true
		if _, town, err := cold.RestoreOriginal(b); err != nil || !town {
			t.Fatal(name, "LOAD", town, err)
		}
		if cold.Town.modGlobals() != nil {
			t.Fatalf("%s: the store is %v", name, cold.Town.modGlobals())
		}
		if err := cold.App("globals initials").OpenMission(cold.MissionOpener(20)); err != nil {
			t.Fatal(err)
		}
		if got := cold.live.world.ScriptGlobals(); !reflect.DeepEqual(got, millInitials) {
			t.Fatalf("%s: mission 20 seated %v", name, got)
		}
	}
}

// A town save loaded under a changed declaration keeps the values of the ids
// still declared, gives a new id its initial and drops an id no longer
// declared, which the next SAVE does not write.
func TestGlobalsCampaignChangedDeclaration(t *testing.T) {
	raw := millWonTown(t, globalsMillFront(t, globalsMillFolder))
	changed := []byte("[[global]]\nid = 1\nname = \"goblins_driven\"\n\n[[global]]\nid = 2\nname = \"mill_coins\"\ninitial = 7\n\n" +
		"[[global]]\nid = 4\nname = \"late\"\ninitial = 8\n")
	mill := globalsYardMap(t)
	dir := globalsMillCopy(t, mill, changed)
	cold := townLoad(t, dir, mill, raw)
	if got, want := cold.Town.modGlobals()[millGlobalsName], millWonGlobals[:2]; !reflect.DeepEqual(got, want) {
		t.Fatalf("the admitted store is %v, want %v", got, want)
	}
	want := []sim.ScriptGlobal{{ID: 1, Value: 1}, {ID: 2, Value: 10}, {ID: 4, Value: 8}}
	if got := cold.live.world.ScriptGlobals(); !reflect.DeepEqual(got, want) {
		t.Fatalf("mission 20 seated %v, want %v", got, want)
	}
	_, mark := savedMark(t, globalsSave(t, cold, true))
	if got := mark.Globals[millGlobalsName]; len(got) != 2 || got[1].ID != 2 {
		t.Fatalf("the next SAVE wrote %v", mark.Globals)
	}
}

// The store a mark holds for a mod that is not active passes the reader
// untouched and is written back with the same values.
func TestGlobalsCampaignMissingModKeepsItsValues(t *testing.T) {
	f := globalsMillFront(t, globalsMillFolder)
	raw := millWonTown(t, f)
	doc, _ := savedMark(t, raw)
	store := modGlobalStore{millGlobalsName: millWonGlobals, "gone": {{ID: 7, Value: 42}, {ID: 900, Value: -3}}}
	if err := markDocumentForMods(&doc, f.ModSet().Marked(), nil, nil, store); err != nil {
		t.Fatal(err)
	}
	marked, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	load, err := applyModMark(marked, f.Table.Mods, rules.Rules{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(load.globals, store) {
		t.Fatalf("the reader gave %v, want %v", load.globals, store)
	}
	first, _, _ := sav.NativeMods(doc.State)
	if err := markDocumentForMods(&doc, f.ModSet().Marked(), nil, nil, load.globals); err != nil {
		t.Fatal(err)
	}
	if again, _, _ := sav.NativeMods(doc.State); !bytes.Equal(first, again) {
		t.Fatalf("written back as %s, was %s", again, first)
	}
	// An id outside the range is refused, naming the mod.
	bad := modGlobalStore{"gone": {{ID: 2000, Value: 1}}}
	if err := markDocumentForMods(&doc, f.ModSet().Marked(), nil, nil, bad); err != nil {
		t.Fatal(err)
	}
	if marked, err = sav.EncodeDocumentData(doc); err != nil {
		t.Fatal(err)
	}
	if _, err := applyModMark(marked, f.Table.Mods, rules.Rules{}); err == nil {
		t.Fatal("a mark naming global 2000 was admitted")
	}
}

// A guest chain's win passes its globals to the waiting party: the stash
// the party waits in holds the values before the chain, and the town it
// comes back to holds the chain's.
func TestGlobalsCampaignGuestChainPassesItsValues(t *testing.T) {
	d := campaignFlowRows(t, campaignFlowRow(20, "guest = 22\nchapter = 30\nmap = \"maps/detour.alm\""))
	f := campaignFlowFront(t, d)
	f.Table.Mods.Globals = mod.GlobalData{Globals: []mod.Global{{Mod: "flow", ID: 1, Name: "chain", Initial: 4}}}
	openMissionFlow(t, f, 10)
	winMissionFlow(t, f, 20)
	openMissionFlow(t, f, 20)
	if f.Town.guest == nil || f.Town.guest.stash == nil {
		t.Fatal("mission 20 is not a guest's")
	}
	if got := f.live.world.ScriptGlobals(); !reflect.DeepEqual(got, []sim.ScriptGlobal{{ID: 1, Value: 4}}) {
		t.Fatalf("the guest's mission seated %v", got)
	}
	chain := []sim.ScriptGlobal{{ID: 1, Value: 9}}
	if err := f.live.world.SeatScriptGlobals(chain); err != nil {
		t.Fatal(err)
	}
	winMissionFlow(t, f, 0)
	if f.Town.guest != nil && f.Town.guest.stash != nil {
		t.Fatal("the waiting party did not come back")
	}
	if got := f.Town.modGlobals()["flow"]; !reflect.DeepEqual(got, chain) {
		t.Fatalf("the waiting party's store is %v, want the chain's %v", got, chain)
	}
}

// The same start and steps through the mill, its win and the yard give the
// same globals and World hash.
func TestGlobalsCampaignReplayIsEqual(t *testing.T) {
	run := func() (uint64, []sim.ScriptGlobal) {
		f := globalsMillFront(t, globalsMillFolder)
		playMill(t, f)
		winIntoYard(t, f)
		playYard(t, f)
		return f.live.world.Hash(), f.live.world.ScriptGlobals()
	}
	h1, g1 := run()
	h2, g2 := run()
	if h1 != h2 || !reflect.DeepEqual(g1, g2) {
		t.Fatalf("the replay gave %x %v, then %x %v", h1, g1, h2, g2)
	}
}

// A mod mark without globals is the mark the writer wrote before the store
// existed: the same bytes as the payload without the globals key.
func TestModMarkWithoutGlobalsKeepsItsBytes(t *testing.T) {
	f := globalsMillFront(t, globalsMillFolder)
	openExtendedMission(t, f, 40)
	doc, mark := savedMark(t, globalsSave(t, f, true))
	if mark.Globals != nil {
		t.Fatalf("a first mission's mark holds %v", mark.Globals)
	}
	before := struct {
		Format    int    `json:"format"`
		SetDigest string `json:"set_digest"`
		mod.SetRecord
		Domain []modMarkObject `json:"domain,omitempty"`
		Items  []modMarkItem   `json:"items,omitempty"`
		Layers []modMarkLayer  `json:"layers,omitempty"`
	}{mark.Format, mark.SetDigest, mark.SetRecord, mark.Domain, mark.Items, mark.Layers}
	want, err := json.Marshal(before)
	if err != nil {
		t.Fatal(err)
	}
	if got, _, _ := sav.NativeMods(doc.State); !bytes.Equal(got, want) {
		t.Fatalf("the mark is %s, want %s", got, want)
	}
}
