package sim

import "testing"

// These tests exercise native producers only. The literal stride is obtained
// by a real rated Step, not installed directly or reconstructed from facing.
func strideRelocationWorld1115(t *testing.T) *World {
	t.Helper()
	caster := effectMage(1, 2, 2, 1<<26)
	caster.Speed, caster.Facing = 16, 64
	beacon := spEnt(2, 7, 5)
	beacon.Domain, beacon.Owner = DomainAir, SelfSlot
	rule := SpellRule{ID: 26, ManaCost: 5, School: 5, MaxRange: 15, Defensive: true}
	w := hlWorld(t, 1115, Relations{}, []SpellRule{rule}, caster, beacon)
	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 9, Y: 2}})
	strideRelocationWant1115(t, w.entities[0], 15)
	return w
}

func strideRelocationWant1115(t *testing.T, e Entity, transit uint16) {
	t.Helper()
	want := NativeStride{Present: true, FromX: 2, FromY: 2, ToX: 3, ToY: 2,
		Rate: 16, StepX: 16, StepY: 0, Direction: 2}
	if e.Stride != want || e.X != 3 || e.Y != 2 || e.Transit != transit || e.TransitTotal != 16 {
		t.Fatalf("stride/position/crossing = %+v (%d,%d) %d/%d, want %+v (3,2) %d/16",
			e.Stride, e.X, e.Y, e.Transit, e.TransitTotal, want, transit)
	}
}

func strideRelocationAbsent1115(t *testing.T, e Entity, x, y int32, transit, total uint16) {
	t.Helper()
	if e.Stride != (NativeStride{}) || e.X != x || e.Y != y || e.Transit != transit || e.TransitTotal != total {
		t.Fatalf("relocated stride/position/crossing = %+v (%d,%d) %d/%d, want absent (%d,%d) %d/%d",
			e.Stride, e.X, e.Y, e.Transit, e.TransitTotal, x, y, transit, total)
	}
}

func TestNativeStrideRelocation1115CallbacksPreserveCoarseTransit(t *testing.T) {
	// Release callbacks isolate the relocation itself: no intervening Step
	// can repay a transit tick and disguise an unintended counter rewrite.
	// Full book and scroll admission/release paths are exercised below.
	for _, kind := range []string{"unit-effect", "point-book", "point-scroll", "unit-scroll", "script-place"} {
		for _, blocked := range []bool{false, true} {
			name := kind + "/open"
			if blocked {
				name = kind + "/blocked"
			}
			t.Run(name, func(t *testing.T) {
				w := strideRelocationWorld1115(t)
				if blocked {
					// The beacon changes to the caster's layer; the exact
					// destination becomes occupied without moving either actor.
					w.entities[1].Domain = DomainGround
				}
				before := w.Hash()
				switch kind {
				case "unit-effect":
					if got := w.ordinaryEffectPayload(0, 1, w.spells[0], 30); !got {
						t.Fatalf("unit payload applied=%v, blocked=%v", got, blocked)
					}
				case "point-book":
					if got := w.castBookAt(0, 7, 5, 26, nil); !got {
						t.Fatalf("point book applied=%v, blocked=%v", got, blocked)
					}
				case "point-scroll":
					w.releaseScroll(0, ScrollCast{Caster: 1, AtCell: true, X: 7, Y: 5}, w.spells[0], 30, nil)
				case "unit-scroll":
					w.releaseScroll(0, ScrollCast{Caster: 1, Target: 2, X: 7, Y: 5}, w.spells[0], 30, nil)
				case "script-place":
					if got := w.placeAt(0, 7, 5); got == blocked {
						t.Fatalf("placement applied=%v, blocked=%v", got, blocked)
					}
				}
				if blocked {
					if kind == "script-place" && w.Hash() != before {
						t.Fatal("refused relocation mutated the world")
					}
					strideRelocationWant1115(t, w.entities[0], 15)
					return
				}
				strideRelocationAbsent1115(t, w.entities[0], 7, 5, 15, 16)
			})
		}
	}
}

