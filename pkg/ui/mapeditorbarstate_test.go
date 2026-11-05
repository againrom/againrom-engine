package ui

import (
	"bytes"
	"image"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/witness"
	"againrom/pkg/words"
)

var barWindowKeys = []string{"panel", "scripts", "players", "groups", "units", "sacks", "structures", "cellspells", "objects", "trees"}

type barPanelHost struct{ *placeableHost }

func (*barPanelHost) MapFacts() InspectionMapFacts { return playable() }

func barWindowEditor(t *testing.T) (*MapEditor, *modesCalls) {
	t.Helper()
	e, calls := placeableHostEditor(t, paletteKinds, paletteObjects...)
	h := &barPanelHost{e.doc.Edits.(*placeableHost)}
	h.self, e.doc.Edits = h, h
	e.doc.Records = append(e.doc.Records, InspectionRecord{Kind: KindEnchantment, Index: 0, Label: "trap", Spatial: true, Cell: image.Pt(8, 8), Size: image.Pt(1, 1)})
	e.closeList()
	e.closeObject()
	e.places[popList].at = image.Pt(8, e.barHeight()+8)
	return e, calls
}

func barWindowPressed(t *testing.T, e *MapEditor, key string) bool {
	t.Helper()
	r, ok := e.BarRect(key)
	if !ok {
		t.Fatalf("missing bar entry %s", key)
	}
	pressed := false
	widgetRecord = func(c widgetCall) {
		if b, ok := c.state.(pushButton); ok && c.rect == r {
			pressed = b.sunk()
		}
	}
	defer func() { widgetRecord = nil }()
	e.ChromeImage()
	return pressed
}

func assertBarWindowsClosed(t *testing.T, e *MapEditor) {
	t.Helper()
	_, _, object := e.objectSpec()
	if e.listKind() != "" || e.prompt.open() || object || e.panel || e.pick != pickNone || e.brushActive() {
		t.Fatalf("windows remain: list=%q prompt=%v object=%v panel=%v pick=%q brush=%v", e.listKind(), e.prompt.kind, object, e.panel, e.pick, e.brushActive())
	}
}

func TestMapEditorBarWindowToggle(t *testing.T) {
	for _, key := range barWindowKeys {
		t.Run(key, func(t *testing.T) {
			e, calls := barWindowEditor(t)
			doc, raw := e.doc, bytes.Clone(e.doc.SourceBytes)
			undo, redo := e.doc.Edits.CanUndo(), e.doc.Edits.CanRedo()
			chromeClick(e, barPoint(t, e, key))
			if e.listKind() == "" {
				t.Fatal("first press opened no window")
			}
			if !barWindowPressed(t, e, key) {
				t.Error("open entry draws released after pointer release")
			}
			chromeClick(e, barPoint(t, e, key))
			assertBarWindowsClosed(t, e)
			if barWindowPressed(t, e, key) {
				t.Error("closed entry draws pressed")
			}
			if e.doc != doc || !bytes.Equal(e.doc.SourceBytes, raw) || len(calls.lines) != 0 || e.doc.Edits.CanUndo() != undo || e.doc.Edits.CanRedo() != redo {
				t.Fatal("toggle changed the document or history")
			}
		})
	}
}

func TestMapEditorBarWindowSwitch(t *testing.T) {
	for i, key := range barWindowKeys {
		next := barWindowKeys[(i+1)%len(barWindowKeys)]
		t.Run(key+"/"+next, func(t *testing.T) {
			e, calls := barWindowEditor(t)
			chromeClick(e, barPoint(t, e, key))
			chromeClick(e, barPoint(t, e, next))
			if e.prompt.open() || e.panel != (next == "panel") || e.brushActive() != (next == "objects" || next == "trees") {
				t.Fatal("switch retained the previous mode")
			}
			if barWindowPressed(t, e, key) || !barWindowPressed(t, e, next) {
				t.Fatal("pressed look did not move to the next entry")
			}
			chromeClick(e, barPoint(t, e, next))
			assertBarWindowsClosed(t, e)
			if len(calls.lines) != 0 {
				t.Fatal("switch edited the map", calls.lines)
			}
		})
	}
}

