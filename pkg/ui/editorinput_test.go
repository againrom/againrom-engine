package ui

import (
	"againrom/pkg/ui/widgetkit"
	"errors"
	"image"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestEditorInputInsertsAtCaret(t *testing.T) {
	e := NewMapEditor(nil, nil, nil)
	e.openPrompt(editorPrompt{kind: promptOpen, path: "abcd"})
	_ = e.Key("left")
	e.TextInput("X")
	if e.prompt.path != "abcXd" {
		t.Fatalf("native TextInput route appended instead of inserting: %q", e.prompt.path)
	}
}

func TestEditorCRLFDisplayKeepsRawText(t *testing.T) {
	e := NewMapEditor(nil, nil, nil)
	e.SetHostTextEncoder(func(s string) string { return strings.ReplaceAll(s, "\r", "?") })
	const raw = "First\r\nSecond\r\n"
	e.openPrompt(editorPrompt{kind: promptWorkspace, scope: "pages", path: raw})
	s, ok := e.promptSpec()
	if !ok || len(s.Rows) != 3 || s.Rows[0].Value != "First" || s.Rows[1].Value != "Second" || s.Rows[2].Value != "" {
		t.Fatalf("CR reached font encoding: %+v", s.Rows)
	}
	if e.prompt.path != raw {
		t.Fatal("rendering normalised stored bytes", e.prompt.path)
	}
}

func TestEditorEndInCRLFLine(t *testing.T) {
	e := NewMapEditor(nil, nil, nil)
	e.openPrompt(editorPrompt{kind: promptWorkspace, scope: "pages", path: "First\r\nSecond"})
	_ = e.Key("ctrl+home")
	_ = e.Key("end")
	e.TextInput("X")
	if e.prompt.path != "FirstX\r\nSecond" {
		t.Fatalf("End split the line ending: %q", e.prompt.path)
	}
}

func TestEditorCRLFMultilineSelectionMatchesLF(t *testing.T) {
	render := func(value string) *image.RGBA {
		pic := image.NewRGBA(image.Rect(0, 0, 240, 100))
		v := &editorTextView{text: value, caret: len([]rune(value)), multiline: true,
			encode: func(s string) string { return strings.ReplaceAll(s, "\r", "?") }}
		row := popupRow{Kind: popupMultiline, Focus: true, Edit: v}
		drawEditorInput(pic, panelFont(), row, pic.Bounds(), editorPopupInk, editorLight)
		return pic
	}
	if !reflect.DeepEqual(render("First\r\nSecond").Pix, render("First\nSecond").Pix) {
		t.Fatal("CRLF changed selected body pixels")
	}
}

func TestEditorCRLFNavigationAndDeletion(t *testing.T) {
	var m TextInputState
	m.Reset("ab\r\nя")
	m.Drag(4, false)
	m.Move("left", false, false)
	if m.Caret != 2 {
		t.Fatal("Left entered CRLF", m.Caret)
	}
	m.Move("right", false, false)
	if m.Caret != 4 {
		t.Fatal("Right entered CRLF", m.Caret)
	}
	m.Erase(true)
	if m.Text != "abя" {
		t.Fatal("Backspace split CRLF", m.Text)
	}
	m.Reset("ab\r\nя")
	m.Drag(2, false)
	m.Erase(false)
	if m.Text != "abя" {
		t.Fatal("Delete split CRLF", m.Text)
	}
	m.Reset("a\r\nlong")
	m.Move("up", false, false)
	if m.Caret != 1 {
		t.Fatal("Up entered CRLF", m.Caret)
	}
	m.Move("down", false, false)
	if m.Caret != 4 {
		t.Fatal("Down lost the logical column", m.Caret)
	}
}

func TestEditorRepeatClock(t *testing.T) {
	for _, key := range []string{"backspace", "delete", "left", "right", "up", "down"} {
		t.Run(key, func(t *testing.T) {
			var m TextInputState
			clock := time.Unix(100, 0)
			held := map[string]bool{key: true}
			for _, c := range []struct {
				ms   int
				want bool
			}{{0, true}, {399, false}, {400, true}, {439, false}, {440, true}, {800, true}, {801, false}} {
				got := m.Repeat(clock.Add(time.Duration(c.ms)*time.Millisecond), held, gateAtLeast.reached)
				if (len(got) == 1) != c.want {
					t.Fatalf("%d ms: %v, want fire=%v", c.ms, got, c.want)
				}
			}
			m.Repeat(clock.Add(time.Second), nil, gateAtLeast.reached)
			if got := m.Repeat(clock.Add(time.Second), held, gateAtLeast.reached); !reflect.DeepEqual(got, []string{key}) {
				t.Fatal("release did not rearm", got)
			}
		})
	}
}

func TestEditorRepeatKeepsDelayWhenChoiceChangesText(t *testing.T) {
	e := NewMapEditor(nil, nil, nil)
	e.openPrompt(editorPrompt{kind: promptOpen, path: "1"})
	m, _ := e.inputModel()
	now := time.Unix(100, 0)
	m.Repeat(now, map[string]bool{"down": true}, gateAtLeast.reached)
	e.prompt.path = "2"
	m, _ = e.inputModel()
	if keys := m.Repeat(now.Add(399*time.Millisecond), map[string]bool{"down": true}, gateAtLeast.reached); len(keys) != 0 {
		t.Fatal("choice restarted repeat immediately", keys)
	}
	if keys := m.Repeat(now.Add(400*time.Millisecond), map[string]bool{"down": true}, gateAtLeast.reached); len(keys) != 1 {
		t.Fatal("choice lost repeat deadline", keys)
	}
}

func TestEditorWordJumps(t *testing.T) {
	var m TextInputState
	m.Reset("Latin, два_3! four")
	for _, want := range []string{"Latin, два_3! ", "Latin, два_3", "Latin, два_", "Latin, два", "Latin, ", "Latin", ""} {
		m.Move("left", true, false)
		if got := string([]rune(m.Text)[:m.Caret]); got != want {
			t.Fatalf("Ctrl+Left: %q, want %q", got, want)
		}
	}
	for _, want := range []string{"Latin", "Latin,", "Latin, два", "Latin, два_", "Latin, два_3", "Latin, два_3!", "Latin, два_3! four"} {
		m.Move("right", true, false)
		if got := string([]rune(m.Text)[:m.Caret]); got != want {
			t.Fatalf("Ctrl+Right: %q, want %q", got, want)
		}
	}
	m.Move("left", true, true)
	if m.Selected() != "four" {
		t.Fatal("Ctrl+Shift+Left selection", m.Selected())
	}
	m.Replace("три")
	if m.Text != "Latin, два_3! три" {
		t.Fatal("typing over selection", m.Text)
	}
}

func TestEditorSelectionAndErase(t *testing.T) {
	var m TextInputState
	m.Reset("abcd\nязык")
	m.Move("home", false, true)
	if m.Selected() != "язык" {
		t.Fatal("Shift+Home", m.Selected())
	}
	m.Erase(false)
	if m.Text != "abcd\n" {
		t.Fatal(m.Text)
	}
	m.Move("home", true, false)
	m.Move("end", false, true)
	if m.Selected() != "abcd" {
		t.Fatal("Shift+End", m.Selected())
	}
	m.Erase(true)
	if m.Text != "\n" {
		t.Fatal(m.Text)
	}
	m.Reset("one\nдва\nthree")
	m.Move("up", false, true)
	if m.Selected() != "\nthree" {
		t.Fatal("Shift+Up", m.Selected())
	}
	m.Drag(0, false)
	m.Drag(3, true)
	if m.Selected() != "one" {
		t.Fatal("drag", m.Selected())
	}
	m.Replace("два")
	if m.Text != "два\nдва\nthree" {
		t.Fatal(m.Text)
	}
}

type fakeEditorClipboard struct {
	widgetkit.MemoryClipboard
	err error
}

func (c *fakeEditorClipboard) Read() (string, error) {
	if c.err != nil {
		return "", c.err
	}
	return c.MemoryClipboard.Read()
}
func (c *fakeEditorClipboard) Write(s string) error {
	if c.err != nil {
		return c.err
	}
	return c.MemoryClipboard.Write(s)
}

func fakeTextClipboard(t *testing.T) *fakeEditorClipboard {
	t.Helper()
	saved := widgetkit.TextClipboard
	c := &fakeEditorClipboard{}
	widgetkit.TextClipboard = c
	t.Cleanup(func() { widgetkit.TextClipboard = saved })
	return c
}

func TestEditorClipboardSelection(t *testing.T) {
	c := fakeTextClipboard(t)
	e := NewMapEditor(nil, nil, nil)
	e.openPrompt(editorPrompt{kind: promptOpen, path: "Путь"})
	_ = e.Key("selectall")
	if e.prompt.path != "Путь" {
		t.Fatal("Ctrl+A erased text")
	}
	_ = e.Key("copy")
	if c.Value != "Путь" {
		t.Fatal("copy", c.Value)
	}
	_ = e.Key("cut")
	if e.prompt.path != "" {
		t.Fatal("cut", e.prompt.path)
	}
	_ = e.Key("paste")
	if e.prompt.path != "Путь" {
		t.Fatal("paste", e.prompt.path)
	}
	c.err = errors.New("clipboard unavailable")
	_ = e.Key("selectall")
	_ = e.Key("cut")
	if e.prompt.path != "Путь" {
		t.Fatal("failed copy lost selection")
	}
	_ = e.Key("paste")
	if e.prompt.path != "Путь" {
		t.Fatal("failed paste lost selection")
	}
}

func TestEditorPasteFilteringAndLimit(t *testing.T) {
	var m TextInputState
	m.Reset("AB")
	p := editorInputPolicy{limit: 5, bytes: func(r rune) int {
		if r == '🙂' || r == 0 {
			return 0
		}
		return 1
	}}
	if got := editorPaste(&m, p, "я🙂\r\nдлинно"); got != "я д" {
		t.Fatal("single-line limit/filter", got)
	}
	p.multiline = true
	if got := editorPaste(&m, p, "я🙂\r\nдлинно"); got != "я\nд" {
		t.Fatal("multiline limit/filter", got)
	}
	m.Anchor, m.Caret = 0, 2
	if got := editorPaste(&m, p, "123456"); got != "12345" {
		t.Fatal("selection replacement capacity", got)
	}
	p.bytes = editorHexBytes
	if got := editorPaste(&m, p, "0xFF\x01"); got != "0xFF" {
		t.Fatal("hex/control filtering", got)
	}
	m.Reset("x\r\n")
	p = editorInputPolicy{multiline: true, limit: 4, bytes: func(r rune) int {
		if r == '\r' || r == '\n' {
			return 0
		}
		return 1
	}}
	if got := editorPaste(&m, p, "yz"); got != "y" {
		t.Fatal("existing CRLF bytes exceeded paste capacity", got)
	}
}

func TestEditorInputsUseOneModel(t *testing.T) {
	for _, c := range []struct {
		name  string
		kind  promptKind
		scope string
	}{
		{"open-path", promptOpen, ""}, {"save-path", promptSave, ""},
		{"new-map-size", promptRectangle, ""}, {"forest-density", promptForestShare, ""},
		{"random-seed", promptSeed, "randomterrain"}, {"new-map-height", promptNewMode, ""},
		{"record-field", promptField, ""}, {"item-recipe", promptItemRecipe, ""},
		{"metadata", promptWorkspace, "metadata"}, {"page-file", promptWorkspace, "pages"},
		{"page-body", promptWorkspace, "pages"}, {"package-folder", promptWorkspace, "package"},
		{"package-id", promptWorkspace, "package"}, {"package-title", promptWorkspace, "package"},
		{"package-route", promptWorkspace, "package"}, {"package-number", promptWorkspace, "package"},
		{"add-field", promptAdd, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			clip := fakeTextClipboard(t)
			e := NewMapEditor(nil, nil, nil)
			e.openPrompt(editorPrompt{kind: c.kind, scope: c.scope, path: "one два"})
			_ = e.Key("ctrl+shift+left")
			e.TextInput("три")
			if e.prompt.path != "one три" {
				t.Fatal("model route", e.prompt.path)
			}
			if c.kind == promptSeed || c.kind == promptNewMode {
				s, _ := e.promptSpec()
				if len(s.Rows) != 1 || s.Rows[0].Edit == nil {
					t.Fatal("input has no shared drawing or pointer selection route")
				}
			}
			_ = e.Key("selectall")
			_ = e.Key("copy")
			if clip.Value != "one три" {
				t.Fatal("text copy route", clip.Value)
			}
			_ = e.Key("cut")
			clip.Value = "12"
			_ = e.Key("paste")
			if e.prompt.path != "12" {
				t.Fatal("text paste route", e.prompt.path)
			}
		})
	}
	for _, kind := range []string{"Unit", "Structure", "Sack", "Stock", "Player", "Enchantment", "Trigger", "Condition", "Action", "message-body", "speaker-number"} {
		t.Run(kind, func(t *testing.T) {
			c := fakeTextClipboard(t)
			e := NewMapEditor(nil, nil, nil)
			e.doc = &InspectionDocument{Records: []InspectionRecord{{Kind: RecordKind(kind)}}}
			e.selected = 0
			e.obj = objectEdit{doc: e.doc, record: 0, focus: 0, open: -1, fields: []InspectionField{{Key: "name", Text: true}}, buf: []string{"one два"}}
			_ = e.Key("ctrl+shift+left")
			e.TextInput("три")
			if e.obj.buf[0] != "one три" {
				t.Fatal("object model route", e.obj.buf)
			}
			_ = e.Key("selectall")
			_ = e.Key("copy")
			if c.Value != "one три" {
				t.Fatal("object clipboard route", c.Value)
			}
		})
	}
}

