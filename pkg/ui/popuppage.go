package ui

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"image"
	"image/color"
	"image/png"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/render/text"
	"againrom/pkg/render/textfit"
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
	// doll is the person's figure at the game's size figure, and missing
	// the line its room shows without one.
	doll    *image.RGBA
	figure  image.Point
	missing string
	// pictureNames are the names of the pictures the rows show, by row:
	// a picture row's own first, then the icons after its value.
	pictureNames map[int][]string
	// gridNames are the names of the grid cells' pictures, by cell.
	gridNames []string
	// asideName names the picture beside the list.
	asideName string
	encode    func(string) string
	chrome    *mockupChrome
	panels    *mockupPanels
}

// mockupPerson is the person a description's card and figure show: the
// card, the font it writes in and the doll. A nil card is the sample's.
// picture is the picture a row names, nil for none; name is the install's
// name a sample word key stands for, when it names one.
type mockupPerson struct {
	card     *PanelSubject
	cardFont *text.Font
	doll     *image.RGBA
	picture  func(name string) *image.RGBA
	name     func(key string) (string, bool)
	decode   func(string) string
}

// mockupBlank stands for a picture nobody gave: the row keeps its room and
// shows nothing.
var mockupBlank = image.NewRGBA(image.Rect(0, 0, 1, 1))

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
	return newEditorMockup(desc, lang, window, title, body, encode, mockupPerson{})
}

