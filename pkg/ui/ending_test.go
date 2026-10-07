package ui

import (
	"bytes"
	"image"
	"testing"
	"time"

	"againrom/pkg/render/text"
	"github.com/hajimehoshi/ebiten/v2"
)

func endingTestApp(t *testing.T) *App {
	a := newTestApp(t, nil, nil)
	a.SetWords(AuthoredWords(), chargenTestFont(), nil)
	a.SetCampaignEnding(func() (EndingView, bool) {
		return EndingView{Hero: "Finisher", Gold: 345, Credits: []string{"Installed author", "Installed role"}, Hall: []EndingHallRow{{Name: "First", Score: -7}, {Name: "Second", Score: 90}}, HallAvailable: true}, true
	})
	if !a.flow.showCampaignEnding() {
		t.Fatal("ending did not open")
	}
	return a
}

func TestComposeScreenSelectsEndingComposer(t *testing.T) {
	a := endingTestApp(t)
	for page := 0; page < 3; page++ {
		a.flow.endingPage = page
		want, err := a.composeEndingScreen()
		if err != nil {
			t.Fatal(err)
		}
		got, err := a.composeScreen()
		if err != nil || !bytes.Equal(want.Pix, got.Pix) {
			t.Fatal("missing composer", page, err)
		}
		a.Draw(ebiten.NewImage(800, 600))
	}
}

func TestEndingControlsAndSourceRowsHaveVisibleGeometry(t *testing.T) {
	a := endingTestApp(t)
	pix, err := a.composeScreen()
	if err != nil {
		t.Fatal(err)
	}
	if pix.RGBAAt(28, 430) != saveFocusColor || pix.RGBAAt(619, 415) != saveFocusColor {
		t.Fatal("missing focus/panel frame")
	}
	a.flow.endingPage = 2
	got, err := a.composeScreen()
	if err != nil {
		t.Fatal(err)
	}
	want := image.NewRGBA(got.Bounds())
	a.flow.menuFont.Draw(want, "1.", 145, 145, gameMenuText)
	a.flow.menuFont.Draw(want, "First", 176, 145, gameMenuText)
	width, _ := a.flow.menuFont.Measure("-7")
	a.flow.menuFont.Draw(want, "-7", 484-width, 145, gameMenuText)
	for y := 145; y < 159; y++ {
		for x := 145; x < 494; x++ {
			if want.RGBAAt(x, y).A != 0 && want.RGBAAt(x, y) != got.RGBAAt(x, y) {
				t.Fatal("source row origin or order changed", x, y)
			}
		}
	}
}

// Without an installed credits roll the ending's text page leads to the hall,
// and the hall's button leaves the ending through the campaign reset
// (FAME-029).
func TestEndingTextCreditsThenHallThenResetAndMenu(t *testing.T) {
	a := endingTestApp(t)
	exits := 0
	a.SetCampaignEndingExit(func() { exits++ })
	if a.Screen() != ScreenEnding || a.flow.endingPage != 1 {
		t.Fatal("ending must start at the credits", a.Screen(), a.flow.endingPage)
	}
	if err := a.HeadlessActivate("Back"); err != nil || a.flow.endingPage != 2 || exits != 0 {
		t.Fatal("credits must lead to the hall", err, a.flow.endingPage, exits)
	}
	if err := a.HeadlessActivate("Back"); err != nil || a.Screen() != ScreenMenu || exits != 1 {
		t.Fatal("hall button must reset the campaign and show the menu", err, a.Screen(), exits)
	}
}

func rollTestApp(t *testing.T, lines int) (*App, *int) {
	a := endingTestApp(t)
	exits := new(int)
	a.SetCampaignEndingExit(func() { *exits++ })
	roll := make([]string, lines)
	for i := range roll {
		roll[i] = "line"
	}
	a.SetCredits(func() CreditsView { return CreditsView{Lines: roll} })
	a.SetCampaignEnding(func() (EndingView, bool) {
		return EndingView{Hall: []EndingHallRow{{Name: "First", Score: 5}}, HallAvailable: true}, true
	})
	if !a.flow.showCampaignEnding() {
		t.Fatal("ending did not open")
	}
	return a, exits
}

func TestEndingOpensCreditsRollThenHallAndKeysEndTheRoll(t *testing.T) {
	for _, press := range []appInput{{AnyKey: true}, {PrimaryPressed: true}, {Enter: true}, {Escape: true}} {
		a, exits := rollTestApp(t, 5)
		if a.Screen() != ScreenCredits {
			t.Fatal("the ending must open the credits roll", a.Screen())
		}
		a.step(press, time.Now())
		if a.Screen() != ScreenEnding || a.flow.endingPage != 2 || *exits != 0 {
			t.Fatal("a press must end the roll into the hall", a.Screen(), a.flow.endingPage, *exits)
		}
	}
}

