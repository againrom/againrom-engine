package ui

import (
	"againrom/pkg/render/backdrop"
	"againrom/pkg/render/text"
	"bytes"
	"image"
	"image/color"
	"testing"
)

func TestDialogueButtonPackedStatesReachNotice(t *testing.T) {
	for _, layout := range []backdrop.Layout{backdrop.RGB565, backdrop.RGB555} {
		for _, state := range []DialogueButtonState{{}, {Hover: true}, {Pressed: true, Inside: true}, {Pressed: true}, {Hover: true, Pressed: true, Inside: true}, {Hover: true, Disabled: true}} {
			l := dialogueClaimLayout().WithPortrait(false)
			l.ButtonState = state
			l.DialogueBackdrop.Layout = layout
			l.ButtonLabel = "A"
			pic := RenderNotice(l, dialogueClaimFont(), "", nil)
			light, dark := color.RGBA{41, 68, 57, 255}, color.RGBA{0, 12, 8, 255}
			if layout == backdrop.RGB555 {
				light, dark = color.RGBA{41, 65, 57, 255}, color.RGBA{0, 8, 8, 255}
			}
			if state.Pressed && state.Inside {
				light, dark = dark, light
			}
			if state.Disabled {
				lookup, _ := backdrop.New(layout, backdrop.Full)
				light, dark = lookup.Color(light), lookup.Color(dark)
			}
			for _, tc := range []struct {
				at   image.Point
				want color.RGBA
			}{{image.Pt(202, 172), light}, {image.Pt(279, 174), dark}} {
				if got := pic.RGBAAt(tc.at.X, tc.at.Y); got != tc.want {
					t.Errorf("layout %v state %+v bevel %v=%v want %v", layout, state, tc.at, got, tc.want)
				}
			}
			ink := color.RGBA{189, 157, 74, 255}
			if layout == backdrop.RGB555 {
				ink.G = 156
			}
			if state.Hover {
				ink = color.RGBA{148, 89, 0, 255}
				if layout == backdrop.RGB555 {
					ink.G = 90
				}
			}
			if state.Disabled {
				lookup, _ := backdrop.New(layout, backdrop.Full)
				ink = lookup.Color(ink)
			}
			if got := pic.RGBAAt(238, 176); got != ink {
				t.Errorf("layout %v state %+v label=%v want %v", layout, state, got, ink)
			}
		}
	}
}

func TestDialogueHoverReachesMissionCache(t *testing.T) {
	f := newDialoguePointerFixture(t, "mission")
	v := f.app.flow.viewer
	before, _, _, _ := v.noticePresent()
	count := v.noticeBuilds
	p := f.button.Min.Add(f.button.Max).Div(2)
	f.pointer(t, "move", p, 0)
	after, _, _, _ := v.noticePresent()
	if before == after || v.noticeBuilds != count+1 {
		t.Fatal("hover did not invalidate reached mission modal")
	}
	f.pointer(t, "press", p, 0)
	pressed, _, _, _ := v.noticePresent()
	if pressed == after {
		t.Fatal("press did not invalidate reached mission modal")
	}
	f.pointer(t, "release", p, 1)
}

type visualPointerDialogueTown struct {
	pointerDialogueTown
	layout NoticeLayout
	state  DialogueButtonState
	policy DialogueBackdrop
}

func (town *visualPointerDialogueTown) TownDialogueVisual(state DialogueButtonState, policy DialogueBackdrop) {
	town.state, town.policy = state, policy
}
func (town *visualPointerDialogueTown) TownDialogue() (*image.RGBA, bool) {
	return RenderNotice(town.layout.WithDialogueButtonState(town.state).WithDialogueBackdrop(town.policy), dialogueClaimFont(), "", nil), true
}
func (town *visualPointerDialogueTown) TownDialogueButton() (image.Rectangle, bool) {
	return town.layout.Button, true
}

func TestDialoguePointerStatesReachTownPixelsAndCapture(t *testing.T) {
	town := &visualPointerDialogueTown{layout: AuthoredDialogueLayout().WithPortrait(false)}
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	a.flow.showTown("")
	a.SetTextSmoothing(false)
	before, _, err := a.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	beforePixels := bytes.Clone(before.Pix)
	b := town.layout.Button.Add(town.layout.Box.Min)
	inside := b.Min.Add(b.Max).Div(2)
	outside := image.Pt(b.Min.X-1, inside.Y)
	for _, tc := range []struct {
		action string
		p      image.Point
		want   DialogueButtonState
	}{
		{"move", inside, DialogueButtonState{Hover: true, Inside: true}},
		{"press", inside, DialogueButtonState{Hover: true, Inside: true, Pressed: true}},
		{"move", outside, DialogueButtonState{Pressed: true}},
		{"move", inside, DialogueButtonState{Hover: true, Inside: true, Pressed: true}},
		{"release", inside, DialogueButtonState{Hover: true, Inside: true}},
	} {
		if err := a.HeadlessPointer(tc.action, tc.p.X, tc.p.Y); err != nil {
			t.Fatal(err)
		}
		if town.state != tc.want {
			t.Fatalf("%s state=%+v want%+v", tc.action, town.state, tc.want)
		}
		picture, _, err := a.HeadlessFrame()
		if err != nil {
			t.Fatal(err)
		}
		if same := bytes.Equal(beforePixels, picture.Pix); same == tc.want.Hover {
			t.Fatalf("%s state %+v has baseline pixels=%v", tc.action, tc.want, same)
		}
		wantTop := color.RGBA{41, 68, 57, 255}
		if tc.want.Pressed && tc.want.Inside {
			wantTop = color.RGBA{0, 12, 8, 255}
		}
		p := b.Min.Add(image.Pt(2, 0))
		if got := picture.RGBAAt(p.X, p.Y); got != wantTop {
			t.Fatalf("%s top bevel=%v want%v", tc.action, got, wantTop)
		}
	}
	if town.advances != 1 {
		t.Fatalf("matching release advanced%d pages", town.advances)
	}
}

