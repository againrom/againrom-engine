package game

import (
	"fmt"
	"image"
	"reflect"
	"testing"

	"againrom/pkg/ui"
)

// schoolOfferChapter is a shipped chapter whose school holds a mission.
func schoolOfferChapter(t *testing.T, f *FrontEnd) int {
	t.Helper()
	c := f.Campaign.Value()
	for _, target := range c.Main {
		candidate := NewTown(c)
		for mission := range c.Chapters {
			if mission < target {
				candidate.Won(mission)
			}
		}
		candidate.Arrive()
		if candidate.Chapter() == target && len(candidate.Offers(TownSchool)) > 0 {
			return target
		}
	}
	t.Fatal("no shipped chapter offers a school mission")
	return 0
}

// TestReleaseDialogueKeysOnEveryPage presses Space, and then Enter, Escape or a
// click on the button, on every page of the installed shop, school and inn
// quest conversations through App input. Space changes nothing on any page.
// Each of the other three turns the page and the last page closes the window,
// and they leave one and the same state behind on every page. The shop's and
// the school's quest is at the gates when their window opens; the inn's is
// there only once the player has left the inn (DLG-KEYS-040, DLG-LIFE-005).
func TestReleaseDialogueKeysOnEveryPage(t *testing.T) {
	for _, q := range []struct {
		name string
		door string
		room townRoom
		inn  bool
	}{
		{"shop", "SHOP", roomShop, false},
		{"school", "SCHOOL", roomSchool, false},
		{"inn", "TAVERN", roomTavern, true},
	} {
		traces := map[string][]string{}
		for _, key := range []string{"enter", "escape", "click"} {
			t.Run(q.name+"/"+key, func(t *testing.T) {
				f := releaseFront(t)
				chapter := 0
				if q.room == roomSchool {
					chapter = schoolOfferChapter(t, f)
				}
				app, s := roomExitApp(t, f, chapter)
				s.CloseTip()

				building := map[townRoom]TownBuilding{roomShop: TownShop, roomSchool: TownSchool, roomTavern: TownTavern}[q.room]
				var offer TownOffer
				for _, o := range f.Town.Offers(building) {
					if o.Mission > 0 {
						offer = o
						break
					}
				}
				if offer.Mission == 0 {
					t.Fatalf("chapter %d's %s holds no mission to hand over", f.Town.Chapter(), q.name)
				}
				before := fmt.Sprint(f.Town.Available())
				roomExitEnter(t, app, s, q.door, q.room)
				if q.inn {
					if err := app.HeadlessActivate(fmt.Sprintf("NPC %d", offer.NPC)); err != nil {
						t.Fatal(err)
					}
				}
				if s.room != roomTalk || s.said != 1 {
					t.Fatalf("the %s conversation is not open on its first page: room %d, page %d", q.name, s.room, s.said)
				}
				pages := len(s.townLines())
				if pages == 0 {
					t.Fatalf("the %s conversation has no page", q.name)
				}
				switch got := fmt.Sprint(f.Town.Available()); {
				case q.inn && got != before:
					t.Fatalf("the inn put %s at the gates when its conversation opened, want %s until the player leaves", got, before)
				case !q.inn && !containsMission(f.Town.Available(), offer.Mission):
					t.Fatalf("the window opened with %s at the gates, want mission %d registered as it opens", got, offer.Mission)
				}

				trace := []string{dialogueMoment(f, app, s)}
				for page := 1; page <= pages; page++ {
					dialogueKeyPress(t, app, s, "space")
					if got := dialogueMoment(f, app, s); got != trace[len(trace)-1] {
						t.Fatalf("Space on page %d of %d changed %q to %q", page, pages, trace[len(trace)-1], got)
					}
					dialogueKeyPress(t, app, s, key)
					if page < pages && (s.room != roomTalk || s.said != page+1) {
						t.Fatalf("%s on page %d of %d left room %d on page %d, want page %d", key, page, pages, s.room, s.said, page+1)
					}
					trace = append(trace, dialogueMoment(f, app, s))
				}
				if s.room != q.room {
					t.Fatalf("the last %s left room %d, want the room behind the window, %d", key, s.room, q.room)
				}
				if q.inn {
					if got := fmt.Sprint(f.Town.Available()); got != before {
						t.Fatalf("the ended conversation put %s at the gates inside the inn, want %s", got, before)
					}
					if err := app.HeadlessKey("escape"); err != nil {
						t.Fatal(err)
					}
					if s.room != roomSquare {
						t.Fatalf("Escape in the inn left room %d, want the square", s.room)
					}
					trace = append(trace, dialogueMoment(f, app, s))
				}
				if got := f.Town.Available(); !containsMission(got, offer.Mission) {
					t.Fatalf("after the conversation the gates hold %v, want mission %d", got, offer.Mission)
				}
				if got := app.HeadlessMessage(); got != "" {
					t.Fatalf("the conversation posted %q, want no line", got)
				}
				t.Logf("%s, %s: %d pages, mission %d, NPC %d; gates before %s, after %v", q.name, key, pages, offer.Mission, offer.NPC, before, f.Town.Available())
				traces[key] = trace
			})
		}
		for _, key := range []string{"escape", "click"} {
			if !reflect.DeepEqual(traces[key], traces["enter"]) {
				t.Errorf("%s: %s left\n%q\nbut Enter left\n%q", q.name, key, traces[key], traces["enter"])
			}
		}
	}
}

