package ui

import (
	"errors"
	"fmt"
	"image"
	"strconv"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
)

// popupHost is an edit host whose map bytes are its units' fields as text,
// with a history of whole byte states.
type popupHost struct {
	units   []map[string]string
	history []string
	cursor  int
	calls   []string
	doc     func(*popupHost) *InspectionDocument
	plays   int
	target  string
	saves   int
}

func (h *popupHost) TestPlay() (string, error) { h.plays++; return "played", nil }
func (h *popupHost) ModTarget() string         { return h.target }
func (h *popupHost) SaveIntoMod() (string, error) {
	h.saves++
	return "saved", nil
}

// regroup moves unit i into group g and commits the edit.
func (h *popupHost) regroup(i, g int) *InspectionDocument {
	h.units[i]["group"] = strconv.Itoa(g)
	return h.commit()
}

func (h *popupHost) bytes() string {
	var b strings.Builder
	for _, u := range h.units {
		fmt.Fprintf(&b, "a=%s owner=%s name=%s group=%s;", u["a"], u["owner"], u["name"], u["group"])
	}
	return b.String()
}

func (h *popupHost) restore(s string) {
	for i, rec := range strings.Split(strings.TrimSuffix(s, ";"), ";") {
		for _, kv := range strings.Fields(rec) {
			k, v, _ := strings.Cut(kv, "=")
			h.units[i][k] = v
		}
	}
}

func (h *popupHost) commit() *InspectionDocument {
	h.history = append(h.history[:h.cursor], h.bytes())
	h.cursor = len(h.history)
	return h.doc(h)
}

func (h *popupHost) Dirty() bool   { return h.cursor > 1 }
func (h *popupHost) CanUndo() bool { return h.cursor > 1 }
func (h *popupHost) CanRedo() bool { return h.cursor < len(h.history) }
func (h *popupHost) Move(string, int, image.Point) (*InspectionDocument, error) {
	return nil, nil
}
func (h *popupHost) Clone(string, int, image.Point) (*InspectionDocument, error) {
	return nil, nil
}
func (h *popupHost) Delete(string, int) (*InspectionDocument, error) { return nil, nil }
func (h *popupHost) AddSack(image.Point, int64) (*InspectionDocument, error) {
	return nil, nil
}
func (h *popupHost) SaveAs(string) (string, error) { return "", errors.New("no writer") }
func (h *popupHost) Undo() (*InspectionDocument, error) {
	h.cursor--
	h.restore(h.history[h.cursor-1])
	return h.doc(h), nil
}
func (h *popupHost) Redo() (*InspectionDocument, error) {
	h.restore(h.history[h.cursor])
	h.cursor++
	return h.doc(h), nil
}

func (h *popupHost) Fields(kind string, index int) []InspectionField {
	if kind != "Unit" {
		return nil
	}
	u := h.units[index]
	a, _ := strconv.ParseInt(u["a"], 10, 64)
	owner, _ := strconv.ParseInt(u["owner"], 10, 64)
	return []InspectionField{
		{Key: "a", Label: "Strength", Value: a},
		{Key: "owner", Label: "Owner", Value: owner, Choices: []InspectionChoice{{1, "Self"}, {2, "Enemy"}, {3, "Neutral"}}},
		{Key: "name", Label: "Name", Text: true, TextValue: u["name"]},
	}
}

func (h *popupHost) set(index int, key string, value string) error {
	if key == "a" && strings.HasPrefix(value, "-") {
		return fmt.Errorf("strength %s is below 0", value)
	}
	h.units[index][key] = value
	return nil
}

func (h *popupHost) SetField(kind string, index int, key string, value int64) (*InspectionDocument, error) {
	h.calls = append(h.calls, "SetField "+key)
	if err := h.set(index, key, strconv.FormatInt(value, 10)); err != nil {
		return nil, err
	}
	return h.commit(), nil
}

func (h *popupHost) SetTextField(kind string, index int, key, value string) (*InspectionDocument, error) {
	h.calls = append(h.calls, "SetTextField "+key)
	if err := h.set(index, key, value); err != nil {
		return nil, err
	}
	return h.commit(), nil
}

