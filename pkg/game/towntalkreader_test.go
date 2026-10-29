package game

import (
	"testing"

	"againrom/internal/synth"
)

// talkReaderGames is each game's reading of one cursor: the one page lookup
// under each game's tag rules and talk profile.
func talkReaderGames() map[string]func(c talkCursor) (string, bool) {
	read := func(tags eventTags, p townTalkProfile) func(c talkCursor) (string, bool) {
		return func(c talkCursor) (string, bool) {
			d := newTownDialogue(c.payload)
			ok := d.lookup(c, EventAudience{tags: tags}, p)
			return d.text, ok
		}
	}
	return map[string]func(c talkCursor) (string, bool){
		"first":  read(firstEventTags{}, firstTownTalkProfile),
		"second": read(secondEventTags{}, secondTownTalkProfile),
	}
}

// Both games read every part case of their own text shape through the one
// reader: a present part, an empty part, a part whose line break lies past the
// next tag (its words run on from there), a part past the last, a missing
// part, the page split at the next tag or the end of the text, and a text
// with no line break at all.
func TestTownTalkReaderPartCases(t *testing.T) {
	shapes := map[string]string{
		"first":  "<part=1 npc=3>\r\nFirst page\r\n<part=2>\r\n\r\n<part=3>no break<part=5 tips=2>\r\nLast\r\nline",
		"second": "<part=1 npc=42>\r\nFirst page\r\n<part=2>\r\n\r\n<part=3>no break<part=5>\r\nLast\r\nline\x00<part=6>\r\nafter NUL\r\n",
	}
	cases := []struct {
		name string
		part int
		text string
		ok   bool
	}{
		{"present", 1, "First page", true},
		{"empty", 2, "", true},
		{"line break past the next tag", 3, "Last\r\nline", true},
		{"missing", 4, "", false},
		{"split at the end of the text", 5, "Last\r\nline", true},
		{"past the last", 6, "", false},
		{"far past the last", 64, "", false},
	}
	for game, read := range talkReaderGames() {
		if text, ok := read(talkCursor{payload: []byte("<part=1>no break"), part: 1}); ok {
			t.Errorf("%s: a part with no line break read %q", game, text)
		}
		for _, tc := range cases {
			text, ok := read(talkCursor{payload: []byte(shapes[game]), part: tc.part})
			if ok != tc.ok || (ok && text != tc.text) {
				t.Errorf("%s %s: part %d = %q %v, want %q %v", game, tc.name, tc.part, text, ok, tc.text, tc.ok)
			}
		}
	}
}

// The town talk of either game reads no part below 1; the shared scan reads
// part 0.
func TestTownTalkReaderLowestPart(t *testing.T) {
	payload := []byte("<part=0>\r\nzero\r\n")
	for game, read := range talkReaderGames() {
		if _, ok := read(talkCursor{payload: payload}); ok {
			t.Errorf("the %s game read part 0", game)
		}
	}
	c := talkCursor{payload: payload}
	if r := c.read(EventAudience{}, nil); !r.ok || r.text != "zero\r\n" {
		t.Errorf("the shared scan read part 0 as %q %v, want zero", r.text, r.ok)
	}
}

// The first game's panel takes the speaker of every tag naming the part,
// including one the audience rejects, and the tips of the accepted tag even
// when its words fail; the shared reader leaves both to it.
func TestTownTalkReaderFirstGameTagState(t *testing.T) {
	payload := []byte("<part=1 iamfemale npc=5>\r\nher\r\n<part=1 iammale>\r\nhim\r\n<part=2 npc=8 iamfemale>\r\nx\r\n<part=3 tips=4>no break")
	d := newTownDialogue(payload)
	p := firstTownTalkProfile
	if !d.lookup(talkCursor{payload: payload, part: 1}, EventAudience{}, p) || d.text != "him" || d.portrait || d.speaker != 5 {
		t.Fatalf("part 1: text %q portrait %v speaker %d, want him false 5", d.text, d.portrait, d.speaker)
	}
	if d.lookup(talkCursor{payload: payload, part: 2}, EventAudience{}, p) || d.text != "him" || !d.portrait || d.speaker != 8 {
		t.Fatalf("rejected part 2: text %q portrait %v speaker %d, want him true 8", d.text, d.portrait, d.speaker)
	}
	if d.lookup(talkCursor{payload: payload, part: 3}, EventAudience{}, p) || d.tips != 4 || d.displayPart != 1 {
		t.Fatalf("part 3: tips %d shown part %d, want 4 and 1", d.tips, d.displayPart)
	}
}

