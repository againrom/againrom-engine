package ui

import (
	"slices"
	"testing"
	"time"
)

// kindCards is a host's offer that changes with the hero: the hero casts
// when he is a mage, and the companion casts beside a fighter hero alone.
// A female fighter is offered nothing.
func kindCards(hero int) []TestPlayCard {
	switch hero {
	case 0:
		return []TestPlayCard{{Hero: true}, {Name: "Reniesta", Key: "npc24", Caster: true}, {Name: "Brian", Key: "npc25"}}
	case 2, 3:
		return []TestPlayCard{{Hero: true, Caster: true}, {Name: "Danath", Key: "npc24"}, {Name: "Brian", Key: "npc25"}}
	}
	return nil
}

func partyNextKind(e *MapEditor) {
	r, _ := e.controlPromptButtonRect(testPartyKind)
	editorClickAt(e, r.Min.Add(r.Size().Div(2)), time.Now())
}

// A round of the hero kind button gives back every box the map maker ticked:
// All spells of a card that does not cast beside the kind shown stays ticked
// in the choice, takes no part in the play, and shows again when the kind
// that casts returns.
func TestThePartyChoiceSurvivesTheHeroKindButton(t *testing.T) {
	p := &partyEdits{mapPanelEdits: mapPanelEdits{facts: playable()}, byHero: kindCards}
	e := partyEditor(t, p, "en")
	_ = e.Key("testplay")
	partyClick(t, e, testPartyRowPlays, 1)
	partyClick(t, e, testPartyRowSpells, 1)
	partyNextKind(e)
	if e.TestPartyHero() != 2 {
		t.Fatalf("the press shows kind %d", e.TestPartyHero())
	}
	partyClick(t, e, testPartyRowSpells, 0)
	partyNextKind(e)
	partyNextKind(e)
	want := []TestPlayMember{{On: true, Spells: true}, {On: true, Spells: true}, {}}
	if e.TestPartyHero() != 0 || !slices.Equal(e.TestPartyChoice(), want) {
		t.Fatalf("after the round: kind %d, choice %+v", e.TestPartyHero(), e.TestPartyChoice())
	}
	if s, _ := e.promptSpec(); s.Grid.Cells[2*3+0].Ticked || !s.Grid.Cells[2*3+1].Ticked {
		t.Fatalf("the spells boxes show %+v", s.Grid.Cells[6:])
	}
	_ = e.Key("enter")
	if len(p.chosen) != 1 || !slices.Equal(p.chosen[0], []TestPlayMember{{On: true}, {On: true, Spells: true}, {}}) {
		t.Fatalf("Start played %+v", p.chosen)
	}
}

// The choice lives in the editor's profile: every box and the kind of hero
// are written as the map maker sets them, and a new editor that reads the
// file opens the window with them. A choice the file keeps that no longer
// fits falls back part by part: a kind no hero is gives a man fighter, a
// card no map offers is passed over, a spells tick on a card that does not
// cast is not shown or played, and with none of the cards chosen the hero
// plays, so Start takes a press.
func TestThePartyChoiceLivesInTheProfile(t *testing.T) {
	var file []byte
	save := func(b []byte) error { file = append([]byte(nil), b...); return nil }
	p := &partyEdits{mapPanelEdits: mapPanelEdits{facts: playable()}, byHero: kindCards}
	e := partyEditor(t, p, "en")
	e.SetPanelProfile(nil, nil, save)
	_ = e.Key("testplay")
	partyNextKind(e)
	partyClick(t, e, testPartyRowSpells, 0)
	partyClick(t, e, testPartyRowPlays, 2)
	partyClick(t, e, testPartyRowSkills, 2)
	want := e.TestPartyChoice()
	_ = e.Key("escape")
	if len(file) == 0 {
		t.Fatal("the choice was not written")
	}
	again := partyEditor(t, &partyEdits{mapPanelEdits: mapPanelEdits{facts: playable()}, byHero: kindCards}, "en")
	again.SetPanelProfile(file, nil, save)
	_ = again.Key("testplay")
	if again.TestPartyHero() != 2 || !slices.Equal(again.TestPartyChoice(), want) {
		t.Fatalf("a new editor opens kind %d with %+v, want kind 2 with %+v", again.TestPartyHero(), again.TestPartyChoice(), want)
	}
	stale := []byte(`{"testparty":{"hero":7,"cards":{"hero":{"on":false},"npc99":{"on":true},"npc25":{"on":false,"spells":true},"npc24":"broken"}}}`)
	q := &partyEdits{mapPanelEdits: mapPanelEdits{facts: playable()}, byHero: kindCards}
	old := partyEditor(t, q, "en")
	old.SetPanelProfile(stale, nil, save)
	_ = old.Key("testplay")
	if got := old.TestPartyChoice(); old.TestPartyHero() != 0 || !slices.Equal(got, []TestPlayMember{{On: true}, {}, {Spells: true}}) {
		t.Fatalf("a stale file opens kind %d with %+v", old.TestPartyHero(), got)
	}
	if s, _ := old.promptSpec(); s.Disabled[testPartyStart] || s.Grid.Cells[2*3+2].Ticked {
		t.Fatalf("Start off %v, spells boxes %+v", s.Disabled, s.Grid.Cells[6:])
	}
	_ = old.Key("enter")
	if len(q.chosen) != 1 || q.heroes[0] != 0 || !slices.Equal(q.chosen[0], []TestPlayMember{{On: true}, {}, {}}) {
		t.Fatalf("Start played kind %v with %+v", q.heroes, q.chosen)
	}
}

// A map whose test play offers other cards or other kinds of hero does not
// cost the choice: back on the first map the window opens with the kind and
// the boxes the map maker left there, and the cards both maps offer keep
// their boxes on the other map too.
func TestThePartyChoiceSurvivesAnotherMap(t *testing.T) {
	p := &partyEdits{mapPanelEdits: mapPanelEdits{facts: playable()}, cards: partyCards()}
	e := partyEditor(t, p, "en")
	_ = e.Key("testplay")
	partyClick(t, e, testPartyRowPlays, 4)
	partyClick(t, e, testPartyRowSkills, 4)
	partyClick(t, e, testPartyRowSkills, 0)
	want := e.TestPartyChoice()
	_ = e.Key("escape")
	// The other map plays for a mage alone and offers the hero and Brian.
	p.cards = nil
	p.byHero = func(hero int) []TestPlayCard {
		if hero < 2 {
			return nil
		}
		cards := partyCards()
		return []TestPlayCard{cards[0], cards[4]}
	}
	_ = e.Key("testplay")
	if got := e.TestPartyChoice(); !e.TestPartyOpen() || len(got) != 2 || got[0] != want[0] || got[1] != want[4] {
		t.Fatalf("the other map's window: kind %d, choice %+v", e.TestPartyHero(), got)
	}
	_ = e.Key("escape")
	p.cards, p.byHero = partyCards(), nil
	_ = e.Key("testplay")
	if e.TestPartyHero() != 0 || !slices.Equal(e.TestPartyChoice(), want) {
		t.Fatalf("back on the first map: kind %d, choice %+v, want %+v", e.TestPartyHero(), e.TestPartyChoice(), want)
	}
}
