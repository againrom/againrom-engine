package ui

import (
	"slices"
	"testing"
	"time"

	"againrom/pkg/words"
)

// declaredEdits is a host whose mission declares a party of its own.
type declaredEdits struct{ partyEdits }

func (d *declaredEdits) TestPlayDeclaresParty() bool { return true }

func partyPressMode(e *MapEditor) {
	r, _ := e.controlPromptButtonRect(testPartyMode)
	editorClickAt(e, r.Min.Add(r.Size().Div(2)), time.Now())
}

// A mission that declares its own party shows the switch Standard / Custom
// in the party window. Standard starts the mission as the game starts it
// today and sends no choice; Custom sends the window's choice. The switch is
// remembered in the editor and in the profile; a mission without a party of
// its own shows no switch.
func TestThePartyWindowSwitchesBetweenTheMissionsPartyAndTheChoice(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			book := words.For(lang)
			var file []byte
			save := func(b []byte) error { file = append([]byte(nil), b...); return nil }
			d := &declaredEdits{partyEdits{mapPanelEdits: mapPanelEdits{facts: playable()}, byHero: kindCards}}
			e := partyEditor(t, &d.partyEdits, lang)
			e.doc.Edits = d
			e.SetPanelProfile(nil, nil, save)
			_ = e.Key("testplay")
			s, _ := e.promptSpec()
			label := func(w string) string { return book.Text("editor.testparty.mode") + ": " + book.Text(w) }
			if len(s.Buttons) != 4 || s.Buttons[testPartyMode] != label("editor.testparty.standard") || s.Disabled[testPartyStart] || !s.Disabled[testPartyKind] {
				t.Fatalf("Standard window: buttons %q disabled %v", s.Buttons, s.Disabled)
			}
			_ = e.Key("enter")
			if d.plays != 1 || len(d.chosen) != 0 || e.TestPartyOpen() {
				t.Fatalf("Standard: plays %d, chosen %v", d.plays, d.chosen)
			}

			_ = e.Key("testplay")
			partyPressMode(e)
			s, _ = e.promptSpec()
			if s.Buttons[testPartyMode] != label("editor.testparty.custom") || s.Disabled[testPartyKind] {
				t.Fatalf("Custom window: buttons %q disabled %v", s.Buttons, s.Disabled)
			}
			partyClick(t, e, testPartyRowSkills, 0)
			_ = e.Key("enter")
			if d.plays != 1 || len(d.chosen) != 1 || !slices.Equal(d.chosen[0], []TestPlayMember{{On: true, Skills: true}, {}, {}}) {
				t.Fatalf("Custom: plays %d, chosen %v", d.plays, d.chosen)
			}

			again := &declaredEdits{partyEdits{mapPanelEdits: mapPanelEdits{facts: playable()}, byHero: kindCards}}
			f := partyEditor(t, &again.partyEdits, lang)
			f.doc.Edits = again
			f.SetPanelProfile(file, nil, save)
			_ = f.Key("testplay")
			if s, _ = f.promptSpec(); s.Buttons[testPartyMode] != label("editor.testparty.custom") {
				t.Fatalf("a new editor opens with %q", s.Buttons)
			}

			plain := partyEditor(t, &partyEdits{mapPanelEdits: mapPanelEdits{facts: playable()}, byHero: kindCards}, lang)
			_ = plain.Key("testplay")
			if s, _ = plain.promptSpec(); len(s.Buttons) != 3 {
				t.Fatalf("a mission without a party of its own shows %q", s.Buttons)
			}
		})
	}
}

// The hero that a window shows in because nobody was chosen belongs to that
// window: a map that offers none of the chosen companions opens with the hero
// in, and the choice the map maker remembered for the other map stays, also
// after Cancel.
func TestThePartyWindowOnAnotherMapKeepsTheRememberedChoice(t *testing.T) {
	var file []byte
	save := func(b []byte) error { file = append([]byte(nil), b...); return nil }
	a := []TestPlayCard{{Hero: true}, {Name: "Reniesta", Key: "npc24"}, {Name: "Brian", Key: "npc25"}}
	b := []TestPlayCard{{Hero: true}, {Name: "Danath", Key: "npc30"}}
	p := &partyEdits{mapPanelEdits: mapPanelEdits{facts: playable()}, cards: a}
	e := partyEditor(t, p, "en")
	e.SetPanelProfile([]byte(`{"testparty":{"hero":0,"cards":{"hero":{"on":false},"npc24":{"on":true}}}}`), nil, save)
	want := []TestPlayMember{{}, {On: true}, {}}
	_ = e.Key("testplay")
	if got := e.TestPartyChoice(); !slices.Equal(got, want) {
		t.Fatalf("map A opens %v", got)
	}
	_ = e.Key("escape")
	p.cards = b
	_ = e.Key("testplay")
	if got := e.TestPartyChoice(); !slices.Equal(got, []TestPlayMember{{On: true}, {}}) {
		t.Fatalf("map B opens %v", got)
	}
	_ = e.Key("escape")
	p.cards = a
	_ = e.Key("testplay")
	if got := e.TestPartyChoice(); !slices.Equal(got, want) {
		t.Fatalf("map A again %v; file %s", got, file)
	}
}

// On Standard the cards and the hero kind button are not in use: no box takes
// a press, and Custom shows the remembered choice as it was.
func TestThePartyWindowOnStandardTakesNoCardPress(t *testing.T) {
	d := &declaredEdits{partyEdits{mapPanelEdits: mapPanelEdits{facts: playable()}, byHero: kindCards}}
	e := partyEditor(t, &d.partyEdits, "en")
	e.doc.Edits = d
	e.SetPanelProfile(nil, nil, func([]byte) error { return nil })
	_ = e.Key("testplay")
	before := slices.Clone(e.TestPartyChoice())
	for row := range testPartyRows {
		if _, ok := e.FindControl(ControlQuery{Role: ControlPromptGridCell, N: row, K: 1}); ok {
			t.Fatalf("row %d takes a press on Standard", row)
		}
	}
	if s, _ := e.promptSpec(); !s.Disabled[testPartyKind] {
		t.Fatalf("the kind button is live on Standard")
	}
	partyPressMode(e)
	if got := e.TestPartyChoice(); !slices.Equal(got, before) {
		t.Fatalf("Custom shows %v, was %v", got, before)
	}
	partyClick(t, e, testPartyRowPlays, 1)
	if got := e.TestPartyChoice(); got[1] != (TestPlayMember{On: true}) {
		t.Fatalf("Custom takes no press: %v", got)
	}
}