func (h *popupHost) SetFields(kind string, index int, values []InspectionFieldValue) (*InspectionDocument, error) {
	h.calls = append(h.calls, fmt.Sprintf("SetFields %d", len(values)))
	before := h.bytes()
	for _, v := range values {
		value := v.TextValue
		if !v.Text {
			value = strconv.FormatInt(v.Value, 10)
		}
		if err := h.set(index, v.Key, value); err != nil {
			h.restore(before)
			return nil, err
		}
	}
	return h.commit(), nil
}

func popupEditor(t *testing.T) (*MapEditor, *popupHost) {
	t.Helper()
	h := &popupHost{}
	for i := 0; i < 6; i++ {
		h.units = append(h.units, map[string]string{"a": "10", "owner": "1", "name": fmt.Sprintf("u%d", i)})
	}
	h.doc = func(h *popupHost) *InspectionDocument {
		v, err := NewViewer("popups", terrain.Grid{Width: 60, Height: 60, Tiles: make([]uint16, 3600)}, &terrain.Tileset{})
		if err != nil {
			t.Fatal(err)
		}
		d := &InspectionDocument{Viewer: v, Source: "popups", Width: 60, Height: 60, SourceBytes: []byte(h.bytes()), Edits: h}
		for i := range h.units {
			group := 1 + i%2
			if i == 5 {
				group = 0
			}
			if g, err := strconv.Atoi(h.units[i]["group"]); err == nil {
				group = g
			}
			d.Records = append(d.Records, InspectionRecord{Kind: "Unit", Index: i, Section: 6, Cell: image.Pt(40+i, 40), Spatial: true, Size: image.Pt(1, 1), Label: h.units[i]["name"], Player: 1 + i/3, Group: group})
		}
		d.Records = append(d.Records, InspectionRecord{Kind: "Scenery", Index: 0, Section: 3, Cell: image.Pt(2, 2), Spatial: true, Size: image.Pt(1, 1), Label: "tree"})
		return d
	}
	h.commit()
	e := NewMapEditor(panelFont(), []EditorMap{{Key: "new/40x40", Label: "New 40"}, {Key: "popups", Label: "Popups"}, {Key: "other", Label: "Other"}}, func(source string) (*InspectionDocument, error) {
		return h.doc(h), nil
	})
	e.Layout(1280, 800)
	return e, h
}

// fieldRowRect is the object popup's row of field i.
func fieldRowRect(t *testing.T, e *MapEditor, i int) popupRowRect {
	t.Helper()
	objectLayout(t, e)
	return objectRowRect(t, e, e.obj.first+i)
}

func editorClickAt(e *MapEditor, p image.Point, now time.Time) {
	e.Pointer(Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, now)
	e.Pointer(Input{CursorX: p.X, CursorY: p.Y}, now)
}

func TestEditorMapPickerOpensByListAndMenu(t *testing.T) {
	e, _ := popupEditor(t)
	if e.listKind() != "catalog" {
		t.Fatal("the picker is not open at start")
	}
	s, _ := e.listSpec()
	rows := s.List.Rows()
	if len(rows) != 5 || rows[0].Choosable || !rows[1].Choosable || rows[2].Choosable || e.model.Selection() != 1 {
		t.Fatalf("picker rows %+v selection %d", rows, e.model.Selection())
	}
	_ = e.Key("down")
	if e.model.Selection() != 3 {
		t.Fatalf("Down skipped to %d, not past the heading to 3", e.model.Selection())
	}
	_ = e.Key("end")
	_ = e.Key("home")
	if e.model.Selection() != 1 {
		t.Fatalf("Home selected %d", e.model.Selection())
	}
	now := time.Unix(10, 0)
	editorClickAt(e, listRowPoint(t, e, 3), now)
	if e.doc != nil || e.model.Selection() != 3 {
		t.Fatal("one click opened the map or missed its row")
	}
	editorClickAt(e, listRowPoint(t, e, 3), now.Add(200*time.Millisecond))
	if e.doc == nil || e.catalog {
		t.Fatal("a double click did not open the map")
	}
	chromeClick(e, barPoint(t, e, "map"))
	chromeClick(e, menuPoint(t, e, 1))
	if !e.catalog || e.menu != "" {
		t.Fatal("Map > Open did not show the picker")
	}
	_ = e.Key("escape")
	if e.catalog || e.doc == nil {
		t.Fatal("Esc did not close the picker over the open map")
	}
	chromeClick(e, barPoint(t, e, "map"))
	chromeClick(e, menuPoint(t, e, 1))
	l := listLayout(t, e)
	chromeClick(e, center(l.Close))
	if e.catalog {
		t.Fatal("the close button did not close the picker")
	}
}

