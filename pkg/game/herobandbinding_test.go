package game

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// heroBandRecord is, for one table, which actor each hero-band value binds
// to on a built party with two placed actors, the party identity and Humans
// row each member carries, and the party identity each saved companion
// carries.
func heroBandRecord(t *testing.T, name string, table *mapload.Table) string {
	t.Helper()
	party := []mapload.PartyMember{
		heroMember("hero", false, true, 3, starting, func(p *mapload.PartyMember) { p.DefinitionRow = 30 }),
		heroMember("npc:22", true, true, 1, func(p *mapload.PartyMember) { p.CompanionNPC = 22 }),
		heroMember("fighter", true, false, 1),
		heroMember("npc:23", false, false, 1, func(p *mapload.PartyMember) { p.CompanionNPC = 23 }),
		heroMember("hire", false, false, 5, func(p *mapload.PartyMember) { p.Temporary, p.CampaignHire = true, 5 }),
		heroMember("merc", false, false, 5, func(p *mapload.PartyMember) { p.MercenaryType, p.PlayerCharacter = 1, false }),
	}
	placed := []heroScanActor{
		{ID: 900, Traits: data.HeroTraits{Face: 1}},
		{ID: 901, Traits: data.HeroTraits{Mage: true, Face: 4}},
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s:", name)
	refs := campaignScriptPartyRefs(&alm.Map{}, table, party, func(i int) sim.EntityID { return sim.EntityID(100 + i) }, placed)
	fmt.Fprintf(&b, " roster=%v hero=%v/%d companion=%v/%d", refs.Roster, refs.HasHero, refs.Hero, refs.HasCompanion, refs.Companion)
	values := make([]int, 0, len(refs.Roles))
	for v := range refs.Roles {
		values = append(values, int(v))
	}
	sort.Ints(values)
	for _, v := range values {
		fmt.Fprintf(&b, " %d=%d", v, refs.Roles[uint32(v)])
	}
	for _, p := range party {
		id, ok := mapload.PartyIdentity(p, table)
		fmt.Fprintf(&b, " %s=%d/%v/row%d", p.ID, id, ok, mapload.PartyHumanRow(p, party[0], table))
	}
	for _, c := range []sav.Character{
		{Class: "Human", DisplayBacking: 22, DefRow: 30},
		{Class: "Human", DisplayBacking: 23, DefRow: 31},
		{Class: "Human", DisplayBacking: 22, DefRow: 31},
		{Class: "Human", DisplayBacking: 21, DefRow: 30},
		{Class: "Human", DisplayBacking: 22, DefRow: 30, Hero: true},
		{Class: "Unit", DisplayBacking: 22, DefRow: 30},
	} {
		id, ok := originalPartyIdentity(c, table)
		fmt.Fprintf(&b, " sav%d/%d=%d/%v", c.DisplayBacking, c.DefRow, id, ok)
	}
	return b.String()
}

// TestHeroBandBindingRecord holds which actor each edition binds to
// 10001+k, and what the party identity readers answer, on one built party:
// the first game by the role test, the second by party identity, and a table
// without an edition as the first game.
func TestHeroBandBindingRecord(t *testing.T) {
	npc := shippedHeroTable(t).NPC
	table := func(e *base.Edition) *mapload.Table {
		tb := identityTable()
		tb.NPC, tb.Edition = npc, e
		return tb
	}
	first, second := base.GameROM1.Edition(), base.GameROM2.Edition()
	got := []string{
		heroBandRecord(t, "first", table(&first)),
		heroBandRecord(t, "none", table(nil)),
		heroBandRecord(t, "second", table(&second)),
		heroBandRecord(t, "nil", nil),
	}
	unbound := " hero=0/false/row0 npc:22=0/false/row0 fighter=0/false/row0 npc:23=0/false/row0 hire=0/false/row0 merc=0/false/row0" +
		" sav22/30=0/false sav23/31=0/false sav22/31=0/false sav21/30=0/false sav22/30=0/false sav22/30=0/false"
	role := " roster=true hero=true/100 companion=true/101 10003=102 10004=104 10005=103 10006=901" + unbound
	want := []string{
		"first:" + role,
		"none:" + role,
		"second: roster=true hero=true/100 companion=true/101 10003=103" +
			" hero=21/true/row30 npc:22=22/true/row30 fighter=0/false/row0 npc:23=23/true/row31 hire=5/true/row0 merc=0/false/row0" +
			" sav22/30=22/true sav23/31=23/true sav22/31=0/false sav21/30=0/false sav22/30=0/false sav22/30=0/false",
		"nil: roster=true hero=true/100 companion=false/0" + unbound,
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("hero-band record:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
