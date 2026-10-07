package ui

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"reflect"
	"testing"

	"againrom/pkg/render/text"
	"github.com/hajimehoshi/ebiten/v2"
)

type editorTextEdits struct{ moves, undo, redo, writes int }

func (*editorTextEdits) Dirty() bool   { return false }
func (*editorTextEdits) CanUndo() bool { return false }
func (*editorTextEdits) CanRedo() bool { return false }
func (p *editorTextEdits) MoveUnit(int, image.Point) (*InspectionDocument, error) {
	p.moves++
	return nil, nil
}
func (p *editorTextEdits) Undo() (*InspectionDocument, error) { p.undo++; return nil, nil }
func (p *editorTextEdits) Redo() (*InspectionDocument, error) { p.redo++; return nil, nil }
func (p *editorTextEdits) SaveAs(string) (string, error) {
	p.writes++
	return "", errors.New("writer must not run during modal presentation")
}

func selectedTextEditor(t *testing.T, font *text.Font, size image.Point) (*MapEditor, *editorTextEdits) {
	t.Helper()
	edits := &editorTextEdits{}
	doc := &InspectionDocument{Viewer: litViewer(t), Source: "fixture-map", SourceBytes: []byte("synthetic authored map"),
		Records: []InspectionRecord{{Kind: "Unit", Label: "selected unit", Details: []string{"detail behind modal", "second covered detail"}}}, Edits: edits}
	e := NewMapEditor(font, nil, func(source string) (*InspectionDocument, error) {
		if source != "fixture-map" {
			return nil, errors.New("refused map; selected record retained")
		}
		return doc, nil
	})
	e.Layout(size.X, size.Y)
	if err := e.Open("fixture-map"); err != nil {
		t.Fatal(err)
	}
	if !e.Select(0, false) {
		t.Fatal("selected record not admitted")
	}
	t.Cleanup(func() {
		if e.panelImage != nil {
			e.panelImage.Dispose()
		}
		if e.textOverlay.tex != nil {
			e.textOverlay.tex.Dispose()
		}
		if e.doc.Viewer.canvas != nil {
			disposeFrameCanvas(e.doc.Viewer.canvas)
		}
	})
	return e, edits
}

func assertEditorPanelUpload(t *testing.T, e *MapEditor, h int) (captured, kept, partial int) {
	t.Helper()
	pic, blit, calls, showing := e.panelPicture(h)
	if !bytes.Equal(pic.Pix, e.PanelImage(h).Pix) {
		t.Fatal("capture changed the final native panel")
	}
	erased := make(map[image.Point]color.RGBA)
	for _, c := range showing {
		shown, covered := 0, 0
		for n, px := range c.Glyph.Pixels {
			at := image.Pt(c.X+n%c.Glyph.Width, c.Y+n/c.Glyph.Width)
			if !px.Painted || !at.In(pic.Bounds()) || !at.In(c.Clip) {
				continue
			}
			if c.Mask != nil && !c.Mask[n] {
				covered++
				continue
			}
			shown++
			if _, earlier := erased[at]; !earlier {
				erased[at] = c.Under[n]
			}
		}
		if shown > 0 && covered > 0 {
			partial++
		}
	}
	for y := 0; y < h; y++ {
		for x := 0; x < editorRailWidth; x++ {
			at := image.Pt(x, y)
			want, showing := erased[at]
			if !showing {
				want = pic.RGBAAt(x, y)
			}
			if got := blit.RGBAAt(x, y); got != want {
				t.Fatalf("panel upload at %v showing=%v: got %v want %v", at, showing, got, want)
			}
		}
	}
	if len(calls) > 0 && (len(showing) == 0 || len(erased) == 0) {
		t.Fatal("showing panel text was suppressed")
	}
	return len(calls), len(showing), partial
}

