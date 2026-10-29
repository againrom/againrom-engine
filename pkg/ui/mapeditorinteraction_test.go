package ui

import (
	"bytes"
	"fmt"
	"image"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
)

type settingsPreviewProbe struct {
	*popupHost
	hide     bool
	readOnly bool
}

func (p *settingsPreviewProbe) Fields(kind string, index int) []InspectionField {
	fields := p.popupHost.Fields(kind, index)
	if p.readOnly && kind == "Unit" {
		fields = append(fields, InspectionField{Key: "computed", ReadOnly: true})
	}
	return fields
}

func TestEditorCatalogCancelRetainsItsCheckpointWhenBrushChanges(t *testing.T) {
	for _, brush := range []bool{false, true} {
		t.Run(strconv.FormatBool(brush), func(t *testing.T) {
			e, _ := newModesEditor(t, "full")
			e.cancelPopup(popList)
			if brush {
				e.toggleBrush(brushTerrain)
			}
			before := e.editorMode
			e.enterCatalog()
			e.toggleBrush(brushTerrain)
			_ = e.Key("escape")
			if !reflect.DeepEqual(before, e.editorMode) {
				t.Fatal("catalog Cancel lost its original mode checkpoint")
			}
		})
	}
}

func TestEditorListCancelRestoresModeAndPreviousPopup(t *testing.T) {
	for _, kind := range []string{"panel", "records", "catalog", "report"} {
		for _, control := range []string{"escape", "close"} {
			t.Run(kind+"/"+control, func(t *testing.T) {
				e, calls := newModesEditor(t, "full")
				e.cancelPopup(popList)
				e.toggleBrush(brushTerrain)
				e.listSpec()
				mode, cam := e.editorMode, *e.doc.Viewer.cam
				selected, top := e.selected, e.objectTop
				raw := bytes.Clone(e.doc.SourceBytes)
				undo, redo := e.doc.Edits.CanUndo(), e.doc.Edits.CanRedo()
				switch kind {
				case "panel":
					e.openPanel()
				case "records":
					e.showRecords(1)
				case "catalog":
					e.enterCatalog()
				case "report":
					e.openReport(nil, "", nil)
				}
				e.selected, e.objectTop, e.objectHidden = 0, top+3, true
				e.doc.Viewer.cam.X += 100
				if control == "escape" {
					_ = e.Key("escape")
				} else {
					s, ok := e.listSpec()
					if !ok {
						t.Fatal("missing list popup")
					}
					chromeClick(e, center(layoutPopup(s, e.font, e.bodyFont()).Close))
				}
				if !reflect.DeepEqual(mode, e.editorMode) || !reflect.DeepEqual(cam, *e.doc.Viewer.cam) || e.selected != selected || e.objectTop != top || !bytes.Equal(raw, e.doc.SourceBytes) || e.doc.Edits.CanUndo() != undo || e.doc.Edits.CanRedo() != redo || len(calls.lines) != 0 {
					t.Fatal("list Cancel changed mode, camera, selection, bytes or history")
				}
				_ = e.Key("escape")
				if e.brushActive() || e.panel || e.listShown || e.catalog || e.report.open {
					t.Fatal("previous palette Cancel did not close")
				}
			})
		}
	}
}

func TestEditorCardKeepsUsedByLinesInContentColumn(t *testing.T) {
	e, host := popupEditor(t)
	_ = e.Open("popups")
	e.doc.Edits = &settingsPreviewProbe{popupHost: host}
	id := len(e.doc.Records)
	e.doc.Records = append(e.doc.Records, InspectionRecord{Kind: "Action", Index: 0})
	e.doc.Script = &InspectionScriptModel{
		Actions: []InspectionScriptNode{{Record: id, Label: strings.Repeat("Long reference ", 12)}},
		Targets: map[uint32][]InspectionScriptTarget{scriptUnit: {{Record: 0, Uses: []InspectionScriptUse{{Node: id}}}}},
	}
	e.Select(0, false)
	s, links, _ := e.objectSpec()
	first := -1
	for i, link := range links {
		if link == -id-1 {
			first = i
			break
		}
	}
	if first < 0 || s.Card == nil {
		t.Fatal("missing unit card or Used by link")
	}
	e.objectTop = first
	s, _, _ = e.objectSpec()
	l := layoutPopup(s, e.font, e.bodyFont())
	for _, rr := range l.Rows {
		row := s.Rows[rr.Index]
		if row.Kind == popupLink && e.bodyFont().Advance(row.Label) > rr.Label.Dx()-2 {
			t.Fatal("Used by label exceeds its content column", row.Label)
		}
	}
}

