package ui

import (
	"image"
	"reflect"
	"slices"
	"testing"
)

func toolWindowsEditor(t *testing.T, key string) *MapEditor {
	t.Helper()
	e, _ := barWindowEditor(t)
	if key == "scripts" {
		s := scriptsListEditor(t)
		e.doc.Script, e.doc.Records = s.doc.Script, s.doc.Records
	}
	e.doc.Records[0].Player, e.doc.Records[0].Group = 1, 7
	e.barAct(key)
	switch key {
	case "scripts":
		e.Select(2, false)
	case "groups":
		e.chooseGroup(e.groups()[0], false)
	case "players", "units", "sacks", "structures", "cellspells":
		rows := e.recordsOf(e.filter)
		if len(rows) == 0 {
			t.Fatal("no record for", key)
		}
		e.Select(rows[len(rows)-1], false)
	case "objects", "trees":
		e.brush.sel[brushObjects] = 1
		e.brush.radius, e.brush.radiusSet = 4, true
		e.brush.forest.share = 50
		e.brush.forest.ticks = map[int]bool{1: true}
	}
	e.objectSpec()
	return e
}

func TestMapEditorToolWindowsReturn(t *testing.T) {
	for _, key := range barWindowKeys {
		for _, toggle := range []bool{false, true} {
			t.Run(key+"/"+map[bool]string{false: "switch", true: "toggle"}[toggle], func(t *testing.T) {
				e := toolWindowsEditor(t, key)
				windows := e.OpenWindows()
				selected, group := e.selected, e.groupSel
				brush := e.brush
				e.floatPlace(popList, image.Rect(24, 210, 570, 690))
				e.floatPlace(popObject, image.Rect(580, 215, 980, 720))
				e.places[popList].key = e.windowKey(popList, "")
				if s, _, ok := e.objectSpecBase(); ok {
					e.places[popObject].key = s.Title
				}
				e.OpenWindows()
				places := e.places
				next := "units"
				if key == next {
					next = "structures"
				}
				if toggle {
					e.barAct(key)
					assertBarWindowsClosed(t, e)
				} else {
					e.barAct(next)
					e.Select(0, false)
					e.floatPlace(popList, image.Rect(50, 230, 600, 680))
					e.floatPlace(popObject, image.Rect(610, 230, 1010, 700))
				}
				e.barAct(key)
				if !slices.Equal(windows, e.OpenWindows()) || e.barWindow() != key {
					t.Fatalf("return opened %v (%s), want %v", e.OpenWindows(), e.barWindow(), windows)
				}
				if selected >= 0 && e.selected != selected || group.set && (e.groupSel.player != group.player || e.groupSel.group != group.group || !e.groupSel.set) {
					t.Fatalf("lost record/group: selected %d -> %d, group %+v", selected, e.selected, e.groupSel)
				}
				if key == "objects" || key == "trees" {
					if e.brush.sel != brush.sel || e.brush.radius != brush.radius || e.brush.forest.share != brush.forest.share || !reflect.DeepEqual(e.brush.forest.ticks, brush.forest.ticks) {
						t.Fatal("lost brush choice")
					}
				}
				for _, id := range []editorPopupID{popList, popObject} {
					if id == popObject && !slices.Contains(windows, "object") {
						continue
					}
					if e.places[id] != places[id] {
						t.Errorf("popup %d standing %+v, want %+v", id, e.places[id], places[id])
					}
				}
			})
		}
	}
}

