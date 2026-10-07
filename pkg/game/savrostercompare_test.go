//go:build sessioncorpusaudit

package game

import (
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/mapload"
)

func TestRoundTripRosterPermutationKeepsMemberChecks(t *testing.T) {
	before := Snapshot{Mission: 111, Party: []mapload.PartyMember{
		{ID: "hero", Name: "same", StartingHero: true},
		{ID: "npc:1", Name: "same", CompanionNPC: 1, KnownSpells: 2},
		{ID: "npc:2", Name: "same", CompanionNPC: 2, KnownSpells: 4},
	}}
	fields := func(s Snapshot) []string {
		var out []string
		for _, difference := range roundTripRosterDiff(before, s) {
			out = append(out, difference.field)
		}
		return out
	}
	if differences := fields(before); len(differences) != 0 {
		t.Fatal(differences)
	}
	permuted := before
	permuted.Party = []mapload.PartyMember{before.Party[0], before.Party[2], before.Party[1]}
	if got := fields(permuted); !reflect.DeepEqual(got, []string{"roster order"}) {
		t.Fatalf("whole-member permutation: %v", got)
	}
	for _, tc := range []struct {
		field string
		edit  func(*mapload.PartyMember)
	}{
		{"name", func(p *mapload.PartyMember) { p.Name = "changed" }},
		{"starting-hero flag", func(p *mapload.PartyMember) { p.StartingHero = true }},
		{"companion npc", func(p *mapload.PartyMember) { p.CompanionNPC++ }},
		{"mercenary type", func(p *mapload.PartyMember) { p.MercenaryType++ }},
		{"hero record", func(p *mapload.PartyMember) { p.Hero.Skill[0]++ }},
		{"known spells", func(p *mapload.PartyMember) { p.KnownSpells++ }},
		{"worn set", func(p *mapload.PartyMember) { p.Worn[0] = 101 }},
		{"carried pack", func(p *mapload.PartyMember) { p.Carried = []uint16{101} }},
	} {
		t.Run(tc.field, func(t *testing.T) {
			changed := permuted
			changed.Party = slices.Clone(permuted.Party)
			tc.edit(&changed.Party[2])
			if got := fields(changed); !reflect.DeepEqual(got, []string{"roster order", tc.field}) {
				t.Fatalf("member mutation disappeared behind permutation: %v", got)
			}
		})
	}
	if _, allowed := originalRoundTripDisclosed["roster order"]; allowed || conversionDisclosedRows["hero record"] != "DIV-1321" {
		t.Fatal("original order loss or spreading hero loss was allowed")
	}
}

func TestRoundTripRosterIdentityCannotHideMissingMembers(t *testing.T) {
	member := func(id string) mapload.PartyMember { return mapload.PartyMember{ID: id, Name: "same"} }
	for _, tc := range []struct {
		name, field   string
		before, after []mapload.PartyMember
	}{
		{"missing member", "roster size", []mapload.PartyMember{member("a"), member("b")}, []mapload.PartyMember{member("a")}},
		{"replaced identity", "roster identity", []mapload.PartyMember{member("a"), member("b")}, []mapload.PartyMember{member("a"), member("c")}},
		{"duplicate before", "roster identity", []mapload.PartyMember{member("a"), member("a")}, []mapload.PartyMember{member("a"), member("b")}},
		{"duplicate after", "roster identity", []mapload.PartyMember{member("a"), member("b")}, []mapload.PartyMember{member("a"), member("a")}},
		{"missing before identity", "roster identity", []mapload.PartyMember{member("")}, []mapload.PartyMember{member("a")}},
		{"missing after identity", "roster identity", []mapload.PartyMember{member("a")}, []mapload.PartyMember{member("")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := roundTripRosterDiff(Snapshot{Mission: 111, Party: tc.before}, Snapshot{Mission: 111, Party: tc.after})
			if len(got) != 1 || got[0].field != tc.field {
				t.Fatalf("identity failure became an allowed permutation: %v", got)
			}
		})
	}
	before := Snapshot{Party: []mapload.PartyMember{{ID: "a", Name: "a"}, {ID: "b", Name: "b"}}}
	after := Snapshot{Party: []mapload.PartyMember{before.Party[1], before.Party[0]}}
	if got := roundTripRosterDiff(before, after); len(got) != 2 || got[0].field != "name" || got[1].field != "name" {
		t.Fatalf("town roster stopped using its ordered comparison: %v", got)
	}
}