func (p *settingsPreviewProbe) UnitSettings(index int, values []InspectionFieldValue) (MapEntity, []InspectionField, *InspectionUnitPreview, error) {
	body, _ := strconv.Atoi(p.units[index]["a"])
	for _, v := range values {
		if v.Key == "a" {
			body = int(v.Value)
		}
	}
	pic := image.NewRGBA(image.Rect(0, 0, 2, 2))
	pic.Pix[0], pic.Pix[3] = byte(body), 255
	fields := p.Fields("Unit", index)
	if p.readOnly {
		fields[len(fields)-1].Lines = []InspectionLine{{Text: fmt.Sprintf("Carrier %d", body)}}
	}
	if p.hide {
		fields = fields[:1]
	}
	return MapEntity{ID: uint32(index), Char: UnitCharacter{Known: true, Band: CharacterBandPerson, Body: body}, HP: body, MaxHP: 100}, fields, &InspectionUnitPreview{Doll: pic, Face: pic}, nil
}

func TestEditorUnitCardRebuildsForDraftWithoutApply(t *testing.T) {
	e, host := popupEditor(t)
	_ = e.Open("popups")
	e.doc.Edits = &settingsPreviewProbe{popupHost: host}
	e.Select(0, false)
	s, _, _ := e.objectSpec()
	if s.Card == nil {
		t.Fatal("unit settings omit game card")
	}
	before := e.ChromeImage()
	raw := append([]byte(nil), e.doc.SourceBytes...)
	e.obj.buf[0] = "42"
	s, _, _ = e.objectSpec()
	if s.Card == nil || s.Card.Char.Body != 42 {
		t.Fatal("card did not update for field draft")
	}
	after := e.ChromeImage()
	l := layoutPopup(s, e.font, e.bodyFont())
	r := image.Rect(l.Rect.Max.X-popupPad-160, l.Title.Max.Y+popupPad, l.Rect.Max.X-popupPad, l.Title.Max.Y+popupPad+242)
	changed := false
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			changed = changed || before.RGBAAt(x, y) != after.RGBAAt(x, y)
		}
	}
	if !changed || !bytes.Equal(raw, e.doc.SourceBytes) || len(host.calls) != 0 {
		t.Fatal("draft card did not repaint, or wrote map")
	}
	s.Rows = make([]popupRow, 30)
	s.MaxRows = len(s.Rows)
	s.Size = image.Point{}
	preview := image.NewRGBA(image.Rect(0, 0, 1280, 800))
	e.drawObjectPreview(preview, s)
	painted := false
	for at := 0; at < len(preview.Pix); at += 4 {
		painted = painted || preview.Pix[at] == 42 && preview.Pix[at+3] == 255
	}
	if !painted {
		t.Fatal("draft doll did not reach the object preview painter")
	}
	s.Card.Char.Band = CharacterBandCreature
	creature := image.NewRGBA(preview.Bounds())
	e.drawObjectPreview(creature, s)
	if !bytes.Equal(creature.Pix, make([]byte, len(creature.Pix))) {
		t.Fatal("a creature received the human doll preview")
	}
	calls := text.Record(func() { e.ChromeImage() })
	for _, call := range calls {
		if call.Y >= l.Title.Max.Y+popupPad && call.Y < l.Rect.Max.Y && call.X < l.Rect.Min.X {
			t.Fatal("card glyph captured outside its popup", call.X, call.Y)
		}
	}
}

