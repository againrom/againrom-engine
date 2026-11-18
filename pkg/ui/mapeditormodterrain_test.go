package ui

import (
	"errors"
	"fmt"
	"image"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/pathdialog"
)

// groundHost is a mod host whose terrain page holds one button that asks
// for a picture.
type groundHost struct{ *modesWorkspace }

func (h groundHost) WorkspaceFields(scope string) []InspectionField {
	if scope == ModTerrainScope {
		return []InspectionField{{Key: "newground", Label: "Add ground", Action: true, Column: 1, Pick: []string{"png", "bmp"}}}
	}
	return h.modesWorkspace.WorkspaceFields(scope)
}

// The terrain page's Add ground button asks the host's path window for a
// PNG or BMP and sends the chosen picture down the button's route.
func TestModWindowAddGroundButtonAsksForAPicture(t *testing.T) {
	chosen := filepath.Join(t.TempDir(), "glacier.png")
	e, calls, requests := modOpenEditor(t, chosen, nil)
	e.doc.Edits = groundHost{e.doc.Edits.(*modesWorkspace)}
	for range len(e.modPages()) {
		if e.modFieldScope() == ModTerrainScope {
			break
		}
		e.turnModPage(1)
	}
	if e.modFieldScope() != ModTerrainScope {
		t.Fatalf("no terrain page: scope %q", e.modFieldScope())
	}
	if !e.FocusModField("newground") || e.Key("enter") != nil {
		t.Fatal("the terrain page has no Add ground button")
	}
	finishEditorDialog(t, e)
	if len(requests) != 1 {
		t.Fatalf("%d path window requests", len(requests))
	}
	if r := <-requests; len(r.Filters) != 1 || !slices.Equal(r.Filters[0].Extensions, []string{"png", "bmp"}) {
		t.Fatalf("request %+v", r)
	}
	if want := fmt.Sprintf("Workspace %s newground %q", ModTerrainScope, chosen); !slices.Contains(calls.lines, want) {
		t.Fatalf("the chosen picture did not reach the button's route: %q, want %q", calls.lines, want)
	}
}

// groundGridHost is a mod host whose terrain page is an Add button and one
// grid of 30 grounds, each a picture with three variations.
type groundGridHost struct{ *modesWorkspace }

func (h groundGridHost) WorkspaceFields(scope string) []InspectionField {
	if scope != ModTerrainScope {
		return h.modesWorkspace.WorkspaceFields(scope)
	}
	grid := InspectionField{Key: "grounds", Label: "Grounds", Grid: true, Value: 3, Icons: map[int64]*image.RGBA{}, Variations: map[int64][]*image.RGBA{}}
	for i := range 30 {
		v := int64(i + 1)
		grid.Choices = append(grid.Choices, InspectionChoice{Value: v, Label: fmt.Sprintf("Ground %d", v)})
		grid.Icons[v] = image.NewRGBA(image.Rect(0, 0, 32, 32))
		for range 3 {
			grid.Variations[v] = append(grid.Variations[v], image.NewRGBA(image.Rect(0, 0, 32, 32)))
		}
	}
	return []InspectionField{{Key: "newground", Label: "Add ground", Action: true, Column: 1, Pick: []string{"png", "bmp"}}, grid}
}

