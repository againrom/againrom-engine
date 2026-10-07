package sim

import (
	"bytes"
	"strings"
	"testing"
)

func assertDeadActionCanonical1045(t *testing.T, w *World, id EntityID) {
	t.Helper()
	e := spAt(t, w, id)
	if e.Alive() || e.Decay == DecayNone || e.HasTarget || e.HasAttackTarget ||
		e.AttackPhase != AttackReady || e.AttackCountdown != 0 || e.CastWait != 0 {
		t.Fatalf("dead action state = alive=%v decay=%d target=%v attack=%v/%d/%d wait=%d",
			e.Alive(), e.Decay, e.HasTarget, e.HasAttackTarget, e.AttackPhase, e.AttackCountdown, e.CastWait)
	}
	if _, ok := w.bookCastIndex(id); ok {
		t.Fatalf("dead caster %d retains a book record: %+v", id, w.bookCasts)
	}
	form := mustMarshal(t, w)
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("canonical dead action did not decode: %v", err)
	}
	if back.Hash() != w.Hash() || !bytes.Equal(form, mustMarshal(t, &back)) {
		t.Fatal("dead action changed across form-62 decode or hash")
	}
}

func TestCadence1045EquipmentHealthLossNormalizesEveryProducer(t *testing.T) {
	healthItem := ItemInstance{Code: 1, Kind: 1, Effects: []ItemEffect{{Kind: 7, Operand: 10}}}
	plainItem := ItemInstance{Code: 2, Kind: 1}
	for _, tc := range []struct {
		name    string
		carried []ItemInstance
		command Command
	}{
		{"replacement", []ItemInstance{plainItem}, Command{Kind: KindEquip, Entity: 1, X: 0, Y: 12}},
		{"unequip", nil, Command{Kind: KindUnequip, Entity: 1, X: 12}},
		{"drop worn", nil, Command{Kind: KindDropWorn, Entity: 1, X: 3, Y: 3, Spell: 12}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			caster := spMage(1, 1, 1, 30, 20, 20, 1<<1)
			caster.HP, caster.MaxHP, caster.CastWait = 5, 100, 7
			var equipped [EquipSlots]ItemInstance
			equipped[11] = healthItem
			w, err := NewStockedWorld(1045, Bounds{Width: 8, Height: 8}, ModeCanonical,
				Terrain{}, []Entity{caster, spEnt(2, 2, 1)}, nil, Relations{}, nil,
				[]Stock{{ID: 1, ItemInstances: tc.carried, EquippedItems: equipped}})
			if err != nil {
				t.Fatalf("NewStockedWorld: %v", err)
			}
			w.bookCasts = []bookCast{{Caster: 1, Target: 2, Spell: 1, Phase: bookCharging,
				Remaining: 7, Retained: true}}
			Step(w, []Command{tc.command})
			if got := spAt(t, w, 1).HP; got != -5 {
				t.Fatalf("health after removing the ten-point item = %d, want -5", got)
			}
			assertDeadActionCanonical1045(t, w, 1)
		})
	}
}

func TestCadence1045SetDerivedNormalizesHealthDeathAtomically(t *testing.T) {
	caster := spMage(1, 1, 1, 30, 20, 20, 1<<1)
	caster.HP, caster.MaxHP, caster.CastWait = 5, 100, 7
	caster.AttackTarget, caster.HasAttackTarget = 2, true
	caster.AttackPhase, caster.AttackCountdown, caster.AttackRelax = AttackRelaxing, 3, 5
	w := spWorld(t, 1045, nil, caster, spEnt(2, 2, 1))
	w.bookCasts = []bookCast{{Caster: 1, Target: 2, Spell: 1, Phase: bookCharging,
		Remaining: 7, Retained: true}}
	if !w.SetDerived(1, DerivedBlock{MaxHP: -5, MaxMana: 20, Speed: 1, ScanRange: 1,
		Combat: CombatBlock{Defence: 20, Reach: 1}}) {
		t.Fatal("SetDerived refused the valid derived block")
	}
	e := spAt(t, w, 1)
	if e.HP != -5 || e.Defence != 10 {
		t.Fatalf("derived death left health/defence %d/%d, want -5/10", e.HP, e.Defence)
	}
	assertDeadActionCanonical1045(t, w, 1)
}

