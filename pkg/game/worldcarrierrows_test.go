package game

import (
	"errors"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// worldCarrierRow is the row of the set whose home begins with prefix.
func worldCarrierRow(t *testing.T, prefix string) worldCarrier {
	t.Helper()
	var found []worldCarrier
	for _, c := range worldCarrierSet() {
		if strings.HasPrefix(c.home, prefix) {
			found = append(found, c)
		}
	}
	if len(found) != 1 {
		t.Fatalf("the set holds %d rows of %q", len(found), prefix)
	}
	return found[0]
}

// Each carrier that kept its own LOAD, seat or writer outside the set is a
// row: the Diaries, the dead bodies, the town purse and the LOAD of the
// carrier absence.
func TestTheLeftCarriersAreRowsOfTheSet(t *testing.T) {
	diary := worldCarrierRow(t, "World saved Diaries")
	if diary.decode[archiveLiving] == nil || diary.decode[documentRecord] == nil || diary.seat[entrySeat] == nil || diary.writer == nil || diary.stage != graphStage {
		t.Fatal("the Diary row does not decode from the archive and the document, seat and write on the graph")
	}
	dead := worldCarrierRow(t, "World original dead bodies")
	if dead.decode[archiveDead] == nil || dead.writer == nil || dead.stage != graphStage {
		t.Fatal("the dead body row does not decode from the archive and write on the graph")
	}
	if purse := worldCarrierRow(t, "World purse"); purse.seat[entrySeat] == nil || purse.seat[lineSeat] == nil {
		t.Fatal("the purse row does not seat at a mission door and at the headless line start")
	}
	absence := worldCarrierRow(t, "World carrier absence")
	if absence.decode[actionsRecord] == nil || absence.writer == nil || absence.stage != keysStage {
		t.Fatal("the carrier absence row does not decode from the actions leaf and write after the keys")
	}
}

// A fresh mission's seat, run through the set, puts the town's purse and the
// Diaries the town knows into the World; a mission whose loaded record binds
// the Player purses keeps its purse.
func TestTheSetSeatsTheTownPurseAndDiaries(t *testing.T) {
	known := []sim.SavedDiary{{Owner: sim.SavedDiaryOwner{Player: true}, Length: 3, Entries: []sim.SavedDiaryEntry{{Index: 1, Count: 2, Remaining: 3}}}}
	town := &Town{gold: 70, knowledge: known}
	w := secondGameScriptWorld(t, nil, nil)
	if err := seatWorldCarriers(entrySeat, carrierSeat{r: missionRequest{town: town}, rules: firstCampaignRules{}, ms: &Mission{World: w}}); err != nil {
		t.Fatal(err)
	}
	if got := w.Purse(sim.SelfSlot); got != 70 {
		t.Fatalf("the seat put purse %d, want 70", got)
	}
	if got := w.SavedDiaries(); len(got) != 1 || got[0].Owner != known[0].Owner || len(got[0].Entries) != 1 {
		t.Fatalf("the seat put Diaries %+v", got)
	}
	bound := secondGameScriptWorld(t, nil, nil)
	bound.SetPurse(sim.SelfSlot, 5)
	ms := &Mission{World: bound, savedDocument: &SnapshotSAVDocument{PlayerPurses: &SnapshotSAVPlayerPurses{}}}
	if err := seatWorldCarriers(entrySeat, carrierSeat{r: missionRequest{town: town}, rules: firstCampaignRules{}, ms: ms}); err != nil {
		t.Fatal(err)
	}
	if got := bound.Purse(sim.SelfSlot); got != 5 {
		t.Fatalf("the seat replaced a bound purse with %d", got)
	}
}

// AdvanceLine's mission start seats the town purse through the set's line
// seat and nothing else: its World is the started mission given the town
// gold, as the start set the purse by itself, and the town's Diaries stay out.
func TestTheLineStartSeatsOnlyTheTownPurse(t *testing.T) {
	f := advanceFront(t)
	f.Town = NewTown(f.Campaign.Value())
	f.Town.gold = 511
	f.Town.knowledge = []sim.SavedDiary{{Owner: sim.SavedDiaryOwner{Player: true}, Length: 3, Entries: []sim.SavedDiaryEntry{{Index: 1, Count: 2, Remaining: 3}}}}
	ms, err := f.startLineMission(10, openDifficulty)
	if err != nil {
		t.Fatal(err)
	}
	want, err := StartMission(f.Archives.Containers, 10, f.Table, openDifficulty, MissionParty(f.StartWeapon.Value(), f.Bodies, f.Table))
	if err != nil {
		t.Fatal(err)
	}
	want.World.SetPurse(sim.SelfSlot, 511)
	if got, base := ms.World.Hash(), want.World.Hash(); got != base || got != lineStartHash {
		t.Fatalf("the line start's World hash is %016x, the start given the purse %016x, pinned %016x", got, base, lineStartHash)
	}
	if got := ms.World.Purse(sim.SelfSlot); got != 511 {
		t.Fatalf("the line start's purse is %d, want 511", got)
	}
	if got := ms.World.SavedDiaries(); len(got) != 0 {
		t.Fatalf("the line start seated the town's Diaries %+v", got)
	}
}

// lineStartHash is the World hash of the fixture's mission 10 given purse
// 511, as AdvanceLine's start set the purse before the line seat.
const lineStartHash uint64 = 0xebce66e3005d76dc

// The mission document's graph stage, run through the set, writes the World's
// Diaries: a live Player's Diary the document has no record for refuses the
// SAVE, and a World with no Diary and no dead body writes nothing.
func TestTheGraphStageWritesTheDiaries(t *testing.T) {
	state := &SnapshotSAVDocument{Version: snapshotSAVDocumentVersion, Document: &sav.DocumentData{}}
	w := secondGameScriptWorld(t, nil, nil)
	if err := writeWorldCarriers(carrierSave{s: &Snapshot{}, state: state, world: w}, graphStage); err != nil {
		t.Fatalf("an empty graph stage refuses: %v", err)
	}
	w.SetSavedDiaries([]sim.SavedDiary{{Owner: sim.SavedDiaryOwner{Player: true}, Length: 1}})
	err := writeWorldCarriers(carrierSave{s: &Snapshot{}, state: state, world: w}, graphStage)
	var unsupported *worldSaveUnsupportedError
	if !errors.As(err, &unsupported) || !strings.Contains(err.Error(), "Diary") {
		t.Fatalf("the graph stage wrote a Diary with no record: %v", err)
	}
}
