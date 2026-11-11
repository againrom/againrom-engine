package mapload

import (
	"testing"

	"againrom/pkg/sim"
)

// darkSourceHero is a class-2 source Human of the player at (40,40): Mind 25,
// Reaction 36, so the derive gives the sight word 1648 the record already
// holds, and a zero sight modifier.
func darkSourceHero() (sim.Entity, sim.ActorLoadSnapshot) {
	s := sim.SourceActor{Class: 2, Fighter: true, TypeID: uint16(sim.HumanTypeID), Sight: 1648, MoverSpeed: 18,
		Stats: [14]uint16{10, 36, 25, 10, 18, 0, 0, 500, 100, 100, 100, 0, 0, 50}}
	e := sim.Entity{ID: 2, X: 40, Y: 40, HP: 100, MaxHP: 100, Speed: 18, TypeID: sim.HumanTypeID, Owner: sim.SelfSlot,
		Humanoid: true, Mind: 25, Reaction: 36, ScanRange: 6, TokenSize: 1}
	load := sim.ActorLoadSnapshot{Inventory: sim.ActorLoad{Present: true, ContainerPresent: true, Source: s},
		Load: 0, Capacity: 500, Speed: 18}
	load.Movement = sim.HumanMovement{Present: true, RawSpeed: 18, NativeSpeed: 18, Capacity: 500}
	return e, load
}

func darkSourceScan(w *sim.World) (sim.Entity, bool) {
	var hero sim.Entity
	for _, e := range w.Entities() {
		if e.ID == 2 {
			hero = e
		}
	}
	for _, a := range w.ActiveEffects() {
		if a.Target == 2 && a.Kind == sim.EffectScanRange {
			return hero, true
		}
	}
	return hero, false
}

func darkSourceLit(w *sim.World) int {
	n := 0
	for _, v := range w.SightOf(sim.SelfSlot, []sim.EntityID{2}) {
		if v != 0 {
			n++
		}
	}
	return n
}

// A Darkness cloud the player's mage casts over a source-backed Human, the
// actor an original SAV loads, shrinks the sight that Human stamps into the
// player's fog; the count returns when the effect ends; Light grows it. Both
// games, through the bound source derive.
func TestADarknessCloudShrinksASourceHumansSight(t *testing.T) {
	t.Parallel()
	for _, second := range []bool{false, true} {
		for _, dark := range []bool{true, false} {
			for _, mind := range []int32{50, 140} {
				id := uint16(12)
				switch {
				case dark && second:
					id = 14
				case dark:
					id = 17
				case second:
					id = 15
				}
				// The shipped durations: area column 20 and effect counter 16 in
				// the first game (MAGIC-398), 15 and the raw 1 in the second
				// (R2-ENGINE-024, R2-ASSET-178).
				rows := []sim.SpellRule{{ID: id, Area: true, Distribution: 3, Radius: 2, AreaDuration: 20, MaxRange: 20,
					ManaCost: 1, TargetsUnit: true, EffectKind: sim.EffectScanRange, EffectMode: sim.EffectDuration, EffectDuration: 16}}
				var script *sim.Script
				if second {
					sim.AssignSecondGameArms(rows)
					rows[0].AreaDuration, rows[0].EffectDuration = 15, 1
					s, err := sim.NewROM2Script(nil, nil, nil)
					if err != nil {
						t.Fatal(err)
					}
					script = s
				}
				caster := sim.Entity{ID: 1, X: 37, Y: 40, HP: 100, MaxHP: 100, Owner: sim.SelfSlot, TokenSize: 1,
					Mind: mind, Mana: 500, MaxMana: 500, KnownSpells: 1 << id, AttackCharge: 4, AttackRelax: 2,
					Book: sim.Spellbook{State: sim.BookPresent}}
				caster.Book.Slots[id-1] = sim.BookSpell{Range: 20, ManaCost: 1}
				hero, load := darkSourceHero()
				w, err := sim.NewStockedSpelledWorld(1, sim.Bounds{Width: 64, Height: 64}, sim.ModeCanonical, sim.Terrain{},
					[]sim.Entity{caster, hero}, script, sim.Relations{}, nil, nil, rows)
				if err != nil {
					t.Fatal(err)
				}
				BindSourceDerive(w)
				if err := w.RestoreActorLoad(2, load); err != nil {
					t.Fatal(err)
				}
				name := func() string {
					return map[bool]string{false: "first", true: "second"}[second] + map[bool]string{false: " Light", true: " Darkness"}[dark]
				}()
				start, _ := darkSourceScan(w)
				before := darkSourceLit(w)
				if why := w.BookSpellCellRefusal(1, 40, 40, uint32(id)); why != "" {
					t.Fatalf("%s mind %d: fixture: the cast is refused: %s", name, mind, why)
				}
				commands := []sim.Command{sim.CastAt(1, sim.SpellID(id), sim.CellPoint{X: 40, Y: 40})}
				landed := false
				for k := 0; k < 400 && !landed; k++ {
					sim.Step(w, commands)
					commands = nil
					_, landed = darkSourceScan(w)
				}
				if !landed {
					t.Fatalf("%s mind %d: the cloud's effect never stood on the source Human (clouds %d, caster %+v)", name, mind, len(w.CellEffects()), w.Entities()[0])
				}
				under, _ := darkSourceScan(w)
				lit := darkSourceLit(w)
				t.Logf("%s mind %d: scan %d -> %d, word %d -> %d, cells %d -> %d", name, mind, start.ScanRange, under.ScanRange,
					start.ActorLoad.Source.Sight, under.ActorLoad.Source.Sight, before, lit)
				if dark && lit >= before || !dark && lit <= before {
					t.Errorf("%s mind %d: the source Human lights %d cells under the cloud, %d before", name, mind, lit, before)
				}
				for k := 0; k < 20000; k++ {
					if _, on := darkSourceScan(w); !on && len(w.CellEffects()) == 0 {
						break
					}
					sim.Step(w, nil)
				}
				after, on := darkSourceScan(w)
				if on || darkSourceLit(w) != before || after.ActorLoad.Source.Sight != start.ActorLoad.Source.Sight {
					t.Errorf("%s mind %d: after the effect (standing %v) the Human lights %d cells at word %d, want %d at %d",
						name, mind, on, darkSourceLit(w), after.ActorLoad.Source.Sight, before, start.ActorLoad.Source.Sight)
				}
			}
		}
	}
}
