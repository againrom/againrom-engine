package ui

import (
	"fmt"
	"image"
	"strings"
)

// The Map panel lists what a map needs to be played, one line per need, each
// stating the map's value and opening the place that sets it. It stands in the
// list popup when a map opens and under any list shown there; the Map menu
// brings it back. Its two checks are rows of the host's rule table, so the
// report, Save, Test play and the mod load read the same findings.

// The rule IDs of the panel's two checks.
const (
	RuleDropCell = "drop-cell" // the start point: none, off the map or blocked
	RuleVictory  = "victory"   // no trigger reaches the action that wins the map
)

// The type-0 +0x70 words a new map is given: single-player is the value every
// campaign map carries, multiplayer the original editor's new-map value.
const (
	MapModeSingle      uint32 = 1
	MapModeMultiplayer uint32 = 4
)

// NewMapMultiplayer ends a new-map source that asks for a multiplayer map.
const NewMapMultiplayer = "/multiplayer"

// InspectionMapFacts is what the Map panel states about a map. Strings are
// UTF-8.
type InspectionMapFacts struct {
	Name, Description string
	Mode              uint32 // type-0 +0x70
	Players, Sides    int
	Drops             []image.Point // the cells the drop actions name
	Wins, Losses      []string      // the names of the triggers that win or lose the map
	Protects          int           // the checks that lose the map when their unit dies
}

// InspectionMapPanel is implemented by edits whose host states the facts of
// the Map panel.
type InspectionMapPanel interface {
	MapFacts() InspectionMapFacts
}

func (e *MapEditor) mapPanel() (InspectionMapPanel, bool) {
	if !e.editable() {
		return nil, false
	}
	p, ok := e.doc.Edits.(InspectionMapPanel)
	return p, ok
}

// panelLink is what a panel line opens.
type panelLink uint8

const (
	linkNone panelLink = iota
	linkName
	linkDescription
	linkPlayers
	linkMode
	linkStart
	linkGoals
	linkReport
	linkTestPlay
)

// mapLine is one line of the panel: its label, its value and its link. Warn
// marks a line a check finds wanting.
type mapLine struct {
	label, value string
	link         panelLink
	warn         bool
}

// panelFindings are the report's findings, or none when the host has no
// report.
func (e *MapEditor) panelFindings() []InspectionFinding {
	if c, ok := e.checker(); ok {
		return c.Report()
	}
	return nil
}

// modeWord is the Mode value in player words: the two values a new map is
// given by name, any other value as itself with the mode it gives.
func (e *MapEditor) modeWord(v uint32) string {
	switch {
	case v == MapModeSingle:
		return e.word("editor.mode.single")
	case v == MapModeMultiplayer:
		return e.word("editor.mode.multi")
	case int32(v) > 1:
		return fmt.Sprintf(e.word("editor.mode.othermulti"), int32(v))
	}
	return fmt.Sprintf(e.word("editor.mode.othersingle"), int32(v))
}

// panelLines are the panel's lines for the facts and findings.
func (e *MapEditor) panelLines(f InspectionMapFacts, findings []InspectionFinding) []mapLine {
	ruled := map[string][]InspectionFinding{}
	for _, x := range findings {
		ruled[x.Rule] = append(ruled[x.Rule], x)
	}
	text := func(s string) string {
		if strings.TrimSpace(s) == "" {
			return e.word("editor.panel.none")
		}
		return e.editorHostText(s)
	}
	lines := []mapLine{
		{label: e.word("editor.panel.size"), value: fmt.Sprintf("%d x %d", e.doc.Width, e.doc.Height)},
		{label: e.word("editor.panel.name"), value: text(f.Name), link: linkName},
		{label: e.word("editor.panel.description"), value: text(f.Description), link: linkDescription},
		{label: e.word("editor.panel.players"), value: fmt.Sprintf(e.word("editor.panel.playersvalue"), f.Players, f.Sides), link: linkPlayers},
		{label: e.word("editor.panel.mode"), value: e.modeWord(f.Mode), link: linkMode},
		e.startLine(f.Drops, ruled[RuleDropCell]),
		e.victoryLine(f.Wins, ruled[RuleVictory]),
		e.defeatLine(f),
	}
	errs, warnings := CountFindings(findings)
	lines = append(lines,
		mapLine{label: e.word("editor.panel.report"), value: fmt.Sprintf(e.word("editor.panel.reportvalue"), errs, warnings), link: linkReport, warn: errs > 0},
		mapLine{label: e.word("editor.panel.testplay"), value: e.word("editor.panel.testplayvalue"), link: linkTestPlay})
	return lines
}

