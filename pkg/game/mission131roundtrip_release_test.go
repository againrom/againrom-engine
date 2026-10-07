package game

import (
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

// A companion restored from an original SAV carries the class its equipment
// selects, and selecting it pushes its composed figure. The hero-band class
// survives only as the fallback for a member whose equipment selects none;
// pushPortrait reads it through unitPicture, whose recorded-figure guard
// still shows the figure for it (DAT-HUMANS-008).
func TestReleasePushPortraitShowsRestoredCompanionFigure(t *testing.T) {
	mw := openRestoredVictory(t, "push portrait companion")
	var companions []sim.EntityID
	for _, id := range mw.mission.ids {
		e, ok := mw.entity(id)
		if !ok || (mw.invSubjectSet && uint32(id) == mw.invSubject.ID) {
			continue
		}
		want := mw.unitFigure(id)
		if want == nil {
			t.Fatalf("party actor %d class %d has no composed figure", id, e.Class)
		}
		if got := mw.unitPicture(id, e.Class); !imagesEqual(got, want) {
			t.Fatalf("party actor %d class %d pushed portrait is not its figure", id, e.Class)
		}
		companions = append(companions, id)
	}
	if len(companions) == 0 {
		t.Fatal("fixture has no non-subject party actor")
	}
	id := companions[0]
	class := giveHeroBandClass(t, mw, id)
	if got, want := mw.unitPicture(id, class), mw.unitFigure(id); want == nil || !imagesEqual(got, want) {
		t.Fatalf("party actor %d hero-band class %d pushed portrait is not its figure", id, class)
	}
}

// The same sack, loaded from a mission SAV this engine itself wrote after
// importing an original save, still draws at the frame its own value
// selects. sackFrameIndex (sacks.go) recomputes from the persisted Sack
// value on every draw and depends on no cached client-side byte, so this
// engine's OWN draw is immune to a stale Sack Token+0x1c regardless of what
// this engine's SAV writer puts there. It says nothing about what the
// original's own client would draw from the bytes this engine writes --
// TestReleaseSackTokenValueRoundTripsFromTheOriginal below pins that.
func TestReleaseRestoredSackDrawsItsOwnValueBand(t *testing.T) {
	_, saved := groundCorpusFile(t, "2026-09-24/exp-engine-lineage/game1017-engine-from-0017.sav", "1c7cb03aecacc2a80b89f1d86be4f9d5bfbadf7214c7eaaa334197ebf0d2e985")
	f := releaseFront(t)
	a := f.App("restored sack frame")
	a.Layout(1024, 768)
	open, _, err := f.RestoreOriginal(saved)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	mw := f.live
	var found *sim.Sack
	for _, s := range mw.world.Sacks() {
		if s.X == 69 && s.Y == 35 {
			sCopy := s
			found = &sCopy
		}
	}
	if found == nil {
		t.Fatal("fixture has no sack at cell (69,35)")
	}
	if got := sackFrameIndex(*found); got != 5 {
		t.Fatalf("sack at (69,35) drew frame %d, want band 5 (its value is a six-figure armor price)", got)
	}
}

// The owner's authority for this rule: in the original client, an owner drop
// of a valuable item onto the ground at (34,116), followed by the original's
// own SAVE and LOAD, kept the sack drawing large. The original's own LOAD
// therefore restores the drawn frame from the persisted Sack Token+0x1c
// (ITEM-SACK-010: gold plus the sum of every carried item's own price), not
// from any live-only client cache, so a SAV this engine writes with 0 there
// is the whole cause of the owner's "every sack draws smallest" report on a
// file this engine produced. Three cells round-trip: (69,35), (123,86) and
// (34,116), the owner's own drop.
func TestReleaseSackTokenValueRoundTripsFromTheOriginal(t *testing.T) {
	_, payload := groundCorpusFile(t, "2026-09-24/game0002-bigsack.sav", "8842212e16cbb97ad05af059bdd637df2ab777b4c7f5bb5430353b68b5312c51")
	source, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	before, present, err := source.GroundSacks()
	if err != nil || !present || len(before) != 3 {
		t.Fatalf("fixture sacks=%d present=%t err=%v, want 3", len(before), present, err)
	}
	f := releaseFront(t)
	a := f.App("sack token round trip")
	a.Layout(1024, 768)
	open, _, err := f.RestoreOriginal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	s, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	written, _, err := f.playerMissionSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	out, err := sav.Open(written)
	if err != nil {
		t.Fatal(err)
	}
	after, present, err := out.GroundSacks()
	if err != nil || !present || len(after) != 3 {
		t.Fatalf("round trip sacks=%d present=%t err=%v, want 3", len(after), present, err)
	}
	byCell := func(list []sav.GroundSack) map[uint16]sav.GroundSack {
		m := make(map[uint16]sav.GroundSack, len(list))
		for _, sack := range list {
			m[sack.Cell] = sack
		}
		return m
	}
	beforeByCell, afterByCell := byCell(before), byCell(after)
	for cell, want := range beforeByCell {
		got, ok := afterByCell[cell]
		if !ok {
			t.Fatalf("round trip lost the sack at cell %d", cell)
		}
		if got.Token1C != want.Token1C {
			t.Fatalf("cell %d: T1C=%d, want the original's own %d (gold %d)", cell, got.Token1C, want.Token1C, want.Gold)
		}
	}
}

// A hero LOADed from a SAV this engine itself wrote must keep drawing its own
// party body, not the NPC class an equipment-derived lookup assigns a placed
// human. FrontEnd.Snapshot's CurrentRoster used to restate the whole entry
// party a second time (resume.go), so restoreCurrentPartyMembers handed
// missionAppearanceArt a Start.Roster that, on THIS engine's own reload,
// contained the party's own ids -- and that function's roster loop has no
// party exclusion, so it overwrote every one of them with NPC art. Loading
// the original directly never exercises that path, so it never showed the
// defect; only a round trip through this engine's own writer does.
//
// units.Bodies and units.Classes are two DIFFERENT maps for two different
// questions (units.go): a party member's own picture is a Bodies entry, a
// placed human's is a Classes entry, and the loader never shares one pointer
// between them even when their Name text happens to match. Each FrontEnd
// loads its own UnitSet from the install, so a class pointer is never shared
// across two FrontEnd values either; membership in the reload's OWN Bodies
// map is what this test can check, and it is exactly the fact the defect
// broke.
func TestReleaseHeroDrawsItsOwnBodyAfterOurSaveAndLoad(t *testing.T) {
	_, saved := groundCorpusFile(t, "2026-09-24/game0017-victory.sav", "a30de01ec3d96d58a3a244b9a536a3e49c04a4d45098cb828dabb908756488c3")

	isPartyBody := func(units *terrain.UnitSet, class *terrain.UnitClass) bool {
		if class == nil {
			return false
		}
		for _, body := range units.Bodies {
			if body == class {
				return true
			}
		}
		return false
	}

	baseline := releaseFront(t)
	ab := baseline.App("hero body baseline")
	ab.Layout(1024, 768)
	openBaseline, _, err := baseline.RestoreOriginal(saved)
	if err != nil {
		t.Fatal(err)
	}
	if err := ab.OpenMission(openBaseline); err != nil {
		t.Fatal(err)
	}
	partyIDs := append([]sim.EntityID(nil), baseline.live.mission.ids...)
	if len(partyIDs) == 0 {
		t.Fatal("fixture opened with no party")
	}
	for _, id := range partyIDs {
		if !isPartyBody(baseline.live.units, baseline.live.art[id]) {
			t.Fatalf("loading game0017 directly: party actor %d art %q is not its own body",
				id, releaseArtName(baseline.live, id))
		}
	}

	f := releaseFront(t)
	a := f.App("hero body round trip")
	a.Layout(1024, 768)
	open, _, err := f.RestoreOriginal(saved)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	s, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	written, _, err := f.playerMissionSave(s, label)
	if err != nil {
		t.Fatal(err)
	}

	g := releaseFront(t)
	ag := g.App("hero body round trip reload")
	ag.Layout(1024, 768)
	reopen, _, err := g.RestoreOriginal(written)
	if err != nil {
		t.Fatal(err)
	}
	if err := ag.OpenMission(reopen); err != nil {
		t.Fatal(err)
	}
	for _, id := range partyIDs {
		if !isPartyBody(g.live.units, g.live.art[id]) {
			t.Fatalf("after our SAVE+LOAD: party actor %d art %q is not its own body, want a party body as loading game0017 directly gives",
				id, releaseArtName(g.live, id))
		}
	}
}
