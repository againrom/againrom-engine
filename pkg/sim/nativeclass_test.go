package sim

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func nativeClassWorld(t *testing.T, base, bonus int32) *World {
	w := trainingWorld(t, base, bonus)
	w.entities[0].Humanoid = true
	return w
}

func TestNativeClassRejectsCreaturePresence(t *testing.T) {
	w := nativeClassWorld(t, 10, 0)
	w.entities[0].Humanoid = false
	before := w.Hash()
	if w.SetNativeClass(1, NativeClass{Present: true, Fighter: true}) || w.Hash() != before {
		t.Fatal("creature adopted a Human class")
	}
	if !w.SetNativeClass(1, NativeClass{}) {
		t.Fatal("creature class absence refused")
	}
	w.entities[0].NativeClass = NativeClass{Present: true}
	if _, err := w.MarshalBinary(); err == nil {
		t.Fatal("creature class entered binary state")
	}
}

func TestNativeClassSelectsBonusAndAwardWithoutManaInference(t *testing.T) {
	for _, fighter := range []bool{false, true} {
		t.Run(map[bool]string{true: "fighter with mana", false: "mage without mana"}[fighter], func(t *testing.T) {
			w := nativeClassWorld(t, 10, 10)
			w.entities[0].MaxMana = 0
			if fighter {
				w.entities[0].MaxMana = 90
			}
			class := NativeClass{Present: true, Fighter: fighter}
			w.SetNativeClass(1, class)
			w.equipment[0][11].Effects = []ItemEffect{{Kind: 29, Operand: 10}, {Kind: 35, Operand: 70}}
			bonus := int32(70)
			if fighter {
				bonus = 10
			}
			w.entities[0].Skill[3] = 10 + bonus
			w.hasSessionClock = true
			if w.wornSkillBonus(0, 3) != bonus || w.NativeTrainingNeedsProducer(1) {
				t.Fatal("known class producer refused", w.nativeSkillBonuses(0))
			}
			raw, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var cold World
			if err := cold.UnmarshalBinary(raw); err != nil || cold.Hash() != w.Hash() {
				t.Fatal("class wire", err)
			}
			named := int32(3)
			if fighter {
				named = 0
			}
			for _, next := range []*World{w, &cold} {
				if !next.awardSkill(0, named, 1_000_000, 1) || next.entities[0].NativeTraining.Levels[3] != 11 || next.entities[0].Skill[3] != 11+bonus {
					t.Fatal("eligible class award", next.entities[0])
				}
				if next.awardSkill(0, 3-named, 1_000_000, 1) {
					t.Fatal("opposite award arm admitted")
				}
			}
			if cold.Hash() != w.Hash() {
				t.Fatal("post-load award differs")
			}
		})
	}
}

func TestNativeClassLegacyDefaultAndOptionalCombinedWire(t *testing.T) {
	w := nativeClassWorld(t, 40, -100)
	w.DeclareStructures([]Structure{{ID: 1, Col: 6, Row: 6, Width: 1, Height: 1, Attach: 1, Blocking: 2}})
	old, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(old); err != nil || cold.entities[0].NativeClass != (NativeClass{}) || cold.Hash() != w.Hash() {
		t.Fatal("legacy default", err)
	}
	if skillMage(cold.entities[0]) {
		t.Fatal("legacy mana fallback changed")
	}
	cold.entities[0].MaxMana = 1
	if !skillMage(cold.entities[0]) {
		t.Fatal("legacy positive mana fallback changed")
	}
	for _, rom2 := range []bool{false, true} {
		next := *w
		if rom2 {
			next.script, _ = NewROM2Script(nil, nil, nil)
			next.rom2 = &rom2ScriptState{}
		}
		base, _ := next.MarshalBinary()
		next.entities = append([]Entity(nil), w.entities...)
		if !next.SetNativeClass(1, NativeClass{Present: true, Fighter: true}) {
			t.Fatal("set known class")
		}
		raw, err := next.MarshalBinary()
		if err != nil || raw[0] != 107 || string(raw[len(raw)-4:]) != "CLS1" || CheckSaveForm(raw) != nil || !HasStructureBlockingForm(raw) {
			t.Fatal("class footer/peeler", err)
		}
		start := len(raw) - 9 - int(binary.LittleEndian.Uint32(raw[len(raw)-9:]))
		peeled := bytes.Clone(raw[:start])
		peeled[0] = raw[len(raw)-5]
		if !bytes.Equal(peeled, base) {
			t.Fatal("optional class changed predecessor")
		}
		if err := cold.UnmarshalBinary(raw); err != nil || cold.Hash() != next.Hash() || cold.entities[0].NativeClass != next.entities[0].NativeClass || cold.script.Dialect() != next.script.Dialect() {
			t.Fatal("combined cold class", err)
		}
		before := cold.Hash()
		for _, edit := range []func([]byte){
			func(b []byte) { b[len(b)-1] = 'X' },
			func(b []byte) { b[len(b)-5] = 107 },
			func(b []byte) { binary.LittleEndian.PutUint32(b[len(b)-9:], 0xffffffff) },
			func(b []byte) { binary.LittleEndian.PutUint32(b[start:], 65536) },
			func(b []byte) { binary.LittleEndian.PutUint32(b[start+4:], 999) },
			func(b []byte) { b[start+8] = 2 },
		} {
			bad := bytes.Clone(raw)
			edit(bad)
			if cold.UnmarshalBinary(bad) == nil || CheckSaveForm(bad) == nil || cold.Hash() != before {
				t.Fatal("bad class footer accepted or changed receiver")
			}
		}
	}
	if w.SetNativeClass(999, NativeClass{Present: true}) || w.SetNativeClass(1, NativeClass{Fighter: true}) {
		t.Fatal("invalid class setter admitted")
	}
}