// startLine is the start point: the drop cell, or the random cell the party
// lands on without one; a drop-cell finding marks it.
func (e *MapEditor) startLine(drops []image.Point, found []InspectionFinding) mapLine {
	l := mapLine{label: e.word("editor.panel.start"), link: linkStart, warn: len(found) > 0}
	switch len(drops) {
	case 0:
		l.value = e.word("editor.panel.start.none")
		return l
	case 1:
		l.value = fmt.Sprintf(e.word("editor.panel.start.cell"), drops[0].X, drops[0].Y)
	default:
		l.value = fmt.Sprintf(e.word("editor.panel.start.cells"), len(drops))
	}
	for _, x := range found {
		if x.HasCell {
			l.value = fmt.Sprintf(e.word("editor.panel.start.bad"), x.Cell.X, x.Cell.Y)
			return l
		}
	}
	if len(found) > 0 {
		l.value += "; " + e.word("editor.panel.start.random")
	}
	return l
}

// victoryLine is the triggers that win the map, or the endless map the
// victory check finds.
func (e *MapEditor) victoryLine(wins []string, found []InspectionFinding) mapLine {
	l := mapLine{label: e.word("editor.panel.victory"), link: linkGoals, warn: len(found) > 0}
	switch {
	case len(found) > 0 || len(wins) == 0:
		l.value = e.word("editor.panel.victory.none")
	case len(wins) == 1:
		l.value = fmt.Sprintf(e.word("editor.panel.victory.one"), e.editorHostText(wins[0]))
	default:
		l.value = fmt.Sprintf(e.word("editor.panel.victory.many"), len(wins))
	}
	return l
}

// defeatLine is the rules that lose the map; the hero's fall loses it with or
// without them.
func (e *MapEditor) defeatLine(f InspectionMapFacts) mapLine {
	l := mapLine{label: e.word("editor.panel.defeat"), link: linkGoals}
	if n := len(f.Losses) + f.Protects; n > 0 {
		l.value = fmt.Sprintf(e.word("editor.panel.defeat.some"), n)
	} else {
		l.value = e.word("editor.panel.defeat.none")
	}
	return l
}

// panelWidth and panelLabelWidth are the panel's width and its label column.
const panelWidth, panelLabelWidth = 440, 120

// panelSpec is the Map panel as a popup description, and the link of each
// row; a wrapped value continues on rows that open the same place.
func (e *MapEditor) panelSpec() (popupSpec, []panelLink) {
	if !e.listPlaced {
		e.listAt, e.listPlaced = image.Pt(8, e.barHeight()+8), true
	}
	s := popupSpec{Title: e.word("editor.panel.title"), At: e.listAt, Width: panelWidth, Bounds: e.popupBounds(), NoteInk: editorPaper, LabelWidth: panelLabelWidth}
	p, ok := e.mapPanel()
	if !ok {
		return s, nil
	}
	var links []panelLink
	width := panelWidth - 2*popupPad - panelLabelWidth - 8
	for _, l := range e.panelLines(p.MapFacts(), e.panelFindings()) {
		kind := popupText
		if l.link != linkNone {
			kind = popupLink
		}
		ink := editorLight
		if l.warn {
			ink = editorWarning
		}
		for n, part := range editorWrap(e.bodyFont(), l.value, width) {
			label := ""
			if n == 0 {
				label = l.label
			}
			s.Rows, links = append(s.Rows, popupRow{Kind: kind, Label: label, Value: part, Ink: ink}), append(links, l.link)
		}
	}
	s.Notes = editorWrap(e.bodyFont(), e.word("editor.panel.hint"), panelWidth-2*popupPad-4)
	return s, links
}

// PanelOpen reports whether the Map panel is what the list popup shows.
func (e *MapEditor) PanelOpen() bool { return e.listKind() == "panel" }

// PanelLines are the shown panel's lines, label and value joined by a space;
// a wrapped value is joined back into its line.
func (e *MapEditor) PanelLines() []string {
	if !e.PanelOpen() {
		return nil
	}
	s, _ := e.panelSpec()
	var out []string
	for _, r := range s.Rows {
		if r.Label == "" && len(out) > 0 {
			out[len(out)-1] += " " + r.Value
			continue
		}
		out = append(out, r.Label+" "+r.Value)
	}
	for i, l := range out {
		out[i] = strings.Join(strings.Fields(l), " ")
	}
	return out
}

