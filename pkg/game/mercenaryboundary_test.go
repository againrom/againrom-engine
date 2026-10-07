package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

const (
	boundaryHiredType    = 3
	boundarySecondType   = 5
	boundaryUnhiredType  = 7
	boundaryCappedType   = 8
	boundaryUnknownType  = 9
	boundaryHiredCap     = 5
	boundarySecondCap    = 4
	boundaryUnhiredCap   = 3
	boundaryCappedCap    = 2
	boundaryUnhiredStart = 1
)

// mercenaryBoundaryMission is continuityMission's shape with a squad standing
// beside the party's own people: two men of one hired type, one of them dead,
// one man of a second hired type, a player character and a temporary member who
// was never hired.
//
// THE DEAD ONE IS THE POINT of the first arm. For a hired type the pool becomes
// the live tally outright, so a man who died is gone from the pool for good;
// a tally that counted party entries rather than live entities would report two
// where the mission ended with one.
//
// THE TEMPORARY MEMBER IS THE POINT of the second. Temporary and Hired are two
// different fields with two different populations, and a cull written against
// the wrong one passes every test that carries only mercenaries.
func mercenaryBoundaryMission(t *testing.T, n int) (*Mission, []sim.EntityID) {
	t.Helper()

	s, err := sim.NewScript(missionChecks(),
		[]sim.ScriptInstant{{Op: sim.ScriptInstantWin}},
		[]sim.ScriptTrigger{missionTrigger(3, 1, true, 0)})
	if err != nil {
		t.Fatalf("NewScript: %v", err)
	}
	ents := []sim.Entity{
		{ID: 0, X: 10, Y: 10, HP: 20, MaxHP: 20, Owner: sim.SelfSlot,
			Domain: sim.DomainGround, TypeID: sim.HumanTypeID,
			SkillXP: continuityXP, GainsXP: true},
		{ID: 1, X: 11, Y: 10, HP: 15, MaxHP: 15, Owner: sim.SelfSlot,
			Domain: sim.DomainGround, TypeID: 24},
		{ID: 2, X: 12, Y: 10, HP: -1, MaxHP: 15, Owner: sim.SelfSlot,
			Domain: sim.DomainGround, TypeID: 24},
		{ID: 3, X: 13, Y: 10, HP: 15, MaxHP: 15, Owner: sim.SelfSlot,
			Domain: sim.DomainGround, TypeID: 19},
		{ID: 4, X: 14, Y: 10, HP: 18, MaxHP: 18, Owner: sim.SelfSlot,
			Domain: sim.DomainGround, TypeID: sim.HumanTypeID, GainsXP: true},
	}
	w, err := sim.NewScriptedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
		sim.ModeCanonical, make([]byte, worldFixtureW*worldFixtureH), ents, s)
	if err != nil {
		t.Fatalf("NewScriptedWorld: %v", err)
	}
	for i := 0; i < 64 && w.Outcome() == sim.OutcomeUndecided; i++ {
		sim.Step(w, nil)
	}
	if w.Outcome() != sim.OutcomeWon {
		t.Fatalf("fixture outcome = %v, want won", w.Outcome())
	}

	ids := []sim.EntityID{0, 1, 2, 3, 4}
	return &Mission{
		Number: n, World: w,
		Party: []mapload.PartyMember{
			{ID: "hero", StartingHero: true, PlayerCharacter: true, Class: 100},
			{ID: "merc:3:1", MercenaryType: boundaryHiredType, Class: 24},
			{ID: "merc:3:2", MercenaryType: boundaryHiredType, Class: 24},
			{ID: "merc:5:1", MercenaryType: boundarySecondType, Class: 19},
			{ID: "ally", Temporary: true, PlayerCharacter: true, Class: 101},
		},
		Start: mapload.Start{IDs: ids},
	}, ids
}

// mercenaryBoundaryFront is continuityFront with the tavern economy set up by
// hand: two hired types whose working pools remain saved, one unhired type
// under its original size, one unhired type already at it, and one type the
// campaign never authored.
func mercenaryBoundaryFront(t *testing.T) *FrontEnd {
	t.Helper()
	f := continuityFront(t)
	tw := f.Town

	tw.mercCapacity = [16]int{}
	tw.mercPool = [16]int{}
	tw.mercHired = [16]bool{}

	tw.mercCapacity[boundaryHiredType] = boundaryHiredCap
	tw.mercPool[boundaryHiredType] = 2
	tw.mercHired[boundaryHiredType] = true

	tw.mercCapacity[boundarySecondType] = boundarySecondCap
	tw.mercPool[boundarySecondType] = 1
	tw.mercHired[boundarySecondType] = true

	tw.mercCapacity[boundaryUnhiredType] = boundaryUnhiredCap
	tw.mercPool[boundaryUnhiredType] = boundaryUnhiredStart

	tw.mercCapacity[boundaryCappedType] = boundaryCappedCap
	tw.mercPool[boundaryCappedType] = boundaryCappedCap

	return f
}

// TestFinishMissionDischargesEveryHiredMercenary is the owner's report at the
// party tier: a mercenary is not in the party the next mission opens with.
//
// It asserts what STAYS as well as what goes. A cull that emptied the party, or
// one written against Temporary instead of MercenaryType, would satisfy the
// first half alone.
func TestFinishMissionDischargesEveryHiredMercenary(t *testing.T) {
	f := mercenaryBoundaryFront(t)
	ms, ids := mercenaryBoundaryMission(t, 10)

	f.FinishMission(10, ms.Party, ms.World, ids)

	var got []string
	for _, p := range f.Carried {
		if p.Hired() {
			t.Errorf("%s crossed the boundary as mercenary type %d", p.ID, p.MercenaryType)
		}
		got = append(got, p.ID)
	}
	if len(got) != 2 || got[0] != "hero" || got[1] != "ally" {
		t.Errorf("carried party = %v, want the player character and the temporary ally in order", got)
	}
}

