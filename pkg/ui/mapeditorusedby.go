package ui

import (
	"fmt"
	"slices"
	"strings"
)

// scriptPopupLink is one link of an object window: the record it opens, its
// text, the script nodes it names, and the spaces its lines stand in by.
type scriptPopupLink struct {
	record int
	title  string
	nodes  []int
	indent int
}

// memberIndent is how far a group's member links stand in under its
// members row.
const memberIndent = 2

// usesFoldMost is the most uses the used-by fold shows open at first.
const usesFoldMost = 5

// scriptPopupLink adds row's link: its text wrapped to the window's width,
// each line under the first standing in by row.indent, and every line a
// link to the record, so a press anywhere on it opens the record.
func (e *MapEditor) scriptPopupLink(s *popupSpec, links *[]int, row scriptPopupLink) {
	pad := strings.Repeat(" ", row.indent)
	width := s.Width - 2*popupPad - 8
	if b := e.bodyFont(); b != nil {
		width -= b.Advance(pad)
	}
	for _, part := range editorTextLines(e.bodyFont(), row.title, width) {
		s.Rows, *links = append(s.Rows, popupRow{Kind: popupLink, Label: pad + part}), append(*links, -row.record-1)
	}
}

// scriptLinkSection adds a block of links under its heading title: "none"
// after the title when there is no link, else a heading that folds as fold
// says, the links shown unless it is closed.
func (e *MapEditor) scriptLinkSection(s *popupSpec, links *[]int, title string, rows []scriptPopupLink, fold RowFold) {
	if len(rows) == 0 {
		title, fold = fmt.Sprintf(e.word("editor.scripts.emptyheading"), title), FoldNone
	}
	s.Rows, *links = append(s.Rows, popupRow{Kind: popupHeading, Label: title, Fold: fold}), append(*links, 0)
	if fold == FoldClosed {
		return
	}
	for _, row := range rows {
		e.scriptPopupLink(s, links, row)
	}
}

// usedBySection adds the shown object's used-by block: a fold over its
// uses, closed when there are more than usesFoldMost and open otherwise,
// until the map maker turns it.
func (e *MapEditor) usedBySection(s *popupSpec, links *[]int, rows []scriptPopupLink) {
	fold := FoldClosed
	if (len(rows) <= usesFoldMost) != e.obj.usesTurned {
		fold = FoldOpen
	}
	if len(rows) > 0 {
		e.obj.usesFold = len(s.Rows) + 1
	}
	e.scriptLinkSection(s, links, e.word("editor.usedby.title"), rows, fold)
}

func (e *MapEditor) objectUses() []InspectionScriptUse {
	if e.doc == nil || e.doc.Script == nil {
		return nil
	}
	m := e.doc.Script
	if e.groupSel.set && e.selected < 0 {
		if target := m.OpenTarget(scriptGroup, uint32(e.groupSel.group)); target != nil {
			return target.Uses
		}
		return nil
	}
	for _, kind := range []uint32{scriptUnit, scriptStructure, scriptPlayer} {
		for _, target := range m.Targets[kind] {
			if target.Record == e.selected {
				return target.Uses
			}
		}
	}
	return nil
}

func (e *MapEditor) usedByRows() []scriptPopupLink { return e.usesRows(e.objectUses()) }

// usesRows are the rows naming uses: one per trigger whose nodes name the
// value, with those nodes, then one per node no trigger runs.
func (e *MapEditor) usesRows(uses []InspectionScriptUse) []scriptPopupLink {
	var rows []scriptPopupLink
	if e.doc.Script != nil {
		for _, trigger := range e.doc.Script.Triggers {
			row := scriptPopupLink{record: trigger.Record, nodes: []int{}}
			for _, use := range uses {
				if slices.Contains(use.Triggers, trigger.Record) && !slices.Contains(row.nodes, use.Node) {
					row.nodes = append(row.nodes, use.Node)
				}
			}
			if len(row.nodes) > 0 {
				rows = append(rows, row)
			}
		}
	}
	for _, use := range uses {
		if len(use.Triggers) == 0 && !slices.ContainsFunc(rows, func(r scriptPopupLink) bool { return r.record == use.Node }) {
			rows = append(rows, scriptPopupLink{record: use.Node, nodes: []int{use.Node}})
		}
	}
	for i := range rows {
		row := &rows[i]
		var names []string
		for _, node := range row.nodes {
			names = append(names, e.scriptRowText(node, 0))
		}
		if e.doc.Records[row.record].Kind == KindTrigger {
			row.title = fmt.Sprintf(e.word("editor.usedby.trigger"), e.scriptRowText(row.record, 0), strings.Join(names, "; "))
		} else {
			row.title = fmt.Sprintf(e.word("editor.usedby.unattached"), strings.Join(names, "; "))
		}
	}
	return rows
}

func (e *MapEditor) hasUsedBy() bool {
	if e.groupSel.set && e.selected < 0 {
		return true
	}
	if e.doc == nil || e.selected < 0 || e.selected >= len(e.doc.Records) {
		return false
	}
	kind := e.doc.Records[e.selected].Kind
	return kind.Owned() || kind == KindPlayer
}

func (e *MapEditor) openObjectUse(record int) {
	e.loadObject()
	p, draft := e.objectPlace(), e.obj
	draft.buf = slices.Clone(draft.buf)
	p.draft = &draft
	back := append(e.objectBack, p)
	if e.Select(record, true) {
		e.objectBack = back
	}
}
