package ui

import (
	"image"
	"reflect"
	"testing"
	"time"
)

// The pane points the second game's inn and shop shots name: the eye, the
// empty or book corner, PREVIOUS HERO and NEXT HERO.
var (
	squarePaneEye      = image.Pt(622, 253)
	squarePaneTopLeft  = image.Pt(490, 255)
	squarePanePrevious = image.Pt(490, 457)
	squarePaneNext     = image.Pt(612, 457)
)

// fakePaneSquareTown is a room page with a character pane: it records the
// corners the pane's seam receives.
type fakePaneSquareTown struct {
	fakeSquareArtTown
	view    TownSquareView
	corners []CharacterPaneCorner
}

func (f *fakePaneSquareTown) TownSquareView() TownSquareView { return f.view }

func (f *fakePaneSquareTown) TownSquarePane(c CharacterPaneCorner) {
	f.corners = append(f.corners, c)
	if c == CharacterPaneMode {
		f.view.Hero.Statistics = !f.view.Hero.Statistics
	}
}

// paneSquareTown is an inn-like room: one member, no book corner.
func paneSquareTown(members int, book bool) *fakePaneSquareTown {
	f := &fakePaneSquareTown{}
	f.view = TownSquareView{Scene: &fakeSquareScene{fill: 0x40}, Font: messageFont(), Pane: true,
		Hero: TownCharacterView{HasSubject: true, Session: CharacterPaneShop, PaneRect: TownCharacterRegion,
			NoBook: !book, MemberCount: members, Font: messageFont()}}
	return f
}

func paneSquareApp(t *testing.T, town *fakePaneSquareTown) *App {
	t.Helper()
	a := newTestApp(t, appRows(3), okLoader(t))
	a.Layout(640, 480)
	w := Words{}
	w.Hover[8], w.Hover[9] = "OPEN SPELLBOOK", "CLOSE SPELLBOOK"
	w.Hover[12], w.Hover[13] = "SWITCH TO PICTURE", "SWITCH TO STATS"
	w.Hover[52], w.Hover[53] = "PREVIOUS HERO", "NEXT HERO"
	a.SetWords(w, messageFont(), nil)
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused the room page")
	}
	return a
}

func TestComposeTownSquareDrawsThePaneOnlyWhenAsked(t *testing.T) {
	town := paneSquareTown(1, false)
	with := ComposeTownSquare(town.view)
	town.view.Pane = false
	without := ComposeTownSquare(town.view)
	in, out := image.Pt(560, 300), image.Pt(470, 300)
	if with.RGBAAt(in.X, in.Y) == without.RGBAAt(in.X, in.Y) {
		t.Fatalf("pixel %v is the scene's with the pane on", in)
	}
	if with.RGBAAt(out.X, out.Y) != without.RGBAAt(out.X, out.Y) {
		t.Fatalf("pixel %v outside the pane moved", out)
	}
}

func TestSquarePaneCornerCaptions(t *testing.T) {
	town := paneSquareTown(1, false)
	a := paneSquareApp(t, town)
	for _, tc := range []struct {
		at   image.Point
		want []string
	}{
		{squarePaneEye, []string{"SWITCH TO STATS"}},
		{squarePaneTopLeft, nil},
		{squarePanePrevious, []string{"PREVIOUS HERO"}},
		{squarePaneNext, []string{"NEXT HERO"}},
	} {
		a.tooltipPoint = tc.at
		got := a.tooltipTarget()
		if !reflect.DeepEqual(got.lines, tc.want) {
			t.Errorf("caption at %v = %q, want %q", tc.at, got.lines, tc.want)
		}
	}
	town.view.Hero.Statistics = true
	a.tooltipPoint = squarePaneEye
	if got := a.tooltipTarget(); !reflect.DeepEqual(got.lines, []string{"SWITCH TO PICTURE"}) {
		t.Fatalf("eye over the sheet = %q", got.lines)
	}
	shop := paneSquareTown(1, true)
	a = paneSquareApp(t, shop)
	a.tooltipPoint = squarePaneTopLeft
	if got := a.tooltipTarget(); !reflect.DeepEqual(got.lines, []string{"OPEN SPELLBOOK"}) {
		t.Fatalf("shop top-left corner = %q", got.lines)
	}
}

