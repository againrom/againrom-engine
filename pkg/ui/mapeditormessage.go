package ui

import (
	"fmt"
	"image"
	"maps"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/herokind"
)

// InspectionMessages edits message pages and their owner as one history entry.
type InspectionMessages interface {
	// MessageFields are the text and speaker fields of page, or of its tag
	// source[0]; source[1], when given, is the HeroAudience of that tag's
	// cell, whose heroes' faces its hero-dependent speakers show.
	MessageFields(page uint32, source ...int) []InspectionField
	MessageParts(page uint32) []InspectionMessagePart
	SetMessageParts(page uint32, parts []InspectionMessagePart) (*InspectionDocument, error)
	CreateMessageParts(owner InspectionMessageOwner, parts []InspectionMessagePart) (*InspectionDocument, uint32, error)
	// PointMessageParts makes owner name the existing page and writes its
	// parts, as one history entry.
	PointMessageParts(owner InspectionMessageOwner, page uint32, parts []InspectionMessagePart) (*InspectionDocument, error)
	SetMessage(page uint32, body string, speaker int64) (*InspectionDocument, error)
	CreateMessage(owner InspectionMessageOwner, body string, speaker int64) (*InspectionDocument, uint32, error)
	// UnitSpeaker is the speaker of the speaker list that shows placement
	// unit as a dialogue shows it at mission start. When none does, the
	// miss says why; the zero miss says a speaker does.
	UnitSpeaker(unit int) (int64, SpeakerMiss)
}

// SpeakerCause is why no speaker record shows a unit. The zero value says a
// record does.
type SpeakerCause uint8

const (
	SpeakerShown       SpeakerCause = iota
	SpeakerNoMission                // the map's mission start could not be built
	SpeakerBorder                   // the border withdraws the unit at mission start
	SpeakerNoPlacement              // the placement's keys name no definition
	SpeakerDown                     // the unit starts down or dead
	SpeakerNoFit                    // no speaker record's terms fit the person
	SpeakerNotFirst                 // a record fits, but its first unit is another
	SpeakerNoPicture                // no speaker record carries the creature's picture
	SpeakerByHero                   // the start composes the unit from the starting hero
)

// SpeakerMiss is the answer of UnitSpeaker when no record shows the unit.
// For SpeakerNotFirst, Record is the best fitting record and Cell the cell
// of the unit the game shows for it.
type SpeakerMiss struct {
	Cause  SpeakerCause
	Record int64
	Cell   image.Point
	// Unit is that unit's name (InspectionRecord.Name); empty names it by
	// its cell.
	Unit string
}

// WordText gives an editor word by its id.
type WordText func(id string) string

// Text is word id.
func (w WordText) Text(id string) string { return w(id) }

// SpeakerMissText says in whole sentences why no speaker shows a unit: no
// unit at all when miss is nil, else the cause. words gives the editor's
// words in the font's bytes; record names miss's record.
func SpeakerMissText(words WordText, miss *SpeakerMiss, record string) string {
	if miss == nil {
		return words.Text("editor.speaker.why.nounit")
	}
	switch miss.Cause {
	case SpeakerBorder:
		return words.Text("editor.speaker.why.border")
	case SpeakerNoPlacement:
		return words.Text("editor.speaker.why.noplacement")
	case SpeakerDown:
		return words.Text("editor.speaker.why.down")
	case SpeakerNoFit:
		return words.Text("editor.speaker.why.nofit")
	case SpeakerNotFirst:
		unit := miss.Unit
		if unit == "" {
			unit = fmt.Sprintf(words.Text("editor.speaker.why.atcell"), miss.Cell.X, miss.Cell.Y)
		}
		return fmt.Sprintf(words.Text("editor.speaker.why.notfirst"), record, unit)
	case SpeakerNoPicture:
		return words.Text("editor.speaker.why.nopicture")
	case SpeakerByHero:
		return words.Text("editor.speaker.why.byhero")
	}
	return words.Text("editor.speaker.why.nomission")
}

// InspectionMessagePart is one cell of a dialogue's part: the tag Source
// holds and the heroes it reaches. Heroes zero is a part written as one tag
// whatever it reaches. Never marks a tag no hero reaches; Ends a cell no tag
// reaches, where its heroes' dialogue ends. Other is the tag's conditions
// that are not its number, speaker or hero literals.
type InspectionMessagePart struct {
	Source, Number int
	Body, Tag      string
	Speaker        int64
	Heroes         HeroAudience
	Never, Ends    bool
	Other          string
}

// HeroAudience is a set of the four starting heroes a dialogue tells apart,
// one bit each in the cells' fixed order: women before men, mages before
// fighters.
type HeroAudience uint8

var (
	HeroWomanMage    HeroAudience = 1 << herokind.FemaleMage.In(herokind.AudienceOrder)
	HeroWomanFighter HeroAudience = 1 << herokind.FemaleFighter.In(herokind.AudienceOrder)
	HeroManMage      HeroAudience = 1 << herokind.MaleMage.In(herokind.AudienceOrder)
	HeroManFighter   HeroAudience = 1 << herokind.MaleFighter.In(herokind.AudienceOrder)
)

const EveryHero HeroAudience = HeroAudience(herokind.Every)

