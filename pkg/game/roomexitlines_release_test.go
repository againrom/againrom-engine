package game

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/pkg/ui"
)

// roomExitReader reads the line the town shows after each input App receives.
type roomExitReader struct {
	t   *testing.T
	app *ui.App
}

// quiet reads the town line and requires it empty.
func (r roomExitReader) quiet(step string) {
	r.t.Helper()
	got := r.app.HeadlessMessage()
	r.t.Logf("%-38s line %q", step, got)
	if got != "" {
		r.t.Errorf("%s showed the line %q", step, got)
	}
}

// posts reads a control: a press the original also answers with a line, which
// proves this route can show one.
func (r roomExitReader) posts(step string) {
	r.t.Helper()
	got := r.app.HeadlessMessage()
	r.t.Logf("%-38s line %q (control)", step, got)
	if got == "" {
		r.t.Fatalf("%s showed no line, so this route cannot show one", step)
	}
}

// finish presses OK, or Escape, until the open conversation ends, reading the
// line after every press.
func (r roomExitReader) finish(s *townScreen, what string, escape bool) {
	r.t.Helper()
	if s.room != roomTalk {
		r.t.Fatalf("%s: no conversation is open, room %d", what, s.room)
	}
	for n := 1; s.room == roomTalk && n <= 64; n++ {
		var err error
		if escape {
			err = r.app.HeadlessKey("escape")
		} else {
			err = r.app.HeadlessActivate("dialogue")
		}
		if err != nil {
			r.t.Fatal(err)
		}
		r.quiet(fmt.Sprintf("%s press %d", what, n))
	}
	if s.room == roomTalk {
		r.t.Fatalf("%s did not end", what)
	}
}

// roomExitApp opens the installed town at chapter (0 is the first town)
// through App's own load and town input.
func roomExitApp(t *testing.T, f *FrontEnd, chapter int) (*ui.App, *townScreen) {
	t.Helper()
	f.SoundPlayer, f.MusicPlayer, f.AmbientPlayer, f.CutsceneAudioPlayer = nil, nil, nil, nil
	if chapter != 0 {
		town := NewTown(f.Campaign.Value())
		for mission := range f.Campaign.Value().Chapters {
			if mission < chapter {
				town.Won(mission)
			}
		}
		if got := town.Chapter(); got != chapter {
			t.Fatalf("prepared chapter = %d, want %d", got, chapter)
		}
		f.Town = town
	}
	f.Carried = f.NextParty()
	f.arriveInTown()
	snap, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := EncodeSave(snap, label)
	if err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	if _, err := store.Write(time.Unix(200, 0), payload); err != nil {
		t.Fatal(err)
	}
	app := f.App("town room exits")
	app.Layout(640, 480)
	app.SetSaveSeams(agsSaveSeams(f, store, OriginalStore{}, nil))
	for _, target := range []string{"load game", "@first"} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenTown {
		t.Fatalf("the save opened screen %s, want the town", app.Screen())
	}
	return app, f.TownScreen().(*townScreen)
}

// roomExitEnter presses one door of the square and settles the room.
func roomExitEnter(t *testing.T, app *ui.App, s *townScreen, door string, room townRoom) {
	t.Helper()
	if err := app.HeadlessActivate(door); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if s.room != room && s.room != roomTalk {
		t.Fatalf("%s opened room %d, want %d or a conversation", door, s.room, room)
	}
}

