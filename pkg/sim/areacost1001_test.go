package sim

import "testing"

// Six is the cell layer table width, not an object cap (MAGIC-CLOUDOWNER-155).
// The old cap-refusal tests are replaced by paid seventh-cast witnesses and
// actual unsupported-owner/program refusals at the same admission seam.
func acFullAnchorWorld(t *testing.T, mana int32, unitTarget bool) (*World, SpellRule) {
	t.Helper()
	rule := SpellRule{ID: 21, Area: true, Distribution: 5, Radius: 4, School: 4,
		ManaCost: 25, MaxRange: 8, DamageMin: 4, DamageMax: 4, Damaging: true}
	caster := Entity{ID: 1, X: 16, Y: 20, HP: 100, MaxHP: 100, TokenSize: 1, Owner: SelfSlot,
		Mind: 30, Mana: mana, MaxMana: 1000, KnownSpells: 1 << 21, ScanRange: 19,
		AttackCharge: 2, AttackRelax: 1}
	actors := []Entity{caster}
	if unitTarget {
		target := spEnt(2, 20, 20)
		target.Owner = 2
		actors = append(actors, target)
		rule.TargetsUnit = true
	}
	w, err := NewStockedSpelledWorld(0xc1, Bounds{Width: 40, Height: 40}, ModeCanonical,
		Terrain{}, actors, nil, Relations{}, nil, nil, []SpellRule{rule})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < cellEffectSlots; i++ {
		if !w.landArea(rule, 0, 0, false, 16, 20, 20, 20, nil) {
			t.Fatalf("the landing path refused record %d of %d", i+1, cellEffectSlots)
		}
	}
	if got := len(w.CellEffects()); got != cellEffectSlots {
		t.Fatalf("the anchor holds %d records, want %d", got, cellEffectSlots)
	}
	return w, rule
}

func TestArea1164SeventhCastPaysOnceAtAdmissionAndRelease(t *testing.T) {
	for _, atUnit := range []bool{false, true} {
		for _, duringWindup := range []bool{false, true} {
			w, rule := acFullAnchorWorld(t, 500, atUnit)
			rule.TargetsUnit = atUnit
			w.spells[0] = rule
			command := Command{Kind: KindCastAt, Entity: 1, X: 20, Y: 20, Spell: 21}
			if atUnit {

				command = Command{Kind: KindCast, Entity: 1, X: 2, Y: 21}
			}
			if duringWindup {
				w.effects = nil
				w.entities[0].AttackCharge = 12
			}
			if r := w.BookSpellCellRefusal(1, 20, 20, 21); r != "" && !atUnit {
				t.Fatal(r)
			}
			before := w.entities[0].Mana
			Step(w, []Command{command})
			if duringWindup {
				if len(w.bookCasts) != 1 {
					t.Fatal("not waiting for release")
				}
				for range 6 {
					if !w.landArea(rule, 0, 0, false, 16, 20, 20, 20, nil) {
						t.Fatal("filler refused")
					}
				}
			}
			for tick := 0; tick < 16 || len(w.bookCasts) > 0; tick++ {
				if tick > 64 {
					t.Fatal("cast did not complete")
				}
				Step(w, nil)
			}
			if w.entities[0].Mana != before-25 || len(w.effects) != 7 {
				t.Fatal("seventh cast cost/population", atUnit, duringWindup, before-w.entities[0].Mana, len(w.effects))
			}
			cold1164(t, w)
		}
	}
}

func TestArea1164MissingOwnerRefusesBeforePaintingOrSpending(t *testing.T) {
	for _, spell := range []uint16{2, 3} {
		w := retainedWallCells1162(t, true)
		w.savedMotion = nil // explicit missing owner, not a legitimate loaded state
		caster := effectMage(1, 15, 16, 1<<3)
		w.entities = []Entity{caster}
		fire := SpellRule{ID: spell, Area: true, Distribution: 4, Radius: 2, AreaDuration: 1}
		before := w.Hash()
		if w.areaLandingRefusal(fire, 0, 16, 16) == "" || w.landArea(fire, 0, 1, true, 15, 16, 16, 16, nil) || w.Hash() != before {
			t.Fatal("missing current owner allowed a partial cast")
		}
	}
}

func TestTheAdmissionPredicateAndTheLandingAgreeOnEveryAreaRow(t *testing.T) {
	for _, tc := range []struct {
		name    string
		rule    SpellRule
		prefill int
		refused bool
	}{
		{"a cloud at an empty anchor lands",
			SpellRule{ID: 3, Area: true, Distribution: 4, Radius: 2, AreaDuration: 15}, 0, false},
		{"Wall of Fire at a full anchor still lands",
			SpellRule{ID: 3, Area: true, Distribution: 4, Radius: 2, AreaDuration: 15}, cellEffectSlots, false},
		{"an ordinary cloud beyond six objects still lands",
			SpellRule{ID: 8, Area: true, Distribution: 3, Radius: 2, AreaDuration: 15}, cellEffectSlots, false},
		{"a blast is not held by a full anchor, because it retains no record",
			SpellRule{ID: 2, Area: true, Distribution: 3, Radius: 2}, cellEffectSlots, false},
		{"a staged row whose id has no cell program lands nothing",
			SpellRule{ID: 7, Area: true, Distribution: 5, Radius: 1}, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			filler := SpellRule{ID: 21, Area: true, Distribution: 5, Radius: 4}
			w, err := NewStockedSpelledWorld(0xc2, Bounds{Width: 40, Height: 40}, ModeCanonical,
				Terrain{}, nil, nil, Relations{}, nil, nil, []SpellRule{tc.rule, filler})
			if err != nil {
				t.Fatal(err)
			}
			for i := 0; i < tc.prefill; i++ {
				if !w.landArea(filler, 0, 0, false, 16, 20, 20, 20, nil) {
					t.Fatalf("the landing path refused filler record %d", i+1)
				}
			}
			said := w.areaLandingRefusal(tc.rule, 0, 20, 20) != ""
			if said != tc.refused {
				t.Fatalf("the admission predicate says refused=%v, want %v", said, tc.refused)
			}
			if got := !w.landArea(tc.rule, 0, 0, false, 16, 20, 20, 20, nil); got != said {
				t.Errorf("the landing refused=%v where admission said %v", got, said)
			}
		})
	}
}