func TestMapEditorToolWindowsKeepSiblings(t *testing.T) {
	for _, key := range []string{"scripts", "players", "groups", "units", "sacks", "structures", "cellspells"} {
		t.Run(key, func(t *testing.T) {
			e := toolWindowsEditor(t, key)
			if !e.listShown {
				t.Fatal("opening a record closed its palette")
			}
			selected, group := e.selected, e.groupSel
			c, _ := categoryByKey(key)
			e.openPalette(c)
			if e.objectHidden || e.selected != selected || group.set && !e.groupSel.set {
				t.Fatal("opening the same palette closed its record")
			}
		})
	}
	t.Run("new-trigger", func(t *testing.T) {
		e := toolWindowsEditor(t, "scripts")
		e.closeObject()
		e.addNow(KindTrigger)
		if !e.listShown || e.filter != editorScriptsFilter {
			t.Fatal("creating a trigger closed Scenario")
		}
	})
	t.Run("object-add", func(t *testing.T) {
		e := toolWindowsEditor(t, "objects")
		e.placeChoose(0)
		if _, ok := e.addRunning(); !ok || e.brush.mode != brushObjects || e.listKind() != "brush" {
			t.Fatal("opening Add closed Objects")
		}
	})
	t.Run("record-and-add", func(t *testing.T) {
		e := toolWindowsEditor(t, "units")
		e.placeChoose(0)
		if _, _, ok := e.objectSpec(); !ok || !e.listShown {
			t.Fatal("opening Add hid the record or its palette")
		}
		e.barAct("structures")
		e.barAct("units")
		if _, _, ok := e.objectSpec(); !ok || !e.listShown || e.prompt.kind != promptAdd {
			t.Fatal("record, Add and palette did not return together")
		}
	})
}

func TestMapEditorToolWindowsReturnAdd(t *testing.T) {
	for _, key := range []string{"players", "units", "sacks", "structures", "cellspells", "objects"} {
		t.Run(key, func(t *testing.T) {
			e := toolWindowsEditor(t, key)
			e.closeObject()
			if key == "units" || key == "structures" || key == "objects" {
				e.placeChoose(len(e.placeShown()) - 1)
			} else {
				c, _ := categoryByKey(key)
				r, _ := addKindFor(c.filter)
				e.openAdd(r.Kind)
			}
			run, ok := e.addRunning()
			if !ok {
				t.Fatal("fixture opened no Add")
			}
			kind, repeat, brush := run.kind, run.repeat, run.brush
			choiceKey := "kind"
			if kind == KindUnit {
				choiceKey = "identity"
			}
			_, choice, _ := e.placing(choiceKey)
			e.prompt.path = "unapplied input"
			e.barAct("panel")
			e.barAct(key)
			run, ok = e.addRunning()
			if !ok || run.kind != kind || run.repeat != repeat || run.brush != brush || e.prompt.path == "unapplied input" {
				t.Fatal("Add did not return with its kind and without draft input")
			}
			if repeat {
				if _, got, ok := e.placing(choiceKey); !ok || got != choice {
					t.Fatal("placing Add did not retain the chosen grid value")
				}
			}
		})
	}
}

func TestMapEditorToolWindowsReturnReports(t *testing.T) {
	for _, key := range barWindowKeys {
		t.Run(key, func(t *testing.T) {
			e := toolWindowsEditor(t, key)
			e.openReport([]InspectionFinding{{Rule: "example", Text: "linked report"}}, "Linked", nil)
			e.barAct("structures")
			e.barAct(key)
			if !e.report.open || len(e.report.rows) != 1 || e.report.rows[0].Text != "linked report" || e.report.note != "Linked" {
				t.Fatal("linked report did not return")
			}
		})
	}
}

type toolWindowHistory struct {
	InspectionEdits
	before, after *InspectionDocument
}

func (h *toolWindowHistory) Delete(RecordKind, int) (*InspectionDocument, error) {
	return h.after, nil
}
func (h *toolWindowHistory) CanUndo() bool                      { return true }
func (h *toolWindowHistory) CanRedo() bool                      { return true }
func (h *toolWindowHistory) Undo() (*InspectionDocument, error) { return h.before, nil }
func (h *toolWindowHistory) Redo() (*InspectionDocument, error) { return h.after, nil }