func TestNativeStrideRelocation1115ManualBookDispatchAndFacing(t *testing.T) {
	for _, point := range []bool{false, true} {
		name := "unit"
		if point {
			name = "point"
		}
		t.Run(name, func(t *testing.T) {
			w := strideRelocationWorld1115(t)
			command := Command{Kind: KindCast, Entity: 1, X: 2, Y: 26}
			if point {
				command = Command{Kind: KindCastAt, Entity: 1, X: 7, Y: 5, Spell: 26}
			}
			if events := StepObserved(w, []Command{command}); len(events) != 0 {
				t.Fatal("book released before its wind-up")
			}
			if len(w.bookCasts) != 1 || w.entities[0].Facing != 96 {
				t.Fatalf("cast admission/facing = %+v facing %d, want one cast facing southeast", w.bookCasts, w.entities[0].Facing)
			}
			// Manual admission turns toward the cast and cancels the old stride.
			strideRelocationAbsent1115(t, w.entities[0], 3, 2, 0, 0)
			released := false
			for n := 0; n < 64; n++ {
				events := StepObserved(w, nil)
				if len(events) != 0 {
					if len(events) != 1 || events[0].Spell != 26 || events[0].ToX != 7 || events[0].ToY != 5 {
						t.Fatalf("unexpected release: %+v", events)
					}
					released = true
					break
				}
				strideRelocationAbsent1115(t, w.entities[0], 3, 2, 0, 0)
			}
			if !released || w.entities[0].Mana != 95 {
				t.Fatalf("book release=%v mana=%d", released, w.entities[0].Mana)
			}
			// Relocation cannot resume the stride cancelled by the manual click.
			strideRelocationAbsent1115(t, w.entities[0], 7, 5, 0, 0)
		})
	}
}

func TestNativeStrideRelocation1115ScrollDispatch(t *testing.T) {
	for _, point := range []bool{false, true} {
		name := "unit"
		if point {
			name = "point"
		}
		t.Run(name, func(t *testing.T) {
			w := strideRelocationWorld1115(t)
			w.spells[0].TargetsUnit = !point
			item := ItemInstance{Code: 0xe10, Kind: 4, Price: 17, Effects: []ItemEffect{{Kind: 41, Operand: 26 | 30<<16}}}
			w.carried[0] = []ItemStack{StackItem(item, 1)}
			command := Command{Kind: KindUseScroll, Entity: 1, X: 2}
			if point {
				command = Command{Kind: KindUseScrollAt, Entity: 1, X: 7, Y: 5}
			}
			Step(w, []Command{command})
			if len(w.scrollCasts) != 1 || len(w.carried[0]) != 0 {
				t.Fatal("scroll was not reserved by command dispatch")
			}
			strideRelocationWant1115(t, w.entities[0], 14)
			started := false
			released := false
			for n := 0; n < 96; n++ {
				events := StepObserved(w, nil)
				if len(events) != 0 {
					if len(events) != 1 || events[0].Spell != 26 || events[0].ToX != 7 || events[0].ToY != 5 {
						t.Fatalf("unexpected release: %+v", events)
					}
					released = true
					break
				}
				remaining := uint16(0)
				if n < 14 {
					remaining = uint16(13 - n)
				}
				strideRelocationWant1115(t, w.entities[0], remaining)
				if len(w.scrollCasts) == 1 && w.scrollCasts[0].Started {
					started = true
					if w.entities[0].Transit != 0 {
						t.Fatal("scroll wind-up started before the crossing was paid")
					}
				}
			}
			if !started || !released || len(w.scrollCasts) != 0 || len(w.carried[0]) != 0 || w.entities[0].Mana != 100 {
				t.Fatalf("scroll started=%v released=%v casts=%+v stock=%+v mana=%d",
					started, released, w.scrollCasts, w.carried[0], w.entities[0].Mana)
			}
			strideRelocationAbsent1115(t, w.entities[0], 7, 5, 0, 16)
		})
	}
}

func TestNativeStrideRelocation1115ScriptReturnAndSameCell(t *testing.T) {
	t.Run("script removal preserves and same-cell return invalidates", func(t *testing.T) {
		w := strideRelocationWorld1115(t)
		w.runInstant(prOffNode(1))
		if !w.entities[0].OffMap {
			t.Fatal("script removal did not run")
		}
		strideRelocationWant1115(t, w.entities[0], 15)
		for n := 0; n < 3; n++ {
			Step(w, nil)
			strideRelocationWant1115(t, w.entities[0], 15)
		}
		w.runInstant(prOnNode(1))
		if w.entities[0].OffMap {
			t.Fatal("script return left actor off map")
		}
		strideRelocationAbsent1115(t, w.entities[0], 3, 2, 15, 16)
	})
	t.Run("on-map same-cell placement invalidates", func(t *testing.T) {
		w := strideRelocationWorld1115(t)
		if !w.placeAt(0, 3, 2) {
			t.Fatal("same-cell placement refused")
		}
		strideRelocationAbsent1115(t, w.entities[0], 3, 2, 15, 16)
	})
	t.Run("failed return preserves entity and stride", func(t *testing.T) {
		w := strideRelocationWorld1115(t)
		w.runInstant(prOffNode(1))
		for i := range w.grid {
			w.grid[i] = 1
		}
		before := w.entities[0]
		w.runInstant(prOnNode(1))
		// Return attempts consume RNG even on failure, so compare the
		// complete actor, not the world digest which includes those draws.
		if w.entities[0] != before {
			t.Fatal("failed script return changed the actor")
		}
	})
}

