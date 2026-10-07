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

func TestTownDialogueRemapsTheReachedRoom(t *testing.T) {
	town := &fakeShopDialogueTown{pic: image.NewRGBA(image.Rect(0, 0, 240, 120))}
	room, err := composeTownRoom(town, "", image.Point{}, false, TownSurfaceControl{}, nil, false, 0, nil, ShopControl{}, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ComposeTownScreen(town, "")
	if err != nil {
		t.Fatal(err)
	}
	p := image.Pt(8, 8)
	c := room.RGBAAt(p.X, p.Y)
	want := color.RGBA{uint8((int(c.R) >> 3) * 13 / 16 * 255 / 31), uint8((int(c.G) >> 2) * 13 / 16 * 255 / 63), uint8((int(c.B) >> 3) * 13 / 16 * 255 / 31), c.A}
	if got.RGBAAt(p.X, p.Y) != want {
		t.Fatalf("room behind dialogue at %v = %v; packed remap wants %v from %v", p, got.RGBAAt(p.X, p.Y), want, c)
	}
}

type repeatedBackdropTown struct {
	fakeShopDialogueTown
	shows uint64
}

func (s *repeatedBackdropTown) TownDialogueShows() uint64 {
	if s.pic == nil {
		return 0
	}
	return s.shows
}

func TestTownBackdropShowPageCloseAndSubmission(t *testing.T) {
	town := &repeatedBackdropTown{fakeShopDialogueTown: fakeShopDialogueTown{pic: image.NewRGBA(image.Rect(0, 0, 240, 120))}, shows: 1}
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	a.flow.showTown("")
	a.Layout(1920, 1080)
	a.SetTextSmoothing(false)
	first, _, err := a.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	page, _, err := a.HeadlessFrame()
	if err != nil || !bytes.Equal(first.Pix, page.Pix) {
		t.Fatal("ordinary repaint compounded")
	}
	for _, depth := range []uint64{1, 2} {
		town.shows = depth
		cpu, _, err := a.HeadlessFrame()
		if err != nil {
			t.Fatal(err)
		}
		a.Draw(ebiten.NewImage(1920, 1080))
		for y := 0; y < 480; y += 17 {
			for x := 0; x < 640; x += 19 {
				got, known := a.DialogueBackdropPixel(x, y)
				if !known || got != cpu.RGBAAt(x, y) {
					t.Fatalf("depth %d submission at %d,%d=%v/%v; CPU=%v", depth, x, y, got, known, cpu.RGBAAt(x, y))
				}
			}
		}
		if depth == 2 && bytes.Equal(first.Pix, cpu.Pix) {
			t.Fatal("explicit show became idempotent")
		}
	}
	town.pic = nil
	closed, _, err := a.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	room, err := a.composeTownRoom()
	if err != nil || !bytes.Equal(room.Pix, closed.Pix) {
		t.Fatal("closed room kept stale remap")
	}
	if bytes.Equal(first.Pix, closed.Pix) {
		t.Fatal("omitted show/close loss control insensitive")
	}
}

func TestDetachedBackdropCoversInsertedColumnsAndKeepsModal(t *testing.T) {
	submissions := 0
	oldSubmit := submitDialogueBackdrop
	t.Cleanup(func() { submitDialogueBackdrop = oldSubmit })
	submitDialogueBackdrop = func(dst *ebiten.Image, vertices []ebiten.Vertex, shader *ebiten.Shader, op *ebiten.DrawTrianglesShaderOptions) {
		submissions++
		if dst.Bounds() != image.Rect(0, 0, 960, 480) || len(vertices) != 4 || vertices[3].SrcX != 960 || vertices[3].SrcY != 480 || op.Images[0] == nil || op.Images[1] == nil || op.Images[1].Bounds() != image.Rect(0, 0, 256, 256) || op.Blend != ebiten.BlendCopy || op.Uniforms["GreenBits"] != float32(5) {
			t.Fatal("invalid packed GPU submission")
		}
		oldSubmit(dst, vertices, shader, op)
	}
	room := image.NewRGBA(image.Rect(0, 0, 640, 480))
	for y := 0; y < 480; y++ {
		for x := 0; x < 640; x++ {
			room.SetRGBA(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 230, 255})
		}
	}
	layout := newWideFrameLayout(960, wideFrameTownColumns)
	wide := layout.compose(room)
	dialogue := image.NewRGBA(image.Rect(0, 0, 580, 240))
	dialogue.SetRGBA(0, 0, color.RGBA{240, 10, 40, 255})
	s := dialogueBackdropState{policy: DialogueBackdrop{Layout: backdrop.RGB555, Mode: backdrop.Reduced}}
	cpu := layout.compose(room)
	s.apply(cpu, 2)
	overlayDetachedTownDialogue(cpu, dialogue)
	tex := ebiten.NewImageFromImage(wide)
	var log pixelLog
	log.reset(wide.Bounds())
	log.upload(wide)
	s.draw(tex, &log, 2)
	if submissions != 2 {
		t.Fatal("omitted/repeated GPU show", submissions)
	}
	paintDetachedTownDialogue(tex, dialogue, &log)
	changed := 0
	for y := 0; y < 480; y += 11 {
		for x := 0; x < 960; x += 13 {
			c, ok := log.value(len(log.ops), x, y)
			if !ok || c != cpu.RGBAAt(x, y) {
				t.Fatalf("detached submission disagrees at %d,%d", x, y)
			}
			if x >= 640 && c != wide.RGBAAt(x, y) {
				changed++
			}
		}
	}
	if changed < 100 {
		t.Fatal("detached omission control has insufficient changed inserted pixels", changed)
	}
	if cpu.RGBAAt(190, 120) != dialogue.RGBAAt(0, 0) {
		t.Fatal("modal itself was remapped or moved")
	}
}

