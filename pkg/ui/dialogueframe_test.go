package ui

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/backdrop"
	"againrom/pkg/render/text"
	"againrom/pkg/render/textsmooth"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestDialogueShadowPackedLevelSixPreservesSkipsAndPadding(t *testing.T) {
	dst := &image.RGBA{Pix: bytes.Repeat([]byte{0xa7}, 80), Stride: 24, Rect: image.Rect(10, 20, 14, 23)}
	marker := color.RGBA{248, 252, 248, 255}
	for y := 20; y < 23; y++ {
		for x := 10; x < 14; x++ {
			dst.SetRGBA(x, y, marker)
		}
	}
	before := bytes.Clone(dst.Pix)
	mask := image.NewRGBA(image.Rect(4, 5, 7, 7))
	mask.SetRGBA(4, 5, color.RGBA{1, 127, 255, 255})
	mask.SetRGBA(6, 5, color.RGBA{255, 1, 127, 255})
	mask.SetRGBA(5, 6, color.RGBA{127, 255, 1, 255})
	stampShadow(dst, mask, image.Pt(9, 20))
	want := color.RGBA{156, 157, 156, 255}
	for y := 20; y < 23; y++ {
		for x := 10; x < 14; x++ {
			expected := marker
			if (x == 11 && y == 20) || (x == 10 && y == 21) {
				expected = want
			}
			if got := dst.RGBAAt(x, y); got != expected {
				t.Fatalf("mask cell %d,%d = %v, want %v", x, y, got, expected)
			}
		}
	}
	for i := range before {
		row, col := i/24, i%24
		if row >= 3 || col >= 16 {
			if before[i] != dst.Pix[i] {
				t.Fatalf("guard %d changed", i)
			}
		}
	}
}

func frameWordExpectation(p uint16, layout backdrop.Layout, mode backdrop.Mode) uint16 {
	greenBits := 6
	if layout == backdrop.RGB555 {
		greenBits = 5
		p &= 0x7fff
	}
	r := int(p>>(5+greenBits)) & 31
	g := int(p>>5) & ((1 << greenBits) - 1)
	b := int(p) & 31
	if mode == backdrop.Reduced {
		b = (b &^ 7) + 4
	}
	return uint16((r*10/16)<<(5+greenBits) | (g*10/16)<<5 | b*10/16)
}

func TestDialogueShadowFullAndReducedWholeWordPopulations(t *testing.T) {
	for _, layout := range []backdrop.Layout{backdrop.RGB565, backdrop.RGB555} {
		for _, mode := range []backdrop.Mode{backdrop.Full, backdrop.Reduced} {
			l, err := backdrop.NewLevel(layout, mode, 6)
			if err != nil {
				t.Fatal(err)
			}
			// RGB555 high-bit words exercise the supplied mask policy.
			for p := 0; p < 65536; p++ {
				if got, want := l.Word(uint16(p)), frameWordExpectation(uint16(p), layout, mode); got != want {
					t.Fatalf("layout %d mode %d word %04x=%04x want %04x", layout, mode, p, got, want)
				}
			}
		}
	}
}

