package ui

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"image"
	"image/color"

	"againrom/pkg/render/text"
	"againrom/pkg/words"
)

// EditorMockup is an editor window laid out from its description, in a
// window of a given size and language, showing the description's sample:
// one layoutPopup layout that both the page and the picture show.
type EditorMockup struct {
	desc         popupDesc
	lang         string
	window       image.Point
	plain, paint popupSpec
	layout       popupLayout
	title, body  *text.Font
}

// ScreenDescription is the embedded description of the editor window name.
func ScreenDescription(name string) ([]byte, error) {
	d, err := screenDesc(name)
	if err != nil {
		return nil, err
	}
	return d.encode(), nil
}

// NewEditorMockup lays the description out in a window of size window with
// the words of lang. title and body are the editor's fonts and encode its
// host text encoder; nil fonts lay out at layoutPopup's fallback sizes and
// a nil encode keeps the text as it is.
func NewEditorMockup(desc []byte, lang string, window image.Point, title, body *text.Font, encode func(string) string) (*EditorMockup, error) {
	d, err := parsePopupDesc(desc)
	if err != nil {
		return nil, err
	}
	if window.X <= 0 || window.Y <= 0 {
		return nil, fmt.Errorf("window size %dx%d", window.X, window.Y)
	}
	if encode == nil {
		encode = func(s string) string { return s }
	}
	sample := d.Sample
	if sample == nil {
		sample = &popupScope{}
	}
	m := &EditorMockup{desc: d, lang: lang, window: window, title: title, body: body}
	m.plain, _ = d.build(sample, words.For(lang).Text)
	m.paint = encodedSpec(m.plain, encode)
	fitButtons(&m.paint, body)
	m.paint.At, m.paint.Bounds, m.paint.MaxRows = triggerPlace(title, body, window)
	m.plain.Width, m.plain.Bounds, m.plain.At, m.plain.MaxRows = m.paint.Width, m.paint.Bounds, m.paint.At, m.paint.MaxRows
	m.layout = layoutPopup(m.paint, title, body)
	return m, nil
}

// Mockup is NewEditorMockup in the editor's own fonts and host text.
func (e *MapEditor) Mockup(desc []byte, lang string, window image.Point) (*EditorMockup, error) {
	return NewEditorMockup(desc, lang, window, e.font, e.bodyFont(), e.editorHostText)
}

// triggerPlace is where the editor first opens its trigger window in an
// editor window of size window with these fonts: the corner, the bounds and
// the rows it shows with its button row. It asks a bare editor, so the page
// and the picture place the window as the editor does.
func triggerPlace(title, body *text.Font, window image.Point) (at image.Point, bounds image.Rectangle, rows int) {
	e := NewMapEditor(title, nil, nil)
	e.SetBodyFont(body)
	e.Layout(window.X, window.Y)
	return e.placedCorner(popObject, homeTrigger), e.popupBounds(), e.objectRowsShown(true)
}

// encodedSpec is s with its text in the font's bytes.
func encodedSpec(s popupSpec, encode func(string) string) popupSpec {
	all := func(list []string) []string {
		out := make([]string, len(list))
		for i, t := range list {
			out[i] = encode(t)
		}
		return out
	}
	s.Title, s.TitleHelp = encode(s.Title), encode(s.TitleHelp)
	s.Tabs, s.Buttons, s.Notes = all(s.Tabs), all(s.Buttons), all(s.Notes)
	rows := make([]popupRow, len(s.Rows))
	for i, r := range s.Rows {
		r.Label, r.Value, r.Hint, r.Help = encode(r.Label), encode(r.Value), encode(r.Hint), encode(r.Help)
		r.Buttons = all(r.Buttons)
		rows[i] = r
	}
	s.Rows = rows
	return s
}

// Picture is the window painted by the editor's popup painter over the
// popup's title ground.
func (m *EditorMockup) Picture() *image.RGBA {
	dst := image.NewRGBA(image.Rectangle{Max: m.window})
	ground := editorPopupInk.TitleGround
	for i := 0; i < len(dst.Pix); i += 4 {
		dst.Pix[i], dst.Pix[i+1], dst.Pix[i+2], dst.Pix[i+3] = ground.R, ground.G, ground.B, ground.A
	}
	drawPopup(dst, m.title, m.body, nil, m.paint, m.layout, editorPopupInk, image.Point{}, false)
	return dst
}

// Page is the window as one HTML page: every part of the layout placed at
// its rectangle in the popup painter's inks, with the description inside.
func (m *EditorMockup) Page() []byte { return popupPage(m.desc, m.plain, m.layout, m.lang, m.window) }

// pageDescriptionTag opens the description a page carries.
const pageDescriptionTag = `<script type="application/json" id="popup-description">`

