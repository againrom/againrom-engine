package ui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io/fs"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/render/text"
	"againrom/pkg/words"
)

// symbolBinds answers every binding with its own name: every flag holds,
// every list has one item, a value reads "=name".
type symbolBinds struct{}

func (symbolBinds) when(string) (bool, bool)         { return true, true }
func (symbolBinds) value(n string) (string, bool)    { return "=" + n, true }
func (symbolBinds) key(string) (string, bool)        { return "", false }
func (symbolBinds) field(string) (int, bool)         { return -1, true }
func (symbolBinds) list(string) ([]popupBinds, bool) { return []popupBinds{symbolBinds{}}, true }

func symbolWord(id string) string { return "{" + id + "}" }

// unSymbol is the word id or binding name a symbolic text shows.
func unSymbol(s string) (id, bind string) {
	if b, ok := strings.CutPrefix(s, "="); ok {
		return "", b
	}
	if i := strings.Index(s, "}"); strings.HasPrefix(s, "{") && i > 0 {
		return s[1:i], ""
	}
	return s, ""
}

// descFromSpec reads a description back from the window tree of the spec
// built from shape under symbolic bindings: kinds, word ids, bindings,
// indents, inks and button words from the tree, the editor's fields and
// actions from made, the conditions and lists from shape.
func descFromSpec(shape popupDesc, s popupSpec, made []popupMade) popupDesc {
	w := windowOf(specTree(s))
	d := popupDesc{Window: shape.Window, TitleArgs: shape.TitleArgs, Width: w.Place.Width, LabelWidth: w.Body.LabelWidth,
		MinWidth: w.Place.MinWidth, ButtonW: w.Buttons.Least, Columns: w.Body.Columns, Card: w.Card.Aside > 0, Sample: shape.Sample}
	d.Title, _ = unSymbol(w.Title.Text)
	d.TitleHelp, _ = unSymbol(w.Title.Help)
	for _, t := range w.Tabs.Captions {
		id, _ := unSymbol(t)
		d.Tabs = append(d.Tabs, id)
	}
	rows := w.Body.Rows
	d.Rows = rowsFromSpec(shape.Rows, &rows, &made)
	for i, bt := range shape.Buttons {
		id, _ := unSymbol(w.Buttons.Captions[i])
		got := popupDescButton{Word: id, When: bt.When}
		if w.Buttons.Disabled[i] {
			got.Off = bt.Off
		}
		d.Buttons = append(d.Buttons, got)
	}
	return d
}

func rowsFromSpec(shape []popupDescRow, rows *[]popupRow, made *[]popupMade) []popupDescRow {
	var out []popupDescRow
	roles := map[color.RGBA]string{}
	for _, role := range []string{"condition", "action", "warning"} {
		roles[inspectionRoleColor(role)] = role
	}
	for _, r := range shape {
		if r.Each != "" {
			out = append(out, popupDescRow{Each: r.Each, When: r.When, Rows: rowsFromSpec(r.Rows, rows, made)})
			continue
		}
		m, row := (*made)[0], (*rows)[0]
		*made, *rows = (*made)[1:], (*rows)[1:]
		got := popupDescRow{Kind: popupKindNames[row.Kind], When: r.When, Column: row.Column, Lines: row.Lines, Wrap: m.Wrap, Field: r.Field}
		label := strings.TrimLeft(row.Label, " ")
		got.Indent = len(row.Label) - len(label)
		got.Label, got.LabelBind = unSymbol(label)
		_, got.Value = unSymbol(row.Value)
		got.Hint, got.HintBind = unSymbol(row.Hint)
		got.Help, _ = unSymbol(row.Help)
		if row.Fold == FoldOpen {
			got.Fold = r.Fold // every flag holds, so a fold reads back open
		}
		got.Ink = roles[row.Ink]
		for k, bt := range r.Buttons {
			id, _ := unSymbol(row.Buttons[k])
			got.Buttons = append(got.Buttons, popupDescButton{Word: id, Act: m.Acts[k], When: bt.When})
		}
		out = append(out, got)
	}
	return out
}