func TestEditorObjectPopupApplyIsOneUndoStep(t *testing.T) {
	e, h := popupEditor(t)
	_ = e.Open("popups")
	original := string(e.doc.SourceBytes)
	camera := e.doc.Viewer.cam.X + 1000*e.doc.Viewer.cam.Y
	chromeClick(e, barPoint(t, e, "units"))
	e.listSpec()
	if e.listKind() != "records" || e.model.Len() != 6 {
		t.Fatalf("the Units entry listed %q with %d rows", e.listKind(), e.model.Len())
	}
	chromeClick(e, listRowPoint(t, e, 2))
	if e.selected != 2 || e.doc.Viewer.cam.X+1000*e.doc.Viewer.cam.Y == camera {
		t.Fatal("a list row did not select and centre its unit")
	}
	s, _ := objectLayout(t, e)
	if s.Title != "Unit #2" || e.obj.first == 0 || len(e.obj.fields) != 3 {
		t.Fatalf("object popup %q with fields %d", s.Title, len(e.obj.fields))
	}
	// Strength: an edit field.
	chromeClick(e, center(fieldRowRect(t, e, 0).Control))
	if !e.objectTyping() {
		t.Fatal("a press on the edit field did not focus it")
	}
	_ = e.Key("clear") // a letter command is deaf while typing
	_ = e.Key("backspace")
	_ = e.Key("backspace")
	e.TextInput("42")
	_ = e.Key("clone")
	// Owner: a drop-down.
	owner := fieldRowRect(t, e, 1)
	chromeClick(e, center(owner.Control))
	if !e.dropdownOpen() {
		t.Fatal("the owner drop-down did not open")
	}
	d, _ := e.openDropdown()
	box := d.listBox(e.bodyFont())
	e.Pointer(Input{CursorX: center(box.Row(0)).X, CursorY: center(box.Row(0)).Y, WheelY: -0.4}, time.Time{})
	if d.List.Selection() != 1 {
		t.Fatalf("the wheel moved the drop-down to %d", d.List.Selection())
	}
	chromeClick(e, center(box.Row(2)))
	if e.dropdownOpen() || e.obj.buf[1] != "3" {
		t.Fatalf("choosing Neutral left %q open=%v", e.obj.buf[1], e.dropdownOpen())
	}
	if string(e.doc.SourceBytes) != original || len(h.calls) != 0 {
		t.Fatal("editing the popup changed the map before Apply")
	}
	chromeClick(e, objectButtonPoint(t, e, 0))
	if len(h.calls) != 1 || h.calls[0] != "SetFields 2" || h.units[2]["a"] != "42" || h.units[2]["owner"] != "3" {
		t.Fatalf("Apply wrote %v, unit %v", h.calls, h.units[2])
	}
	if e.selected != 2 || e.obj.buf[0] != "42" {
		t.Fatal("Apply lost the selection or the popup's values")
	}
	chromeClick(e, barPoint(t, e, "undo"))
	if string(e.doc.SourceBytes) != original {
		t.Fatal("one Undo did not restore the bytes")
	}
	// Cancel drops typed values and writes nothing.
	chromeClick(e, center(fieldRowRect(t, e, 2).Control))
	e.TextInput("zz")
	chromeClick(e, objectButtonPoint(t, e, 1))
	if e.obj.buf[2] != "u2" || len(h.calls) != 1 {
		t.Fatal("Cancel kept the typed text or wrote")
	}
	// A refused value shows its reason and writes nothing.
	chromeClick(e, center(fieldRowRect(t, e, 0).Control))
	e.TextInput("0")
	_ = e.Key("backspace")
	_ = e.Key("backspace")
	_ = e.Key("backspace")
	e.TextInput("-5")
	chromeClick(e, objectButtonPoint(t, e, 0))
	s, _ = objectLayout(t, e)
	if string(e.doc.SourceBytes) != original || !strings.Contains(strings.Join(s.Notes, " "), "below 0") || e.message != "" {
		t.Fatalf("refusal: bytes %q notes %v message %q calls %v buf %q note %q", e.doc.SourceBytes, s.Notes, e.message, h.calls, e.obj.buf, e.obj.note)
	}
	// Esc leaves the field, then closes the popup; the selection stays.
	_ = e.Key("escape")
	_ = e.Key("escape")
	if _, _, ok := e.objectSpec(); ok || e.selected != 2 {
		t.Fatal("Esc did not close the object popup")
	}
	chromeClick(e, listRowPoint(t, e, 3))
	_, l := objectLayout(t, e)
	chromeClick(e, center(l.Close))
	if _, _, ok := e.objectSpec(); ok {
		t.Fatal("the close button did not close the object popup")
	}
	chromeClick(e, center(listLayout(t, e).Close))
	if e.listKind() != "" {
		t.Fatal("the close button did not close the list popup")
	}
}

