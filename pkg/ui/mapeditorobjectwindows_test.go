package ui

import (
	"fmt"
	"image"
	"slices"
	"strings"
	"testing"
)

// windowsScenery is the scenery record objectWindowsEditor adds.
const windowsScenery = 5

// objectWindowsEditor is the views editor in lang with pictures on the
// structure and the sack, the structure named by script number 3, a scenery
// record of forty lines, a group of twelve units of player 2, and uses uses
// of the structure and of the group. The first added trigger's name is long
// enough to wrap its used-by line.
func objectWindowsEditor(t *testing.T, lang string, uses int) *MapEditor {
	t.Helper()
	e, _ := viewsEditor(t, lang)
	d := e.doc
	d.Records[viewsStructure].Preview = syntheticStaticFrame(24, 24)
	d.Records[viewsStructure].ScriptID = 3
	d.Records[viewsSack].Preview = syntheticStaticFrame(16, 16)
	scenery := InspectionRecord{Kind: KindScenery, Section: 5, Cell: image.Pt(30, 30), Spatial: true, Size: image.Pt(1, 1), Label: "Tree", Preview: syntheticStaticFrame(16, 16)}
	for i := range 40 {
		scenery.Details = append(scenery.Details, fmt.Sprintf("Detail line %d", i+1))
	}
	d.Records = append(d.Records, scenery)
	for i := range 12 {
		d.Records = append(d.Records, InspectionRecord{Kind: KindUnit, Index: 1 + i, Section: 6, Cell: image.Pt(4+i, 30), Spatial: true, Size: image.Pt(1, 1), Label: "Guard", Player: 2, Group: 1, Preview: syntheticStaticFrame(8, 8)})
	}
	m := d.Script
	target := &m.Targets[scriptStructure][0]
	for i := range uses - 1 {
		action, trigger := len(d.Records), len(d.Records)+1
		d.Records = append(d.Records, InspectionRecord{Kind: KindAction, Index: 1 + i, Section: 7}, InspectionRecord{Kind: KindTrigger, Index: 1 + i, Section: 7})
		m.Actions = append(m.Actions, InspectionScriptNode{Record: action, ID: uint32(2 + i), Opcode: 32, Label: fmt.Sprintf("Burn the hall %d", i+1), Types: [10]uint32{scriptStructure}, Values: [10]uint32{3}})
		name := fmt.Sprintf("Raid %d", i+1)
		if i == 0 {
			name = strings.Repeat("Raid on the northern hall ", 8)
		}
		m.Triggers = append(m.Triggers, InspectionTrigger{Record: trigger, Name: name, Left: make([]uint32, 3), Right: make([]uint32, 3), Cmp: make([]uint32, 3), Acts: []uint32{uint32(2 + i), 0, 0, 0}})
		target.Uses = append(target.Uses, InspectionScriptUse{Node: action, Triggers: []int{trigger}})
	}
	m.Targets[scriptGroup] = append(m.Targets[scriptGroup], InspectionScriptTarget{Value: 1, Record: -1, Members: 12, Uses: target.Uses})
	// Back finds a record by its key, as a loaded map gives each record.
	for i := range d.Records {
		d.Records[i].Key = uint64(i + 1)
	}
	return e
}

// windowsGroup is objectWindowsEditor's group of player 2.
func windowsGroup(t *testing.T, e *MapEditor) editorGroup {
	t.Helper()
	for _, g := range e.groups() {
		if g.player == 2 && g.group == 1 {
			return g
		}
	}
	t.Fatal("no group 1 of player 2")
	return editorGroup{}
}

// objectWindowScreens are the six object windows, each opened on e.
var objectWindowScreens = []struct {
	name string
	open func(*testing.T, *MapEditor)
}{
	{"structure", func(t *testing.T, e *MapEditor) { e.Select(viewsStructure, false) }},
	{"structure detailed", func(t *testing.T, e *MapEditor) { e.props.plain = false; e.Select(viewsStructure, false) }},
	{"sack", func(t *testing.T, e *MapEditor) { e.Select(viewsSack, false); e.toggleBlock("details") }},
	{"sack detailed", func(t *testing.T, e *MapEditor) { e.props.plain = false; e.Select(viewsSack, false) }},
	{"record", func(t *testing.T, e *MapEditor) { e.Select(windowsScenery, false) }},
	{"group", func(t *testing.T, e *MapEditor) { e.chooseGroup(windowsGroup(t, e), false) }},
	{"navigation", func(t *testing.T, e *MapEditor) {
		e.Select(viewsStructure, false)
		e.openObjectUse(viewsSack)
	}},
	{"used by", func(t *testing.T, e *MapEditor) {
		e.Select(viewsStructure, false)
		e.objectSpec()
		e.obj.usesTurned = true
	}},
}