// panelLineRow is the row line n of the panel starts on, or -1.
func panelLineRow(s popupSpec, n int) int {
	line := -1
	for i, r := range s.Rows {
		if r.Label != "" {
			line++
		}
		if line == n {
			return i
		}
	}
	return -1
}

// PanelRowRect is the window rectangle of the first row of the shown panel's
// line n.
func (e *MapEditor) PanelRowRect(n int) (image.Rectangle, bool) {
	if !e.PanelOpen() {
		return image.Rectangle{}, false
	}
	s, _ := e.panelSpec()
	row := panelLineRow(s, n)
	for _, r := range layoutPopup(s, e.font, e.bodyFont()).Rows {
		if r.Index == row {
			return r.Rect, true
		}
	}
	return image.Rectangle{}, false
}

// panelPress is a press on the panel.
func (e *MapEditor) panelPress(p image.Point) {
	s, links := e.panelSpec()
	switch part, n := layoutPopup(s, e.font, e.bodyFont()).hit(p); part {
	case popupClose:
		e.cancelPopup(popList)
	case popupRowLine, popupRowControl:
		if n < len(links) {
			e.panelFollow(links[n])
		}
	}
}

// panelFollow opens the place a panel line names.
func (e *MapEditor) panelFollow(link panelLink) {
	switch link {
	case linkName:
		e.openMetadata("name")
	case linkDescription:
		e.openMetadata("description")
	case linkMode:
		e.openMetadata(ModeFieldKey)
	case linkPlayers:
		e.showRecords(editorPlayersFilter)
	case linkStart:
		e.scriptKey("drop")
	case linkGoals:
		if len(e.report.goals) == 0 {
			e.scriptKey("goals")
		}
	case linkReport:
		if !e.report.open {
			e.reportKey()
		}
	case linkTestPlay:
		_ = e.keyTestPlay("")
	}
}

// ModeFieldKey is the key of the map metadata field that holds the mode.
const ModeFieldKey = "70"

// openMetadata opens the map metadata prompt on the field key.
func (e *MapEditor) openMetadata(key string) {
	host, ok := e.workspace()
	if !ok {
		return
	}
	fields := host.WorkspaceFields("metadata")
	for n, f := range fields {
		if f.Key == key {
			e.leave(leaveDrag | leavePick)
			e.openPrompt(editorPrompt{kind: promptWorkspace, scope: "metadata", field: n, path: fieldInput(f)})
			return
		}
	}
}

// noticeStep is what the notice continues to.
type noticeStep uint8

const (
	noticeNone noticeStep = iota
	noticeTestPlay
	noticeModSave
)

// The notice's buttons, in order.
const (
	noticeGo = iota
	noticeStay
)

// panelWarnings are the panel's lines the checks find wanting, as the notice
// lists them; none while the report holds an error, which refuses the step on
// its own.
func (e *MapEditor) panelWarnings() []mapLine {
	p, ok := e.mapPanel()
	if !ok {
		return nil
	}
	findings := e.panelFindings()
	if errs, _ := CountFindings(findings); errs > 0 {
		return nil
	}
	var out []mapLine
	for _, l := range e.panelLines(p.MapFacts(), findings) {
		if l.warn && (l.link == linkStart || l.link == linkGoals) {
			out = append(out, l)
		}
	}
	return out
}

// askNotice opens the notice before step when the panel's checks find the
// map wanting, and reports whether it did.
func (e *MapEditor) askNotice(step noticeStep) bool {
	if len(e.panelWarnings()) == 0 {
		return false
	}
	e.leave(leaveDrag | leavePick)
	e.openPrompt(editorPrompt{kind: promptNotice, notice: step})
	return true
}

