package game

import (
	"testing"

	"againrom/internal/synth"
)

// pagedMessages are authored messages whose page numbers skip, repeat or run
// out of order, with the pages DLG-PAGES-091 says every dialogue panel shows:
// pages by number from 1, each from the first tag in file order the audience
// accepts, ending at the first missing number.
var pagedMessages = []struct {
	name, payload string
	pages         []string
}{
	{"missing page number", "<part=1>\r\none\r\n<part=3>\r\nthree", []string{"one"}},
	{"repeated page number", "<part=1>\r\nfirst\r\n<part=1>\r\nsecond\r\n<part=2>\r\ntwo", []string{"first", "two"}},
	{"out of order", "<part=2>\r\ntwo\r\n<part=3>\r\nthree\r\n<part=1>\r\none", []string{"one", "two", "three"}},
}

// noFirstPage is a message with no page 1: the town panel shows its default
// text and then page 2. The mission panel opens no notice for it
// (openDialogue), so only the town panel is checked with it.
var noFirstPage = struct {
	name, payload string
	pages         []string
}{"no page 1", "<part=2>\r\ntwo", []string{"Nothing to say", "two"}}

func TestMissionDialogueShowsPagesByNumber(t *testing.T) {
	for _, tc := range pagedMessages {
		t.Run(tc.name, func(t *testing.T) {
			driver, viewer := missionDialogue(t, tc.payload, nil)
			for i, want := range tc.pages {
				if i > 0 {
					driver.advanceNotice()
				}
				if got, _, open := viewer.NoticeState(); !open || got != want {
					t.Fatalf("page %d = %q (open %v), want %q", i+1, got, open, want)
				}
			}
			driver.advanceNotice()
			if _, _, open := viewer.NoticeState(); open {
				t.Fatal("the dialogue went on past its last numbered page")
			}
		})
	}
}

func TestTownDialogueShowsPagesByNumber(t *testing.T) {
	for _, tc := range append(pagedMessages, noFirstPage) {
		t.Run(tc.name, func(t *testing.T) {
			f, app, town := visibleOfferApp(t, 30)
			f.Archives.Containers = townTextFS(t, []synth.File{{Path: "text/shop/npc31m31.txt", Data: []byte(tc.payload)}})
			if err := app.HeadlessActivate("SHOP"); err != nil {
				t.Fatal(err)
			}
			for i, want := range tc.pages {
				if i > 0 {
					if err := app.HeadlessKey("enter"); err != nil {
						t.Fatal(err)
					}
				}
				if town.room != roomTalk || town.dialogue.text != want {
					t.Fatalf("page %d = %q (room %v), want %q", i+1, town.dialogue.text, town.room, want)
				}
			}
			if err := app.HeadlessKey("enter"); err != nil {
				t.Fatal(err)
			}
			if town.room == roomTalk {
				t.Fatal("the dialogue went on past its last numbered page")
			}
		})
	}
}
