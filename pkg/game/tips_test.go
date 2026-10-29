package game

import (
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/ui"
)

func tipTownFixture(t *testing.T, files []synth.File) *FrontEnd {
	t.Helper()
	c := townCampaign(t)
	return &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil), Table: shopTable(), Archives: &Archives{Containers: townTextFS(t, files)}}, CampaignSession: CampaignSession{Town: NewTown(c)}}
}

// withTipTable swaps in a changed copy of a tip table; callers are serial.
func withTipTable(t *testing.T, name string, change func(*ui.TipTable)) {
	t.Helper()
	old := tipTables[name]
	changed := *old
	changed.Rows = append([]ui.TipRow(nil), old.Rows...)
	change(&changed)
	tipTables[name] = &changed
	t.Cleanup(func() { tipTables[name] = old })
}

func tipRowNamed(table *ui.TipTable, name string) *ui.TipRow {
	for i := range table.Rows {
		if table.Rows[i].Name == name {
			return &table.Rows[i]
		}
	}
	return nil
}

func TestTownScreenConstructionBuildsTheRulesRoom(t *testing.T) {
	f := tipTownFixture(t, []synth.File{{Path: "text/tips/town.txt", Data: []byte("the shop centre")}})
	s := f.TownScreen().(*townScreen)
	if s.tip.room != "square" || s.tip.text != "the shop centre" || !s.tipShown("square") {
		t.Fatalf("popup after construction = %+v, want the install's own town.txt over the square", s.tip)
	}
}

func TestEachEntryBuildsItsRoomsPopup(t *testing.T) {
	f := tipTownFixture(t, []synth.File{
		{Path: "text/tips/inn.txt", Data: []byte("tavern tip")},
		{Path: "text/tips/training.txt", Data: []byte("school tip")},
	})
	s := f.TownScreen().(*townScreen)
	s.Choose(0) // tavern
	if s.tip.room != "tavern" || s.tip.text != "tavern tip" || s.TownSurface().Tip.Text != "tavern tip" {
		t.Fatalf("tavern popup = %+v", s.tip)
	}
	s.Back()
	s.Choose(2) // school
	if s.tip.room != "school" || s.tip.text != "school tip" || s.tipShown("tavern") {
		t.Fatalf("school popup = %+v", s.tip)
	}
}

func TestEntryWithNoShippedTextShowsNothing(t *testing.T) {
	f := tipTownFixture(t, nil)
	s := f.TownScreen().(*townScreen)
	s.Choose(0) // tavern, nothing shipped
	if s.tip.room != "tavern" || s.tip.text != "" {
		t.Fatalf("popup with no shipped inn.txt = %+v, want an empty tavern popup", s.tip)
	}
	if v := s.roomTipView("tavern"); v != (ui.TipPanelView{}) {
		t.Fatalf("view with no shipped text = %+v, want none", v)
	}
}

func TestBuildWithTipsOffNeverReadsTheInstall(t *testing.T) {
	f := tipTownFixture(t, []synth.File{{Path: "text/tips/town.txt", Data: []byte("would show")}})
	f.tipsOff = true
	s := f.TownScreen().(*townScreen)
	if s.tip.text != "" {
		t.Fatalf("popup text with TipsOff = %q, want empty even though the install ships the text", s.tip.text)
	}
}

func TestCloseTipHoldsUntilTheNextEntry(t *testing.T) {
	f := tipTownFixture(t, []synth.File{
		{Path: "text/tips/town.txt", Data: []byte("square")},
		{Path: "text/tips/inn.txt", Data: []byte("tavern")},
	})
	s := f.TownScreen().(*townScreen)
	s.Choose(0) // tavern
	s.CloseTip()
	if s.tipShown("tavern") || s.roomTipView("tavern") != (ui.TipPanelView{}) {
		t.Fatal("a closed popup still shows")
	}
	s.Back()
	if !s.tipShown("square") || s.tip.closed {
		t.Fatalf("the square's entry did not build its popup: %+v", s.tip)
	}
	s.Choose(0)
	if !s.tipShown("tavern") {
		t.Fatal("the tavern's next entry did not build its popup again")
	}
}

func TestToggleTipsFlipsSuppressionNotTheOpenPanel(t *testing.T) {
	f := tipTownFixture(t, []synth.File{{Path: "text/tips/town.txt", Data: []byte("square")}})
	s := f.TownScreen().(*townScreen)
	if !s.roomTipView("square").ToggleOn {
		t.Fatal("ToggleOn before ToggleTips = false, want true (default tips shown)")
	}
	s.ToggleTips()
	if !f.TipsOff() {
		t.Fatal("ToggleTips did not set the front end's own TipsOff")
	}
	if v := s.roomTipView("square"); v.ToggleOn || v.Text != "square" {
		t.Fatalf("after ToggleTips: ToggleOn %v text %q, want false and the unchanged text", v.ToggleOn, v.Text)
	}
}

