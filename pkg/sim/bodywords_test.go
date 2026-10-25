package sim

import (
	"bytes"
	"testing"
)

// A body's loaded words are entity state: a living actor and an illegal
// health remainder are refused before any write, and the words survive the
// byte form.
func TestBodyWordsSetABodyAndSurviveBinary(t *testing.T) {
	w := mustWorld(t, 3, Bounds{Width: 8, Height: 8}, []Entity{
		{ID: 1, X: 1, Y: 1, HP: 20, MaxHP: 20},
		{ID: 2, X: 2, Y: 1, HP: -57, MaxHP: 20, Decay: DecayBones},
	})
	before := w.Hash()
	words := BodyReferenceRecord{ID: 2, Words: &BodyWords{AttackCharge: 8, HealthHundredths: 60, CreditSpell: 1}}
	for _, refused := range [][]BodyReferenceRecord{
		{words, {ID: 1, Words: &BodyWords{AttackCharge: 8}}},
		{{ID: 2, Words: &BodyWords{HealthHundredths: 100}}},
		{{ID: 9, Words: &BodyWords{}}},
	} {
		if err := w.RestoreBodyReferences(refused); err == nil || w.Hash() != before {
			t.Fatalf("batch %+v was admitted", refused)
		}
	}
	if err := w.RestoreBodyReferences([]BodyReferenceRecord{words}); err != nil {
		t.Fatal(err)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	again, err := cold.MarshalBinary()
	e := cold.entities[1]
	if err != nil || !bytes.Equal(again, form) || e.AttackCharge != 8 || e.HealthHundredths != 60 || e.KillCreditSpell != 1 {
		t.Fatalf("body words after the byte form: %d %d %d %v", e.AttackCharge, e.HealthHundredths, e.KillCreditSpell, err)
	}
}