func TestCadence1045DeathAfterBookAdvanceRemovesTheRecordBeforeSave(t *testing.T) {
	build := func(withCast bool, laterCombat bool) *World {
		caster := spMage(1, 1, 1, 30, 20, 20, 1<<1)
		caster.HP, caster.MaxHP = 5, 100
		ents := []Entity{caster, spEnt(2, 2, 1)}
		if laterCombat {
			attacker := cbFighter(3, 0, 1, 1, 4)
			attacker.DamageBase, attacker.AlwaysHits = 20, true
			attacker.AttackTarget, attacker.HasAttackTarget = 1, true
			attacker.AttackPhase, attacker.AttackCountdown = AttackCharging, 0
			ents = append(ents, attacker)
		}
		w := spWorld(t, 1045, nil, ents...)
		if withCast {
			w.bookCasts = []bookCast{{Caster: 1, Target: 2, Spell: 1, Phase: bookCharging,
				Remaining: 7, Retained: true}}
		}
		return w
	}
	for _, tc := range []struct {
		name        string
		laterCombat bool
		commands    []Command
	}{
		{"command death", false, []Command{{Kind: KindKill, Entity: 1}}},
		{"later-id combat death", true, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			withCast, clean := build(true, tc.laterCombat), build(false, tc.laterCombat)
			Step(withCast, tc.commands)
			Step(clean, tc.commands)
			assertDeadActionCanonical1045(t, withCast, 1)
			if withCast.Hash() != clean.Hash() || !bytes.Equal(mustMarshal(t, withCast), mustMarshal(t, clean)) {
				t.Fatal("cancelled book record remains in the immediate hash or save")
			}
		})
	}
}

func TestCadence1045SelfKillingApplicationsDoNotRestoreRecovery(t *testing.T) {
	rule := SpellRule{ID: 2, Area: true, Radius: 2, School: 1, MaxRange: 8,
		DamageMin: 100, DamageMax: 100, Damaging: true}
	runUntilDead := func(t *testing.T, w *World, first []Command) {
		t.Helper()
		for tick := 0; tick < 32 && spAt(t, w, 1).Alive(); tick++ {
			Step(w, first)
			first = nil
		}
		assertDeadActionCanonical1045(t, w, 1)
	}
	for _, tc := range []struct {
		name  string
		build func(*testing.T) (*World, []Command)
	}{
		{"retained unit book", func(t *testing.T) (*World, []Command) {
			caster := spMage(1, 0, 0, 30, 20, 20, 1<<2)
			caster.HP, caster.AttackCharge, caster.AttackRelax = 5, 1, 12
			return spWorld(t, 1045, []SpellRule{rule}, caster, spEnt(2, 1, 0)), []Command{spCast(1, 2, 2)}
		}},
		{"retained cell book", func(t *testing.T) (*World, []Command) {
			caster := spMage(1, 0, 0, 30, 20, 20, 1<<2)
			caster.HP, caster.AttackCharge, caster.AttackRelax = 5, 1, 12
			return spWorld(t, 1045, []SpellRule{rule}, caster, spEnt(2, 1, 0)),
				[]Command{{Kind: KindCastAt, Entity: 1, X: 1, Y: 0, Spell: 2}}
		}},
		{"one-shot autocast", func(t *testing.T) (*World, []Command) {
			caster := spMage(1, 0, 0, 30, 20, 20, 1<<2)
			caster.HP, caster.AttackCharge, caster.AttackRelax, caster.AutoSpell, caster.Owner = 5, 1, 12, 2, 1
			victim := spEnt(2, 1, 0)
			victim.Owner = 2
			return hlWorld(t, 1045, acEnemies(t), []SpellRule{rule}, caster, victim), nil
		}},
		{"mage weapon diversion", func(t *testing.T) (*World, []Command) {
			caster := wpnCaster(1, 0, 0, 2, 30, 1, 12)
			caster.HP, caster.MaxMana, caster.Mana = 5, 20, 20
			return spWorld(t, 1045, []SpellRule{rule}, caster, spEnt(2, 1, 0)), []Command{cbOrder(1, 2)}
		}},
		{"fighter weapon rider", func(t *testing.T) (*World, []Command) {
			caster := wpnCaster(1, 0, 0, 2, 30, 1, 12)
			caster.HP, caster.MaxMana, caster.Mana = 5, 0, 0
			caster.DamageBase, caster.AlwaysHits = 1, true
			return spWorld(t, 1045, []SpellRule{rule}, caster, spEnt(2, 1, 0)), []Command{cbOrder(1, 2)}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, first := tc.build(t)
			runUntilDead(t, w, first)
		})
	}
}

