package mapload

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

// dyingTimeTable is cheatActorTable with a dying time on each Humans row. A
// hero-band binding reads row 5 (sim.SourceBinding.DefinitionRow).
func dyingTimeTable() *Table {
	table := cheatActorTable()
	table.Humans = ghostResistanceCollection{
		{},
		{name: "First Person", params: cheatActorParams(26, map[int]int32{0: 10, 16: 7, 17: 1, 18: 0, 23: 4, 24: 100})},
		{name: "Named Person", params: cheatActorParams(26, map[int]int32{0: 30, 16: 7, 17: 2, 18: 0, 23: 8, 24: 100})},
		{name: "Third Person", params: cheatActorParams(26, map[int]int32{0: 10, 16: 9, 17: 1, 18: 0, 23: 4, 24: 100})},
		{name: "Fourth Person", params: cheatActorParams(26, map[int]int32{0: 10, 16: 9, 17: 1, 18: 0, 23: 4, 24: 100})},
		{name: "Hero Row", params: cheatActorParams(26, map[int]int32{0: 10, 16: 33, 17: 1, 18: 0, 23: 12, 24: 100})},
	}
	return table
}

// A fresh party member takes his bound Humans row's dying time, the value
// SourceActorSeed gives the same member loaded from a SAV (DAT-SCHEMA-007),
// and his body lies for that dwell before teardown (HERO-DWELL-065).
func TestFreshPartyMemberBodyKeepsItsRowDyingTime(t *testing.T) {
	table := dyingTimeTable()
	for _, tc := range []struct {
		name   string
		member PartyMember
		row    uint8
		dwell  int
	}{
		{"by type", PartyMember{Name: "Joined", Class: 7, FigureFace: 1, Hero: data.NewCampaignHero(data.SkillBlade)}, 1, 12},
		{"hired", PartyMember{Name: "Named Person", Class: 7, FigureFace: 2, MercenaryType: 3, DefinitionRow: 2, Hero: data.NewCampaignHero(data.SkillBlade)}, 2, 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, st, err := StartMission(&alm.Map{Width: 40, Height: 40}, table, DifficultyNormal, []PartyMember{tc.member})
			if err != nil {
				t.Fatal(err)
			}
			id := st.IDs[0]
			e, _ := w.Entity(id)
			loaded, err := SourceActorSeed(sim.SourceBinding{Class: sim.GeneratedHumanBinding, TokenRow: tc.row, TypeID: uint16(e.TypeID)}, table)
			if err != nil {
				t.Fatal(err)
			}
			if e.DyingTime != int32(tc.dwell) || loaded.DyingTime != e.DyingTime {
				t.Fatalf("fresh dying time %d, loaded %d, want row %d's %d", e.DyingTime, loaded.DyingTime, tc.row, tc.dwell)
			}
			sim.Step(w, []sim.Command{sim.TerminalKill(id)})
			for tick := 1; ; tick++ {
				e, ok := w.Entity(id)
				if !ok {
					t.Fatalf("body gone on tick %d", tick)
				}
				if e.Alive() {
					t.Fatalf("body alive on tick %d", tick)
				}
				if e.Decay != sim.DecayFallen {
					if tick != tc.dwell {
						t.Fatalf("body torn down on tick %d, want after the row's dwell %d", tick, tc.dwell)
					}
					break
				}
				if tick > tc.dwell {
					t.Fatalf("body still at stage %d with dwell %d on tick %d", e.Decay, e.Dwell, tick)
				}
				sim.Step(w, nil)
			}
		})
	}
}
