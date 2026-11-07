package archtest

import (
	"os"
	"strings"
	"testing"
)

func TestCheckMembershipNamesEachShape(t *testing.T) {
	cases := []struct{ name, file, src, want string }{
		{"companion", "pkg/game/a.go", "func f(p P) bool { return p.CompanionNPC == 22 }", "tests a membership field"},
		{"tavern conversion", "pkg/game/a.go", "func f(p P) bool { return int(p.MercenaryType) > 2 }", "tests a membership field"},
		{"campaign", "pkg/game/a.go", "func f(p P) bool { return p.CampaignHire != 0 }", "tests a membership field"},
		{"arm", "pkg/game/a.go", "func f(p P) { if p.CampaignUnit {} }", "tests a membership field"},
		{"alias", "pkg/game/a.go", "func f(p P) bool { a := p.CompanionNPC; b := a; return b > 0 }", "tests a membership field"},
		{"variable", "pkg/game/a.go", "func f(p P) bool { var a = p.MercenaryType; return a != 0 }", "tests a membership field"},
		{"loop", "pkg/game/a.go", "func f(p P) { for p.CampaignHire > 0 {} }", "tests a membership field"},
		{"switch", "pkg/game/a.go", "func f(p P) { switch p.MercenaryType {case 1:} }", "switches on a membership field"},
		{"comparison call", "pkg/game/a.go", "func f(p P) bool { return slices.Contains(ids, p.CompanionNPC) }", "compares a membership field"},
		{"predicate copy", "pkg/mapload/a.go", "func (p P) Hired() bool {return true}", "second membership predicate"},
		{"codec copy", "pkg/mapload/a.go", "func DecodePartyIdentity() {}", "second membership predicate"},
		{"word conversion", "pkg/game/a.go", "func f(c C) int {return int(uint16(c.DisplayBacking))}", "outside the identity codec"},
		{"word alias", "pkg/game/a.go", "func f(c C) int {a := c.DisplayBacking; b := a; return int(uint16(b))}", "outside the identity codec"},
		{"word mask", "pkg/game/a.go", "func f(c C) int {a := c.DisplayBacking; return int(a & 0xffff)}", "outside the identity codec"},
		{"join reader copy", "pkg/game/a.go", "func (s S) addTalkIdentities() {}", "second join catalog reader"},
		{"retired join", joinCatalogFile, "func joinOnTalk() {}", "second join catalog reader"},
		{"raw profile", "pkg/game/a.go", "func f(e E) { _ = e.PartyProfile }", "outside the join catalog"},
		{"raw rows", "pkg/game/a.go", "func f(e E) { _ = e.JoinRows() }", "outside the join catalog"},
		{"talk", "pkg/game/a.go", "func f(r R) bool {return r.TalkNPC == 22}", "outside the join catalog"},
		{"topic", "pkg/game/a.go", "func f(r R) bool {return r.TalkTopic == 30}", "outside the join catalog"},
		{"grant", "pkg/game/a.go", "func f(r R) bool {return r.TownGrant}", "outside the join catalog"},
		{"slot", "pkg/game/a.go", "func f(r R) int {return r.BankSlot}", "outside the join catalog"},
		{"leave bank", "pkg/game/a.go", "func f(c C) int {return int(c.bank[773])}", "leave join slot"},
		{"leave bank expression", "pkg/game/a.go", "func f(c C) int {return int(c.bank[773+0])}", "leave join slot"},
		{"another join call", "pkg/game/a.go", "func f(s S, in I) {s.addPartyIdentity(in,22)}", "outside the join catalog"},
		{"census adapter", "pkg/game/secondcensus.go", "func secondCensusParty(s S, in I) {s.addPartyIdentity(in,22)}", ""},
		{"wire copy", "pkg/game/a.go", "func f(p P) P {return P{CompanionNPC:p.CompanionNPC}}", ""},
		{"one membership", membershipFile, "func (p P) Hired() bool {return p.MercenaryType != 0}", ""},
		{"one join reader", joinCatalogFile, "func addTalkIdentities(e E) {for _, r := range e.JoinRows() {_ = r.TalkNPC}}", ""},
		{"one profile", joinProfileFile, "func f(e E) { _ = e.PartyProfile }", ""},
		{"test excluded", "pkg/game/a_test.go", "func f(p P) bool {return p.CompanionNPC != 0}", ""},
		{"tavern Units arm", "pkg/game/hire.go", "func f(typ int) bool { return typ <= 2 }", "profile literal"},
		{"tavern Humans arm", "pkg/game/taverntalk.go", "func f(typ int) bool { return typ > 2 }", "profile literal"},
		{"tavern type call", "pkg/game/taverntalk.go", "func f(p P) bool { return p.TavernType() > 2 }", "profile literal"},
		{"tavern lower bound", "pkg/game/originalparty.go", "func originalMercenaryType(typ int) bool { return typ >= 3 }", "profile literal"},
		{"tavern upper bound", "pkg/game/originalparty.go", "func originalMercenaryType(typ int) bool { return 15 < typ }", "profile literal"},
		{"tavern exclusive bound", "pkg/game/campaignprogress.go", "func validateMercenaryTypes(typ int) bool { return typ >= 16 }", "profile literal"},
		{"tavern range constant", "pkg/game/taverntalk.go", "const tavernMercenaryTypes = 15", "outside the profile"},
		{"command tavern bound", "cmd/tavern/main.go", "func f(npc int) bool { return npc <= 15 }", "profile literal"},
		{"tavern profile reads", "pkg/game/taverntalk.go", "func f(e E, typ int) bool { return e.TavernType(typ) && !e.TavernUnit(typ) }", ""},
		{"siege name entry", "pkg/game/tavern.go", "func siegeHireName(typ int) bool { return typ == 2 }", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			vs := CheckMembership(map[string]string{c.file: "package p\n" + c.src})
			if c.want == "" {
				if len(vs) != 0 {
					t.Fatalf("unexpected %v", vs)
				}
				return
			}
			for _, v := range vs {
				if strings.Contains(v.From, c.file) && strings.Contains(v.Reason, c.want) {
					return
				}
			}
			t.Fatalf("%v does not name %q", vs, c.want)
		})
	}
	if vs := CheckMembership(nil); len(vs) != 1 {
		t.Fatalf("empty scan: %v", vs)
	}
}

func TestPartyMembershipHasOneBuilder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files := loadPartyGuardSources(t, root)
	for _, path := range []string{membershipFile, joinCatalogFile, joinProfileFile} {
		if _, ok := files[path]; !ok {
			t.Fatalf("missing builder %s", path)
		}
	}
	for _, v := range CheckMembership(files) {
		t.Errorf("%s: %s", v.From, v.Reason)
	}
}