func TestMapEditorBarClosesPlacingWindows(t *testing.T) {
	for _, key := range []string{"units", "structures", "objects"} {
		for _, next := range []string{key, "players"} {
			t.Run(key+"/"+next, func(t *testing.T) {
				e, calls := barWindowEditor(t)
				chromeClick(e, barPoint(t, e, key))
				r, ok := e.PlaceGridCellRect(0)
				if !ok {
					t.Fatal("no placement cell")
				}
				chromeClick(e, center(r))
				if run, ok := e.addRunning(); !ok || !run.repeat {
					t.Fatal("grid opened no placing Add window")
				}
				_ = e.Key("escape")
				if !e.prompt.open() || e.pick != pickNone {
					t.Fatal("fixture did not leave Add modal after disarming its cell pick")
				}
				e.prompt.path = "uncommitted"
				chromeClick(e, barPoint(t, e, next))
				if e.prompt.open() || e.pick != pickNone || barWindowPressed(t, e, key) {
					t.Fatal("bar retained the placing Add or its entry")
				}
				if next == key {
					assertBarWindowsClosed(t, e)
				} else if !barWindowPressed(t, e, next) {
					t.Fatal("switch did not open the next entry")
				}
				if len(calls.lines) != 0 {
					t.Fatal("closing committed unfinished Add input", calls.lines)
				}
			})
		}
	}
}

func TestMapEditorBarClosesRecordAndWizard(t *testing.T) {
	for _, key := range []string{"cellspells", "scripts"} {
		for _, next := range []string{key, "units"} {
			t.Run(key+"/"+next, func(t *testing.T) {
				e, calls := barWindowEditor(t)
				if key == "scripts" {
					script := scriptsListEditor(t)
					e.doc.Script, e.doc.Records = script.doc.Script, script.doc.Records
				}
				chromeClick(e, barPoint(t, e, key))
				row := 0
				if key == "scripts" {
					_, kinds, _ := e.scenarioRows()
					for n, k := range kinds {
						if k.member && k.record == 2 {
							row = n
							break
						}
					}
				}
				e.listChoose(row, true)
				if _, _, ok := e.objectSpec(); !ok {
					t.Fatal("list did not open its record window")
				}
				chromeClick(e, barPoint(t, e, next))
				if _, _, ok := e.objectSpec(); ok {
					t.Fatal("bar left the record or wizard window open")
				}
				if next == key {
					assertBarWindowsClosed(t, e)
				}
				if len(calls.lines) != 0 {
					t.Fatal("close edited the map", calls.lines)
				}
			})
		}
	}
}

func TestMapEditorBarCloseKeepsPlacedCopies(t *testing.T) {
	for _, key := range []string{"units", "structures", "objects"} {
		for _, next := range []string{key, "players"} {
			t.Run(key+"/"+next, func(t *testing.T) {
				e, calls := barWindowEditor(t)
				chromeClick(e, barPoint(t, e, key))
				r, ok := e.PlaceGridCellRect(0)
				if !ok {
					t.Fatal("no placement cell")
				}
				chromeClick(e, center(r))
				e.addPicked(image.Pt(25, 20))
				if len(calls.lines) != 1 || !e.prompt.open() {
					t.Fatal("fixture did not place a copy and keep Add open", calls.lines)
				}
				doc, raw := e.doc, bytes.Clone(e.doc.SourceBytes)
				undo, redo := e.doc.Edits.CanUndo(), e.doc.Edits.CanRedo()
				chromeClick(e, barPoint(t, e, next))
				if e.doc != doc || !bytes.Equal(raw, e.doc.SourceBytes) || e.doc.Edits.CanUndo() != undo || e.doc.Edits.CanRedo() != redo || len(calls.lines) != 1 || e.prompt.open() {
					t.Fatal("closing Add changed a placed copy or its history", calls.lines)
				}
			})
		}
	}
}