// The cursor opens at part 1, turns one part at a time, closes to nothing and
// reopens at part 1.
func TestTalkCursorTurnsAndReopens(t *testing.T) {
	var c talkCursor
	c.open([]byte("<part=1>\r\none\r\n<part=2>\r\ntwo"))
	for want, text := range []string{"one", "two"} {
		r := c.read(EventAudience{}, nil)
		if c.part != want+1 || !r.ok || r.text != text {
			t.Fatalf("part %d read %q %v", c.part, r.text, r.ok)
		}
		c.next()
	}
	if r := c.read(EventAudience{}, nil); r.accepted || c.part != 3 {
		t.Fatalf("past the last page: part %d accepted %v", c.part, r.accepted)
	}
	c.close()
	if c.payload != nil || c.part != 0 {
		t.Fatal("close kept the conversation")
	}
	c.open([]byte("<part=1>\r\nagain"))
	if r := c.read(EventAudience{}, nil); c.part != 1 || r.text != "again" {
		t.Fatalf("reopen: part %d %q", c.part, r.text)
	}
}

// The first game's screen pages its conversation through the seat: OK turns
// to the next part, the last OK closes to the room, a reopen starts at part 1.
func TestFirstTownTalkCursorThroughTheSeat(t *testing.T) {
	f, app, s := visibleOfferApp(t, 30)
	f.Archives.Containers = townTextFS(t, []synth.File{{Path: "text/shop/npc31m31.txt", Data: []byte("<part=1 npc=31>\r\nOne\r\n<part=2>\r\nTwo\r\n")}})
	if err := app.HeadlessActivate("SHOP"); err != nil {
		t.Fatal(err)
	}
	for part, text := range []string{"One", "Two\r\n"} {
		if s.room != roomTalk || s.talkPage.part != part+1 || s.dialogue.text != text {
			t.Fatalf("page %d: room %v part %d %q", part+1, s.room, s.talkPage.part, s.dialogue.text)
		}
		if _, open := s.TownDialogue(); !open || s.TownDialogueShows() == 0 {
			t.Fatalf("page %d: the window is closed", part+1)
		}
		s.AdvanceTownDialogue()
	}
	if s.room != roomShop || s.talkPage.part != 3 || s.TownDialogueShows() != 0 {
		t.Fatalf("the last OK left room %v part %d", s.room, s.talkPage.part)
	}
	if !s.openTownDialogue(s.dialogueBuilding, s.offer, s.npc) || s.talkPage.part != 1 || s.dialogue.text != "One" {
		t.Fatalf("reopen: part %d %q", s.talkPage.part, s.dialogue.text)
	}
}

// The second game's campaign pages its conversation through the seat the
// same way, and a closed conversation leaves no cursor behind.
func TestSecondTownTalkCursorThroughTheSeat(t *testing.T) {
	c := newSecondCampaign()
	s := &townScreen{shopController: shopController{sess: &CampaignSession{Town: &Town{second: c}}, in: &InstallResources{}},
		art: fakeTownArt{}, tags: secondEventTags{}}
	for round := 0; round < 2; round++ {
		s.setTownDialogue([]byte("<part=1>\r\nOne\r\n<part=2>\r\nTwo\r\n"))
		for part, text := range []string{"One", "Two\r\n"} {
			body, ok := s.dialogueBody()
			if c.talk.part != part+1 || !ok || body != text || s.TownDialogueShows() != 1 {
				t.Fatalf("round %d page %d: part %d %q %v", round, part+1, c.talk.part, body, ok)
			}
			s.AdvanceTownDialogue()
		}
		if c.talk.payload != nil || c.talk.part != 0 || s.TownDialogueShows() != 0 || c.noticePending() {
			t.Fatalf("round %d: the last OK left part %d", round, c.talk.part)
		}
	}
}