// The terrain page shows its grounds one after another as picture cells
// named under their pictures, wrapping into rows, never as rows of text; a
// ground's hover shows every variation at the cell's picture size, and a
// press on a cell picks that ground.
func TestModTerrainGridWrapsGroundsWithVariationTips(t *testing.T) {
	for _, size := range []image.Point{{640, 480}, {1024, 768}} {
		e, calls, _ := manyGroundsEditor(t, size, false)
		e.doc.Edits = groundGridHost{e.doc.Edits.(manyGroundsHost).modesWorkspace}
		e.showModPage(e.modPage)
		s, _, ok := e.modSpec()
		if !ok || s.Grid == nil || len(s.Grid.Cells) != 30 {
			t.Fatalf("%v: no grid of 30 grounds: %+v", size, s.Grid)
		}
		for _, r := range s.Rows {
			if r.Label == "Grounds" {
				t.Fatalf("%v: the grid is also a row", size)
			}
		}
		l := layoutPopup(s, e.font, e.bodyFont())
		lines := 0
		for i, r := range l.GridCells {
			c := s.Grid.Cells[l.gridIndex(i)]
			// A 44-pixel cell names its ground in its tip, no caption.
			if c.Caption != "" || c.Tip != fmt.Sprintf("Ground %d", l.gridIndex(i)+1) || s.Grid.Side != gridCellSide || s.Grid.CaptionH != 0 || c.Pressed != (l.gridIndex(i) == 2) {
				t.Fatalf("%v: cell %d is %+v", size, i, c)
			}
			if i > 0 && r.Min.Y > l.GridCells[i-1].Min.Y {
				lines++
			} else if i > 0 && r.Min.X <= l.GridCells[i-1].Min.X {
				t.Fatalf("%v: cell %d does not follow cell %d", size, i, i-1)
			}
		}
		if lines == 0 || len(l.GridCells) == 0 {
			t.Fatalf("%v: %d cells in one line", size, len(l.GridCells))
		}
		first := l.GridCells[0]
		cell, _, ok := e.gridCellAt(center(first))
		side := s.Grid.pictureSide()
		if !ok || cell.TipPicture == nil || side != first.Dx()-8 {
			t.Fatalf("%v: hover %+v, picture side %d of a cell %v", size, cell, side, first)
		}
		if b := cell.TipPicture.Bounds(); b.Dx() != 3*side+4 || b.Dy() != side {
			t.Fatalf("%v: the tip sheet is %v, want three pictures of %d", size, b, side)
		}
		modesClick(e, center(l.GridCells[5]))
		if want := fmt.Sprintf("Workspace %s grounds %q", ModTerrainScope, "6"); !slices.Contains(calls.lines, want) {
			t.Fatalf("%v: the press sent %q, want %q", size, calls.lines, want)
		}
	}
}

// manyGroundsHost is a mod host whose terrain page lists grounds, each with a
// name, a kind of walking and a Delete button that carries help, after an Add
// button that carries help.
type manyGroundsHost struct {
	*modesWorkspace
	bare bool // no Delete rows: the page is name and kind rows only
}

func (h manyGroundsHost) WorkspaceFields(scope string) []InspectionField {
	if scope != ModTerrainScope {
		return h.modesWorkspace.WorkspaceFields(scope)
	}
	kinds := make([]InspectionChoice, 10)
	for i := range kinds {
		kinds[i] = InspectionChoice{Value: int64(i + 1), Label: fmt.Sprintf("Kind %d", i+1)}
	}
	out := []InspectionField{{Key: "newground", Label: "Add ground", Meaning: "Adds a ground from a picture.", Action: true, Column: 1, Pick: []string{"png", "bmp"}}}
	for i := range 8 {
		key := fmt.Sprintf("g%d.", i)
		out = append(out,
			InspectionField{Key: key + "name", Label: "Name", Text: true, TextValue: "Ground"},
			InspectionField{Key: key + "kind", Label: "Walked like", Value: 1, Choices: kinds})
		if h.bare {
			continue
		}
		out = append(out,
			InspectionField{Key: key + "remove", Label: "Delete ground", Meaning: "Deletes this ground.", Action: true, Column: 1})
	}
	return out
}

func manyGroundsEditor(t *testing.T, size image.Point, bare bool) (*MapEditor, *modesCalls, chan pathdialog.Request) {
	t.Helper()
	e := modesWordsEditor(t, "workspace", "en", size)
	e.places[popObject].placed = false // W opens centred
	requests := make(chan pathdialog.Request, 4)
	bindEditorDialog(e, func(r pathdialog.Request) (pathdialog.Result, error) {
		requests <- r
		return pathdialog.Result{}, errors.New("cancelled")
	}, t.TempDir())
	calls := e.doc.Edits.(*modesWorkspace).calls
	e.doc.Edits = manyGroundsHost{e.doc.Edits.(*modesWorkspace), bare}
	if err := e.Key("package"); err != nil {
		t.Fatal(err)
	}
	for range len(e.modPages()) {
		if e.modFieldScope() == ModTerrainScope {
			return e, calls, requests
		}
		e.turnModPage(1)
	}
	t.Fatalf("no terrain page: scope %q", e.modFieldScope())
	return nil, nil, nil
}