func newEditorMockup(desc []byte, lang string, window image.Point, title, body *text.Font, encode func(string) string, person mockupPerson) (*EditorMockup, error) {
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
	m := &EditorMockup{desc: d, lang: lang, window: window, title: title, body: body, encode: encode}
	if d.Chrome != nil {
		m.chrome, err = newMockupChrome(*d.Chrome, lang, window, title, body, encode, person.decode)
		return m, err
	}
	if d.Panels != nil {
		m.panels, err = buildMockupPanels(d, lang, window, title, body, encode, person.decode)
		if err != nil {
			return nil, err
		}
		return m, nil
	}
	word := words.For(lang).Text
	if person.name != nil {
		table := word
		word = func(key string) string {
			if n, ok := person.name(key); ok {
				return n
			}
			return table(key)
		}
	}
	var made []popupMade
	m.plain, made = d.build(sample, word)
	if body != nil && m.plain.Width > 0 {
		measure := func(s string) int { return body.Advance(encode(s)) }
		column := (m.plain.Width-2*popupPad)/max(1, m.plain.Columns) - popupPad
		m.plain.Rows, made = madeRowLines(m.plain.Rows, made, lineBox{Measure: measure, Width: column})
	}
	m.pictureNames = rowPictures(m.plain.Rows, made, person.picture)
	_, m.gridNames = popupChain{sample}.grid(d.Grid, word)
	gridPictures(m.plain.Grid, m.gridNames, person.picture)
	if d.Canvas != nil && person.picture != nil {
		if p := person.picture(d.Canvas.Picture); p != nil {
			m.plain.Canvas = &popupCanvas{Sheet: p.Bounds().Size(), Height: p.Bounds().Dy(), Picture: p}
		}
	}
	if g := m.plain.Grid; g != nil && body != nil && !g.HeadFlat {
		// A flow grid row's head stands in a column as wide as its widest
		// head.
		for _, h := range g.RowHeads {
			g.HeadW = max(g.HeadW, body.Advance(encode(h))+8)
		}
	}
	if d.List != nil {
		m.plain.List, m.plain.ListRows, m.plain.ListAside, m.asideName = (popupChain{sample}).listBox(d.List, word, person.picture)
		if person.picture != nil && m.asideName != "" {
			m.plain.ListPicture = person.picture(m.asideName)
		}
	}
	if d.Question {
		p := m.plain
		q := question{title: p.Title, help: p.TitleHelp, rows: p.Rows, list: p.List, listRows: p.ListRows, grid: p.Grid,
			notes: p.Notes, state: p.NoteInk != editorWarning, buttons: p.Buttons, disabled: p.Disabled}
		if d.Grid != nil {
			q.high = d.Grid.High
		}
		m.plain = q.spec(word("editor.question.keys"), title, body, encode, window, nil)
	} else if body != nil && m.plain.Width > 0 {
		// Notes are lines: each note wraps to the window's text width, as the
		// prompt wraps its message.
		measure := func(s string) int { return body.Advance(encode(s)) }
		var lines []string
		for _, n := range m.plain.Notes {
			lines = append(lines, editorRule.plain(lineBox{Measure: measure, Width: m.plain.Width - 2*popupPad}, n)...)
		}
		m.plain.Notes = lines
	}
	m.paint = encodedSpec(m.plain, encode)
	for i, r := range m.paint.Rows {
		if r.Focus && body != nil {
			m.paint.Rows[i].Caret = body.Advance(r.Value)
		}
	}
	if m.paint.Width <= 0 && !d.Question {
		// A window the description gives no width takes its natural width.
		if len(m.paint.ListColumns) > 0 {
			m.paint.Width = min(fitListWidth(&m.paint, body, listColumnsFit(&m.paint, title, window.X-16)), window.X-16)
		} else {
			m.paint.Width = min(naturalWidth(&m.paint, title, body, m.paint.LabelWidth), window.X-16)
		}
	}
	fitButtons(&m.paint, body)
	if d.Question {
		m.paint.At, m.paint.Bounds = questionPlace(m.paint, title, body, window)
		m.plain.Bounds, m.plain.At = m.paint.Bounds, m.paint.At
		m.layout = layoutPopup(m.paint, title, body)
		return m, nil
	}
	if d.Preview != nil && d.Preview.panelPlacement() && !d.Card {
		m.placePreview(d.Preview, word)
		return m, nil
	}
	if d.Card {
		if person.card == nil {
			person.card = sample.card()
		}
		m.doll, m.figure, m.missing = person.doll, gameFigureSize(person.doll, window), encode(word("editor.unit.look.missing"))
		m.paint.Card, m.paint.CardFont, m.paint.CardAside = person.card, person.cardFont, figureAside(m.figure)
	}
	if p := d.Preview; p != nil {
		if p.Width > 0 {
			m.paint.Width = max(buttonRowWidth(specWindow(m.paint), body), min(p.Width, window.X-16))
		}
		m.paint.Top = max(0, p.Top)
	}
	triggerPlace(&m.paint, title, body, window, m.figure)
	m.plain.Width, m.plain.Bounds, m.plain.At, m.plain.MaxRows = m.paint.Width, m.paint.Bounds, m.paint.At, m.paint.MaxRows
	m.plain.Columns, m.plain.Size, m.plain.MinWidth = m.paint.Columns, m.paint.Size, m.paint.MinWidth
	m.plain.Card, m.plain.CardFont, m.plain.CardAside = m.paint.Card, m.paint.CardFont, m.paint.CardAside
	m.layout = layoutPopup(m.paint, title, body)
	return m, nil
}

// rowPictures gives rows the pictures made names, through picture: a
// picture row its portrait and a row its icons, a blank where none is
// found, so the layout keeps their room. It returns each such row's
// picture names, the portrait's first.
func rowPictures(rows []popupRow, made []popupMade, picture func(string) *image.RGBA) map[int][]string {
	find := func(name string) *image.RGBA {
		if picture != nil {
			if p := picture(name); p != nil {
				return p
			}
		}
		return mockupBlank
	}
	out := map[int][]string{}
	for k, md := range made {
		if md.Picture == "" && len(md.Icons) == 0 {
			continue
		}
		var names []string
		if md.Picture != "" {
			rows[k].Portrait, names = find(md.Picture), []string{md.Picture}
		}
		for _, name := range md.Icons {
			rows[k].Icons = append(rows[k].Icons, find(name))
		}
		out[k] = append(names, md.Icons...)
	}
	return out
}

