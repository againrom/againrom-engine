package sim

import (
	"bytes"
	"fmt"
	"slices"
	"testing"
)

func TestDamageMessagesKeepPhysicalApplicationOrder(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		for _, zeroFirst := range []bool{false, true} {
			a, b, victim := cbFighter(1, 0, 0, 1, 7), cbFighter(2, 2, 0, 1, 11), cbEnt(3, 1, 0)
			a.AlwaysHits, b.AlwaysHits = true, true
			a.DamageBase, a.DamageSpread, b.DamageBase, b.DamageSpread = 7, 0, 11, 0
			if zeroFirst {
				a.DamageBase, a.DamageSpread = 0, 0
			}
			entities := []Entity{a, b, victim}
			if reverse {
				slices.Reverse(entities)
			}
			plain, observed := cbWorld(t, 99, entities...), cbWorld(t, 99, entities...)
			commands := []Command{cbOrder(1, 3), cbOrder(2, 3)}
			report := StepReported(observed, commands)
			Step(plain, commands)
			want := []DamageEvent{{Target: 3, BeforeHP: 100, AfterHP: 93}, {Target: 3, BeforeHP: 93, AfterHP: 82}}
			if zeroFirst {
				want = []DamageEvent{{Target: 3, BeforeHP: 100, AfterHP: 100}, {Target: 3, BeforeHP: 100, AfterHP: 89}}
			}
			if !bytes.Equal(hlBytes(t, observed), hlBytes(t, plain)) || observed.Hash() != plain.Hash() {
				t.Fatal("observation changed canonical state")
			}
			t.Logf("reverse=%v zeroFirst=%v: observed/plain World bytes equal; hash=%x finalHP=%d", reverse, zeroFirst, observed.Hash(), cbAt(t, observed, 3).HP)
			if !slices.Equal(report.Damages, want) {
				t.Fatalf("reverse=%v zeroFirst=%v: messages=%+v want %+v", reverse, zeroFirst, report.Damages, want)
			}
		}
	}
}

func TestDamageMessagesKeepDrainVictimAndExcludeCasterRestoration(t *testing.T) {
	rule := SpellRule{ID: 11, School: 1, MaxRange: 7, DamageMin: 3, DamageMax: 3, TargetsUnit: true}
	caster := wpnCaster(1, 0, 0, 11, 30, 1, 0)
	caster.HP = 50
	w := spWorld(t, 1, []SpellRule{rule}, caster, spEnt(2, 1, 0))
	want := []DamageEvent{{Target: 2, BeforeHP: 100, AfterHP: 94}}
	if got := StepReported(w, []Command{Attack(1, 2)}).Damages; !slices.Equal(got, want) || cbAt(t, w, 1).HP != 56 {
		t.Fatalf("Drain victim/caster=%+v HP=%d", got, cbAt(t, w, 1).HP)
	}
}

func TestDamageMessagesKeepSacrificeExpenditureBeforeAreaHit(t *testing.T) {
	rule := SpellRule{ID: 4, Area: true, Distribution: distributionStaged, School: 1, Damaging: true}
	caster, victim := spEnt(1, 1, 1), spEnt(2, 16, 16)
	w := structureAreaWorld(t, nil, caster, victim)
	w.spells = []SpellRule{rule}
	cells := w.ringStageCells(cellEffect{Key: 0x1010, Spell: 4}, 0)
	w.entities[1].X, w.entities[1].Y = keyCell(cells[0])
	w.damageObservation = &damageObservation{}
	if !w.landArea(rule, 0, 1, true, 1, 1, 16, 16, nil) {
		t.Fatal("sacrifice refused")
	}
	want := []DamageEvent{{Target: 1, BeforeHP: 100, AfterHP: 1}, {Target: 2, BeforeHP: 100, AfterHP: 0}}
	if !slices.Equal(w.damageObservation.events, want) {
		t.Fatalf("sacrifice=%+v want %+v", w.damageObservation.events, want)
	}
	w.damageObservation = nil
}