// HeroCells are the cells of a part told apart by the hero's sex, class,
// both or neither, in order.
func HeroCells(sex, class bool) []HeroAudience {
	var groups []herokind.Kind
	var cells []HeroAudience
	for _, k := range herokind.AudienceOrder {
		at := slices.IndexFunc(groups, func(g herokind.Kind) bool {
			return (!sex || g.Female() == k.Female()) && (!class || g.Mage() == k.Mage())
		})
		if at < 0 {
			at = len(cells)
			groups = append(groups, k)
			cells = append(cells, 0)
		}
		cells[at] |= HeroOf(k.Mage(), k.Female())
	}
	return cells
}

// Kinds reads the cell's bits through the declared audience order.
func (h HeroAudience) Kinds() herokind.Set {
	var set herokind.Set
	for _, k := range herokind.AudienceOrder {
		if h&(1<<k.In(herokind.AudienceOrder)) != 0 {
			set |= herokind.SetOf(k)
		}
	}
	return set
}

// HeroSplit is whether cells tell the heroes apart by sex and by class: a
// cell holding one sex, or one class, of the heroes.
func HeroSplit(cells []HeroAudience) (sex, class bool) {
	for _, h := range cells {
		women, men, mages, fighters := h.Kinds().Axes()
		sex = sex || h != 0 && (!women || !men)
		class = class || h != 0 && (!mages || !fighters)
	}
	return sex, class
}

// messageTags are cells as the tags they hold: each tag once, in the page's
// order, and no cell where a dialogue ends. The editor's commands read and
// name a page's parts by tag.
func messageTags(cells []InspectionMessagePart) []InspectionMessagePart {
	var out []InspectionMessagePart
	seen := map[int]bool{}
	for _, c := range cells {
		if c.Ends || c.Source >= 0 && seen[c.Source] {
			continue
		}
		seen[c.Source] = true
		c.Heroes, c.Never, c.Other = 0, false, ""
		out = append(out, c)
	}
	slices.SortStableFunc(out, func(a, b InspectionMessagePart) int { return a.Source - b.Source })
	return out
}

// heroWords label the cells HeroCells makes, by their heroes.
var heroWords = []struct {
	Heroes HeroAudience
	Word   string
}{
	{Heroes: HeroWomanMage, Word: "editor.message.hero.womanmage"},
	{Heroes: HeroWomanFighter, Word: "editor.message.hero.womanfighter"},
	{Heroes: HeroManMage, Word: "editor.message.hero.manmage"},
	{Heroes: HeroManFighter, Word: "editor.message.hero.manfighter"},
	{Heroes: HeroWomanMage | HeroWomanFighter, Word: "editor.message.hero.woman"},
	{Heroes: HeroManMage | HeroManFighter, Word: "editor.message.hero.man"},
	{Heroes: HeroWomanMage | HeroManMage, Word: "editor.message.hero.mage"},
	{Heroes: HeroWomanFighter | HeroManFighter, Word: "editor.message.hero.fighter"},
}

// HeroOf is the one hero of the class and sex given.
func HeroOf(mage, female bool) HeroAudience {
	return 1 << herokind.Of(mage, female).In(herokind.AudienceOrder)
}

// HeroWord is the word that labels a cell of heroes h, empty for none.
func HeroWord(h HeroAudience) string {
	for _, w := range heroWords {
		if w.Heroes == h {
			return w.Word
		}
	}
	return ""
}

// NewMessagePage is the message popup's page while it writes a new message:
// the page is made only by Apply.
const NewMessagePage uint32 = 1<<32 - 1

// InspectionMessageOwner names the field or wizard action receiving a page.
type InspectionMessageOwner struct {
	Kind    RecordKind
	Index   int
	Key     string
	Trigger *WizardTrigger
	Action  int
}

// The dialog editor: the message popup as one card per page, in order. A
// card holds the page's number on a grip, its speaker and its text; the
// grip drags the card, Alt+Up and Alt+Down move it, its cross strikes it
// until Apply or Cancel, and "+" adds an empty card at the end. A part the
// game tells apart by the hero's sex or class is one card holding a cell
// per hero audience (InspectionMessagePart.Heroes). Opened from a trigger's
// Start dialog, it also chooses which dialog the action starts.

// beginMessage opens the message popup on a message not yet made.
func (e *MapEditor) beginMessage(owner InspectionMessageOwner, back bool) {
	e.messageNew, e.messageFor = owner, nil
	if back {
		e.openMessage(NewMessagePage)
		return
	}
	e.messagePage, e.objectHidden, e.objectTop = NewMessagePage, false, 0
	e.obj.doc = nil
	e.loadObject()
	e.obj.focus = 0
}

func (e *MapEditor) messageField(field int) bool {
	v, ok := e.scriptPopup()
	if !ok || field < 0 || field >= len(e.obj.fields) {
		return false
	}
	if v.trigger != nil && e.obj.wiz != nil {
		r, p := e.wizardRowOf(field)
		if r < 0 || p < 0 {
			return false
		}
		row := e.obj.wiz.rows[r]
		return !row.condition && row.item >= 0 && p < len(wizItems(false)[row.item].params) && wizItems(false)[row.item].params[p].kind == wpMessage
	}
	return v.node != nil && e.obj.fields[field].Key == "value0" && e.nodeParameterKind(v, uint32(e.obj.signature[0]), e.obj.types, 0) == ScriptMessageParameter
}

