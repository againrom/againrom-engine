package game

import (
	"image"
	"image/color"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/town"
	"againrom/pkg/ui"
)

type entryPaintTrace struct {
	*townScreen
	events []string
	paints int
	points int
}

func (s *entryPaintTrace) AdvanceTownSquareAnimation() {
	s.events = append(s.events, "paint")
	s.paints++
	s.townScreen.AdvanceTownSquareAnimation()
}

func (s *entryPaintTrace) TownSquarePointer(p image.Point) {
	s.events = append(s.events, "pointer")
	s.points++
	s.townScreen.TownSquarePointer(p)
}

func entryPaintArt() *town.Art {
	frames := func(n int) []image.Image {
		out := make([]image.Image, n)
		for i := range out {
			pic := image.NewRGBA(image.Rect(0, 0, 1, 1))
			pic.SetRGBA(0, 0, color.RGBA{R: byte(i + 1), A: 255})
			out[i] = pic
		}
		return out
	}
	return squareArt(image.NewPaletted(image.Rect(0, 0, 640, 480), make(color.Palette, 256)), map[string][]image.Image{
		"base":  {image.NewRGBA(image.Rect(0, 0, 640, 480))},
		"guard": frames(8), "door": frames(9), "sign": frames(10), "fluger": frames(8),
	})
}

type entryPaintFixture struct {
	front *FrontEnd
	app   *ui.App
	town  *entryPaintTrace
	now   time.Time
	rolls int
}

func newEntryPaintFixture(t *testing.T) *entryPaintFixture {
	t.Helper()
	x := &entryPaintFixture{front: shellFrontEnd(), now: time.Unix(100, 0)}
	x.front.TownSquareArt = resolved(entryPaintArt(), nil)
	x.front.TownTavernArt = resolved(&ui.TownTavernArt{}, nil)
	x.front.TownAnimationNow = func() time.Time { return x.now }
	x.front.TownAnimationRandom = func(n int) int {
		if n != 100 {
			t.Fatalf("unexpected exterior random population %d", n)
		}
		x.rolls++
		return 0
	}
	x.front.TownAmbientRandom = func(int) int { return 0 }
	x.app = x.front.App("town-entry-order-witness")
	x.app.Layout(640, 480)
	x.town = &entryPaintTrace{townScreen: x.front.townUI}
	x.app.SetTown(x.town)
	x.app.SetSaveSeams(nil,
		func() []ui.SaveEntry { return []ui.SaveEntry{{Name: "town.sav", Label: "Town"}} },
		func(string) (ui.MapOpener, bool, error) {
			x.front.townUI.resetForNewGame()
			return nil, true, nil
		})
	return x
}

func (x *entryPaintFixture) enter(t *testing.T) {
	t.Helper()
	if err := x.app.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	if err := x.app.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if x.app.Screen() != ui.ScreenTown || !x.town.AtTownSquare() {
		t.Fatalf("load route did not reach square: %s", x.app.Screen())
	}
	x.town.CloseTip()
}

func (x *entryPaintFixture) paint(t *testing.T) {
	t.Helper()
	pix, note, err := x.app.HeadlessFrame()
	if err != nil || note != "" || pix == nil {
		t.Fatalf("CPU production composition: %q, %v, image nil=%v", note, err, pix == nil)
	}
}

func TestTownEntryPaintPrecedesNextPointer(t *testing.T) {
	for _, route := range []string{"fresh-load", "tavern-return"} {
		t.Run(route, func(t *testing.T) {
			x := newEntryPaintFixture(t)
			x.enter(t)
			if route == "tavern-return" {
				x.paint(t)
				if err := x.app.HeadlessActivate("Tavern"); err != nil {
					t.Fatal(err)
				}
				if x.town.room != roomTavern {
					t.Fatal("real town model did not enter tavern")
				}
				x.town.events, x.town.paints, x.town.points = nil, 0, 0
				if err := x.app.HeadlessKey("escape"); err != nil {
					t.Fatal(err)
				}
				if !x.town.AtTownSquare() || x.app.Screen() != ui.ScreenTown {
					t.Fatal("Escape did not return to square")
				}
			}
			if x.town.points != 0 {
				t.Fatalf("entry delivered %d unsolicited pointers", x.town.points)
			}
			if x.town.sqGuard().Dir != 0 || x.town.sqGuard().Frame != 7 {
				t.Fatalf("fresh entry changed idle guard: frame=%d step=%d", x.town.sqGuard().Frame, x.town.sqGuard().Dir)
			}
			if x.town.paints != 1 {
				t.Errorf("entry returned with %d synchronous paint calls; want 1 before next explicit pointer", x.town.paints)
			}
			if err := x.app.HeadlessPointer("hover", 2, 2); err != nil {
				t.Fatal(err)
			}
			if x.town.points != 1 || x.town.sqGuard().Dir != 1 {
				t.Fatal("explicit next pointer did not reach real guard selection")
			}
			if !reflect.DeepEqual(x.town.events, []string{"paint", "pointer"}) {
				t.Errorf("entry/next-pointer order %v; want [paint pointer]", x.town.events)
			}
			t.Logf("route=%s frame=%d step=%d events=%v rolls=%d", route, x.town.sqGuard().Frame, x.town.sqGuard().Dir, x.town.events, x.rolls)
		})
	}
}

