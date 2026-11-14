package ui

import (
	"image"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
	"againrom/pkg/words"
)

// partyEdits is a host that offers cards, the same for every kind of hero
// unless byHero says otherwise, and records each chosen party.
type partyEdits struct {
	mapPanelEdits
	cards  []TestPlayCard
	byHero func(hero int) []TestPlayCard
	heroes []int
	chosen [][]TestPlayMember
}

func (p *partyEdits) TestPlayCards(hero int) []TestPlayCard {
	if p.byHero != nil {
		return p.byHero(hero)
	}
	return p.cards
}
func (p *partyEdits) TestPlayParty(hero int, choice []TestPlayMember) (string, error) {
	p.heroes, p.chosen = append(p.heroes, hero), append(p.chosen, choice)
	return "Test play: the game opens mission 21", nil
}

func partyCards() []TestPlayCard {
	face := image.NewRGBA(image.Rect(0, 0, 32, 40))
	return []TestPlayCard{{Hero: true, Caster: true}, {Name: "Mage", Key: "npc22", Caster: true, Portrait: face}, {Name: "Monk", Key: "npc23"},
		{Name: "Prisoner", Key: "npc24", Caster: true}, {Name: "Brian", Key: "npc25"}, {Name: "Rood Glaen", Key: "npc26", Caster: true}}
}

func partyEditor(t *testing.T, p *partyEdits, lang string) *MapEditor {
	t.Helper()
	open := func(source string) (*InspectionDocument, error) {
		v, err := NewViewer("editor", terrain.Grid{Width: 50, Height: 50, Tiles: make([]uint16, 2500)}, &terrain.Tileset{})
		if err != nil {
			return nil, err
		}
		return &InspectionDocument{Viewer: v, Source: source, Width: 50, Height: 50, SourceBytes: []byte("map"), Edits: p}, nil
	}
	e := NewMapEditor(panelFont(), nil, open)
	e.SetWords(words.For(lang))
	e.Layout(1024, 768)
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	return e
}

func partyClick(t *testing.T, e *MapEditor, row, card int) {
	t.Helper()
	r, ok := e.FindControl(ControlQuery{Role: ControlPromptGridCell, N: row, K: card})
	if !ok {
		t.Fatalf("row %d of card %d takes no press", row, card)
	}
	editorClickAt(e, r.Min.Add(r.Size().Div(2)), time.Now())
}

// F5 asks who plays the map: six cards, the hero alone at first; a click
// switches a card and its boxes, Start plays with exactly the choice, Cancel
// plays nothing and the next window starts from the last choice.
func TestTestPlayAsksWhoPlaysTheMap(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			book := words.For(lang)
			p := &partyEdits{mapPanelEdits: mapPanelEdits{facts: playable()}, cards: partyCards()}
			e := partyEditor(t, p, lang)
			if err := e.Key("testplay"); err != nil {
				t.Fatal(err)
			}
			if !e.TestPartyOpen() || p.plays != 0 || len(p.chosen) != 0 {
				t.Fatalf("F5 did not ask: prompt %+v, %d plays", e.prompt, p.plays)
			}
			s, _ := e.promptSpec()
			g := s.Grid
			if s.Title != book.Text("editor.testparty.title") || g == nil || g.Cols != 6 || len(g.Cells) != 18 || len(s.Notes) != 0 {
				t.Fatalf("window %q: grid %+v notes %q", s.Title, g, s.Notes)
			}
			wantHeads := []string{book.Text("editor.testparty.hero"), "Mage", "Monk", "Prisoner", "Brian", "Rood Glaen"}
			wantRows := []string{book.Text("editor.testparty.plays"), book.Text("editor.testparty.skills"), book.Text("editor.testparty.spells")}
			if !slices.Equal(g.ColHeads, wantHeads) || !slices.Equal(g.RowHeads, wantRows) || s.Buttons[0] != book.Text("editor.testparty.start") {
				t.Fatalf("heads %q rows %q buttons %q", g.ColHeads, g.RowHeads, s.Buttons)
			}
			if g.Cells[1].Icon == nil || g.Cells[0].Icon != nil || g.Cells[0].Label != wantHeads[0] {
				t.Fatalf("the portrait box: mage %+v hero %+v", g.Cells[1], g.Cells[0])
			}
			if want := []TestPlayMember{{On: true}, {}, {}, {}, {}, {}}; !slices.Equal(e.TestPartyChoice(), want) || s.Disabled[testPartyStart] {
				t.Fatalf("the first choice is %+v, Start off %v", e.TestPartyChoice(), s.Disabled)
			}
			// A card whose character does not cast has no spells box.
			for card, c := range p.cards {
				if _, ok := e.FindControl(ControlQuery{Role: ControlPromptGridCell, N: testPartyRowSpells, K: card}); ok != c.Caster {
					t.Fatalf("card %d: spells box %t, caster %t", card, ok, c.Caster)
				}
			}
			partyClick(t, e, testPartyRowPlays, 1)
			partyClick(t, e, testPartyRowSkills, 1)
			partyClick(t, e, testPartyRowSpells, 1)
			partyClick(t, e, testPartyRowPlays, 4)
			partyClick(t, e, testPartyRowSkills, 0)
			want := []TestPlayMember{{On: true, Skills: true}, {On: true, Skills: true, Spells: true}, {}, {}, {On: true}, {}}
			if !slices.Equal(e.TestPartyChoice(), want) || !e.TestPartyOpen() {
				t.Fatalf("after the clicks the choice is %+v", e.TestPartyChoice())
			}
			if s, _ = e.promptSpec(); !s.Grid.Cells[1].Pressed || !s.Grid.Cells[1].Ticked || !s.Grid.Cells[6].Ticked || s.Grid.Cells[2].Pressed {
				t.Fatalf("the boxes do not show the choice: %+v", s.Grid.Cells[:8])
			}
			_ = e.Key("escape")
			if e.prompt.open() || len(p.chosen) != 0 || p.plays != 0 {
				t.Fatal("Cancel played the map or left the window open")
			}
			_ = e.Key("testplay")
			if !slices.Equal(e.TestPartyChoice(), want) {
				t.Fatalf("the second window starts from %+v", e.TestPartyChoice())
			}
			start, _ := e.controlPromptButtonRect(testPartyStart)
			editorClickAt(e, start.Min.Add(start.Size().Div(2)), time.Now())
			if e.prompt.open() || len(p.chosen) != 1 || !slices.Equal(p.chosen[0], want) || p.plays != 0 || !strings.HasPrefix(e.Message(), "Test play:") {
				t.Fatalf("Start: chosen %+v, message %q", p.chosen, e.Message())
			}
			// With nobody chosen Start takes no press and Enter plays nothing.
			_ = e.Key("testplay")
			for _, card := range []int{0, 1, 4} {
				partyClick(t, e, testPartyRowPlays, card)
			}
			if s, _ = e.promptSpec(); !s.Disabled[testPartyStart] {
				t.Fatal("Start takes a press with nobody chosen")
			}
			start, _ = e.controlPromptButtonRect(testPartyStart)
			editorClickAt(e, start.Min.Add(start.Size().Div(2)), time.Now())
			_ = e.Key("enter")
			if !e.TestPartyOpen() || len(p.chosen) != 1 {
				t.Fatalf("an empty party played: %+v", p.chosen)
			}
			partyClick(t, e, testPartyRowPlays, 5)
			_ = e.Key("enter")
			if e.prompt.open() || len(p.chosen) != 2 || !p.chosen[1][5].On || p.chosen[1][0].On {
				t.Fatalf("Enter: chosen %+v", p.chosen)
			}
		})
	}
}

