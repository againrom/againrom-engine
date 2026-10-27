package ui

import (
	"image"
	"testing"

	"againrom/pkg/render/text"
)

// litShopTown is a shop whose Undo and Buy cannot act, drawn in a lit style.
type litShopTown struct {
	fakeTipShopCrossingTown
	style *RoomButtonStyle
}

func (f *litShopTown) ShopScreen() ShopScreenView {
	v := f.fakeTipShopCrossingTown.ShopScreen()
	v.Live = [4]bool{false, false, true, true}
	v.ButtonStyle = f.style
	return v
}

func litTestStyle(lit bool) *RoomButtonStyle {
	return &RoomButtonStyle{Ink: testRoomStyleInk, Hover: testRoomStyleHover, Lit: lit}
}

// A lit style draws a button that cannot act as one that can: not
// disabled, in the rest ink, and in the hover ink under the pointer. The
// same style without Lit draws it disabled.
func TestLitRoomButtonsDrawAsButtonsThatCanAct(t *testing.T) {
	if (*RoomButtonStyle)(nil).disabled(false) != true || litTestStyle(false).disabled(false) != true {
		t.Fatal("a button that cannot act must draw disabled without a lit style")
	}
	if litTestStyle(true).disabled(false) {
		t.Fatal("a lit style draws a button that cannot act disabled")
	}
	v := ShopScreenView{Font: shippedShapeFont(), Live: [4]bool{false, false, true, true}}
	v.Words.ShopUndo, v.Words.ShopBuy, v.Words.ShopSell, v.Words.ShopExit = "A", "B", "AA", "BB"
	l := v.layout()
	over := l.Button[0].Min.Add(l.Button[0].Size().Div(2))
	for _, lit := range []bool{false, true} {
		v.ButtonStyle = litTestStyle(lit)
		p := shopButtonPanel(v, over, true)
		if p.buttons[0].Disabled == lit || p.buttons[1].Disabled == lit {
			t.Errorf("lit %t: Undo disabled %t, Buy disabled %t", lit, p.buttons[0].Disabled, p.buttons[1].Disabled)
		}
		if !lit {
			continue
		}
		dst := image.NewRGBA(image.Rect(0, 0, 640, 480))
		calls := text.Record(func() { p.drawButtons(dst, v.Font) })
		if len(calls) == 0 || calls[0].Color != testRoomStyleHover {
			t.Errorf("a lit Undo under the pointer draws %v, want the hover ink %v", calls, testRoomStyleHover)
		}
	}
	school := TownSurfaceView{Kind: TownSurfaceSchool, SchoolClass: -1,
		Buttons: []TownSurfaceButton{{Label: "Train", Value: "0"}, {Label: "Exit", Enabled: true}}}
	for _, lit := range []bool{false, true} {
		school.ButtonStyle = litTestStyle(lit)
		if p := townSurfacePanel(school, false, false, false); p.buttons[0].Disabled == lit {
			t.Errorf("lit %t: Train at 0 disabled %t", lit, p.buttons[0].Disabled)
		}
	}
}

// A lit Hire, Train, Undo or Buy the purse or the table cannot serve still
// refuses the press: the town surface finds no control under it and the shop
// latches no command.
func TestLitRoomButtonsThatCannotActStillRefuse(t *testing.T) {
	buttons := []TownSurfaceButton{{Label: "Hire", Value: "520"}, {Label: "Exit", Enabled: true}}
	town := &fakeTipSurfaceEnumTown{view: TownSurfaceView{Kind: TownSurfaceSchool, SchoolClass: -1,
		Buttons: buttons, ButtonStyle: litTestStyle(true)}}
	a := latchTownApp(t, town)
	inside, _ := sampleInside(TownSurfaceButtonRect(TownSurfaceSchool, 0))
	if _, hit := TownSurfaceControlAt(town.view, inside); hit {
		t.Fatal("a lit button the purse cannot pay is a control")
	}
	site := townLatchSite{app: a}
	site.pointer(t, "press", inside)
	site.pointer(t, "release", inside)
	for _, c := range town.surfaceClicks {
		if c.Kind == TownSurfaceControlButton {
			t.Fatalf("a lit button the purse cannot pay fired %+v", c)
		}
	}
	shop := &litShopTown{fakeTipShopCrossingTown{mask: shopTestMask()}, litTestStyle(true)}
	a = latchTownApp(t, shop)
	site = townLatchSite{app: a}
	for _, i := range []int{0, 1} {
		at, _ := sampleInside(shopButtonRects[i])
		site.pointer(t, "press", at)
		site.pointer(t, "release", at)
	}
	for _, c := range shop.clicked {
		if c.Kind == ShopControlButton {
			t.Fatalf("a lit shop button that cannot act fired %+v", c)
		}
	}
	// Control: a live Sell under the same style fires.
	at, _ := sampleInside(shopButtonRects[2])
	site.pointer(t, "press", at)
	site.pointer(t, "release", at)
	fired := 0
	for _, c := range shop.clicked {
		if c.Kind == ShopControlButton {
			fired++
		}
	}
	if fired != 1 {
		t.Fatalf("a live Sell fired %d times, want 1", fired)
	}
}

