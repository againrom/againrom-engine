package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"testing"
)

// widgetTestFrames is a synthetic scrlbars.256: 26 opaque 24x24 frames,
// each a distinct gradient, with a transparent 4-pixel right column so a
// part's own shadow shows beside it.
func widgetTestFrames() []*image.RGBA {
	frames := make([]*image.RGBA, 26)
	for i := range frames {
		pic := image.NewRGBA(image.Rect(0, 0, 24, 24))
		for y := 0; y < 24; y++ {
			for x := 0; x < 20; x++ {
				pic.SetRGBA(x, y, color.RGBA{uint8(40 + i*8), uint8(20 + x*5), uint8(30 + y*7), 255})
			}
		}
		frames[i] = pic
	}
	return frames
}

func recordWidgets(t *testing.T, draw func()) []widgetCall {
	t.Helper()
	var calls []widgetCall
	widgetRecord = func(c widgetCall) { calls = append(calls, c) }
	defer func() { widgetRecord = nil }()
	draw()
	return calls
}

// TestVScrollBarPaintsTheClaimedParts checks MENU-117 and MENU-119 on a
// 24x192 bar: top 18 (21 hot), track 19 tiles, bottom 20 (23 hot), the fixed
// thumb 22 at (L,T+W+q-4) with q=trunc(p*(H-3W+8)/(N-1)), every part's
// level-4 shadow at (+4,+4), and a disabled bar's level-3 remap.
func TestVScrollBarPaintsTheClaimedParts(t *testing.T) {
	frames := widgetTestFrames()
	bg := color.RGBA{120, 140, 160, 255}
	r := image.Rect(40, 20, 64, 212)
	shade := mustLevel(t, 4)
	for _, tc := range []struct {
		pos, count        int
		topHot, bottomHot bool
		disabled          bool
	}{{0, 27, false, false, false}, {13, 27, true, false, false}, {26, 27, false, true, false}, {5, 1, false, false, true}} {
		t.Run(fmt.Sprintf("%d of %d", tc.pos, tc.count), func(t *testing.T) {
			dst := image.NewRGBA(image.Rect(0, 0, 100, 240))
			draw.Draw(dst, dst.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)
			b := vScrollBar{Rect: r, Pos: tc.pos, Count: tc.count, TopHot: tc.topHot, BottomHot: tc.bottomHot, Disabled: tc.disabled}
			drawVScrollBar(dst, frames, b)
			q := 0
			if tc.count >= 2 {
				q = tc.pos * (192 - 3*24 + 8) / (tc.count - 1)
			}
			thumbY := 20 + 24 + q - 4
			top, bottom := 18, 20
			if tc.topHot {
				top = 21
			}
			if tc.bottomHot {
				bottom = 23
			}
			want := map[image.Point]color.RGBA{
				{40, 20}:          frames[top].RGBAAt(0, 0),
				{59, 30}:          frames[top].RGBAAt(19, 10),
				{40, 211}:         frames[bottom].RGBAAt(0, 23),
				{40, 44 + 24 + 5}: frames[19].RGBAAt(0, 5),
				{40, thumbY}:      frames[22].RGBAAt(0, 0),
				{59, thumbY + 23}: frames[22].RGBAAt(19, 23),
				// Right of the bar: background under the parts' shadows.
				{61, 30}:  shade.Color(bg),
				{64, 30}:  bg,
				{68, 30}:  bg,
				{44, 213}: shade.Color(bg),
				{43, 213}: bg,
			}
			if tc.disabled {
				l := mustLevel(t, 3)
				for p, c := range want {
					if p.In(r) {
						want[p] = l.Color(c)
					}
				}
			}
			for p, c := range want {
				if got := dst.RGBAAt(p.X, p.Y); got != c {
					t.Fatalf("%v = %v, want %v", p, got, c)
				}
			}
		})
	}
}

// TestVScrollBarInputRegionsAndDrag checks MENU-119's press regions and the
// drag arm clamp(trunc((N-1)*(y-T-24)/(H-3*(W-4))),0,N-1).
func TestVScrollBarInputRegionsAndDrag(t *testing.T) {
	b := vScrollBar{Rect: image.Rect(40, 20, 64, 212), Pos: 13, Count: 27}
	thumb := b.Thumb()
	for _, tc := range []struct {
		y    int
		req  barRequest
		grab bool
	}{{20, barLineUp, false}, {43, barLineUp, false}, {44, barPageUp, false}, {thumb.Min.Y - 1, barPageUp, false},
		{thumb.Min.Y, barNone, true}, {thumb.Max.Y - 1, barNone, true}, {thumb.Max.Y, barPageDown, false},
		{191, barPageDown, false}, {192, barLineDown, false}, {211, barLineDown, false}} {
		if req, grab := b.press(image.Pt(50, tc.y)); req != tc.req || grab != tc.grab {
			t.Errorf("press at y %d = %v/%t, want %v/%t", tc.y, req, grab, tc.req, tc.grab)
		}
	}
	for _, y := range []int{0, 44, 100, 176, 300} {
		want := min(max(26*(y-20-24)/(192-3*(24-4)), 0), 26)
		if got := b.dragPos(y); got != want {
			t.Errorf("dragPos(%d) = %d, want %d", y, got, want)
		}
	}
	if (vScrollBar{Rect: b.Rect, Count: 1}).dragPos(150) != 0 {
		t.Error("a one-item range dragged off zero")
	}
}

