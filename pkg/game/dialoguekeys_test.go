package game

import (
	"fmt"
	"image"
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/ui"
)

// Enter, Escape and a click on the dialogue's button are one command in the
// original: each turns the page and the last page closes the window, and Space
// is bound to nothing. The window has no accept or decline state, so which key
// ended a conversation decides nothing (DLG-KEYS-040). The shop's and the
// school's quest is registered as the window opens; the inn only queues it and
// registers it when the player leaves (DLG-LIFE-005).

// dialogueKeysPages is the page count of every conversation below.
const dialogueKeysPages = 3

// dialogueKeysFiles gives the chapter 30 shop and inn conversations and the
// chapter 40 school conversation of the town fixture three pages each.
func dialogueKeysFiles() []synth.File {
	pages := func(who string) []byte {
		return []byte(fmt.Sprintf("<part=1>\r\n%s one\r\n<part=2>\r\n%s two\r\n<part=3>\r\n%s three", who, who, who))
	}
	return []synth.File{
		{Path: "text/inn/npc/npc22m30.txt", Data: pages("The innkeeper")},
		{Path: "text/shop/npc31m31.txt", Data: pages("The merchant")},
		{Path: "text/training/npc34m41.txt", Data: pages("The trainer")},
	}
}

// dialogueKeysApp is the fixture town at chapter 30, or at chapter 40 once its
// main mission is won, behind App's own input.
func dialogueKeysApp(t *testing.T, chapter int) (*FrontEnd, *ui.App, *townScreen) {
	t.Helper()
	c := townCampaign(t)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil), Font: resolved(missionFont(), nil), Archives: &Archives{Containers: townTextFS(t, dialogueKeysFiles())}}, CampaignSession: CampaignSession{Town: NewTown(c)}}
	f.Town.Arrive()
	if chapter == 40 {
		f.Town.Won(30)
	}
	s := f.TownScreen().(*townScreen)
	app := f.App("dialogue keys")
	app.Layout(640, 480)
	app.SetSaveSeams(nil, func() []ui.SaveEntry { return []ui.SaveEntry{{Name: "town.ags", Label: "Town"}} }, func(string) (ui.MapOpener, bool, error) { s.resetForNewGame(); return nil, true, nil })
	if err := app.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("@first"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenTown {
		t.Fatalf("the load route opened screen %v, want the town", app.Screen())
	}
	s.CloseTip()
	return f, app, s
}

// dialogueButtonPoint is the middle of the open dialogue's button in window
// pixels. At the 640x480 layout the window and the frame are the same pixels.
func dialogueButtonPoint(t *testing.T, s *townScreen) image.Point {
	t.Helper()
	pic, open := s.TownDialogue()
	button, shown := s.TownDialogueButton()
	if !open || pic == nil || !shown {
		t.Fatal("no dialogue button is shown")
	}
	r := button.Add(image.Pt((640-pic.Bounds().Dx())/2, (480-pic.Bounds().Dy())/2))
	return image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
}

// dialogueKeyPress sends one Enter, Escape or Space edge through App, or one
// press and release on the open dialogue's button.
func dialogueKeyPress(t *testing.T, app *ui.App, s *townScreen, key string) {
	t.Helper()
	switch key {
	case "enter", "escape", "space":
		if err := app.HeadlessKey(key); err != nil {
			t.Fatal(err)
		}
	case "click":
		p := dialogueButtonPoint(t, s)
		for _, edge := range []string{"press", "release"} {
			if err := app.HeadlessPointer(edge, p.X, p.Y); err != nil {
				t.Fatal(err)
			}
		}
	default:
		t.Fatalf("no key %q", key)
	}
}

// dialogueMoment is everything a player could tell apart after one press: the
// room, the page, whether the window is up, the missions at the gates and the
// line the town posted.
func dialogueMoment(f *FrontEnd, app *ui.App, s *townScreen) string {
	_, open := s.TownDialogue()
	return fmt.Sprintf("room %d, page %d, window %v, gates %v, line %q", s.room, s.said, open, f.Town.Available(), app.HeadlessMessage())
}

