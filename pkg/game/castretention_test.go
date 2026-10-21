package game

import (
	"encoding/json"
	"fmt"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestWalkingManualBookCastSAVRetention(t *testing.T) {
	for _, creature := range []bool{false, true} {
		for _, cell := range []bool{false, true} {
			t.Run(fmt.Sprintf("creature=%v/cell=%v", creature, cell), func(t *testing.T) {
				f := castOrderFront(t)
				actors := f.live.world.Entities()
				actors[1].X = 25
				if creature {
					actors[0].CreatureSpells[0] = sim.CreatureSpell{ID: 6}
				} else {
					actors[0].Humanoid, actors[0].TypeID, actors[0].Class = true, 50, 50
					f.Table.Humans.(dbCollection)[5].params[16] = 50
				}
				w, err := sim.NewSpelledWorld(1, f.live.world.Bounds(), sim.ModeCanonical, nil, actors, nil, mapload.SpellRules(f.Table))
				if err != nil {
					t.Fatal(err)
				}
				f.live.world, f.live.mission.state.World = w, w
				command, kind := sim.Cast(1, 2, 6), byte(8)
				if cell {
					command, kind = sim.CastAt(1, 3, sim.CellPoint{X: 25, Y: 10}), 9
				}
				sim.Step(w, []sim.Command{command})
				if books := w.Actions().Books; len(books) != 1 || books[0].Phase != sim.BookPhaseApproach || !books[0].Retained {
					t.Fatalf("walking book order missing: %+v", books)
				}
				raw, doc, actions := saveCurrentEffect(t, f)
				record := castOrderRecord(t, doc, actions, 1)
				if creature && record.Class != "Unit" || !creature && record.Class != "Human" {
					t.Fatalf("caster class=%s, creature=%v", record.Class, creature)
				}
				order := savedRecordRawForTest(t, record, "U158")
				t.Logf("%s order kind=%d progress=%d retention=%d", record.Class, order[8], order[9], order[0x60])
				if order[8] != kind || order[9] != 0 || order[0x60] != 0 {
					t.Errorf("walking manual order: kind=%d progress=%d retention=%d; want %d/0/0", order[8], order[9], order[0x60], kind)
				}
				cold := openCurrentEffectSave(t, f, raw)
				legacy, err := sav.CloneDocumentData(doc)
				if err != nil {
					t.Fatal(err)
				}
				for _, binding := range actions.Bindings {
					if binding.ID == 1 && !binding.Structure {
						r := &legacy.Objects[binding.Object-1]
						old, err := savedMotionRaw(r, "U158", 148)
						if err != nil {
							t.Fatal(err)
						}
						if creature {
							old[0x60] = 1
						}
					}
				}
				legacyActions, err := readCurrentActions(&legacy)
				if err != nil {
					t.Fatal(err)
				}
				if err := finalizeCurrentOrderProjection(&legacy, legacyActions, w); err != nil {
					t.Fatal(err)
				}
				leaf, err := json.Marshal(legacyActions)
				if err != nil {
					t.Fatal(err)
				}
				if err := sav.SetNativeActions(&legacy.State, leaf); err != nil {
					t.Fatal(err)
				}
				oldRaw, err := sav.EncodeDocumentData(legacy)
				if err != nil {
					t.Fatal(err)
				}
				old := openCurrentEffectSave(t, f, oldRaw)
				landed := false
				for tick := 1; tick <= 300; tick++ {
					events := sim.StepObserved(w, nil)
					sim.Step(cold.live.world, nil)
					sim.Step(old.live.world, nil)
					if w.Hash() != cold.live.world.Hash() || w.Hash() != old.live.world.Hash() {
						t.Fatalf("LOAD continuation differs at tick %d", tick)
					}
					for _, event := range events {
						if event.Caster == 1 && event.AtCell == cell {
							landed = true
							t.Logf("cast lands at tick %d; fixed LOAD, old-byte LOAD and uninterrupted hashes equal", tick)
						}
					}
				}
				if !landed || w.Entities()[0].X <= actors[0].X {
					t.Fatal("walking cast did not land")
				}
			})
		}
	}
}

func TestRetainedCreatureBookCastSAVRetention(t *testing.T) {
	for _, control := range []struct {
		name                     string
		approach, pursue, second bool
	}{
		{"charging", false, false, false},
		{"pursuing", true, true, false},
		{"second game manual walk", true, false, true},
	} {
		t.Run(control.name, func(t *testing.T) {
			f := castOrderFront(t)
			actors := f.live.world.Entities()
			actors[0].CreatureSpells[0] = sim.CreatureSpell{ID: 6}
			var script *sim.Script
			if control.second {
				var err error
				script, err = sim.NewROM2Script(nil, nil, nil)
				if err != nil {
					t.Fatal(err)
				}
			}
			w, err := sim.NewSpelledWorld(1, f.live.world.Bounds(), sim.ModeCanonical, nil, actors, script, mapload.SpellRules(f.Table))
			if err != nil {
				t.Fatal(err)
			}
			f.live.world, f.live.mission.state.World = w, w
			sim.Step(w, []sim.Command{sim.Cast(1, 2, 6)})
			a := w.Actions()
			if len(a.Books) != 1 {
				t.Fatal("fixture cast missing")
			}
			a.Books[0].Retained = true
			if control.approach {
				a.Books[0].Phase, a.Books[0].Remaining, a.Books[0].Paid = sim.BookPhaseApproach, 0, false
			}
			if control.pursue {
				a.Actors[0].HasAttackTarget, a.Actors[0].AttackTarget = true, 2
				a.Actors[0].AttackTargetKind = sim.AttackTargetUnit
			}
			if err := w.RestoreActions(a, nil); err != nil {
				t.Fatal(err)
			}
			_, doc, actions := saveCurrentEffect(t, f)
			order := savedRecordRawForTest(t, castOrderRecord(t, doc, actions, 1), "U158")
			if order[0x60] != 1 {
				t.Fatalf("retained creature cast flag=%d, want 1", order[0x60])
			}
		})
	}
}