func embeddedDescs(t *testing.T) map[string]popupDesc {
	t.Helper()
	names, err := fs.Glob(screenFiles, "screens/*.json")
	if err != nil || len(names) == 0 {
		t.Fatalf("no embedded window description: %v", err)
	}
	out := map[string]popupDesc{}
	for _, name := range names {
		base := strings.TrimSuffix(strings.TrimPrefix(name, "screens/"), ".json")
		d, err := screenDesc(base)
		if err != nil {
			t.Fatal(err)
		}
		if d.Window != base {
			t.Errorf("%s names its window %q", name, d.Window)
		}
		out[base] = d
	}
	return out
}

// descWordIDs are the word ids a description names.
func descWordIDs(d popupDesc) []string {
	ids := append([]string{d.Title, d.TitleHelp}, d.Tabs...)
	for _, b := range d.Buttons {
		ids = append(ids, b.Word)
	}
	var rows func([]popupDescRow)
	rows = func(list []popupDescRow) {
		for _, r := range list {
			ids = append(ids, r.Label, r.Hint, r.Help)
			for _, b := range r.Buttons {
				ids = append(ids, b.Word)
			}
			rows(r.Rows)
		}
	}
	rows(d.Rows)
	var sample func(*popupScope)
	sample = func(s *popupScope) {
		for _, id := range s.Words {
			ids = append(ids, id)
		}
		for _, list := range s.Lists {
			for i := range list {
				sample(&list[i])
			}
		}
	}
	if d.Sample != nil {
		sample(d.Sample)
	}
	var out []string
	for _, id := range ids {
		if id != "" {
			out = append(out, id)
		}
	}
	return out
}

// Every embedded window description reads, names each row kind the popup
// builder has, and names only words both the English and the Russian
// tables hold.
func TestWindowDescriptionsNameKindsAndWords(t *testing.T) {
	if len(popupKindNames) != int(popupDirection)+1 {
		t.Fatalf("%d kind names for %d kinds", len(popupKindNames), popupDirection+1)
	}
	for k, name := range popupKindNames {
		if got, ok := popupKindNamed(name); !ok || got != popupRowKind(k) || name == "" {
			t.Errorf("kind %d named %q reads back as %d", k, name, got)
		}
	}
	tables := map[string][]byte{}
	for _, lang := range []string{"en", "ru"} {
		data, err := fs.ReadFile(words.Engine(), "text/"+lang+"/strings.toml")
		if err != nil {
			t.Fatal(err)
		}
		tables[lang] = data
	}
	for name, d := range embeddedDescs(t) {
		for _, id := range descWordIDs(d) {
			for lang, table := range tables {
				if !bytes.Contains(table, []byte("\n\""+id+"\" = ")) {
					t.Errorf("%s: the %s table lacks %q", name, lang, id)
				}
			}
		}
	}
	if _, err := parsePopupDesc([]byte(`{"window":"x","rows":[{"kind":"slider"}]}`)); err == nil {
		t.Error("an unknown row kind reads")
	}
	if _, err := parsePopupDesc([]byte(`{"window":"x","rows":[{"kind":"text","colour":"red"}]}`)); err == nil {
		t.Error("an unknown field reads")
	}
}

// A description built into a popupSpec under bindings that answer with
// their own names reads back from that spec as the same description.
func TestWindowDescriptionRoundTripsThroughTheSpec(t *testing.T) {
	for name, d := range embeddedDescs(t) {
		s, made := d.build(symbolBinds{}, symbolWord)
		if got := descFromSpec(d, s, made); !reflect.DeepEqual(got, d) {
			t.Errorf("%s: read back from its spec as\n%s\nnot\n%s", name, got.encode(), d.encode())
		}
		if len(s.Rows) != len(made) {
			t.Errorf("%s: %d spec rows for %d built rows", name, len(s.Rows), len(made))
		}
	}
}