func TestDialogueFrameRequestsReplacementAndIncomingClip(t *testing.T) {
	art := dialogueClaimFrame()
	body := image.Rect(12, 17, 492, 241)
	tiles, shadows := art.dialogueTiles(body), art.dialogueShadows(body)
	if len(tiles) != 24 || len(shadows) != 9 {
		t.Fatal("request counts", len(tiles), len(shadows))
	}
	wantPieces := []int{3, 6, 8, 7, 7, 7, 7, 5, 5}
	for i, tile := range shadows {
		if tile.piece != wantPieces[i] {
			t.Fatal("retained geometry order", i, tile)
		}
	}
	marker := color.RGBA{248, 252, 248, 255}
	dst := image.NewRGBA(image.Rect(0, 0, 512, 256))
	for y := 0; y < 256; y++ {
		for x := 0; x < 512; x++ {
			dst.SetRGBA(x, y, marker)
		}
	}
	art.Pieces[1] = image.NewRGBA(image.Rect(4, 5, 52, 53))
	art.Pieces[1].SetRGBA(4, 5, color.RGBA{A: 255})
	art.Pieces[1].SetRGBA(6, 5, color.RGBA{40, 20, 10, 80})
	clip := image.Rect(12, 17, 16, 18)
	art.drawDialogueBodyWithPolicy(dst, body, DialogueBackdrop{FrameClipped: true, FrameClip: clip})
	if got := dst.RGBAAt(12, 17); got != (color.RGBA{A: 255}) {
		t.Fatal("palette zero did not replace", got)
	}
	if got := dst.RGBAAt(13, 17); got != marker {
		t.Fatal("skip did not preserve", got)
	}
	if got := dst.RGBAAt(14, 17); got != (color.RGBA{40, 20, 10, 80}) {
		t.Fatal("normal literal blended instead of replacing", got)
	}
	if got := dst.RGBAAt(20, 20); got != marker {
		t.Fatal("incoming clip was ignored", got)
	}
}

func TestDialogueSceneShadowUsesRemappedRoomAndOrnamentalSkips(t *testing.T) {
	l := dialogueClaimLayout()
	l.Frame = dialogueClaimFrame()
	l.Button, l.Portrait = image.Rectangle{}, image.Rectangle{}
	// Only one covered pixel on the top-right piece reaches the outer band.
	clear(l.Frame.Pieces[3].Pix)
	l.Frame.Pieces[3].SetRGBA(47, 12, color.RGBA{A: 255})
	room := image.NewRGBA(image.Rect(0, 0, 640, 480))
	marker := color.RGBA{248, 252, 248, 255}
	for y := 0; y < 480; y++ {
		for x := 0; x < 640; x++ {
			room.SetRGBA(x, y, marker)
		}
	}
	s := dialogueBackdropState{}
	s.apply(room, 1)
	before := room.RGBAAt(563, 144)
	ComposeDialogueNotice(room, l, dialogueClaimFont(), "", nil, l.Box.Min)
	// top-right at x432, +8 shadow, source x47 => local487.
	got := room.RGBAAt(l.Box.Min.X+487, l.Box.Min.Y+20)
	want := color.RGBA{123, 125, 123, 255}
	if got != want {
		t.Fatalf("packed reached shadow=%v want %v (washed room %v)", got, want, before)
	}
	if skipped := room.RGBAAt(l.Box.Min.X+486, l.Box.Min.Y+20); skipped != before {
		t.Fatal("ornamental skip filled", skipped, before)
	}
	if before == want {
		t.Fatal("omitted shadow loss control insensitive")
	}
	// Empty background clip keeps the modal; frame clip is a separate seam.
	l.DialogueBackdrop = DialogueBackdrop{Clipped: true}
	clear(room.Pix)
	ComposeDialogueNotice(room, l, dialogueClaimFont(), "", nil, l.Box.Min)
	if room.RGBAAt(l.Box.Min.X, l.Box.Min.Y).A == 0 {
		t.Fatal("background clip suppressed frame")
	}
}

