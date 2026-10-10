package game

import (
	"slices"
	"testing"

	"againrom/pkg/formats/reg"
	"againrom/pkg/formats/sav"
)

// shippedTownCompanion is the companion both shipped campaigns grant in town
// ([Mission30] AddHero = 22, REG-SCN-098). TestReleaseTownCompanionIsTheRegistryAddHero
// proves the installed registry names exactly it.
const shippedTownCompanion = 22

// releaseScenario parses the installed scenario registry straight from the
// campaign container, apart from ReadCampaign.
func releaseScenario(t *testing.T, f *FrontEnd) *reg.Reg {
	t.Helper()
	raw, err := f.Archives.Containers.ReadFile(ScenarioRegistry)
	if err != nil {
		t.Fatal(err)
	}
	r, err := reg.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// releaseSectionInts reads key of section in either shape the file writes.
func releaseSectionInts(r *reg.Reg, section, key string) []int32 {
	if v, ok := r.GetIntArray(section, key); ok {
		return v
	}
	if v, ok := r.GetInt(section, key); ok {
		return []int32{v}
	}
	return nil
}

// TestReleaseTownCompanionIsTheRegistryAddHero binds the town companion to the
// installed AddHero arrays: the registry read apart from ReadCampaign names
// npc 22 once, in [Mission30], and the campaign grants that one companion and
// no other subscript.
func TestReleaseTownCompanionIsTheRegistryAddHero(t *testing.T) {
	f := releaseFront(t)
	r := releaseScenario(t, f)
	grants := map[string][]int32{}
	for _, sec := range r.Root.Children {
		if v := releaseSectionInts(r, sec.Name, "AddHero"); len(v) != 0 {
			grants[sec.Name] = v
		}
	}
	if len(grants) != 1 || !slices.Equal(grants["Mission30"], []int32{shippedTownCompanion}) {
		t.Fatalf("installed AddHero arrays = %v, want [Mission30] = [%d]", grants, shippedTownCompanion)
	}
	c := f.Campaign.Value()
	for npc := 0; npc <= 255; npc++ {
		if got := c.townGrants(npc); got != (npc == shippedTownCompanion) {
			t.Fatalf("townGrants(%d) = %t", npc, got)
		}
	}
}

// TestReleaseLegacyCityCompanionIsTheRegistryRecord binds the identity a city
// save before version 7 gives a companion to the installed npc.reg record: on
// both shipped roots Humans rows 28 and 29, and no other row, are npc 22.
func TestReleaseLegacyCityCompanionIsTheRegistryRecord(t *testing.T) {
	f := releaseFront(t)
	in := f.townInstall()
	for row := 0; row < f.Table.Humans.Len() && row <= 255; row++ {
		id, npc := in.legacyCityPartyIdentity(sav.Character{DefRow: uint8(row)})
		companion := row == 28 || row == 29
		if companion && (id != "npc:22" || npc != shippedTownCompanion) || !companion && (id != "" || npc != 0) {
			t.Fatalf("Humans row %d legacy identity = %q/%d", row, id, npc)
		}
	}
	if id, npc := in.legacyCityPartyIdentity(sav.Character{Hero: true, DefRow: 28}); id != "hero" || npc != 0 {
		t.Fatalf("legacy hero identity = %q/%d", id, npc)
	}
}
