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
		want := []sim.PlayerParticipant{{1, 1}, {2, 0}, {3, 0xf1234567}, {4, 1}}
		if got, present := w.PlayerParticipants(); !present || !reflect.DeepEqual(got, want) {
			t.Fatal("fresh Participant did not use explicit map/owned/fallback policy", got)
		}
		m.Groups[1].Participant++
		if got, _ := w.PlayerParticipants(); !reflect.DeepEqual(got, want) {
			t.Fatal("constructed Participant still aliases input Map")
		}
		m.Groups[1].Participant--
		if _, _, present := w.SavedGroups(); present {
			t.Fatal("Player constructor installed command Group dispatch")
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
