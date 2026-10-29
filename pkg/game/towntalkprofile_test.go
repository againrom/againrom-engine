package game

import (
	"image"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/ui"
)

// fakeTalkHost is a town talk host made of plain data.
type fakeTalkHost struct {
	cursor  talkCursor
	open    bool
	ended   int
	tags    eventTags
	profile townTalkProfile
	faces   map[int]*image.RGBA
	windows map[int]image.Rectangle
}

func (h *fakeTalkHost) campaignTalk() *talkCursor {
	if !h.open {
		return nil
	}
	return &h.cursor
}
func (h *fakeTalkHost) talkOpen() bool              { return h.open }
func (h *fakeTalkHost) endTalk()                    { h.open, h.ended = false, h.ended+1; h.cursor.close() }
func (h *fakeTalkHost) talkAudience() EventAudience { return EventAudience{tags: h.tags} }
func (h *fakeTalkHost) talkArt() ui.TownTalkArt     { return ui.TownTalkArt{} }
func (h *fakeTalkHost) talkProfile() townTalkProfile {
	return h.profile
}
func (h *fakeTalkHost) talkFace(speaker int) (*image.RGBA, image.Rectangle, bool) {
	pic := h.faces[speaker]
	return pic, h.windows[speaker], pic != nil
}

type talkGame struct {
	tags    eventTags
	profile townTalkProfile
}

func talkGames() map[string]talkGame {
	return map[string]talkGame{
		"first":  {firstEventTags{}, firstTownTalkProfile},
		"second": {secondEventTags{}, secondTownTalkProfile},
	}
}

func openFakeTalk(g talkGame, p townTalkProfile, payload string) (*townTalkSeat, *fakeTalkHost) {
	h := &fakeTalkHost{open: true, tags: g.tags, profile: p,
		faces:   map[int]*image.RGBA{3: image.NewRGBA(image.Rect(0, 0, 3, 3)), 4: image.NewRGBA(image.Rect(0, 0, 4, 4))},
		windows: map[int]image.Rectangle{3: image.Rect(3, 3, 6, 6), 4: image.Rect(4, 4, 8, 8)}}
	h.cursor.open([]byte(payload))
	s := &townTalkSeat{}
	s.openTalk(h, []byte(payload), false)
	return s, h
}

// The part trim is profile data under either game's tag rules.
func TestTownTalkTrimIsProfileData(t *testing.T) {
	const payload = "<part=1 npc=3>\r\n  Words  \r\n"
	for game, g := range talkGames() {
		for _, trim := range []bool{false, true} {
			p := g.profile
			p.trim = trim
			s, _ := openFakeTalk(g, p, payload)
			v := s.view()
			want := s.dialogue.text
			if trim {
				want = "Words"
			}
			if !v.Open || v.Page.Text != want || (!trim && v.Page.Text == "Words") {
				t.Errorf("%s trim %v: text %q open %v, want %q", game, trim, v.Page.Text, v.Open, want)
			}
		}
	}
}

// The speaker tag and its face are data in both games.
func TestTownTalkSpeakerAndFaceAreData(t *testing.T) {
	for game, g := range talkGames() {
		three, h := openFakeTalk(g, g.profile, "<part=1 npc=3>\r\nOne\r\n")
		four, h4 := openFakeTalk(g, g.profile, "<part=1 npc=4>\r\nOne\r\n")
		v3, v4 := three.view(), four.view()
		if v3.Page.Face != h.faces[3] || v3.Page.FaceWindow != h.windows[3] || !v3.Page.Portrait {
			t.Errorf("%s: npc=3 shows face %p window %v portrait %v", game, v3.Page.Face, v3.Page.FaceWindow, v3.Page.Portrait)
		}
		if v4.Page.Face != h4.faces[4] || v4.Page.FaceWindow != h4.windows[4] || v4.Page.FaceWindow == v3.Page.FaceWindow {
			t.Errorf("%s: npc=4 shows face %p window %v", game, v4.Page.Face, v4.Page.FaceWindow)
		}
		other := image.NewRGBA(image.Rect(0, 0, 5, 5))
		h.faces[3], h.windows[3] = other, image.Rect(9, 9, 12, 12)
		if !three.readTalk() || three.view().Page.Face != other || three.view().Page.FaceWindow != image.Rect(9, 9, 12, 12) {
			t.Errorf("%s: a changed face did not reach the page", game)
		}
		nobody, _ := openFakeTalk(g, g.profile, "<part=1>\r\nOne\r\n")
		if v := nobody.view(); v.Page.Face != nil || v.Page.Portrait {
			t.Errorf("%s: a part naming nobody shows face %p portrait %v", game, v.Page.Face, v.Page.Portrait)
		}
	}
}