type rowBackdropTown struct {
	fakeTown
	pic *image.RGBA
}

func (s *rowBackdropTown) TownDialogue() (*image.RGBA, bool) { return s.pic, s.pic != nil }
func (s *rowBackdropTown) TownDialogueButton() (image.Rectangle, bool) {
	return image.Rectangle{}, false
}
func (s *rowBackdropTown) AdvanceTownDialogue() TownAction { s.pic = nil; return TownAction{} }

func TestRowListBackdropIsReachedAndCloses(t *testing.T) {
	town := &rowBackdropTown{pic: image.NewRGBA(image.Rect(0, 0, 240, 120))}
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	a.flow.showTown("")
	a.Layout(640, 480)
	a.SetTextSmoothing(false)
	if pix, err := a.composeTownRoom(); err != nil || pix.RGBAAt(639, 479) != pickerBackground {
		t.Fatal("shared row-list backdrop unavailable", err)
	}
	a.Draw(ebiten.NewImage(640, 480))
	got, ok := a.DialogueBackdropPixel(639, 479)
	l, _ := backdrop.New(backdrop.RGB565, backdrop.Full)
	want := l.Color(pickerBackground)
	if !ok || got != want {
		t.Fatal("row fallback omitted backdrop", got, ok, want)
	}
	town.pic = nil
	a.Draw(ebiten.NewImage(640, 480))
	got, ok = a.DialogueBackdropPixel(639, 479)
	if !ok || got != pickerBackground {
		t.Fatal("row fallback close retained backdrop", got, ok)
	}
}