func TestDamageMessagesSkipMaximumHealthLossOfNativeEquipmentCommands(t *testing.T) {
	for _, cmd := range []Command{Equip(1, 0, 12), Unequip(1, 12), DropWorn(1, 12, CellPoint{X: 3, Y: 3})} {
		actor := spMage(1, 1, 1, 30, 20, 20, 1<<1)
		actor.HP = 5
		var equipment [EquipSlots]ItemInstance
		equipment[11] = ItemInstance{Code: 1, Kind: 1, Effects: []ItemEffect{{Kind: 7, Operand: 10}}}
		w, err := NewStockedWorld(1, Bounds{Width: 8, Height: 8}, ModeCanonical, Terrain{}, []Entity{actor}, nil, Relations{}, nil, []Stock{{ID: 1, ItemInstances: []ItemInstance{{Code: 2, Kind: 1}}, EquippedItems: equipment}})
		if err != nil {
			t.Fatal(err)
		}
		if got := StepReported(w, []Command{cmd}).Damages; len(got) != 0 {
			t.Fatalf("command%v sent %+v for a lost maximum-health lift", cmd.Kind, got)
		}
	}
}

func TestDamageMessagesKeepNativeDirectHealthLossOfEquipment(t *testing.T) {
	actor := spMage(1, 1, 1, 30, 20, 20, 1<<1)
	actor.HP = 5
	var equipment [EquipSlots]ItemInstance
	equipment[11] = ItemInstance{Code: 1, Kind: 1, Effects: []ItemEffect{{Kind: 6, Operand: 10}}}
	w, err := NewStockedWorld(1, Bounds{Width: 8, Height: 8}, ModeCanonical, Terrain{}, []Entity{actor}, nil, Relations{}, nil, []Stock{{ID: 1, ItemInstances: []ItemInstance{{Code: 2, Kind: 1}}, EquippedItems: equipment}})
	if err != nil {
		t.Fatal(err)
	}
	want := []DamageEvent{{Target: 1, BeforeHP: 5, AfterHP: -5}}
	if got := StepReported(w, []Command{Unequip(1, 12)}).Damages; !slices.Equal(got, want) {
		t.Fatalf("direct health unequip=%+v want %+v", got, want)
	}
}

func TestDamageMessagesCoverDebugKillAndTerminalKill(t *testing.T) {
	for _, tc := range []struct {
		command Command
		after   int32
	}{{Kill(1), -1}, {TerminalKill(1), -10}} {
		w := cbWorld(t, 1, cbEnt(1, 1, 1))
		want := []DamageEvent{{Target: 1, BeforeHP: 100, AfterHP: tc.after}}
		if got := StepReported(w, []Command{tc.command}).Damages; !slices.Equal(got, want) {
			t.Fatalf("debug=%+v want %+v", got, want)
		}
	}
}

func TestDamageMessagesKeepPhysicalBeforeWeaponRider(t *testing.T) {
	rule := wpnRule(6, 6, 5)
	a := wpnCaster(1, 0, 0, 1, 30, 1, 0)
	a.MaxMana, a.Mana, a.DamageBase, a.DamageSpread, a.AlwaysHits = 0, 0, 40, 0, true
	w := spWorld(t, 1, []SpellRule{rule}, a, spEnt(2, 1, 0))
	want := []DamageEvent{{Target: 2, BeforeHP: 100, AfterHP: 60}, {Target: 2, BeforeHP: 60, AfterHP: 48}}
	if got := StepReported(w, []Command{Attack(1, 2)}).Damages; !slices.Equal(got, want) {
		t.Fatalf("messages=%+v want %+v", got, want)
	}
}