func TestSquarePaneEyeAndTabSwapTheSheet(t *testing.T) {
	town := paneSquareTown(1, false)
	a := paneSquareApp(t, town)
	now := time.Unix(1_700_000_000, 0)
	click := func(p image.Point) {
		a.stepTownAt(appInput{CursorX: p.X, CursorY: p.Y, PrimaryReleased: true}, now)
	}
	click(squarePaneEye)
	if !town.view.Hero.Statistics {
		t.Fatal("the eye left the portrait")
	}
	click(squarePaneEye)
	if town.view.Hero.Statistics {
		t.Fatal("the second eye click kept the sheet")
	}
	a.stepTownAt(appInput{PaneMode: true}, now)
	a.stepTownAt(appInput{PaneMode: true}, now)
	want := []CharacterPaneCorner{CharacterPaneMode, CharacterPaneMode, CharacterPaneMode, CharacterPaneMode}
	if !reflect.DeepEqual(town.corners, want) {
		t.Fatalf("corners = %v, want %v", town.corners, want)
	}
	a.stepTownAt(appInput{Book: true}, now)
	click(squarePaneTopLeft)
	if len(town.corners) != 4 {
		t.Fatalf("an inn without a book corner took %v", town.corners[4:])
	}
}

func TestSquarePanePickerKeepsTheOnlyMember(t *testing.T) {
	town := paneSquareTown(1, false)
	a := paneSquareApp(t, town)
	now := time.Unix(1_700_000_000, 0)
	for _, p := range []image.Point{squarePanePrevious, squarePaneNext} {
		a.stepTownAt(appInput{CursorX: p.X, CursorY: p.Y, PrimaryReleased: true}, now)
	}
	if len(town.corners) != 0 {
		t.Fatalf("one member stepped: %v", town.corners)
	}
	two := paneSquareTown(2, true)
	a = paneSquareApp(t, two)
	for _, p := range []image.Point{squarePanePrevious, squarePaneNext, squarePaneTopLeft} {
		a.stepTownAt(appInput{CursorX: p.X, CursorY: p.Y, PrimaryReleased: true}, now)
	}
	want := []CharacterPaneCorner{CharacterPanePrev, CharacterPaneNext, CharacterPaneBook}
	if !reflect.DeepEqual(two.corners, want) {
		t.Fatalf("corners = %v, want %v", two.corners, want)
	}
}

// The pane's figure and a room doll answer their worn slots' item text.
func TestSquarePaneAndDollItemText(t *testing.T) {
	town := paneSquareTown(1, false)
	surface := townHeroSurface(false, dollHoverInfo)
	town.view.HeroSlotMask, town.view.HeroSlotInfo = surface.HeroSlotMask, surface.HeroSlotInfo
	doll := TownSquareDoll{At: image.Pt(11, 240), Clip: image.Rect(0, 238, 160, 480), Mask: surface.HeroSlotMask, Info: dollHoverInfo}
	town.view.Dolls = []TownSquareDoll{doll}
	a := paneSquareApp(t, town)
	a.tooltipPoint = shopFigureRect.Min.Add(townDollPixels[0])
	if got := a.tooltipTarget(); !reflect.DeepEqual(got.lines, dollHoverInfo[2]) {
		t.Fatalf("pane figure = %q, want %q", got.lines, dollHoverInfo[2])
	}
	a.tooltipPoint = doll.At.Add(townDollPixels[1])
	if got := a.tooltipTarget(); !reflect.DeepEqual(got.lines, dollHoverInfo[4]) {
		t.Fatalf("room doll = %q, want %q", got.lines, dollHoverInfo[4])
	}
	if got, ok := doll.LinesAt(doll.At.Add(image.Pt(71, 100))); ok {
		t.Fatalf("a blank doll pixel = %q", got)
	}
	clipped := doll
	clipped.Clip = image.Rect(0, 238, 10, 480)
	if got, ok := clipped.LinesAt(doll.At.Add(townDollPixels[1])); ok {
		t.Fatalf("a pixel outside the clip = %q", got)
	}
	town.view.Hero.Statistics = true
	a.tooltipPoint = shopFigureRect.Min.Add(townDollPixels[0])
	if got := a.tooltipTarget(); got.lines != nil && reflect.DeepEqual(got.lines, dollHoverInfo[2]) {
		t.Fatal("the sheet answers the figure's item text")
	}
}