func TestDialogueShadowSubmissionLogAndMethodCCapture(t *testing.T) {
	room := image.NewRGBA(image.Rect(0, 0, 640, 480))
	for y := 0; y < 480; y++ {
		for x := 0; x < 640; x++ {
			room.SetRGBA(x, y, color.RGBA{40, 80, 120, 255})
		}
	}
	f := dialogueClaimFont()
	calls := text.Record(func() { f.Draw(room, "AAA", 560, 140, color.RGBA{248, 252, 248, 255}) })
	art := dialogueClaimFrame()
	body := image.Rect(0, 0, 480, 224)
	s := dialogueBackdropState{policy: DialogueBackdrop{Layout: backdrop.RGB555, Mode: backdrop.Reduced, FrameClipped: true, FrameClip: image.Rect(560, 135, 565, 150)}}
	tex := ebiten.NewImageFromImage(room)
	var log pixelLog
	log.reset(room.Bounds())
	log.upload(room)
	cpu := &image.RGBA{Pix: bytes.Clone(room.Pix), Stride: room.Stride, Rect: room.Rect}
	gotCalls := append([]text.DrawCall(nil), calls...)
	submissions := 0
	old := submitDialogueShadow
	t.Cleanup(func() { submitDialogueShadow = old })
	submitDialogueShadow = func(dst *ebiten.Image, vertices []ebiten.Vertex, shader *ebiten.Shader, op *ebiten.DrawTrianglesShaderOptions) {
		submissions++
		if dst.Bounds() != s.policy.FrameClip.Intersect(image.Rect(84, 132, 564, 356)) || op.Images[0] == nil || op.Images[1] == nil || op.Images[2] == nil || op.Uniforms["GreenBits"] != float32(5) {
			t.Fatal("masked packed submission omitted")
		}
		old(dst, vertices, shader, op)
	}
	s.drawFrameShadows(tex, &log, art, body, image.Pt(76, 124), 1, gotCalls)
	s.applyFrameShadows(cpu, art, body, image.Pt(76, 124), nil)
	if submissions != 1 {
		t.Fatal("shadow not submitted", submissions)
	}
	for y := 130; y < 155; y++ {
		for x := 555; x < 570; x++ {
			got, ok := log.value(len(log.ops), x, y)
			if !ok || got != cpu.RGBAAt(x, y) {
				t.Fatalf("log/CPU %d,%d=%v/%v want %v", x, y, got, ok, cpu.RGBAAt(x, y))
			}
		}
	}
	if len(gotCalls) == 0 || len(gotCalls[0].RasterColors) == 0 || gotCalls[0].RasterColors[0] == calls[0].NativeColor(text.MaxLevel, 0) {
		t.Fatal("captured glyph did not receive packed shadow")
	}
	if log.verdict(gotCalls[0].X, gotCalls[0].Y, gotCalls[0].NativeColor(text.MaxLevel, 0)) != textsmooth.Matches {
		t.Fatal("method C cannot classify shadowed ink")
	}
	kept, certain := textsmooth.Decide(gotCalls, room.Bounds(), log.verdict)
	if !certain || len(kept) != len(gotCalls) {
		t.Fatal("method C lost partial-clip glyph cells", len(kept), len(gotCalls), certain)
	}
	if len(gotCalls[0].Under) == 0 || gotCalls[0].Under[0] == calls[0].Under[0] {
		t.Fatal("shadow did not remap glyph underlay")
	}
}

type frameDialogueTown struct {
	fakeShopDialogueTown
	layout NoticeLayout
}

func (s *frameDialogueTown) TownDialogue() (*image.RGBA, bool) {
	if s.pic == nil {
		return nil, false
	}
	return RenderNotice(s.layout, dialogueClaimFont(), "", nil), true
}

func (s *frameDialogueTown) TownDialogueFrame() (*DialogFrame, image.Rectangle) {
	return s.layout.Frame, image.Rectangle{Max: s.layout.Box.Size().Sub(image.Pt(8, 8))}
}