func TestMapEditorToolWindowsKeepRecordIdentityAcrossDeleteAndHistory(t *testing.T) {
	for _, target := range []int{0, 1} {
		for _, history := range []string{"delete", "undo", "redo"} {
			t.Run(string(rune('0'+target))+"/"+history, func(t *testing.T) {
				e := toolWindowsEditor(t, "units")
				e.Select(target, false)
				h := &toolWindowHistory{InspectionEdits: e.doc.Edits, before: e.doc}
				e.doc.Edits = h
				e.barAct("structures")
				next := *e.doc
				next.Records, next.SourceBytes = slices.Clone(next.Records), []byte("deleted first unit")
				next.Records = slices.DeleteFunc(next.Records, func(r InspectionRecord) bool { return r.Kind == KindUnit && r.Index == 0 })
				for i := range next.Records {
					if next.Records[i].Kind == KindUnit {
						next.Records[i].Index--
					}
				}
				h.after = &next
				if _, err := e.commandDelete(RecordRef{Kind: KindUnit, Index: 0}); err != nil {
					t.Fatal(err)
				}
				if history != "delete" {
					if _, err := e.commandHistory(false); err != nil {
						t.Fatal(err)
					}
				}
				if history == "redo" {
					if _, err := e.commandHistory(true); err != nil {
						t.Fatal(err)
					}
				}
				e.barAct("units")
				_, _, open := e.objectSpec()
				want := target == 1 || history == "undo"
				if open != want || open && e.doc.Records[e.selected].Label != h.before.Records[target].Label {
					t.Fatal("delete/history restored another record or lost a surviving record")
				}
			})
		}
	}
}

func TestMapEditorToolWindowsSkipStockDeletedWithUnit(t *testing.T) {
	for _, target := range []int{0, 1} {
		t.Run(string(rune('0'+target)), func(t *testing.T) {
			e := toolWindowsEditor(t, "units")
			at := len(e.doc.Records)
			e.doc.Records = append(e.doc.Records, InspectionRecord{Kind: KindStock, Index: 0, Label: "owner 1"}, InspectionRecord{Kind: KindStock, Index: 1, Label: "owner 2"})
			e.Select(at+target, false)
			h := &toolWindowHistory{InspectionEdits: e.doc.Edits, before: e.doc}
			e.doc.Edits = h
			e.barAct("structures")
			next := *e.doc
			next.Records, next.SourceBytes = slices.Clone(next.Records), []byte("deleted unit and stock")
			next.Records = slices.DeleteFunc(next.Records, func(r InspectionRecord) bool {
				return r.Kind == KindUnit && r.Index == 0 || r.Kind == KindStock && r.Index == 0
			})
			for i := range next.Records {
				if next.Records[i].Kind == KindUnit || next.Records[i].Kind == KindStock {
					next.Records[i].Index--
				}
			}
			h.after = &next
			if _, err := e.commandDelete(RecordRef{Kind: KindUnit, Index: 0}); err != nil {
				t.Fatal(err)
			}
			e.barAct("units")
			_, _, open := e.objectSpec()
			if open != (target == 1) || open && e.doc.Records[e.selected].Label != "owner 2" {
				t.Fatal("unit deletion restored another stock or lost the surviving one")
			}
		})
	}
}

func TestMapEditorToolWindowsSkipMissingRecord(t *testing.T) {
	for _, missing := range []bool{false, true} {
		t.Run(map[bool]string{false: "reordered", true: "deleted"}[missing], func(t *testing.T) {
			e := toolWindowsEditor(t, "units")
			ref := RecordRef{Kind: e.doc.Records[e.selected].Kind, Index: e.doc.Records[e.selected].Index}
			e.barAct("structures")
			next := *e.doc
			next.Records = slices.Clone(next.Records)
			if missing {
				next.Records = slices.DeleteFunc(next.Records, func(r InspectionRecord) bool { return r.Kind == ref.Kind && r.Index == ref.Index })
			} else {
				slices.Reverse(next.Records)
			}
			if err := e.adoptEdit(&next, nil); err != nil {
				t.Fatal(err)
			}
			e.barAct("units")
			_, _, open := e.objectSpec()
			if !e.listShown || open == missing {
				t.Fatalf("palette %v record %v missing %v", e.listShown, open, missing)
			}
			if !missing && (e.selected < 0 || e.doc.Records[e.selected].Kind != ref.Kind || e.doc.Records[e.selected].Index != ref.Index) {
				t.Fatal("restored a slice position instead of the record identity")
			}
		})
	}
}