// roomExitWalk leaves every room of the town at the front end's chapter and ends
// every conversation it holds, through App input, reading the line after each
// press. It returns how many conversations of each kind it ended.
func roomExitWalk(t *testing.T, f *FrontEnd, app *ui.App, s *townScreen) (shop, tavern, mercenary, school int) {
	t.Helper()
	r := roomExitReader{t, app}
	tap := func(surface string, index int) {
		t.Helper()
		x, y, err := app.HeadlessShopPoint(surface, index)
		if err != nil {
			t.Fatal(err)
		}
		for _, edge := range []string{"press", "release"} {
			if err := app.HeadlessPointer(edge, x, y); err != nil {
				t.Fatal(err)
			}
		}
	}

	// The shop's quest conversation opens on entry and hands its mission over
	// then; a press on the merchant shows no line, and the Exit button leaves.
	offers := f.Town.Offers(TownShop)
	roomExitEnter(t, app, s, "SHOP", roomShop)
	if s.room == roomTalk {
		if len(offers) == 0 || !slices.Contains(f.Town.Available(), offers[0].Mission) {
			t.Fatalf("opening the shop conversation left available %v, want the offered mission %v", f.Town.Available(), offers)
		}
		r.finish(s, "shop conversation", false)
		shop++
	}
	if s.room != roomShop {
		t.Fatalf("the shop conversation returned to room %d", s.room)
	}
	closeShopTip(t, app, s)
	tap("merchant", 0)
	r.quiet("shop merchant press")
	tap("button", 3)
	r.quiet("shop Exit button")
	if s.room != roomSquare {
		t.Fatalf("the shop's Exit button left room %d", s.room)
	}

	// The tavern: every NPC with a conversation, alternately ended by OK and by
	// Escape, then the first mercenary the chapter offers, then a hire as a control
	// and the Exit button. The inn commits nothing while the player is in it: the
	// missions its conversations queue reach the gates when the Exit button leaves.
	npcs := f.Town.Offers(TownTavern)
	roomExitEnter(t, app, s, "TAVERN", roomTavern)
	var queued []int
	for i, offer := range npcs {
		held := slices.Contains(f.Town.Available(), offer.Mission)
		if err := app.HeadlessActivate(fmt.Sprintf("NPC %d", offer.NPC)); err != nil {
			t.Fatal(err)
		}
		if s.room != roomTalk {
			t.Logf("NPC %d has no conversation in this chapter", offer.NPC)
			continue
		}
		r.finish(s, fmt.Sprintf("NPC %d conversation", offer.NPC), i%2 == 1)
		tavern++
		if offer.Mission > 0 && !held {
			if slices.Contains(f.Town.Available(), offer.Mission) {
				t.Fatalf("ending NPC %d's conversation put mission %d at the gates inside the tavern", offer.NPC, offer.Mission)
			}
			queued = append(queued, offer.Mission)
		}
	}
	for _, cell := range s.TownSurface().Cells {
		if !strings.HasPrefix(cell.Semantic, "Mercenary ") {
			continue
		}
		if err := app.HeadlessActivate(cell.Semantic); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessActivate(s.TownSurface().Buttons[tavernButtonTalk].Label); err != nil {
			t.Fatal(err)
		}
		if s.room != roomTalk {
			t.Fatalf("Talk on %s opened room %d", cell.Semantic, s.room)
		}
		t.Logf("%s conversation: source %d, diagnostic %q", cell.Semantic, s.mercenaryTalk, s.talkDiagnostic)
		r.finish(s, cell.Semantic+" conversation", false)
		mercenary++
		break
	}
	if s.room != roomTavern {
		t.Fatalf("the conversations returned to room %d, want the tavern", s.room)
	}
	if hireControl(t, f, app, s) {
		r.posts("tavern hire")
	} else {
		t.Log("the chapter lists no mercenary squad, so no tavern press shows a line")
	}
	if err := app.HeadlessActivate(s.TownSurface().Buttons[tavernButtonExit].Label); err != nil {
		t.Fatal(err)
	}
	r.quiet("tavern Exit button")
	if s.room != roomSquare {
		t.Fatalf("the tavern's Exit button left room %d", s.room)
	}
	for _, mission := range queued {
		if !slices.Contains(f.Town.Available(), mission) {
			t.Fatalf("leaving the tavern left available %v, want the queued mission %d", f.Town.Available(), mission)
		}
	}

	// The school: its quest conversation opens on entry, and the Exit button
	// leaves.
	offers = f.Town.Offers(TownSchool)
	roomExitEnter(t, app, s, "SCHOOL", roomSchool)
	if s.room == roomTalk {
		if len(offers) == 0 || !slices.Contains(f.Town.Available(), offers[0].Mission) {
			t.Fatalf("opening the school conversation left available %v, want the offered mission %v", f.Town.Available(), offers)
		}
		r.finish(s, "school conversation", false)
		school++
	}
	if s.room != roomSchool {
		t.Fatalf("the school conversation returned to room %d", s.room)
	}
	if err := app.HeadlessActivate(s.TownSurface().Buttons[1].Label); err != nil {
		t.Fatal(err)
	}
	r.quiet("school Exit button")
	if s.room != roomSquare {
		t.Fatalf("the school's Exit button left room %d", s.room)
	}

	// The gates: the world map's town card leaves.
	roomExitEnter(t, app, s, "GATES", roomGates)
	x, y, err := app.HeadlessWorldMapTownPoint()
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"press", "release"} {
		if err := app.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	r.quiet("gates town card")
	if s.room != roomSquare {
		t.Fatalf("the gates' town card left room %d", s.room)
	}

	// Escape leaves each room as well.
	for _, door := range []struct {
		name string
		room townRoom
	}{{"SHOP", roomShop}, {"TAVERN", roomTavern}, {"SCHOOL", roomSchool}, {"GATES", roomGates}} {
		roomExitEnter(t, app, s, door.name, door.room)
		if s.room != door.room {
			t.Fatalf("%s opened a conversation after its offer was taken", door.name)
		}
		if err := app.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
		r.quiet("Escape from " + door.name)
		if s.room != roomSquare {
			t.Fatalf("Escape left room %d standing in %s", s.room, door.name)
		}
	}
	return shop, tavern, mercenary, school
}

