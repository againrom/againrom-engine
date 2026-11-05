package ui

import (
	"maps"
	"slices"
)

type barWindowSet struct {
	list, panel, object bool
	filter              int
	record              objectPlace
	back                []objectPlace
	selection           []uint64
	page                uint32
	metadata            string
	document            [32]byte
	report              reportState
	brush               brushState
	add                 barAddWindow
	choice              int64
	gridTop, brushTop   int
	places              [popCount]popupStanding
}

type barAddWindow struct {
	kind     RecordKind
	title    string
	steps    []addStep
	seed     InspectionNewRecord
	listed   uint32
	register uint32
	relabel  bool
	repeat   bool
}

func (e *MapEditor) barWindowSet() barWindowSet {
	s := barWindowSet{list: e.listShown, panel: e.panel, filter: e.filter,
		object: !e.objectHidden, record: e.objectPlace(), page: e.messagePage,
		brush: e.brush, report: e.barReportCopy(), places: e.places,
		gridTop: e.place.top, brushTop: e.place.brushTop}
	s.record.group.members = nil
	if e.doc != nil {
		s.document = barRecordState(e.doc)
	}
	s.record.identity = e.barRecordKey(RecordRef{Kind: s.record.kind, Index: s.record.index})
	s.back = slices.Clone(e.objectBack)
	for i := range s.back {
		s.back[i].draft, s.back[i].group.members = nil, nil
		s.back[i].identity = e.barRecordKey(RecordRef{Kind: s.back[i].kind, Index: s.back[i].index})
	}
	s.brush.endStroke()
	s.brush.forest.ticks = maps.Clone(e.brush.forest.ticks)
	if e.selectionWindow() {
		for _, ref := range e.selection.refs {
			s.selection = append(s.selection, e.barRecordKey(ref))
		}
	}
	if e.prompt.kind == promptWorkspace && e.prompt.scope == "metadata" {
		if fields := e.promptFields(); e.prompt.field >= 0 && e.prompt.field < len(fields) {
			s.metadata = fields[e.prompt.field].Key
		}
	}
	if run, ok := e.addRunning(); ok {
		s.add = barAddWindow{kind: run.kind, title: run.title, steps: slices.Clone(run.steps),
			seed: run.seed, listed: run.listed, repeat: run.repeat}
		s.add.seed.Values = slices.Clone(run.seed.Values)
		if run.relabel != nil {
			s.add.register, s.add.relabel = *run.relabel, true
		}
		if run.repeat {
			key := "kind"
			if run.kind == KindUnit {
				key = "identity"
			}
			_, s.choice, _ = e.placing(key)
		}
	}
	return s
}

func (e *MapEditor) restoreBarWindows(s barWindowSet) {
	e.closeObject()
	if !s.list {
		e.leave(leaveRecords)
	}
	if !s.panel {
		e.leave(leavePanel)
	}
	if s.brush.mode == brushOff {
		e.leave(leaveBrush)
	} else {
		e.restoreBarBrush(s.brush)
	}
	e.filter = s.filter
	e.place.top, e.place.brushTop = s.gridTop, s.brushTop
	if s.object {
		opened := false
		switch p := s.record; {
		case p.kind == KindVariable:
			for _, v := range e.scriptVariables() {
				if v.register == p.register {
					e.selectVariable(p.register)
					opened = true
					break
				}
			}
		case p.group.set:
			for _, g := range e.groups() {
				if g.player == p.group.player && g.group == p.group.group {
					e.chooseGroup(g, false)
					opened = true
					break
				}
			}
		default:
			if n := e.linkedRecord(RecordRef{Kind: p.kind, Index: p.index}, p.identity); n >= 0 {
				opened = e.Select(n, false)
			}
		}
		if opened {
			e.objectBack = s.back
			e.ShowTriggerDetails(s.record.details != 0)
			e.obj.doc = nil
			e.loadObject()
			e.obj.section = s.record.section
			e.objectTop = s.record.top
		}
		if s.page != 0 && s.page != NewMessagePage {
			if host, ok := e.doc.Edits.(InspectionMessages); ok && len(host.MessageFields(s.page)) != 0 {
				e.openMessage(s.page)
				e.objectBack = s.back
			}
		}
	}
	if s.report.open {
		e.openReport(s.report.rows, s.report.note, s.report.goals)
		e.report.sel, e.report.copies, e.report.hint = s.report.sel, s.report.copies, s.report.hint
		if e.doc != nil && s.document != barRecordState(e.doc) {
			e.refreshBarReport()
		}
	}
	if len(s.selection) > 0 {
		var refs []RecordRef
		for _, key := range s.selection {
			if n := e.linkedRecord(RecordRef{}, key); n >= 0 {
				r := e.doc.Records[n]
				refs = append(refs, RecordRef{Kind: r.Kind, Index: r.Index})
			}
		}
		e.setSelection(refs)
	}
	if s.add.kind != "" {
		if s.add.repeat {
			for n, choice := range e.placeShown() {
				if choice.Value == s.choice {
					e.placeChoose(n)
					break
				}
			}
		} else {
			run := &addRun{kind: s.add.kind, title: s.add.title, seed: s.add.seed, listed: s.add.listed}
			valid := true
			if s.add.relabel {
				valid = false
				for _, v := range e.scriptVariables() {
					valid = valid || v.register == s.add.register
				}
				register := s.add.register
				run.relabel = &register
			}
			if s.add.listed != 0 {
				_, valid = e.nodeByID(true, s.add.listed)
			}
			if valid {
				run.setSteps(s.add.steps)
				e.openAddRun(run)
			}
		}
	}
	if s.metadata != "" {
		e.openMetadata(s.metadata)
	}
	e.restoreBarPlaces(s.places)
	e.cancelRestore, e.listRestore = nil, nil
	e.input = editorInputState{}
	e.message = ""
}