func TestBackdropKeepsMethodCSmoothingWithPartialClip(t *testing.T) {
	room := image.NewRGBA(image.Rect(0, 0, 64, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 64; x++ {
			room.SetRGBA(x, y, color.RGBA{124, 90, 201, 255})
		}
	}
	f := panelFont()
	calls := text.Record(func() { f.Draw(room, "AAA", 2, 3, color.RGBA{220, 175, 70, 255}) })
	l, _ := backdrop.New(backdrop.RGB565, backdrop.Full)
	clip := image.Rect(3, 2, 18, 16)
	var log pixelLog
	log.reset(room.Bounds())
	log.upload(room)
	// Preserve the log's immutable upload before CPU remapping.
	copyRoom := &image.RGBA{Pix: bytes.Clone(room.Pix), Stride: room.Stride, Rect: room.Rect}
	l.Apply(copyRoom, copyRoom.Bounds(), clip, 2)
	log.remap(clip, l, 2)
	remapCaptured(calls, clip, l, 2)
	kept, certain := textsmooth.Decide(calls, room.Bounds(), log.verdict)
	if !certain || len(kept) != len(calls) {
		t.Fatalf("method C lost remapped glyphs: %d of %d, certain %v", len(kept), len(calls), certain)
	}
	out := image.NewRGBA(image.Rect(0, 0, 128, 64))
	textsmooth.Composite(out, kept, 2, 0, 0)
	if bytes.Count(out.Pix, []byte{0}) == len(out.Pix) {
		t.Fatal("smoothed remapped overlay is empty")
	}
	for y := 0; y < 32; y++ {
		for x := 0; x < 64; x++ {
			c, ok := log.value(len(log.ops), x, y)
			if !ok || c != copyRoom.RGBAAt(x, y) {
				t.Fatal("clip remap and log differ")
			}
		}
	}
}

func TestMissionBackdropSeparatesShowFromPageAndCustomAlpha(t *testing.T) {
	v := backdropViewer(t)
	v.SetDialogue(Dialogue{Text: "one"})
	_, n, r := v.DialogueBackdropPlan()
	if n != 1 || r != image.Rect(0, 0, v.frameW, v.frameH) {
		t.Fatal("mission show omitted full frame", n, r)
	}
	v.PageDialogue(Dialogue{Text: "two"})
	_, n, _ = v.DialogueBackdropPlan()
	if n != 1 {
		t.Fatal("page remapped")
	}
	v.SetDialogue(Dialogue{Text: "two"})
	_, n, _ = v.DialogueBackdropPlan()
	if n != 2 {
		t.Fatal("explicit second show did not compound")
	}
	v.SetNoticeBackdrop(color.RGBA{22, 33, 44, 99})
	_, n, _ = v.DialogueBackdropPlan()
	if n != 0 {
		t.Fatal("custom alpha consumer changed")
	}
	if _, c, ok := v.noticeBackdropOf(); !ok || c != (color.RGBA{22, 33, 44, 99}) {
		t.Fatal("custom alpha contract lost")
	}
	v.SetDialogueBackdrop(DialogueBackdrop{Clipped: true, Clip: image.Rect(2, 3, 10, 11)})
	_, n, r = v.DialogueBackdropPlan()
	if n != 2 || r != image.Rect(2, 3, 10, 11) {
		t.Fatal("incoming clip reset", n, r)
	}
	v.ClearNotice()
	_, n, _ = v.DialogueBackdropPlan()
	if n != 0 {
		t.Fatal("close retained remap")
	}
}

func TestBackdropSmoothingCacheUpdatesAfterAnotherShow(t *testing.T) {
	room := image.NewRGBA(image.Rect(0, 0, 64, 32))
	f := panelFont()
	calls := text.Record(func() { f.Draw(room, "AAA", 2, 3, color.RGBA{220, 175, 70, 255}) })
	l, _ := backdrop.New(backdrop.RGB565, backdrop.Full)
	remapCaptured(calls, room.Bounds(), l, 1)
	var overlay textOverlay
	canvas := ebiten.NewImage(128, 64)
	overlay.draw(canvas, calls, 2, 0, 0)
	first := bytes.Clone(overlay.buf.Pix)
	remapCaptured(calls, room.Bounds(), l, 1)
	if sameCalls(overlay.calls, calls, false) {
		t.Fatal("second-show loss control retained the first palette")
	}
	overlay.draw(canvas, calls, 2, 0, 0)
	if bytes.Equal(first, overlay.buf.Pix) {
		t.Fatal("method C reused first-show colours")
	}
}