func TestEditorInputPointerSelectionAndDraw(t *testing.T) {
	e := NewMapEditor(panelFont(), nil, nil)
	e.Layout(640, 480)
	e.openPrompt(editorPrompt{kind: promptOpen, path: "abcdef"})
	s, _ := e.promptSpec()
	l := layoutPopup(s, e.font, e.bodyFont())
	box := l.Rows[0].Control
	start := image.Pt(box.Min.X+4, box.Min.Y+box.Dy()/2)
	end := image.Pt(start.X+e.textMeasure("abc"), start.Y)
	e.Pointer(Input{CursorX: start.X, CursorY: start.Y, PrimaryDown: true}, time.Time{})
	e.Pointer(Input{CursorX: end.X, CursorY: end.Y, PrimaryDown: true}, time.Time{})
	e.Pointer(Input{CursorX: end.X, CursorY: end.Y}, time.Time{})
	if got := e.input.model.Selected(); got != "abc" {
		t.Fatal("native pointer route", got)
	}
	selected := e.ChromeImage()
	_ = e.Key("right")
	if reflect.DeepEqual(selected.Pix, e.ChromeImage().Pix) {
		t.Fatal("selection was not painted over unchanged text at the same caret")
	}
	_ = e.Key("shift+home")
	e.TextInput("Я")
	if e.prompt.path != "Яdef" {
		t.Fatal("typing over mouse selection", e.prompt.path)
	}
	if reflect.DeepEqual(selected.Pix, e.ChromeImage().Pix) {
		t.Fatal("selection/input did not affect drawing")
	}
}

func TestEditorNoInputKeepsMapClipboard(t *testing.T) {
	c := fakeTextClipboard(t)
	e := NewMapEditor(nil, nil, nil)
	for _, key := range []string{"copy", "paste", "selectall"} {
		if e.textKey(key) {
			t.Fatal("text consumed map key", key)
		}
	}
	if c.Value != "" {
		t.Fatal("map keys reached system clipboard")
	}
}

func TestEditorTextSpansRetainWhitespaceAndBoundaries(t *testing.T) {
	s := "one  два\n\nlongword"
	spans := editorTextSpans(s, 5, func(s string) int { return len([]rune(s)) })
	r := []rune(s)
	var out []string
	for _, span := range spans {
		out = append(out, string(r[span.from:span.to]))
	}
	if strings.Join(out, "|") != "one  |два||longw|ord" {
		t.Fatal(out)
	}
	spans = editorTextSpans("ab\rX", 3, func(s string) int { return len([]rune(s)) })
	if spans[0].to != 3 {
		t.Fatal("soft wrap dropped an embedded CR", spans)
	}
}
