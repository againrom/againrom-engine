package game

import (
	"image"
	"slices"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/ui"
)

// The original prints nothing when the player leaves a town room or ends a
// conversation: the shop's leave and a dialogue's close send their own messages
// and no text (SHOP-SCREEN-035, DLG-LIFE-005), and none of the message-post call
// sites sits in a shop, tavern, school or dialogue routine (MISSION-MSGPOST-058).
// Each press below is the one pkg/ui sends across the seam, and the line it
// answers is the line the town would print.

// townTalkFiles is the shipped-shaped dialogue text the conversations below
// read: the chapter 30 tavern NPCs and shop offer, the chapter 40 shop and school
// offers, and the tavern NPC of the shell fixture.
func townTalkFiles() []synth.File {
	return []synth.File{
		{Path: "text/inn/npc/npc22m30.txt", Data: []byte("<part=1>\r\nFirst real line\r\n<part=2>\r\nSecond real line")},
		{Path: "text/inn/npc/npc90m30.txt", Data: []byte("<part=1>\r\nNothing for you today")},
		{Path: "text/inn/npc/npc09m30.txt", Data: []byte("<part=1>\r\nNothing for you\r\n<part=2>\r\nCome back later")},
		{Path: "text/shop/npc31m31.txt", Data: []byte("<part=1>\r\nShipped shop offer")},
		{Path: "text/shop/npc31m40.txt", Data: []byte("<part=1>\r\nThe merchant has a job")},
		{Path: "text/training/npc34m41.txt", Data: []byte("<part=1>\r\nThe trainer has a job")},
		{Path: "text/inn/mercenary/npc03.txt", Data: []byte("<part=1>\r\nSquad three speaks\r\n<part=2>\r\nand answers")},
	}
}

// roomExitFront is the fixture town at chapter 30, or at chapter 40 once its main
// mission is won, holding the dialogue text above.
func roomExitFront(t *testing.T, chapter int) (*FrontEnd, *townScreen) {
	t.Helper()
	c := townCampaign(t)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil), Font: resolved(missionFont(), nil), Archives: &Archives{Containers: townTextFS(t, townTalkFiles())}}, CampaignSession: CampaignSession{Town: NewTown(c)}}
	f.Town.Arrive()
	if chapter == 40 {
		f.Town.Won(30)
	}
	return f, f.TownScreen().(*townScreen)
}

// endConversation presses OK until the open conversation closes and returns the
// answer of the press that closed it. Every earlier press turns a page and must
// post nothing as well.
func endConversation(t *testing.T, s *townScreen) ui.TownAction {
	t.Helper()
	for n := 0; n < 16; n++ {
		if s.room != roomTalk {
			t.Fatal("no conversation is open")
		}
		act := s.AdvanceTownDialogue()
		if s.room != roomTalk {
			return act
		}
		if act.Msg != "" {
			t.Fatalf("turning a page posted %q", act.Msg)
		}
	}
	t.Fatal("the conversation did not end within 16 presses")
	return ui.TownAction{}
}