// PromptButtonRect is the window rectangle of button n of the open prompt or
// question.
func (e *MapEditor) PromptButtonRect(n int) (image.Rectangle, bool) {
	s, ok := e.promptSpec()
	if !ok {
		return image.Rectangle{}, false
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	if n < 0 || n >= len(l.Buttons) {
		return image.Rectangle{}, false
	}
	return l.Buttons[n], true
}

// PromptRect is the window rectangle of the open prompt or question.
func (e *MapEditor) PromptRect() (image.Rectangle, bool) {
	s, ok := e.promptSpec()
	if !ok {
		return image.Rectangle{}, false
	}
	return layoutPopup(s, e.font, e.bodyFont()).Rect, true
}

// PromptNotes are the note lines of the open prompt or question.
func (e *MapEditor) PromptNotes() []string {
	s, ok := e.promptSpec()
	if !ok {
		return nil
	}
	return s.Notes
}

// NoticeOpen reports whether the notice before test play or a save into the
// mod is open.
func (e *MapEditor) NoticeOpen() bool { return e.prompt.kind == promptNotice }

// noticeContinue leaves the notice and runs the step it held.
func (e *MapEditor) noticeContinue() error {
	step := e.prompt.notice
	e.leave(leavePrompt)
	switch step {
	case noticeTestPlay:
		return e.testPlay()
	case noticeModSave:
		if m, ok := e.modSave(); ok {
			return e.saveIntoMod(m)
		}
	}
	return nil
}

// noticeSpec is the notice as a popup description.
func (e *MapEditor) noticeSpec() popupSpec {
	title := e.word("editor.notice.testplay")
	if e.prompt.notice == noticeModSave {
		title = e.word("editor.notice.modsave")
	}
	var notes []string
	notes = append(notes, e.word("editor.notice.text"))
	for _, l := range e.panelWarnings() {
		notes = append(notes, l.label+": "+l.value)
	}
	return e.questionSpec(title, notes, []string{e.word("editor.notice.continue"), e.word("editor.button.cancel")})
}

// noticePress is a press on the notice.
func (e *MapEditor) noticePress(p image.Point) {
	part, n := layoutPopup(e.noticeSpec(), e.font, e.bodyFont()).hit(p)
	switch {
	case part == popupClose || part == popupButton && n == noticeStay:
		e.cancelPopup()
	case part == popupButton && n == noticeGo:
		_ = e.noticeContinue()
	}
}

// The new-map question's buttons, in order.
const (
	newModeSingle = iota
	newModeMulti
	newModeStay
)

// askNewMode asks whether the new map of source is single-player or
// multiplayer before it opens.
func (e *MapEditor) askNewMode(source string) {
	e.leave(leaveDrag | leavePick)
	e.openPrompt(editorPrompt{kind: promptNewMode, then: leaveStep{open: true, source: source}})
}

// newModeOpen leaves the question and opens its new map in mode.
func (e *MapEditor) newModeOpen(multi bool) error {
	source := e.prompt.then.source
	e.leave(leavePrompt)
	if multi {
		source += NewMapMultiplayer
	}
	return e.Open(source)
}

// newModeSpec is the new-map question as a popup description.
func (e *MapEditor) newModeSpec() popupSpec {
	return e.questionSpec(e.word("editor.newmap.title"), []string{e.word("editor.newmap.text")},
		[]string{e.word("editor.mode.single"), e.word("editor.mode.multi"), e.word("editor.button.cancel")})
}

// newModePress is a press on the new-map question.
func (e *MapEditor) newModePress(p image.Point) {
	part, n := layoutPopup(e.newModeSpec(), e.font, e.bodyFont()).hit(p)
	switch {
	case part == popupClose || part == popupButton && n == newModeStay:
		e.cancelPopup()
	case part == popupButton && (n == newModeSingle || n == newModeMulti):
		_ = e.newModeOpen(n == newModeMulti)
	}
}

// questionSpec is a question in the window's middle: a title, wrapped note
// lines and a row of buttons wide enough for the longest caption.
func (e *MapEditor) questionSpec(title string, notes, buttons []string) popupSpec {
	w := min(560, e.window.X-16)
	s := popupSpec{Title: title, Width: w, Bounds: e.popupBounds(), NoteInk: editorWarning, LabelWidth: 1, Buttons: buttons}
	if f := e.bodyFont(); f != nil {
		for _, b := range s.Buttons {
			s.ButtonW = max(s.ButtonW, f.Advance(b)+16)
		}
		s.Width = min(max(s.Width, len(buttons)*s.ButtonW+(len(buttons)+1)*popupPad), e.window.X-16)
	}
	text := s.Width - 2*popupPad - 60
	for _, n := range notes {
		s.Notes = append(s.Notes, editorWrap(e.bodyFont(), n, text)...)
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	s.At = e.promptPlace(image.Pt((e.window.X-l.Rect.Dx())/2, max(e.barHeight()+8, (e.window.Y-l.Rect.Dy())/3)))
	return s
}