func (e *MapEditor) messageChoiceAlone(field int, value int64) bool {
	if !e.messageField(field) {
		return false
	}
	found := false
	for _, target := range e.doc.Script.Targets[ScriptMessageParameter] {
		found = found || !target.NewMessage && int64(target.Value) == value
	}
	if !found {
		return false
	}
	old := e.obj.buf[field]
	e.obj.buf[field] = fieldInput(e.obj.fields[field])
	dirty := e.objectDirty()
	e.obj.buf[field] = old
	return !dirty
}

// messageText is the whole text of the page a message field names.
func (e *MapEditor) messageText(field int) (string, bool) {
	if !e.messageField(field) || e.doc.Script == nil {
		return "", false
	}
	v, err := strconv.ParseInt(strings.TrimSpace(e.obj.buf[field]), 10, 64)
	if err != nil {
		return "", false
	}
	for _, t := range e.doc.Script.Targets[ScriptMessageParameter] {
		if !t.NewMessage && int64(t.Value) == v {
			return t.Text, t.Text != ""
		}
	}
	return "", false
}

func (e *MapEditor) openMessage(page uint32) {
	e.messageFor = nil
	e.objectBack = append(e.objectBack, e.objectPlace())
	e.messagePage, e.objectHidden, e.objectTop = page, false, 0
	e.obj.doc = nil
	e.loadObject()
	e.obj.focus = 0
}

func (e *MapEditor) refreshMessagePage() {
	if e.messagePage == 0 {
		return
	}
	if host, ok := e.doc.Edits.(InspectionMessages); ok && len(host.MessageFields(e.messagePage)) != 0 {
		return
	}
	e.messagePage = 0
	e.obj.doc = nil
	if len(e.objectBack) != 0 {
		e.backObject()
	} else {
		e.closeObject()
	}
}

// messageTextField reports that field i is a card's text.
func (e *MapEditor) messageTextField(i int) bool {
	return e.messagePage != 0 && i >= 0 && i < 2*len(e.obj.messageParts) && i%2 == 0
}

// dialogChoiceField is the field of the dialog choice, -1 when the popup
// was not opened from a trigger's action.
func (e *MapEditor) dialogChoiceField() int {
	if f := 2 * len(e.obj.messageParts); e.messagePage != 0 && e.messageFor != nil && f < len(e.obj.fields) {
		return f
	}
	return -1
}

// DialogChoiceField is the field of the dialog editor's dialog choice, -1
// for none.
func (e *MapEditor) DialogChoiceField() int { return e.dialogChoiceField() }

// messageLive is how many page numbers have a card not struck.
func (e *MapEditor) messageLive() int {
	numbers := map[int]bool{}
	for _, part := range e.obj.messageParts {
		if !e.obj.messageGone[part.Number] {
			numbers[part.Number] = true
		}
	}
	return len(numbers)
}

func (e *MapEditor) messageSpec() (popupSpec, []int, bool) {
	e.loadObject()
	o := &e.obj
	if len(o.fields) == 0 {
		return popupSpec{}, nil, false
	}
	title := fmt.Sprintf(e.word("editor.message.page"), e.messagePage)
	if e.messagePage == NewMessagePage {
		title = e.word("editor.message.newdialog")
	}
	s := popupSpec{Title: title, At: e.placedCorner(popObject, homeMessage), Width: e.popupWide(popObject, 660), Bounds: e.popupBounds(), LabelWidth: 64, NoteInk: editorWarning,
		Buttons: []string{e.word("editor.button.apply"), e.word("editor.button.cancel")}}
	o.rowField, o.rowOpen, o.rowActs = map[int]int{}, map[int]int{}, map[int][]string{}
	if f := e.dialogChoiceField(); f >= 0 {
		o.rowField[len(s.Rows)] = f
		s.Rows = append(s.Rows, popupRow{Kind: popupChoice, Label: o.fields[f].Label, Value: choiceText(o.fields[f], o.buf[f]), Focus: o.focus == f})
	}
	live := e.messageLive()
	numbers := map[int]int{}
	textRow := -1
	pickCard := ""
	for i, end := 0, 0; i < len(o.messageParts); i = end {
		part := o.messageParts[i]
		for end = i + 1; end < len(o.messageParts) && o.messageParts[end].Number == part.Number; end++ {
		}
		head := len(s.Rows)
		o.rowOpen[head] = i
		if o.messageGone[part.Number] {
			s.Rows = append(s.Rows, popupRow{Kind: popupCard, Struck: true, Label: e.word("editor.message.removed"), Buttons: []string{e.word("editor.message.restore")}})
			continue
		}
		number := part.Number
		if o.messageChanged {
			if _, ok := numbers[number]; !ok {
				numbers[number] = len(numbers) + 1
			}
			number = numbers[number]
		}
		for f := 2*i + 1; f < 2*end; f += 2 {
			if f == o.pickField {
				pickCard = strconv.Itoa(number)
			}
		}
		textRow = e.messageCard(&s, i, end, number, live)
		s.Rows[head].Span = len(s.Rows) - head - 1
	}
	o.rowOpen[len(s.Rows)] = -1
	s.Rows = append(s.Rows, popupRow{Kind: popupText, Buttons: []string{"+"}})
	s.LabelWidth = e.messageLabelWidth(s)
	if note := e.dialogSharedNote(); note != "" {
		s.Notes = append(s.Notes, note)
	}
	if o.note != "" {
		s.Notes = append(s.Notes, e.editorHostText(o.note))
	}
	// A refused speaker pick answers at the card whose pick is armed.
	if e.speakerPicking() && e.pickWhy != "" && pickCard != "" {
		why := fmt.Sprintf(e.word("editor.speaker.why.card"), pickCard, e.pickWhy)
		s.Notes = append(s.Notes, editorTextLines(e.bodyFont(), why, e.hostedWrap(s, 620))...)
	}
	if e.dragCard > 0 {
		s.Drag = &popupRowDrag{Row: e.dragCard - 1, At: e.pointer}
	}
	e.objectBackButton(&s)
	if font := e.bodyFont(); font != nil {
		for _, caption := range s.Buttons {
			s.ButtonW = max(s.ButtonW, font.Advance(caption)+16)
		}
	}
	if len(o.messageParts) == 1 && textRow >= 0 {
		e.fitMessageText(&s, textRow)
	} else {
		s.MaxRows = max(3, e.objectRowsShown(true)/2)
		e.objectTop = max(0, min(e.objectTop, len(s.Rows)-s.MaxRows))
		s.Top = e.objectTop
	}
	return s, make([]int, len(s.Rows)), true
}