func TestMapEditorModalCoversSurviveTextPreparationAndDraw(t *testing.T) {
	t.Cleanup(text.ResetCapture)
	for _, size := range []image.Point{image.Pt(640, 480), image.Pt(1280, 800), image.Pt(1920, 1080)} {
		for _, mode := range []string{"open", "saveas", "message"} {
			for _, nilFont := range []bool{false, true} {
				t.Run(fmt.Sprintf("%dx%d/%s/nil-%v", size.X, size.Y, mode, nilFont), func(t *testing.T) {
					font := panelFont()
					if nilFont {
						font = nil
					}
					e, edits := selectedTextEditor(t, font, size)
					doc, records, source := e.doc, append([]InspectionRecord(nil), e.doc.Records...), bytes.Clone(e.doc.SourceBytes)
					plain := e.PanelImage(size.Y)
					if mode == "message" {
						if err := e.Open("refused"); err == nil || e.Message() == "" {
							t.Fatal("refused open did not display its message")
						}
					} else if err := e.Key(mode); err != nil || !e.pathMode || e.pathSave != (mode == "saveas") {
						t.Fatalf("production %s input did not enter its modal: %v", mode, err)
					}
					screen := ebiten.NewImage(size.X, size.Y)
					defer screen.Dispose()
					e.Draw(screen)
					captured, kept, _ := assertEditorPanelUpload(t, e, size.Y)
					if gotCaptured, gotKept, readbacks := e.TextSettle(); gotCaptured != captured || gotKept != kept || readbacks != 0 || e.doc.Viewer.textSettleFallbacks != 0 {
						t.Fatalf("production Draw disagrees with its upload seam: %d/%d/%d want %d/%d", gotCaptured, gotKept, readbacks, captured, kept)
					}
					if nilFont && (captured != 0 || kept != 0) {
						t.Fatal("nil font acquired text")
					}
					if !nilFont && (captured == 0 || kept == 0 || kept >= captured) {
						t.Fatal("selected record cover was not exercised", captured, kept)
					}
					v := e.doc.Viewer
					ox, oy := v.place.Origin()
					if v.place.Scale() != 1 || ox != 0 || oy != 0 || v.cam.ViewW != size.X-editorRailWidth {
						t.Fatal("rail placement changed")
					}
					for _, f := range e.TextFates() {
						if f.Call.X < v.cam.ViewW || f.Call.X >= size.X {
							t.Fatal("captured text left the rail", f.Call.X)
						}
					}
					e.SetTextSmoothing(false)
					e.Draw(screen)
					pic, blit, calls, showing := e.panelPicture(size.Y)
					if !bytes.Equal(pic.Pix, blit.Pix) || len(calls) != 0 || len(showing) != 0 {
						t.Fatal("smoothing off changed the native upload")
					}
					if c, k, _ := e.TextSettle(); c != 0 || k != 0 {
						t.Fatal("smoothing off retained capture")
					}
					if err := e.Key("escape"); err != nil || e.pathMode || e.Message() != "" {
						t.Fatal("production escape did not dismiss the cover", err)
					}
					if !bytes.Equal(plain.Pix, e.PanelImage(size.Y).Pix) {
						t.Fatal("modal dismissal did not restore the selected-record panel")
					}
					e.SetTextSmoothing(true)
					e.Draw(screen)
					assertEditorPanelUpload(t, e, size.Y)
					if e.doc != doc || e.Selected() != 0 || !reflect.DeepEqual(e.doc.Records, records) || !bytes.Equal(e.doc.SourceBytes, source) || e.Dirty() || *edits != (editorTextEdits{}) {
						t.Fatal("presentation mutated the document, selection, bytes or edit history")
					}
					t.Logf("production %s at %v: captured=%d kept=%d; covered cells exact, scale=1, no readback or edit", mode, size, captured, kept)
				})
			}
		}
	}
}

func TestMapEditorPartlyCoveredGlyphCellsStayNative(t *testing.T) {
	t.Cleanup(text.ResetCapture)
	font := panelFont()
	for i := 1; i < len(font.Glyphs); i++ {
		g := &font.Glyphs[i]
		g.Width, g.Height, g.Advance = 3, 40, 4
		g.Pixels = make([]text.Pixel, g.Width*g.Height)
		for n := range g.Pixels {
			g.Pixels[n] = text.Pixel{Level: text.MaxLevel, Painted: true}
		}
	}
	e, _ := selectedTextEditor(t, font, image.Pt(1280, 800))
	if err := e.Key("open"); err != nil {
		t.Fatal(err)
	}
	screen := ebiten.NewImage(1280, 800)
	defer screen.Dispose()
	e.Draw(screen)
	_, _, partial := assertEditorPanelUpload(t, e, 800)
	if partial == 0 {
		t.Fatal("fixture did not cross the modal's opaque edge")
	}
	t.Logf("partly covered glyphs=%d; every covered cell retains its final native value", partial)
}
