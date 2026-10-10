package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// TestReleasePlacedHeroOrdinals derives the scan inputs of the npc placements
// TRIG-MAPORD-107 names and binds each primary alone against them: only
// 100.alm unit 245 takes an ordinal, 10004 for every primary. A placement
// whose npcnames line is empty is not scanned. No shipped map names an
// ordinal above 10006 (TRIG-HEROFAIL-078).
func TestReleasePlacedHeroOrdinals(t *testing.T) {
	f := releaseFront(t)
	type want struct {
		unit   uint16
		traits data.HeroTraits
	}
	fixed := map[int][]want{
		30:  {{56, data.HeroTraits{Face: 28}}},
		81:  {{52, data.HeroTraits{Face: 19}}},
		151: {{205, data.HeroTraits{Face: 18}}, {589, data.HeroTraits{Mage: true, Face: 5}}},
	}
	// Unit 245 per primary: male fighter, female fighter, male mage, female mage.
	unit245 := map[string]data.HeroTraits{
		"primary-male-fighter":   {Mage: true, Face: 3},
		"primary-female-fighter": {Female: true, Mage: true, Face: 1},
		"primary-male-mage":      {Face: 5},
		"primary-female-mage":    {Female: true, Face: 1},
	}
	maps := map[int]*alm.Map{}
	for _, n := range []int{30, 81, 100, 151} {
		addr, _ := MissionMap(n)
		b, err := f.Archives.Containers.ReadFile(addr)
		if err != nil {
			t.Fatal(err)
		}
		m, err := alm.Open(b)
		if err != nil {
			t.Fatal(err)
		}
		mapload.WithdrawBorderPlacements(m)
		maps[n] = m
	}
	for _, roster := range unbuiltCensusRosters(f)[:4] {
		for n, m := range maps {
			got := map[uint16]mapload.PlacedHero{}
			for _, p := range mapload.PlacedHeroes(m, f.Table, roster.party, n) {
				got[p.UnitID] = p
			}
			expect := fixed[n]
			if n == 100 {
				expect = []want{{245, unit245[roster.name]}}
			}
			for _, w := range expect {
				p, ok := got[w.unit]
				if !ok || p.Traits != w.traits {
					t.Errorf("%s %d.alm unit %d: placed %v %+v, want %+v", roster.name, n, w.unit, ok, p.Traits, w.traits)
				}
			}
			refs := campaignScriptRefs(m, f.Table, roster.party, n)
			party := map[sim.EntityID]bool{}
			for i := range roster.party {
				party[mapload.PartyEntity(m, i)] = true
			}
			bound := map[uint32]sim.EntityID{}
			if refs.HasCompanion && !party[refs.Companion] {
				bound[10002] = refs.Companion
			}
			for v, id := range refs.Roles {
				if v <= 10006 && !party[id] {
					bound[v] = id
				}
			}
			units := mapload.ScriptUnits(m, roster.party)
			if n == 100 {
				if len(bound) != 1 || bound[10004] != units[245] {
					t.Errorf("%s 100.alm: placements bound %v, want 10004 to unit 245", roster.name, bound)
				}
			} else if len(bound) != 0 {
				t.Errorf("%s %d.alm: placements bound %v, want none", roster.name, n, bound)
			}
		}
	}
	unnamed := *f.Table
	unnamed.NPCNames = nil
	if placed := mapload.PlacedHeroes(maps[100], &unnamed, unbuiltCensusRosters(f)[0].party, 100); len(placed) != 0 {
		t.Fatalf("placements without an npcnames line were scanned: %+v", placed)
	}
}
