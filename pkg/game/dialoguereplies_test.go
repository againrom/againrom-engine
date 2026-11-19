package game

import (
	"os"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// dialogueFixture is the one dialogue file the tests serve.
func dialogueFixture(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("../mod/testdata/dialogue/elder.toml")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// dialogueActions are the action node ids of the fixture: 12 sets variable 20,
// 13 pays the player a hundred gold, 14 sets variable 21.
func dialogueActions() []sim.ScriptInstant {
	return []sim.ScriptInstant{
		{Op: sim.ScriptInstantMessage, Args: [10]int32{4}},
		{Op: sim.ScriptInstantSetVariable, Args: [10]int32{20, 7}},
		{Op: sim.ScriptInstantGiveMoney, Args: [10]int32{100}, HasPlayer: true, Player: 1},
		{Op: sim.ScriptInstantSetVariable, Args: [10]int32{21, 9}},
	}
}

// interactiveMission opens the fixture driver on an interactive dialogue whose
// replies 3 and 4 test condition 3 against condition ref (4 holds, 5 fails).
func interactiveMission(t *testing.T, ref string) (*mapWorld, *ui.Viewer) {
	t.Helper()
	file := strings.ReplaceAll(dialogueFixture(t), "right = 5", "right = "+ref)
	file = strings.ReplaceAll(file, "mark = 140", "mark = 40")
	return interactiveMissionFile(t, file)
}

func interactiveMissionFile(t *testing.T, file string) (*mapWorld, *ui.Viewer) {
	t.Helper()
	w := missionWorld(t, dialogueActions(), []sim.ScriptTrigger{missionTrigger(3, 1, true, 0)})
	src := missionSource{dialogueFilePath(7, "elder"): []byte(file)}
	m := worldFixtureMap()
	v := worldFixtureViewer(t, m)
	v.SetFont(missionFont())
	mw := openMission(&Mission{
		Number: 7, Map: m, World: w,
		Raises:    []mapload.ScriptRaise{{Latch: 3, Event: mapload.InteractiveEvent(0)}},
		Dialogues: []string{"elder"},
		ActionOf:  map[uint32]int32{12: 1, 13: 2, 14: 3},
		CheckOf:   map[uint32]int32{3: 0, 4: 1, 5: 2},
	}, nil, nil, v, src, nil, nil)
	missionSteps(mw, 2)
	if _, _, open := v.NoticeState(); !open {
		t.Fatal("no dialogue opened")
	}
	return mw, v
}

func TestInteractiveDialogueShowsItsOfferedReplies(t *testing.T) {
	mw, v := interactiveMission(t, "4")
	if text, kind, _ := v.NoticeState(); kind != ui.NoticeDialogue || !strings.HasPrefix(text, "Strangers!") {
		t.Fatalf("the first node shows %q", text)
	}
	if v.NoticeReplyCount() != 4 || len(mw.mission.replies) != 4 {
		t.Fatalf("rows %d, replies %d, want 4", v.NoticeReplyCount(), len(mw.mission.replies))
	}
	mw.advanceNotice(ui.NoticeAdvance)
	if v.NoticeReplyCount() != 4 || len(mw.pending) != 0 {
		t.Fatalf("a plain advance moved the page: rows %d pending %v", v.NoticeReplyCount(), mw.pending)
	}
}

func TestFailingConditionHidesOrGreysTheReplyAsItsFieldSays(t *testing.T) {
	mw, v := interactiveMission(t, "5")
	if v.NoticeReplyCount() != 3 {
		t.Fatalf("rows %d, want 3: the hidden reply is left out and the disabled one stays", v.NoticeReplyCount())
	}
	if v.NoticeReplyOff(1) || v.NoticeReplyOff(2) || !v.NoticeReplyOff(3) {
		t.Errorf("greyed rows: %v %v %v, want only the third", v.NoticeReplyOff(1), v.NoticeReplyOff(2), v.NoticeReplyOff(3))
	}
	mw.advanceNotice(ui.NoticeReply3)
	if len(mw.pending) != 0 || v.NoticeReplyCount() != 3 {
		t.Errorf("a greyed reply was chosen: pending %v rows %d", mw.pending, v.NoticeReplyCount())
	}
}

func TestChoosingAReplyRunsItsOutcomeThroughTheCommand(t *testing.T) {
	mw, v := interactiveMission(t, "4")
	gold := mw.world.Purse(1)
	mw.advanceNotice(ui.NoticeReply3)
	if _, _, open := v.NoticeState(); open {
		t.Fatal("a reply with no next node left the dialogue open")
	}
	if len(mw.pending) != 1 || mw.pending[0] != sim.RunInstant(2) {
		t.Fatalf("pending %v, want the one run-instant", mw.pending)
	}
	if mw.world.Purse(1) != gold {
		t.Fatal("the gold changed before the world stepped")
	}
	missionSteps(mw, 1)
	if got := mw.world.Purse(1); got != gold+100 {
		t.Errorf("gold %d, want %d", got, gold+100)
	}
	mw, _ = interactiveMission(t, "4")
	mw.advanceNotice(ui.NoticeReply4)
	missionSteps(mw, 1)
	if got := mw.world.ScriptRegister(21); got != 9 {
		t.Errorf("variable 21 is %d, want 9", got)
	}
}

func TestOnceReplyIsNotOfferedAgainAndNextPagesTheNode(t *testing.T) {
	mw, v := interactiveMission(t, "4")
	mw.advanceNotice(ui.NoticeReply2)
	if text, _, open := v.NoticeState(); !open || text != "A hundred gold." || v.NoticeReplyCount() != 0 {
		t.Fatalf("after reply 2: %q open %v rows %d", text, open, v.NoticeReplyCount())
	}
	mw.advanceNotice(ui.NoticeAdvance)
	if text, _, _ := v.NoticeState(); !strings.HasPrefix(text, "Strangers!") || v.NoticeReplyCount() != 3 {
		t.Fatalf("the first node came back as %q with %d rows, want 3 without the once reply", text, v.NoticeReplyCount())
	}
	missionSteps(mw, 1)
	if got := mw.world.ScriptRegister(40); got != 1 {
		t.Errorf("the once mark is %d, want 1", got)
	}
}

func TestReplayOfTheChoiceGivesTheSameWorldHash(t *testing.T) {
	run := func(reply ui.NoticeAction) uint64 {
		mw, _ := interactiveMission(t, "4")
		if reply != ui.NoticeAdvance {
			mw.advanceNotice(reply)
		}
		missionSteps(mw, 2)
		return mw.world.Hash()
	}
	a, b, other, none := run(ui.NoticeReply3), run(ui.NoticeReply3), run(ui.NoticeReply4), run(ui.NoticeAdvance)
	if a != b {
		t.Errorf("the same choice gave hashes %x and %x", a, b)
	}
	if a == other || a == none {
		t.Errorf("the choice did not reach the world hash: %x %x %x", a, other, none)
	}
}

func TestLegacyDialogueKeepsItsWorldHash(t *testing.T) {
	run := func(payload string) uint64 {
		mw, _ := missionDialogue(t, payload, nil)
		mw.advanceNotice()
		mw.advanceNotice()
		missionSteps(mw, 2)
		return mw.world.Hash()
	}
	plain := "<part=1>\r\nhello\r\n<part=2>\r\nagain\r\n"
	if a, b := run(plain), run(plain); a != b {
		t.Errorf("a legacy dialogue hashed %x and %x", a, b)
	}
	mw, v := missionDialogue(t, plain, nil)
	if v.NoticeReplyCount() != 0 || mw.mission.dialogue != nil {
		t.Error("a legacy dialogue opened as an interactive one")
	}
}

const savedDialogue = `[[node]]
id = 1
en = "Hello."

[[reply]]
en = "Take the offer."
once = true
mark = 140
do = [12]

[[reply]]
en = "Walk away."
`

// interactiveExtendedMap is an extended map whose one trigger raises the
// interactive dialogue "elder" once variable 120 is 1.
func interactiveExtendedMap(t *testing.T) []byte {
	t.Helper()
	actions := []scriptNode{
		{label: "Drop location", opcode: dropActionOpcode, id: 99, values: []uint32{16, 16}, tags: []uint32{5, 6}},
		{label: "Set120", opcode: 3, id: 1, values: []uint32{120, 1}, tags: []uint32{1, 1}},
		{label: "elder", opcode: 2, id: 20, values: []uint32{1, 1}, tags: []uint32{1, 1}},
		{label: "Set141", opcode: 3, id: 12, values: []uint32{141, 3}, tags: []uint32{1, 1}},
	}
	triggers := []alm.ScriptTrigger{
		{Name: "Setup", Once: 1, Acts: []uint32{1}},
		{Name: "Ask", Once: 1, Group: []uint32{0}, Left: []uint32{1}, Right: []uint32{2}, Cmp: []uint32{0}, Acts: []uint32{20}},
	}
	b := synth.ALM(synth.ALMOptions{Width: 40, Height: 40, Units: extendedUnits, Type7Payload: scriptPayload(actions, extendedConditions)})
	out, err := alm.AppendScriptRecord(b, alm.ScriptRecord{Variables: 150, Triggers: triggers})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestOnceReplyStaysUsedAfterSaveAndColdLoad(t *testing.T) {
	m := interactiveExtendedMap(t)
	front := func() *FrontEnd {
		f := extendedScriptFront(t, m)
		f.Archives.Containers.Overlay(dialogueFilePath(10, "elder"), func() ([]byte, error) { return []byte(savedDialogue), nil })
		return f
	}
	f := front()
	w := openExtendedMission(t, f, 40)
	mw := f.live
	if mw.mission.dialogue == nil || len(mw.mission.replies) != 2 {
		t.Fatalf("the interactive dialogue did not open with two replies: %+v", mw.mission.replies)
	}
	mw.advanceNotice(ui.NoticeReply1)
	missionSteps(mw, 1)
	if w.ScriptRegister(140) != 1 || w.ScriptRegister(141) != 3 {
		t.Fatalf("the choice left variables 140 and 141 at %d and %d", w.ScriptRegister(140), w.ScriptRegister(141))
	}
	snapshot, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sav.DecodeDocumentData(raw); err != nil {
		t.Fatalf("the save is not a SAV: %v", err)
	}
	cold := front()
	open, town, err := cold.RestoreOriginal(raw)
	if err == nil && !town {
		err = cold.App("dialogue reload").OpenMission(open)
	}
	if err != nil || town {
		t.Fatal("LOAD", town, err)
	}
	if got := cold.live.world.ScriptRegister(140); got != 1 {
		t.Fatalf("the once mark after LOAD is %d, want 1", got)
	}
	if !cold.live.openInteractive(0) || len(cold.live.mission.replies) != 1 || cold.live.mission.replies[0].text != "Walk away." {
		t.Errorf("the used reply is offered again after LOAD: %+v", cold.live.mission.replies)
	}
}

func TestMissingLanguageShowsTheOtherLanguageWithoutRefusingTheFile(t *testing.T) {
	file := `[[node]]
id = 1
ru = "Привет."

[[reply]]
ru = "Да."

[[reply]]
en = "No."
ru = "Нет."
`
	mw, v := interactiveMissionFile(t, file)
	if text, _, _ := v.NoticeState(); text != EncodeModText("Привет.", 0) {
		t.Errorf("the node shows %q, want the other language's text", text)
	}
	if len(mw.mission.replies) != 2 || mw.mission.replies[0].text != EncodeModText("Да.", 0) || mw.mission.replies[1].text != "No." {
		t.Errorf("replies %+v", mw.mission.replies)
	}
}