// Mockup is NewEditorMockup in the editor's own fonts and host text; a
// description with a card shows the first person of the open map that has
// a figure, as the unit window builds the card and the figure.
func (e *MapEditor) Mockup(desc []byte, lang string, window image.Point) (*EditorMockup, error) {
	person := mockupPerson{cardFont: e.cardFont, picture: e.mockupPicture, name: e.mockupName, decode: e.editorSearchText}
	if d, err := parsePopupDesc(desc); err == nil && d.Card && e.doc != nil {
		e.Layout(window.X, window.Y)
		for i, r := range e.doc.Records {
			if r.Kind != KindUnit || !e.Select(i, false) {
				continue
			}
			e.loadObject()
			card, _ := e.unitSettings()
			if p := e.obj.unit.picture; card != nil && card.Char.Band == CharacterBandPerson && p != nil && p.Doll != nil {
				person.card, person.doll = card, p.Doll
				break
			}
		}
	}
	return newEditorMockup(desc, lang, window, e.font, e.bodyFont(), e.editorHostText, person)
}

// messagePicture is the portrait a "message:P:K" sample names: the face the message window shows beside part K of message page P, as that part's heroes see it; nil for any other name or without a map.
func (e *MapEditor) messagePicture(name string) *image.RGBA {
	var page, part int
	if _, err := fmt.Sscanf(name, "message:%d:%d", &page, &part); err != nil || e.doc == nil {
		return nil
	}
	host, ok := e.doc.Edits.(InspectionMessages)
	if !ok {
		return nil
	}
	parts := host.MessageParts(uint32(page))
	if part < 0 || part >= len(parts) {
		return nil
	}
	p := parts[part]
	f := host.MessageFields(uint32(page), p.Source, int(p.Heroes))
	if len(f) != 2 {
		return nil
	}
	return choicePicture(f[1], strconv.FormatInt(p.Speaker, 10))
}

// triggerPlace stands spec s where the editor first opens its trigger window
// in an editor window of size window with these fonts: its bounds, the rows
// it shows with its button row, and the corner the placement store gives it.
// It asks a bare editor with its map catalog closed, so the page and the
// picture place the window as the editor does when no list window stands.
// A card widens the window by its column, field columns given up while the
// window is wider than its bounds, and the window grows to hold a figure of
// size figure under the card.
func triggerPlace(s *popupSpec, title, body *text.Font, window, figure image.Point) {
	e := NewMapEditor(title, nil, nil)
	e.catalog = false
	e.SetBodyFont(body)
	e.Layout(window.X, window.Y)
	s.At, s.Bounds, s.MaxRows = e.keptCorner(popObject), e.popupBounds(), e.objectRowsShown(true)
	s.Top = min(s.Top, max(0, popupRowCount(specWindow(*s))-s.MaxRows))
	if s.Card != nil {
		column := s.Width / max(1, s.Columns)
		s.Width += cardAside(specWindow(*s)) + popupPad
		for s.Columns > 1 && s.Width > s.Bounds.Dx() {
			s.Columns, s.Width = s.Columns-1, s.Width-column
		}
		if l := layoutPopup(*s, title, body); figureHeight(*s, l, figure) != l.Rect.Dy() {
			s.MaxRows = popupRowCount(specWindow(*s))
			naturalSize(s, image.Pt(s.Width, min(figureHeight(*s, l, figure), s.Bounds.Dy())))
			s.MaxRows = layoutPopup(*s, title, body).RowsFit
		}
	}
	e.keepInRoom(popObject, s)
}