// A press on the (?) mark of a row shows that row's help and never runs the
// row's action; a press on the row's button still runs it.
func TestPressOnARowHelpMarkNeverRunsItsAction(t *testing.T) {
	for _, size := range []image.Point{{640, 480}, {1024, 768}} {
		e, calls, requests := manyGroundsEditor(t, size, false)
		s, _, _ := e.modSpec()
		l := layoutPopup(s, e.font, e.bodyFont())
		marks, buttons := 0, 0
		for _, rr := range l.Rows {
			if rr.Help.Empty() || !rr.Help.In(l.RowsViewport) || !strings.Contains(s.Rows[rr.Index].Help, "Add") && !strings.Contains(s.Rows[rr.Index].Help, "Deletes") {
				continue
			}
			marks++
			before := len(calls.lines)
			at := center(rr.Help)
			if part, _ := l.hit(at); part != popupRowHelp {
				t.Fatalf("%v: hit at the help mark of row %d is %d", size, rr.Index, part)
			}
			modesClick(e, at)
			if len(calls.lines) != before || len(requests) != 0 {
				t.Fatalf("%v: a press on the help mark ran the row: %v", size, calls.lines[before:])
			}
			if tip, ok := popupTipAt(s.Bounds, e.font, e.bodyFont(), s, l, at, true); !ok || len(tip.Lines) == 0 {
				t.Fatalf("%v: the press shows no help", size)
			}
			if len(rr.Buttons) > 0 && !rr.Buttons[0].Empty() && strings.Contains(s.Rows[rr.Index].Help, "Deletes") && buttons == 0 {
				modesClick(e, center(rr.Buttons[0]))
				if len(calls.lines) == before {
					t.Fatalf("%v: a press on the row's button ran nothing", size)
				}
				buttons++
			}
			s, _, _ = e.modSpec()
			l = layoutPopup(s, e.font, e.bodyFont())
		}
		if marks == 0 {
			t.Fatalf("%v: no row with a help mark", size)
		}
	}
}

// An open choice list that does not fit below its field rises to end at the
// editor window's bottom; where it fits it keeps its place below the field.
func TestChoiceListRisesToTheWindowBottom(t *testing.T) {
	for _, size := range []image.Point{{640, 480}, {1024, 768}} {
		e, _, _ := manyGroundsEditor(t, size, true)
		s, _, _ := e.modSpec()
		l := layoutPopup(s, e.font, e.bodyFont())
		// The lowest kind field at 640x480, the highest at 1024x768, where
		// the centred window leaves the room below it.
		var field image.Rectangle
		var key string
		for _, rr := range l.Rows {
			i, ok := e.obj.rowField[rr.Index]
			if ok && strings.HasSuffix(e.obj.fields[i].Key, "kind") && rr.Control.Max.Y <= l.RowsViewport.Max.Y && rr.Control.Min.Y >= field.Min.Y && (key == "" || size.Y == 480) {
				field, key = rr.Control, e.obj.fields[i].Key
			}
		}
		if key == "" || !e.FocusModField(key) || e.Key("enter") != nil {
			t.Fatalf("%v: no kind field to open", size)
		}
		d, ok := e.openDropdown()
		if !ok {
			t.Fatalf("%v: the list did not open", size)
		}
		top, count := d.List.Visible()
		last, ok := e.FindControl(ControlQuery{Role: ControlDropdownRow, N: top + count - 1})
		if !ok {
			t.Fatalf("%v: no last row", size)
		}
		if last.Max.Y > size.Y {
			t.Fatalf("%v: the list ends at %d, past the window's %d", size, last.Max.Y, size.Y)
		}
		if size.Y == 768 && d.listBox(e.bodyFont()).Rect.Min.Y != d.Rect.Max.Y+1 {
			t.Fatalf("%v: a list that fits left its place below the field", size)
		}
		if size.Y == 480 && d.listBox(e.bodyFont()).Rect.Max.Y != size.Y {
			t.Fatalf("%v: the list did not rise to end at the window bottom", size)
		}
		before := e.obj.open
		modesClick(e, center(last))
		if e.obj.open == before && e.obj.open >= 0 {
			t.Fatalf("%v: a press on the last row left the list open", size)
		}
	}
}