// The dialog editor's own row actions: a card's split by the hero's sex or
// class, and the cross of a tag no hero reaches.
const (
	actSplitSex    = "splitsex"
	actSplitClass  = "splitclass"
	actDropVariant = "dropvariant"
)

// messageCard appends the rows of the card of cells i to end, one part
// number, and returns its last text row. A part of one variant keeps its
// speaker on the card's head and its text under it. A part told apart by the
// hero is one frame of stacked rows: its number alone on the head, then per
// cell its audience label with its speaker, and its text. Each cell shows
// the conditions its tag holds besides, or that its heroes' dialogue ends;
// the tags no hero reaches close the card, each with a cross.
func (e *MapEditor) messageCard(s *popupSpec, i, end, number, live int) int {
	o := &e.obj
	var heroes []HeroAudience
	first := -1
	for c := i; c < end; c++ {
		if !o.messageParts[c].Never {
			heroes = append(heroes, o.messageParts[c].Heroes)
			if first < 0 {
				first = c
			}
		}
	}
	sex, class := HeroSplit(heroes)
	branched := sex || class
	head := popupRow{Kind: popupCard, Label: strconv.Itoa(number)}
	var acts []string
	if live > 1 {
		head.Buttons, acts = []string{"x"}, []string{actRemoveRow}
	}
	if !branched && first >= 0 {
		speaker := 2*first + 1
		head.Value, head.Focus = choiceText(o.fields[speaker], o.buf[speaker]), o.focus == speaker
		head.Buttons, acts = append(head.Buttons, e.actWord(actPickParam)), append(acts, actPickParam)
		o.rowField[len(s.Rows)] = speaker
	}
	head.Buttons, acts = append(head.Buttons, e.word("editor.message.bysex"), e.word("editor.message.byclass")), append(acts, actSplitSex, actSplitClass)
	head.Armed = make([]bool, len(head.Buttons))
	head.Armed[len(head.Buttons)-2], head.Armed[len(head.Buttons)-1] = sex, class
	o.rowActs[len(s.Rows)] = acts
	s.Rows = append(s.Rows, head)
	textRow := -1
	for c := i; c < end; c++ {
		part := o.messageParts[c]
		if part.Never {
			continue
		}
		speaker := 2*c + 1
		if branched {
			o.rowField[len(s.Rows)], o.rowActs[len(s.Rows)] = speaker, []string{actPickParam}
			s.Rows = append(s.Rows, popupRow{Kind: popupChoice, Label: e.word(HeroWord(part.Heroes)), Value: choiceText(o.fields[speaker], o.buf[speaker]), Focus: o.focus == speaker, Buttons: []string{e.actWord(actPickParam)}})
		}
		if part.Other != "" {
			s.Rows = append(s.Rows, popupRow{Kind: popupText, Label: e.word("editor.message.other"), Value: e.editorHostText(part.Other)})
		}
		if part.Ends {
			s.Rows = append(s.Rows, popupRow{Kind: popupText, Label: e.word("editor.message.ends")})
		}
		lines := strings.Split(o.buf[2*c], "\n")
		for k := range lines {
			lines[k] = e.editorHostText(strings.TrimSuffix(lines[k], "\r"))
		}
		textRow = len(s.Rows)
		o.rowField[textRow] = 2 * c
		s.Rows = append(s.Rows, popupRow{Kind: popupMultiline, Value: strings.Join(lines, "\n"), Focus: o.focus == 2*c, Lines: 3, Portrait: choicePicture(o.fields[speaker], o.buf[speaker]), PortraitTip: choiceNote(o.fields[speaker], o.buf[speaker])})
	}
	for c := i; c < end; c++ {
		if part := o.messageParts[c]; part.Never {
			line, _, _ := strings.Cut(part.Body, "\n")
			o.rowOpen[len(s.Rows)], o.rowActs[len(s.Rows)] = c, []string{actDropVariant}
			s.Rows = append(s.Rows, popupRow{Kind: popupText, Label: fmt.Sprintf(e.word("editor.message.never"), e.editorHostText(strings.TrimSuffix(line, "\r"))), Buttons: []string{"x"}})
		}
	}
	return textRow
}

