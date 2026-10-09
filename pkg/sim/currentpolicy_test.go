package sim

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestCurrentSpellPolicyKeepsAdmissionAndOrdinaryBookAuthority(t *testing.T) {
	for _, delivery := range []int32{0, 1, 2} {
		native, cold := deliveryTestWorld(t, 1), deliveryTestWorld(t, 1)
		native.spells[0].Delivery, native.spells[0].EffectSpeed = delivery, 64
		policy := native.CurrentPolicy()
		policy.KeepSpellDeviations(cold.Spells())
		if len(policy.SpellPolicies) != 1 {
			t.Fatal("missing current delivery deviation", policy.SpellPolicies)
		}
		raw, err := json.Marshal(policy.SpellPolicies)
		if err != nil {
			t.Fatal(err)
		}
		for _, field := range []string{`"ManaCost":`, `"MaxRange":`, `"DamageMin":`} {
			if strings.Contains(string(raw), field) {
				t.Fatal("rule value duplicated by current policy", field)
			}
		}
		for _, w := range []*World{native, cold} {
			w.entities[0].Book = Spellbook{State: BookPresent}
			w.entities[0].Book.Slots[0] = BookSpell{Range: 12, ManaCost: 7}
		}
		if err := cold.RestoreCurrentContinuation(&policy, nil, cold.Actions(), nil); err != nil {
			t.Fatal(err)
		}
		rule, ok := cold.bookSpell(cold.entities[0], 1)
		if !ok || rule.Delivery != delivery || rule.EffectSpeed != 64 || rule.ManaCost != 7 || rule.MaxRange != 12 {
			t.Fatal("ordinary book or current timing lost", rule)
		}
		Step(native, []Command{spCast(1, 2, 1)})
		Step(cold, []Command{spCast(1, 2, 1)})
		if (native.entities[0].Mana == 93) != (delivery != 0) {
			t.Fatal("admission fixture did not exercise delivery policy")
		}
		for tick := 0; tick < 80; tick++ {
			if native.Hash() != cold.Hash() {
				t.Fatalf("delivery %d diverged at tick %d", delivery, tick)
			}
			Step(native, nil)
			Step(cold, nil)
		}
		if cold.entities[1].HP >= 1000 || cold.entities[0].Mana != 93 {
			t.Fatal("release did not apply exactly once", cold.entities[1].HP, cold.entities[0].Mana)
		}
	}
	w := deliveryTestWorld(t, 1)
	p := w.CurrentPolicy()
	p.KeepSpellDeviations(w.Spells())
	if len(p.SpellPolicies) != 0 {
		t.Fatal("unchanged definition copied into policy")
	}
	p.SpellPolicies = []CurrentSpellPolicy{{ID: 1}, {ID: 1, Delivery: 2, EffectSpeed: 64}}
	before := w.Hash()
	if err := w.RestoreCurrentContinuation(&p, nil, w.Actions(), nil); err == nil || w.Hash() != before {
		t.Fatal("invalid policy partially changed spell table", err)
	}
}

func TestCurrentEffectCarrierAbsencePreservesInactiveTree(t *testing.T) {
	w := transportGraphWorld(t, 1)
	tree := w.SavedSpellEffects()
	p := w.CurrentPolicy()
	no := false
	p.SpellGraphCarrier, p.EffectDriverCarrier = &no, &no
	if err := w.RestoreCurrentContinuation(&p, nil, w.Actions(), nil); err != nil {
		t.Fatal(err)
	}
	if w.savedSpellGraph != nil || w.savedWorldEffects != nil || !reflect.DeepEqual(tree, w.SavedSpellEffects()) {
		t.Fatal("carrier absence erased ordinary carried effects")
	}
	for range 3 {
		w.stepWorldSpellEffects(nil)
	}
	if w.entities[1].HP != 1000 || !reflect.DeepEqual(tree, w.SavedSpellEffects()) {
		t.Fatal("inactive carried transport acquired an execution driver")
	}
	requireSpellGraphBinary(t, w)
}

func TestCurrentSpellDurationDeviationDefaultsAndZero(t *testing.T) {
	installed := spWorld(t, 7, []SpellRule{{ID: 8, EffectDuration: 128}}, spEnt(1, 2, 2))
	predecessor := spWorld(t, 7, []SpellRule{{ID: 8, EffectDuration: 8}}, spEnt(1, 2, 2))
	policy := predecessor.CurrentPolicy()
	policy.KeepSpellDeviations(installed.Spells())
	if len(policy.SpellPolicies) != 1 || policy.SpellPolicies[0].EffectDuration == nil || *policy.SpellPolicies[0].EffectDuration != 8 {
		t.Fatal("duration-only deviation was discarded", policy.SpellPolicies)
	}
	raw, err := json.Marshal(policy)
	if err != nil || !strings.Contains(string(raw), `"EffectDuration":8`) {
		t.Fatal("duration deviation was not written", string(raw), err)
	}
	var decoded CurrentWorldPolicy
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if err := installed.RestoreCurrentSpellDurations(decoded.SpellPolicies); err != nil {
		t.Fatal(err)
	}
	if err := installed.RestoreCurrentSpellDurations(decoded.SpellPolicies); err != nil {
		t.Fatal("early duration restore is not idempotent", err)
	}
	if err := installed.RestoreCurrentContinuation(&decoded, nil, installed.Actions(), nil); err != nil || installed.Hash() != predecessor.Hash() {
		t.Fatal("late continuation lost duration deviation", err)
	}
	legacy := spWorld(t, 7, []SpellRule{{ID: 8, EffectDuration: 128}}, spEnt(1, 2, 2))
	old := legacy.CurrentPolicy()
	old.SpellPolicies = []CurrentSpellPolicy{{ID: 8}}
	if err := legacy.RestoreCurrentSpellDurations(old.SpellPolicies); err != nil {
		t.Fatal(err)
	}
	if err := legacy.RestoreCurrentContinuation(&old, nil, legacy.Actions(), nil); err != nil {
		t.Fatal(err)
	}
	if rule, _ := legacy.Spell(8); rule.EffectDuration != 128 {
		t.Fatal("old SAV without duration deviation lost installed default", rule.EffectDuration)
	}
	zero := uint16(0)
	old.SpellPolicies[0].EffectDuration = &zero
	if err := legacy.RestoreCurrentSpellDurations(old.SpellPolicies); err != nil {
		t.Fatal(err)
	}
	if rule, _ := legacy.Spell(8); rule.EffectDuration != 0 {
		t.Fatal("explicit zero duration was treated as absence", rule.EffectDuration)
	}
	bad := []CurrentSpellPolicy{{ID: 8, EffectDuration: &zero}, {ID: 8, EffectDuration: &zero}}
	before := predecessor.Hash()
	if err := predecessor.RestoreCurrentSpellDurations(bad); err == nil || predecessor.Hash() != before {
		t.Fatal("invalid early policy partially changed World", err)
	}
}
