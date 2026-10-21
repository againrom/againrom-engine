package ui

import (
	"fmt"
	"image"
	"image/draw"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// The units' groups as the window edits them. A group is the group number
// its member units carry (ALM-UNIT-048), one player's units; there is no
// group record, and a script names a group by its number alone
// (ALM-TRIG-046). Putting units in a group writes the number on each unit
// and taking them out writes 0, each as one edit through the host.

// InspectionGroups is implemented by a host that sets units' group numbers.
// SetUnitsGroup writes group on every unit refs names as one edit, 0 taking
// them out of their groups; it refuses units of two players and a number a
// unit of another player carries. FreeGroup is the lowest number no unit of
// the map carries.
type InspectionGroups interface {
	SetUnitsGroup(refs []RecordRef, group int) (*InspectionDocument, error)
	FreeGroup() int
}

func (e *MapEditor) groupsHost() (InspectionGroups, bool) {
	if !e.editable() || e.catalog {
		return nil, false
	}
	h, ok := e.doc.Edits.(InspectionGroups)
	return h, ok
}

// recordIndex maps each record of the document to its index.
func (e *MapEditor) recordIndex() map[RecordRef]int {
	out := map[RecordRef]int{}
	if e.doc != nil {
		for i, r := range e.doc.Records {
			out[RecordRef{r.Kind, r.Index}] = i
		}
	}
	return out
}

// groupSplit is the units of refs a group of player takes, in the order of
// refs; player 0 takes the first unit's player and -1 every unit. others
// counts the units of other players and notUnits the records that are not
// units.
type groupSplit struct {
	units            []RecordRef
	player           int
	others, notUnits int
}

func (e *MapEditor) splitGroup(refs []RecordRef, player int) groupSplit {
	at := e.recordIndex()
	out := groupSplit{player: player}
	for _, ref := range refs {
		i, ok := at[ref]
		if !ok || ref.Kind != "Unit" {
			out.notUnits++
			continue
		}
		owner := e.doc.Records[i].Player
		if out.player == 0 {
			out.player = owner
		}
		if out.player >= 0 && owner != out.player {
			out.others++
			continue
		}
		out.units = append(out.units, ref)
	}
	return out
}

// skipNotes are the notes of what a group action left out.
func (e *MapEditor) skipNotes(s groupSplit) []string {
	var out []string
	if s.others > 0 {
		out = append(out, fmt.Sprintf(e.word("editor.group.skipped"), s.others, s.player))
	}
	if s.notUnits > 0 {
		out = append(out, fmt.Sprintf(e.word("editor.group.notunits"), s.notUnits))
	}
	return out
}

// groupNumbers are the groups with number n, by player.
func (e *MapEditor) groupNumbered(n int) []editorGroup {
	var out []editorGroup
	for _, g := range e.groups() {
		if g.group == n {
			out = append(out, g)
		}
	}
	return out
}

// groupUsers are the script rows that name group n, as the group popup's
// Used by section names them: each trigger with its nodes, then each node no
// trigger runs.
func (e *MapEditor) groupUsers(n int) []string {
	if e.doc == nil || e.doc.Script == nil || n == 0 {
		return nil
	}
	target := e.doc.Script.OpenTarget(scriptGroup, uint32(n))
	if target == nil {
		return nil
	}
	var out []string
	for _, row := range e.usesRows(target.Uses) {
		out = append(out, row.title)
	}
	return out
}

// groupWarning is a group a trigger names that an edit would leave with no
// unit, and the triggers.
type groupWarning struct {
	group int
	users []string
}

// groupEmptied are the groups a trigger names that moving units to group
// would leave with no member; group -1 names no group, which is how deleting
// the records asks.
func (e *MapEditor) groupEmptied(units []RecordRef, group int) []groupWarning {
	moving := map[int]bool{}
	at := e.recordIndex()
	for _, ref := range units {
		if i, ok := at[ref]; ok {
			moving[i] = true
		}
	}
	var out []groupWarning
	for _, g := range e.groups() {
		if g.group == group {
			continue
		}
		left := slices.ContainsFunc(g.members, func(id int) bool { return !moving[id] })
		if users := e.groupUsers(g.group); !left && len(users) > 0 && !slices.ContainsFunc(out, func(w groupWarning) bool { return w.group == g.group }) {
			out = append(out, groupWarning{group: g.group, users: users})
		}
	}
	return out
}

// groupPrompt is what a group question holds: the units, the group they go
// to (0: out of their groups) with its player, the notes of what was left
// out, and the To group list's groups (0: a new one) and its model.
type groupPrompt struct {
	units  []RecordRef
	group  int
	player int
	skips  []string
	warn   []groupWarning
	// del is a delete the warning holds, of units; single is the selected
	// record's own delete rather than the selection's.
	del, single bool
	groups      []int
	list        *Picker
}

// groupUnitsSelected is how many units the selection, or the selected
// record, holds.
func (e *MapEditor) groupUnitsSelected() int {
	return len(e.splitGroup(e.selectionRefs(), -1).units)
}

// toGroupAsk opens the To group question for the selection: a new group,
// then each group of the first unit's player.
func (e *MapEditor) toGroupAsk() {
	h, ok := e.groupsHost()
	if !ok {
		e.noteSelection(e.word("editor.selection.nohost"), nil)
		return
	}
	s := e.splitGroup(e.selectionRefs(), 0)
	if len(s.units) == 0 {
		e.noteSelection(e.word("editor.group.none"), nil)
		return
	}
	free := h.FreeGroup()
	p := &groupPrompt{units: s.units, player: s.player, skips: e.skipNotes(s), groups: []int{0}}
	rows := []PickerRow{{Text: fmt.Sprintf(e.word("editor.group.new"), free), Choosable: true}}
	for _, g := range e.groups() {
		if g.player == s.player {
			p.groups = append(p.groups, g.group)
			rows = append(rows, PickerRow{Text: fmt.Sprintf(e.word("editor.group.row"), g.group, len(g.members)), Choosable: true})
		}
	}
	p.list = NewPicker(rows).SetWindow(min(groupListRows, len(rows)))
	p.list.Select(0)
	e.leave(leaveDrag | leavePick)
	e.openPrompt(editorPrompt{kind: promptToGroup, group: p})
}

const groupListRows = 8

// toGroupSpec is the To group question as data.
func (e *MapEditor) toGroupSpec() popupSpec {
	p := e.prompt.group
	notes := append([]string{fmt.Sprintf(e.word("editor.group.totext"), len(p.units), p.player)}, p.skips...)
	s := e.questionSpec(e.word("editor.group.totitle"), notes, []string{e.word("editor.button.ok"), e.word("editor.button.cancel")})
	s.List, s.ListRows = p.list, min(groupListRows, p.list.Len())
	list := p.list
	s.ListText = func(row, width int) string {
		rows := list.Rows()
		if row < 0 || row >= len(rows) {
			return ""
		}
		return editorClip(e.bodyFont(), rows[row].Text, width)
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	s.At = e.promptPlace(image.Pt((e.window.X-l.Rect.Dx())/2, max(e.barHeight()+8, (e.window.Y-l.Rect.Dy())/3)))
	return s
}

// toGroupPress is a press on the To group question.
func (e *MapEditor) toGroupPress(p image.Point, double bool) {
	s, _ := e.promptSpec()
	q := e.prompt.group
	switch part, n := layoutPopup(s, e.font, e.bodyFont()).hit(p); {
	case part == popupClose || part == popupButton && n == 1:
		e.cancelPopup()
	case part == popupButton && n == 0:
		e.toGroupChosen()
	case part == popupListRow:
		top, count := q.list.Visible()
		if n < count {
			q.list.Select(top + n)
			if double {
				e.toGroupChosen()
			}
		}
	}
}

// toGroupChosen puts the question's units in the highlighted group.
func (e *MapEditor) toGroupChosen() {
	q := e.prompt.group
	k := q.list.Selection()
	if k < 0 || k >= len(q.groups) {
		return
	}
	group := q.groups[k]
	if group == 0 {
		h, ok := e.groupsHost()
		if !ok {
			return
		}
		group = h.FreeGroup()
	}
	e.leave(leavePrompt)
	e.groupTo(q.units, group, q.player, q.skips, false)
}

// groupTo puts units in group, or with group 0 takes them out of their
// groups, as one edit. An edit that would leave a group a trigger names
// with no unit asks first, unless confirmed. skips are the notes of what
// the action left out, shown with its outcome.
func (e *MapEditor) groupTo(units []RecordRef, group, player int, skips []string, confirmed bool) {
	h, ok := e.groupsHost()
	if !ok {
		e.noteSelection(e.word("editor.selection.nohost"), nil)
		return
	}
	if len(units) == 0 {
		e.noteSelection(strings.Join(append([]string{e.word("editor.group.none")}, skips...), "; "), nil)
		return
	}
	if warn := e.groupEmptied(units, group); len(warn) > 0 && !confirmed {
		e.leave(leaveDrag | leavePick)
		e.openPrompt(editorPrompt{kind: promptGroupWarn, group: &groupPrompt{units: units, group: group, player: player, skips: skips, warn: warn}})
		return
	}
	next, err := h.SetUnitsGroup(units, group)
	if err != nil {
		e.noteSelection(e.editorHostText(err.Error()), nil)
		return
	}
	if next != nil {
		if err := e.adoptEdit(next, nil); err != nil {
			return
		}
	}
	note := fmt.Sprintf(e.word("editor.group.removed"), len(units))
	if group != 0 {
		note = fmt.Sprintf(e.word("editor.group.put"), len(units), group)
	}
	chosen := false
	for _, g := range e.groups() {
		if group != 0 && g.group == group && g.player == player {
			e.chooseGroup(g, false)
			chosen = true
		}
	}
	if !chosen {
		e.setSelection(units)
	}
	e.noteSelection(strings.Join(append([]string{note}, skips...), "; "), nil)
}

// groupWarnSpec is the question before a group a trigger names is emptied.
func (e *MapEditor) groupWarnSpec() popupSpec {
	var notes []string
	for _, w := range e.prompt.group.warn {
		notes = append(notes, fmt.Sprintf(e.word("editor.group.warntext"), w.group, strings.Join(w.users, "; ")))
	}
	return e.questionSpec(e.word("editor.group.warntitle"), notes, []string{e.word("editor.group.warngo"), e.word("editor.button.cancel")})
}

// groupWarnPress is a press on the warning.
func (e *MapEditor) groupWarnPress(p image.Point) {
	s, _ := e.promptSpec()
	switch part, n := layoutPopup(s, e.font, e.bodyFont()).hit(p); {
	case part == popupClose || part == popupButton && n == 1:
		e.cancelPopup()
	case part == popupButton && n == 0:
		e.groupWarnGo()
	}
}

// groupWarnGo runs the edit the warning held.
func (e *MapEditor) groupWarnGo() {
	q := e.prompt.group
	e.leave(leavePrompt)
	switch {
	case q.del && q.single:
		_ = e.deleteSelected(q.units[0].Kind, q.units[0].Index)
	case q.del:
		_ = e.deleteRecords()
	default:
		e.groupTo(q.units, q.group, q.player, q.skips, true)
	}
}

// removeFromGroup takes the selection's units out of their groups: the units
// of the first grouped unit's player, or of the first unit's player when none
// is grouped. The units of other players are left out and noted.
func (e *MapEditor) removeFromGroup() {
	refs := e.selectionRefs()
	at := e.recordIndex()
	player := 0
	for _, ref := range refs {
		i, ok := at[ref]
		if !ok || ref.Kind != "Unit" {
			continue
		}
		r := e.doc.Records[i]
		if player == 0 {
			player = r.Player
		}
		if r.Group != 0 {
			player = r.Player
			break
		}
	}
	s := e.splitGroup(refs, player)
	e.groupTo(s.units, 0, s.player, e.skipNotes(s), false)
}

// askDelete opens the warning before a delete of refs empties a group a
// trigger names, and reports whether it did.
func (e *MapEditor) askDelete(refs []RecordRef, single bool) bool {
	warn := e.groupEmptied(refs, -1)
	if len(warn) == 0 {
		return false
	}
	e.leave(leaveDrag | leavePick)
	e.openPrompt(editorPrompt{kind: promptGroupWarn, group: &groupPrompt{units: refs, warn: warn, del: true, single: single}})
	return true
}

// removeGroupOf empties group g: every member's number goes to 0.
func (e *MapEditor) removeGroupOf(g editorGroup) {
	var units []RecordRef
	for _, id := range g.members {
		r := e.doc.Records[id]
		units = append(units, RecordRef{r.Kind, r.Index})
	}
	e.groupTo(units, 0, g.player, nil, false)
}

// groupKey is a number key: 1..9 selects group n and frames it, Ctrl+1..9
// puts the selection's units in group n, the units of the group's player
// when it has one, else of the first unit's player.
func (e *MapEditor) groupKey(key string) error {
	if e.doc == nil || e.catalog {
		return nil
	}
	put := strings.HasPrefix(key, "togroup")
	n, err := strconv.Atoi(strings.TrimPrefix(strings.TrimPrefix(key, "togroup"), "group"))
	if err != nil {
		return nil
	}
	have := e.groupNumbered(n)
	if !put {
		if len(have) == 0 {
			e.noteSelection(fmt.Sprintf(e.word("editor.group.nogroup"), n), nil)
			return nil
		}
		e.leave(leaveDrag | leavePick)
		e.chooseGroup(have[0], true)
		return nil
	}
	player := 0
	if len(have) > 0 {
		player = have[0].player
	}
	s := e.splitGroup(e.selectionRefs(), player)
	e.leave(leaveDrag | leavePick)
	e.groupTo(s.units, n, s.player, e.skipNotes(s), false)
	return nil
}

// groupsAct is what a row of the Groups window does: the group it chooses,
// unset for a row that does nothing.
type groupsAct struct{ group editorGroup }

// groupsShown reports that the list popup shows the Groups window.
func (e *MapEditor) groupsShown() bool {
	return e.listKind() == "records" && e.filter == editorGroupsFilter
}

// memberIcons are the pictures of g's members, as the map draws them.
func (e *MapEditor) memberIcons(g editorGroup) []*image.RGBA {
	var out []*image.RGBA
	for _, id := range g.members {
		if id < len(e.doc.Records) && e.doc.Records[id].Preview != nil {
			out = append(out, e.doc.Records[id].Preview.RGBA())
		}
	}
	return out
}

// groupsSpec is the Groups window as data: per player a heading, then each
// group's number, its member count and pictures, a cross that removes it
// and the triggers that name it; below, the selection's unit count and the
// two buttons that put the selection in a group and take it out.
func (e *MapEditor) groupsSpec() (popupSpec, []groupsAct) {
	if !e.listPlaced {
		e.listAt, e.listPlaced = image.Pt(8, e.barHeight()+8), true
	}
	body := e.bodyFont()
	s := popupSpec{Title: e.word("editor.bar.groups"), At: e.listAt, Width: e.popupWide(popList, 460), Bounds: e.popupBounds(), NoteInk: editorPaper, LabelWidth: 96}
	var acts []groupsAct
	add := func(row popupRow, g editorGroup) {
		s.Rows, acts = append(s.Rows, row), append(acts, groupsAct{group: g})
	}
	text := s.Width - 2*popupPad - 8
	player := 0
	groups := e.groups()
	for _, g := range groups {
		if g.player != player {
			player = g.player
			add(popupRow{Kind: popupHeading, Label: fmt.Sprintf(e.word("editor.group.playerheading"), g.player)}, editorGroup{})
		}
		label := fmt.Sprintf(e.word("editor.group.number"), g.group)
		row := popupRow{Kind: popupLink, Label: label, Value: fmt.Sprintf(e.word("editor.group.count"), len(g.members)), Buttons: []string{"x"}, Icons: e.memberIcons(g)}
		if e.groupSel.set && g.player == e.groupSel.player && g.group == e.groupSel.group {
			row.Label, row.Ink = "> "+label, editorGold
		}
		add(row, g)
		if users := e.groupUsers(g.group); len(users) > 0 {
			for _, part := range editorWrap(body, fmt.Sprintf(e.word("editor.group.usedby"), strings.Join(users, "; ")), text) {
				add(popupRow{Kind: popupText, Label: part}, editorGroup{})
			}
		}
	}
	if len(groups) == 0 {
		s.Notes = append(s.Notes, editorWrap(body, e.word("editor.group.empty"), text)...)
	}
	units := e.groupUnitsSelected()
	s.Notes = append(s.Notes, fmt.Sprintf(e.word("editor.group.selected"), units))
	if note := e.wholeSelectionNote(); note != "" {
		s.Notes = append(s.Notes, editorWrap(body, note, text)...)
	}
	s.Buttons = []string{e.word("editor.group.togroup"), e.word("editor.group.remove")}
	s.Disabled = []bool{units == 0, units == 0}
	if body != nil {
		for _, b := range s.Buttons {
			s.ButtonW = max(s.ButtonW, body.Advance(b)+16)
		}
	}
	s.MaxRows = max(3, e.listWindow())
	e.groupsTop = max(0, min(e.groupsTop, len(s.Rows)-s.MaxRows))
	s.Top = e.groupsTop
	return s, acts
}

// groupsPress is a press on the Groups window.
func (e *MapEditor) groupsPress(p image.Point) {
	s, acts := e.groupsSpec()
	l := layoutPopup(s, e.font, e.bodyFont())
	switch part, n := l.hit(p); part {
	case popupClose:
		e.cancelPopup(popList)
	case popupRowButton:
		if n < len(acts) && acts[n].group.set {
			e.removeGroupOf(acts[n].group)
		}
	case popupRowLine, popupRowControl:
		if n < len(acts) && acts[n].group.set {
			e.leave(leaveDrag | leavePick)
			e.chooseGroup(acts[n].group, true)
		}
	case popupButton:
		if n < len(s.Disabled) && s.Disabled[n] {
			return
		}
		if n == 0 {
			e.toGroupAsk()
		} else {
			e.removeFromGroup()
		}
	}
}

// GroupsRowRect is the window rectangle of the Groups window's row of
// player's group number, and of its cross when cross is set.
func (e *MapEditor) GroupsRowRect(player, group int, cross bool) (image.Rectangle, bool) {
	if !e.groupsShown() {
		return image.Rectangle{}, false
	}
	s, acts := e.groupsSpec()
	for _, rr := range layoutPopup(s, e.font, e.bodyFont()).Rows {
		if g := acts[rr.Index].group; g.set && g.player == player && g.group == group {
			if cross {
				return rr.Button, true
			}
			return rr.Label, true
		}
	}
	return image.Rectangle{}, false
}

// GroupsButtonRect is the window rectangle of button n of the Groups window:
// 0 To group, 1 Remove from group.
func (e *MapEditor) GroupsButtonRect(n int) (image.Rectangle, bool) {
	if !e.groupsShown() {
		return image.Rectangle{}, false
	}
	s, _ := e.groupsSpec()
	l := layoutPopup(s, e.font, e.bodyFont())
	if n < 0 || n >= len(l.Buttons) {
		return image.Rectangle{}, false
	}
	return l.Buttons[n], true
}

// GroupsText is the Groups window's rows and notes as text, in the font's
// bytes, for a headless driver.
func (e *MapEditor) GroupsText() []string {
	if !e.groupsShown() {
		return nil
	}
	s, _ := e.groupsSpec()
	var out []string
	for _, r := range s.Rows {
		out = append(out, strings.TrimSpace(r.Label+" "+r.Value))
	}
	return append(out, s.Notes...)
}

// PromptListRowRect is the window rectangle of row n of the open question's
// list, when it is shown.
func (e *MapEditor) PromptListRowRect(n int) (image.Rectangle, bool) {
	s, ok := e.promptSpec()
	if !ok || s.List == nil {
		return image.Rectangle{}, false
	}
	top, count := s.List.Visible()
	if n < top || n >= top+count {
		return image.Rectangle{}, false
	}
	return layoutPopup(s, e.font, e.bodyFont()).List.Row(n - top), true
}

// selectionSpec is the popup of a selection of several records: what it
// holds and the group buttons and Delete. A chosen group's popup stands
// instead.
func (e *MapEditor) selectionSpec() (popupSpec, bool) {
	if !e.selectionShown() || e.selection.hidden || e.groupSel.set || e.pick != pickNone || e.prompt.kind == promptAdd || e.messagePage != 0 {
		return popupSpec{}, false
	}
	if !e.objPlaced {
		e.objectAt, e.objPlaced = image.Pt(e.window.X-8-420, e.barHeight()+8), true
	}
	s := popupSpec{Title: fmt.Sprintf(e.word("editor.selection.count"), len(e.selection.refs)), At: e.objectAt, Width: e.popupWide(popObject, 420), Bounds: e.popupBounds(), NoteInk: editorPaper}
	counts := map[string]int{}
	groups := map[int]bool{}
	at := e.recordIndex()
	for _, ref := range e.selection.refs {
		counts[ref.Kind]++
		if i, ok := at[ref]; ok && ref.Kind == "Unit" && e.doc.Records[i].Group != 0 {
			groups[e.doc.Records[i].Group] = true
		}
	}
	for _, k := range []struct{ kind, word string }{{"Unit", "editor.bar.units"}, {"Structure", "editor.bar.structures"}, {"Sack", "editor.bar.sacks"}} {
		if counts[k.kind] > 0 {
			s.Rows = append(s.Rows, popupRow{Kind: popupText, Label: e.word(k.word), Value: strconv.Itoa(counts[k.kind])})
		}
	}
	var numbers []string
	for _, n := range slices.Sorted(maps.Keys(groups)) {
		numbers = append(numbers, strconv.Itoa(n))
	}
	if len(numbers) > 0 {
		s.Rows = append(s.Rows, popupRow{Kind: popupText, Label: e.word("editor.bar.groups"), Value: strings.Join(numbers, ", ")})
	}
	if note := e.wholeSelectionNote(); note != "" {
		s.Notes = editorWrap(e.bodyFont(), note, s.Width-2*popupPad-8)
	}
	units := counts["Unit"]
	s.Buttons = []string{e.word("editor.group.togroup"), e.word("editor.group.remove"), e.word("editor.selection.delete")}
	s.Disabled = []bool{units == 0, units == 0, false}
	if f := e.bodyFont(); f != nil {
		for _, b := range s.Buttons {
			s.ButtonW = max(s.ButtonW, f.Advance(b)+16)
		}
		s.Width = max(s.Width, len(s.Buttons)*s.ButtonW+(len(s.Buttons)+1)*popupPad)
	}
	return s, true
}

// selectionPress is a press on the selection's popup.
func (e *MapEditor) selectionPress(p image.Point) {
	s, _ := e.selectionSpec()
	switch part, n := layoutPopup(s, e.font, e.bodyFont()).hit(p); part {
	case popupClose:
		e.selection.hidden = true
	case popupButton:
		if n < len(s.Disabled) && s.Disabled[n] {
			return
		}
		switch n {
		case 0:
			e.toGroupAsk()
		case 1:
			e.removeFromGroup()
		case 2:
			_ = e.deleteSelection()
		}
	}
}

// SelectionButtonRect is the window rectangle of button n of the selection's
// popup: 0 To group, 1 Remove from group, 2 Delete.
func (e *MapEditor) SelectionButtonRect(n int) (image.Rectangle, bool) {
	s, ok := e.selectionSpec()
	if !ok {
		return image.Rectangle{}, false
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	if n < 0 || n >= len(l.Buttons) {
		return image.Rectangle{}, false
	}
	return l.Buttons[n], true
}

// groupBadges are the number badges of the chosen group's members: a box
// above each member's cell holding the group number, in window pixels; a
// badge an open popup covers is left out.
func (e *MapEditor) groupBadges() []areaLabel {
	if !e.groupSel.set || e.doc == nil || e.catalog {
		return nil
	}
	f := e.bodyFont()
	text := strconv.Itoa(e.groupSel.group)
	w, h := 8, fontHeight(f, 15)+2
	if f != nil {
		w += f.Advance(text)
	}
	area, popups := e.mapArea(), e.popupRects()
	var out []areaLabel
	for _, id := range e.groupSel.members {
		if id >= len(e.doc.Records) {
			continue
		}
		c := e.CellPoint(e.doc.Records[id].Cell)
		r := image.Rect(c.X-w/2, c.Y-24-h, c.X-w/2+w, c.Y-24)
		covered := false
		for _, p := range popups {
			covered = covered || p.Overlaps(r)
		}
		if r.In(area) && !covered {
			out = append(out, areaLabel{rect: r, text: text})
		}
	}
	return out
}

// GroupBadges is how many badges the chosen group shows.
func (e *MapEditor) GroupBadges() int { return len(e.groupBadges()) }

// drawGroupBadges draws the chosen group's badges.
func (e *MapEditor) drawGroupBadges(pic *image.RGBA) {
	f := e.bodyFont()
	for _, b := range e.groupBadges() {
		draw.Draw(pic, b.rect, image.NewUniform(editorInk), image.Point{}, draw.Src)
		drawFrame(pic, panelFrame(b.rect, editorInk, editorCondition))
		if f != nil {
			f.Draw(pic.SubImage(b.rect).(*image.RGBA), b.text, b.rect.Min.X+4, b.rect.Min.Y+1, editorCondition)
		}
	}
}
