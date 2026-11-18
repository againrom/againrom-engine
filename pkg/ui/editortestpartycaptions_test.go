package ui

import (
	"fmt"
	"testing"

	"againrom/pkg/words"
)

// Every button caption of the test party window comes whole from the words
// table: Start is a word, the hero button is the table's Hero format filled
// with the kind's word, Cancel is the shared word.
func TestThePartyWindowButtonCaptionsComeFromTheWordsTable(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			book := words.For(lang)
			p := &partyEdits{mapPanelEdits: mapPanelEdits{facts: playable()}, cards: partyCards()}
			e := partyEditor(t, p, lang)
			if err := e.Key("testplay"); err != nil {
				t.Fatal(err)
			}
			s, _ := e.promptSpec()
			want := []string{
				book.Text("editor.testparty.start"),
				fmt.Sprintf(book.Text("editor.testparty.herokind"), book.Text(testPartyKinds[e.prompt.party.hero])),
				book.Text("editor.button.cancel"),
			}
			if len(s.Buttons) != len(want) {
				t.Fatalf("buttons %q; want %q", s.Buttons, want)
			}
			for i, w := range want {
				if s.Buttons[i] != w {
					t.Errorf("button %d is %q; the words table gives %q", i, s.Buttons[i], w)
				}
			}
		})
	}
}
