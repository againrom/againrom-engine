package game

import "testing"

func TestQuickSpellsScenarioVocabularyKeepsOlderVersionsUnchanged(t *testing.T) {
	spell, armed, slots := uint32(16), false, [4]uint32{16}
	for _, step := range []HeadlessStep{
		{Command: "pointer", Action: "hover", At: &HeadlessPoint{Ground: true}},
		{Command: "pointer", Action: "move", At: &HeadlessPoint{Spell: &spell}},
		{Command: "assert_state", State: &HeadlessStateAssertion{QuickSpells: &slots}},
		{Command: "assert_state", State: &HeadlessStateAssertion{CurrentSpell: &spell}},
		{Command: "assert_state", State: &HeadlessStateAssertion{SpellArmed: &armed}},
	} {
		for version := 1; version < 8; version++ {
			if err := (HeadlessScenario{Version: version, Steps: []HeadlessStep{step}}).Validate(); err == nil {
				t.Fatalf("version %d accepted new quick-spell vocabulary", version)
			}
		}
		if err := (HeadlessScenario{Version: 8, Steps: []HeadlessStep{step}}).Validate(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestQuickSpellsHeadlessPointAndAssertions(t *testing.T) {
	for _, n := range []uint32{1, 16, 65535} {
		if err := (&HeadlessPoint{Spell: &n}).validate(); err != nil {
			t.Fatal(err)
		}
		if err := (&HeadlessPoint{Spell: &n, Ground: true}).validate(); err == nil {
			t.Fatal("ambiguous point accepted")
		}
	}
	for _, n := range []uint32{0, 65536} {
		if err := (&HeadlessPoint{Spell: &n}).validate(); err == nil {
			t.Fatal("invalid spell ID accepted")
		}
	}
	ids, current, armed := [4]uint32{16, 1, 6, 19}, uint32(16), true
	want := HeadlessStateAssertion{QuickSpells: &ids, CurrentSpell: &current, SpellArmed: &armed}
	good := HeadlessState{QuickSpells: ids, CurrentSpell: current, SpellArmed: armed}
	if err := assertHeadlessState(good, want, nil); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []HeadlessState{{QuickSpells: ids, CurrentSpell: current}, {QuickSpells: ids, SpellArmed: armed}, {CurrentSpell: current, SpellArmed: armed}} {
		if err := assertHeadlessState(bad, want, nil); err == nil {
			t.Fatal("wrong quick spell state accepted")
		}
	}
}