// The standing face is profile data under either game's tag rules.
func TestTownTalkStandingFaceIsProfileData(t *testing.T) {
	for game, g := range talkGames() {
		for _, standing := range []bool{false, true} {
			p := g.profile
			p.standingFace = standing
			s, h := openFakeTalk(g, p, "<part=1 npc=3>\r\nOne\r\n<part=2>\r\nTwo\r\n")
			if !s.turnTownTalk() || h.ended != 0 {
				t.Fatalf("%s standing %v: the first turn ended the talk", game, standing)
			}
			face := s.view().Page.Face
			if (face == h.faces[3]) != standing || (face == nil) == standing {
				t.Errorf("%s standing %v: part 2 shows face %p", game, standing, face)
			}
			if !s.turnTownTalk() || h.ended != 1 || s.view().Open {
				t.Errorf("%s standing %v: the last turn left the talk open", game, standing)
			}
		}
	}
}

// Opening before a part is profile data.
func TestTownTalkOpenBeforePartIsProfileData(t *testing.T) {
	for game, g := range talkGames() {
		for _, before := range []bool{false, true} {
			p := g.profile
			p.openBeforePart = before
			s, _ := openFakeTalk(g, p, "<part=2>\r\nTwo\r\n")
			v := s.view()
			if v.Open != before || (before && v.Page.Text != "Nothing to say") {
				t.Errorf("%s before %v: open %v text %q", game, before, v.Open, v.Page.Text)
			}
		}
	}
}

func setTalkRule(t *testing.T, rule string, p townTalkProfile) {
	t.Helper()
	old := townTalkRules[rule]
	townTalkRules[rule] = p
	t.Cleanup(func() { townTalkRules[rule] = old })
}

// The first game's screen follows its host rule's trim and the record window.
func TestFirstTownScreenTalkFollowsData(t *testing.T) {
	f, app, s := visibleOfferApp(t, 30)
	f.Archives.Containers = townTextFS(t, []synth.File{{Path: "text/shop/npc31m31.txt", Data: []byte("<part=1 npc=31>\r\n  One  \r\n<part=2>\r\nTwo\r\n")}})
	if err := app.HeadlessActivate("SHOP"); err != nil {
		t.Fatal(err)
	}
	if got := s.view().Page.Text; got != "  One  " {
		t.Fatalf("untrimmed page %q", got)
	}
	p := firstTownTalkProfile
	p.trim = true
	setTalkRule(t, s.hostRules().Talk, p)
	if got := s.view().Page.Text; got != "One" {
		t.Fatalf("trimmed page %q", got)
	}
	pic := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for _, x := range []int{2, 7} {
		s.resolver = speakerResolver{npcFaces: map[int32]data.NPCFace{31: {HasWindow: true, WindowX: x, WindowY: 3}},
			playerFigures: map[int]*image.RGBA{31: pic}}
		if !s.readTalk() {
			t.Fatal("the page did not read")
		}
		if v := s.view(); v.Page.Face != pic || v.Page.FaceWindow != ui.NoticeFaceWindow(x, 3) {
			t.Errorf("window x %d: face %p window %v", x, v.Page.Face, v.Page.FaceWindow)
		}
	}
}

// The second game's screen follows the same data.
func TestSecondTownScreenTalkFollowsData(t *testing.T) {
	const payload = "<part=1 npc=42>\r\n  One  \r\n"
	for _, x := range []int{2, 7} {
		s := &townScreen{shopController: shopController{sess: &CampaignSession{Town: &Town{second: newSecondCampaign()}},
			in: &InstallResources{NPCFaces: map[int32]data.NPCFace{42: {HasWindow: true, WindowX: x, WindowY: 3}}}},
			art: fakeTownArt{}, tags: secondEventTags{}}
		s.setTownDialogue([]byte(payload))
		v := s.view()
		if !v.Open || v.Page.Text != "One" || v.Page.FaceWindow != ui.NoticeFaceWindow(x, 3) || !v.Page.Portrait {
			t.Errorf("window x %d: text %q window %v portrait %v", x, v.Page.Text, v.Page.FaceWindow, v.Page.Portrait)
		}
		p := secondTownTalkProfile
		p.trim = false
		setTalkRule(t, s.hostRules().Talk, p)
		if got := s.view().Page.Text; !strings.HasPrefix(got, "  One") {
			t.Errorf("window x %d: untrimmed page %q", x, got)
		}
		townTalkRules[s.hostRules().Talk] = secondTownTalkProfile
	}
}