// card is the sample person's card: the name and every value the sample
// gives, an override standing for the value in game it replaces.
func (s *popupScope) card() *PanelSubject {
	value := func(values map[string]string, name string) int {
		n, err := strconv.Atoi(values[name])
		if err != nil {
			n, _ = strconv.Atoi(values["game."+name])
		}
		return n
	}
	v := func(name string) int { return value(s.Values, name) }
	c := &PanelSubject{Kind: InspectionUnit, Name: s.Values["name"], HP: v("hp"), MaxHP: v("hpmax"), Mana: v("mana"), MaxMana: v("mana"), Speed: v("speed"),
		Char:   UnitCharacter{Known: true, Band: CharacterBandPerson, Body: v("body"), Mind: v("mind"), Spirit: v("spirit"), Reaction: v("reaction")},
		Combat: UnitCombat{Known: true, Defence: v("defence"), Absorption: v("absorption")}}
	// The sample lists the five skills the card shows, slots 1 to 5.
	for i, skill := range s.Lists["skills"] {
		if i+1 < len(c.Char.Skills) {
			n, err := strconv.Atoi(skill.Values["value"])
			if err != nil {
				n, _ = strconv.Atoi(skill.Values["game"])
			}
			c.Char.Skills[i+1] = n
		}
	}
	return c
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
	s.Title, s.TitleHelp, s.CloseLabel, s.SwitchButton = encode(s.Title), encode(s.TitleHelp), encode(s.CloseLabel), encode(s.SwitchButton)
	s.Tabs, s.Buttons, s.Notes = all(s.Tabs), all(s.Buttons), all(s.Notes)
	columns := make([]listColumn, len(s.ListColumns))
	for i, c := range s.ListColumns {
		columns[i] = listColumn{Head: encode(c.Head), Share: c.Share}
	}
	s.ListColumns = columns
	rows := make([]popupRow, len(s.Rows))
	for i, r := range s.Rows {
		r.Label, r.Value, r.Hint, r.Help = encode(r.Label), encode(r.Value), encode(r.Hint), encode(r.Help)
		r.Buttons = all(r.Buttons)
		rows[i] = r
	}
	s.Rows = rows
	if s.List != nil {
		var list []PickerRow
		for _, r := range s.List.Rows() {
			if len(s.ListColumns) > 0 {
				// Tabs belong to the list structure, not the font alphabet.
				r.Text = strings.Join(all(strings.SplitN(r.Text, "\t", len(s.ListColumns))), "\t")
			} else {
				r.Text = encode(r.Text)
			}
			list = append(list, r)
		}
		chosen := s.List.Selection()
		s.List = NewPicker(list).SetWindow(s.ListRows)
		s.List.Select(chosen)
	}
	if s.Grid != nil {
		g := *s.Grid
		g.RowHeads, g.ColHeads = all(g.RowHeads), all(g.ColHeads)
		g.Cells = make([]popupGridCell, len(s.Grid.Cells))
		for i, c := range s.Grid.Cells {
			c.Label, c.Tip = encode(c.Label), encode(c.Tip)
			g.Cells[i] = c
		}
		s.Grid = &g
	}
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
	if m.chrome != nil {
		m.chrome.draw(dst)
		return dst
	}
	if m.panels != nil {
		m.panels.draw(dst, m.title, m.body)
		return dst
	}
	drawPopup(dst, m.title, m.body, nil, m.paint, m.layout, editorPopupInk, image.Point{}, false)
	if m.paint.Card != nil {
		drawFigure(dst, m.paint, m.layout, m.doll, m.figure, m.body, m.missing)
	}
	drawPopupTip(dst, m.body, m.previewTip().popupTip, editorPopupInk)
	return dst
}

// pagePicture is a picture's box on the page and its name.
type pagePicture struct {
	row  int
	box  image.Rectangle
	name string
}

// pagePictures are the boxes of the pictures layout l's shown rows hold,
// where the painter draws them (rowPictureBoxes), each named by names in
// the same order, the portrait's first; a row names none without names.
func pagePictures(body *text.Font, l popupLayout, rows []popupRow, names map[int][]string) []pagePicture {
	var out []pagePicture
	for _, rr := range l.Rows {
		portrait, icons := rowPictureBoxes(body, rr, rows[rr.Index])
		boxes := icons
		if !portrait.Empty() {
			boxes = append([]image.Rectangle{portrait}, icons...)
		}
		for k, box := range boxes {
			name := ""
			if k < len(names[rr.Index]) {
				name = names[rr.Index][k]
			}
			out = append(out, pagePicture{rr.Index, box, name})
		}
	}
	return out
}

