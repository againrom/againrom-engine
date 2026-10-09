package mapload

import (
	"testing"

	"againrom/pkg/sim"
)

func TestResolvedGhostRaiseUsesFreshNativeConstructorWithoutWornHistory(t *testing.T) {
	params := make([]int32, 38)
	for i := range params {
		params[i] = -1
	}
	params[0], params[4], params[14], params[29], params[30] = 37, 90, 17, 61, 0
	units := ghostResistanceCollection{{}, {name: ghostRowName, params: params}}
	table := &Table{Units: units}
	for _, cold := range []bool{false, true} {
		name := "direct"
		if cold {
			name = "cold-policy"
		}
		t.Run(name, func(t *testing.T) {
			ghost := ghostTemplate(table)
			caster := sim.Entity{ID: 1, X: 1, Y: 1, HP: 100, MaxHP: 100,
				Mana: 100, MaxMana: 100, Mind: 30, Reaction: 120,
				Owner: sim.SelfSlot, TypeID: sim.HumanTypeID, TokenSize: 1, ScanRange: 6,
				KnownSpells: 1 << 25, DyingTime: 200}
			corpse := sim.Entity{ID: 2, X: 2, Y: 1, HP: 100, MaxHP: 100,
				Reaction: 41, Mind: 39, Spirit: 29, ToHit: 71, Defence: 73,
				TokenSize: 1, DyingTime: 1, Owner: 2}
			corpse.NativeBasis = sim.NativeActorBasis{BasePresent: true, BaseKnown: 1,
				Base: [24]byte{0: 173}, ModifierPresent: true, ModifierKnown: 1 << 18,
				Modifier: [64]byte{18: 199}, BodyPresent: true, BodyKnown: true, Body: 503}
			world, err := sim.NewSummoningWorld(47, sim.Bounds{Width: 16, Height: 16},
				sim.ModeCanonical, sim.Terrain{}, []sim.Entity{caster, corpse}, nil,
				sim.Relations{}, nil, nil,
				[]sim.SpellRule{{ID: 25, ManaCost: 2, TargetsUnit: true, MaxRange: 4}}, ghost)
			if err != nil {
				t.Fatal(err)
			}
			if cold {
				policy := world.CurrentPolicy()
				actions := world.Actions()
				raw, err := world.MarshalBinary()
				if err != nil {
					t.Fatal(err)
				}
				var restored sim.World
				if err := restored.UnmarshalBinary(raw); err != nil {
					t.Fatal(err)
				}
				if err := restored.RestoreCurrentContinuation(&policy, nil, actions, nil); err != nil {
					t.Fatal(err)
				}
				world = &restored
			}
			if err := world.HeadlessKill(corpse.ID); err != nil {
				t.Fatal(err)
			}
			bones := false
			for n := 0; n < 256; n++ {
				sim.Step(world, nil)
				if current, ok := world.Entity(corpse.ID); ok && current.Decay == sim.DecayBones {
					bones = true
					break
				}
			}
			if !bones {
				t.Fatal("fixture did not retain a bones corpse")
			}
			if why := world.BookSpellRefusal(caster.ID, corpse.ID, 25); why != "" {
				t.Fatal("fixture cannot cast Control Spirit:", why)
			}
			consumed, ok := world.Entity(corpse.ID)
			if !ok {
				t.Fatal("fixture lost its bones corpse")
			}
			sim.Step(world, []sim.Command{sim.Cast(caster.ID, corpse.ID, 25)})
			var raised sim.Entity
			found := false
			for n := 0; n < 256; n++ {
				for _, current := range world.Entities() {
					if current.Class == 61 && current.Alive() {
						raised, found = current, true
						break
					}
				}
				if found {
					break
				}
				sim.Step(world, nil)
			}
			if !found {
				t.Fatal("actual Control Spirit did not raise the resolved row")
			}
			b := raised.NativeBasis
			if !b.BasePresent || b.BaseKnown != 0x003fffff || b.Base != ([24]byte{}) {
				t.Errorf("fresh Ghost Base = %+v, want known zero prefix22, unknown tail", b)
			}
			if !b.ModifierPresent || b.ModifierKnown != ^uint64(0) || b.Modifier != ([64]byte{}) {
				t.Errorf("fresh unworn Ghost Modifier = %+v, want known zero64", b)
			}
			if !b.BodyPresent || !b.BodyKnown || b.Body != 37 {
				t.Errorf("fresh Ghost Body = %+v, want installed row37 independently of corpse503", b)
			}
			if raised.ToHit != consumed.ToHit || raised.Defence != consumed.Defence || raised.Reaction != consumed.Reaction/2+1 || raised.Mind != consumed.Mind || raised.Spirit != consumed.Spirit {
				t.Errorf("corpse overrides were lost: to-hit/defence %d/%d, stats %d/%d/%d", raised.ToHit, raised.Defence, raised.Reaction, raised.Mind, raised.Spirit)
			}
			if items, present := world.EquippedItems(raised.ID); present {
				for _, item := range items {
					if !item.Empty() {
						t.Fatal("raised Ghost invented a worn item", items)
					}
				}
			}
			for n := 0; n < 3; n++ {
				sim.Step(world, nil)
			}
			if after, ok := world.Entity(raised.ID); !ok || after.NativeBasis != b {
				t.Fatal("unrelated ticks reconstructed native Ghost history", after.NativeBasis, b)
			}
		})
	}
}