func TestDialogueButtonBevelWholeBufferAndDisabledModes(t *testing.T) {
	for _, layout := range []backdrop.Layout{backdrop.RGB565, backdrop.RGB555} {
		for _, mode := range []backdrop.Mode{backdrop.Full, backdrop.Reduced} {
			for _, pressed := range []bool{false, true} {
				for _, disabled := range []bool{false, true} {
					l := dialogueClaimLayout().WithPortrait(false)
					l.ButtonLabel = ""
					l.DialogueBackdrop = DialogueBackdrop{Layout: layout, Mode: mode}
					l.ButtonState = DialogueButtonState{Pressed: pressed, Inside: true, Disabled: disabled}
					original := l
					original.Button = image.Rectangle{}
					want := RenderNotice(original, dialogueClaimFont(), "", nil)
					got := RenderNotice(l, dialogueClaimFont(), "", nil)
					light, dark := color.RGBA{41, 68, 57, 255}, color.RGBA{0, 12, 8, 255}
					if layout == backdrop.RGB555 {
						light, dark = color.RGBA{41, 65, 57, 255}, color.RGBA{0, 8, 8, 255}
					}
					if pressed {
						light, dark = dark, light
					}
					painted := 0
					for y := 0; y < got.Bounds().Dy(); y++ {
						for x := 0; x < got.Bounds().Dx(); x++ {
							c := want.RGBAAt(x, y)
							darkPixel := (x == 279 && y >= 174 && y <= 195) || (x == 278 && y >= 173 && y <= 196) || (y == 197 && x >= 202 && x <= 277) || (y == 196 && x >= 201 && x <= 278) || (x == 277 && y == 195)
							lightPixel := (y == 172 && x >= 202 && x <= 277) || (x == 200 && y >= 174 && y <= 195) || (x == 201 && y == 173)
							if darkPixel {
								c = dark
								painted++
							}
							if lightPixel {
								c = light
								painted++
							}
							if disabled && image.Pt(x, y).In(image.Rect(200, 172, 280, 198)) {
								r, g, b := int(c.R)>>3, int(c.G)>>2, int(c.B)>>3
								gm := 63
								if layout == backdrop.RGB555 {
									g, gm = int(c.G)>>3, 31
								}
								if mode == backdrop.Reduced {
									b = (b/8)*8 + 4
								}
								c = color.RGBA{uint8(r * 13 / 16 * 255 / 31), uint8(g * 13 / 16 * 255 / gm), uint8(b * 13 / 16 * 255 / 31), c.A}
							}
							if actual := got.RGBAAt(x, y); actual != c {
								t.Fatalf("layout%d mode%d pressed%v disabled%v pixel%d,%d=%v want%v", layout, mode, pressed, disabled, x, y, actual, c)
							}
						}
					}
					if painted != 299 {
						t.Fatalf("bevel has%d distinct pixels", painted)
					}
				}
			}
		}
	}
}

func TestDialogueDisabledMissionKeepsHoverAndRejectsAdvance(t *testing.T) {
	f := newDialoguePointerFixture(t, "mission")
	v := f.app.flow.viewer
	l := v.noticeLayouts[NoticeDialogue]
	l.ButtonState.Disabled = true
	v.SetNoticeLayouts(l, v.noticeLayouts[NoticeOutcome])
	p := f.button.Min.Add(f.button.Max).Div(2)
	f.pointer(t, "move", p, 0)
	if state := v.noticeLayout().ButtonState; !state.Disabled || !state.Hover || !state.Inside {
		t.Fatalf("disabled hover=%+v", state)
	}
	f.pointer(t, "press", p, 0)
	f.pointer(t, "release", p, 0)
}

func TestDialogueButtonCapturedUnderUsesPackedOverlappingGlyphs(t *testing.T) {
	f := dialogueClaimFont()
	f.Spacing = 0
	glyph := text.Glyph{Width: 4, Height: 1, Advance: 1, Pixels: []text.Pixel{{Level: 15, Painted: true}, {Level: 8, Painted: true}, {Level: 3, Painted: true}, {Level: 0, Painted: true}}}
	for i := range f.Glyphs {
		f.Glyphs[i] = glyph
	}
	l := dialogueClaimLayout().WithPortrait(false)
	l.ButtonLabel = "AA"
	var picture *image.RGBA
	calls := text.Record(func() { picture = RenderNotice(l, f, "", nil) })
	if len(calls) != 4 {
		t.Fatalf("captured%d glyphs want4", len(calls))
	}
	first, second := calls[2], calls[3]
	for n, level := range []int{8, 3, 0} {
		want := color.RGBA{uint8((185 * level / 15 >> 3) * 255 / 31), uint8((159 * level / 15 >> 2) * 255 / 63), uint8((73 * level / 15 >> 3) * 255 / 31), 255}
		if got := second.Under[n]; got != want {
			t.Fatalf("overlap cell%d under=%v wantpacked%v", n, got, want)
		}
		if got := first.NativeColor(uint8(level), n+1); got != want {
			t.Fatalf("firstglyph rastercell%d=%v want%v", n+1, got, want)
		}
	}
	if picture == nil {
		t.Fatal("no reached notice")
	}
}
