package sim

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

func requireSpellbookLegacyDigest(t *testing.T, w *World, want uint64) {
	t.Helper()
	if got := fnv1a(strippedWorldOfSpellbook(mustMarshal(t, w))); got != want {
		t.Fatalf("form70 digest changed: %016x want %016x", got, want)
	}
}

func TestSourceBookCanonicalPresenceAndAtomicDecode(t *testing.T) {
	var hashes []uint64
	for _, state := range []BookState{BookLegacy, BookAbsent, BookPresent} {
		e := spEnt(1, 2, 2)
		e.Book.State = state
		w := spWorld(t, 1101, nil, e)
		form := mustMarshal(t, w)
		var back World
		if err := back.UnmarshalBinary(form); err != nil {
			t.Fatal(err)
		}
		if back.entities[0].Book != e.Book || back.Hash() != w.Hash() {
			t.Fatal("presence lost")
		}
		for _, h := range hashes {
			if h == w.Hash() {
				t.Fatal("absent, present-empty and legacy alias")
			}
		}
		hashes = append(hashes, w.Hash())
	}
	w := spWorld(t, 1101, nil, sourceMage1101(1, 1, BookSpell{9, 2, 32768}))
	before := mustMarshal(t, w)
	base := 34 + 3*int(binary.LittleEndian.Uint32(before[30:34])) + 341
	for _, at := range []int{base, base + 5} {
		bad := bytes.Clone(before)
		bad[at] = 3 // invalid tag or parameters on the unlearned second slot
		if err := w.UnmarshalBinary(bad); err == nil {
			t.Fatal("malformed instance accepted")
		}
		if !bytes.Equal(before, mustMarshal(t, w)) {
			t.Fatal("failed decode mutated receiver")
		}
	}
}

func TestSourceBookSavedRetryAndWindupUseInstance(t *testing.T) {
	rule := SpellRule{ID: 1, ManaCost: 90, School: 1, MaxRange: 1, DamageMin: 2, DamageMax: 2, TargetsUnit: true, Damaging: true}
	caster := sourceMage1101(1, 1, BookSpell{3, 2, 7})
	caster.Mana = 0
	w := spWorld(t, 1101, []SpellRule{rule}, caster, spEnt(2, 5, 2))
	if !w.beginBookSpell(0, 2, 1) || len(w.bookCasts) != 1 || w.bookCasts[0].Phase != bookPending {
		t.Fatal("no retained retry")
	}
	var back World
	if err := back.UnmarshalBinary(mustMarshal(t, w)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		Step(w, nil)
		Step(&back, nil)
		if w.Hash() != back.Hash() {
			t.Fatal("pending diverged")
		}
	}
	w.entities[0].Mana, back.entities[0].Mana = 7, 7
	Step(w, nil)
	Step(&back, nil)
	if w.bookCasts[0].Phase != bookCharging || w.Hash() != back.Hash() {
		t.Fatal("retry used table mana/range")
	}
	if err := back.UnmarshalBinary(mustMarshal(t, w)); err != nil {
		t.Fatal(err)
	}
	released := false
	for i := 0; i < 64; i++ {
		a, b := StepObserved(w, nil), StepObserved(&back, nil)
		if !reflect.DeepEqual(a, b) || w.Hash() != back.Hash() {
			t.Fatal("wind-up diverged")
		}
		if len(a) > 0 {
			released = true
			break
		}
	}
	if !released || w.entities[0].Mana != 0 || w.entities[1].HP != 96 || w.entities[0].Book.Slots[0] != (BookSpell{2, 2, 7}) {
		t.Fatal("release ignored saved source")
	}
}

func TestSourceBookDoesNotOverrideScrollOrWeaponSources(t *testing.T) {
	w := scrollWorld1090(t, 3, 1)
	w.entities[0].KnownSpells = 1 << 1
	w.entities[0].Book = Spellbook{State: BookPresent}
	w.entities[0].Book.Slots[0] = BookSpell{0, 2, 32767}
	book := w.entities[0].Book
	Step(w, []Command{{Kind: KindUseScroll, Entity: 1, X: 2}})
	for i := 0; i < 60 && len(w.ScrollCasts()) != 0; i++ {
		Step(w, nil)
	}
	if w.entities[1].HP >= 100 || w.entities[0].Book != book || w.entities[0].Mana != 0 {
		t.Fatal("book replaced scroll source")
	}
	e := w.entities[0]
	e.MaxMana = 100
	e.WeaponSpell, e.WeaponSpellLevel, e.WeaponSpellSource = 1, 60, WeaponSpellInnate
	raw, ok := WeaponSpellCharacteristicsFor(Rules{}, e, w.Spells())
	if !ok || raw.MaxRange != 3 {
		t.Fatalf("book replaced weapon source: %+v %v", raw, ok)
	}
	resolved, _ := BookRuleFor(e, w.Spells()[0])
	if _, err := normaliseSpells([]SpellRule{resolved}); err == nil {
		t.Fatal("transient instance entered canonical table")
	}
}
