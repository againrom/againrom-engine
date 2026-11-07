package sim

import "testing"

func TestComputerMageSelfShield(t *testing.T) {
	for _, tc := range []struct {
		name       string
		second     bool
		owner      uint32
		control    uint32
		mana       int32
		heal       bool
		shield     bool
		setting    AutoCastSetting
		wantCast   bool
		shieldCost uint16
	}{
		{"ROM2 scenario owner", true, 7, 1, 60, true, true, 0, true, 3},
		{"ROM2 nonzero control word", true, 7, 0x100, 11, true, true, 0, true, 3},
		{"ROM2 computer in self slot ignores levels", true, SelfSlot, 1, 11, true, true, AutoCastMaximum, true, 3},
		{"ROM2 mana ten refuses", true, 7, 1, 10, true, true, 0, false, 3},
		{"ROM2 computer self respects mana gate", true, SelfSlot, 1, 10, true, true, AutoCastMaximum, false, 3},
		{"ROM2 no Heal refuses", true, 7, 1, 60, false, true, 0, false, 3},
		{"ROM2 computer self respects Heal gate", true, SelfSlot, 1, 60, false, true, AutoCastMaximum, false, 3},
		{"ROM2 no Shield refuses", true, 7, 1, 60, true, false, 0, false, 3},
		{"ROM2 unaffordable Shield refuses", true, 7, 1, 11, true, true, 0, false, 12},
		{"ROM2 human other slot keeps existing policy", true, 7, 0, 60, true, true, AutoCastMinimum, false, 3},
		{"ROM2 human self keeps minimum", true, SelfSlot, 0, 5, false, true, AutoCastMinimum, true, 3},
		{"ROM1 computer other slot stays unchanged", false, 7, 1, 60, true, true, AutoCastMinimum, false, 3},
		{"ROM1 human self keeps minimum", false, SelfSlot, 0, 5, false, true, AutoCastMinimum, true, 3},
		{"ROM1 computer self keeps existing slot policy", false, SelfSlot, 1, 5, false, true, AutoCastMinimum, true, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, shield, _ := computerSupportFixture(t, tc.second, tc.owner, tc.control, tc.mana, tc.heal, tc.shield, tc.shieldCost)
			if tc.setting != 0 && !w.SetAutoCast(tc.owner, tc.setting|AutoCastOwn) {
				t.Fatal("setting refused")
			}
			initial := w.entities[0].Mana
			casts, first := 0, 0
			for tick := 1; tick <= 160; tick++ {
				for _, event := range StepObserved(w, nil) {
					if event.Caster != 1 {
						continue
					}
					if event.Spell != shield || event.Target != 1 {
						t.Fatalf("unexpected spell %d at %d", event.Spell, event.Target)
					}
					casts++
					if first == 0 {
						first = tick
					}
				}
				if tick == 1 {
					at, admitted := w.bookCastIndex(1)
					if admitted != tc.wantCast || admitted && (w.bookCasts[at].Spell != shield || w.bookCasts[at].Target != 1) {
						t.Errorf("first idle tick: admitted %v, casts %+v, want Shield %v", admitted, w.bookCasts, tc.wantCast)
					}
				}
			}
			wantCasts, wantMana, wantAbsorption := 0, initial, int32(0)
			if tc.wantCast {
				wantCasts, wantMana, wantAbsorption = 1, initial-int32(tc.shieldCost), 3
			}
			if casts != wantCasts || w.entities[0].Mana != wantMana || w.entities[0].Absorption != wantAbsorption {
				t.Fatalf("casts %d mana %d absorption %d, want %d/%d/%d", casts, w.entities[0].Mana, w.entities[0].Absorption, wantCasts, wantMana, wantAbsorption)
			}
			t.Logf("first release tick %d; casts %d; mana %d; absorption %d", first, casts, wantMana, wantAbsorption)
		})
	}
}