func TestMapEditorToolWindowsForgetClosedAndNewMap(t *testing.T) {
	for _, close := range []string{"escape", "close", "new-map", "open-map"} {
		t.Run(close, func(t *testing.T) {
			e := toolWindowsEditor(t, "units")
			e.barAct("structures")
			e.barAct("units")
			switch close {
			case "escape":
				_ = e.Key("escape")
			case "close":
				e.cancelPopup(popObject)
			default:
				source := "map1"
				if close == "new-map" {
					source = "new/64x64"
				}
				if err := e.Open(source); err != nil {
					t.Fatal(err)
				}
			}
			e.barAct("structures")
			e.barAct("units")
			if _, _, ok := e.objectSpec(); ok {
				t.Fatal("restored a closed window or a previous map's window")
			}
			if !e.listShown {
				t.Fatal("lost the palette")
			}
		})
	}
}

func TestMapEditorToolWindowsMenuClosesBrush(t *testing.T) {
	e := toolWindowsEditor(t, "objects")
	e.barAct("units")
	e.barAct("objects")
	e.menuAct("select")
	if e.listKind() != "" {
		t.Fatal("Select did not close the brush")
	}
	e.barAct("units")
	e.barAct("objects")
	if e.prompt.open() || e.brush.radius != 4 {
		t.Fatal("the menu restored a stale window or lost the session's brush setting")
	}
}

func TestMapEditorToolWindowsReturnSelection(t *testing.T) {
	for _, palette := range []bool{false, true} {
		for _, closed := range []bool{false, true} {
			e := toolWindowsEditor(t, "units")
			refs := []RecordRef{{Kind: KindUnit, Index: 0}, {Kind: KindUnit, Index: 1}}
			e.setSelection(refs)
			if closed {
				e.closeObject()
			}
			if !palette {
				e.closeList()
			}
			e.barAct("objects")
			e.barAct("units")
			_, _, open := e.objectSpec()
			if open == closed || !slices.Equal(e.selection.refs, refs) || !closed && e.listShown != palette {
				t.Fatal("selection window or closed palette returned with the wrong state")
			}
		}
	}
}

func TestMapEditorToolWindowsKeepLinkedReportWhenCreatingRecord(t *testing.T) {
	e := toolWindowsEditor(t, "scripts")
	e.closeObject()
	e.openReport([]InspectionFinding{{Text: "check"}}, "Report", nil)
	e.addNow(KindTrigger)
	if !e.report.open || !e.listShown {
		t.Fatal("creating a record closed its palette or linked report")
	}
}

func TestMapEditorToolWindowsDropDraft(t *testing.T) {
	e := toolWindowsEditor(t, "units")
	e.focusObjectField(0)
	e.TextInput("99")
	e.barAct("structures")
	e.barAct("units")
	if _, _, ok := e.objectSpec(); !ok || e.objectDirty() || e.obj.buf[0] != "11" {
		t.Fatal("return lost the record or restored an unapplied field")
	}
}

func TestMapEditorToolWindowsReturnRecordWithoutPalette(t *testing.T) {
	for _, key := range []string{"scripts", "players", "groups", "units", "sacks", "structures", "cellspells"} {
		t.Run(key, func(t *testing.T) {
			e := toolWindowsEditor(t, key)
			e.closeList()
			e.barAct("objects")
			e.barAct(key)
			if e.listShown {
				t.Fatal("a closed palette returned")
			}
			if _, _, ok := e.objectSpec(); !ok {
				t.Fatal("the record without its palette did not return")
			}
		})
	}
}