func TestEditorUnitDraftHidesFieldsTheResolvedIdentityDoesNotRead(t *testing.T) {
	e, host := popupEditor(t)
	_ = e.Open("popups")
	e.doc.Edits = &settingsPreviewProbe{popupHost: host, hide: true}
	e.Select(0, false)
	if _, _, ok := e.objectSpec(); !ok {
		t.Fatal("missing object popup")
	}
	if e.fieldRow(0) < 0 || e.fieldRow(1) >= 0 || e.fieldRow(2) >= 0 {
		t.Fatal("draft kept fields the host no longer admits")
	}
	if e.nextObjectField(0) != 0 {
		t.Fatal("keyboard focuses a hidden field")
	}
}

func TestEditorReadOnlyDraftLinesCannotReceiveFocusOrWrites(t *testing.T) {
	e, host := popupEditor(t)
	_ = e.Open("popups")
	e.doc.Edits = &settingsPreviewProbe{popupHost: host, readOnly: true}
	e.Select(0, false)
	_, _, _ = e.objectSpec()
	last := len(e.obj.fields) - 1
	if e.fieldRow(last) >= 0 || e.nextObjectField(last-1) != 0 {
		t.Fatal("read-only line takes keyboard focus")
	}
	e.obj.buf[last] = "attempted write"
	if e.objectDirty() {
		t.Fatal("read-only line makes the draft dirty")
	}
	e.obj.buf[0] = "37"
	spec, _, _ := e.objectSpec()
	seen := false
	for _, row := range spec.Rows {
		seen = seen || row.Label == "Carrier 37"
	}
	if !seen {
		t.Fatal("read-only line did not rebuild for the draft")
	}
	e.applyObject()
	if host.units[0]["a"] != "37" || host.units[0]["computed"] != "" {
		t.Fatal("Apply included the read-only value", host.units[0])
	}
}

func TestEditorListsClampAndThumbFollowsPointer(t *testing.T) {
	for _, where := range []string{"records", "add", "dropdown", "details"} {
		t.Run(where, func(t *testing.T) {
			e, _ := popupEditor(t)
			_ = e.Open("popups")
			var m *Picker
			var bar vScrollBar
			switch where {
			case "records":
				e.showRecords(1)
				s, _ := e.listSpec()
				l := layoutPopup(s, e.font, e.bodyFont())
				m = s.List
				bar = listBar(l.List, m)
			case "add":
				e, _ = newModesEditor(t, "full")
				e.openAdd("Unit")
				r, _ := e.addRunning()
				for i := 0; i < 80; i++ {
					r.offer.Choices = append(r.offer.Choices, InspectionChoice{Value: int64(i + 100), Label: fmt.Sprint(i)})
				}
				e.addList()
				s, _ := e.addSpec()
				l := layoutPopup(s, e.font, e.bodyFont())
				m = s.List
				bar = listBar(l.List, m)
			case "dropdown":
				e.Select(0, false)
				e.objectSpec()
				o := &e.obj
				for i := 0; i < 80; i++ {
					o.fields[1].Choices = append(o.fields[1].Choices, InspectionChoice{Value: int64(i + 100), Label: fmt.Sprint(i)})
				}
				e.objectRowPress(popupRowControl, e.fieldRow(1))
				d, ok := e.openDropdown()
				if !ok {
					t.Fatal("no dropdown")
				}
				m = d.List
				bar = listBar(d.listBox(e.bodyFont()), m)
			case "details":
				e.Select(0, false)
				for i := 0; i < 100; i++ {
					e.doc.Records[0].Details = append(e.doc.Records[0].Details, fmt.Sprint(i))
				}
				s, _, _ := e.objectSpec()
				l := layoutPopup(s, e.font, e.bodyFont())
				bar = vScrollBar{Rect: l.RowBar, Count: len(s.Rows) - s.MaxRows + 1}
			}
			if where != "details" {
				m.Select(0)
				editorListStep(m, -1000)
				if m.Selection() != 0 {
					t.Fatal("top wrapped")
				}
				editorListStep(m, 1000)
				if m.Selection() != m.Len()-1 {
					t.Fatal("bottom not reached")
				}
				editorListStep(m, 1)
				if m.Selection() != m.Len()-1 {
					t.Fatal("bottom wrapped")
				}
				m.Select(0)
			}
			bar.Pos = 0
			track := image.Pt(center(bar.Rect).X, bar.Thumb().Max.Y+4)
			e.Pointer(Input{CursorX: track.X, CursorY: track.Y, PrimaryDown: true}, time.Time{})
			if where == "details" {
				if e.objectTop == 0 {
					t.Fatal("track press did not move at once")
				}
			} else {
				if m.Selection() == 0 {
					t.Fatal("track press did not move at once")
				}
			}
			e.Pointer(Input{CursorX: track.X, CursorY: track.Y}, time.Time{})
			if where == "details" {
				e.objectTop = 0
			} else {
				m.Select(0)
			}
			p := center(bar.Thumb())
			e.Pointer(Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, time.Time{})
			lastBar := bar
			lastBar.Pos = bar.Count - 1
			end := center(lastBar.Thumb())
			middle := image.Pt(p.X, (p.Y+end.Y)/2)
			e.Pointer(Input{CursorX: middle.X, CursorY: middle.Y, PrimaryDown: true}, time.Time{})
			at := e.objectTop
			if m != nil {
				at = m.Selection()
			}
			if want := (bar.Count - 1) / 2; at < want-1 || at > want+1 {
				t.Fatalf("half travel selects %d want near %d", at, want)
			}
			e.Pointer(Input{CursorX: end.X, CursorY: end.Y, PrimaryDown: true}, time.Time{})
			if where == "details" {
				if e.objectTop != bar.Count-1 {
					t.Fatalf("details drag=%d want %d", e.objectTop, bar.Count-1)
				}
			} else if m.Selection() != m.Len()-1 {
				t.Fatalf("drag=%d want %d", m.Selection(), m.Len()-1)
			}
			e.Pointer(Input{CursorX: end.X, CursorY: end.Y}, time.Time{})
			if e.scroll.input.active() {
				t.Fatal("bar kept capture")
			}
		})
	}
}

