package mapload_test

import (
	"reflect"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestCurrentPlayersConstructInitialPolicyWithoutCommandGroups(t *testing.T) {
	m := &alm.Map{Width: 8, Height: 8,
		Groups: []alm.Group{{Participant: 77}, {Participant: 0xf1234567}},
		Units:  []alm.Unit{{Owner: 0}, {Owner: 5}}}
	for _, party := range [][]mapload.PartyMember{nil, {{ID: "hero", PlayerCharacter: true}}} {
		w, _, err := mapload.StartMission(m, nil, mapload.DifficultyNormal, party)
		if err != nil {
			t.Fatal(err)
		}
		players := []sim.SavedGroupPlayer{{ID: 1, Slot: 0}, {ID: 2, Slot: 1}, {ID: 3, Slot: 2}, {ID: 4, Slot: 5}}
		if got, present := w.CurrentPlayers(); !present || !reflect.DeepEqual(got, players) {
			t.Fatal("fresh Player IDs/Slots did not reach current state", got, present)
		}
		want := []sim.PlayerParticipant{{PlayerID: 1, Value: 1}, {PlayerID: 2, Value: 77}, {PlayerID: 3, Value: 0xf1234567}, {PlayerID: 4, Value: 1}}
		if got, present := w.PlayerParticipants(); !present || !reflect.DeepEqual(got, want) {
			t.Fatal("fresh Participant did not use explicit map/owned/fallback policy", got)
		}
		m.Groups[1].Participant++
		if got, _ := w.PlayerParticipants(); !reflect.DeepEqual(got, want) {
			t.Fatal("constructed Participant still aliases input Map")
		}
		m.Groups[1].Participant--
		// A first-game World holds the mission-start Group registry (AI-449);
		// the Player constructor adds nothing to it.
		if _, _, present := w.SavedGroups(); !present || !w.GroupsConstructed() {
			t.Fatal("Player constructor changed the constructed Group registry")
		}
	}
	w, err := mapload.FromALMWith(m, &mapload.Table{UnitKeys: mapload.ServerUnitKeys, SpellArms: mapload.SecondGameSpellArms, FreshPlayers: mapload.NoFreshPlayers}, mapload.DifficultyNormal)
	if err != nil {
		t.Fatal(err)
	}
	if _, present := w.CurrentPlayers(); present {
		t.Fatal("ROM1 constructor policy was applied to ROM2")
	}
}

func TestAuthoredPlayerControlSurvivesMissionAndColdRestore(t *testing.T) {
	for _, control := range []uint32{0, 1, 2, 0x100, 0xf1234567} {
		m := &alm.Map{Width: 8, Height: 8,
			Groups: []alm.Group{{Participant: control}, {Participant: control}},
			Units:  []alm.Unit{{Owner: 1}, {Owner: 2}}}
		w, _, err := mapload.StartMission(m, nil, mapload.DifficultyNormal, nil)
		if err != nil {
			t.Fatal(err)
		}
		want := []sim.PlayerParticipant{{PlayerID: 1, Value: control}, {PlayerID: 2, Value: control}}
		if got, present := w.PlayerParticipants(); !present || !reflect.DeepEqual(got, want) {
			t.Fatalf("authored control %#x changed on mission start: %v/%t", control, got, present)
		}
		raw, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var cold sim.World
		if err := cold.UnmarshalBinary(raw); err != nil || cold.Hash() != w.Hash() {
			t.Fatal("authored control changed on cold restore", err)
		}
		if got, present := cold.PlayerParticipants(); !present || !reflect.DeepEqual(got, want) {
			t.Fatalf("authored control %#x changed after cold restore: %v/%t", control, got, present)
		}
	}
	m := &alm.Map{Width: 8, Height: 8, Units: []alm.Unit{{Owner: sim.SelfSlot}}}
	w, _, err := mapload.StartMission(m, nil, mapload.DifficultyNormal, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, present := w.PlayerParticipants(); !present || !reflect.DeepEqual(got, []sim.PlayerParticipant{{PlayerID: 1, Value: 0}}) {
		t.Fatal("synthesized SelfSlot lost its legacy human control", got, present)
	}
}