// TestDialogueKeysDoOneThingOnEveryPage presses Enter, Escape and the button on
// every page of a shop, a school and an inn quest conversation through App
// input. Space is pressed before each of them and changes nothing. Every key
// leaves the same state behind on every page, the quest is at the gates when
// the shop's or the school's window opens, and the inn's quest is there only
// once the player has left the inn.
func TestDialogueKeysDoOneThingOnEveryPage(t *testing.T) {
	for _, q := range []struct {
		name    string
		chapter int
		door    string
		room    townRoom
		mission int
		inn     bool
	}{
		{"shop", 30, "SHOP", roomShop, 31, false},
		{"school", 40, "SCHOOL", roomSchool, 41, false},
		{"inn", 30, "TAVERN", roomTavern, 30, true},
	} {
		traces := map[string][]string{}
		for _, key := range []string{"enter", "escape", "click"} {
			t.Run(q.name+"/"+key, func(t *testing.T) {
				f, app, s := dialogueKeysApp(t, q.chapter)
				before := fmt.Sprint(f.Town.Available())
				if err := app.HeadlessActivate(q.door); err != nil {
					t.Fatal(err)
				}
				if q.inn {
					if s.room != roomTavern {
						t.Fatalf("the inn door opened room %d", s.room)
					}
					if err := app.HeadlessActivate("NPC 22"); err != nil {
						t.Fatal(err)
					}
				}
				if s.room != roomTalk || s.said != 1 {
					t.Fatalf("the conversation is not open on its first page: room %d, page %d", s.room, s.said)
				}
				switch got := fmt.Sprint(f.Town.Available()); {
				case q.inn && got != before:
					t.Fatalf("the inn put %s at the gates when its conversation opened, want %s until the player leaves", got, before)
				case !q.inn && !containsMission(f.Town.Available(), q.mission):
					t.Fatalf("the window opened with %s at the gates, want mission %d registered as it opens", got, q.mission)
				}
				trace := []string{dialogueMoment(f, app, s)}
				for page := 1; page <= dialogueKeysPages; page++ {
					dialogueKeyPress(t, app, s, "space")
					if got := dialogueMoment(f, app, s); got != trace[len(trace)-1] {
						t.Fatalf("Space on page %d changed %q to %q", page, trace[len(trace)-1], got)
					}
					dialogueKeyPress(t, app, s, key)
					if page < dialogueKeysPages && (s.room != roomTalk || s.said != page+1) {
						t.Fatalf("%s on page %d left room %d on page %d, want page %d", key, page, s.room, s.said, page+1)
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
				if got := f.Town.Available(); !containsMission(got, q.mission) {
					t.Fatalf("after the conversation the gates hold %v, want mission %d", got, q.mission)
				}
				if got := app.HeadlessMessage(); got != "" {
					t.Fatalf("the conversation posted %q, want no line", got)
				}
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

// TestNoDialogueRowAcceptsOrDeclines ends an inn conversation through the
// town's row dispatch by every row a caller could press past its last page. The
// window has one button, so no row is an accept or a decline: each closes the
// conversation, and the mission it queued is at the gates once the inn is left.
func TestNoDialogueRowAcceptsOrDeclines(t *testing.T) {
	_, probe := roomExitFront(t, 30)
	tavern := -1
	for i, d := range townDoors {
		if d.room == roomTavern {
			tavern = i
		}
	}
	probe.Choose(tavern)
	probe.Choose(0)
	lines := len(probe.townLines())
	if probe.room != roomTalk || lines < 2 {
		t.Fatalf("setup: room %d with %d pages", probe.room, lines)
	}
	for _, row := range []int{0, lines - 1, lines, lines + 1} {
		t.Run(fmt.Sprintf("row %d", row), func(t *testing.T) {
			f, s := roomExitFront(t, 30)
			s.Choose(tavern)
			s.Choose(0)
			for n := 0; n < lines-1; n++ {
				if act := s.Choose(0); act.Msg != "" {
					t.Fatalf("page %d posted %q", n+2, act.Msg)
				}
			}
			if s.room != roomTalk || s.said != lines {
				t.Fatalf("room %d on page %d, want the last page %d", s.room, s.said, lines)
			}
			if act := s.Choose(row); act.Msg != "" {
				t.Fatalf("row %d posted %q", row, act.Msg)
			}
			if s.room != roomTavern {
				t.Fatalf("row %d on the last page left room %d, want the inn", row, s.room)
			}
			if !s.Back() || s.room != roomSquare {
				t.Fatalf("leaving the inn reached room %d", s.room)
			}
			if got := f.Town.Available(); !reflect.DeepEqual(got, []int{30}) {
				t.Fatalf("after row %d the gates hold %v, want [30]", row, got)
			}
		})
	}
}