// popupPage writes the page from the layout l of s; it lays nothing out.
func popupPage(d popupDesc, s popupSpec, l popupLayout, lang string, window image.Point) []byte {
	ink := editorPopupInk
	hex := func(c color.RGBA) string { return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B) }
	var b bytes.Buffer
	fmt.Fprintf(&b, "<!DOCTYPE html>\n<html lang=\"%s\">\n<head>\n<meta charset=\"utf-8\">\n", html.EscapeString(lang))
	fmt.Fprintf(&b, "<title>%s %dx%d %s</title>\n<style>\n", html.EscapeString(d.Window), window.X, window.Y, html.EscapeString(lang))
	fmt.Fprintf(&b, "body{margin:0;background:%s}\n", hex(ink.TitleGround))
	fmt.Fprintf(&b, "#screen{position:relative;width:%dpx;height:%dpx;overflow:hidden}\n", window.X, window.Y)
	b.WriteString(".p{position:absolute;box-sizing:border-box;overflow:hidden;white-space:pre;font-family:monospace;font-size:12px;padding:0 2px}\n")
	fmt.Fprintf(&b, ".window{background:%s;border:1px solid %s}\n", hex(ink.Ground), hex(ink.Edge))
	fmt.Fprintf(&b, ".title,.heading{background:%s;color:%s}\n.heading{border:1px solid %s}\n", hex(ink.TitleGround), hex(ink.Title), hex(ink.Edge))
	fmt.Fprintf(&b, ".label{color:%s}\n.control{background:%s;border:1px solid %s}\n", hex(ink.Label), hex(ink.Ground), hex(ink.Edge))
	fmt.Fprintf(&b, ".button{text-align:center;color:%s;border:1px solid;border-color:%s %s %s %s}\n",
		hex(pushButtonIdleInk), hex(dialogueBevelLight), hex(dialogueBevelDark), hex(dialogueBevelDark), hex(dialogueBevelLight))
	fmt.Fprintf(&b, ".off{color:%s}\n</style>\n</head>\n<body>\n<div id=\"screen\">\n", hex(popupHintInk))
	part := func(class, name string, row int, r image.Rectangle, ink color.RGBA, txt string) {
		if r.Empty() {
			return
		}
		fmt.Fprintf(&b, `<div class="p %s" data-part="%s"`, class, name)
		if row >= 0 {
			fmt.Fprintf(&b, ` data-row="%d"`, row)
		}
		fmt.Fprintf(&b, ` data-box="%d %d %d %d" style="left:%dpx;top:%dpx;width:%dpx;height:%dpx;line-height:%dpx`,
			r.Min.X, r.Min.Y, r.Dx(), r.Dy(), r.Min.X, r.Min.Y, r.Dx(), r.Dy(), r.Dy())
		if ink != (color.RGBA{}) {
			fmt.Fprintf(&b, ";color:%s", hex(ink))
		}
		fmt.Fprintf(&b, "\">%s</div>\n", html.EscapeString(txt))
	}
	none := color.RGBA{}
	part("window", "window", -1, l.Rect, none, "")
	part("title", "title", -1, l.Title, none, s.Title)
	part("button", "close", -1, l.Close, none, "x")
	part("button", "title-help", -1, l.TitleHelp, none, "?")
	for i, r := range l.Tabs {
		part("button", "tab", i, r, none, s.Tabs[i])
	}
	for _, rr := range l.Rows {
		row := s.Rows[rr.Index]
		value := row.Ink
		if value == none {
			value = ink.Value
		}
		if row.Kind == popupHeading {
			part("heading", "heading", rr.Index, rr.Rect, none, row.Label)
			continue
		}
		part("label", "label", rr.Index, rr.Label, none, row.Label)
		switch row.Kind {
		case popupEdit, popupPick, popupMultiline, popupChoice, popupCard:
			part("control", popupKindNames[row.Kind], rr.Index, rr.Control, value, row.Value)
		case popupSwitch:
			part("button", "switch", rr.Index, rr.Control, none, row.Value)
		default:
			part("value", "value", rr.Index, rr.Control, value, row.Value)
		}
		captions := rowButtons(s, row)
		for k, r := range rr.Buttons {
			class := "button"
			if k < len(rr.Off) && rr.Off[k] {
				class += " off"
			}
			caption := ""
			if k < len(captions) {
				caption = captions[k]
			}
			part(class, "row-button", rr.Index, r, none, caption)
		}
		part("button", "row-help", rr.Index, rr.Help, none, "?")
	}
	for i, r := range l.Notes {
		part("note", "note", i, r, s.NoteInk, s.Notes[i])
	}
	for i, r := range l.Buttons {
		class := "button"
		if i < len(s.Disabled) && s.Disabled[i] {
			class += " off"
		}
		part(class, "button", i, r, none, s.Buttons[i])
	}
	b.WriteString("</div>\n" + pageDescriptionTag)
	desc, err := json.Marshal(d) // escapes <, > and &, so the block cannot end early
	if err != nil {
		panic(err) // a description holds only strings, numbers and lists
	}
	b.Write(desc)
	b.WriteString("</script>\n</body>\n</html>\n")
	return b.Bytes()
}

// readPopupPage is the description a page carries.
func readPopupPage(page []byte) (popupDesc, error) {
	_, after, ok := bytes.Cut(page, []byte(pageDescriptionTag))
	if !ok {
		return popupDesc{}, errors.New("the page carries no window description")
	}
	body, _, ok := bytes.Cut(after, []byte("</script>"))
	if !ok {
		return popupDesc{}, errors.New("the page's window description has no end")
	}
	return parsePopupDesc(bytes.TrimSpace(body))
}

// PageDescription is the description a page carries, or a description
// file itself, as a description file.
func PageDescription(data []byte) ([]byte, error) {
	var d popupDesc
	var err error
	if bytes.Contains(data, []byte(pageDescriptionTag)) {
		d, err = readPopupPage(data)
	} else {
		d, err = parsePopupDesc(data)
	}
	if err != nil {
		return nil, err
	}
	return d.encode(), nil
}