func TestCadence1045AreaReleaseRebindsTheBookSweepAfterEveryRemovalShape(t *testing.T) {
	const (
		seed      = uint64(0x1045)
		releasing = EntityID(4)
		anchor    = EntityID(100)
	)
	rule := SpellRule{ID: 2, Area: true, Radius: 2, School: 1, MaxRange: 8,
		DamageMin: 100, DamageMax: 100, Damaging: true}
	for _, tc := range []struct {
		name          string
		felled        map[EntityID]bool
		removeRelease bool
	}{
		{"lower", map[EntityID]bool{2: true}, false},
		{"higher", map[EntityID]bool{6: true}, false},
		{"releasing caster", map[EntityID]bool{releasing: true}, false},
		{"multiple lower and higher", map[EntityID]bool{1: true, 2: true, 6: true, 7: true}, false},
		{"multiple lower, releasing caster and higher", map[EntityID]bool{1: true, 2: true, releasing: true, 6: true, 7: true}, false},
		{"all peers", map[EntityID]bool{1: true, 2: true, 3: true, 5: true, 6: true, 7: true}, false},
		{"one-shot with multiple lower and higher", map[EntityID]bool{1: true, 2: true, 6: true, 7: true}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entities := make([]Entity, 0, 8)
			for id := EntityID(1); id <= 7; id++ {
				e := spMage(id, 8, 8, 30, 20, 20, 1<<2)
				e.AttackRelax = 20
				if tc.felled[id] {
					e.HP = 5
				} else {
					e.HP, e.MaxHP = 1000, 1000
				}
				entities = append(entities, e)
			}
			target := spEnt(anchor, 8, 8)
			target.HP, target.MaxHP = 1000, 1000
			entities = append(entities, target)
			w := spWorld(t, seed, []SpellRule{rule}, entities...)
			for id := EntityID(1); id <= 7; id++ {
				cast := bookCast{Caster: id, Target: anchor, Spell: 2, X: 8, Y: 8, Phase: bookBoundaryOne,
					Complete: true, Retained: true}
				if id == releasing {
					cast.Phase, cast.Remaining, cast.Complete, cast.Retained =
						bookCharging, 1, false, !tc.removeRelease
				}
				w.bookCasts = append(w.bookCasts, cast)
			}

			released := w.stepBookCasts(nil, nil)

			for id := EntityID(1); id <= 7; id++ {
				i, ok := w.bookCastIndex(id)
				if tc.felled[id] {
					if ok {
						t.Errorf("felled caster %d retains book state %+v", id, w.bookCasts[i])
					}
					assertDeadActionCanonical1045(t, w, id)
					continue
				}
				if !ok {
					if id == releasing && tc.removeRelease {
						if got := spAt(t, w, id).CastWait; got < 20 {
							t.Errorf("one-shot releasing survivor wait = %d, want recovery", got)
						}
						continue
					}
					t.Errorf("surviving caster %d lost its book state", id)
					continue
				}
				got := w.bookCasts[i]
				if id == releasing {
					if got.Phase != bookRelaxing || got.Remaining < 20 || !got.Complete || !got.Retained {
						t.Errorf("releasing survivor state = %+v, want one recovery transition", got)
					}
				} else if got.Phase != bookBoundaryTwo || got.Remaining != 0 || !got.Complete || !got.Retained {
					t.Errorf("surviving caster %d state = %+v, want exactly boundary one to boundary two", id, got)
				}
			}
			if got := released[releasing]; got != !tc.felled[releasing] {
				t.Errorf("released[%d] = %v, want %v", releasing, got, !tc.felled[releasing])
			}
			wantRNG := seed
			for range 9 { // Eight area applications and one unconditional recovery draw.
				wantRNG += gamma
			}
			if w.rng.state != wantRNG {
				t.Errorf("rng state = %#x, want %#x after nine draws", w.rng.state, wantRNG)
			}
			form := mustMarshal(t, w)
			var back World
			if err := back.UnmarshalBinary(form); err != nil {
				t.Fatalf("post-release form 62 did not decode: %v", err)
			}
			if back.Hash() != w.Hash() || !bytes.Equal(form, mustMarshal(t, &back)) {
				t.Fatal("post-release state changed across form-62 decode or hash")
			}
		})
	}
}

