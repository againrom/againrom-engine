package game

import (
	"encoding/json"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

func TestCurrentTownSavedPoolsPresenceAndZero(t *testing.T) {
	p := mapload.PartyMember{Hero: data.Hero{Body: 20, Reaction: 25, Mind: 30, Spirit: 35}, Profile: data.Profile{ManaColumn: true, HealthColumn: true}}
	d, hp, mp := mapload.PartyDisplayWithTable(p, nil)
	unit := sav.CityUnitData{Token: make([]byte, 37)}
	for _, present := range []bool{false, true} {
		p.Saved = nil
		want := [6]uint16{uint16(hp), uint16(d.HealthMax), uint16(mp), uint16(d.ManaMax), 100, 50}
		if present {
			p.Saved = &mapload.Saved{HP: 17, MaxHP: 257}
			want = [6]uint16{17, 257, 0, 0, 0, 0}
		}
		h, err := nativeCityHumanFromDerived(p, nil, unit, d, hp, mp)
		if err != nil {
			t.Fatal(err)
		}
		got := [6]uint16{h.Health, h.HealthMax, h.Mana, h.ManaMax, h.HealthPeriod, h.ManaPeriod}
		if got != want {
			t.Fatalf("Saved present=%v got %v want %v", present, got, want)
		}
	}
}

func TestAbsentPartyTemplateCurrentSourcePoolsWin(t *testing.T) {
	member, table := absentMemberFixture(t)
	want := [6]uint16{40, 90, 71, 19, 113, 79}
	for i, value := range want {
		member.Carry.LiveLoad.Inventory.Source.Stats[8+i] = value
	}
	member.Saved.HP, member.Saved.MaxHP, member.Saved.HealthRegenPeriod = 110, 130, 100
	member.Saved.Mana, member.Saved.MaxMana, member.Saved.ManaRegenPeriod = 27, 137, 50
	for cycle := 0; cycle < 2; cycle++ {
		p, err := captureCurrentPartyTemplate(777, member, member, table)
		if err != nil {
			t.Fatal(err)
		}
		characters, err := sav.ReadDocumentCharacters(sav.DocumentData{Objects: p.Template.Records.Objects}, []uint16{p.Template.Records.Actor})
		if err != nil {
			t.Fatal(err)
		}
		var wire [6]uint16
		copy(wire[:], characters[0].Character.Stats[8:14])
		if wire != want {
			t.Errorf("cycle %d ordinary pools %v want source %v", cycle, wire, want)
		}
		raw, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		var cold currentPartyMember
		if err := json.Unmarshal(raw, &cold); err != nil {
			t.Fatal(err)
		}
		got, err := cold.restoreFromCurrent(emptyTemplateWorld(t), table)
		if err != nil {
			t.Fatal(err)
		}
		saved := [6]uint16{uint16(got.Saved.HP), uint16(got.Saved.MaxHP), uint16(got.Saved.HealthRegenPeriod), uint16(got.Saved.Mana), uint16(got.Saved.MaxMana), uint16(got.Saved.ManaRegenPeriod)}
		if saved != want {
			t.Errorf("cycle %d cold Saved pools %v want source %v", cycle, saved, want)
		}
		var source [6]uint16
		copy(source[:], got.Carry.LiveLoad.Inventory.Source.Stats[8:14])
		if source != want {
			t.Errorf("cycle %d cold source pools %v want %v", cycle, source, want)
		}
		member = got
	}
}