// Each object window lies inside the screen at 640x480, 800x600 and
// 1920x1080 with twelve uses, twelve members and forty lines, shows whole
// labels at 1920x1080, hits each of its rectangles, and its HTML page boxes
// equal its layout, in English and Russian.
func TestObjectWindowsHoldTheLayoutProperties(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, screen := range objectWindowScreens {
			for _, size := range []image.Point{{640, 480}, {800, 600}, {1920, 1080}} {
				t.Run(fmt.Sprintf("%s %s %v", lang, screen.name, size), func(t *testing.T) {
					e := objectWindowsEditor(t, lang, 12)
					e.props.plain = true
					e.Layout(size.X, size.Y)
					screen.open(t, e)
					s, _, ok := e.objectSpec()
					if !ok {
						t.Fatal("no window")
					}
					checkWindowProperties(t, e, s, size)
				})
			}
		}
	}
}

// (1) The record's picture is the window's first row, three rows high, its
// box in the label column, in the plain and the detailed view.
func TestObjectWindowPictureIsTheFirstRow(t *testing.T) {
	for _, record := range []int{viewsStructure, viewsSack, windowsScenery} {
		for _, plain := range []bool{true, false} {
			e := objectWindowsEditor(t, "en", 1)
			e.props.plain = plain
			e.Select(record, false)
			s, l := objectLayout(t, e)
			if len(s.Rows) < 2 || s.Rows[0].Portrait == nil || s.Rows[0].Lines != 3 {
				t.Fatalf("record %d plain %v: the first row is %+v, want the picture three rows high", record, plain, s.Rows[0])
			}
			if slices.ContainsFunc(s.Rows[1:], func(r popupRow) bool { return r.Portrait != nil }) {
				t.Errorf("record %d plain %v: a second picture row", record, plain)
			}
			rr := l.Rows[0]
			if box := rowPortraitBox(rr); rr.Index != 0 || box.Empty() || box.Max.X > rr.Label.Max.X || box.Dy() < 2*l.Rows[1].Rect.Dy() {
				t.Errorf("record %d plain %v: the picture box %v of row %v is not the label column three rows high", record, plain, box, rr)
			}
		}
	}
}

// (2) A structure the scripts name by number is titled by that number and
// its name.
func TestObjectWindowStructureTitleIsTheScriptName(t *testing.T) {
	for lang, want := range map[string]string{"en": "Structure 3: Hall", "ru": "Постройка 3: Hall"} {
		e := objectWindowsEditor(t, lang, 1)
		e.Select(viewsStructure, false)
		if s, _ := objectLayout(t, e); s.Title != e.editorHostText(want) {
			t.Errorf("%s: title %q, want %q", lang, s.Title, want)
		}
	}
}

// (3) The sack's Details fold names an item's dispatch in Russian words.
func TestObjectWindowSackDetailsAreRussian(t *testing.T) {
	e := objectWindowsEditor(t, "ru", 1)
	e.props.plain = true
	e.Select(viewsSack, false)
	e.toggleBlock("details")
	s, _ := objectLayout(t, e)
	var labels []string
	for _, r := range s.Rows {
		labels = append(labels, r.Label)
	}
	if !slices.Contains(labels, e.editorHostText("Предмет 1: встроенный вызов")) {
		t.Errorf("the Details fold has no Russian dispatch row: %q", labels)
	}
	if slices.ContainsFunc(labels, func(l string) bool { return strings.Contains(l, "native dispatch") }) {
		t.Errorf("the Details fold keeps the English dispatch row: %q", labels)
	}
}

// (4) A record without editable fields says so behind the title's (?).
func TestObjectWindowReadOnlyNoteIsTheTitleHelp(t *testing.T) {
	e := objectWindowsEditor(t, "ru", 1)
	e.Select(windowsScenery, false)
	s, l := objectLayout(t, e)
	if s.TitleHelp != e.editorHostText("У этой записи нет изменяемых полей.") || len(s.Notes) != 0 || l.TitleHelp.Empty() {
		t.Errorf("title help %q notes %q mark %v", s.TitleHelp, s.Notes, l.TitleHelp)
	}
}

// (5) The group window's members row is "Members N" with the members'
// icons, and the member links follow it.
func TestObjectWindowGroupMembersRow(t *testing.T) {
	e := objectWindowsEditor(t, "ru", 1)
	e.Layout(1920, 1080)
	g := windowsGroup(t, e)
	e.chooseGroup(g, false)
	s, links, ok := e.groupSpec()
	if !ok {
		t.Fatal("no group window")
	}
	row := s.Rows[1]
	if row.Label != e.editorHostText("Состав") || row.Value != "12" || len(row.Icons) != 12 {
		t.Fatalf("members row %q %q with %d icons", row.Label, row.Value, len(row.Icons))
	}
	for i, id := range g.members {
		if links[2+i] != -(id + 1) {
			t.Errorf("row %d links %d, want member %d", 2+i, links[2+i], id)
		}
	}
}