// Page is the window as one HTML page: every part of the layout placed at
// its rectangle in the popup painter's inks, with the description inside.
func (m *EditorMockup) Page() []byte {
	if m.chrome != nil {
		var raster bytes.Buffer
		if m.chrome.body != nil {
			if err := png.Encode(&raster, m.Picture()); err != nil {
				panic(err) // an RGBA image and an in-memory writer
			}
		}
		return mockupPage(m.desc, m.lang, m.window, m.chrome.writePage, raster.Bytes())
	}
	if m.panels != nil {
		return m.panels.page(m.desc, m.lang, m.window)
	}
	figure := image.Rectangle{}
	if m.paint.Card != nil {
		figure = figureRoom(m.paint, m.layout, m.figure)
	}
	pictures := pagePictures(m.body, m.layout, m.paint.Rows, m.pictureNames)
	for k, r := range m.layout.GridCells {
		if n := m.layout.GridFirst + k; n < len(m.gridNames) && m.gridNames[n] != "" {
			pictures = append(pictures, pagePicture{-1, r, m.gridNames[n]})
		}
	}
	if m.desc.Canvas != nil && m.paint.Canvas != nil {
		pictures = append(pictures, pagePicture{-1, m.layout.Canvas, m.desc.Canvas.Picture})
	}
	if m.asideName != "" && !m.layout.Aside.Empty() {
		pictures = append(pictures, pagePicture{-1, m.layout.Aside, m.asideName})
	}
	w := specWindow(m.plain)
	measure := func(s string) int {
		if m.body != nil {
			return m.body.Advance(m.encode(s))
		}
		return len([]rune(s)) * 8
	}
	l := m.layout
	if w.List.Picker != nil {
		box := l.List
		box.HeadingFace = m.title
		w.List.Text = func(row, width int) string {
			item := w.List.Picker.Rows()[row]
			if len(w.List.Columns) == 0 {
				return m.previewFit(box.rowFace(m.body, item), item.Text, width)
			}
			return box.fitCells(textfit.Spec{Measure: measure}, item.Text, width)
		}
	}
	if len(w.List.Columns) > 0 {
		headMeasure := func(s string) int {
			if f := listHeadFont(m.title, m.body); f != nil {
				return f.Advance(m.encode(s))
			}
			return len([]rune(s)) * 8
		}
		l.ListHeadLines = make([][]string, len(l.ListHeads))
		for k, r := range l.ListHeads {
			l.ListHeadLines[k] = editorRule.plain(lineBox{Measure: headMeasure, Width: r.Dx() - listColumnGap/2}, w.List.Columns[k].Head)
		}
	}
	if m.desc.Preview != nil {
		w.Title.Help = m.previewHelp()
		w.Notes.Lines = w.Notes.Lines[len(w.Notes.Lines)-len(l.Notes):]
		w.Body.Rows = slices.Clone(w.Body.Rows)
		for _, r := range l.Rows {
			row := &w.Body.Rows[r.Index]
			switch row.Kind {
			case popupText, popupLink:
				row.Value = m.previewFit(m.body, row.Value, r.Control.Dx()-2)
			case popupChoice:
				row.Value = m.previewFit(m.body, row.Value, choiceTextRoom(r.Control))
			}
		}
	}
	tip := m.previewTip()
	if !tip.Rect.Empty() {
		width := tip.Rect.Dx() - 2*tipPad
		if tip.Rows > 0 {
			width -= widgetSpriteSize
		}
		tip.Lines = editorRule.plain(lineBox{Measure: measure, Width: width}, tip.Full)
	}
	return popupPage(m.desc, w, l, figure, pictures, m.lang, m.window, tip)
}

// pageDescriptionTag opens the description a page carries.
const pageDescriptionTag = `<script type="application/json" id="popup-description">`

// popupPage writes the page from the window's tree w and its layout l, with
// the person's figure room figure when the window has a card and the
// rows' pictures, in the inks the painter reads; it lays nothing out.
func popupPage(d popupDesc, w uiWindow, l popupLayout, figure image.Rectangle, pictures []pagePicture, lang string, window image.Point, tips ...pageTip) []byte {
	return mockupPage(d, lang, window, func(part pagePartWriter) {
		writePopupParts(part, w, l, figure, pictures, d.Preview != nil)
		for _, tip := range tips {
			part.tip(tip.popupTip, nil, tip.Full)
		}
	})
}

type pagePartWriter func(class, name string, row int, r image.Rectangle, ink color.RGBA, txt, tip string)

func (part pagePartWriter) tip(t popupTip, decode func(string) string, full string) {
	if t.Rect.Empty() {
		return
	}
	txt := strings.Join(t.shown(), "\n")
	if decode != nil {
		txt = decode(txt)
	}
	part("window tip", "tip", -1, t.Rect, editorPopupInk.Value, txt, full)
}