// messageLabelWidth is the label column of the dialog editor: wide enough
// for the longest label beside a control, from the 64 pixels of a card's
// portrait, up to two fifths of the popup.
func (e *MapEditor) messageLabelWidth(s popupSpec) int {
	w := 64
	if body := e.bodyFont(); body != nil {
		for _, row := range s.Rows {
			if rowHasControl(row) && row.Kind != popupMultiline {
				w = max(w, body.Advance(row.Label)+8)
			}
		}
	}
	return min(w, s.Width*2/5)
}

// splitMessageCard tells the heroes of the card holding cell at apart by
// sex, or by class, or merges them back where the card already does. Each
// new cell takes the text and speaker of the first cell that held one of
// its heroes, a cell with a tag before one where the dialogue ends; the
// card's never-shown tags stay after its cells.
func (e *MapEditor) splitMessageCard(at int, bySex bool) {
	o := &e.obj
	if at < 0 || at >= len(o.messageParts) || !e.captureMessageParts() {
		return
	}
	start, end := at, at+1
	for start > 0 && o.messageParts[start-1].Number == o.messageParts[at].Number {
		start--
	}
	for end < len(o.messageParts) && o.messageParts[end].Number == o.messageParts[at].Number {
		end++
	}
	group := o.messageParts[start:end]
	var heroes []HeroAudience
	for _, c := range group {
		if !c.Never {
			heroes = append(heroes, c.Heroes)
		}
	}
	sex, class := HeroSplit(heroes)
	if bySex {
		sex = !sex
	} else {
		class = !class
	}
	var cells, never []InspectionMessagePart
	for _, h := range HeroCells(sex, class) {
		pick := -1
		for k, c := range group {
			held := c.Heroes
			if held == 0 {
				held = EveryHero
			}
			if !c.Never && held&h != 0 && (pick < 0 || group[pick].Ends && !c.Ends) {
				pick = k
			}
		}
		if pick >= 0 {
			cell := group[pick]
			cell.Heroes = h
			cells = append(cells, cell)
		}
	}
	for _, c := range group {
		if c.Never {
			never = append(never, c)
		}
	}
	e.reshapeMessageCards(slices.Concat(o.messageParts[:start], cells, never, o.messageParts[end:]))
}

// dropMessageVariant takes the never-shown tag of cell at off its card, until
// Apply or Cancel.
func (e *MapEditor) dropMessageVariant(at int) {
	o := &e.obj
	if at < 0 || at >= len(o.messageParts) || !o.messageParts[at].Never || !e.captureMessageParts() {
		return
	}
	e.reshapeMessageCards(slices.Delete(slices.Clone(o.messageParts), at, at+1))
}

// reshapeMessageCards puts parts in the cards' place after a split, a merge
// or a dropped tag. A speaker pick armed on a card ends: its field moved.
func (e *MapEditor) reshapeMessageCards(parts []InspectionMessagePart) {
	o := &e.obj
	if e.pick == pickParamUnit && o.pickField >= 0 && o.pickField < 2*len(o.messageParts) {
		e.leave(leavePick)
		o.pickField = -1
	}
	gone := o.messageGone
	e.loadMessageFields(parts)
	o.messageGone, o.messageChanged, o.open, o.focus = gone, true, -1, -1
}

// fitMessageText grows the text box of row k until it shows the whole
// message, as far as the window allows; a longer text scrolls by the wheel,
// from its first line.
func (e *MapEditor) fitMessageText(s *popupSpec, k int) {
	body := e.bodyFont()
	if body == nil {
		return
	}
	l := layoutPopup(*s, e.font, body)
	if len(l.Rows) <= k {
		return
	}
	width := l.Rows[k].Control.Dx() - 8
	count := 0
	for _, line := range strings.Split(s.Rows[k].Value, "\n") {
		count += len(editorTextLines(body, line, width))
	}
	pitch, line := l.Pitch, fontHeight(body, 15)
	spare := (s.Bounds.Dy() - l.Rect.Dy()) / pitch
	rows := min(max(multilineRows(s.Rows[k]), (count*line+8+pitch-1)/pitch), multilineRows(s.Rows[k])+max(0, spare))
	s.Rows[k].Lines = rows
	visible := max(1, (rows*pitch-2-8)/line)
	e.objectTop = max(0, min(e.objectTop, count-visible))
	s.Rows[k].First = e.objectTop
}