func TestMapEditorBarTracksOtherDoors(t *testing.T) {
	for _, key := range barWindowKeys {
		for _, close := range []string{"escape", "close", "mcp"} {
			t.Run(key+"/"+close, func(t *testing.T) {
				e, _ := barWindowEditor(t)
				chromeClick(e, barPoint(t, e, key))
				switch close {
				case "escape":
					for range 4 {
						_ = e.Key("escape")
					}
				case "close":
					for range 3 {
						if e.listKind() == "" {
							break
						}
						chromeClick(e, center(listLayout(t, e).Close))
					}
				case "mcp":
					e.closeList()
				}
				if barWindowPressed(t, e, key) {
					t.Fatal("closed window retained its pressed bar entry")
				}
			})
		}
	}
	for _, key := range []string{"scripts", "players", "cellspells", "objects"} {
		e, _ := barWindowEditor(t)
		_ = e.Key(key)
		if !barWindowPressed(t, e, key) {
			t.Errorf("key %s opened a released entry", key)
		}
	}
	e, _ := barWindowEditor(t)
	e.menuAct("panel")
	if !barWindowPressed(t, e, "panel") {
		t.Error("menu opened a released Map entry")
	}
	for _, key := range tableMenus {
		chromeClick(e, barPoint(t, e, key))
		if e.menu != key {
			t.Fatal("menu did not open", key)
		}
		chromeClick(e, barPoint(t, e, key))
		if e.menu != "" {
			t.Fatal("second press did not close menu", key)
		}
	}
}

func TestMapEditorBarDropsDraftLikeEscape(t *testing.T) {
	for _, control := range []string{"escape", "units", "structures"} {
		e, calls := barWindowEditor(t)
		chromeClick(e, barPoint(t, e, "units"))
		e.Select(0, false)
		e.objectSpec()
		e.focusObjectField(0)
		e.TextInput("99")
		if !e.objectDirty() {
			t.Fatal("fixture did not keep an unapplied field edit")
		}
		if control == "escape" {
			_ = e.Key("escape")
		} else {
			chromeClick(e, barPoint(t, e, control))
		}
		e.Select(0, false)
		e.loadObject()
		if len(calls.lines) != 0 || e.objectDirty() || e.obj.buf[0] != "11" {
			t.Fatalf("%s committed or retained unfinished input: calls=%v fields=%v", control, calls.lines, e.obj.buf)
		}
	}
}

func TestMapEditorBarCloseDropsUnderlyingList(t *testing.T) {
	e, _ := barWindowEditor(t)
	e.showRecords(1)
	e.toggleBrush(brushObjects)
	chromeClick(e, barPoint(t, e, "objects"))
	assertBarWindowsClosed(t, e)
}

func TestMapEditorBarClosesLinkedReports(t *testing.T) {
	for _, link := range []panelLink{linkReport, linkGoals} {
		e, calls := barWindowEditor(t)
		e.openPanel()
		e.panelFollow(link)
		if !e.report.open || !barWindowPressed(t, e, "panel") {
			t.Fatal("Map lost its pressed entry when its linked report opened")
		}
		chromeClick(e, barPoint(t, e, "panel"))
		assertBarWindowsClosed(t, e)
		if len(calls.lines) != 0 {
			t.Fatal("closing a linked report edited the map", calls.lines)
		}
	}
	for _, next := range []string{"units", "structures"} {
		e, _ := barWindowEditor(t)
		e.showRecords(1)
		e.reportKey()
		if !barWindowPressed(t, e, "units") {
			t.Fatal("report lost the palette that opened it")
		}
		chromeClick(e, barPoint(t, e, next))
		if e.report.open {
			t.Fatal("bar left the palette's report open")
		}
		if next == "units" {
			assertBarWindowsClosed(t, e)
		}
	}
}

func TestMapEditorBarPressedFrames(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			e, _ := barWindowEditor(t)
			e.SetWords(words.For(lang))
			e.Layout(1920, 1080)
			r, _ := e.BarRect("units")
			rest := e.ChromeImage().RGBAAt(r.Min.X+2, r.Min.Y)
			chromeClick(e, barPoint(t, e, "units"))
			e.pointerOK = false
			pic := e.ChromeImage()
			if !barWindowPressed(t, e, "units") || pic.RGBAAt(r.Min.X+2, r.Min.Y) == rest {
				t.Error("open Units entry has no persistent pressed bevel")
			}
			if dir := os.Getenv("EDITOR_WINDOW_PICS"); dir != "" {
				witness.PNG(t, filepath.Join(dir, "bar-pressed-"+lang+".png"), pic)
			}
		})
	}
}
