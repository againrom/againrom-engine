package ui

import (
	"image"
	"image/color"
	"testing"
	"time"
)

func tipPhaseApp(t *testing.T) (*App, *fakeTipSquareEnumTown) {
	t.Helper()
	town := &fakeTipSquareEnumTown{
		scene: &fakeSquareScene{},
		tip:   TipPanelView{Rect: TownTipRect, Text: "tip", Art: tipTestArt(), Font: shopTipTestFont()},
	}
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("fixture could not show town")
	}
	return a, town
}

func TestTownTipCheckboxCommitsOnDown(t *testing.T) {
	a, town := tipPhaseApp(t)
	p, ok := sampleInside(TipPanelToggleRect(TownTipRect))
	if !ok {
		t.Fatal("fixture has no checkbox")
	}
	if kind, consumed := TipPanelControlAt(town.tip, p); !consumed || kind != TipControlToggle {
		t.Fatal("fixture point does not reach checkbox")
	}
	a.step(appInput{CursorX: p.X, CursorY: p.Y, PrimaryPressed: true}, time.Unix(1, 0))
	if town.toggled != 1 {
		t.Fatalf("checkbox down stored %d flag changes; want1 before any release", town.toggled)
	}
	a.step(appInput{CursorX: 0, CursorY: 479, PrimaryReleased: true}, time.Unix(2, 0))
	if town.toggled != 1 {
		t.Fatalf("outside release stored another checkbox change: %d", town.toggled)
	}
}

func TestTownTipCloseRequiresRegisteredDown(t *testing.T) {
	a, town := tipPhaseApp(t)
	p, ok := sampleInside(TipPanelCloseRect(TownTipRect))
	if !ok {
		t.Fatal("fixture has no Close button")
	}
	a.step(appInput{CursorX: p.X, CursorY: p.Y, PrimaryReleased: true}, time.Unix(1, 0))
	if town.closed != 0 {
		t.Fatalf("unregistered release closed %d popups; want0", town.closed)
	}
}

func TestTownTipEventConsumption(t *testing.T) {
	for _, r := range []image.Rectangle{TownTipRect, TavernTipRect, SchoolTipRect, shopTipRect} {
		v := TipPanelView{Rect: r, Text: "tip", Art: tipTestArt(), Font: shopTipTestFont()}
		toggle, _ := sampleInside(TipPanelToggleRect(r))
		close, _ := sampleInside(TipPanelCloseRect(r))
		for _, tc := range []struct {
			p        image.Point
			down, up bool
		}{
			{r.Min.Add(image.Pt(2, 2)), false, false},
			{r.Min.Add(image.Pt(30, 30)), false, true},
			{toggle, true, false}, {close, true, true},
			{TipPanelCloseRect(r).Min.Add(image.Pt(1, 1)), true, true},
		} {
			_, down := TipPanelEventAt(v, tc.p, true)
			_, up := TipPanelEventAt(v, tc.p, false)
			if down != tc.down || up != tc.up {
				t.Fatalf("rect %v point %v consumption down/up %v/%v want %v/%v", r, tc.p, down, up, tc.down, tc.up)
			}
		}
	}
}

func TestTownTipCloseOwnershipCanceled(t *testing.T) {
	for _, route := range []string{"outside", "focus", "menu", "text", "popup", "revision", "town"} {
		t.Run(route, func(t *testing.T) {
			a, town := tipPhaseApp(t)
			p, _ := sampleInside(TipPanelCloseRect(TownTipRect))
			pressAt(a, time.Unix(1, 0), p)
			switch route {
			case "outside":
				releaseAt(a, time.Unix(2, 0), image.Pt(0, 479))
			case "focus":
				a.step(appInput{Unfocused: true}, time.Unix(2, 0))
			case "menu":
				a.step(appInput{Escape: true}, time.Unix(2, 0))
				a.step(appInput{Escape: true}, time.Unix(3, 0))
			case "text":
				town.tip.Text = "replacement"
			case "popup":
				town.tip.Art = nil
				a.step(appInput{}, time.Unix(2, 0))
				town.tip.Art = tipTestArt()
			case "revision":
				town.tip.Revision++
			case "town":
				a.SetTown(town)
			}
			releaseAt(a, time.Unix(4, 0), p)
			if town.closed != 0 {
				t.Fatal("canceled Close ownership closed popup")
			}
			clickAt(a, time.Unix(5, 0), p)
			if town.closed != 1 {
				t.Fatal("fresh registered Close gesture was lost")
			}
		})
	}
}

