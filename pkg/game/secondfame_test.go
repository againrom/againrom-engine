package game

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/formats/fame"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

const (
	secondFameXP     = 62653614
	secondFameEvents = 19
	secondFameTime   = 1944
	secondFameScore  = 61235
)

// secondFameHero carries experience summing to secondFameXP.
func secondFameHero() secondOriginalHero {
	h := secondMissionFixtureHero()
	h.xp = [5]uint32{12531468, 12531139, 12533993, 12527285, 12529729}
	h.experience = secondFameXP
	return h
}

// secondFameFile is a mission file at the campaign's last stage whose tail
// holds the two Hall words.
func secondFameFile(t *testing.T, hero secondOriginalHero, time, events uint32) []byte {
	t.Helper()
	tail := secondSavedTail(0, [2]uint32{10, placeMission})
	binary.LittleEndian.PutUint32(tail[4:], time)
	binary.LittleEndian.PutUint32(tail[8:], events)
	binary.LittleEndian.PutUint32(tail[12+4*768:], uint32(secondEndStage(t)))
	return labelledContainer(sav.SecondGame.Magic, "mission", secondSavedDocument(10, 44, &hero, true),
		append(secondMissionStore(4, 0, -1), tail...))
}

// secondFameWin takes the loaded mission through the win's one route at
// subtick and returns the hall file's rows after the ending.
func secondFameWin(t *testing.T, f *FrontEnd, subtick uint32) []fame.Record {
	t.Helper()
	f.Campaign = resolved(Campaign{EndStage: secondEndStage(t)}, nil)
	f.hallStore = fameStore{Dir: t.TempDir()}
	w := f.live.world
	fameSetClock(t, w, subtick)
	hero := secondMissionActor(t, w)
	if _, refused := f.finishWin(winInput{n: 10, party: f.LiveParty(), w: w, ids: []sim.EntityID{hero.ID}, seam: true, won: true}); refused != "" {
		t.Fatalf("the win refused: %s", refused)
	}
	if _, complete := f.campaignEnding(); !complete {
		t.Fatal("the ending did not open")
	}
	if _, complete := f.campaignEnding(); !complete {
		t.Fatal("the ending did not reopen")
	}
	rows, err := f.hallStore.read()
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

// The second game's score is the one builder's arithmetic: 61235 for the
// values of one EN run (R2-ENGINE-1090), the other branch at Hall +4 zero
// (R2-ENGINE-1066).
func TestSecondFameScoreTakesBothBranches(t *testing.T) {
	if got := earnedFameScore(secondFameTime, secondFameEvents, secondFameXP); got != secondFameScore {
		t.Fatalf("score %d, want %d", got, secondFameScore)
	}
	if got := earnedFameScore(0, secondFameEvents, secondFameXP); got != 2380 {
		t.Fatalf("Hall +4 zero: score %d, want 2380", got)
	}
}

// A LOAD seats the file's Hall words, the last mission's Victory adds its
// time and writes one row, and that row stays recorded through a SAVE.
func TestSecondFameLoadedCampaignWritesOneRow(t *testing.T) {
	f, _, _ := secondMissionLoadFile(t, secondFameFile(t, secondFameHero(), 0, secondFameEvents))
	if !f.fame.Known || f.fame.Time != 0 || f.fame.Events != secondFameEvents || f.fame.Result != nil {
		t.Fatalf("loaded fame %+v", f.fame)
	}
	rows := secondFameWin(t, f, 31104)
	if f.fame.Time != secondFameTime {
		t.Fatalf("time %d, want %d", f.fame.Time, secondFameTime)
	}
	if len(rows) != 1 || rows[0].Name != "Gildarius" || rows[0].Score != secondFameScore {
		t.Fatalf("hall %+v", rows)
	}
	if f.fame.Result == nil || !f.fame.Result.Recorded {
		t.Fatalf("result %+v", f.fame.Result)
	}
	held := cloneFame(f.fame)
	s := Snapshot{game: base.GameROM2, Fame: &held}
	snapshotTown(f.Town, &s, f.campaign())
	if s.second.Kept.PairTail != [2]uint32{secondFameTime, secondFameEvents} {
		t.Fatalf("saved words %v", s.second.Kept.PairTail)
	}
	saved := fameFromSnapshot(s)
	if saved.Result == nil || !saved.Result.Recorded || saved.Result.Score != secondFameScore {
		t.Fatalf("snapshot fame %+v", saved)
	}
	cold := fameFromCurrentTown(nil, &currentSessionData{Game: base.GameROM2, Second: s.second, Fame: &currentFamePolicy{Known: true,
		Result: &currentFameResult{[]byte(saved.Result.Name), saved.Result.Score, saved.Result.Recorded}}})
	if !cold.Known || cold.Time != secondFameTime || cold.Events != secondFameEvents || cold.Result == nil || !cold.Result.Recorded {
		t.Fatalf("session fame %+v result %+v", cold, cold.Result)
	}
	f.fame = cold
	if _, complete := f.campaignEnding(); !complete {
		t.Fatal("the ending did not reopen after the session came back")
	}
	if rows, err := f.hallStore.read(); err != nil || len(rows) != 1 {
		t.Fatalf("a recorded result wrote again: %+v %v", rows, err)
	}
}

// A file whose Hall words are zero still earns its row; the score is the
// formula's over those zeros.
func TestSecondFameUnknownInputsStillWriteARow(t *testing.T) {
	f, _, _ := secondMissionLoadFile(t, secondFameFile(t, secondFameHero(), 0, 0))
	rows := secondFameWin(t, f, 0)
	if len(rows) != 1 || rows[0].Name != "Gildarius" || rows[0].Score != 0 {
		t.Fatalf("hall %+v", rows)
	}
	noTail := fameFromCurrentTown(nil, &currentSessionData{Game: base.GameROM2, Second: &currentSecondCampaign{}, Fame: &currentFamePolicy{}})
	if !noTail.Known || noTail.Time != 0 || noTail.Events != 0 {
		t.Fatalf("a session with no tail %+v", noTail)
	}
}

// A SAVE takes the two words from the session's counters and leaves the
// pair list and the location bytes as they were; no loaded tail gives the
// new game's pair list.
func TestSecondFameSaveWritesTheCurrentWords(t *testing.T) {
	seat := func(c *secondCampaign, fame *SnapshotFame) Snapshot {
		town := NewTown(Campaign{})
		town.second = c
		s := Snapshot{Fame: fame}
		snapshotTown(town, &s, secondCampaignRules{})
		return s
	}
	loaded := newSecondCampaign()
	loaded.kept = &secondTailKept{Pairs: [][2]uint32{{3, 4}}, PairTail: [2]uint32{1, 2}}
	s := seat(loaded, &SnapshotFame{Known: true, Time: secondFameTime, Events: secondFameEvents})
	tail, err := secondCampaignTail(s)
	if err != nil {
		t.Fatal(err)
	}
	if tail.PairTail != [2]uint32{secondFameTime, secondFameEvents} || len(tail.Pairs) != 1 || tail.Pairs[0] != [2]uint32{3, 4} {
		t.Fatalf("tail pairs %v words %v", tail.Pairs, tail.PairTail)
	}
	if loaded.kept.PairTail != [2]uint32{1, 2} {
		t.Fatalf("the snapshot wrote the live campaign's words: %v", loaded.kept.PairTail)
	}
	if zero := seat(newSecondCampaign(), &SnapshotFame{Known: true}); zero.second.Kept != nil {
		t.Fatalf("a new game with no counters kept %+v", zero.second.Kept)
	}
	fresh := seat(newSecondCampaign(), &SnapshotFame{Known: true, Time: 7, Events: 8})
	tail, err = secondCampaignTail(fresh)
	if err != nil {
		t.Fatal(err)
	}
	if tail.PairTail != [2]uint32{7, 8} || len(tail.Pairs) != 1 || tail.Pairs[0] != [2]uint32{1, 1} {
		t.Fatalf("new game tail pairs %v words %v", tail.Pairs, tail.PairTail)
	}
}

// A first-game session states no second campaign: its unknown history still
// earns no row and its known history keeps its counters.
func TestFirstGameFameKeepsItsResult(t *testing.T) {
	unknown := fameFromCurrentTown(nil, &currentSessionData{Fame: &currentFamePolicy{}})
	if unknown.Known || unknown.Time != 0 || unknown.Events != 0 {
		t.Fatalf("unknown history %+v", unknown)
	}
	c := Campaign{Main: []int{10}, Last: map[int]bool{10: true}}
	hero := mapload.PartyMember{ID: "hero", Name: "Hero", PlayerCharacter: true, StartingHero: true}
	run := func(history SnapshotFame) SnapshotFame {
		w := fameWorld(t, sim.Relations{}, sim.Entity{ID: 9, X: 3, Y: 1, Owner: 1, HP: 100, MaxHP: 100, SkillXP: [6]int32{1000000}})
		f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil)}, CampaignSession: CampaignSession{Town: NewTown(c), fame: history}}
		f.FinishMission(10, []mapload.PartyMember{hero}, w, []sim.EntityID{9})
		return f.fame
	}
	if got := run(unknown); got.Result != nil {
		t.Fatalf("unknown history earned %+v", got.Result)
	}
	got := run(SnapshotFame{Known: true, Time: 100, Events: 4})
	if got.Result == nil || got.Result.Score != 4000 || got.Time != 100 {
		t.Fatalf("known history %+v result %+v", got, got.Result)
	}
}
