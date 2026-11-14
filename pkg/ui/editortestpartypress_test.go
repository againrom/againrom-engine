package ui

import (
	"fmt"
	"testing"
	"time"
)

// herdCards is an offer whose set, order and name widths change with the hero
// kind, as the game's companions do.
func herdCards(hero int) []TestPlayCard {
	hc := TestPlayCard{Hero: true, Caster: hero >= 2}
	rn := TestPlayCard{Name: "Reniesta", Key: "npc22", Caster: true}
	nj := TestPlayCard{Name: "Nayra the Long Named", Key: "npc23", Caster: true}
	dn := TestPlayCard{Name: "Danath", Key: "npc24"}
	br := TestPlayCard{Name: "Brian", Key: "npc25"}
	rg := TestPlayCard{Name: "Rood Glaen", Key: "npc26", Caster: true}
	switch hero {
	case 0:
		return []TestPlayCard{hc, {Name: "Ferghard", Key: "npc21"}, dn, rn, br, rg}
	case 1:
		return []TestPlayCard{hc, {Name: "Ferghard", Key: "npc21"}, dn, rn, br, rg}
	case 2:
		return []TestPlayCard{hc, rn, nj, dn, br, rg}
	}
	return []TestPlayCard{hc, rn, nj, br, dn}
}

// Every box of the window, at every hero kind, takes a press at the middle of
// the rectangle that is drawn, and the press switches that character's box
// and no other; what the choice holds stays with the character.
func TestThePartyWindowPressSwitchesTheBoxDrawn(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, width := range []int{1024, 640} {
			t.Run(fmt.Sprintf("%s%d", lang, width), func(t *testing.T) {
				p := &partyEdits{mapPanelEdits: mapPanelEdits{facts: playable()}, byHero: herdCards}
				e := partyEditor(t, p, lang)
				e.Layout(width, 768)
				_ = e.Key("testplay")
				for round := 0; round < 5; round++ {
					cards := e.prompt.party.cards
					for row := range testPartyRows {
						for col, c := range cards {
							if row == testPartyRowSpells && !c.Caster {
								continue
							}
							before := map[string]TestPlayMember{}
							for i, k := range cards {
								before[testPartyKey(k)] = e.TestPartyChoice()[i]
							}
							s, _ := e.promptSpec()
							l := layoutPopup(s, e.font, e.bodyFont())
							r := l.GridCells[row*len(cards)+col]
							editorClickAt(e, r.Min.Add(r.Size().Div(2)), time.Now())
							got := e.TestPartyChoice()
							for i, k := range cards {
								want := before[testPartyKey(k)]
								if i == col {
									switch row {
									case testPartyRowPlays:
										want.On = !want.On
									case testPartyRowSkills:
										want.Skills = !want.Skills
									default:
										want.Spells = !want.Spells
									}
								}
								if got[i] != want {
									t.Fatalf("kind %d: press row %d card %d (%s): card %d (%s) holds %+v, want %+v", e.TestPartyHero(), row, col, c.Name, i, k.Name, got[i], want)
								}
							}
							s, _ = e.promptSpec()
							for i, k := range cards {
								cell := s.Grid.Cells[testPartyRowPlays*len(cards)+i]
								if cell.Ticked != got[i].On {
									t.Fatalf("kind %d: card %d (%s) plays %v, box shows %v", e.TestPartyHero(), i, k.Name, got[i].On, cell.Ticked)
								}
								if k.Caster {
									if s.Grid.Cells[testPartyRowSpells*len(cards)+i].Ticked != got[i].Spells {
										t.Fatalf("kind %d: spells box of %s", e.TestPartyHero(), k.Name)
									}
								}
							}
						}
					}
					partyNextKind(e)
				}
			})
		}
	}
}

// On Standard the window holds no grid of boxes, since the mission's own party
// plays and no box could change it; Custom shows it, and the window is the
// taller for it.
func TestThePartyWindowOnStandardShowsNoGrid(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			d := &declaredEdits{partyEdits{mapPanelEdits: mapPanelEdits{facts: playable()}, byHero: herdCards}}
			e := partyEditor(t, &d.partyEdits, lang)
			e.doc.Edits = d
			e.SetPanelProfile(nil, nil, func([]byte) error { return nil })
			_ = e.Key("testplay")
			s, _ := e.promptSpec()
			if s.Grid != nil {
				t.Fatal("Standard shows the grid")
			}
			standard := layoutPopup(s, e.font, e.bodyFont()).Rect
			partyPressMode(e)
			s, _ = e.promptSpec()
			if s.Grid == nil || layoutPopup(s, e.font, e.bodyFont()).Rect.Dy() <= standard.Dy() {
				t.Fatalf("Custom shows no grid or no taller window: %v", s.Grid != nil)
			}
		})
	}
}