func TestNativeStrideRelocation1115HeadlessDeathAndCompaction(t *testing.T) {
	t.Run("headless teleport", func(t *testing.T) {
		w := strideRelocationWorld1115(t)
		before := w.Hash()
		if err := w.HeadlessPlace(1, -1, 2); err == nil || w.Hash() != before {
			t.Fatal("invalid headless destination succeeded or changed state")
		}
		transit, total := w.entities[0].Transit, w.entities[0].TransitTotal
		if err := w.HeadlessPlace(1, 11, 10); err != nil {
			t.Fatal(err)
		}
		e := headlessPlacedNear(t, w, 1, 11, 10)
		strideRelocationAbsent1115(t, e, e.X, e.Y, transit, total)
	})
	t.Run("ordinary death clears crossing", func(t *testing.T) {
		w := strideRelocationWorld1115(t)
		Step(w, []Command{{Kind: KindDamage, Entity: 1, X: 100}})
		if w.entities[0].Alive() {
			t.Fatal("damage did not kill the actor")
		}
		strideRelocationAbsent1115(t, w.entities[0], 3, 2, 0, 0)
	})
	t.Run("compaction preserves survivor identity", func(t *testing.T) {
		a := spEnt(1, 2, 2)
		b := spEnt(3, 2, 5)
		a.Speed, b.Speed = 16, 32
		a.Facing, b.Facing = 64, 64
		w := spWorld(t, 1115, nil, a, b)
		Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 9, Y: 2}, {Kind: KindMoveTo, Entity: 3, X: 9, Y: 5}})
		strideRelocationWant1115(t, w.entities[0], 15)
		want := NativeStride{Present: true, FromX: 2, FromY: 5, ToX: 3, ToY: 5,
			Rate: 32, StepX: 32, Direction: 2}
		if w.entities[1].Stride != want || w.entities[1].Transit != 7 || w.entities[1].TransitTotal != 8 {
			t.Fatalf("survivor did not take the independent faster step: %+v", w.entities[1])
		}
		before := w.entities[1]
		// remove is the production terminal/Control Spirit compactor; the
		// callback is isolated so survivor transit cannot advance meanwhile.
		w.remove([]EntityID{1})
		if len(w.entities) != 1 || w.entities[0] != before {
			t.Fatal("compaction moved another actor's stride onto the survivor")
		}
		form, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var loaded World
		if err := loaded.UnmarshalBinary(form); err != nil {
			t.Fatal(err)
		}
		if len(loaded.entities) != 1 || loaded.entities[0] != before || loaded.Hash() != w.Hash() {
			t.Fatal("compacted survivor did not persist exactly")
		}
	})
}

func TestNativeStrideRelocation1115RateInputChangesPreserveAcceptedStep(t *testing.T) {
	w := strideRelocationWorld1115(t)
	if landed, ok := w.applyEffectDelta(0, EffectSpeed, 9); !ok || landed != 9 || w.entities[0].Speed != 25 {
		t.Fatal("speed effect did not change the next-step input")
	}
	strideRelocationWant1115(t, w.entities[0], 15)
	if !w.SetHumanMovement(1, 7, 12) || w.entities[0].Load != 12 {
		t.Fatal("Human speed/load producer did not apply")
	}
	strideRelocationWant1115(t, w.entities[0], 15)
	if !w.SetDerived(1, DerivedBlock{Combat: CombatBlock{Reach: 1}, MaxHP: 100, MaxMana: 100, Speed: 31, Capacity: 100}) || w.entities[0].Speed != 31 {
		t.Fatal("derived equipment/stat producer did not apply")
	}
	strideRelocationWant1115(t, w.entities[0], 15)
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var loaded World
	if err := loaded.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	strideRelocationWant1115(t, loaded.entities[0], 15)
	if loaded.entities[0].Speed != 31 || loaded.entities[0].Load != 12 {
		t.Fatal("native reload replaced current inputs with accepted stride inputs")
	}
}