// groundsOfHost is a mod host whose terrain page is one grid of n grounds, the
// last with many variations, and the rows of a pressed mod ground below it.
type groundsOfHost struct {
	*modesWorkspace
	n, many int
}

func (h groundsOfHost) WorkspaceFields(scope string) []InspectionField {
	if scope != ModTerrainScope {
		return h.modesWorkspace.WorkspaceFields(scope)
	}
	grid := InspectionField{Key: "grounds", Label: "Grounds", Grid: true, Value: int64(h.n), Icons: map[int64]*image.RGBA{}, Variations: map[int64][]*image.RGBA{}}
	for i := range h.n {
		v := int64(i + 1)
		grid.Choices = append(grid.Choices, InspectionChoice{Value: v, Label: fmt.Sprintf("Ground %d", v)})
		grid.Icons[v] = image.NewRGBA(image.Rect(0, 0, 32, 32))
		count := 3
		if i == h.n-1 {
			count = h.many
		}
		for range count {
			grid.Variations[v] = append(grid.Variations[v], image.NewRGBA(image.Rect(0, 0, 32, 32)))
		}
	}
	return []InspectionField{{Key: "newground", Label: "Add ground", Meaning: "Adds a ground from a picture.", Action: true, Column: 1, Pick: []string{"png", "bmp"}}, grid,
		{Key: "name", Label: "Name", Text: true, TextValue: "Ground"},
		{Key: "remove", Label: "Delete ground", Meaning: "Deletes this ground.", Action: true, Column: 1}}
}

// With 29 grounds at 640x480 and at 800x600 every cell the page shows, the
// popup and the window's buttons stand inside the editor window, the rest of
// the grounds scroll, and the tip of a ground with 56 variations stands
// inside the window too.
func TestModTerrainGridStaysInsideTheEditorWindow(t *testing.T) {
	for _, size := range []image.Point{{640, 480}, {800, 600}} {
		e, _, _ := manyGroundsEditor(t, size, false)
		e.doc.Edits = groundsOfHost{e.doc.Edits.(manyGroundsHost).modesWorkspace, 29, 56}
		e.showModPage(e.modPage)
		window := image.Rectangle{Max: size}
		s, _, ok := e.modSpec()
		if !ok || s.Grid == nil || len(s.Grid.Cells) != 29 {
			t.Fatalf("%v: no grid of 29 grounds", size)
		}
		l := layoutPopup(s, e.font, e.bodyFont())
		if !l.Rect.In(window) || len(l.GridCells) == 0 {
			t.Fatalf("%v: the popup %v is not inside the window, %d cells", size, l.Rect, len(l.GridCells))
		}
		for i, r := range l.GridCells {
			if !r.In(window) {
				t.Fatalf("%v: cell %d %v is outside the window", size, i, r)
			}
		}
		if len(l.Buttons) != len(modButtons) {
			t.Fatalf("%v: %d buttons, want %d", size, len(l.Buttons), len(modButtons))
		}
		for i, r := range l.Buttons {
			if !r.In(window) {
				t.Fatalf("%v: button %d %v is outside the window", size, i, r)
			}
		}
		// Scroll to the last ground and hover it.
		for range 40 {
			if len(l.GridCells) > 0 && l.gridIndex(len(l.GridCells)-1) == 28 {
				break
			}
			e.pointer = center(l.GridCells[0])
			e.chromeWheel(popObject, Input{WheelY: -1})
			s, _, _ = e.modSpec()
			l = layoutPopup(s, e.font, e.bodyFont())
		}
		if l.gridIndex(len(l.GridCells)-1) != 28 {
			t.Fatalf("%v: the wheel never reached the last ground", size)
		}
		last := l.GridCells[len(l.GridCells)-1]
		cell, _, ok := e.gridCellAt(center(last))
		if !ok || cell.TipPicture == nil {
			t.Fatalf("%v: no tip sheet over the last ground", size)
		}
		tip, ok := e.tipAt(center(last))
		if !ok || tip.Picture == nil || !tip.Rect.In(window) {
			t.Fatalf("%v: the tip %v is not inside the window %v", size, tip.Rect, window)
		}
	}
}