// The party window holds the lines the notice before a test play lists, and
// a host that offers no cards keeps the notice.
func TestThePartyWindowHoldsTheNoticeLines(t *testing.T) {
	f, rows := endless()
	p := &partyEdits{mapPanelEdits: mapPanelEdits{facts: f, rows: rows}, cards: partyCards()}
	e := partyEditor(t, p, "en")
	_ = e.Key("testplay")
	s, _ := e.promptSpec()
	if notes := strings.Join(s.Notes, " / "); !e.TestPartyOpen() || !strings.Contains(notes, "The map can be played, but:") ||
		!strings.Contains(notes, "Start point:") || !strings.Contains(notes, "Victory:") {
		t.Fatalf("the window's notes: %q", s.Notes)
	}
	_ = e.Key("enter")
	if e.prompt.open() || len(p.chosen) != 1 {
		t.Fatalf("Start after the lines: prompt %+v, chosen %+v", e.prompt, p.chosen)
	}
	p.cards = nil
	_ = e.Key("testplay")
	if e.prompt.kind != promptNotice || p.plays != 0 {
		t.Fatalf("a host with no cards: prompt %+v", e.prompt)
	}
	_ = e.Key("enter")
	if p.plays != 1 || len(p.chosen) != 1 {
		t.Fatalf("Continue: %d plays, chosen %+v", p.plays, p.chosen)
	}
}