// (6) A used-by line wider than the window wraps under its link, and a
// press on its second line opens the use.
func TestObjectWindowWrappedUseIsOneLink(t *testing.T) {
	e := objectWindowsEditor(t, "en", 2)
	e.Layout(800, 600)
	e.Select(viewsStructure, false)
	trigger := e.doc.Script.Triggers[1].Record
	_, links, _ := e.objectSpec()
	var rows []int
	for n, l := range links {
		if l == -(trigger + 1) {
			rows = append(rows, n)
		}
	}
	if len(rows) < 2 {
		t.Fatalf("the long use takes rows %v, want it wrapped", rows)
	}
	chromeClick(e, center(objectRowRect(t, e, rows[1]).Rect))
	if e.selected != trigger || len(e.objectBack) != 1 {
		t.Fatalf("the second line opened record %d with back %d, want %d", e.selected, len(e.objectBack), trigger)
	}
}

// (7) A window opened from a link offers Back as its one navigation
// button, last in the row, and Back returns.
func TestObjectWindowNavigationIsBackOnly(t *testing.T) {
	e := objectWindowsEditor(t, "en", 1)
	e.Select(viewsStructure, false)
	e.openObjectUse(viewsSack)
	s, _ := objectLayout(t, e)
	back := e.word("editor.trigger.back")
	if n := len(s.Buttons); n == 0 || s.Buttons[n-1] != back || slices.Index(s.Buttons, back) != n-1 {
		t.Fatalf("buttons %q, want one Back, last", s.Buttons)
	}
	for _, b := range s.Buttons {
		if strings.Contains(strings.ToLower(b), "forward") {
			t.Errorf("buttons %q offer forward", s.Buttons)
		}
	}
	chromeClick(e, objectButtonPoint(t, e, len(s.Buttons)-1))
	if e.selected != viewsStructure || len(e.objectBack) != 0 {
		t.Fatalf("Back opened %d with back %d", e.selected, len(e.objectBack))
	}
}

// (8) The used-by block is a fold, closed over five uses and open at five or
// fewer; a press on its heading turns it.
func TestObjectWindowUsedByFold(t *testing.T) {
	for _, c := range []struct {
		uses int
		want RowFold
	}{{5, FoldOpen}, {6, FoldClosed}} {
		e := objectWindowsEditor(t, "en", c.uses)
		e.Layout(1920, 1080)
		e.Select(viewsStructure, false)
		s, links, _ := e.objectSpec()
		head := e.obj.usesFold - 1
		if head < 0 || s.Rows[head].Fold != c.want {
			t.Fatalf("%d uses: fold row %d, want %v", c.uses, head, c.want)
		}
		shown := 0
		for _, l := range links[head+1:] {
			if l < 0 && e.doc.Records[-l-1].Kind.Scripted() {
				shown++
			}
		}
		if (c.want == FoldClosed) != (shown == 0) {
			t.Errorf("%d uses: %d use links under the fold %v", c.uses, shown, c.want)
		}
		chromeClick(e, center(objectRowRect(t, e, head).Rect))
		if s, _ := objectLayout(t, e); s.Rows[head].Fold == c.want {
			t.Errorf("%d uses: the heading press left the fold %v", c.uses, c.want)
		}
	}
}

// (9) The view switch is a button of the button row and names the view.
func TestObjectWindowViewSwitchIsARowButton(t *testing.T) {
	e := objectWindowsEditor(t, "ru", 1)
	e.props.plain = true
	e.Select(viewsStructure, false)
	s, l := objectLayout(t, e)
	if s.SwitchButton != e.editorHostText("Вид: Обычный") || len(l.Buttons) == 0 || l.SwitchButton.Empty() {
		t.Fatalf("switch %q at %v, buttons %v", s.SwitchButton, l.SwitchButton, l.Buttons)
	}
	if l.SwitchButton.Min.Y != l.Buttons[0].Min.Y || l.SwitchButton.Dy() != l.Buttons[0].Dy() {
		t.Errorf("switch %v is not in the button row %v", l.SwitchButton, l.Buttons)
	}
	clickSwitch(t, e)
	if s, _ := objectLayout(t, e); s.SwitchButton != e.editorHostText("Вид: Подробный") {
		t.Errorf("switched caption %q", s.SwitchButton)
	}
}