func TestTownEntryPaintAdmissionControls(t *testing.T) {
	t.Run("fresh-and-strict-due-step-zero-then-pointer", func(t *testing.T) {
		x := newEntryPaintFixture(t)
		x.enter(t)
		x.town.events, x.town.paints, x.town.points = nil, 0, 0
		x.paint(t)
		if x.rolls != 0 || x.town.sqGuard().Dir != 0 || x.town.sqGuard().Frame != 7 {
			t.Fatalf("first actual paint admitted hub or guard movement: rolls=%d frame=%d step=%d", x.rolls, x.town.sqGuard().Frame, x.town.sqGuard().Dir)
		}
		if x.town.paints != 1 || x.town.points != 0 || x.town.sqPaintLast() != x.now {
			t.Fatal("first actual paint/lifecycle fixture is not initialized")
		}
		x.town.sqGuard().Frame = 4
		x.now = x.now.Add(67 * time.Millisecond)
		x.paint(t)
		if x.rolls != 0 || x.town.sqGuard().Frame != 4 {
			t.Fatal("67ms admitted hub")
		}
		x.now = x.now.Add(time.Millisecond)
		x.paint(t)
		if x.rolls != 2 || x.town.sqGuard().Frame != 4 || x.town.sqGuard().Dir != 0 {
			t.Fatalf("68ms must admit hub while step0 holds guard: rolls=%d frame=%d step=%d", x.rolls, x.town.sqGuard().Frame, x.town.sqGuard().Dir)
		}
		x.paint(t)
		if x.rolls != 2 || x.town.sqGuard().Frame != 4 {
			t.Fatal("same-time duplicate paint admitted another hub")
		}
		if err := x.app.HeadlessPointer("hover", 2, 2); err != nil {
			t.Fatal(err)
		}
		if x.town.sqGuard().Dir != 1 || x.town.sqGuard().Frame != 4 {
			t.Fatal("next pointer must arm, not paint, the guard")
		}
		x.now = x.now.Add(68 * time.Millisecond)
		x.paint(t)
		if x.rolls != 4 || x.town.sqGuard().Frame != 5 || x.town.points != 1 {
			t.Fatalf("explicit pointer followed by due paint missing: rolls=%d frame=%d points=%d", x.rolls, x.town.sqGuard().Frame, x.town.points)
		}
		t.Logf("fresh0, not-due67, due68/step0, duplicate0, explicit-pointer1, due68/frame5; events=%v", x.town.events)
	})
	t.Run("inactive-loss-control", func(t *testing.T) {
		x := newEntryPaintFixture(t)
		x.enter(t)
		if err := x.app.HeadlessFocus(false); err != nil {
			t.Fatal(err)
		}
		x.town.sqGuard().Frame, x.town.sqGuard().Dir = 4, 0
		x.now = x.now.Add(time.Second)
		x.paint(t)
		if x.town.squareView().Active() || x.rolls != 0 || x.town.sqGuard().Frame != 4 {
			t.Fatal("inactive paint admitted hub")
		}
	})
	t.Run("missing-exterior-loss-control", func(t *testing.T) {
		x := newEntryPaintFixture(t)
		x.enter(t)
		x.front.TownSquareArt = resolved[*town.Art](nil, nil)
		x.town.sqGuard().Frame, x.town.sqGuard().Dir = 4, 0
		x.now = x.now.Add(time.Second)
		x.paint(t)
		if x.rolls != 0 || x.town.sqGuard().Frame != 4 {
			t.Fatal("missing exterior paint admitted hub")
		}
	})
}

func TestTownEntryProcessTimerAndBlockedAdmission(t *testing.T) {
	for _, tc := range []struct {
		name    string
		dt      time.Duration
		blocked bool
		want    int
	}{
		{"not-due", 67 * time.Millisecond, false, 0},
		{"due", 68 * time.Millisecond, false, 2},
		{"blocked-due", time.Second, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			x := newEntryPaintFixture(t)
			x.enter(t)
			last := x.front.townUI.sqPaintLast()
			if err := x.app.HeadlessActivate("Tavern"); err != nil {
				t.Fatal(err)
			}
			x.now = x.now.Add(tc.dt)
			x.app.SetTownPaintAdmission(func() bool { return !tc.blocked })
			x.town.events, x.town.paints, x.town.points = nil, 0, 0
			if err := x.app.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if x.rolls != tc.want || x.town.sqGuard().Dir != 0 || x.town.sqGuard().Frame != 7 || x.town.points != 0 {
				t.Fatalf("entry admission rolls=%d guard=%d step=%d points=%d", x.rolls, x.town.sqGuard().Frame, x.town.sqGuard().Dir, x.town.points)
			}
			if tc.want == 0 && x.front.townUI.sqPaintLast() != last {
				t.Fatal("non-admitted entry reset process timer")
			}
			if tc.blocked {
				x.paint(t)
				if x.rolls != 0 {
					t.Fatal("blocked ordinary paint advanced hub")
				}
				x.app.SetTownPaintAdmission(nil)
				x.paint(t)
				if x.rolls != 2 {
					t.Fatal("unblocked due paint lost process admission")
				}
			}
			last = x.front.townUI.sqPaintLast()
			if err := x.app.HeadlessKey("f3"); err != nil {
				t.Fatal(err)
			}
			if err := x.app.HeadlessKey("enter"); err != nil {
				t.Fatal(err)
			}
			if x.front.townUI.sqPaintLast() != last {
				t.Fatal("LOAD reset process timer")
			}
		})
	}
}

func TestTownEntryBlockedFirstPaintDoesNotInitializeTimer(t *testing.T) {
	x := newEntryPaintFixture(t)
	x.app.SetTownPaintAdmission(func() bool { return false })
	x.enter(t)
	if !x.front.townUI.sqPaintLast().IsZero() || x.town.paints != 0 {
		t.Fatal("blocked entry initialized process paint")
	}
	x.now = x.now.Add(time.Hour)
	x.app.SetTownPaintAdmission(nil)
	x.paint(t)
	if x.rolls != 0 || x.front.townUI.sqPaintLast() != x.now {
		t.Fatal("first admitted paint ran hub instead of initializing timer")
	}
}