func TestTipRoomsConstructNamesARoomRow(t *testing.T) {
	row := `{"name": "a", "event": "room-enter", "room": "square", "text": {"file": "f"}, "rect": [0, 0, 1, 1], "once": "entry", "pointer": "town"}`
	for construct, ok := range map[string]bool{"": true, "square": true, "inn": false} {
		_, err := ui.DecodeTipTable([]byte(`{"tips": "t", "art": "a", "rooms": {"construct": "` + construct + `"}, "rows": [` + row + `]}`))
		if (err == nil) != ok {
			t.Fatalf("construct %q: err %v", construct, err)
		}
	}
}

func TestNilTownScreenTipMethodsDoNotPanic(t *testing.T) {
	var s *townScreen
	s.CloseTip()
	s.ToggleTips()
	s.ResumeTown()
	s.rereadTip("square")
	s.advanceShopSecondTip()
	if s.TownSquareEntryRevision() != 0 || s.tipShown("square") {
		t.Fatal("a nil screen reports a popup")
	}
}

func TestTownTipRereadKeepsTheBuild(t *testing.T) {
	for _, room := range []string{"square", "shop", "school", "tavern"} {
		f := tipTownFixture(t, []synth.File{
			{Path: "text/tips/town.txt", Data: []byte("initial")},
			{Path: "text/tips/shop1.txt", Data: []byte("initial")},
			{Path: "text/tips/training.txt", Data: []byte("initial")},
			{Path: "text/tips/inn.txt", Data: []byte("initial")},
		})
		s := f.TownScreen().(*townScreen)
		s.room = townRoomNames[room]
		s.loadRoomTip(room)
		addr := s.roomTipRow(room).Text.File
		if s.tip.text != "initial" {
			t.Fatalf("room %s text %q lacks initial content", room, addr)
		}
		revision := s.tip.revision
		s.CloseTip()
		f.Archives.Containers = townTextFS(t, []synth.File{{Path: addr[len(mainPrefix):], Data: []byte("replacement")}})
		s.rereadTip(room)
		if s.tip.text != "replacement" || !s.tip.closed || s.tip.revision != revision {
			t.Fatal("content replacement renewed popup lifetime")
		}
		f.tipsOff = true
		s.rereadTip(room)
		if s.tip.text != "replacement" || !s.tip.closed || s.tip.revision != revision {
			t.Fatal("global option changed existing popup content or lifetime")
		}
		s.loadRoomTip(room)
		if s.tip.text != "" || s.tip.closed {
			t.Fatal("suppressed build did not clear content/dismissal")
		}
		revision = s.tip.revision
		f.tipsOff = false
		s.rereadTip(room)
		if s.tip.text != "" || s.tip.revision != revision {
			t.Fatal("content re-read created a previously absent popup")
		}
		s.loadRoomTip(room)
		if s.tip.text != "replacement" || s.tip.revision != revision+1 {
			t.Fatal("room build did not reload allowed popup")
		}
		s.rereadTip("another room")
		if s.tip.text != "replacement" || s.tip.room != room {
			t.Fatal("a reread for another room touched the popup")
		}
	}
}

// secondTipTown is a second-game campaign started in town 1 with its square,
// inn and shop tip sections shipped.
func secondTipTown(t *testing.T) (*FrontEnd, *townScreen) {
	t.Helper()
	f, app, screen := secondCampaignFixtureFiles(t, false, synth.File{Path: "text/town.txt",
		Data: []byte("#tips1\r\nsquare tip\r\n#tips3\r\nshop tip\r\n#tips4\r\nshop second\r\n")})
	if err := app.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	// The fixture ships no town art, so the square's shown entry is this
	// build.
	screen.loadRoomTip("square")
	if !screen.tipShown("square") {
		t.Fatalf("the second game's square entry built no popup: %+v", screen.tip)
	}
	return f, screen
}

// Resume in the rules decides whether the game menu's return reopens a
// closed popup, in both games.
func TestResumeRuleDecidesTheMenuReturn(t *testing.T) {
	t.Run("first game as shipped", func(t *testing.T) {
		s := tipTownFixture(t, []synth.File{{Path: "text/tips/town.txt", Data: []byte("square")}}).TownScreen().(*townScreen)
		s.CloseTip()
		s.ResumeTown()
		if !s.tipShown("square") {
			t.Fatal("the menu's return did not reopen the closed square popup")
		}
	})
	t.Run("first game without resume", func(t *testing.T) {
		withTipTable(t, "rom1", func(tt *ui.TipTable) { tt.Rooms.Resume = false })
		s := tipTownFixture(t, []synth.File{{Path: "text/tips/town.txt", Data: []byte("square")}}).TownScreen().(*townScreen)
		revision := s.tip.revision
		s.CloseTip()
		s.ResumeTown()
		if s.tipShown("square") || s.tip.revision != revision {
			t.Fatal("the menu's return rebuilt the popup though the rules have no resume")
		}
	})
	t.Run("second game as shipped", func(t *testing.T) {
		_, s := secondTipTown(t)
		revision := s.tip.revision
		s.CloseTip()
		s.ResumeTown()
		if s.tipShown("square") || s.tip.revision != revision {
			t.Fatal("the menu's return rebuilt the second game's popup (DIV-4063)")
		}
	})
	t.Run("second game with resume", func(t *testing.T) {
		withTipTable(t, "rom2", func(tt *ui.TipTable) { tt.Rooms.Resume = true })
		_, s := secondTipTown(t)
		s.CloseTip()
		s.ResumeTown()
		if !s.tipShown("square") || strings.TrimSpace(s.tip.text) != "square tip" {
			t.Fatalf("the rules' resume did not reopen the second game's popup: %+v", s.tip)
		}
	})
}