func TestComputerSupportUsesTheExistingAdmission(t *testing.T) {
	for _, tc := range []struct {
		name  string
		apply func(*World)
		spell bool
	}{
		{"self Heal precedes Shield", func(w *World) { w.entities[0].HP = 50 }, true},
		{"a standing Shield is not refreshed", func(w *World) {
			w.attached = append(w.attached, attachedEffect{Target: 1, Spell: 27, Remaining: 200, Kind: EffectAbsorption, Mode: EffectDuration, Magnitude: 3})
		}, false},
		{"a paid cast keeps the caster", func(w *World) {
			if !w.beginBookSpellOnce(0, 1, 27) {
				t.Fatal("fixture Shield refused")
			}
		}, false},
		{"a movement order keeps the caster", func(w *World) {
			w.entities[0].HasTarget, w.entities[0].TargetX, w.entities[0].TargetY = true, 10, 2
		}, false},
		{"an attack order keeps the caster", func(w *World) {
			w.entities[0].HasAttackTarget, w.entities[0].AttackTarget, w.entities[0].AttackTargetKind = true, 2, AttackTargetUnit
		}, false},
		{"a hostile striking the wounded caster keeps it", func(w *World) {
			w.entities[0].HP = 50
			w.relations.Set(7, SelfSlot, relationHostile)
			w.relations.Set(SelfSlot, 7, relationHostile)
			w.entities[1].HasAttackTarget, w.entities[1].AttackTarget, w.entities[1].AttackTargetKind = true, 1, AttackTargetUnit
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, _, heal := computerSupportFixture(t, true, 7, 1, 11, true, true, 3)
			tc.apply(w)
			beforeMana, beforeCasts := w.entities[0].Mana, len(w.bookCasts)
			Step(w, nil)
			if tc.spell {
				at, ok := w.bookCastIndex(1)
				if !ok || w.bookCasts[at].Spell != heal || w.bookCasts[at].Target != 1 {
					t.Fatalf("self Heal did not precede Shield: %+v", w.bookCasts)
				}
			} else if w.entities[0].Mana != beforeMana || len(w.bookCasts) > beforeCasts {
				t.Fatalf("refused support spent mana or started a cast: %d -> %d, %+v", beforeMana, w.entities[0].Mana, w.bookCasts)
			}
		})
	}
}

func TestComputerSupportSurvivesColdRestore(t *testing.T) {
	w, shield, _ := computerSupportFixture(t, true, 7, 1, 11, true, true, 3)
	cold := worldRoundTripForTest(t, w)
	for tick := 1; tick <= 160; tick++ {
		Step(w, nil)
		Step(cold, nil)
		if w.Hash() != cold.Hash() {
			t.Fatalf("cold restore changed tick %d", tick)
		}
		if tick == 1 {
			for _, live := range []*World{w, cold} {
				at, ok := live.bookCastIndex(1)
				if !ok || live.bookCasts[at].Spell != shield || live.bookCasts[at].Target != 1 {
					t.Fatalf("restored computer support did not start Shield: %+v", live.bookCasts)
				}
			}
			cold = worldRoundTripForTest(t, cold)
		}
	}
	if w.entities[0].Mana != 8 || w.entities[0].Absorption != 3 || !w.HasEffectArm(1, autoCastRoleArm(autoCastShielder)) {
		t.Fatal("Shield effect or its single mana charge was lost")
	}
}

func computerSupportFixture(t *testing.T, second bool, owner, control uint32, mana int32, knowsHeal, knowsShield bool, cost uint16) (*World, uint16, uint16) {
	t.Helper()
	shield, heal := acbShield(), hlHeal()
	shield.ManaCost, heal.ManaCost = int32(cost), 3
	if second {
		shield.ID, heal.ID = 27, 24
		shield, heal = secondRow(shield), secondRow(heal)
	}
	var known uint32
	if knowsHeal {
		known |= 1 << heal.ID
	}
	if knowsShield {
		known |= 1 << shield.ID
	}
	mage := spMage(1, 2, 2, 30, 60, mana, known)
	mage.Owner, mage.Book.State = owner, BookPresent
	if knowsShield {
		mage.Book.Slots[shield.ID-1] = BookSpell{Defensive: 1, ManaCost: cost}
	}
	if knowsHeal {
		mage.Book.Slots[heal.ID-1] = BookSpell{Range: 6, Defensive: 1, ManaCost: 3}
	}
	companion := spEnt(2, 6, 2)
	companion.Owner = SelfSlot
	var script *Script
	if second {
		var err error
		script, err = newScript(ScriptROM2, nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
	}
	w, err := NewStockedSpelledWorld(31, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{}, []Entity{mage, companion}, script, engRel(t,
		[3]uint32{owner, SelfSlot, relationLocked},
		[3]uint32{SelfSlot, owner, relationLocked}), nil, nil, []SpellRule{shield, heal})
	if err != nil {
		t.Fatal(err)
	}
	players := []SavedGroupPlayer{{ID: 91, Slot: owner}}
	controls := []PlayerParticipant{{PlayerID: 91, Value: control}}
	if owner != SelfSlot {
		players = append(players, SavedGroupPlayer{ID: 103, Slot: SelfSlot})
		controls = append(controls, PlayerParticipant{PlayerID: 103, Value: 0})
	}
	if err := w.RestoreCurrentPlayers(players, controls, true); err != nil {
		t.Fatal(err)
	}
	if !w.rom2ActivityMask().actorActive(&w.entities[0]) || !w.autoCastMoment(0) {
		t.Fatal("fixture caster is inactive or busy")
	}
	holdGroups(w)
	return w, shield.ID, heal.ID
}