// The tavern roster's double click is the detector's: a second press inside
// the system's time and rectangle, wherever the first press was released and
// whatever the cell held, runs the roster press again at its point and acts on
// the cell it hits (MENU-146, TAVERN-CLICK-019).
func TestTavernRosterDoubleClick(t *testing.T) {
	for _, tc := range []struct {
		name   string
		dt     time.Duration
		cancel bool
		key    bool
		want   bool
	}{
		{"inside the fallback time", 500 * time.Millisecond, false, false, true},
		{"late", 501 * time.Millisecond, false, false, false},
		{"released off the cell", 100 * time.Millisecond, true, false, true},
		{"another occupant under the second press", 100 * time.Millisecond, false, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			town := &downPairTown{view: TownSurfaceView{Kind: TownSurfaceTavern, Cells: []TownSurfaceCell{{Portrait: true, Key: "merc:3"}}}}
			a := newTestApp(t, appRows(3), okLoader(t))
			a.SetTown(town)
			a.flow.showTown("")
			p := image.Pt(200, 450)
			now := time.Unix(1, 0)
			pressAt(a, now, p)
			if len(town.actions) != 1 || town.actions[0] {
				t.Fatal("first down did not select alone")
			}
			if tc.cancel {
				releaseAt(a, now, image.Pt(2, 2))
			} else {
				releaseAt(a, now, p)
			}
			if len(town.actions) != 1 {
				t.Fatal("release enacted a cell action")
			}
			if tc.key {
				town.view.Cells[0].Key = "merc:7"
			}
			pressAt(a, now.Add(tc.dt), p)
			if len(town.actions) != 2 || town.actions[1] != tc.want {
				t.Fatalf("actions %v want second double %v", town.actions, tc.want)
			}
		})
	}
}

// The school takes a double click's second press as a single press.
func TestSchoolTakesNoDoubleClick(t *testing.T) {
	town := &downPairTown{view: TownSurfaceView{Kind: TownSurfaceSchool, Cells: []TownSurfaceCell{{Portrait: true, Key: "skill"}}}}
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	a.flow.showTown("")
	p := image.Pt(200, 450)
	now := time.Unix(1, 0)
	pressAt(a, now, p)
	releaseAt(a, now, p)
	pressAt(a, now.Add(50*time.Millisecond), p)
	for i, double := range town.actions {
		if double {
			t.Fatalf("school action %d of %v was a double click", i, town.actions)
		}
	}
}

type downPairTown struct {
	fakeTown
	view    TownSurfaceView
	actions []bool
}

func (t *downPairTown) AtTownSurface() bool          { return true }
func (t *downPairTown) TownSurface() TownSurfaceView { return t.view }
func (t *downPairTown) TownSurfaceClick(c TownSurfaceControl, double bool) TownAction {
	t.actions = append(t.actions, double)
	return TownAction{}
}

func TestSchoolAppClassMaskAndFigureOverlap(t *testing.T) {
	for class := 0; class < 2; class++ {
		art := &TownSchoolArt{}
		for i, r := range schoolPanelRects {
			art.Masks[i] = image.NewPaletted(image.Rect(0, 0, r.Dx(), r.Dy()), make(color.Palette, 256))
		}
		code := uint8(0xff)
		if class == 1 {
			code = 0x87
		}
		art.Masks[class].SetColorIndex(10, 10, code)
		cells := make([]TownSurfaceCell, 10)
		for i := class * 5; i < class*5+5; i++ {
			cells[i].Enabled = true
		}
		town := &fakeTipSurfaceEnumTown{view: TownSurfaceView{Kind: TownSurfaceSchool, SchoolArt: art, SchoolClass: class, Cells: cells,
			Buttons: []TownSurfaceButton{{Enabled: true}, {Enabled: true}}}}
		a := newTestApp(t, appRows(3), okLoader(t))
		a.SetTown(town)
		a.flow.showTown("")
		now := time.Unix(1, 0)
		for _, p := range []image.Point{image.Pt(80, 300), image.Pt(350, 300), image.Pt(230, 100), image.Pt(470, 220)} {
			clickAt(a, now, p)
		}
		if len(town.surfaceClicks) != 0 {
			t.Fatal("trainer/diamond/button-panel overlap created an action", town.surfaceClicks)
		}
		p := schoolPanelRects[class].Min.Add(image.Pt(10, 10))
		pressAt(a, now, p)
		if len(town.surfaceClicks) != 1 || town.surfaceClicks[0] != (TownSurfaceControl{Kind: TownSurfaceControlCell, Index: class * 5}) {
			t.Fatal("class mask down did not reach its own skill", town.surfaceClicks)
		}
		releaseAt(a, now, p)
		if len(town.surfaceClicks) != 1 {
			t.Fatal("school cell release repeated action")
		}
		button := TownSurfaceButtonRect(TownSurfaceSchool, 0).Min.Add(image.Pt(10, 10))
		pressAt(a, now, button)
		if len(town.surfaceClicks) != 1 {
			t.Fatal("school button acted on down")
		}
		releaseAt(a, now, button)
		if len(town.surfaceClicks) != 2 || town.surfaceClicks[1].Kind != TownSurfaceControlButton {
			t.Fatal("enabled school button release did not reach action")
		}
	}
}

