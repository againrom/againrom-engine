package game

import (
	"image"
	"testing"
	"time"

	"againrom/pkg/mapload"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// The town room trace extends the square trace through each room page: the
// tavern, the shop, the school and the world map. Each room is entered from
// the square, its centre is watched long enough for every page clock to run,
// every page control is hovered and the harmless ones clicked, the page is
// paused and resumed, and the room is left. Beside the square trace's frame,
// sound, music, room and SAV records, each tick names the room entry and exit
// step hooks the town ran. The recorded lines live in testdata/townroomtrace.

// traceStepHooks wraps every hook a room or the square names in its entry or
// exit steps so the trace records each call by name.
func traceStepHooks(t *testing.T, tr *townTrace) {
	t.Helper()
	names := map[string]bool{}
	for _, s := range ROM1TownDescription().Square.Enter {
		names[s.Hook] = true
	}
	for _, r := range ROM1TownDescription().Rooms {
		for _, s := range append(append([]town.Step{}, r.Enter...), r.Exit...) {
			names[s.Hook] = true
		}
	}
	saved := map[string]func(*townScreen, townRoom){}
	for name := range names {
		hook := townHooks[name]
		if hook == nil {
			continue
		}
		saved[name] = hook
		name := name
		townHooks[name] = func(s *townScreen, room townRoom) {
			tr.sound.log = append(tr.sound.log, "hook "+name+"@"+townRoomName(room))
			hook(s, room)
		}
	}
	t.Cleanup(func() {
		for name, hook := range saved {
			townHooks[name] = hook
		}
	})
}

func centre(r image.Rectangle) image.Point {
	return image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
}

// pause drops the window focus for n ticks and takes it back.
func (tr *townTrace) pause(n int) {
	if err := tr.a.HeadlessFocus(false); err != nil {
		tr.t.Fatal(err)
	}
	tr.record("unfocus")
	for i := 0; i < n; i++ {
		*tr.now = tr.now.Add(97 * time.Millisecond)
		if err := tr.a.HeadlessStep(); err != nil {
			tr.t.Fatal(err)
		}
		tr.record("paused")
	}
	if err := tr.a.HeadlessFocus(true); err != nil {
		tr.t.Fatal(err)
	}
	tr.record("focus")
}

// shopPoint is the shop's own hit point for one control.
func (tr *townTrace) shopPoint(kind string, i int) image.Point {
	x, y, err := tr.a.HeadlessShopPoint(kind, i)
	if err != nil {
		tr.t.Fatal(err)
	}
	return image.Pt(x, y)
}

func TestReleaseTownRoomTraceIsUnchanged(t *testing.T) {
	tr, points, off := startTownTrace(t, func(f *FrontEnd) {
		fighter := f.ChargenParty(ui.ChargenResult{Name: "Trace fighter", Choices: []int{0, 0, 0}, Stats: []int{30, 30, 20, 20}})
		mage := f.ChargenParty(ui.ChargenResult{Name: "Trace mage", Choices: []int{1, 1, 0}, Stats: []int{20, 20, 30, 30}})
		if len(fighter) == 0 || len(mage) == 0 || fighter[0].Mage || !mage[0].Mage {
			t.Fatal("installed chargen did not produce both school classes")
		}
		mage[0].StartingHero = false
		f.Carried = mapload.OwnParty([]mapload.PartyMember{fighter[0], mage[0]})
	})
	traceStepHooks(t, tr)
	f := tr.f
	f.Town.gold = 100000
	tr.action("gold")
	idle := image.Pt(320, 120)
	prev := centre(ui.CharacterPaneCornerRect(ui.TownCharacterRegion, ui.CharacterPanePrev))
	next := centre(ui.CharacterPaneCornerRect(ui.TownCharacterRegion, ui.CharacterPaneNext))

	// Tavern.
	tr.hover(points[0x80][0], 34*time.Millisecond, 3)
	tr.click(points[0x80][0])
	tr.expect("surface")
	tr.closeDialogue()
	tr.hover(idle, 41*time.Millisecond, 160)
	for i := 0; i < 4; i++ {
		tr.hover(centre(ui.TownSurfaceButtonRect(ui.TownSurfaceTavern, i)), 37*time.Millisecond, 6)
	}
	for i := 0; i < 6; i++ {
		cell := image.Pt(176+48*i+24, 480-32)
		tr.hover(cell, 37*time.Millisecond, 5)
		tr.click(cell)
		tr.closeDialogue()
		tr.hover(cell, 43*time.Millisecond, 12)
	}
	tr.hover(next, 37*time.Millisecond, 4)
	tr.click(next)
	tr.hover(idle, 47*time.Millisecond, 120)
	tr.pause(25)
	tr.hover(idle, 47*time.Millisecond, 240)
	tr.key("escape")
	tr.closeDialogue()
	tr.expect("square")
	tr.hover(off, 41*time.Millisecond, 20)

	// Shop.
	tr.hover(points[0x90][0], 34*time.Millisecond, 3)
	tr.click(points[0x90][0])
	tr.expect("shop")
	tr.closeDialogue()
	tr.hover(idle, 41*time.Millisecond, 120)
	for i := 3; i >= 0; i-- {
		p := tr.shopPoint("shelf_pick", i)
		tr.hover(p, 37*time.Millisecond, 4)
		tr.click(p)
		tr.closeDialogue()
		tr.hover(p, 43*time.Millisecond, 30)
	}
	tr.hover(tr.shopPoint("merchant", 0), 41*time.Millisecond, 10)
	for i := 0; i < 4; i++ {
		tr.hover(tr.shopPoint("button", i), 37*time.Millisecond, 4)
	}
	tr.hover(idle, 41*time.Millisecond, 260)
	cell := tr.shopPoint("shelf", 0)
	tr.hover(cell, 37*time.Millisecond, 4)
	tr.click(cell)
	tr.closeDialogue()
	tr.hover(cell, 37*time.Millisecond, 20)
	buy := tr.shopPoint("button", 1)
	tr.hover(buy, 37*time.Millisecond, 4)
	tr.click(buy)
	tr.closeDialogue()
	tr.hover(buy, 37*time.Millisecond, 30)
	undo := tr.shopPoint("button", 0)
	tr.click(undo)
	tr.closeDialogue()
	tr.hover(idle, 43*time.Millisecond, 40)
	tr.pause(25)
	tr.hover(idle, 43*time.Millisecond, 160)
	tr.key("escape")
	tr.closeDialogue()
	tr.expect("square")
	tr.hover(off, 41*time.Millisecond, 20)

	// School.
	tr.hover(points[0xc0][0], 34*time.Millisecond, 3)
	tr.click(points[0xc0][0])
	tr.expect("surface")
	tr.closeDialogue()
	tr.hover(idle, 41*time.Millisecond, 160)
	for class := 0; class < 2; class++ {
		for slot := 0; slot < 5; slot++ {
			tr.hover(centre(ui.SchoolSkillRect(class, slot)), 37*time.Millisecond, 4)
		}
	}
	for _, picker := range []image.Point{next, next, prev} {
		tr.hover(picker, 37*time.Millisecond, 3)
		tr.click(picker)
		tr.hover(idle, 41*time.Millisecond, 70)
	}
	for slot := 0; slot < 5; slot++ {
		for class := 0; class < 2; class++ {
			p := centre(ui.SchoolSkillRect(class, slot))
			tr.click(p)
			tr.closeDialogue()
			tr.hover(p, 41*time.Millisecond, 4)
		}
	}
	train := centre(ui.TownSurfaceButtonRect(ui.TownSurfaceSchool, 0))
	tr.hover(train, 37*time.Millisecond, 4)
	tr.click(train)
	tr.closeDialogue()
	tr.hover(train, 37*time.Millisecond, 40)
	tr.hover(idle, 53*time.Millisecond, 200)
	tr.pause(25)
	tr.hover(idle, 53*time.Millisecond, 160)
	exit := centre(ui.TownSurfaceButtonRect(ui.TownSurfaceSchool, 1))
	tr.hover(exit, 37*time.Millisecond, 4)
	tr.click(exit)
	tr.closeDialogue()
	tr.expect("square")
	tr.hover(off, 41*time.Millisecond, 20)

	// World map.
	if f.Town.gateMission() == -1 {
		f.Town.announceMission(f.Town.currentMain())
		tr.action("announce")
	}
	tr.hover(points[0xa0][1], 34*time.Millisecond, 30)
	tr.click(points[0xa0][1])
	tr.expect("map")
	tr.hover(idle, 70*time.Millisecond, 30)
	tr.pause(10)
	tr.hover(idle, 70*time.Millisecond, 20)
	tr.key("escape")
	tr.expect("square")
	tr.hover(off, 34*time.Millisecond, 20)
	tr.compare("townroomtrace")
}