// The roll ends after 480 plus line count times line pitch steps (FAME-030).
func TestEndingCreditsRollEndsAfterStartPlusLinesTimesPitch(t *testing.T) {
	a, _ := rollTestApp(t, 5)
	pitch := a.creditsPitch()
	end := 480 + 5*pitch
	now := time.Now()
	a.step(appInput{}, now)
	a.media.steps = end - 2
	a.step(appInput{}, now.Add(23*time.Millisecond))
	if a.Screen() != ScreenCredits || a.media.steps != end-1 {
		t.Fatal("roll ended before its last step", a.Screen(), a.media.steps)
	}
	a.step(appInput{}, now.Add(46*time.Millisecond))
	if a.Screen() != ScreenEnding || a.flow.endingPage != 2 {
		t.Fatal("roll did not end at its last step", a.Screen())
	}
}

// One pixel per step, a step only when 23 ms have passed since the last one,
// and the first step waits for nothing (FAME-030).
func TestCreditsRollStepsOnePixelPer23Milliseconds(t *testing.T) {
	a, _ := rollTestApp(t, 5)
	now := time.Unix(900, 0)
	a.step(appInput{}, now)
	if a.media.steps != 1 {
		t.Fatal("the first step must not wait", a.media.steps)
	}
	a.step(appInput{}, now.Add(22*time.Millisecond))
	if a.media.steps != 1 {
		t.Fatal("a step ran before 23 ms", a.media.steps)
	}
	a.step(appInput{}, now.Add(23*time.Millisecond))
	if a.media.steps != 2 {
		t.Fatal("no step after 23 ms", a.media.steps)
	}
	a.step(appInput{}, now.Add(10*time.Second))
	if a.media.steps != 3 {
		t.Fatal("a long gap must still be one pixel", a.media.steps)
	}
}

// The right button is a stub in the credits class (FAME-031).
func TestCreditsRightButtonDoesNotEndTheRoll(t *testing.T) {
	a, _ := rollTestApp(t, 5)
	a.step(appInput{SecondaryPressed: true}, time.Now())
	if a.Screen() != ScreenCredits {
		t.Fatal("the right button ended the roll")
	}
}

// A left-button release inside the hall's rectangle ends the hall whether or
// not the press began there (FAME-029).
func TestEndingHallEndsOnReleaseInsideItsRectangle(t *testing.T) {
	a, exits := rollTestApp(t, 5)
	a.step(appInput{AnyKey: true}, time.Now())
	a.flow.ending.HallBackground = image.NewRGBA(image.Rect(0, 0, 640, 480))
	click := func(p image.Point) {
		x, y, ok := a.nativeFrameToWindow(p)
		if !ok {
			t.Fatal("no window point for", p)
		}
		a.step(appInput{CursorX: x, CursorY: y, PrimaryReleased: true}, time.Now())
	}
	click(image.Pt(0x230-1, 0x1a0+4))
	click(image.Pt(0x25c, 0x1a0+4))
	if a.Screen() != ScreenEnding || *exits != 0 {
		t.Fatal("a release outside the rectangle left the hall", a.Screen())
	}
	click(image.Pt(0x230, 0x1a0))
	if a.Screen() != ScreenMenu || *exits != 1 {
		t.Fatal("a release inside the rectangle did not end the hall", a.Screen(), *exits)
	}
}

func TestEndingHallEndsOnlyOnItsButton(t *testing.T) {
	a, exits := rollTestApp(t, 5)
	a.step(appInput{AnyKey: true}, time.Now())
	a.step(appInput{AnyKey: true, WheelY: 1}, time.Now())
	if a.Screen() != ScreenEnding || *exits != 0 {
		t.Fatal("a wheel or key event left the hall", a.Screen())
	}
	if err := a.HeadlessActivate("Back"); err != nil || a.Screen() != ScreenMenu || *exits != 1 {
		t.Fatal("the button must leave", err, a.Screen(), *exits)
	}
}

func TestEndingMenuHallDoesNotResetCampaign(t *testing.T) {
	a := endingTestApp(t)
	exits := 0
	a.SetCampaignEndingExit(func() { exits++ })
	a.SetHallOfFame(func() EndingView { return EndingView{Hall: []EndingHallRow{{Name: "First"}}, HallAvailable: true} })
	a.flow.showHallOfFame()
	if err := a.HeadlessActivate("Back"); err != nil || a.Screen() != ScreenMenu || exits != 0 {
		t.Fatal("the menu's hall view must not reset", err, a.Screen(), exits)
	}
}

func TestEndingMissingAndEmptyCreditsShowLocalizedFallback(t *testing.T) {
	for _, selector := range []int{0, text.SelectorConverting} {
		for _, source := range [][]string{nil, {""}, {" 	", ""}} {
			a := endingTestApp(t)
			a.flow.menuFont.Selector = selector
			a.SetCampaignEnding(func() (EndingView, bool) { return EndingView{Credits: source}, true })
			a.flow.showCampaignEnding()
			paint := a.endingPaint()
			found := false
			for _, label := range paint.texts {
				if label.text == a.flow.endingWords().MissingCredits && label.at == image.Pt(48, 86) {
					found = true
				}
			}
			if !found || len(a.flow.ending.Credits) != 0 {
				t.Fatal("missing localized fallback", selector, source)
			}
		}
	}
}