func TestLeavingATownRoomPostsNoLine(t *testing.T) {
	t.Run("shop", func(t *testing.T) {
		_, s := shopRoom(t, nil)
		if act := click(s, ui.ShopControlButton, 0); act.Msg == "" {
			t.Fatal("clearing the table posted no line, so this seam cannot show one")
		}
		if act := click(s, ui.ShopControlButton, 3); act.Msg != "" {
			t.Errorf("the shop's Exit posted %q", act.Msg)
		}
		if s.room != roomSquare {
			t.Errorf("the shop's Exit left room %v, want the square", s.room)
		}
	})

	t.Run("tavern", func(t *testing.T) {
		f := shellFrontEnd()
		s := f.townUI
		exit := ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: tavernButtonExit}
		if act := s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: tavernButtonHire}, false); act.Msg != "" || !f.Town.MercenaryHired(3) {
			t.Fatalf("hire = %q, hired %v, want the squad hired with no line", act.Msg, f.Town.MercenaryHired(3))
		}
		if act := s.TownSurfaceClick(exit, false); act.Msg != "" {
			t.Errorf("the tavern's Exit posted %q", act.Msg)
		}
		if s.room != roomSquare {
			t.Errorf("the tavern's Exit left room %v, want the square", s.room)
		}
	})

	t.Run("school", func(t *testing.T) {
		f := shellFrontEnd()
		s := f.townUI
		s.Back()
		s.Choose(2)
		if s.room != roomSchool {
			t.Fatalf("the school door opened room %v", s.room)
		}
		s.schoolCell = 0
		if act := s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: 0}, false); act.Msg == "" {
			t.Fatal("a completed training posted no line, so this seam cannot show one")
		}
		if act := s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: 1}, false); act.Msg != "" {
			t.Errorf("the school's Exit posted %q", act.Msg)
		}
		if s.room != roomSquare {
			t.Errorf("the school's Exit left room %v, want the square", s.room)
		}
	})

	t.Run("gates", func(t *testing.T) {
		c := townCampaign(t)
		town := NewTown(c)
		town.Arrive()
		town.announceMission(30)
		f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil), Archives: &Archives{Containers: worldMapFixtureFS(t)}, Font: resolved(missionFont(), nil)}, CampaignSession: CampaignSession{Town: town}}
		s := f.TownScreen().(*townScreen)
		s.Choose(3)
		if s.room != roomGates {
			t.Fatalf("the gates door opened room %v", s.room)
		}
		card := ui.WorldMapCardRect(-1)
		if act := s.WorldMapClick(image.Pt((card.Min.X+card.Max.X)/2, (card.Min.Y+card.Max.Y)/2)); act.Msg != "" {
			t.Errorf("the town card posted %q", act.Msg)
		}
		if !s.AtTownSquare() {
			t.Errorf("the town card left room %v, want the square", s.room)
		}
	})
}

func TestEndingAConversationPostsNoLine(t *testing.T) {
	for _, tc := range []struct {
		name    string
		chapter int
		route   []int
		escape  bool
		room    townRoom
	}{
		{"tavern mission", 30, []int{0, 0}, false, roomTavern},
		{"tavern mission ended by Escape", 30, []int{0, 0}, true, roomTavern},
		{"tavern NPC with nothing to give", 30, []int{0, 1}, false, roomTavern},
		{"shop offer", 30, []int{1}, false, roomShop},
		{"shop offer ended by Escape", 30, []int{1}, true, roomShop},
		{"school offer", 40, []int{2}, false, roomSchool},
		{"school offer ended by Escape", 40, []int{2}, true, roomSchool},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, s := roomExitFront(t, tc.chapter)
			for _, i := range tc.route {
				s.Choose(i)
			}
			if s.room != roomTalk {
				t.Fatalf("the route opened room %v, want a conversation", s.room)
			}
			// The shop and the school register the offer as the window opens; the
			// tavern queues it until the player leaves.
			mission, tavern := s.offer.Mission, s.dialogueBuilding == TownTavern
			if got := slices.Contains(f.Town.Available(), mission); got != (mission > 0 && !tavern) {
				t.Fatalf("with the conversation open and no key pressed, available %v, mission %d", f.Town.Available(), mission)
			}
			var act ui.TownAction
			if tc.escape {
				for n := 0; s.room == roomTalk && n < 16; n++ {
					s.Back()
				}
			} else {
				act = endConversation(t, s)
			}
			if act.Msg != "" {
				t.Errorf("the press that ended the conversation posted %q", act.Msg)
			}
			if s.room != tc.room {
				t.Errorf("the conversation returned to room %v, want %v", s.room, tc.room)
			}
			if got := slices.Contains(f.Town.Available(), mission); got != (mission > 0 && !tavern) {
				t.Errorf("after the conversation, available %v, mission %d", f.Town.Available(), mission)
			}
			if tavern {
				s.Back()
				if got := slices.Contains(f.Town.Available(), mission); got != (mission > 0) {
					t.Errorf("after leaving the tavern, available %v, mission %d", f.Town.Available(), mission)
				}
			}
		})
	}
}

