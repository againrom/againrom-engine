package game

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const enemyCardMission, enemyCardRow = 10, 96

func enemyCardWant(count uint32) int { return min(int(count>>1), 7) }

func enemyCardOpen(t *testing.T) (*FrontEnd, *ui.App, *mapWorld) {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	app := f.App("enemy card")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	if err := app.OpenMission(f.DirectNewGame(enemyCardMission)); err != nil {
		t.Fatal(err)
	}
	live := f.live
	for k := 0; k < 16 && live.mission.open; k++ {
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	return f, app, live
}

// enemyCardInstances are the entities of the placements of one Units row, in
// placement order; an entity is found by the map unit id its placement carries.
func enemyCardInstances(f *FrontEnd, live *mapWorld, row int) []sim.EntityID {
	var out []sim.EntityID
	for _, u := range live.mission.state.Map.Units {
		if r := mapload.Resolve(u, f.Table); !r.Found() || r.Index != row {
			continue
		}
		for _, e := range live.world.Entities() {
			if e.MapUnitID == u.UnitID && u.UnitID != 0 {
				out = append(out, e.ID)
			}
		}
	}
	return out
}

// enemyCardSubject hovers one unit and returns the card the viewer composes for
// it, with the presentation fog opened so the pointer may rest on it.
func enemyCardSubject(t *testing.T, app *ui.App, live *mapWorld, id sim.EntityID) (ui.PanelSubject, []string) {
	t.Helper()
	for i := range live.fog.visible {
		live.fog.visible[i], live.fog.explored[i] = 1, 1
	}
	live.push()
	e, ok := live.entity(id)
	if !ok {
		t.Fatalf("entity %d is absent", id)
	}
	inspectionCentre(live, int(e.X), int(e.Y))
	releaseHoverInspection(t, app, live, ui.InspectionSubject{Kind: ui.InspectionUnit, ID: uint32(id)})
	s, ok := live.view.InspectionPanel()
	statement, ok2 := live.view.PanelStatement()
	if !ok || !ok2 {
		t.Fatalf("no card for entity %d", id)
	}
	return s, statement
}

// enemyCardHover points at one unit and returns the card of the unit the
// pointer actually rests on, which another body may cover.
func enemyCardHover(app *ui.App, live *mapWorld, id sim.EntityID) (ui.PanelSubject, sim.EntityID, bool) {
	for i := range live.fog.visible {
		live.fog.visible[i], live.fog.explored[i] = 1, 1
	}
	live.push()
	e, ok := live.entity(id)
	if !ok {
		return ui.PanelSubject{}, 0, false
	}
	inspectionCentre(live, int(e.X), int(e.Y))
	x, y, err := live.view.InspectionPoint(ui.InspectionSubject{Kind: ui.InspectionUnit, ID: uint32(id)})
	if err != nil || app.HeadlessPointer("hover", x, y) != nil {
		return ui.PanelSubject{}, 0, false
	}
	got, ok := live.view.Inspection()
	if !ok || got.Kind != ui.InspectionUnit {
		return ui.PanelSubject{}, 0, false
	}
	s, ok := live.view.InspectionPanel()
	return s, sim.EntityID(got.ID), ok
}

func mustEntity(t *testing.T, live *mapWorld, id sim.EntityID) sim.Entity {
	t.Helper()
	e, ok := live.entity(id)
	if !ok {
		t.Fatalf("entity %d is absent", id)
	}
	return e
}

func enemyCardCount(w *sim.World, row int) uint32 {
	for _, d := range w.SavedDiaries() {
		if d.Owner.Player {
			for _, e := range d.Entries {
				if e.Index == row {
					return e.Count
				}
			}
		}
	}
	return 0
}

func enemyCardSetCount(w *sim.World, row int, count uint32) {
	var entries []sim.SavedDiaryEntry
	if count != 0 {
		entries = []sim.SavedDiaryEntry{{Index: row, Count: count, Remaining: uint16(1024 - count)}}
	}
	w.SetSavedDiaries([]sim.SavedDiary{{Owner: sim.SavedDiaryOwner{Player: true}, Length: 119, Entries: entries}})
}

// enemyCardKill has the hero kill one placement through the ordinary attack
// order. The hero is healed each tick so the neighbours cannot end the run.
func enemyCardKill(t *testing.T, live *mapWorld, victim sim.EntityID) {
	t.Helper()
	w, hero := live.world, live.mission.ids[0]
	v, ok := live.entity(victim)
	if !ok {
		t.Fatalf("victim %d is absent", victim)
	}
	if err := w.HeadlessPlace(hero, v.X-1, v.Y); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 30000; i++ {
		if i%300 == 0 {
			live.pending = append(live.pending, sim.Attack(hero, victim))
		}
		if h, _ := live.entity(hero); h.HP < h.MaxHP {
			_ = w.HeadlessHeal(hero)
		}
		live.tick()
		if e, ok := live.entity(victim); !ok || e.HP < 0 {
			return
		}
	}
	t.Fatalf("hero did not kill entity %d", victim)
}

func enemyCardAliveInstances(live *mapWorld, ids []sim.EntityID) (out []sim.EntityID) {
	for _, id := range ids {
		if e, ok := live.entity(id); ok && e.Alive() {
			out = append(out, id)
		}
	}
	return out
}

// enemyCardRows lists the Units rows the card level reads: rows from 64 whose
// type is 64..80, found through the same typeID and face search the install's
// own table builder is described by.
func enemyCardRows(f *FrontEnd) (rows map[[2]int32]int, types map[int32]bool) {
	rows, types = map[[2]int32]int{}, map[int32]bool{}
	for typeID := int32(64); typeID <= 80; typeID++ {
		for face := int32(1); face <= 4; face++ {
			if r := data.FindUnit(f.Table.Units, typeID, face); r >= 64 {
				rows[[2]int32{typeID, face}] = r
				types[typeID] = true
			}
		}
	}
	return rows, types
}

func TestReleaseEnemyCardUnitsTableHasTheAdmittedRows(t *testing.T) {
	f := releaseFront(t)
	rows, types := enemyCardRows(f)
	seen := map[int]bool{}
	for key, r := range rows {
		if seen[r] {
			t.Errorf("row %d is named by two (type, face) keys, one of them %v", r, key)
		}
		seen[r] = true
	}
	if f.Table.Units.Len() != 119 || len(rows) != 53 || len(types) != 14 {
		t.Fatalf("Units length %d, admitted rows %d, types %d; want 119, 53, 14", f.Table.Units.Len(), len(rows), len(types))
	}
}

// Each count of one monster row gives that row's level on every instance and
// on no other row; the card draws the groups of its level.
func TestReleaseEnemyCardLevelFollowsTheDiaryCountPerRow(t *testing.T) {
	f, app, live := enemyCardOpen(t)
	w := live.world
	same := enemyCardInstances(f, live, enemyCardRow)
	other := enemyCardInstances(f, live, 100)
	if len(same) < 2 || len(other) == 0 {
		t.Fatalf("mission %d has %d instances of row %d and %d of row 100", enemyCardMission, len(same), enemyCardRow, len(other))
	}
	own := live.mission.ids[0]
	prev, prevLevel := 0, -1
	for count := uint32(0); count <= 17; count++ {
		enemyCardSetCount(w, enemyCardRow, count)
		want := enemyCardWant(count)
		first, statement := enemyCardSubject(t, app, live, same[0])
		second, _ := enemyCardSubject(t, app, live, same[1])
		neighbour, _ := enemyCardSubject(t, app, live, other[0])
		hero, _ := enemyCardSubject(t, app, live, own)
		if first.DetailLevel != want || second.DetailLevel != want {
			t.Fatalf("count %d: instance levels %d and %d, want %d", count, first.DetailLevel, second.DetailLevel, want)
		}
		if neighbour.DetailLevel != 0 || hero.DetailLevel != 7 {
			t.Fatalf("count %d: other row level %d, own unit level %d, want 0 and 7", count, neighbour.DetailLevel, hero.DetailLevel)
		}
		joined := strings.Join(statement, " ")
		health := fmt.Sprintf("%d/%d", first.HP, first.MaxHP)
		if strings.Contains(joined, health) != (want >= 1) {
			t.Fatalf("count %d level %d: health pair %q in %q", count, want, health, joined)
		}
		if want != prevLevel {
			if prevLevel >= 0 && want <= 6 && len(joined) <= prev {
				t.Fatalf("level %d card %q draws no more than level %d", want, joined, prevLevel)
			}
			prev, prevLevel = len(joined), want
		} else if len(joined) != prev {
			t.Fatalf("count %d: the card changed within level %d", count, want)
		}
	}
}

// Shift+F4 draws every group while it is on and then returns the card, the
// World, the Diary and the written state to what they were.
func TestReleaseEnemyCardShiftF4ShowsEveryGroupForDisplayOnly(t *testing.T) {
	f, app, live := enemyCardOpen(t)
	w := live.world
	ids := enemyCardInstances(f, live, enemyCardRow)
	enemyCardSetCount(w, enemyCardRow, 2)
	before, beforeStatement := enemyCardSubject(t, app, live, ids[0])
	if before.DetailLevel != 1 {
		t.Fatalf("level %d before the chord, want 1", before.DetailLevel)
	}
	hash, diaries := w.Hash(), w.SavedDiaries()
	state := missionAutosaveSampleNow(t, f).State
	for edge := range 2 {
		if err := app.HeadlessKey("shift-f4"); err != nil {
			t.Fatal(err)
		}
		s, statement := enemyCardSubject(t, app, live, ids[0])
		if edge == 0 {
			if s.DetailLevel != 7 || len(strings.Join(statement, " ")) <= len(strings.Join(beforeStatement, " ")) {
				t.Fatalf("revealed card level %d statement %v", s.DetailLevel, statement)
			}
		} else if !reflect.DeepEqual(s, before) || !reflect.DeepEqual(statement, beforeStatement) {
			t.Fatalf("card after the second chord differs:\n%+v\n%+v", s, before)
		}
		if w.Hash() != hash || !reflect.DeepEqual(w.SavedDiaries(), diaries) || !reflect.DeepEqual(state, missionAutosaveSampleNow(t, f).State) {
			t.Fatal("the display chord changed the World, the Diary or the saved state")
		}
	}
}

// Real kills raise the count, SAVE writes it, a cold LOAD restores it and the
// next kill continues it. Two loss controls prove the check can fail.
func TestReleaseEnemyCardKnowledgeSurvivesSaveAndLoad(t *testing.T) {
	f, _, live := enemyCardOpen(t)
	w := live.world
	ids := enemyCardInstances(f, live, enemyCardRow)
	for kills := uint32(1); kills <= 2; kills++ {
		enemyCardKill(t, live, enemyCardAliveInstances(live, ids)[0])
		if got := enemyCardCount(w, enemyCardRow); got != kills {
			t.Fatalf("after %d kill(s) the Diary count is %d", kills, got)
		}
	}
	if l := w.KnowledgeLevel(mustEntity(t, live, enemyCardAliveInstances(live, ids)[0])); l != 1 {
		t.Fatalf("level %d after two kills, want 1", l)
	}
	store, _, raw := campaignSave(t, f, true)
	if got := enemyCardSAVCount(t, raw, enemyCardRow); got != 2 {
		t.Fatalf("the written SAV carries count %d for the row, want 2", got)
	}

	cold, coldApp := campaignCold(t, store)
	if got := enemyCardCount(cold.live.world, enemyCardRow); got != 2 {
		t.Fatalf("cold LOAD restored count %d, want 2", got)
	}
	coldIDs := enemyCardInstances(cold, cold.live, enemyCardRow)
	if err := coldApp.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	alive := enemyCardAliveInstances(cold.live, coldIDs)
	if len(alive) < 2 {
		t.Fatal("too few living instances after LOAD")
	}
	if s, _, ok := enemyCardHover(coldApp, cold.live, alive[0]); ok && s.DetailLevel != 1 {
		t.Fatalf("cold LOAD card level %d, want 1", s.DetailLevel)
	}
	if l := cold.live.world.KnowledgeLevel(mustEntity(t, cold.live, alive[0])); l != 1 {
		t.Fatalf("cold LOAD level %d, want 1", l)
	}
	enemyCardKill(t, cold.live, alive[0])
	if got := enemyCardCount(cold.live.world, enemyCardRow); got != 3 {
		t.Fatalf("the next kill after LOAD gives count %d, want 3", got)
	}
	enemyCardKill(t, cold.live, enemyCardAliveInstances(cold.live, coldIDs)[0])
	next := enemyCardAliveInstances(cold.live, coldIDs)[0]
	if l := cold.live.world.KnowledgeLevel(mustEntity(t, cold.live, next)); l != 2 || enemyCardCount(cold.live.world, enemyCardRow) != 4 {
		t.Fatalf("after four kills level %d count %d, want 2 and 4", l, enemyCardCount(cold.live.world, enemyCardRow))
	}

	// Loss control: a SAVE written without the Diary loads with no knowledge.
	w.SetSavedDiaries(nil)
	lossStore, _, lossRaw := campaignSave(t, f, true)
	if got := enemyCardSAVCount(t, lossRaw, enemyCardRow); got != 0 {
		t.Fatalf("the dropped-Diary SAVE still carries count %d", got)
	}
	lossCold, _ := campaignCold(t, lossStore)
	if got := enemyCardCount(lossCold.live.world, enemyCardRow); got != 0 {
		t.Fatalf("a SAVE without the Diary loaded count %d", got)
	}

	// Loss control: the first SAV with its Diary zeroed after the write loads at
	// zero, so LOAD reads the Diary from the file.
	zeroDir := t.TempDir()
	entries, err := os.ReadDir(store.Dir)
	if err != nil || len(entries) != 1 {
		t.Fatal("store contents", entries, err)
	}
	if err := os.WriteFile(filepath.Join(zeroDir, entries[0].Name()), enemyCardZeroSAV(t, raw), 0600); err != nil {
		t.Fatal(err)
	}
	zeroCold, _ := campaignCold(t, SaveStore{Dir: zeroDir})
	if got := enemyCardCount(zeroCold.live.world, enemyCardRow); got != 0 {
		t.Fatalf("LOAD of the zeroed SAV holds count %d", got)
	}
}

// enemyCardLocalDiary is the local Player's Diary record in a decoded SAV.
func enemyCardLocalDiary(t *testing.T, doc *sav.DocumentData) *sav.DocumentRecordData {
	t.Helper()
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Player" {
			continue
		}
		if participant, err := savedStructureValue(r, "Participant"); err != nil || participant != 0 {
			continue
		}
		for j := range r.Inline {
			if r.Inline[j].Record.Class == "Diary" {
				return &r.Inline[j].Record
			}
		}
	}
	t.Fatal("the SAV has no local Player Diary")
	return nil
}

