package game

import (
	"image"
	"testing"

	"againrom/internal/synth"
)

// The tip widget's own load, dismissal and permanent suppression for the
// four rooms besides the shop (1018 spec behaviours 2, 3, 4; DIV-160..164).
// shoptip_test.go already covers the shop's own load and re-read.

func tipTownFixture(t *testing.T, files []synth.File) *FrontEnd {
	t.Helper()
	c := townCampaign(t)
	return &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil), Table: shopTable(), Archives: &Archives{Containers: townTextFS(t, files)}}, CampaignSession: CampaignSession{Town: NewTown(c)}}
}

// TownScreen's own construction loads the square's tip (spec behaviour 2):
// the screen opens at roomSquare and this is that room's own entry.
func TestTownScreenConstructionLoadsTheSquareTip(t *testing.T) {
	f := tipTownFixture(t, []synth.File{{Path: "text/tips/town.txt", Data: []byte("the shop centre")}})
	s := f.TownScreen().(*townScreen)
	if s.townTip != "the shop centre" {
		t.Fatalf("townTip after construction = %q, want the install's own town.txt", s.townTip)
	}
}

// Choosing a door loads that room's own tip, one node per room, keyed by
// townDoors' own order (0 tavern, 1 shop, 2 school, 3 gates; shoptip_test.go's
// own comment).
func TestChooseLoadsEachRoomsOwnTip(t *testing.T) {
	f := tipTownFixture(t, []synth.File{
		{Path: "text/tips/inn.txt", Data: []byte("tavern tip")},
		{Path: "text/tips/training.txt", Data: []byte("school tip")},
	})
	s := f.TownScreen().(*townScreen)

	s.Choose(0) // tavern
	if s.tavernTip != "tavern tip" {
		t.Fatalf("tavernTip = %q, want %q", s.tavernTip, "tavern tip")
	}
	s.Back()
	s.Choose(2) // school
	if s.schoolTip != "school tip" {
		t.Fatalf("schoolTip = %q, want %q", s.schoolTip, "school tip")
	}
}

// A room whose node the install does not ship loads no text, and tipView
// answers a zero (non-Showing) panel for it (spec behaviour 2, "a screen
// whose node the install doesn't ship draws no panel").
func TestChooseWithNoShippedNodeLeavesTipEmpty(t *testing.T) {
	f := tipTownFixture(t, nil)
	s := f.TownScreen().(*townScreen)
	s.Choose(0) // tavern, nothing shipped
	if s.tavernTip != "" {
		t.Fatalf("tavernTip with no shipped inn.txt = %q, want empty", s.tavernTip)
	}
	v := s.tipView(roomTavern, s.tavernTip, ui1018TestRect)
	if v.Showing() {
		t.Fatal("tipView with no shipped text reports Showing() = true")
	}
}

// The permanent suppression store gates the READ itself, not only the
// panel's own draw (TOWN-186's own gate-tested-at-construction reading): a
// suppressed front end never opens the node, so a room whose node is
// missing and a suppressed room reach the same empty tip through different
// paths.
func TestLoadTipWithTipsOffNeverReadsTheInstall(t *testing.T) {
	f := tipTownFixture(t, []synth.File{{Path: "text/tips/town.txt", Data: []byte("would show")}})
	f.tipsOff = true
	s := f.TownScreen().(*townScreen)
	if s.townTip != "" {
		t.Fatalf("townTip with TipsOff = %q, want empty even though the install ships the node", s.townTip)
	}
}

// CloseTip dismisses only the room that is open when it is called, for the
// rest of the session (spec behaviour 3): a different room's own tip is
// unaffected.
func TestCloseTipDismissesOnlyTheOpenRoom(t *testing.T) {
	f := tipTownFixture(t, []synth.File{
		{Path: "text/tips/town.txt", Data: []byte("square")},
		{Path: "text/tips/inn.txt", Data: []byte("tavern")},
	})
	s := f.TownScreen().(*townScreen)
	s.Choose(0) // tavern
	s.CloseTip()
	if !s.tipClosed[roomTavern] {
		t.Fatal("CloseTip while in the tavern did not close roomTavern")
	}
	if s.tipClosed[roomSquare] {
		t.Fatal("CloseTip while in the tavern closed roomSquare too")
	}
	if v := s.tipView(roomTavern, s.tavernTip, ui1018TestRect); v.Showing() {
		t.Fatal("tipView for a closed room reports Showing() = true")
	}
	// The untouched room's own text still resolves (Art is a separate
	// concern with its own coverage, tipart_test.go; the fixture here ships
	// no BMP/spr256 nodes, so tipArt() answers nil and Showing() alone would
	// read false regardless of tipClosed).
	if v := s.tipView(roomSquare, s.townTip, ui1018TestRect); v.Text != "square" {
		t.Fatalf("tipView for an untouched room: Text = %q, want %q", v.Text, "square")
	}
}

