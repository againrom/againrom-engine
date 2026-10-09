package ui

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"againrom/pkg/render/text"
)

var dialogContentEdge = color.RGBA{201, 71, 139, 255}

func dialogContentArt() *DialogFrame {
	sizes := [9]image.Point{{96, 64}, {48, 48}, {96, 48}, {48, 48}, {48, 64}, {48, 64}, {48, 48}, {96, 48}, {48, 48}}
	art := &DialogFrame{}
	for i, size := range sizes {
		pic := image.NewRGBA(image.Rectangle{Max: size})
		draw.Draw(pic, pic.Bounds(), &image.Uniform{C: color.RGBA{41, 59, 47, 255}}, image.Point{}, draw.Src)
		for y := 0; y < size.Y; y++ {
			for x := 0; x < size.X; x++ {
				edge := false
				switch i {
				case 1:
					edge = x < 16 || y < 16 || x < 32 && y < 32
				case 2:
					edge = y < 16
				case 3:
					edge = x >= 32 || y < 16 || x >= 16 && y < 32
				case 4:
					edge = x < 16
				case 5:
					edge = x >= 32
				case 6:
					edge = x < 16 || y >= 32 || x < 32 && y >= 16
				case 7:
					edge = y >= 32
				case 8:
					edge = x >= 32 || y >= 32 || x >= 16 && y >= 16
				}
				if edge {
					pic.SetRGBA(x, y, dialogContentEdge)
				}
			}
		}
		art.Pieces[i] = pic
	}
	return art
}

func dialogContentHeadlessFrame(t *testing.T, a *App) *image.RGBA {
	t.Helper()
	pix, note, err := a.HeadlessFrame()
	if err != nil || note != "" || pix.Bounds() != image.Rect(0, 0, 640, 480) {
		t.Fatalf("dialog frame: note=%q error=%v", note, err)
	}
	return pix
}

func assertDialogListBevel(t *testing.T, pix *image.RGBA, calls []widgetCall) {
	t.Helper()
	lists := 0
	for _, call := range calls {
		if call.kind != widgetListBox {
			continue
		}
		lists++
		r := call.rect.Inset(-1)
		dark, light := color.RGBA{8, 8, 8, 255}, color.RGBA{94, 115, 101, 255}
		for x := r.Min.X + 1; x < r.Max.X-1; x++ {
			if got := pix.RGBAAt(x, r.Min.Y); got != dark {
				t.Errorf("list top bevel at %d,%d = %v, want %v", x, r.Min.Y, got, dark)
				break
			}
		}
		for y := r.Min.Y + 1; y < r.Max.Y-1; y++ {
			if got := pix.RGBAAt(r.Min.X, y); got != dark {
				t.Errorf("list left bevel at %d,%d = %v, want %v", r.Min.X, y, got, dark)
				break
			}
		}
		for x := r.Min.X + 1; x < r.Max.X-1; x++ {
			if got := pix.RGBAAt(x, r.Max.Y-1); got != light {
				t.Errorf("list bottom bevel at %d,%d = %v, want %v", x, r.Max.Y-1, got, light)
				break
			}
		}
	}
	if lists != 1 {
		t.Errorf("dialog drew %d lists, want one", lists)
	}
}

func TestSaveAndLoadComposeSunkenListFrames(t *testing.T) {
	for _, screen := range []string{"save", "load"} {
		t.Run(screen, func(t *testing.T) {
			var a *App
			if screen == "save" {
				a, _, _ = saveChooserStyleApp(t, 9)
			} else {
				var loaded []string
				a = loadScrollTestApp(t, loadScrollTestFrames(), &loaded)
			}
			a.SetGameMenuArt(dialogContentArt())
			var pix *image.RGBA
			calls := recordWidgets(t, func() { pix = dialogContentHeadlessFrame(t, a) })
			assertDialogListBevel(t, pix, calls)
		})
	}
}

func assertDialogRectAvoidsFrame(t *testing.T, r image.Rectangle, bare *image.RGBA) {
	t.Helper()
	if !r.In(bare.Bounds()) {
		t.Errorf("content %v leaves frame canvas %v", r, bare.Bounds())
		return
	}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if bare.RGBAAt(x, y) == dialogContentEdge {
				t.Errorf("content %v crosses ornate frame at %d,%d", r, x, y)
				return
			}
		}
	}
}