func TestDamageMessagesKeepRepeatedAreaReferences(t *testing.T) {
	big := Entity{ID: 1, X: 9, Y: 9, HP: 100, MaxHP: 100, TokenSize: 2}
	w := structureAreaWorld(t, nil, big)
	w.spells = []SpellRule{{ID: 9, Area: true, Damaging: true, DamageMin: 10, DamageMax: 10}}
	w.effects = []cellEffect{{Key: 0x0a0a, Spell: 9, Mode: areaModeRing, Remaining: 20, Phase: 2}}
	want := []DamageEvent{{Target: 1, BeforeHP: 100, AfterHP: 90}, {Target: 1, BeforeHP: 90, AfterHP: 80}}
	if got := StepReported(w, nil).Damages; !slices.Equal(got, want) {
		t.Fatalf("messages=%+v want %+v", got, want)
	}
}

func TestDamageMessagesIsolateRefusedSourceReplacement(t *testing.T) {
	w := sourceMutationWorld(t, PlainItem(0xe01))
	if !w.attachEffect(1, 1, SpellRule{ID: 13}, EffectHealth, 5, 100, EffectDuration) {
		t.Fatal("seed attachment")
	}
	w.damageObservation = &damageObservation{}
	w.applySpellDamage(0, SpellRule{DamageMin: 3, DamageMax: 3}, 0)
	w.BindSourceDerive(func(s SourceActor, _ int32, _ Rules) (SourceActor, error) {
		if s.Stats[8] == 49 {
			return s, fmt.Errorf("refused replacement")
		}
		return s, nil
	})
	before, hash := hlBytes(t, w), w.Hash()
	if w.attachEffect(1, 1, SpellRule{ID: 13}, EffectHealth, 2, 100, EffectDuration) {
		t.Fatal("replacement should refuse")
	}
	want := []DamageEvent{{Target: 1, BeforeHP: 55, AfterHP: 52}}
	if !slices.Equal(w.damageObservation.events, want) || !bytes.Equal(before, hlBytes(t, w)) || hash != w.Hash() {
		t.Fatalf("refused source leaked state/events: %+v", w.damageObservation.events)
	}
	w.BindSourceDerive(func(s SourceActor, _ int32, _ Rules) (SourceActor, error) { return s, nil })
	if !w.attachEffect(1, 1, SpellRule{ID: 13}, EffectHealth, 2, 100, EffectDuration) {
		t.Fatal("successful replacement")
	}
	want = append(want, DamageEvent{Target: 1, BeforeHP: 52, AfterHP: 47})
	if !slices.Equal(w.damageObservation.events, want) || w.entities[0].HP != 49 {
		t.Fatalf("committed replacement lost earlier or tentative events: %+v, HP=%d", w.damageObservation.events, w.entities[0].HP)
	}
	w.damageObservation = nil
}

func TestDamageMessagesCoverHealthApplicationsAndExcludeRestoration(t *testing.T) {
	w := cbWorld(t, 1, cbEnt(1, 1, 1))
	w.damageObservation = &damageObservation{}
	w.applyEffectDelta(0, EffectHealth, -7)
	w.applyEffectDelta(0, EffectHealth, 4)
	w.attachEffect(1, 1, SpellRule{ID: 13}, EffectHealth, 3, 1, EffectDuration)
	w.stepAttachedEffects()
	w.setUnitProperty(0, propertyHealth, 90)
	w.setUnitProperty(0, propertyHealth, 100)
	w.attached = []attachedEffect{{Target: 1, Spell: 8, Kind: EffectHealth, Mode: EffectContinuous, Magnitude: -2, Remaining: 8}}
	w.applyPoisonAttachment(0, 0)
	w.stepAttachedEffects()
	applyEquipmentItemState(&w.entities[0], ItemInstance{Effects: []ItemEffect{{Kind: 6, Operand: 3}}}, ItemInstance{}, nil, w.damageObservation)
	want := []DamageEvent{{Target: 1, BeforeHP: 100, AfterHP: 93}, {Target: 1, BeforeHP: 100, AfterHP: 97}, {Target: 1, BeforeHP: 97, AfterHP: 90}, {Target: 1, BeforeHP: 100, AfterHP: 98}, {Target: 1, BeforeHP: 98, AfterHP: 96}, {Target: 1, BeforeHP: 96, AfterHP: 93}}
	if !slices.Equal(w.damageObservation.events, want) {
		t.Fatalf("applications=%+v want %+v", w.damageObservation.events, want)
	}
	w.damageObservation = nil
}