// TestReleaseTownRoomExitsAndConversationEndsPostNoLine leaves every room of
// the installed town and ends every kind of conversation through App input,
// reading the town line after each press: the original prints nothing there
// (SHOP-SCREEN-035, DLG-LIFE-005, MISSION-MSGPOST-058). The tavern's hire
// press, which shows a line, proves the route can show one.
func TestReleaseTownRoomExitsAndConversationEndsPostNoLine(t *testing.T) {
	t.Run("first town", func(t *testing.T) {
		f := releaseFront(t)
		app, s := roomExitApp(t, f, 0)
		shop, tavern, _, _ := roomExitWalk(t, f, app, s)
		if shop == 0 || tavern == 0 {
			t.Fatalf("the first town held %d shop and %d tavern conversations, want both kinds", shop, tavern)
		}
	})

	t.Run("chapter with a school offer", func(t *testing.T) {
		f := releaseFront(t)
		c := f.Campaign.Value()
		chapter := 0
		for _, target := range c.Main {
			candidate := NewTown(c)
			for mission := range c.Chapters {
				if mission < target {
					candidate.Won(mission)
				}
			}
			candidate.Arrive()
			if candidate.Chapter() == target && len(candidate.Offers(TownSchool)) > 0 {
				chapter = target
				break
			}
		}
		if chapter == 0 {
			t.Fatal("no shipped chapter offers a school mission")
		}
		app, s := roomExitApp(t, f, chapter)
		if _, _, _, school := roomExitWalk(t, f, app, s); school == 0 {
			t.Fatalf("chapter %d held no school conversation", chapter)
		}
	})

	t.Run("chapter with a mercenary", func(t *testing.T) {
		f := releaseFront(t)
		chapter := releaseMercenaryChapter(t, f.Campaign.Value())
		app, s := roomExitApp(t, f, chapter)
		if _, _, mercenary, _ := roomExitWalk(t, f, app, s); mercenary == 0 {
			t.Fatalf("chapter %d held no mercenary conversation", chapter)
		}
	})

	t.Run("map dialogue", func(t *testing.T) {
		f := releaseFront(t)
		f.SoundPlayer, f.MusicPlayer, f.AmbientPlayer, f.CutsceneAudioPlayer = nil, nil, nil, nil
		f.SetDeterministicFrames(true)
		app := f.App("map dialogue end")
		app.Layout(640, 480)
		if err := app.OpenMission(f.MissionOpenerWith(10, f.NextParty())); err != nil {
			t.Fatal(err)
		}
		r := roomExitReader{t, app}
		for i := 0; i < 1500; i++ {
			if _, kind, open := f.LiveNotice(); open && kind == ui.NoticeDialogue {
				break
			}
			if err := app.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
		}
		texts := func() []string {
			var out []string
			for _, line := range f.live.view.MessageLines() {
				out = append(out, line.Text)
			}
			return out
		}
		pressed := 0
		for ; pressed < 16; pressed++ {
			if _, kind, open := f.LiveNotice(); !open || kind != ui.NoticeDialogue {
				break
			}
			before := texts()
			if err := app.HeadlessActivate("notice"); err != nil {
				t.Fatal(err)
			}
			r.quiet(fmt.Sprintf("map dialogue press %d", pressed+1))
			for _, line := range texts() {
				if !slices.Contains(before, line) {
					t.Errorf("map dialogue press %d put %q on the message line", pressed+1, line)
				}
			}
		}
		if pressed == 0 {
			t.Fatal("mission 10 raised no dialogue notice within 1500 ticks")
		}
		t.Logf("ended %d map dialogue press(es); message line holds %q", pressed, texts())
	})
}

// hireControl hires the first squad the tavern lists, with a purse that covers
// it, and reports whether it did. A chapter that lists no squad is given the
// installed type-14 squad first. The hire posts a line, which proves the route
// can show one.
func hireControl(t *testing.T, f *FrontEnd, app *ui.App, s *townScreen) bool {
	t.Helper()
	const typ = 14
	if len(s.tavernMercenaries()) == 0 {
		chapter := f.Town.camp.Chapters[f.Town.Chapter()]
		chapter.Mercenaries = []int{typ}
		f.Town.camp.Chapters[f.Town.Chapter()] = chapter
		f.Town.mercEnabled[typ], f.Town.mercPool[typ] = true, 2
	}
	for _, cell := range s.TownSurface().Cells {
		if !strings.HasPrefix(cell.Semantic, "Mercenary ") {
			continue
		}
		f.Town.gold = 1 << 24
		if err := app.HeadlessActivate(cell.Semantic); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessActivate(s.TownSurface().Buttons[tavernButtonHire].Label); err != nil {
			t.Fatal(err)
		}
		return true
	}
	return false
}