func TestShopMerchantReleaseCancelsHeldItem(t *testing.T) {
	a, town := shopDragTestApp(t)
	town.dollIcon = image.NewRGBA(image.Rect(0, 0, 4, 4))
	now := time.Unix(1, 0)
	merchant := image.Pt(300, 180)
	if c, ok := shopScreenControlAt(town.ShopScreen(), merchant); !ok || c.Kind != ShopControlMerchant {
		t.Fatal("fixture does not reach merchant")
	}
	pressAt(a, now, shopDragDollPoint)
	a.step(appInput{CursorX: merchant.X, CursorY: merchant.Y}, now)
	if _, held := a.shopDragItemPresent(); !held {
		t.Fatal("merchant release control lacks held item")
	}
	releaseAt(a, now, merchant)
	if _, held := a.shopDragItemPresent(); held || a.shopDragArmed {
		t.Fatal("merchant release retained held item")
	}
	if len(town.clicked) != 0 || len(town.dragged) != 0 {
		t.Fatal("merchant cancellation mutated model", town.clicked, town.dragged)
	}
	clickAt(a, now, merchant)
	if len(town.clicked) != 0 || len(town.dragged) != 0 {
		t.Fatal("unheld merchant click mutated model")
	}
	clickAt(a, now, shopDragDollPoint)
	if len(town.clicked) != 1 {
		t.Fatal("merchant control swallowed following doll action")
	}
}

func TestTownEntryPaintAdmissionKeepsDialogueAndPopupInputOwners(t *testing.T) {
	a, base := tipPhaseApp(t)
	town := &entryAdmissionTown{fakeTipSquareEnumTown: base, dialogue: true}
	a.SetTown(town)
	a.paintTownEntry(false, 0, true)
	if town.paints != 0 {
		t.Fatal("synchronous entry paint advanced through dialogue")
	}
	if v, _ := a.currentTownTip(); v.Showing() {
		t.Fatal("dialogue exposed underlying popup input")
	}
	town.dialogue = false
	a.paintTownEntry(false, 0, true)
	if town.paints != 1 {
		t.Fatal("admitted unobscured entry did not paint")
	}
	if v, _ := a.currentTownTip(); !v.Showing() {
		t.Fatal("constructed popup lost input after dialogue")
	}
}

type entryAdmissionTown struct {
	*fakeTipSquareEnumTown
	dialogue bool
	paints   int
}

func (t *entryAdmissionTown) TownDialogue() (*image.RGBA, bool) {
	return image.NewRGBA(image.Rect(0, 0, 1, 1)), t.dialogue
}
func (t *entryAdmissionTown) TownSquareActive(bool)           {}
func (t *entryAdmissionTown) AdvanceTownSquareAnimation()     { t.paints++ }
func (t *entryAdmissionTown) TownSquarePointer(image.Point)   {}
func (t *entryAdmissionTown) AdvanceTownDialogue() TownAction { return TownAction{} }
func (t *entryAdmissionTown) TownDialogueButton() (image.Rectangle, bool) {
	return image.Rect(0, 0, 1, 1), t.dialogue
}

var _ TownDialogueScreen = (*entryAdmissionTown)(nil)