func mockupPage(d popupDesc, lang string, window image.Point, write func(pagePartWriter), raster ...[]byte) []byte {
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
	fmt.Fprintf(&b, ".tip{background:%s;color:%s;border:1px solid %s;white-space:pre-wrap;line-height:normal!important;padding:4px}\n", hex(ink.TitleGround), hex(ink.Value), hex(ink.Edge))
	fmt.Fprintf(&b, ".button{text-align:center;color:%s;border:1px solid;border-color:%s %s %s %s}\n",
		hex(ink.Button), hex(ink.BevelLight), hex(ink.BevelDark), hex(ink.BevelDark), hex(ink.BevelLight))
	fmt.Fprintf(&b, ".pressed{border:2px solid %s}\n.ticked{outline:2px solid %s;outline-offset:-4px}\n", hex(ink.Title), hex(ink.Value))
	fmt.Fprintf(&b, ".off{color:%s}\n", hex(ink.Hint))
	fmt.Fprintf(&b, ".tip{background:%s;border:1px solid %s;padding:0 %dpx;white-space:pre}\n", hex(ink.TitleGround), hex(ink.Edge), tipPad)
	if d.Panels != nil {
		fmt.Fprintf(&b, ".map{background:%s;border:1px solid %s}\n", hex(ink.Ground), hex(editorGridInk))
		fmt.Fprintf(&b, ".splitter{background:%s;cursor:ew-resize}\n.splitter.hover{background:%s}\n", hex(ink.Edge), hex(ink.Value))
		fmt.Fprintf(&b, ".grip{cursor:nwse-resize;background:repeating-linear-gradient(135deg,transparent 0 2px,%s 2px 4px);clip-path:polygon(100%% 0,100%% 100%%,0 100%%)}\n", hex(ink.Edge))
		fmt.Fprintf(&b, ".grip.hover{background:repeating-linear-gradient(135deg,transparent 0 2px,%s 2px 4px)}\n", hex(ink.Value))
	}
	if len(raster) > 0 && len(raster[0]) > 0 {
		fmt.Fprintf(&b, "#screen{background-image:url(data:image/png;base64,%s)}\n#screen>.p{opacity:0}\n", base64.StdEncoding.EncodeToString(raster[0]))
	}
	b.WriteString("</style>\n</head>\n<body>\n<div id=\"screen\">\n")
	write(func(class, name string, row int, r image.Rectangle, ink color.RGBA, txt, tip string) {
		if r.Empty() {
			return
		}
		fmt.Fprintf(&b, `<div class="p %s"`, class)
		if tip != "" {
			fmt.Fprintf(&b, ` title="%s"`, html.EscapeString(tip))
		}
		fmt.Fprintf(&b, ` data-part="%s"`, name)
		if row >= 0 {
			fmt.Fprintf(&b, ` data-row="%d"`, row)
		}
		fmt.Fprintf(&b, ` data-box="%d %d %d %d" style="left:%dpx;top:%dpx;width:%dpx;height:%dpx;line-height:%dpx`,
			r.Min.X, r.Min.Y, r.Dx(), r.Dy(), r.Min.X, r.Min.Y, r.Dx(), r.Dy(), r.Dy()/(1+strings.Count(txt, "\n")))
		if class == "swatch" {
			fmt.Fprintf(&b, ";background:%s", hex(ink))
		} else if ink != (color.RGBA{}) {
			fmt.Fprintf(&b, ";color:%s", hex(ink))
		}
		fmt.Fprintf(&b, "\">%s</div>\n", html.EscapeString(txt))
	})
	b.WriteString("</div>\n" + pageDescriptionTag)
	desc, err := json.Marshal(d)
	if err != nil {
		panic(err)
	}
	b.Write(desc)
	b.WriteString("</script>\n</body>\n</html>\n")
	return b.Bytes()
}