func TestNativeClassDelayedCreditSurvivesColdLoad(t *testing.T) {
	for _, fighter := range []bool{false, true} {
		w := nativeClassWorld(t, 10, 10)
		w.entities[0].MaxMana = 0
		kind := uint8(35)
		if fighter {
			w.entities[0].MaxMana, kind = 90, 29
		}
		w.SetNativeClass(1, NativeClass{Present: true, Fighter: fighter})
		w.equipment[0][11].Effects[0].Kind = kind
		w.spells = []SpellRule{{ID: 3, School: 3}}
		w.hasSessionClock = true
		w.resolveDamageAttribution(0, 1, 3)
		want := int8(3)
		if fighter {
			want = 0
		}
		if e := w.entities[1]; !e.HasKillCredit || e.KillCreditSource != 1 || e.KillCreditSpell != want {
			t.Fatal("class-selected retained damage credit", e)
		}
		w.entities[1].HP, w.entities[1].XPValue = -1, 1_000_000
		w.entities[1].Decay = 1
		raw, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var cold World
		if err := cold.UnmarshalBinary(raw); err != nil {
			t.Fatal(err)
		}
		for _, next := range []*World{w, &cold} {
			next.processNewKillCredits(map[EntityID]int32{2: 0})
			if e := next.entities[0]; e.NativeTraining.Levels[3] != 11 || e.Skill[3] != 21 {
				t.Fatal("delayed class-selected award", e.NativeTraining, e.Skill)
			}
		}
		if w.Hash() != cold.Hash() {
			t.Fatal("delayed award diverged after cold LOAD")
		}
	}
}

func TestNativeClassHashAndSourcePrecedence(t *testing.T) {
	w := nativeClassWorld(t, 10, 0)
	before := w.Hash()
	w.SetNativeClass(1, NativeClass{Present: true, Fighter: true})
	if w.Hash() == before {
		t.Fatal("known class omitted from hash")
	}
	for _, fighter := range []bool{false, true} {
		e := w.entities[0]
		e.ActorLoad.Source = SourceActor{Class: 2, Fighter: fighter}
		if skillMage(e) == fighter {
			t.Fatal("source class did not select its own arm")
		}
	}
	w.entities[0].ActorLoad.Source = SourceActor{Class: 2, Fighter: true}
	if w.SetNativeClass(1, NativeClass{Present: true}) {
		t.Fatal("native setter overwrote source class")
	}
	w.publishSource(0, w.entities[0].ActorLoad.Source)
	if w.entities[0].NativeClass != (NativeClass{}) {
		t.Fatal("source publication retained native class")
	}
}

func TestNativeClassWireRejectsDuplicateAndUnorderedActors(t *testing.T) {
	w := nativeClassWorld(t, 10, 0)
	w.entities[1].Humanoid = true
	w.entities[1].NativeClass = NativeClass{Present: true}
	w.SetNativeClass(1, NativeClass{Present: true, Fighter: true})
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(raw); err != nil || cold.Hash() != w.Hash() {
		t.Fatal("valid class population", err)
	}
	start := len(raw) - 9 - int(binary.LittleEndian.Uint32(raw[len(raw)-9:]))
	for _, second := range []uint32{0, 1, 3} {
		bad := bytes.Clone(raw)
		binary.LittleEndian.PutUint32(bad[start+9:], second)
		before := w.Hash()
		if w.UnmarshalBinary(bad) == nil || w.Hash() != before || CheckSaveForm(bad) == nil {
			t.Fatal("invalid class population admitted", second)
		}
	}
}

func TestOriginalBookRangeUsesOrdinarySignedWords(t *testing.T) {
	for _, tc := range []struct {
		name        string
		school      uint8
		level, mind int32
		base, want  uint8
	}{
		{"projected named level", 1, 110, 10, 10, 12},
		{"signed low sum", 1, -10, -10, 10, 10},
		{"signed wide Mind word", 1, 100, 65546, 10, 12},
		{"general has no named projection", 0, 200, -100, 10, 12},
		{"power ceiling", 1, 110, 40, 10, 13},
		{"zero base", 1, 100, 100, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := Entity{KnownSpells: 1 << 2, Mind: tc.mind, Book: Spellbook{State: BookPresent}}
			e.Skill[tc.school] = tc.level
			e.Book.Slots[1] = BookSpell{Range: 99, Defensive: 3, ManaCost: 54321}
			rule := SpellRule{ID: 2, School: tc.school, MaxRange: tc.base}
			RefreshOriginalBook(&e, []SpellRule{rule})
			if e.Book.Slots[1] != (BookSpell{Range: tc.want, Defensive: 3, ManaCost: 54321}) {
				t.Fatal("ordinary signed cache", e.Book.Slots[1])
			}
		})
	}
}