func TestDialogueFrameReachedNativeAndDetachedTownCPUAndSubmission(t *testing.T) {
	for _, width := range []int{640, 960} {
		name := "native"
		if width == 960 {
			name = "detached"
		}
		t.Run(name, func(t *testing.T) {
			l := dialogueClaimLayout()
			l.Frame = dialogueClaimFrame()
			l.Button, l.Portrait = image.Rectangle{}, image.Rectangle{}
			town := &frameDialogueTown{fakeShopDialogueTown: fakeShopDialogueTown{pic: image.NewRGBA(image.Rect(0, 0, 488, 232))}, layout: l}
			a := newTestApp(t, appRows(3), okLoader(t))
			a.SetTown(town)
			a.flow.showTown("")
			a.Layout(width, 480)
			a.SetTextSmoothing(false)
			cpu, _, err := a.HeadlessFrame()
			if err != nil {
				t.Fatal(err)
			}
			art := l.Frame
			// A second scene with empty source masks is the omitted-shadow control.
			for _, p := range []int{3, 5, 6, 7, 8} {
				clear(art.Pieces[p].Pix)
			}
			without, _, err := a.HeadlessFrame()
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Equal(cpu.Pix, without.Pix) {
				t.Fatal("reached shadow omission insensitive")
			}
			band := image.Pt((cpu.Rect.Dx()-488)/2+487, 144)
			if cpu.RGBAAt(band.X, band.Y) == without.RGBAAt(band.X, band.Y) {
				t.Fatal("outer-band omitted-shadow control insensitive", band)
			}
			l.Frame = dialogueClaimFrame()
			town.layout = l
			a.Draw(ebiten.NewImage(width, 480))
			for y := 0; y < 480; y += 13 {
				for x := 0; x < cpu.Rect.Dx(); x += 11 {
					got, known := a.DialogueBackdropPixel(x, y)
					if !known || got != cpu.RGBAAt(x, y) {
						t.Fatalf("town width%d CPU/submission %d,%d=%v/%v want%v", width, x, y, got, known, cpu.RGBAAt(x, y))
					}
				}
			}
		})
	}
}

func TestDialogueFrameReachedMissionCacheAndCustomWash(t *testing.T) {
	v := backdropViewer(t)
	layoutViewport(v, 640, 480)
	v.SetDialogFrame(dialogueClaimFrame())
	v.SetDialogue(Dialogue{Text: "AAA"})
	room := image.NewRGBA(image.Rect(0, 0, 640, 480))
	for y := 0; y < 480; y++ {
		for x := 0; x < 640; x++ {
			room.SetRGBA(x, y, color.RGBA{248, 252, 248, 255})
		}
	}
	tex := ebiten.NewImageFromImage(room)
	v.canvasLog.reset(room.Bounds())
	v.canvasLog.upload(room)
	v.dialogueBackdrop.draw(tex, &v.canvasLog, 1)
	v.paintNotice(tex)
	got, known := v.DialogueBackdropPixel(563, 144)
	if !known || got != (color.RGBA{123, 125, 123, 255}) {
		t.Fatal("mission scene shadow absent", got, known)
	}
	builds := v.noticeBuilds
	tex.WritePixels(room.Pix)
	v.canvasLog.reset(room.Bounds())
	v.canvasLog.upload(room)
	v.dialogueBackdrop.draw(tex, &v.canvasLog, 1)
	v.paintNotice(tex)
	if v.noticeBuilds != builds {
		t.Fatal("repaint rebuilt immutable modal")
	}
	got, known = v.DialogueBackdropPixel(563, 144)
	if !known || got != (color.RGBA{123, 125, 123, 255}) {
		t.Fatal("cached image lost scene shadow", got, known)
	}
	if err := v.SetDialogueBackdrop(DialogueBackdrop{Layout: backdrop.RGB555, Mode: backdrop.Reduced}); err != nil {
		t.Fatal(err)
	}
	tex.WritePixels(room.Pix)
	v.canvasLog.reset(room.Bounds())
	v.canvasLog.upload(room)
	v.paintNotice(tex)
	if v.noticeBuilds != builds+1 {
		t.Fatal("packed policy retained stale modal")
	}
	got, known = v.DialogueBackdropPixel(563, 144)
	if !known || got != (color.RGBA{156, 156, 139, 255}) {
		t.Fatal("reduced RGB555 policy did not reach scene", got, known)
	}
}

