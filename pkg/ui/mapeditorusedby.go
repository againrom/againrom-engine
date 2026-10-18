package ui

import (
	"fmt"
	"slices"
	"strings"
)

type scriptPopupLink struct {
	record int
	title  string
	nodes  []int
}

func (e *MapEditor) scriptPopupLink(s *popupSpec, links *[]int, row scriptPopupLink) {
	for i, part := range editorWrap(e.bodyFont(), row.title, s.Width-2*popupPad-8) {
		kind, link := popupText, 0
		if i == 0 || row.nodes == nil {
			kind, link = popupLink, -row.record-1
		}
		s.Rows, *links = append(s.Rows, popupRow{Kind: kind, Label: part}), append(*links, link)
	}
}

func (e *MapEditor) scriptLinkSection(s *popupSpec, links *[]int, title string, rows []scriptPopupLink) {
	if len(rows) == 0 {
		title = fmt.Sprintf(e.word("editor.scripts.emptyheading"), title)
	}
	s.Rows, *links = append(s.Rows, popupRow{Kind: popupText, Label: title}), append(*links, 0)
	for _, row := range rows {
		e.scriptPopupLink(s, links, row)
	}
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

func (e *MapEditor) usedByRows() []scriptPopupLink {
	uses := e.objectUses()
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
		if e.doc.Records[row.record].Kind == "Trigger" {
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
	return kind == "Unit" || kind == "Structure" || kind == "Player"
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