func TestDamageObservationClearsEveryStepAndEarlyRefusal(t *testing.T) {
	w := cbWorld(t, 1, cbEnt(1, 1, 1))
	first := StepReported(w, []Command{Damage(1, 7), Damage(1, 3)})
	want := []DamageEvent{{Target: 1, BeforeHP: 100, AfterHP: 93}, {Target: 1, BeforeHP: 93, AfterHP: 90}}
	if !slices.Equal(first.Damages, want) || w.damageObservation != nil {
		t.Fatalf("messages or retained sink: %+v", first.Damages)
	}
	if len(StepReported(w, nil).Damages) != 0 || w.damageObservation != nil {
		t.Fatal("next tick replayed events")
	}
	w.entities[0].ActorLoad.Source.Class = 2
	if len(StepReported(w, nil).Damages) != 0 || w.damageObservation != nil {
		t.Fatal("refused step retained observation")
	}
}

func TestDamageMessagesCoverSessionScriptBeforeCommand(t *testing.T) {
	w := scriptWorld(t, nil, []Entity{{ID: 1, HP: 100, MaxHP: 100}})
	session := originalSessionFixture()
	session.HasClock, session.Clock = true, SessionClock{6, 584}
	if err := w.ImportOriginalSession(session); err != nil {
		t.Fatal(err)
	}
	w.script = mustScript(t, nil, []ScriptInstant{{Op: ScriptInstantProperty, Unit: 1, HasUnit: true, Args: [scriptParams]int32{propertyHealth, 70}}}, []ScriptTrigger{{Instants: acts(0), Once: true}})
	want := []DamageEvent{{Target: 1, BeforeHP: 100, AfterHP: 70}, {Target: 1, BeforeHP: 70, AfterHP: 40}}
	if got := StepReported(w, []Command{Damage(1, 30)}).Damages; !slices.Equal(got, want) {
		t.Fatalf("session script/command=%+v want %+v", got, want)
	}
}

func TestDamageMessagesKeepNonzeroPoisonRestoration(t *testing.T) {
	w := cbWorld(t, 1, cbEnt(1, 1, 1))
	w.entities[0].Protection[1] = 150
	w.attached = []attachedEffect{{Target: 1, Spell: 8, Kind: EffectHealth, Mode: EffectContinuous, Magnitude: -7, Remaining: 8}}
	want := []DamageEvent{{Target: 1, BeforeHP: 100, AfterHP: 103}}
	if got := StepReported(w, nil).Damages; !slices.Equal(got, want) {
		t.Fatalf("nonzero Token8=%+v want %+v", got, want)
	}
	w.entities[0].Protection[1] = 100
	w.attached[0].Remaining = 8
	if got := StepReported(w, nil).Damages; len(got) != 0 {
		t.Fatalf("zero Token8 sent %+v", got)
	}
}

func TestDamageMessagesSkipSourceMaximumHealthReduction(t *testing.T) {
	w := sourceMutationWorld(t, PlainItem(0xe01))
	item := ItemInstance{Code: 0x0701, WeightPresent: true, Weight: 2, SourceEquipment: SourceEquipment{Class: SourceArmor, DefinitionRow: 1, OwnKind: 7}, Effects: []ItemEffect{{Kind: 7, Operand: 10}}}
	w.equipment[0][6] = item
	w.entities[0].ActorLoad.OwnWeight = 2
	w.entities[0].ActorLoad.Source.Modifier[8] = 10
	if got := StepReported(w, []Command{Unequip(1, 7)}).Damages; len(got) != 0 {
		t.Fatalf("source unequip sent %+v for a lost maximum-health lift", got)
	}
}