func TestDialogueShadowFoldsPrecedingWashBeforePackedLookup(t *testing.T) {
	room := image.NewRGBA(image.Rect(0, 0, 640, 480))
	f := dialogueClaimFont()
	calls := text.Record(func() { f.Draw(room, "AAA", 560, 140, color.RGBA{255, 255, 255, 255}) })
	var log pixelLog
	log.reset(room.Bounds())
	log.upload(room)
	wash := color.RGBA{20, 30, 40, 85}
	log.overSolid(room.Bounds(), wash)
	tex := ebiten.NewImageFromImage(room)
	s := dialogueBackdropState{}
	s.drawFrameShadows(tex, &log, dialogueClaimFrame(), image.Rect(0, 0, 480, 224), image.Pt(76, 124), 1, calls)
	got := calls[0].NativeColor(text.MaxLevel, 0)
	want, known := log.value(len(log.ops), 560, 140)
	if !known || got != want || got == (color.RGBA{156, 157, 156, 255}) {
		t.Fatal("pre-shadow wash was not folded", got, want, known)
	}
	if log.verdict(560, 140, got) != textsmooth.Matches {
		t.Fatal("method C lost tinted shadow ink")
	}
}

func TestDialogueShadowMaskNearestDownscaleKeepsSelectedLiteral(t *testing.T) {
	art := dialogueClaimFrame()
	for _, pic := range art.Pieces {
		clear(pic.Pix)
	}
	art.Pieces[3].SetRGBA(45, 11, color.RGBA{A: 255})
	mask := dialogueShadowMask(art, image.Rect(0, 0, 480, 224), image.Pt(76, 124), 0.5, image.Rect(0, 0, 640, 480))
	if mask == nil || mask.RGBAAt(318, 133).A != 1 {
		t.Fatal("nearest pixel-centre selection lost literal")
	}
	count := 0
	for i := 3; i < len(mask.Pix); i += 4 {
		count += int(mask.Pix[i])
	}
	if count != 1 {
		t.Fatal("scaled skip silhouette became rectangle", count)
	}
}

type frameRowDialogueTown struct {
	rowBackdropTown
	layout NoticeLayout
}

func (s *frameRowDialogueTown) TownDialogue() (*image.RGBA, bool) {
	return RenderNotice(s.layout, dialogueClaimFont(), "", nil), s.pic != nil
}

func (s *frameRowDialogueTown) TownDialogueFrame() (*DialogFrame, image.Rectangle) {
	return s.layout.Frame, image.Rectangle{Max: s.layout.Box.Size().Sub(image.Pt(8, 8))}
}

func TestDialogueFrameReachedTownRowFallback(t *testing.T) {
	l := dialogueClaimLayout()
	l.Frame = dialogueClaimFrame()
	l.Button, l.Portrait = image.Rectangle{}, image.Rectangle{}
	town := &frameRowDialogueTown{rowBackdropTown: rowBackdropTown{pic: image.NewRGBA(image.Rect(0, 0, 488, 232))}, layout: l}
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	a.flow.showTown("")
	a.Layout(640, 480)
	a.SetTextSmoothing(false)
	if pix, err := a.composeTownRoom(); err != nil || pix.RGBAAt(563, 144) != pickerBackground {
		t.Fatal("shared row-list backdrop unavailable", err)
	}
	a.Draw(ebiten.NewImage(640, 480))
	got, known := a.DialogueBackdropPixel(563, 144)
	washed, _ := backdrop.New(backdrop.RGB565, backdrop.Full)
	c := washed.Color(pickerBackground)
	p := uint16(int(c.R)>>3<<11 | int(c.G)>>2<<5 | int(c.B)>>3)
	w := frameWordExpectation(p, backdrop.RGB565, backdrop.Full)
	want := color.RGBA{uint8(int(w>>11) * 255 / 31), uint8(int(w>>5&63) * 255 / 63), uint8(int(w&31) * 255 / 31), 255}
	if !known || got != want {
		t.Fatal("row fallback shadow absent", got, known, want)
	}
	if got == c {
		t.Fatal("row shadow omission insensitive")
	}
}