func TestEditorObjectPopupWithoutFieldsIsReadOnly(t *testing.T) {
	e, _ := popupEditor(t)
	_ = e.Open("popups")
	e.Select(6, false)
	s, _ := objectLayout(t, e)
	if len(s.Buttons) != 0 || len(e.obj.fields) != 0 || !strings.Contains(strings.Join(s.Notes, " "), "editor.object.readonly") && !strings.Contains(strings.Join(s.Notes, " "), "no editable") {
		t.Fatalf("scenery popup %+v", s)
	}
}

func TestEditorGroupRowHighlightsExactlyItsMembers(t *testing.T) {
	e, _ := popupEditor(t)
	_ = e.Open("popups")
	chromeClick(e, barPoint(t, e, "groups"))
	e.listSpec()
	rows := e.model.Rows()
	if len(rows) != 4 {
		t.Fatalf("groups %v", rows)
	}
	chromeClick(e, listRowPoint(t, e, 0)) // player 1, group 1: units 0 and 2
	if got := fmt.Sprint(e.GroupMembers()); got != "[0 2]" {
		t.Fatalf("group members %s", got)
	}
	if e.ChromeImage() == nil { // the group popup paints with no record selected
		t.Fatal("no chrome")
	}
	if n := len(e.groupMarkerLines()); n != 8 {
		t.Fatalf("the group marks %d lines, want two boxes", n)
	}
	s, links := func() (popupSpec, []int) { s, l, _ := e.objectSpec(); return s, l }()
	if len(s.Rows) != 4 || links[2] != -1 || links[3] != -3 {
		t.Fatalf("group popup rows %+v links %v", s.Rows, links)
	}
	chromeClick(e, center(objectRowRect(t, e, 3).Rect))
	if e.selected != 2 || len(e.GroupMembers()) != 0 {
		t.Fatal("a member row did not open its unit")
	}
	if s, _, _ := e.objectSpec(); s.Title != "Unit #2" {
		t.Fatalf("member row opened %q", s.Title)
	}
}

func TestEditorViewMenuSwitchesTheAnnotations(t *testing.T) {
	e, _ := popupEditor(t)
	_ = e.Open("popups")
	e.Select(0, false)
	with := len(e.markerLines())
	chromeClick(e, barPoint(t, e, "view"))
	chromeClick(e, menuPoint(t, e, 1)) // footprints
	if without := len(e.markerLines()); without != with-4 {
		t.Fatalf("footprints off: %d lines, was %d", without, with)
	}
	chromeClick(e, barPoint(t, e, "view"))
	chromeClick(e, menuPoint(t, e, 3)) // grid
	e.doc.Viewer.cam.SetZoom(1)
	if !e.view.grid || len(e.gridLines()) == 0 {
		t.Fatal("the grid did not switch on")
	}
}

func TestEditorObjectPopupStepsAsideWhileACellIsPicked(t *testing.T) {
	e, _ := popupEditor(t)
	_ = e.Open("popups")
	chromeClick(e, barPoint(t, e, "units"))
	chromeClick(e, listRowPoint(t, e, 1))
	if _, _, ok := e.objectSpec(); !ok {
		t.Fatal("no object popup")
	}
	e.armPick(pickMove)
	if _, _, ok := e.objectSpec(); ok {
		t.Fatal("the object popup covers the map during a pick")
	}
	_ = e.Key("escape")
	if _, _, ok := e.objectSpec(); !ok {
		t.Fatal("the object popup did not return after the pick")
	}
}
