package ui

import (
	"reflect"
	"testing"
)

func knowledgeEntity(level int) MapEntity {
	return MapEntity{ID: 4, HP: 10, MaxHP: 20, Mana: 3, MaxMana: 7, Knowledge: level, KnowledgeKnown: true,
		Combat: UnitCombat{Known: true, DamageBase: 4, DamageSpread: 2, ToHit: 5, Defence: 6, Absorption: 7},
		Char:   UnitCharacter{Known: true, Body: 1, Reaction: 2, Mind: 3, Spirit: 4, Sight: 8},
		Speed:  9}
}

// cardGroups lists the first level each card group is drawn at.
var cardGroups = []struct {
	field PanelField
	level int
}{
	{PanelFieldHealth, 1}, {PanelFieldMana, 1},
	{PanelFieldSight, 2}, {PanelFieldMoveSpeed, 2},
	{PanelFieldDamage, 3}, {PanelFieldToHit, 3},
	{PanelFieldDefence, 4}, {PanelFieldAbsorption, 4},
	{PanelFieldBody, 5}, {PanelFieldReaction, 5}, {PanelFieldMind, 5}, {PanelFieldSpirit, 5},
	{PanelFieldProtFire, 6}, {PanelFieldResistHeading, 6},
	{PanelFieldSkillBlade, 7}, {PanelFieldSkillsHeading, 7},
}

func TestCardDrawsEachGroupFromItsKnowledgeLevel(t *testing.T) {
	v := &Viewer{}
	for level := 0; level <= 7; level++ {
		s := v.panelSubjectFromPresent([]MapEntity{knowledgeEntity(level)})
		if s.DetailLevel != level || !s.DetailSet {
			t.Fatalf("level %d card subject level %d set %v", level, s.DetailLevel, s.DetailSet)
		}
		for _, g := range cardGroups {
			if _, got := panelText(s, g.field); got != (level >= g.level) {
				t.Errorf("level %d field %d visible=%v, want %v", level, g.field, got, level >= g.level)
			}
		}
	}
}

func TestCardWithoutAStatedLevelDrawsEveryGroup(t *testing.T) {
	e := knowledgeEntity(0)
	e.KnowledgeKnown = false
	s := (&Viewer{}).panelSubjectFromPresent([]MapEntity{e})
	for _, g := range cardGroups {
		if _, ok := panelText(s, g.field); !ok {
			t.Errorf("unstated level hid field %d", g.field)
		}
	}
}

func TestRevealShowsEveryCardGroupAndRestoresTheLevelExactly(t *testing.T) {
	v := &Viewer{}
	e := knowledgeEntity(2)
	before := e
	low := v.panelSubjectFromPresent([]MapEntity{e})
	v.SetFogReveal(true)
	full := v.panelSubjectFromPresent([]MapEntity{e})
	if full.DetailLevel != 7 {
		t.Fatalf("revealed card level %d, want 7", full.DetailLevel)
	}
	for _, g := range cardGroups {
		if _, ok := panelText(full, g.field); !ok {
			t.Errorf("reveal left field %d hidden", g.field)
		}
	}
	v.SetFogReveal(false)
	if back := v.panelSubjectFromPresent([]MapEntity{e}); !reflect.DeepEqual(back, low) {
		t.Fatalf("subject after reveal off differs from the subject before: %+v vs %+v", back, low)
	}
	if !reflect.DeepEqual(e, before) {
		t.Fatal("reveal wrote the entity")
	}
}

func TestCardItemsAreDrawnAtExactlyLevelSeven(t *testing.T) {
	s := PanelSubject{DetailSet: true, OriginalPanel: OriginalPanelActor{Known: true, Byte14A: 1, XPValue: 5},
		Char: UnitCharacter{Band: CharacterBandPerson}, KnownSpells: 1}
	for level := 0; level <= 15; level++ {
		s.DetailLevel = level
		for _, f := range []PanelField{PanelFieldArmorPiercing, PanelFieldSpellcaster} {
			if _, got := panelText(s, f); got != (level == 7) {
				t.Errorf("level %d field %d visible=%v", level, f, got)
			}
		}
	}
}