type placementPreviewProbe struct {
	*modesFull
	refused bool
	view    *InspectionDocument
}

func (p *placementPreviewProbe) PreviewPlacement(r InspectionNewRecord) (*InspectionDocument, error) {
	if p.refused {
		return nil, fmt.Errorf("refused cell")
	}
	p.view.Viewer.SetEntities([]MapEntity{{ID: 0, Cell: r.Cell, Frame: sackFrame(18, 25), Art: &terrain.UnitClass{TileSize: 1}}})
	p.view.Records = []InspectionRecord{{Kind: "Unit", Index: 0}}
	return p.view, nil
}

func TestEditorAddArmedPlacementFrameAndRefusal(t *testing.T) {
	e, calls := newModesEditor(t, "full")
	p := &placementPreviewProbe{modesFull: e.doc.Edits.(*modesFull), view: func() *InspectionDocument { preview, _ := newModesEditor(t, "full"); return preview.doc }()}
	p.self, e.doc.Edits = p, p
	e.openAdd("Unit")
	if !e.addPicking() {
		t.Fatal("Add opened without armed pick")
	}
	e.Fit()
	point := e.CellPoint(image.Pt(25, 20))
	e.Pointer(Input{CursorX: point.X, CursorY: point.Y}, time.Time{})
	f, ok := e.placement()
	if !ok || f.Cell != image.Pt(25, 20) || f.Refused || len(f.Sprites) != 1 || !f.Sprites[0].Translucent {
		t.Fatalf("placement frame %+v open=%v", f, ok)
	}
	if len(e.markerLines()) == 0 || e.placementPicture() == nil || len(calls.lines) != 0 {
		t.Fatal("missing hover or preview wrote")
	}
	p.refused = true
	e.placementKey = ""
	f, _ = e.placement()
	if !f.Refused || len(f.Sprites) != 0 {
		t.Fatal("refused placement not marked")
	}
	e.Pointer(Input{SecondaryDown: true}, time.Time{})
	if e.pick != pickNone {
		t.Fatal("right click kept pick")
	}
	e.armPick(pickAddCell)
	_ = e.Key("escape")
	if e.pick != pickNone || !e.prompt.open() {
		t.Fatal("Esc did not disarm")
	}
	_ = e.Key("escape")
	if e.prompt.open() {
		t.Fatal("Esc kept Add")
	}
	p.refused = false
	e.openAdd("Unit")
	modesClick(e, point)
	if len(calls.lines) != 1 || e.prompt.open() {
		t.Fatalf("first map click did not place: %v", calls.lines)
	}
}