// EntryPaint in the rules decides whether a build moves the square's entry
// revision, in both games.
func TestEntryPaintRulePublishesTheRevision(t *testing.T) {
	s := tipTownFixture(t, nil).TownScreen().(*townScreen)
	if s.TownSquareEntryRevision() == 0 || s.TownSquareEntryRevision() != s.tip.revision {
		t.Fatal("the first game's square entry revision does not follow its popup")
	}
	_, second := secondTipTown(t)
	if second.TownSquareEntryRevision() != 0 {
		t.Fatal("the second game publishes a square entry revision")
	}
	withTipTable(t, "rom1", func(tt *ui.TipTable) { tt.Rooms.EntryPaint = false })
	if s.TownSquareEntryRevision() != 0 {
		t.Fatal("the first game without entry paint still publishes a revision")
	}
}

// The shop row's second text replaces the first once per build while the
// shop's table holds an item; which game has one is its row data.
func TestSecondTipRowDecidesTheRetext(t *testing.T) {
	item := func(s *townScreen) {
		shop := &Shop{}
		shop.table = []ShopPlace{{}}
		s.sess.Shop = shop
	}
	t.Run("first game as shipped", func(t *testing.T) {
		s := tipTownFixture(t, []synth.File{{Path: "text/tips/shop1.txt", Data: []byte("one")}, {Path: "text/tips/shop2.txt", Data: []byte("two")}}).TownScreen().(*townScreen)
		s.Choose(1) // shop
		s.advanceShopSecondTip()
		if s.tip.text != "one" {
			t.Fatal("the second text replaced the first with an empty table")
		}
		item(s)
		s.advanceShopSecondTip()
		f := s.tip
		s.rereadTip("shop")
		s.advanceShopSecondTip()
		if f.text != "two" || !f.second || s.tip.text != "one" {
			t.Fatalf("second text %q latch %v, after a reread %q; want two, set, one and no second retext", f.text, f.second, s.tip.text)
		}
		s.Back()
		s.Choose(1)
		if s.tip.second || s.tip.text != "one" {
			t.Fatal("the shop's next entry kept the second text")
		}
	})
	t.Run("first game without a second row", func(t *testing.T) {
		withTipTable(t, "rom1", func(tt *ui.TipTable) { tipRowNamed(tt, "shop").Second = ui.TipText{} })
		s := tipTownFixture(t, []synth.File{{Path: "text/tips/shop1.txt", Data: []byte("one")}, {Path: "text/tips/shop2.txt", Data: []byte("two")}}).TownScreen().(*townScreen)
		s.Choose(1)
		item(s)
		s.advanceShopSecondTip()
		if s.tip.text != "one" || s.tip.second {
			t.Fatal("a shop row without a second text retexted the popup")
		}
	})
	t.Run("second game as shipped", func(t *testing.T) {
		_, s := secondTipTown(t)
		s.loadRoomTip("shop")
		item(s)
		s.advanceShopSecondTip()
		if strings.TrimSpace(s.tip.text) != "shop tip" || s.tip.second {
			t.Fatalf("the second game's shop popup took a second text: %+v (DIV-2968)", s.tip)
		}
	})
	t.Run("second game with a second row", func(t *testing.T) {
		withTipTable(t, "rom2", func(tt *ui.TipTable) {
			tipRowNamed(tt, "shop").Second = ui.TipText{File: "main/text/town.txt", Section: "tips4"}
		})
		_, s := secondTipTown(t)
		s.loadRoomTip("shop")
		item(s)
		s.advanceShopSecondTip()
		if strings.TrimSpace(s.tip.text) != "shop second" || !s.tip.second {
			t.Fatalf("a second row did not retext the second game's shop popup: %+v", s.tip)
		}
	})
}

// A reset keeps the build the square's entry just made and forgets the
// campaign it belonged to, so a second-game popup waits for its arrival.
func TestResetRoomTipForgetsTheCampaign(t *testing.T) {
	_, s := secondTipTown(t)
	s.CloseTip()
	revision := s.tip.revision
	s.resetRoomTip()
	if s.tip.of != nil || s.tip.closed || s.tip.revision != revision || s.tipShown("square") {
		t.Fatalf("reset popup %+v", s.tip)
	}
	s.arriveRoomTip(true, "square")
	if !s.tipShown("square") || s.tip.revision != revision+1 {
		t.Fatalf("arrival did not build the popup again: %+v", s.tip)
	}
}
