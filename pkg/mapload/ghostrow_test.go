package mapload

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// TestRaisedGhostCarriesItsWholeUnitsRow: row columns, the seven corpse
// stores (MAGIC-249) and no difficulty scaling (MAGIC-250).
func TestRaisedGhostCarriesItsWholeUnitsRow(t *testing.T) {
	params := make([]int32, 41)
	for i := range params {
		params[i] = -1
	}
	params[0], params[4], params[5], params[6], params[7] = 37, 90, 200, 12, 60
	params[14], params[15], params[29], params[30] = 40, 55, 61, 0
	params[36], params[37], params[38], params[39], params[40] = 2, 15, 50, 3, 9
	table := &Table{Units: ghostResistanceCollection{{}, {name: ghostRowName, params: params}}}

	ghost := ghostTemplate(table)
	if ghost.MaxHP != 90 || ghost.ToHit != 40 || ghost.Defence != 55 {
		t.Fatalf("template health/to-hit/defence = %d/%d/%d, want the unscaled row 90/40/55", ghost.MaxHP, ghost.ToHit, ghost.Defence)
	}

	caster := sim.Entity{ID: 1, X: 1, Y: 1, HP: 100, MaxHP: 100, Mana: 100, MaxMana: 100, Mind: 30, Reaction: 120,
		Owner: sim.SelfSlot, Group: 4, TypeID: sim.HumanTypeID, TokenSize: 1, ScanRange: 6, KnownSpells: 1 << 25, DyingTime: 200}
	corpse := sim.Entity{ID: 2, X: 2, Y: 1, HP: 101, MaxHP: 101, Reaction: 41, Mind: 39, Spirit: 29, ToHit: 71, Defence: 73,
		TokenSize: 1, DyingTime: 1, Owner: 2}
	world, err := sim.NewSummoningWorld(47, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{caster, corpse}, nil, sim.Relations{}, nil, nil,
		[]sim.SpellRule{{ID: 25, ManaCost: 2, TargetsUnit: true, MaxRange: 4}}, ghost)
	if err != nil {
		t.Fatal(err)
	}
	if err := world.HeadlessKill(corpse.ID); err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 256; n++ {
		sim.Step(world, nil)
		if c, ok := world.Entity(corpse.ID); ok && c.Decay == sim.DecayBones {
			break
		}
	}
	consumed, ok := world.Entity(corpse.ID)
	if !ok || consumed.Decay != sim.DecayBones {
		t.Fatal("fixture did not retain a bones corpse")
	}
	sim.Step(world, []sim.Command{sim.Cast(caster.ID, corpse.ID, 25)})
	var raised sim.Entity
	for n := 0; n < 256 && raised.ID == 0; n++ {
		for _, e := range world.Entities() {
			if e.Class == 61 && e.Alive() {
				raised = e
			}
		}
		if raised.ID == 0 {
			sim.Step(world, nil)
		}
	}
	if raised.ID == 0 {
		t.Fatal("Control Spirit did not raise the Ghost row")
	}

	row := []struct {
		name      string
		got, want int32
	}{
		{"health regeneration period", raised.HealthRegenPeriod, 200},
		{"mana maximum", raised.MaxMana, 12},
		{"mana regeneration period", raised.ManaRegenPeriod, 60},
		{"see-invisible", int32(raised.SeeInvisible), 2},
		{"capacity", raised.Capacity, data.UnitCapacity()},
		{"general skill", raised.Skill[data.SkillGeneral], 40},
		{"experience value", raised.XPValue, 15},
		{"gold chance", raised.GoldChance, 50},
		{"treasure minimum", raised.TreasureMin, 3},
		{"treasure maximum", raised.TreasureMax, 9},
		{"type", raised.TypeID, 61},
	}
	for _, c := range row {
		if c.got != c.want {
			t.Errorf("raised %s = %d, want the row's %d", c.name, c.got, c.want)
		}
	}
	stores := []struct {
		name      string
		got, want int32
	}{
		{"health maximum", raised.MaxHP, consumed.MaxHP / 2},
		{"health", raised.HP, consumed.MaxHP / 2},
		{"reaction", raised.Reaction, consumed.Reaction/2 + 1},
		{"Mind", raised.Mind, consumed.Mind},
		{"Spirit", raised.Spirit, consumed.Spirit},
		{"to-hit", raised.ToHit, consumed.ToHit},
		{"defence", raised.Defence, consumed.Defence},
	}
	for _, c := range stores {
		if c.got != c.want {
			t.Errorf("raised %s = %d, want the corpse store %d", c.name, c.got, c.want)
		}
	}
	if raised.Owner != caster.Owner || raised.Group != caster.Group || raised.GainsXP {
		t.Errorf("raised owner/group/gains = %d/%d/%v", raised.Owner, raised.Group, raised.GainsXP)
	}
}