// dialogStarters are the triggers, by index, with an action that starts
// page.
func (e *MapEditor) dialogStarters(page uint32) []int {
	var out []int
	if e.doc == nil || e.doc.Script == nil {
		return nil
	}
	for i, t := range e.doc.Script.Triggers {
		for _, id := range t.Acts {
			if n, ok := e.nodeByID(false, id); id != 0 && ok && n.Values[0] == page && e.nodeParameterKind(scriptView{}, n.Opcode, n.Types, 0) == ScriptMessageParameter {
				out = append(out, i)
				break
			}
		}
	}
	return out
}

// dialogSharedNote names the other triggers that start the shown dialog.
func (e *MapEditor) dialogSharedNote() string {
	if e.messagePage == NewMessagePage {
		return ""
	}
	var others []string
	for _, t := range e.dialogStarters(e.messagePage) {
		if e.messageFor == nil || t != e.messageFor.Index {
			others = append(others, strconv.Itoa(t+1))
		}
	}
	if len(others) == 0 || e.messageFor == nil && len(others) < 2 {
		return ""
	}
	return fmt.Sprintf(e.word("editor.message.shared"), strings.Join(others, ", "))
}

// messageDraftDirty reports a card changed, added, moved or struck.
func (e *MapEditor) messageDraftDirty() bool {
	o := &e.obj
	if o.messageChanged {
		return true
	}
	for i := 0; i < 2*len(o.messageParts) && i < len(o.fields); i++ {
		if o.buf[i] != fieldInput(o.fields[i]) {
			return true
		}
	}
	return false
}

// chooseDialog shows the dialog the choice names, or a new one; the action
// is pointed at it by Apply. A draft with changes stays until it is applied
// or cancelled.
func (e *MapEditor) chooseDialog(value int64) {
	o := &e.obj
	o.open = -1
	page := uint32(value)
	if value == NewMessageChoice {
		page = NewMessagePage
	}
	if page == e.messagePage || value <= 0 || value > NewMessageChoice {
		return
	}
	if e.messageDraftDirty() {
		o.note = e.word("editor.message.switchfirst")
		return
	}
	if page == NewMessagePage {
		owner, err := e.dialogOwner()
		if err != nil {
			o.note = err.Error()
			return
		}
		e.messageNew = owner
	}
	e.messagePage, e.objectTop = page, 0
	o.doc = nil
	e.loadObject()
	o.focus = 0
}

// dialogOwner is the action the dialog editor was opened for, with its
// trigger as it stands.
func (e *MapEditor) dialogOwner() (InspectionMessageOwner, error) {
	owner := *e.messageFor
	if owner.Index < 0 || owner.Index >= len(e.doc.Script.Triggers) {
		return owner, fmt.Errorf("%s", e.word("editor.message.unavailable"))
	}
	plan, err := e.keptPlan(owner.Index)
	if err != nil {
		return owner, err
	}
	owner.Trigger = &plan
	return owner, nil
}

// liveMessageParts are the parts Apply writes: every part not struck.
func (e *MapEditor) liveMessageParts() []InspectionMessagePart {
	var out []InspectionMessagePart
	for _, part := range e.obj.messageParts {
		if !e.obj.messageGone[part.Number] {
			out = append(out, part)
		}
	}
	return out
}

func (e *MapEditor) applyMessage() {
	if _, ok := e.doc.Edits.(InspectionMessages); !ok || !e.captureMessageParts() {
		return
	}
	parts := e.liveMessageParts()
	if e.messagePage == NewMessagePage {
		e.createMessage(parts)
		return
	}
	if e.messageFor != nil && e.messagePage != e.messageFrom {
		owner, err := e.dialogOwner()
		if err == nil {
			_, err = e.commandWriteMessage(messageWrite{Owner: &owner, Page: e.messagePage, Parts: parts})
		}
		if err != nil {
			e.obj.note = err.Error()
			return
		}
		e.messageFrom = e.messagePage
		e.obj.doc = nil
		e.loadObject()
		return
	}
	if !e.objectDirty() {
		return
	}
	if _, err := e.commandWriteMessage(messageWrite{Page: e.messagePage, Parts: parts}); err != nil {
		e.obj.note = err.Error()
		return
	}
	e.obj.doc = nil
	e.loadObject()
}

// createMessage writes the draft and its owner; Back returns to the owner.
func (e *MapEditor) createMessage(parts []InspectionMessagePart) {
	owner := e.messageNew
	if e.messageFor != nil {
		var err error
		if owner, err = e.dialogOwner(); err != nil {
			e.obj.note = err.Error()
			return
		}
	}
	ch, err := e.commandWriteMessage(messageWrite{Owner: &owner, Page: NewMessagePage, Parts: parts})
	if err != nil {
		e.obj.note = err.Error()
		return
	}
	if ch.doc == nil {
		return
	}
	page := *ch.page
	back := e.objectBack
	if owner.Kind == KindMessage {
		e.selectNewest(KindAction)
		e.obj.doc = nil
		e.loadObject()
		back = append(e.objectBack, e.objectPlace())
	}
	e.objectBack, e.messagePage, e.objectHidden = back, page, false
	e.messageNew, e.messageFrom = InspectionMessageOwner{}, page
	e.obj.doc = nil
	e.loadObject()
}

