package game

import (
	"image"
	"testing"
	"time"

	"againrom/internal/synth"
	"againrom/pkg/ui"
)

// titleDrag presses the front popup's title bar, moves the pointer by d and
// lets go; it returns the title bar before and after.
func titleDrag(t *testing.T, e *ui.MapEditor, d image.Point) (image.Rectangle, image.Rectangle) {
	t.Helper()
	before, ok := e.TitleRect()
	if !ok {
		t.Fatal("no popup")
	}
	p := before.Min.Add(image.Pt(10, 4))
	q := p.Add(d)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, time.Time{})
	e.Pointer(ui.Input{CursorX: q.X, CursorY: q.Y, PrimaryDown: true}, time.Time{})
	e.Pointer(ui.Input{CursorX: q.X, CursorY: q.Y}, time.Time{})
	after, _ := e.TitleRect()
	return before, after
}

// popupCase is one editor window and how to open it.
type popupCase struct {
	name string
	open func(t *testing.T) *ui.MapEditor
}

// popupCases lists every kind of editor window the editor opens as a popup,
// list or dialog.
func popupCases() []popupCase {
	scriptMap := func(t *testing.T) *ui.MapEditor {
		e, _ := messagePopupFixture(t, 77)
		return e
	}
	return []popupCase{
		{"record list", func(t *testing.T) *ui.MapEditor {
			e := scriptMap(t)
			_ = e.Key("escape")
			_ = e.Key("structure")
			return e
		}},
		{"scripts list", func(t *testing.T) *ui.MapEditor {
			e := scriptMap(t)
			_ = e.Key("escape")
			_ = e.Key("scripts")
			return e
		}},
		{"node popup", scriptMap},
		{"unit popup", func(t *testing.T) *ui.MapEditor {
			_, e := openAuthoring(t)
			e.Select(recordIndex(t, e, "Unit", 0), false)
			return e
		}},
		{"message popup", func(t *testing.T) *ui.MapEditor {
			e := scriptMap(t)
			messageField(t, e, 2)
			r, ok := e.DropdownRowRect(0)
			messageClick(t, e, r, ok)
			return e
		}},
		{"variable popup", func(t *testing.T) *ui.MapEditor {
			e, _ := openGoalMap(t, synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Type7Payload: make([]byte, 12)}))
			scriptWindowAdd(t, e, "Condition")
			scriptWindowAdd(t, e, "Action")
			scriptWindowSet(t, e, "Action", 0, "value0", "0")
			scriptWindowSet(t, e, "Action", 0, "opcode", "3")
			_ = e.Key("scripts")
			_ = e.Key("escape")
			r, ok := treeMemberRect(e, "scripts", 0)
			trigClick(t, e, r, ok, "variable row")
			return e
		}},
		{"Add", func(t *testing.T) *ui.MapEditor {
			e := scriptMap(t)
			_ = e.Key("scripts")
			_ = e.Key("player")
			return e
		}},
		{"Save As path", func(t *testing.T) *ui.MapEditor {
			e := scriptMap(t)
			_ = e.Key("saveas")
			return e
		}},
		{"map metadata field", func(t *testing.T) *ui.MapEditor {
			e := scriptMap(t)
			_ = e.Key("metadata")
			return e
		}},
		{"new map size", func(t *testing.T) *ui.MapEditor {
			e := scriptMap(t)
			_ = e.Key("new")
			return e
		}},
		{"leave question", func(t *testing.T) *ui.MapEditor {
			e, s := messagePopupFixture(t, 77)
			if _, err := s.SetField("Action", 0, "value0", 78); err != nil {
				t.Fatal(err)
			}
			_ = e.Key("close")
			return e
		}},
		{"area trigger choice", func(t *testing.T) *ui.MapEditor {
			e, _ := areaEditors(t, areaEditMap(t, true))
			areaDraw(t, e, image.Pt(4, 4), image.Pt(7, 7))
			return e
		}},
		{"area shared question", func(t *testing.T) *ui.MapEditor {
			e, _ := areaEditors(t, areaEditMap(t, true))
			areaSelect(t, e, 2)
			a := areaOf(t, e, 2, ui.AreaCell, image.Pt(12, 25))
			areaPress(e, e.CellPoint(a.Rect.Min), e.CellPoint(image.Pt(10, 28)), true)
			return e
		}},
	}
}

// Every editor popup and input window moves by its title bar and stays in
// the window: a drag by d moves it by d, a drag far past the corner keeps its
// title row and 48 pixels inside the window, and the next window of the same place opens where its
// own place is.
func TestEditorPopupsMoveByTheirTitle(t *testing.T) {
	for _, c := range popupCases() {
		t.Run(c.name, func(t *testing.T) {
			e := c.open(t)
			d := image.Pt(-40, 30)
			if r, _ := e.TitleRect(); r.Min.X < 100 {
				d.X = 40
			}
			before, after := titleDrag(t, e, d)
			if after.Min != before.Min.Add(d) {
				t.Fatalf("title %v moved to %v, not by %v", before, after, d)
			}
			_, far := titleDrag(t, e, image.Pt(-5000, -5000))
			if far.Max.X != 48 || far.Min.Y <= 0 {
				t.Fatalf("a drag past the corner left the title at %v", far)
			}
			_, back := titleDrag(t, e, image.Pt(5000, 5000))
			if back.Min.X != 1280-48 || back.Min.Y <= far.Min.Y {
				t.Fatalf("a drag past the far corner left the title at %v", back)
			}
		})
	}
}

// A press below the title does not drag, and a prompt opened after a moved
// one opens at its own place.
func TestEditorPopupDragNeedsTheTitle(t *testing.T) {
	e, s := messagePopupFixture(t, 77)
	detachSource(t, s.source)
	_ = e.Key("saveas")
	home, _ := e.TitleRect()
	body, _ := e.PromptRect()
	p := image.Pt(body.Min.X+12, body.Max.Y-12)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, time.Time{})
	e.Pointer(ui.Input{CursorX: p.X - 60, CursorY: p.Y + 40, PrimaryDown: true}, time.Time{})
	e.Pointer(ui.Input{CursorX: p.X - 60, CursorY: p.Y + 40}, time.Time{})
	if r, ok := e.TitleRect(); !ok || r != home {
		t.Fatalf("a press below the title moved the prompt from %v to %v", home, r)
	}
	_, moved := titleDrag(t, e, image.Pt(-60, 40))
	if moved == home {
		t.Fatal("the title drag did not move the prompt")
	}
	_ = e.Key("escape")
	_ = e.Key("saveas")
	if r, _ := e.TitleRect(); r != home {
		t.Fatalf("the next prompt opened at %v, not its own place %v", r, home)
	}
}