func TestMapEditorToolWindowsWizardAndVariable(t *testing.T) {
	for _, view := range []string{"wizard", "details", "variable"} {
		t.Run(view, func(t *testing.T) {
			e := toolWindowsEditor(t, "scripts")
			if view == "details" {
				e.ShowTriggerDetails(true)
			}
			if view == "variable" {
				e.selectVariable(1)
			}
			e.barAct("objects")
			e.barAct("scripts")
			if _, _, ok := e.objectSpec(); !ok {
				t.Fatal("the script window did not return")
			}
			if view == "variable" {
				if v, ok := e.scriptVariableRow(e.selected); !ok || v.register != 1 {
					t.Fatal("variable register did not return")
				}
			} else if e.selectedTrigger() != 0 || (e.obj.wiz == nil) != (view == "details") {
				t.Fatal("trigger or its view did not return")
			}
		})
	}
}

func TestMapEditorToolWindowsReturnMetadata(t *testing.T) {
	e, _ := mapPanelEditor(t, &mapPanelEdits{facts: playable()}, nil)
	e.openMetadata("name")
	if e.prompt.kind != promptWorkspace {
		t.Fatal("metadata fixture did not open")
	}
	e.prompt.path = "unapplied name"
	e.barAct("units")
	e.barAct("panel")
	if e.prompt.kind != promptWorkspace || e.prompt.scope != "metadata" || e.prompt.path == "unapplied name" || !e.panel {
		t.Fatal("Map and its metadata did not return without the draft")
	}
}

func TestMapEditorToolWindowsCloseReleasesOwnership(t *testing.T) {
	e, _ := barWindowEditor(t)
	e.barAct("panel")
	e.closeList()
	e.Select(0, false)
	e.barAct("panel")
	if !e.PanelOpen() {
		t.Fatal("a closed tool claimed an unrelated record and swallowed its next press")
	}
}

func TestMapEditorToolWindowsHistoryBeforeFirstSwitch(t *testing.T) {
	e, _ := barWindowEditor(t)
	h := &toolWindowHistory{InspectionEdits: e.doc.Edits, before: e.doc}
	e.doc.Edits = h
	next := *e.doc
	next.Records, next.SourceBytes = slices.Clone(next.Records), []byte("delete before first switch")
	next.Records = slices.DeleteFunc(next.Records, func(r InspectionRecord) bool { return r.Kind == KindUnit && r.Index == 0 })
	for i := range next.Records {
		if next.Records[i].Kind == KindUnit {
			next.Records[i].Index--
		}
	}
	h.after = &next
	if _, err := e.commandDelete(RecordRef{Kind: KindUnit, Index: 0}); err != nil {
		t.Fatal(err)
	}
	e.barAct("units")
	e.Select(0, false)
	e.barAct("structures")
	if _, err := e.commandHistory(false); err != nil {
		t.Fatal(err)
	}
	e.barAct("units")
	if _, _, ok := e.objectSpec(); !ok || e.doc.Records[e.selected].Label != "second" {
		t.Fatal("Undo before the first tool switch restored another record")
	}
}

func TestMapEditorToolWindowsHistoryKeepsIdenticalRecordGenerations(t *testing.T) {
	e := toolWindowsEditor(t, "units")
	e.doc.Records, e.selected = slices.Clone(e.doc.Records[:1]), 0
	e.doc.SourceBytes = []byte("one unit")
	h := &toolWindowHistory{InspectionEdits: e.doc.Edits, before: e.doc}
	e.doc.Edits = h
	original := e.doc
	e.barAct("structures")
	empty := *original
	empty.Records, empty.SourceBytes = nil, []byte("empty")
	h.after = &empty
	if _, err := e.commandDelete(RecordRef{Kind: KindUnit, Index: 0}); err != nil {
		t.Fatal(err)
	}
	readded := *original
	if err := e.adoptEdit(&readded, nil); err != nil {
		t.Fatal(err)
	}
	h.before = &empty
	if _, err := e.commandHistory(false); err != nil {
		t.Fatal(err)
	}
	h.before = original
	if _, err := e.commandHistory(false); err != nil {
		t.Fatal(err)
	}
	e.barAct("units")
	if _, _, ok := e.objectSpec(); !ok {
		t.Fatal("equal document bytes hid the original record behind its replacement's identity")
	}
}