func (e *MapEditor) loadMessageFields(parts []InspectionMessagePart) {
	host, ok := e.doc.Edits.(InspectionMessages)
	if !ok {
		return
	}
	o := &e.obj
	o.messageParts, o.fields, o.buf = slices.Clone(parts), nil, nil
	for _, part := range parts {
		fields := host.MessageFields(e.messagePage, part.Source, int(part.Heroes))
		if len(fields) != 2 {
			continue
		}
		fields[0].TextValue, fields[1].Value = part.Body, part.Speaker
		o.fields = append(o.fields, fields...)
		for _, field := range fields {
			o.buf = append(o.buf, fieldInput(field))
		}
	}
	if e.messageFor != nil && len(o.fields) == 2*len(parts) {
		value := int64(e.messagePage)
		if e.messagePage == NewMessagePage {
			value = NewMessageChoice
		}
		f := InspectionField{Key: "dialog", Label: e.word("editor.message.dialog"), Value: value, Choices: e.parameterChoices(ScriptMessageParameter, value)}
		for k := range f.Choices {
			if f.Choices[k].Value == NewMessageChoice {
				f.Choices[k].Label = e.word("editor.message.newdialog")
			}
		}
		o.fields, o.buf = append(o.fields, f), append(o.buf, fieldInput(f))
	}
	e.input = editorInputState{}
}

func (e *MapEditor) captureMessageParts() bool {
	o := &e.obj
	if len(o.buf) < len(o.messageParts)*2 {
		return false
	}
	for i := range o.messageParts {
		speaker, err := strconv.ParseInt(o.buf[2*i+1], 10, 32)
		if err != nil {
			o.note = e.word("editor.message.badspeaker")
			return false
		}
		o.messageParts[i].Body, o.messageParts[i].Speaker = o.buf[2*i], speaker
	}
	return true
}

// messageRowButton is a press on a card's cross, its "on the map" button,
// its split by sex or class or Restore, on a never-shown tag's cross, or on
// "+".
func (e *MapEditor) messageRowButton(row, button int) {
	o := &e.obj
	if acts := o.rowActs[row]; button >= 0 && button < len(acts) {
		at, open := o.rowOpen[row]
		switch acts[button] {
		case actPickParam:
			if f, ok := o.rowField[row]; ok {
				e.armParamPick(f, false)
			}
			return
		case actSplitSex, actSplitClass:
			if open {
				e.splitMessageCard(at, acts[button] == actSplitSex)
			}
			return
		case actDropVariant:
			if open {
				e.dropMessageVariant(at)
			}
			return
		}
	}
	at, ok := o.rowOpen[row]
	if !ok || button != 0 || !e.captureMessageParts() {
		return
	}
	if at < 0 {
		e.addMessageCard()
		return
	}
	number := o.messageParts[at].Number
	// A new map each time: a Cancel checkpoint keeps the one it saw.
	gone := maps.Clone(o.messageGone)
	if gone[number] {
		delete(gone, number)
		o.messageGone = gone
		return
	}
	if e.messageLive() < 2 {
		return
	}
	if gone == nil {
		gone = map[int]bool{}
	}
	gone[number], o.messageGone, o.messageChanged = true, gone, true
	if f := o.focus; f >= 0 && f < 2*len(o.messageParts) && o.messageParts[f/2].Number == number {
		o.focus = -1
	}
	o.open = -1
}

// addMessageCard adds an empty card at the end, the caret in its text.
func (e *MapEditor) addMessageCard() {
	parts := slices.Clone(e.obj.messageParts)
	id := 0
	for _, part := range parts {
		id = min(id, part.Number)
	}
	parts = append(parts, InspectionMessagePart{Source: -1, Number: id - 1, Speaker: -1})
	gone := e.obj.messageGone
	e.loadMessageFields(parts)
	e.obj.messageGone, e.obj.messageChanged, e.obj.focus, e.obj.open = gone, true, 2*(len(parts)-1), -1
	e.objectTop = len(parts) * 2
}

// moveMessageCard moves card at, with every card of its page number, to
// before card to; to is the card count for after the last.
func (e *MapEditor) moveMessageCard(at, to int) bool {
	o := &e.obj
	if at < 0 || at >= len(o.messageParts) || !e.captureMessageParts() {
		return false
	}
	number := o.messageParts[at].Number
	var group, rest []InspectionMessagePart
	before, lead := 0, 0
	for i, part := range o.messageParts {
		if part.Number == number {
			if i < at {
				lead++
			}
			group = append(group, part)
			continue
		}
		rest = append(rest, part)
		if i < to {
			before++
		}
	}
	moved := slices.Insert(rest, before, group...)
	if slices.Equal(moved, o.messageParts) {
		return false
	}
	gone := o.messageGone
	focus, pick := o.focus, o.pickField
	// The card the pick is armed on stands where moved puts it: the moved
	// group first, every other card in its order around it.
	if pick >= 0 && pick < 2*len(o.messageParts) {
		old, rank := pick/2, 0
		for i := 0; i < old; i++ {
			if (o.messageParts[i].Number == number) == (o.messageParts[old].Number == number) {
				rank++
			}
		}
		switch {
		case o.messageParts[old].Number == number:
			rank += before
		case rank >= before:
			rank += len(group)
		}
		pick = 2*rank + pick%2
	}
	e.loadMessageFields(moved)
	o.messageGone, o.messageChanged, o.open, o.focus = gone, true, -1, -1
	if focus >= 0 && focus < 2*len(moved) && focus/2 == at {
		o.focus = 2*(before+lead) + focus%2
	}
	// An armed pick follows its card, as the focus does.
	if pick >= 0 && pick < 2*len(moved) {
		o.pickField = pick
	}
	return true
}