func enemyCardSAVDiary(t *testing.T, raw []byte) sav.Diary {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	d, err := sav.ReadDocumentDiary(*enemyCardLocalDiary(t, &doc))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func enemyCardSAVCount(t *testing.T, raw []byte, row int) uint32 {
	t.Helper()
	for _, e := range enemyCardSAVDiary(t, raw).Entries {
		if e.Index == row {
			return e.Count
		}
	}
	return 0
}

func enemyCardZeroSAV(t *testing.T, raw []byte) []byte {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	r := enemyCardLocalDiary(t, &doc)
	d, err := sav.ReadDocumentDiary(*r)
	if err != nil {
		t.Fatal(err)
	}
	next, err := sav.ProjectDocumentDiary(*r, sav.Diary{Length: d.Length, Self: d.Self})
	if err != nil {
		t.Fatal(err)
	}
	*r = next
	out, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(out, raw) {
		t.Fatal("zeroing the Diary changed no byte")
	}
	return out
}

// Original SAVs with a nonzero Diary load with the levels the table builder
// gives from their counts: one nibble per (type, face), min(count>>1, 7).
func TestReleaseEnemyCardOriginalSAVDiaryLevels(t *testing.T) {
	dir := os.Getenv("AGAINROM_SAVE_CORPUS")
	if dir == "" {
		t.Skip("AGAINROM_SAVE_CORPUS is not set")
	}
	for _, name := range []string{"game0002-bigsack", "game0017-victory"} {
		t.Run(name, func(t *testing.T) {
			matches, _ := filepath.Glob(filepath.Join(dir, "*", name+".sav"))
			if len(matches) == 0 {
				t.Skipf("%s is not in the save corpus", name)
			}
			raw, err := os.ReadFile(matches[0])
			if err != nil {
				t.Fatal(err)
			}
			f := releaseFront(t)
			app, _ := openOriginalSAVApp(t, f, raw, name+".sav")
			if err := app.HeadlessKey("0"); err != nil {
				t.Fatal(err)
			}
			live := f.live
			counts := map[int]uint32{}
			for _, e := range enemyCardSAVDiary(t, raw).Entries {
				counts[e.Index] = e.Count
			}
			rows, _ := enemyCardRows(f)
			levelOf := func(typeID int, face uint8) int {
				row, ok := rows[[2]int32{int32(typeID), int32(face)}]
				if !ok {
					return 0
				}
				return enemyCardWant(counts[row])
			}
			levels := map[int]int{}
			for _, e := range live.world.Entities() {
				if !e.Alive() || e.Owner == sim.SelfSlot || e.TypeID < 64 || e.TypeID > 80 {
					continue
				}
				want := levelOf(int(e.TypeID), e.SourceBinding.Face)
				if got := live.world.KnowledgeLevel(e); got != want {
					t.Fatalf("entity %d type %d face %d: level %d, table nibble %d", e.ID, e.TypeID, e.SourceBinding.Face, got, want)
				}
				levels[want]++
			}
			// The same level reaches the viewer's entity data, which the card reads.
			drawn := map[sim.EntityID]ui.MapEntity{}
			for _, d := range live.entityDraws() {
				drawn[sim.EntityID(d.ID)] = d
			}
			checked := 0
			for _, e := range live.world.Entities() {
				if !e.Alive() || e.Owner == sim.SelfSlot || e.TypeID < 64 || e.TypeID > 80 {
					continue
				}
				d, ok := drawn[e.ID]
				if !ok {
					continue
				}
				checked++
				if want := levelOf(int(e.TypeID), e.SourceBinding.Face); !d.KnowledgeKnown || d.Knowledge != want {
					t.Fatalf("entity %d drawn at level %d (known %v), want %d", e.ID, d.Knowledge, d.KnowledgeKnown, want)
				}
			}
			if checked == 0 {
				t.Fatal("no creature reached the viewer's entity data")
			}
			if len(levels) < 2 {
				t.Fatalf("creature levels %v show no spread", levels)
			}
			t.Logf("%s: creature levels %v", name, levels)
			// Loss control: a world that ignored the SAV Diary reports no knowledge.
			live.world.SetSavedDiaries(nil)
			for _, e := range live.world.Entities() {
				if e.Alive() && e.Owner != sim.SelfSlot && e.TypeID >= 64 && e.TypeID <= 80 && live.world.KnowledgeLevel(e) != 0 {
					t.Fatal("a world with no Diary still reports knowledge")
				}
			}
		})
	}
}

// A won mission hands its Diary to the next mission of the same campaign; a new
// campaign starts without one.
func TestReleaseEnemyCardKnowledgeCarriesToTheNextMission(t *testing.T) {
	f, app, live := enemyCardOpen(t)
	enemyCardSetCount(live.world, enemyCardRow, 5)
	state := live.mission.state
	if successor, message := f.FinishMission(enemyCardMission, state.Party, live.world, state.Start.IDs); successor < 0 {
		t.Fatal("mission return refused:", message)
	}
	if err := app.OpenMission(f.MissionOpener(20)); err != nil {
		t.Fatal(err)
	}
	if got := enemyCardCount(f.live.world, enemyCardRow); got != 5 {
		t.Fatalf("the next mission opened with count %d, want 5", got)
	}
	if err := app.OpenMission(f.DirectNewGame(20)); err != nil {
		t.Fatal(err)
	}
	if got := enemyCardCount(f.live.world, enemyCardRow); got != 0 {
		t.Fatalf("a new campaign opened with count %d, want 0", got)
	}
}
