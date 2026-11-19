package mapload

import (
	"slices"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/rules"
	"againrom/pkg/sim"
)

// The first game's book takes spells 1 to 28 and the second game's 1 to 29;
// a cell or mask bit past the game's last spell is dropped.
func TestBookSpellCountWalksEveryIDOfBothGames(t *testing.T) {
	if got := SpellGameOf(nil); got != rules.SpellGameFirst {
		t.Fatalf("no table game %v, want first", got)
	}
	var rows []sim.SpellRule
	for id := 1; id <= 31; id++ {
		rows = append(rows, sim.SpellRule{ID: uint16(id), MaxRange: 5, ManaCost: 10})
	}
	for _, c := range []struct {
		spells string
		top    int32
	}{{"", 28}, {"rom1", 28}, {"rom2", 29}} {
		tab := &Table{builtSpellRules: rows}
		if c.spells != "" {
			tab.Edition = &base.Edition{Spells: c.spells}
		}
		if got := sim.BookSpellCount(SpellGameOf(tab)); got != int(c.top) {
			t.Fatalf("%q book count %d, want %d", c.spells, got, c.top)
		}
		for id := int32(0); id <= 31; id++ {
			want := uint32(0)
			if id >= 1 && id <= c.top {
				want = uint32(1) << uint(id)
			}
			params := make([]int32, 64)
			params[48] = id
			cellKnown, placed, _ := unitSpellbook(params, tab, 0x46)
			if cellKnown != want {
				t.Fatalf("%q cell read id %d: known %#x, want %#x", c.spells, id, cellKnown, want)
			}
			if slot := placed.Slot(uint16(id)); want != 0 && *slot != (sim.BookSpell{Range: 5, ManaCost: 10}) {
				t.Fatalf("%q cell read id %d: book slot %+v, want the row's range and cost", c.spells, id, *slot)
			}
			b := &CreatureBooks{KnownCell: 40}
			params = make([]int32, 64)
			params[40] = int32(uint32(1) << uint(id))
			var known uint32
			var book sim.Spellbook
			var skill [6]int32
			b.add(0x46, 1, params, tab, &known, &book, &skill)
			if known != want {
				t.Fatalf("%q mask insert id %d: known %#x, want %#x", c.spells, id, known, want)
			}
			if want != 0 && *book.Slot(uint16(id)) != (sim.BookSpell{Range: 5, ManaCost: 10}) {
				t.Fatalf("%q mask insert id %d: book slot %+v, want the row's range and cost", c.spells, id, *book.Slot(uint16(id)))
			}
		}
	}
}

// A creature whose Units row mask holds Slow, as the Necromancer rows do,
// keeps spell 29 in its book with the row's range and cost and casts it.
func TestMaskBookCreatureCastsSlow(t *testing.T) {
	var rows []sim.SpellRule
	for id := 1; id <= 29; id++ {
		rows = append(rows, sim.SpellRule{ID: uint16(id), MaxRange: 5, ManaCost: 10, School: 4, TargetsUnit: true, SpellDuration: 5, EffectKind: sim.EffectSpeed, EffectMode: sim.EffectDuration})
	}
	sim.AssignSecondGameArms(rows)
	tab := &Table{builtSpellRules: rows, Edition: &base.Edition{Spells: "rom2"}}
	b := &CreatureBooks{KnownCell: 40}
	params := make([]int32, 64)
	params[40] = int32(uint32(1)<<26 | uint32(1)<<29)
	var known uint32
	var book sim.Spellbook
	var skill [6]int32
	b.add(0x63, 1, params, tab, &known, &book, &skill)
	mage := sim.Entity{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10, Mana: 20, MaxMana: 20, Book: book, KnownSpells: known}
	s, err := sim.NewROM2Script(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, nil, []sim.Entity{mage}, s, rows)
	if err != nil {
		t.Fatal(err)
	}
	if why := w.BookSpellRefusal(7, 7, 29); why != "" {
		t.Fatalf("Slow from the mask's book is refused: %s", why)
	}
	for range 40 {
		if len(w.ActiveEffects()) == 0 {
			sim.Step(w, []sim.Command{sim.Cast(7, 7, 29)})
		}
		sim.Step(w, nil)
	}
	if !slices.ContainsFunc(w.ActiveEffects(), func(e sim.ActiveEffect) bool { return e.Spell == 29 }) {
		t.Fatal("the cast of Slow left no effect")
	}
}