// keyCard moves the focused card one place up or down, past the whole of
// the neighbouring page number.
func (e *MapEditor) keyCard(key string) error {
	o := &e.obj
	f := o.focus
	if e.messagePage == 0 || f < 0 || f >= 2*len(o.messageParts) {
		return nil
	}
	at := f / 2
	parts := o.messageParts
	number := parts[at].Number
	if key == "cardup" {
		for i := at - 1; i >= 0; i-- {
			if parts[i].Number != number {
				e.moveMessageCard(at, slices.IndexFunc(parts, func(p InspectionMessagePart) bool { return p.Number == parts[i].Number }))
				return nil
			}
		}
		return nil
	}
	for i := at + 1; i < len(parts); i++ {
		if parts[i].Number != number {
			next := parts[i].Number
			last := i
			for k := i; k < len(parts); k++ {
				if parts[k].Number == next {
					last = k
				}
			}
			e.moveMessageCard(at, last+1)
			return nil
		}
	}
	return nil
}

// dropCard ends a card drag: the held card lands where the marker stood.
func (e *MapEditor) dropCard() {
	s, _, ok := e.objectSpec()
	row := e.dragCard - 1
	e.dragCard = 0
	if !ok || e.messagePage == 0 {
		return
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	at, ok := e.obj.rowOpen[row]
	if !ok || at < 0 || l.Drop < 0 {
		return
	}
	// The drop counts cards; a card holds every cell of its part number.
	to, card := len(e.obj.messageParts), 0
	for i, r := range s.Rows {
		if r.Kind != popupCard {
			continue
		}
		if card == l.Drop {
			to = e.obj.rowOpen[i]
			break
		}
		card++
	}
	e.moveMessageCard(at, to)
}

// CardRects are the window rectangles of card c of the dialog editor, when
// it is shown: the whole card and its grip, empty for a struck card.
func (e *MapEditor) CardRects(c int) (card, grip image.Rectangle, ok bool) {
	s, _, shown := e.objectSpec()
	if !shown || e.messagePage == 0 {
		return card, grip, false
	}
	for _, rr := range layoutPopup(s, e.font, e.bodyFont()).Rows {
		if at, has := e.obj.rowOpen[rr.Index]; has && at == c && s.Rows[rr.Index].Kind == popupCard {
			return rr.Card, rr.Grip, true
		}
	}
	return card, grip, false
}

// newMessage opens the message popup for a new message the field will
// name; nothing is made until its Apply.
func (e *MapEditor) newMessage(field int) {
	if _, ok := e.doc.Edits.(InspectionMessages); !ok || field < 0 || field >= len(e.obj.fields) {
		return
	}
	if e.obj.buf[field] == strconv.FormatInt(NewMessageChoice, 10) {
		e.obj.buf[field] = fieldInput(e.obj.fields[field])
	}
	if !e.objectMayLeave() {
		return
	}
	kind, index := e.selectedRecord()
	owner := InspectionMessageOwner{Kind: kind, Index: index, Key: e.obj.fields[field].Key}
	if v, ok := e.scriptPopup(); ok && v.trigger != nil && e.obj.wiz != nil {
		plan, err := e.wizardPlan(v)
		if err != nil {
			e.obj.note = err.Error()
			return
		}
		row, _ := e.wizardRowOf(field)
		for r := 0; r < row; r++ {
			if !e.obj.wiz.rows[r].condition {
				owner.Action++
			}
		}
		owner.Trigger = &plan
	}
	e.beginMessage(owner, true)
}

// configureDialog opens the dialog editor for the Start dialog action whose
// message field is field: on the dialog it starts, or on a new one when it
// starts none. A trigger with changes not yet applied is applied first.
func (e *MapEditor) configureDialog(field int) {
	v, ok := e.scriptPopup()
	if !ok || v.trigger == nil || e.obj.wiz == nil || field < 0 || field >= len(e.obj.fields) {
		return
	}
	key := e.obj.fields[field].Key
	if e.objectDirty() {
		e.applyWizard(v)
		if e.obj.note != "" {
			return
		}
		e.loadObject()
		if field = slices.IndexFunc(e.obj.fields, func(f InspectionField) bool { return f.Key == key }); field < 0 {
			return
		}
	}
	row, _ := e.wizardRowOf(field)
	action := 0
	for r := 0; r < row; r++ {
		if !e.obj.wiz.rows[r].condition {
			action++
		}
	}
	owner := &InspectionMessageOwner{Kind: KindTrigger, Index: v.index, Action: action}
	if target, ok := e.parameterTarget(field); ok {
		e.openMessage(target.Value)
		e.messageFor, e.messageFrom = owner, target.Value
	} else {
		e.newMessage(field)
		if e.messagePage != NewMessagePage {
			return
		}
		e.messageFor, e.messageFrom = owner, NewMessagePage
	}
	e.obj.doc = nil
	e.loadObject()
	e.obj.focus = 0
}