func listTestPicker(n int) *Picker {
	rows := make([]PickerRow, n)
	for i := range rows {
		rows[i] = PickerRow{Text: fmt.Sprint(i), Choosable: true}
	}
	return NewPicker(rows).SetWindow(10)
}

// TestListKeysFollowTheSharedList is MENU-120's four keys: Up and Down move
// by one; Page Up first selects the top row and then pages; Page Down first
// selects the last visible row and then pages.
func TestListKeysFollowTheSharedList(t *testing.T) {
	m := listTestPicker(40)
	step := func(up, down, pageUp, pageDown bool, sel, top int) {
		t.Helper()
		listKey(m, up, down, pageUp, pageDown)
		if gotTop, _ := m.Visible(); m.Selection() != sel || gotTop != top {
			t.Fatalf("selection/top = %d/%d, want %d/%d", m.Selection(), gotTop, sel, top)
		}
	}
	step(false, false, false, true, 9, 0)
	step(false, false, false, true, 19, 10)
	step(false, true, false, false, 20, 11)
	step(false, false, true, false, 11, 11)
	step(false, false, true, false, 1, 1)
	step(true, false, false, false, 0, 0)
	step(true, false, false, false, 0, 0)
}

// TestListAndBarMoveTogether drives the shared bar input over a 27-row list
// in a 10-row window: a thumb drag selects the dragged position and the
// visible rows follow it; a track press pages; an endcap press steps a line.
func TestListAndBarMoveTogether(t *testing.T) {
	m := listTestPicker(27)
	l := listBox{Rect: image.Rect(10, 20, 300, 212), Pitch: 19, Rows: 10}
	var in scrollBarInput
	tick := func(p image.Point, ev appInput) {
		ev.CursorX, ev.CursorY = p.X, p.Y
		if req, pos := in.step(listBar(l, m), p, true, ev); req != barNone {
			listBarRequest(m, req, pos)
		}
	}
	held := appInput{Viewer: Input{PrimaryDown: true}}
	bar := listBar(l, m)
	tick(bar.Thumb().Min.Add(image.Pt(5, 5)), appInput{PrimaryPressed: true, Viewer: Input{PrimaryDown: true}})
	for _, y := range []int{60, 120, 175, 205, 30} {
		tick(image.Pt(310, y), held)
		want := min(max(26*(y-20-24)/(192-60), 0), 26)
		top, _ := m.Visible()
		if m.Selection() != want || want < top || want >= top+10 {
			t.Fatalf("drag to y %d: selection %d top %d, want selection %d visible", y, m.Selection(), top, want)
		}
		if listBar(l, m).Pos != want {
			t.Fatal("the thumb does not follow the selection")
		}
	}
	tick(image.Pt(310, 205), appInput{PrimaryReleased: true})
	if in.active() {
		t.Fatal("release kept the drag")
	}
	m.Select(0)
	tick(image.Pt(310, 150), appInput{PrimaryPressed: true, Viewer: Input{PrimaryDown: true}})
	tick(image.Pt(310, 150), appInput{PrimaryReleased: true})
	if m.Selection() != 9 {
		t.Fatalf("track press below the thumb selected %d, want 9", m.Selection())
	}
	tick(image.Pt(310, 25), appInput{PrimaryPressed: true, Viewer: Input{PrimaryDown: true}})
	tick(image.Pt(310, 25), appInput{PrimaryReleased: true})
	if m.Selection() != 8 {
		t.Fatalf("top endcap selected %d, want 8", m.Selection())
	}
	tick(image.Pt(310, 205), appInput{PrimaryPressed: true, Viewer: Input{PrimaryDown: true}})
	if m.Selection() != 9 {
		t.Fatalf("bottom endcap selected %d, want 9", m.Selection())
	}
	for i := 0; i < 40; i++ {
		tick(image.Pt(310, 205), held)
	}
	tick(image.Pt(310, 205), appInput{PrimaryReleased: true})
	if m.Selection() <= 9 {
		t.Fatal("a held endcap did not repeat")
	}
}
