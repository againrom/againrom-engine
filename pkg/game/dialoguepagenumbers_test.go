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
	{"page 12 before page 1", "<part=12>\r\ntwelve\r\n<part=1>\r\none\r\n<part=2>\r\ntwo", []string{"twelve", "two"}},
	{"no page 1", "<part=2>\r\ntwo", []string{"Nothing to say", "two"}},
	{"no page 1 or 2", "<part=3>\r\nthree", []string{"Nothing to say"}},
}

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
	for _, tc := range pagedMessages {
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

// An accepted tag without words still stores its tips (DIALOGUE-050).
func TestMissionDialogueKeepsTheTipsOfAPageWithoutWords(t *testing.T) {
	for _, tc := range []struct{ name, payload string }{
		{"page 1", "<part=1 tips=7>"},
		{"page 2", "<part=1>\r\none\r\n<part=2 tips=7>"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			driver, viewer := missionDialogue(t, tc.payload, nil)
			path, ok := MissionTipPath(driver.mission.edition(), driver.mission.number, 7)
			if !ok {
				t.Fatal("no mission tip path for tip 7")
			}
			driver.mission.src.(missionSource)[path] = []byte("the tip")
			for presses := 0; ; presses++ {
				if _, _, open := viewer.NoticeState(); !open {
					break
				}
				if presses == 3 {
					t.Fatal("the dialogue did not close")
				}
				driver.advanceNotice()
			}
			if got := viewer.MissionTip().Text; got != "the tip" {
				t.Fatalf("mission tip = %q, want the tip of the wordless page", got)
			}
		})
	}
}

// A failed page-1 lookup keeps the pane and skips the face (DIALOGUE-049).
func TestMissionDefaultPageShowsNoFace(t *testing.T) {
	red, blue := faceDot(), faceDot()
	mw, v := missionDialogue(t, "<npc=25,part=2>\r\ntwo\r\n<npc=21,part=1>", missionFaces{21: red, 25: blue})
	if text, _, _ := v.NoticeState(); text != "Nothing to say" {
		t.Fatalf("page 1 = %q, want the default text", text)
	}
	if portrait, face := v.NoticeSpeaker(); !portrait || face != nil {
		t.Fatalf("default page portrait %v face %v, want the pane and no face", portrait, face != nil)
	}
	mw.advanceNotice()
	if _, face := v.NoticeSpeaker(); face != blue {
		t.Fatal("page 2's speaker's picture did not arrive")
	}
}

// The second game follows the first while no second-game claim states its
// panel's default (DIV-4053).
func TestBothGamesOpenAMessageWithoutPageOneOnTheDefault(t *testing.T) {
	for _, g := range []struct {
		name  string
		files *gameFiles
	}{{"first", &firstGameFiles}, {"second", &secondGameFiles}} {
		if g.files.dialogueDefault != dialogueDefaultText {
			t.Errorf("%s game default = %q, want %q", g.name, g.files.dialogueDefault, dialogueDefaultText)
		}
	}
}
