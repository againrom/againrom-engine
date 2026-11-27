package game

import (
	"testing"

	"againrom/pkg/sim"
)

func TestFallRuleDecidesTheGuardedCharacterLoss(t *testing.T) {
	for _, tc := range []struct {
		fall fallRule
		lost bool
	}{{fallDefeats, true}, {fallReturns, false}} {
		entities := []sim.Entity{{ID: 1, Owner: sim.SelfSlot, HP: 10, MaxHP: 10},
			{ID: 2, Owner: sim.SelfSlot, HP: -10001, MaxHP: 10, TypeID: sim.HumanTypeID, Humanoid: true}}
		mw := partyWorld(t, heroWorld(t, nil, nil, entities), townCompanionParty(), []sim.EntityID{1, 2})
		mw.mission.fall = tc.fall
		if _, lost := mw.guardedCharacterLost(); lost != tc.lost {
			t.Fatalf("fall rule %d: lost %v, want %v", tc.fall, lost, tc.lost)
		}
	}
}

func TestEntryOutpostValuesDecideTheDeclaration(t *testing.T) {
	address, ok := MissionMap(10)
	if !ok {
		t.Fatal("no map for mission 10")
	}
	declared := func(fill mapEntry) int {
		e := newEntryFixture(t, &recordingAudio{})
		r := e.request()
		r.fill = fill
		if _, err := enterMission(r, e.ports); err != nil {
			t.Fatalf("enterMission: %v", err)
		}
		return 0
	}
	calls := 0
	campaign := campaignEntry()
	campaign.outposts = func(*sim.World) sim.OutpostMap { calls++; return sim.OutpostMap{} }
	declared(campaign)
	if calls != 1 {
		t.Fatalf("campaign entry with values declared %d times, want 1", calls)
	}
	s := &mapSession{entry: MapEntry{Source: address}, end: func(*Mission) string { return "" }}
	session := sessionEntry(s)
	session.mapFile = mapInContainer
	session.outposts = nil
	declared(session)
	if calls != 1 {
		t.Fatalf("session entry without values declared, calls %d", calls)
	}
}
