package game

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// secondObjectiveSave writes the live mission through the save page as
// name.sav in dir and answers its bytes.
func secondObjectiveSave(t *testing.T, f *FrontEnd, dir, name string) []byte {
	t.Helper()
	prepared, err := f.SaveDialogSeams(SaveStore{Dir: dir}, OriginalStore{}).Prepare(ui.SaveRequest{
		OnMap: true, Directory: dir, Name: name, Label: name, Format: ui.SaveSAV})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := prepared.Commit(true); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, name+".sav"))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// objectiveSlotsOf is slots 752..767 of a bank.
func objectiveSlotsOf(bank [1024]int32) [16]int32 {
	var out [16]int32
	copy(out[:], bank[752:768])
	return out
}

// liveObjectiveSlots is the objective slots of f's live mission World.
func liveObjectiveSlots(t *testing.T, f *FrontEnd) [16]int32 {
	t.Helper()
	if f.live == nil {
		t.Fatal("no live mission")
	}
	bank, ok := f.live.world.ROM2ScenarioState()
	if !ok {
		t.Fatal("the mission World has no scenario bank")
	}
	return objectiveSlotsOf(bank)
}

// A second-game mission SAVE carries the live objective slots in the
// campaign tail's bank, the bank the original's LOAD restores raw
// (R2-SESSION-031, R2-SESSION-348), and both LOADs restore them: the
// engine's own document and the file's records alone. The mission entry
// cleared the campaign's stale slots first. A changed slot reaches the next
// SAVE while the first file keeps its own value (loss control), and a slot
// changed in the file's bytes reaches the restored World. Fails while the
// tail carries the campaign's bank of the departure.
func TestSecondMissionSaveCarriesTheLiveObjectiveSlots(t *testing.T) {
	f, app, screen, _ := secondBagFront(t, nil)
	dir := t.TempDir()
	f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	f.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, func() time.Time { return time.Unix(100, 0) })
	f.Town.second.bank[753], f.Town.second.bank[760] = 5, 9
	secondBagEnter(t, f, app, screen)
	if got := liveObjectiveSlots(t, f); got != ([16]int32{}) {
		t.Fatalf("the mission entry left objective slots %v", got)
	}
	bank, _ := f.live.world.ROM2ScenarioState()
	sim.WriteObjective(&bank, 1, 1)
	sim.WriteObjective(&bank, 2, 2)
	sim.WriteObjective(&bank, 3, 4)
	f.live.world.SetROM2ScenarioState(bank)
	want := liveObjectiveSlots(t, f)

	first := secondObjectiveSave(t, f, dir, "first")
	doc := checkSecondMissionFile(t, first, 10)
	if got := objectiveSlotsOf(doc.SecondTail.Bank); got != want {
		t.Fatalf("the tail carries objective slots %v, want the live %v", got, want)
	}
	if got := liveObjectiveSlots(t, secondMissionLoadBytes(t, first)); got != want {
		t.Fatalf("the engine document's LOAD restores %v, want %v", got, want)
	}
	bare := withoutEngineDocument(t, first)
	if got := liveObjectiveSlots(t, secondMissionLoadBytes(t, bare)); got != want {
		t.Fatalf("the records' LOAD restores %v, want %v", got, want)
	}

	bank[754] = 0
	sim.WriteObjective(&bank, 2, 4)
	f.live.world.SetROM2ScenarioState(bank)
	changed := liveObjectiveSlots(t, f)
	second := checkSecondMissionFile(t, secondObjectiveSave(t, f, dir, "second"), 10)
	if got := objectiveSlotsOf(second.SecondTail.Bank); got != changed || changed[2] != 5 {
		t.Fatalf("the second SAVE carries %v, want the changed %v", got, changed)
	}
	if got := liveObjectiveSlots(t, secondMissionLoadBytes(t, bare)); got[2] != 3 {
		t.Fatalf("the first file's LOAD now restores slot 754 = %d, want its own 3", got[2])
	}

	// The tail's bank is 4096 raw bytes in the file (R2-SESSION-031).
	var image [4096]byte
	for i, v := range doc.SecondTail.Bank {
		binary.LittleEndian.PutUint32(image[4*i:], uint32(v))
	}
	at := bytes.Index(bare, image[:])
	if at < 0 || bytes.Index(bare[at+1:], image[:]) >= 0 {
		t.Fatal("the records' file does not hold the tail's bank once")
	}
	patched := bytes.Clone(bare)
	binary.LittleEndian.PutUint32(patched[at+4*756:], 7)
	got := liveObjectiveSlots(t, secondMissionLoadBytes(t, patched))
	want[4] = 7
	if got != want {
		t.Fatalf("the patched file's LOAD restores %v, want %v", got, want)
	}
	if _, err := sav.DecodeDocumentData(patched); err != nil {
		t.Fatal(err)
	}
}