// A page carries its description: read back it is the same description,
// and a page whose description was edited gives the edited one, which the
// editor lays out.
func TestWindowDescriptionRoundTripsThroughThePage(t *testing.T) {
	for name, d := range embeddedDescs(t) {
		m, err := NewEditorMockup(d.encode(), "ru", image.Pt(800, 600), nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		page := m.Page()
		for _, asset := range []string{"<img", "url(", "@font-face", "base64", "<link", "src="} {
			if bytes.Contains(page, []byte(asset)) {
				t.Errorf("%s: the page holds %q; it carries no image, font file or outside resource", name, asset)
			}
		}
		got, err := readPopupPage(page)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, d) {
			t.Errorf("%s: the page gives\n%s\nnot\n%s", name, got.encode(), d.encode())
		}
		file, err := PageDescription(page)
		if err != nil || !bytes.Equal(file, d.encode()) {
			t.Errorf("%s: the page's description file differs: %v", name, err)
		}
	}
	m, err := NewEditorMockup(triggerWizardDesc().encode(), "en", image.Pt(800, 600), nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	edited := bytes.Replace(m.Page(), []byte(`"label":"editor.trigger.name"`), []byte(`"label":"editor.trigger.comparison"`), 1)
	file, err := PageDescription(edited)
	if err != nil {
		t.Fatal(err)
	}
	again, err := NewEditorMockup(file, "en", image.Pt(800, 600), nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := again.plain.Rows[0].Label; got != words.For("en").Text("editor.trigger.comparison") {
		t.Errorf("the edited page's first row reads %q", got)
	}
}

var pageBox = regexp.MustCompile(`data-part="([a-z-]+)"(?: data-row="(\d+)")? data-box="(-?\d+) (-?\d+) (\d+) (\d+)"`)

// pagePart is one placed element of a page.
type pagePart struct {
	part string
	box  image.Rectangle
}

func layoutParts(s popupSpec, l popupLayout) []pagePart {
	var out []pagePart
	add := func(part string, r image.Rectangle) {
		if !r.Empty() {
			out = append(out, pagePart{part, r})
		}
	}
	add("window", l.Rect)
	add("title", l.Title)
	add("close", l.Close)
	add("title-help", l.TitleHelp)
	for _, r := range l.Tabs {
		add("tab", r)
	}
	for _, rr := range l.Rows {
		row := s.Rows[rr.Index]
		if row.Kind == popupHeading {
			add("heading", rr.Rect)
			continue
		}
		add("label", rr.Label)
		switch row.Kind {
		case popupEdit, popupPick, popupMultiline, popupChoice, popupCard:
			add(popupKindNames[row.Kind], rr.Control)
		case popupSwitch:
			add("switch", rr.Control)
		default:
			add("value", rr.Control)
		}
		for _, b := range rr.Buttons {
			add("row-button", b)
		}
		add("row-help", rr.Help)
	}
	for _, r := range l.GridColHeads {
		add("column-head", r)
	}
	if s.Grid != nil && l.GridCols > 0 {
		var heads []string
		for k, row := range s.Grid.rows(l.GridCols) {
			if k >= l.GridTop && (row.Head != "" || len(s.Grid.Groups) == 0 && k < len(s.Grid.RowHeads)) {
				heads = append(heads, row.Head)
			}
		}
		for i, r := range l.GridRowHeads {
			if i < len(heads) && heads[i] != "" {
				add("grid-head", r)
			}
		}
	}
	for _, r := range l.GridCells {
		add("cell", r)
	}
	for _, r := range l.Notes {
		add("note", r)
	}
	if l.HasList && s.List != nil {
		top, shown := s.List.Visible()
		for i := 0; i < l.List.Rows && i < shown && top+i < s.List.Len(); i++ {
			add("list-row", l.List.Row(i))
		}
	}
	add("switch-button", l.SwitchButton)
	for _, r := range l.Buttons {
		add("button", r)
	}
	return out
}

// Every element of the page stands at the rectangle layoutPopup gives it,
// at three window sizes, in both languages, with and without fonts.
func TestPageElementsAreTheLayoutsRectangles(t *testing.T) {
	d := triggerWizardDesc()
	for _, size := range []image.Point{{640, 480}, {800, 600}, {1920, 1080}} {
		for _, lang := range []string{"en", "ru"} {
			for _, font := range []*text.Font{nil, wordsFont(lang)} {
				m, err := NewEditorMockup(d.encode(), lang, size, font, font, nil)
				if err != nil {
					t.Fatal(err)
				}
				want := layoutParts(m.paint, layoutPopup(m.paint, font, font))
				var got []pagePart
				for _, p := range pageBox.FindAllStringSubmatch(string(m.Page()), -1) {
					n := func(i int) int { v, _ := strconv.Atoi(p[i]); return v }
					got = append(got, pagePart{p[1], image.Rect(n(3), n(4), n(3)+n(5), n(4)+n(6))})
				}
				if len(want) < 20 || !reflect.DeepEqual(got, want) {
					t.Errorf("%v %s font %v: page parts\n%v\nlayout\n%v", size, lang, font != nil, got, want)
				}
				if !m.layout.Rect.In(image.Rectangle{Max: size}) {
					t.Errorf("%v %s: the window %v leaves the screen", size, lang, m.layout.Rect)
				}
			}
		}
	}
}

// The picture is the popup painter's window on the screen's ground: paint
// inside the window's rectangle, the ground outside it.
func TestMockupPictureIsThePopupPainters(t *testing.T) {
	m, err := NewEditorMockup(triggerWizardDesc().encode(), "en", image.Pt(800, 600), wordsFont("en"), wordsFont("en"), nil)
	if err != nil {
		t.Fatal(err)
	}
	p := m.Picture()
	inside, outside := 0, 0
	for y := p.Rect.Min.Y; y < p.Rect.Max.Y; y++ {
		for x := p.Rect.Min.X; x < p.Rect.Max.X; x++ {
			ground := p.RGBAAt(x, y) == editorPopupInk.TitleGround
			switch in := image.Pt(x, y).In(m.layout.Rect); {
			case in && !ground:
				inside++
			case !in && !ground:
				outside++
			}
		}
	}
	if inside < m.layout.Rect.Dx()*m.layout.Rect.Dy()/2 || outside != 0 {
		t.Errorf("%d painted pixels inside the window %v, %d outside", inside, m.layout.Rect, outside)
	}
}

var pageInk = regexp.MustCompile(`#[0-9a-f]{6}`)

// TestPageInksAreThePaintersTable: every colour a page names is an ink of
// the table the painter reads, or a row's or the notes' own ink.
func TestPageInksAreThePaintersTable(t *testing.T) {
	hex := func(c color.RGBA) string { return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B) }
	k := editorPopupInk
	if k.Hint != popupHintInk || k.Button != pushButtonIdleInk || k.BevelLight != dialogueBevelLight || k.BevelDark != dialogueBevelDark {
		t.Fatalf("the table's hint and button inks %+v are not the painters'", k)
	}
	for name, d := range embeddedDescs(t) {
		m, err := NewEditorMockup(d.encode(), "en", image.Pt(800, 600), nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		known := map[string]bool{}
		for _, c := range []color.RGBA{k.Ground, k.Edge, k.TitleGround, k.Title, k.Label, k.Value, k.Hint, k.Button, k.BevelLight, k.BevelDark, m.plain.NoteInk} {
			known[hex(c)] = true
		}
		for _, r := range m.plain.Rows {
			known[hex(r.Ink)] = true
		}
		for _, got := range pageInk.FindAllString(string(m.Page()), -1) {
			if !known[got] {
				t.Errorf("%s: the page names %s, an ink outside the painter's table", name, got)
			}
		}
	}
}