func TestEditorCancelClosesAndRestoresDraftAndPromptState(t *testing.T) {
	for _, control := range []string{"button", "close", "escape"} {
		t.Run(control, func(t *testing.T) {
			e, h := popupEditor(t)
			_ = e.Open("popups")
			e.Select(0, false)
			e.objectSpec()
			raw := append([]byte(nil), e.doc.SourceBytes...)
			selection := e.selected
			undo, redo := e.doc.Edits.CanUndo(), e.doc.Edits.CanRedo()
			e.obj.buf[0] = "999"
			e.obj.focus = 0
			s, _, _ := e.objectSpec()
			l := layoutPopup(s, e.font, e.bodyFont())
			switch control {
			case "button":
				chromeClick(e, center(l.Buttons[1]))
			case "close":
				chromeClick(e, center(l.Close))
			case "escape":
				_ = e.Key("escape")
			}
			if _, _, ok := e.objectSpec(); ok {
				t.Fatal("Cancel left object popup open")
			}
			if !bytes.Equal(raw, e.doc.SourceBytes) || e.selected != selection || undo != e.doc.Edits.CanUndo() || redo != e.doc.Edits.CanRedo() || len(h.calls) != 0 {
				t.Fatal("Cancel mutated document, history or selection")
			}
		})
	}
	for _, kind := range addKinds {
		if len(kind.Steps) == 0 {
			continue
		}
		e, calls := newModesEditor(t, "full")
		e.showRecords(1)
		e.listSpec()
		e.model.Select(2)
		listSelection := e.model.Selection()
		before := e.editorMode
		selected := e.selected
		e.openAdd(kind.Kind)
		e.cancelPopup()
		if e.prompt.open() || e.pick != before.pick || e.selected != selected || e.model.Selection() != listSelection || len(calls.lines) != 0 {
			t.Fatalf("%s Cancel changed state", kind.Kind)
		}
	}
}

func TestEditorObjectCellPickRemainsADraftOnCancel(t *testing.T) {
	e, h := popupEditor(t)
	_ = e.Open("popups")
	e.Select(0, false)
	e.objectSpec()
	raw := append([]byte(nil), e.doc.SourceBytes...)
	e.objectRowPress(popupRowButton, e.obj.cell)
	e.pickAt(e.CellPoint(image.Pt(12, 12)))
	if e.obj.move == nil || len(h.calls) != 0 || !bytes.Equal(raw, e.doc.SourceBytes) {
		t.Fatal("cell pick wrote before Apply")
	}
	e.cancelPopup()
	if e.obj.move != nil || len(h.calls) != 0 || !bytes.Equal(raw, e.doc.SourceBytes) {
		t.Fatal("Cancel kept or applied cell draft")
	}
}

func TestEditorPromptCancelCensusRestoresSelectionAndUndo(t *testing.T) {
	for kind := promptOpen; kind <= promptAreaShared; kind++ {
		t.Run(fmt.Sprint(kind), func(t *testing.T) {
			e, calls := newModesEditor(t, "full")
			selected := e.selected
			raw := append([]byte(nil), e.doc.SourceBytes...)
			undo, redo := e.doc.Edits.CanUndo(), e.doc.Edits.CanRedo()
			e.openPrompt(editorPrompt{kind: kind})
			e.selected = -100
			e.prompt.path = "edited"
			e.cancelPopup()
			if e.prompt.open() || e.selected != selected || !bytes.Equal(raw, e.doc.SourceBytes) || e.doc.Edits.CanUndo() != undo || e.doc.Edits.CanRedo() != redo || len(calls.lines) != 0 {
				t.Fatal("Cancel census changed state")
			}
		})
	}
}
