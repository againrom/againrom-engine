package game

import (
	"strings"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// spellcasterCaptionShown reports whether a card states the SPELLCASTER
// caption, by the statement the compact card draws.
func spellcasterCaptionShown(s ui.PanelSubject) bool {
	for _, row := range ui.PanelStatement(ui.CompactPanelLayout(nil), s) {
		if strings.Contains(row, "SPELLCASTER") {
			return true
		}
	}
	return false
}

// spellcasterMember is a generated person off no table: a mana maximum,
// trained skill experience whose one percent is nonzero, and a spell mask.
func spellcasterMember(maxMana int32, spells uint32) mapload.PartyMember {
	p := mapload.PartyMember{Name: "NPC03_3", Class: 3, KnownSpells: spells,
		Saved: &mapload.Saved{HP: 40, MaxHP: 40, Mana: maxMana, MaxMana: maxMana},
		Carry: &mapload.Carry{}}
	p.Carry.SkillXP[1] = 52000
	return p
}

// The first game's inn card states SPELLCASTER for a mercenary whose spell
// mask is nonzero and whose mana maximum is positive, and for no other
// mercenary; a party hero never states it (TEXT-135, TAVERN-035, TAVERN-036).
// The club fighter's card is the loss control: one percent of his trained
// skill experience is nonzero, which the earlier value captioned.
func TestFirstGameInnCardStatesSpellcasterForAMageOnly(t *testing.T) {
	for _, c := range []struct {
		what  string
		p     mapload.PartyMember
		hero  bool
		shown bool
	}{
		{"club fighter, no spell", spellcasterMember(0, 0), false, false},
		{"fighter with mana, no spell", spellcasterMember(30, 0), false, false},
		{"spell mask without a mana maximum", spellcasterMember(0, 1<<3), false, false},
		{"mercenary mage", spellcasterMember(120, 1<<3|1<<10), false, true},
		{"party hero mage", spellcasterMember(120, 1<<3), true, false},
	} {
		c.p.PlayerCharacter = c.hero
		s := partyPanelSubject(c.p, nil)
		if got := spellcasterCaptionShown(s); got != c.shown {
			t.Errorf("%s: SPELLCASTER %t, want %t (actor value %d)", c.what, got, c.shown, s.OriginalPanel.XPValue)
		}
	}
}

// The second game's card keeps its rule: the first game's experience value,
// held at zero without a mana pool, whatever the spell mask (R2-ENGINE-580,
// DIV-4086).
func TestSecondGameCardSpellcasterKeepsItsRule(t *testing.T) {
	second := base.GameROM2.Edition()
	table := &mapload.Table{Edition: &second}
	for _, c := range []struct {
		what  string
		p     mapload.PartyMember
		shown bool
	}{
		{"fighter without mana", spellcasterMember(0, 0), false},
		{"mage without a spell", spellcasterMember(120, 0), true},
	} {
		s := partyPanelSubject(c.p, table)
		if got := spellcasterCaptionShown(s); got != c.shown {
			t.Errorf("%s: SPELLCASTER %t, want %t (actor value %d)", c.what, got, c.shown, s.OriginalPanel.XPValue)
		}
	}
}

// On the first game's mission card a person's actor value is his spell mask
// behind a positive mana maximum, and a creature's stays its Units-row value,
// which the card's creature rule reads with its book (DIV-1809). The person
// fighter's trained skill experience gives a nonzero earlier value, and the
// plain creature's value is nonzero without a spell.
func TestFirstGameMissionCardSpellcasterValueByActorKind(t *testing.T) {
	fighterXP, mageXP := [data.SkillSlots]int32{}, [data.SkillSlots]int32{}
	fighterXP[1], mageXP[3] = 52000, 42000
	ents := []sim.Entity{
		{ID: 1, X: 1, Y: 1, TypeID: 3, HP: 40, MaxHP: 40, MaxMana: 20, SkillXP: fighterXP},
		{ID: 2, X: 3, Y: 1, TypeID: 4, HP: 40, MaxHP: 40, Mana: 90, MaxMana: 90, KnownSpells: 1<<3 | 1<<10, SkillXP: mageXP},
		{ID: 3, X: 5, Y: 1, TypeID: 35, HP: 40, MaxHP: 40, XPValue: 1079},
		{ID: 4, X: 1, Y: 3, TypeID: 36, HP: 40, MaxHP: 40, Mana: 60, MaxMana: 60, XPValue: 2300, KnownSpells: 1 << 9},
	}
	w, err := sim.NewWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil, ents)
	if err != nil {
		t.Fatal(err)
	}
	m := &alm.Map{Width: 8, Height: 8, Tiles: make([]uint16, 64), Altitudes: make([]uint8, 64)}
	classes := map[int32]*terrain.UnitClass{}
	for _, e := range ents {
		classes[e.TypeID] = worldFixtureArt(16, 16, 8, 16, 3, 3, 1)
	}
	mw := newMapWorld(w, nil, &terrain.UnitSet{Classes: classes}, worldFixtureViewer(t, m))
	want := map[uint32]int32{1: 0, 2: 1<<3 | 1<<10, 3: 1079, 4: 2300}
	seen := 0
	for _, d := range mw.entityDraws() {
		if v, ok := want[d.ID]; ok {
			seen++
			if d.OriginalPanel.XPValue != v {
				t.Errorf("entity %d: actor value %d, want %d", d.ID, d.OriginalPanel.XPValue, v)
			}
		}
	}
	if seen != len(want) {
		t.Fatalf("%d of %d entities drawn", seen, len(want))
	}
}