func TestCadence1045BookApplyWritersRefuseDeadCasters(t *testing.T) {
	rule := SpellRule{ID: 2, Area: true, Radius: 2, School: 1, MaxRange: 8,
		DamageMin: 100, DamageMax: 100, Damaging: true}
	for _, tc := range []struct {
		name  string
		apply func(*World) bool
	}{
		{"unit form", func(w *World) bool { return w.castSpell(0, 2, 2, 1, 0, nil) }},
		{"cell form", func(w *World) bool { return w.castBookAt(0, 1, 0, 2, nil) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			caster := spMage(1, 0, 0, 30, 20, 20, 1<<2)
			caster.HP, caster.AttackRelax = 5, 12
			w := spWorld(t, 1045, []SpellRule{rule}, caster, spEnt(2, 1, 0))
			if !tc.apply(w) {
				t.Fatal("self-killing area application was refused")
			}
			assertDeadActionCanonical1045(t, w, 1)
		})
	}
}

func TestCadence1045ManaLostDuringChargeSeparatesRetainedAndOneShot(t *testing.T) {
	rule := SpellRule{ID: 1, ManaCost: 5, School: 1, MaxRange: 8,
		DamageMin: 1, DamageMax: 1, TargetsUnit: true, Damaging: true}
	for _, tc := range []struct {
		name     string
		retained bool
		arm      func(*World)
	}{
		{"retained", true, func(w *World) {
			if !w.beginBookSpell(0, 2, 1) {
				t.Fatal("retained AI producer refused the fixture")
			}
		}},
		{"one-shot", false, func(w *World) {
			w.entities[0].AutoSpell, w.entities[0].Owner, w.entities[1].Owner = 1, 1, 2
			Step(w, nil)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			caster := spMage(1, 0, 0, 30, 5, 5, 1<<1)
			caster.AttackCharge = 1
			var w *World
			if tc.retained {
				w = spWorld(t, 1045, []SpellRule{rule}, caster, spEnt(2, 1, 0))
			} else {
				w = hlWorld(t, 1045, acEnemies(t), []SpellRule{rule}, caster, spEnt(2, 1, 0))
			}
			tc.arm(w)
			if len(w.bookCasts) != 1 || w.bookCasts[0].Phase != bookCharging {
				t.Fatalf("producer did not begin charging: %+v", w.bookCasts)
			}
			w.entities[0].Mana = 0
			for tick := 0; tick < castPeriod; tick++ {
				Step(w, nil)
			}
			if got := spAt(t, w, 2).HP; got != 100 {
				t.Fatalf("mana-lost cast applied and left target health %d", got)
			}
			if tc.retained {
				if len(w.bookCasts) != 1 || !w.bookCasts[0].Retained || w.bookCasts[0].Phase != bookPending ||
					w.bookCasts[0].Remaining != 0 || w.bookCasts[0].Complete || w.bookCasts[0].Progress != 0 {
					t.Fatalf("retained release refusal = %+v", w.bookCasts)
				}
			} else if len(w.bookCasts) != 0 {
				t.Fatalf("one-shot release refusal retained %+v", w.bookCasts)
			}
			if got := spAt(t, w, 1).CastWait; got != 0 {
				t.Fatalf("mana-lost release started recovery %d", got)
			}
		})
	}
}

func TestCadenceForm62RefusesBookStateWithoutALiveCaster(t *testing.T) {
	cast := bookCast{Caster: 1, Target: 2, Spell: 1, Phase: bookCharging, Remaining: 3, Retained: true}
	for _, tc := range []struct {
		name string
		edit func(*World)
		want string
	}{
		{"absent", func(w *World) { w.bookCasts[0].Caster = 99 }, "absent caster"},
		{"dead", func(w *World) {
			e := &w.entities[0]
			e.HP, e.Decay, e.Dwell = -1, DecayFallen, dwellOf(*e)
		}, "not alive"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := cadenceFormWorld(t, cast)
			tc.edit(w)
			var back World
			err := back.UnmarshalBinary(mustMarshal(t, w))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("decode error = %v, want %q", err, tc.want)
			}
		})
	}
}