// The shop buttons a style marks one-line draw the caption and the number
// as one string (SHOP-050); the others keep two lines, their label and value
// moved by the style's shifts.
func TestOneLineShopButtonsJoinCaptionAndNumber(t *testing.T) {
	v := ShopScreenView{Font: shippedShapeFont(), Purse: 600, Buy: 0, Sell: 0, Total: 600, Live: [4]bool{true, true, true, true}}
	v.Words.ShopUndo, v.Words.ShopBuy, v.Words.ShopSell, v.Words.ShopExit = "Undo", "Buy", "Sell", "Exit"
	v.ButtonStyle = &RoomButtonStyle{Ink: testRoomStyleInk, Hover: testRoomStyleHover, OneLine: [4]bool{false, true, true, false},
		LabelShift: image.Pt(0, 6), ValueShift: image.Pt(0, 1)}
	p := shopButtonPanel(v, image.Point{}, false)
	want := [][]string{{"Undo", "600"}, {"Buy 0"}, {"Sell 0"}, {"Exit", "600"}}
	for i, w := range want {
		var got []string
		for _, c := range p.buttons[i].Face.Captions {
			got = append(got, c.Text)
		}
		if len(got) != len(w) {
			t.Fatalf("button %d captions %q, want %q", i, got, w)
		}
		for j := range w {
			if got[j] != w[j] {
				t.Fatalf("button %d captions %q, want %q", i, got, w)
			}
		}
	}
	r := v.layout().Button[0]
	c := p.buttons[0].Face.Captions
	if c[0].Rect != townButtonLabelRect(r).Add(image.Pt(0, 6)) || c[1].Rect != townButtonValueRect(r).Add(image.Pt(0, 1)) {
		t.Errorf("Undo label band %v value band %v, want the shared bands moved by (0,6) and (0,1)", c[0].Rect, c[1].Rect)
	}
	school := TownSurfaceView{Kind: TownSurfaceSchool, SchoolClass: -1, ButtonStyle: v.ButtonStyle,
		Buttons: []TownSurfaceButton{{Label: "Train", Value: "0", Enabled: true}, {Label: "Exit", Value: "600", Enabled: true}}}
	c = townSurfacePanel(school, false, false, false).buttons[1].Face.Captions
	r = TownSurfaceButtonRect(TownSurfaceSchool, 1)
	if c[0].Rect != townButtonLabelRect(r).Add(image.Pt(0, 6)) || c[1].Rect != townButtonValueRect(r).Add(image.Pt(0, 1)) {
		t.Errorf("school Exit label band %v value band %v, want the shared bands moved by (0,6) and (0,1)", c[0].Rect, c[1].Rect)
	}
}

// The first game's Load page frames each of its ten rows, keeps no line
// under the list, sets the bar 3 pixels off the rows and places Ok, Delete
// and Cancel as the EN original draws them.
func TestFirstLoadPageFramesTenRowsAndPlacesItsButtons(t *testing.T) {
	p := firstLoadPage(LoadWindowWords{})
	if !p.Framed || p.Ruled || !p.RowHint || p.BarGap != 3 || p.Rows != 10 {
		t.Fatalf("framed %t ruled %t row hint %t bar gap %d rows %d", p.Framed, p.Ruled, p.RowHint, p.BarGap, p.Rows)
	}
	for i, want := range []image.Rectangle{image.Rect(149, 368, 246, 392), image.Rect(271, 368, 368, 392), image.Rect(393, 368, 490, 392)} {
		if p.Buttons[i].Rect != want {
			t.Errorf("button %d at %v, want %v", i, p.Buttons[i].Rect, want)
		}
	}
	if p.List.Min != image.Pt(116, 152) || p.Panel != image.Rect(76, 60, 556, 412) {
		t.Errorf("list at %v, panel %v", p.List, p.Panel)
	}
}