// TestFinishMissionMergesTheHiredPoolFromTheLiveTally is the first merge arm.
// Two men of the hired type walked in and one walked out, so the pool becomes 1
// rather than 2: for a hired type the dead are gone from the pool for good.
func TestFinishMissionMergesTheHiredPoolFromTheLiveTally(t *testing.T) {
	f := mercenaryBoundaryFront(t)
	ms, ids := mercenaryBoundaryMission(t, 10)

	f.FinishMission(10, ms.Party, ms.World, ids)

	if got := f.Town.MercenaryPool(boundaryHiredType); got != 1 {
		t.Errorf("hired type %d pool = %d, want 1: two walked in, one died",
			boundaryHiredType, got)
	}
	if got := f.Town.MercenaryPool(boundarySecondType); got != 1 {
		t.Errorf("hired type %d pool = %d, want 1: its one man survived",
			boundarySecondType, got)
	}
	if got := f.Town.MercenaryCapacity(boundaryHiredType); got != boundaryHiredCap {
		t.Errorf("hired type %d capacity = %d, want the pristine %d: the merge writes the pool, not the original size",
			boundaryHiredType, got, boundaryHiredCap)
	}
}

// TestFinishMissionGrowsAnUnhiredSquadByOneUpToItsOriginalSize is the second
// merge arm, and the reason a wiped squad rebuilds only while it is left at
// home. A type the campaign never authored has capacity 0 and does not grow,
// which is the arm a bare increment would get wrong.
func TestFinishMissionGrowsAnUnhiredSquadByOneUpToItsOriginalSize(t *testing.T) {
	f := mercenaryBoundaryFront(t)
	ms, ids := mercenaryBoundaryMission(t, 10)

	f.FinishMission(10, ms.Party, ms.World, ids)

	if got, want := f.Town.MercenaryPool(boundaryUnhiredType), boundaryUnhiredStart+1; got != want {
		t.Errorf("unhired type %d pool = %d, want %d", boundaryUnhiredType, got, want)
	}
	if got := f.Town.MercenaryPool(boundaryCappedType); got != boundaryCappedCap {
		t.Errorf("unhired type %d pool = %d, want it held at its original size %d",
			boundaryCappedType, got, boundaryCappedCap)
	}
	if got := f.Town.MercenaryPool(boundaryUnknownType); got != 0 {
		t.Errorf("unauthored type %d pool = %d, want 0", boundaryUnknownType, got)
	}
}

// TestFinishMissionClearsEveryHireFlag is the third statement of the merge: no
// hire survives a mission. It is what makes the type hireable again in the town
// the mission ends in, against a pool the same call just wrote.
func TestFinishMissionClearsEveryHireFlag(t *testing.T) {
	f := mercenaryBoundaryFront(t)
	ms, ids := mercenaryBoundaryMission(t, 10)

	f.FinishMission(10, ms.Party, ms.World, ids)

	for typ := 1; typ < 16; typ++ {
		if f.Town.MercenaryHired(typ) {
			t.Errorf("type %d is still hired after the mission", typ)
		}
	}
}

// TestFinishMissionTalliesBeforeItCulls is the ordering, stated on its own
// because it is the half that can be silently wrong: both functions can be
// correct and the boundary still lose every squad if the cull runs first.
//
// The instrument is the difference between the two arms. If the tally ran after
// the cull it would count zero live men of the hired type, and the first arm
// writes the tally outright, so the pool would be 0 rather than 1. The unhired
// type's pool is asserted beside it, because a merge that did not run at all
// would leave BOTH unchanged and look the same from the hired type alone.
func TestFinishMissionTalliesBeforeItCulls(t *testing.T) {
	f := mercenaryBoundaryFront(t)
	ms, ids := mercenaryBoundaryMission(t, 10)

	f.FinishMission(10, ms.Party, ms.World, ids)

	if got := f.Town.MercenaryPool(boundaryHiredType); got == 0 {
		t.Errorf("hired type %d pool = 0: the tally saw no live man, so it ran after the cull",
			boundaryHiredType)
	}
	if got, want := f.Town.MercenaryPool(boundaryUnhiredType), boundaryUnhiredStart+1; got != want {
		t.Errorf("unhired type %d pool = %d, want %d: the merge did not run at all",
			boundaryUnhiredType, got, want)
	}
}

// TestFinishMissionWithNoWorldLeavesTheTavernEconomyAlone is the nil-world
// caller: cmd/screenshot, cmd/shopdump and cmd/plaquescreens open a town
// without playing a mission. With no world there is no live population to
// count, and an all-zero tally would wipe every hired pool, so the merge is
// skipped rather than run.
func TestFinishMissionWithNoWorldLeavesTheTavernEconomyAlone(t *testing.T) {
	f := mercenaryBoundaryFront(t)
	f.Town.mercPool[boundaryHiredType] = 4

	f.FinishMission(10, nil, nil, nil)

	if got := f.Town.MercenaryPool(boundaryHiredType); got != 4 {
		t.Errorf("hired type %d pool = %d, want the 4 it held: a nil world has no tally",
			boundaryHiredType, got)
	}
	if !f.Town.MercenaryHired(boundaryHiredType) {
		t.Errorf("type %d was un-hired by a boundary that could not tally", boundaryHiredType)
	}
}