func TestEndingAMercenaryConversationPostsNoLine(t *testing.T) {
	for _, tc := range []struct {
		name       string
		text       bool
		diagnostic bool
	}{
		{"shipped text", true, false},
		{"missing text shows the diagnostic", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := shellFrontEnd()
			if tc.text {
				f.Archives = &Archives{Containers: townTextFS(t, townTalkFiles())}
			}
			s := f.townUI
			s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 0}, false)
			if act := s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: tavernButtonTalk}, false); act.Msg != "" {
				t.Fatalf("Talk posted %q", act.Msg)
			}
			if s.room != roomTalk || (s.talkDiagnostic != "") != tc.diagnostic {
				t.Fatalf("Talk opened room %v with diagnostic %q", s.room, s.talkDiagnostic)
			}
			if act := endConversation(t, s); act.Msg != "" {
				t.Errorf("the press that ended the conversation posted %q", act.Msg)
			}
			if s.room != roomTavern {
				t.Errorf("the conversation returned to room %v, want the tavern", s.room)
			}
		})
	}
}

// The same routes through App input, where the flow replaces the line with
// whatever the press answered.
func TestLeavingTheShopThroughAppInputPostsNoLine(t *testing.T) {
	shelf := []ShopItem{{Code: data.ItemCode(shopShiftDragHelm), Price: 25, Count: 11}}
	f, a, s := shopShiftDragApp(t, nil, shelf)
	tap := func(surface string, index int) {
		t.Helper()
		x, y, err := a.HeadlessShopPoint(surface, index)
		if err != nil {
			t.Fatal(err)
		}
		shopTap(t, a, image.Pt(x, y), "press", "release")
	}
	tap("shelf", 0)
	if len(f.Shop.Table()) == 0 {
		t.Fatal("the shelf press put nothing on the table")
	}
	tap("button", 0)
	if got := a.HeadlessMessage(); got != "" {
		t.Fatalf("clearing the table showed the status echo %q", got)
	}
	tap("button", 3)
	if s.room != roomSquare {
		t.Fatalf("the shop's Exit left room %v, want the square", s.room)
	}
	if got := a.HeadlessMessage(); got != "" {
		t.Errorf("leaving the shop showed %q", got)
	}
}

func TestEndingATavernConversationThroughAppInputPostsNoLine(t *testing.T) {
	f := shellFrontEnd()
	f.Font = resolved(missionFont(), nil)
	f.Archives = &Archives{Containers: townTextFS(t, townTalkFiles())}
	f.Shop = NewShop(0)
	a := tavernInteriorApp(t, f)
	s := f.townUI
	if err := a.HeadlessActivate("NPC 9"); err != nil {
		t.Fatal(err)
	}
	if s.room != roomTalk {
		t.Fatalf("NPC 9 opened room %v, want a conversation", s.room)
	}
	for n := 0; s.room == roomTalk && n < 8; n++ {
		if err := a.HeadlessActivate("dialogue"); err != nil {
			t.Fatal(err)
		}
		if got := a.HeadlessMessage(); got != "" {
			t.Fatalf("press %d of the conversation showed %q", n+1, got)
		}
	}
	if s.room != roomTavern {
		t.Fatalf("the conversation returned to room %v, want the tavern", s.room)
	}
	s.tavernSelection = tavernCandidateKey{kind: tavernCandidateMercenary, id: 3}
	if err := a.HeadlessActivate(s.TownSurface().Buttons[tavernButtonHire].Label); err != nil {
		t.Fatal(err)
	}
	if !f.Town.MercenaryHired(3) {
		t.Fatal("the hire press did not hire the squad")
	}
	if got := a.HeadlessMessage(); got != "" {
		t.Fatalf("a hire showed %q", got)
	}
	if err := a.HeadlessActivate(s.TownSurface().Buttons[tavernButtonExit].Label); err != nil {
		t.Fatal(err)
	}
	if s.room != roomSquare {
		t.Fatalf("the tavern's Exit left room %v, want the square", s.room)
	}
	if got := a.HeadlessMessage(); got != "" {
		t.Errorf("leaving the tavern showed %q", got)
	}
}