func writePopupParts(write pagePartWriter, w uiWindow, l popupLayout, figure image.Rectangle, pictures []pagePicture, preview bool) {
	ink := editorPopupInk
	tip := ""
	part := func(class, name string, row int, r image.Rectangle, ink color.RGBA, txt string) {
		if !r.Empty() {
			write(class, name, row, r, ink, txt, tip)
			tip = ""
		}
	}
	none := color.RGBA{}
	part("window", "window", -1, l.Rect, none, "")
	part("title", "title", -1, l.Title, none, w.Title.Text)
	closeLabel := w.Title.Close
	if closeLabel == "" {
		closeLabel = "x"
	}
	part("button", "close", -1, l.Close, none, closeLabel)
	tip = w.Title.Help
	part("button", "title-help", -1, l.TitleHelp, none, "?")
	tip = ""
	for i, r := range l.Tabs {
		part("button", "tab", i, r, none, w.Tabs.Captions[i])
	}
	for _, rr := range l.Rows {
		row := w.Body.Rows[rr.Index]
		value := row.Ink
		if value == none {
			value = ink.Value
		}
		if row.Kind == popupHeading {
			label := row.Label
			if row.Fold != FoldNone {
				label = rowFoldMarks[row.Fold] + " " + label
			}
			part("heading", "heading", rr.Index, rr.Rect, none, label)
			continue
		}
		part("label", "label", rr.Index, rr.Label, none, row.Label)
		switch shown := row.Value; row.Kind {
		case popupEdit, popupPick, popupMultiline, popupChoice, popupCard:
			if shown == "" && row.Hint != "" {
				shown, value = row.Hint, ink.Hint
			}
			part("control", popupKindNames[row.Kind], rr.Index, rr.Control, value, shown)
		case popupSwitch:
			part("button", "switch", rr.Index, rr.Control, none, row.Value)
		default:
			part("value", "value", rr.Index, rr.Control, value, row.Value)
		}
		captions := rowButtons(w, row)
		for k, r := range rr.Buttons {
			class := "button"
			if k < len(rr.Off) && rr.Off[k] {
				class += " off"
			}
			caption := ""
			if k < len(captions) {
				caption = captions[k]
			}
			if k < len(row.ButtonTips) {
				tip = row.ButtonTips[k]
			}
			part(class, "row-button", rr.Index, r, none, caption)
			tip = ""
		}
		tip = row.Help
		part("button", "row-help", rr.Index, rr.Help, none, "?")
		tip = ""
	}
	for i, r := range l.GridColHeads {
		part("heading", "column-head", i, r, none, w.Grid.ColHeads[i])
	}
	if w.Grid != nil && l.GridCols > 0 {
		// The shown rows' heads, as the layout found them.
		var heads []string
		for k, row := range w.Grid.rows(l.GridCols) {
			if k >= l.GridTop && (row.Head != "" || len(w.Grid.Groups) == 0 && k < len(w.Grid.RowHeads)) {
				heads = append(heads, row.Head)
			}
		}
		for i, r := range l.GridRowHeads {
			if i < len(heads) && heads[i] != "" {
				part("heading", "grid-head", i, r, none, heads[i])
			}
		}
	}
	for i, r := range l.GridCells {
		if w.Grid != nil {
			c := w.Grid.Cells[l.gridIndex(i)]
			tip = c.Tip
			txt := c.Label
			if c.Icon == nil && w.Grid.Cols > 0 {
				// A grid of columns is boxes: the page writes each tick.
				txt = strings.TrimSpace(CheckSpec{On: c.Ticked, Style: CheckText}.Mark() + txt)
			}
			class := "control"
			if c.Pressed {
				class += " pressed"
			}
			if c.Ticked {
				class += " ticked"
			}
			if c.Inert {
				class += " off"
			}
			part(class, "cell", i, r, none, txt)
		}
		tip = ""
	}
	part("control", "grid-scroll", -1, l.GridBar, none, "")
	for i, r := range l.Notes {
		part("note", "note", i, r, w.Notes.Ink, w.Notes.Lines[i])
	}
	if preview {
		part("control", "row-scroll", -1, l.RowBar, none, "")
	}
	if l.HasList && w.List.Picker != nil {
		for k, r := range l.ListHeads {
			part("heading", "list-head", k, r, none, strings.Join(l.ListHeadLines[k], "\n"))
		}
		rows := w.List.Picker.Rows()
		top, shown := w.List.Picker.Visible()
		for i := 0; i < l.List.Rows && i < shown && top+i < len(rows); i++ {
			n := top + i
			class := "control"
			if n == w.List.Picker.Selection() {
				class = "button"
			}
			if len(l.List.Columns) == 0 {
				r, item := l.List.Row(i), rows[n]
				shown := item.Text
				if w.List.Text != nil {
					shown = w.List.Text(n, l.List.rowTextBox(r, item).Dx())
					if shown != item.Text {
						tip = item.Text
					}
				}
				if item.Fold != FoldNone {
					shown = rowFoldMarks[item.Fold] + " " + shown
				}
				part(class, "list-row", n, r, item.Ink, strings.Repeat("  ", max(0, item.Indent))+shown)
				tip = ""
			} else {
				shown := strings.SplitN(rows[n].Text, "\t", len(l.List.Columns))
				if w.List.Text != nil {
					shown = strings.SplitN(w.List.Text(n, l.List.Rect.Dx()-2*listTextX), "\t", len(l.List.Columns))
				}
				for k, full := range strings.SplitN(rows[n].Text, "\t", len(l.List.Columns)) {
					if shown[k] != full {
						tip = full
					}
					part(class, "list-cell", n, l.List.Cell(i, k), none, shown[k])
				}
			}
		}
		if len(rows) > l.List.Rows || len(l.List.Columns) > 0 || preview {
			part("control", "list-scroll", -1, l.List.Bar(), none, "")
		}
	}
	part("button", "switch-button", -1, l.SwitchButton, none, w.Buttons.Switch)
	for i, r := range l.Buttons {
		class := "button"
		if i < len(w.Buttons.Disabled) && w.Buttons.Disabled[i] {
			class += " off"
		}
		part(class, "button", i, r, none, w.Buttons.Captions[i])
	}
	part("control", "card", -1, l.Card, none, "")
	part("control", "figure", -1, figure, none, "")
	for _, p := range pictures {
		part("control", "picture", p.row, p.box, none, p.name)
	}
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

// mockupPicture is the picture of the open map's install a sample names:
// "object:N" object class N's, "structure:N" structure kind N's grid
// picture, "terrain:Name" the ground's swatch, "heightmode:N" the height
// mode's button, "message:P:K" a message part's portrait, "card:N" the test
// party's portrait; nil for any other name
// or without a map.
func (e *MapEditor) mockupPicture(name string) *image.RGBA {
	kind, arg, _ := strings.Cut(name, ":")
	n, err := strconv.Atoi(arg)
	switch {
	case kind == "card":
		return e.cardPicture(name)
	case kind == "message":
		return e.messagePicture(name)
	case kind == "object" && err == nil:
		if h, ok := e.brushHost().(InspectionBrushIcons); ok {
			return h.BrushObjectIcon(n)
		}
	case kind == "structure" && err == nil:
		return e.placeIcon(KindStructure, "kind", int64(n))
	case kind == "spell" && err == nil:
		if offer, ok := e.castOffer("spell"); ok && offer.Pictures != nil {
			value := int64(n)
			return offer.Pictures([]InspectionChoice{{Value: value}})[value]
		}
	case kind == "terrain":
		if h := e.brushHost(); h != nil {
			for _, t := range h.BrushTerrains() {
				if t.Name == arg {
					return t.Swatch
				}
			}
		}
	case kind == "heightmode" && err == nil:
		return heightModeIcon(BrushHeightMode(n))
	}
	return nil
}

// mockupName is the name the open map's install gives what a sample's
// word key "object:N" or "structure:N" names, as the palettes show it.
func (e *MapEditor) mockupName(key string) (string, bool) {
	kind, arg, _ := strings.Cut(key, ":")
	n, err := strconv.Atoi(arg)
	if err != nil {
		return "", false
	}
	switch kind {
	case "object":
		if h := e.brushHost(); h != nil {
			for _, o := range h.BrushObjects() {
				if o.Code == n {
					return e.objectWord(o), true
				}
			}
		}
	case "structure":
		for _, c := range e.placeChoices(KindStructure, "kind") {
			if c.Value == int64(n) {
				return c.Label, true
			}
		}
	}
	return "", false
}