// escMenuMoment is what a click could change on the installed square: the
// screen, the room, whether a line is open over it, the missions at the gates
// and the line the town posted.
func escMenuMoment(f *FrontEnd, app *ui.App, s *townScreen) string {
	return fmt.Sprintf("screen %v, room %d, line open %v, gates %v, posted %q", app.Screen(), s.room, gateDialogueOpen(s), f.Town.Available(), app.HeadlessMessage())
}

// TestReleaseAClickOutsideTheEscMenuReachesNothing opens the Esc menu on the
// installed square and presses and releases both buttons on every door and on
// the statue outside the panel, and over a grid of the frame, through App
// input. Nothing behind the panel answers: the screen, the room, the missions
// at the gates and the line stay as they were. With the menu closed the same
// door presses do act, so the silence is the menu's (MENU-INPUT-016).
func TestReleaseAClickOutsideTheEscMenuReachesNothing(t *testing.T) {
	f := releaseFront(t)
	app, s := roomExitApp(t, f, 0)
	s.CloseTip()

	// The decoded town panel (MENU-ESC-010).
	panel := image.Rect(100, 100, 440, 340)
	mask := f.TownSquareArt.Value().Mask
	firstDoor := map[int]image.Point{}
	var controls []image.Point
	for y := 0; y < mask.Bounds().Dy(); y += 3 {
		for x := 0; x < mask.Bounds().Dx(); x += 3 {
			p := image.Pt(x, y)
			c, ok := ui.TownSquareControlAt(mask, p)
			if !ok || p.In(panel) {
				continue
			}
			controls = append(controls, p)
			if _, seen := firstDoor[int(c.Door)]; c.Kind == ui.TownSquareControlDoor && !seen {
				firstDoor[int(c.Door)] = p
			}
		}
	}
	if len(firstDoor) == 0 {
		t.Fatal("no door of the square lies outside the town's Esc panel")
	}
	press := func(p image.Point, buttons ...string) {
		t.Helper()
		for _, edge := range buttons {
			if err := app.HeadlessPointer(edge, p.X, p.Y); err != nil {
				t.Fatal(err)
			}
		}
	}
	both := []string{"press", "release", "right-press", "right-release"}
	acting := 0

	// Control: with the menu closed each of those doors acts. Entering the shop
	// or the school hands its mission over, so the baseline is read again once
	// the square is back.
	for door := 0; door < 4; door++ {
		p, ok := firstDoor[door]
		if !ok {
			continue
		}
		before := escMenuMoment(f, app, s)
		press(p, "press", "release")
		if got := escMenuMoment(f, app, s); got == before {
			t.Fatalf("with the menu closed a press on door %d at %v changed nothing", door, p)
		}
		for n := 0; n < 16 && (s.room != roomSquare || gateDialogueOpen(s) || s.AtWorldMap()); n++ {
			if err := app.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
		}
		if s.room != roomSquare || gateDialogueOpen(s) || app.Screen() != ui.ScreenTown {
			t.Fatalf("Escape did not bring door %d back to the square: room %d, screen %v", door, s.room, app.Screen())
		}
		acting++
	}
	square := escMenuMoment(f, app, s)

	// The menu up: every control pixel outside the panel and a grid of the frame.
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenGameMenu {
		t.Fatalf("Escape on the square landed on screen %v, want the menu", app.Screen())
	}
	menu := escMenuMoment(f, app, s)
	for _, p := range controls {
		press(p, both...)
	}
	for y := 4; y < 480; y += 8 {
		for x := 4; x < 640; x += 8 {
			if p := image.Pt(x, y); !p.In(panel) {
				press(p, both...)
			}
		}
	}
	if got := escMenuMoment(f, app, s); got != menu {
		t.Fatalf("clicks outside the panel changed %q to %q", menu, got)
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if got := escMenuMoment(f, app, s); got != square {
		t.Fatalf("after closing the menu the square reads %q, want %q", got, square)
	}
	t.Logf("%d control pixels and a %d-point grid outside the panel, both buttons: nothing behind the menu answered; %d doors act with the menu closed", len(controls), (640/8)*(480/8), acting)
}
