package game

import (
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/reg"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

const kindStr = 0x00

func shippedHeroTable(t *testing.T) *mapload.Table {
	t.Helper()
	tpl := func(name, flags string, face int32) synth.RegNode {
		n := synth.RegNode{Name: name, Kind: kindDir, Children: []synth.RegNode{{Name: "Flags", Kind: kindStr, Str: flags}}}
		if face != 0 {
			n.Children = append(n.Children, synth.RegNode{Name: "Face", Kind: kindInt, Int: face})
		}
		return n
	}
	archetype := func(name string, face int32) synth.RegNode {
		return synth.RegNode{Name: name, Kind: kindDir, Children: []synth.RegNode{{Name: "Face", Kind: kindInt, Int: face}}}
	}
	r, err := reg.Parse(synth.Reg(kindRoot, []synth.RegNode{
		tpl("npc21", "Hero,Me,Start", 0), tpl("npc22", "Hero,Mage,!MySex,Start", 0),
		tpl("npc23", "Hero,!Mage,!MySex,Start", 0), tpl("npc24", "Hero,!MyClass,MySex,Start", 0),
		tpl("npc25", "Hero,Face,!Female,!Mage", 1), tpl("npc26", "Hero,Face,Mage,!Female", 4),
		archetype("MaleFighter", 5), archetype("FemaleFighter", 1), archetype("MaleMage", 3), archetype("FemaleMage", 1),
	}))
	if err != nil {
		t.Fatal(err)
	}
	return &mapload.Table{NPC: data.LoadNPCDefs(r)}
}

func heroMember(id string, female, mage bool, face int, mutate ...func(*mapload.PartyMember)) mapload.PartyMember {
	dir := data.FigureDirFor(mage, female)
	p := mapload.PartyMember{ID: id, PlayerCharacter: true, FigureDir: string(dir), Mage: mage, FigureFace: face}
	for _, f := range mutate {
		f(&p)
	}
	return p
}

func starting(p *mapload.PartyMember) { p.StartingHero = true }

func TestHeroOrdinalRefsFollowTheRolePredicateNotTheRosterSlot(t *testing.T) {
	table := shippedHeroTable(t)
	m := &alm.Map{}
	hire := heroMember("hire", false, false, 5, func(p *mapload.PartyMember) { p.MercenaryType = 1; p.PlayerCharacter = false })
	party := []mapload.PartyMember{
		heroMember("hero", false, true, 3, starting),
		hire,
		heroMember("fighter", true, false, 1),
		heroMember("mage", true, true, 1),
		heroMember("brian", false, false, 1),
	}
	ids := []sim.EntityID{40, 41, 42, 0, 44}
	refs := campaignScriptPartyRefs(m, table, party, func(i int) sim.EntityID { return ids[i] }, nil)
	if !refs.HasHero || refs.Hero != 40 {
		t.Fatal("ordinal 0 is the primary", refs)
	}
	if !refs.HasCompanion || refs.Companion != 0 {
		t.Fatal("ordinal 1 is the other-sex mage with its default face, whatever its slot or its entity id", refs)
	}
	for ordinal, want := range map[uint32]sim.EntityID{10003: 42, 10005: 44} {
		if got, ok := refs.Roles[ordinal]; !ok || got != want {
			t.Fatalf("ordinal %d = %d/%v, want %d", ordinal, got, ok, want)
		}
	}
	if got, ok := refs.Roles[10004]; !ok || got != 41 {
		t.Fatalf("ordinal 3 = %d/%v, want the male fighter", got, ok)
	}
	if _, ok := refs.Roles[10006]; ok {
		t.Fatal("no male mage with face 4 exists, yet ordinal 5 resolved")
	}
	if _, ok := refs.Roles[10007]; ok {
		t.Fatal("an ordinal without a template record resolved")
	}
	party[1] = heroMember("hire", true, false, 1, func(p *mapload.PartyMember) { p.MercenaryType = 1; p.PlayerCharacter = false })
	party[4] = heroMember("other", true, true, 4)
	refs = campaignScriptPartyRefs(m, table, party, func(i int) sim.EntityID { return ids[i] }, nil)
	if got, ok := refs.Roles[10005]; !ok || got != 41 {
		t.Fatal("hired unit was read as a woman", got, ok)
	}
}

func TestHeroOrdinalRefsUnresolvedAndGatedCases(t *testing.T) {
	table := shippedHeroTable(t)
	id := func(i int) sim.EntityID { return sim.EntityID(i) }
	alone := []mapload.PartyMember{heroMember("hero", false, true, 3, starting)}
	refs := campaignScriptPartyRefs(&alm.Map{}, table, alone, id, nil)
	if !refs.HasHero || refs.HasCompanion || len(refs.Roles) != 0 {
		t.Fatal("a party of the primary alone resolved a template ordinal", refs)
	}
	if refs := campaignScriptPartyRefs(&alm.Map{}, table, nil, id, nil); refs.HasHero || refs.HasCompanion || refs.Roles != nil {
		t.Fatal("an empty party resolved a reference")
	}
	multi := &alm.Map{}
	multi.Meta.Word70 = 4
	two := []mapload.PartyMember{heroMember("hero", false, true, 3, starting), heroMember("mage", true, true, 1)}
	if refs := campaignScriptPartyRefs(multi, table, two, id, nil); refs.HasHero || refs.HasCompanion {
		t.Fatal("a map whose player capacity exceeds one resolved a hero ordinal", refs)
	}
	if refs := campaignScriptPartyRefs(&alm.Map{}, nil, two, id, nil); !refs.HasHero || refs.HasCompanion {
		t.Fatal("without a registry only ordinal 0 resolves", refs)
	}
	twin := []mapload.PartyMember{heroMember("twin", false, true, 3), heroMember("hero", false, true, 3, starting)}
	if refs := campaignScriptPartyRefs(&alm.Map{}, table, twin, id, nil); !refs.HasHero || refs.Hero != 0 {
		t.Fatal("ordinal 0 did not take the first actor sharing the primary's values", refs)
	}
}

func TestLoadReresolvesHeroOrdinalsFromTheLoadedParty(t *testing.T) {
	table := shippedHeroTable(t)
	party := []mapload.PartyMember{
		heroMember("hero", false, true, 3, starting),
		heroMember("mage", true, true, 1),
	}
	saved := currentScriptRoleProgram(t, mapload.ScriptRefs{Units: map[uint16]sim.EntityID{77: 7}, Hero: 0, HasHero: true})
	world := currentScriptRoleWorld(t, saved)
	ids := []sim.EntityID{0, 1}
	refs := campaignScriptPartyRefs(&alm.Map{}, table, party, func(i int) sim.EntityID { return ids[i] }, nil)
	// The saved program built every node; the roster leaves 10003 unresolved,
	// so the roles are compiled in the saved program's shape.
	roles, err := currentScriptRolesProgram(world, nil, refs, func(_ *alm.Map, r mapload.ScriptRefs) (*sim.Script, mapload.ScriptReport, error) {
		return mapload.CompileScriptFrom(currentScriptRoleSource(), r)
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := restoreCurrentScriptRoles(world, roles); err != nil {
		t.Fatal(err)
	}
	checks := world.Script().Checks()
	if !checks[0].HasUnit2 || checks[0].Unit2 != 1 || !checks[0].HasUnit || checks[0].Unit != 0 {
		t.Fatal("load did not resolve ordinal 1 against the loaded party", checks[0])
	}
	again := currentScriptRoleWorld(t, saved)
	if err := restoreCurrentScriptRoles(again, roles); err != nil || again.Hash() != world.Hash() {
		t.Fatal("repeating the load changed the bindings", err)
	}
	refs = campaignScriptPartyRefs(&alm.Map{}, table, party[:1], func(i int) sim.EntityID { return ids[i] }, nil)
	if refs.HasCompanion {
		t.Fatal("a party without the role resolved it")
	}
}

func TestHeroOrdinalScanReachesNamedPlacementsAfterTheParty(t *testing.T) {
	table := shippedHeroTable(t)
	party := []mapload.PartyMember{
		heroMember("hero", false, true, 3, starting),
		heroMember("mage", true, true, 1),
	}
	ids := []sim.EntityID{100, 101}
	placed := []heroScanActor{
		{ID: 7, Traits: data.HeroTraits{Female: true, Mage: true, Face: 1}},
		{ID: 8, Traits: data.HeroTraits{Female: true, Mage: false, Face: 1}},
	}
	refs := campaignScriptPartyRefs(&alm.Map{}, table, party, func(i int) sim.EntityID { return ids[i] }, placed)
	if !refs.HasCompanion || refs.Companion != 101 {
		t.Fatal("a member satisfying ordinal 1 lost it to a placement", refs)
	}
	if got, ok := refs.Roles[10003]; !ok || got != 8 {
		t.Fatalf("ordinal 2 = %d/%v, want the placement no member satisfies", got, ok)
	}
	refs = campaignScriptPartyRefs(&alm.Map{}, table, party[:1], func(i int) sim.EntityID { return ids[i] }, placed)
	if !refs.HasCompanion || refs.Companion != 7 {
		t.Fatal("ordinal 1 did not reach the first satisfying placement", refs)
	}
	if refs := campaignScriptPartyRefs(&alm.Map{}, table, party[:1], func(i int) sim.EntityID { return ids[i] }, nil); refs.HasCompanion || len(refs.Roles) != 0 {
		t.Fatal("an ordinal resolved with no member or placement satisfying it", refs)
	}
}