// ToggleTips flips the front end's own permanent suppression; a room's
// already-loaded text is untouched (TOWN-186's own construction-time gate:
// toggling does not retroactively hide an open panel), but the toggle's own
// drawn state (tipView's ToggleOn) reflects the flip immediately.
func TestToggleTipsFlipsSuppressionNotTheOpenPanel(t *testing.T) {
	f := tipTownFixture(t, []synth.File{{Path: "text/tips/town.txt", Data: []byte("square")}})
	s := f.TownScreen().(*townScreen)
	before := s.tipView(roomSquare, s.townTip, ui1018TestRect)
	if !before.ToggleOn {
		t.Fatal("ToggleOn before ToggleTips = false, want true (default tips shown)")
	}
	s.ToggleTips()
	if !f.TipsOff() {
		t.Fatal("ToggleTips did not set the front end's own TipsOff")
	}
	after := s.tipView(roomSquare, s.townTip, ui1018TestRect)
	if after.ToggleOn {
		t.Fatal("ToggleOn after ToggleTips = true, want false")
	}
	if s.townTip != "square" {
		t.Fatalf("townTip changed by ToggleTips alone: %q, want %q unchanged", s.townTip, "square")
	}
}

// A nil townScreen's CloseTip and ToggleTips do not panic — TipScreen is
// satisfied by *townScreen, which flow.closeTip/toggleTips reach through a
// live interface value that is never nil in production, but a defensive nil
// receiver is this package's own convention throughout townscreen.go.
func TestNilTownScreenTipMethodsDoNotPanic(t *testing.T) {
	var s *townScreen
	s.CloseTip()
	s.ToggleTips()
}

// ui1018TestRect stands in for one of tippanel.go's own exported room rects;
// this file tests tipView's own state machine (closed, suppressed, missing
// text), not which literal rectangle each room is handed — the room view
// builders in townscreen.go and townshell.go pass the real ones.
var ui1018TestRect = image.Rect(0, 0, 312, 200)

func TestTownTipContentReadKeepsConstruction(t *testing.T) {
	for _, room := range []townRoom{roomSquare, roomShop, roomSchool, roomTavern} {
		f := tipTownFixture(t, []synth.File{
			{Path: "text/tips/town.txt", Data: []byte("initial")},
			{Path: "text/tips/shop1.txt", Data: []byte("initial")},
			{Path: "text/tips/training.txt", Data: []byte("initial")},
			{Path: "text/tips/inn.txt", Data: []byte("initial")},
		})
		s := f.TownScreen().(*townScreen)
		s.room = room
		s.loadTip(room)
		dst, addr := s.tipTextSource(room)
		if *dst != "initial" {
			t.Fatalf("room %d source %q lacks initial content", room, addr)
		}
		revision := s.tipRevision
		s.CloseTip()
		f.Archives.Containers = townTextFS(t, []synth.File{{Path: addr[len(mainPrefix):], Data: []byte("replacement")}})
		s.rereadTip(room)
		if *dst != "replacement" || !s.tipClosed[room] || s.tipRevision != revision {
			t.Fatal("content replacement renewed popup lifetime")
		}
		f.tipsOff = true
		s.rereadTip(room)
		if *dst != "replacement" || !s.tipClosed[room] || s.tipRevision != revision {
			t.Fatal("global option changed existing popup content or lifetime")
		}
		s.loadTip(room)
		if *dst != "" || s.tipClosed[room] {
			t.Fatal("suppressed construction did not clear content/dismissal")
		}
		revision = s.tipRevision
		f.tipsOff = false
		s.rereadTip(room)
		if *dst != "" || s.tipRevision != revision {
			t.Fatal("content re-read created a previously absent popup")
		}
		s.loadTip(room)
		if *dst != "replacement" || s.tipRevision != revision+1 {
			t.Fatal("room construction did not reload allowed popup")
		}
	}
}