// The window names the kind of hero its cards stand beside. A press on that
// button takes the next kind the host offers cards for, the cards change
// with it, a spells tick stays remembered but unshown and unplayed on a card
// that no longer casts, Start plays for the kind shown, and the next window
// opens on it.
func TestThePartyWindowNamesTheHeroKind(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			book := words.For(lang)
			p := &partyEdits{mapPanelEdits: mapPanelEdits{facts: playable()}}
			p.byHero = func(hero int) []TestPlayCard {
				switch hero {
				case 0:
					return []TestPlayCard{{Hero: true}, {Name: "Reniesta", Key: "npc22", Caster: true}}
				case 2:
					return []TestPlayCard{{Hero: true, Caster: true}, {Name: "Danath", Key: "npc22"}}
				case 3:
					return []TestPlayCard{{Hero: true, Caster: true}, {Name: "Naira", Key: "npc22"}}
				}
				return nil
			}
			e := partyEditor(t, p, lang)
			_ = e.Key("testplay")
			label := func(hero int) string {
				return book.Text("editor.testparty.hero") + ": " + book.Text(testPartyKinds[hero])
			}
			s, _ := e.promptSpec()
			if e.TestPartyHero() != 0 || len(s.Buttons) != 3 || s.Buttons[testPartyKind] != label(0) || s.Grid.ColHeads[1] != "Reniesta" {
				t.Fatalf("the first window: kind %d, buttons %q, heads %q", e.TestPartyHero(), s.Buttons, s.Grid.ColHeads)
			}
			if _, ok := e.FindControl(ControlQuery{Role: ControlPromptGridCell, N: testPartyRowSpells, K: 0}); ok {
				t.Fatal("a fighter hero has a spells box")
			}
			partyClick(t, e, testPartyRowPlays, 1)
			partyClick(t, e, testPartyRowSpells, 1)
			next := func() {
				r, _ := e.controlPromptButtonRect(testPartyKind)
				editorClickAt(e, r.Min.Add(r.Size().Div(2)), time.Now())
			}
			// The host offers nothing for a female fighter: the press passes her.
			next()
			s, _ = e.promptSpec()
			want := []TestPlayMember{{On: true}, {On: true}}
			if e.TestPartyHero() != 2 || s.Buttons[testPartyKind] != label(2) || s.Grid.ColHeads[1] != "Danath" ||
				!slices.Equal(e.TestPartyChoice(), []TestPlayMember{{On: true}, {On: true, Spells: true}}) || s.Grid.Cells[2*2+1].Ticked {
				t.Fatalf("after the press: kind %d, buttons %q, heads %q, choice %+v", e.TestPartyHero(), s.Buttons, s.Grid.ColHeads, e.TestPartyChoice())
			}
			if _, ok := e.FindControl(ControlQuery{Role: ControlPromptGridCell, N: testPartyRowSpells, K: 1}); ok {
				t.Fatal("the fighter companion has a spells box")
			}
			_ = e.Key("enter")
			if len(p.heroes) != 1 || p.heroes[0] != 2 || !slices.Equal(p.chosen[0], want) {
				t.Fatalf("Start played for kinds %v with %+v", p.heroes, p.chosen)
			}
			_ = e.Key("testplay")
			if e.TestPartyHero() != 2 {
				t.Fatalf("the next window opens on kind %d", e.TestPartyHero())
			}
			next()
			next()
			if s, _ = e.promptSpec(); e.TestPartyHero() != 0 || s.Grid.ColHeads[1] != "Reniesta" {
				t.Fatalf("the round ends on kind %d, heads %q", e.TestPartyHero(), s.Grid.ColHeads)
			}
		})
	}
}

// At the least window a name wider than its card stands whole on two lines,
// in both languages; a window wide enough keeps one line.
func TestThePartyWindowShowsALongNameWhole(t *testing.T) {
	names := map[string]string{"en": "Rood Glaen", "ru": "\x90\xe3\xa4 \x83\xab\xa0\xed\xad"}
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			cards := partyCards()
			cards[5].Name = names[lang]
			p := &partyEdits{mapPanelEdits: mapPanelEdits{facts: playable()}, cards: cards}
			e := partyEditor(t, p, lang)
			// The body font is about as wide as the install's: nine pixels a letter.
			body := panelFont()
			for k := range body.Glyphs {
				body.Glyphs[k].Advance = 8
			}
			// The install's fonts hold a letter in one byte, as the host's text does.
			e.hostText = func(s string) string {
				var b []byte
				for _, r := range s {
					if r >= 0x80 {
						r = 0x80 + r%0x40
					}
					b = append(b, byte(r))
				}
				return string(b)
			}
			e.body = body
			for _, size := range []image.Point{{640, 480}, {1280, 960}} {
				e.Layout(size.X, size.Y)
				_ = e.Key("testplay")
				s, _ := e.promptSpec()
				l := layoutPopup(s, e.font, body)
				if !l.Rect.In(image.Rectangle{Max: size}) || len(l.GridColHeads) != len(cards) {
					t.Fatalf("%v: the window %v holds %d heads", size, l.Rect, len(l.GridColHeads))
				}
				for c, r := range l.GridColHeads {
					name := s.Grid.ColHeads[c]
					lines := headLines(body, name, r.Dx()-4)
					if strings.Join(lines, " ") != name || r.Dy() < len(lines)*fontHeight(body, 15) {
						t.Fatalf("%v: head %q is %q in a row %d high", size, name, lines, r.Dy())
					}
					for _, line := range lines {
						if body.Advance(line) > r.Dx()-4 || editorClip(body, line, r.Dx()-4) != line {
							t.Fatalf("%v: line %q of %q is %d wide in %d", size, line, name, body.Advance(line), r.Dx()-4)
						}
					}
					if narrow := size.X == 640; c == 5 && (len(lines) == 2) != narrow {
						t.Fatalf("%v: the long name takes %d lines (%d wide in %d)", size, len(lines), body.Advance(name), r.Dx()-4)
					}
				}
				_ = e.Key("escape")
			}
		})
	}
}