func TestSaveButtonsKeepClearOfTheOrnateFrame(t *testing.T) {
	for _, state := range []string{"save", "overwrite", "delete"} {
		t.Run(state, func(t *testing.T) {
			a, _, _ := saveChooserStyleApp(t, 1)
			art := dialogContentArt()
			a.SetGameMenuArt(art)
			switch state {
			case "overwrite":
				a.flow.saveDialog.prepared = &PreparedSave{Paths: []string{"saves/slot.sav"}, Existing: []string{"saves/slot.sav"}}
			case "delete":
				a.flow.saveDialog.remove = func() error { return nil }
				a.flow.saveDialog.removePath = "saves/slot.sav"
			}
			bare := image.NewRGBA(image.Rect(0, 0, 640, 480))
			art.Draw(bare, image.Rect(8, 0, 632, 480))
			calls := recordWidgets(t, func() { dialogContentHeadlessFrame(t, a) })
			buttons := 0
			for _, call := range calls {
				if call.kind == widgetPushButton && call.rect.Min.Y >= 400 {
					buttons++
					assertDialogRectAvoidsFrame(t, call.rect, bare)
				}
			}
			if buttons < 2 {
				t.Errorf("Save %s drew %d action buttons, want at least two", state, buttons)
			}
			if state == "save" {
				row := saveControlRect(saveDeleteControl).Union(saveControlRect(saveWriteControl)).Union(saveControlRect(saveCancelControl))
				if left, right := row.Min.X-8, 632-row.Max.X; left != right {
					t.Errorf("Save action row margins = %d,%d, want equal margins", left, right)
				}
			}
		})
	}
}

func assertDialogGlyphsAvoidFrame(t *testing.T, calls []text.DrawCall, bare *image.RGBA) {
	t.Helper()
	for _, call := range calls {
		if call.Glyph == nil || call.Glyph.Width == 0 {
			continue
		}
		for i, pixel := range call.Glyph.Pixels {
			p := image.Pt(call.X+i%call.Glyph.Width, call.Y+i/call.Glyph.Width)
			if !pixel.Painted || !call.Clip.Empty() && !p.In(call.Clip) {
				continue
			}
			if !p.In(bare.Bounds()) || bare.RGBAAt(p.X, p.Y) == dialogContentEdge {
				t.Errorf("glyph or shadow crosses frame at %v", p)
				return
			}
		}
	}
}

func TestGameOptionsContentHasEqualMarginsAndClearsItsFrame(t *testing.T) {
	values := GameOptionValues{}
	var log []optionWrite
	a := optionsApp(t, &values, &log, nil)
	a.flow.menuFont = solidFont15()
	art := dialogContentArt()
	a.SetGameMenuArt(art)
	bare := image.NewRGBA(image.Rect(0, 0, 640, 480))
	body := image.Rect(76, 28, 556, 444)
	art.Draw(bare, body)
	var calls []widgetCall
	glyphs := text.Record(func() {
		calls = recordWidgets(t, func() {
			if pix, err := a.HeadlessGameOptionsFrame(); err != nil || pix == nil {
				t.Fatalf("options frame: %v", err)
			}
		})
	})
	content := image.Rectangle{}
	for _, call := range calls {
		if call.kind == widgetFrame {
			continue
		}
		content = content.Union(call.rect)
		assertDialogRectAvoidsFrame(t, call.rect.Union(call.rect.Add(image.Pt(4, 4))), bare)
	}
	if content.Empty() {
		t.Fatal("options drew no widgets")
	}
	if left, right := content.Min.X-body.Min.X, body.Max.X-content.Max.X; left != right {
		t.Errorf("options content margins = %d,%d, want equal margins", left, right)
	}
	buttons := gameOptionRect(gameMenuPageReturn).Union(gameOptionRect(gameMenuOptionsCancel))
	if delta := buttons.Min.X + buttons.Max.X - body.Min.X - body.Max.X; delta < -1 || delta > 1 {
		t.Errorf("options button pair centre differs from frame centre by %d half-pixels", delta)
	}
	title := image.Rectangle{}
	for _, call := range glyphs {
		if call.Y == body.Min.Y+20 && call.Glyph != nil {
			title = title.Union(image.Rect(call.X, call.Y, call.X+call.Glyph.Width, call.Y+call.Glyph.Height))
		}
	}
	if title.Empty() {
		t.Error("options title was not drawn")
	} else if delta := title.Min.X + title.Max.X - body.Min.X - body.Max.X; delta < -1 || delta > 1 {
		t.Errorf("options title centre differs from frame centre by %d half-pixels", delta)
	}
	assertDialogGlyphsAvoidFrame(t, glyphs, bare)
}
