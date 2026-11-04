package ui

import (
	"bytes"
	"fmt"
	"image"
	"reflect"
	"runtime/debug"
	"strings"
	"testing"

	"againrom/pkg/pathdialog"
	"againrom/pkg/words"
	"github.com/hajimehoshi/ebiten/v2"
)

type panelCloseEdits struct {
	*mapPanelEdits
	next func() (*InspectionDocument, error)
}

func (p *panelCloseEdits) CanUndo() bool                      { return true }
func (p *panelCloseEdits) Undo() (*InspectionDocument, error) { return p.next() }
func (p *panelCloseEdits) SetWorkspaceField(scope, key, value string) (*InspectionDocument, error) {
	_, err := p.mapPanelEdits.SetWorkspaceField(scope, key, value)
	p.facts.Mode = uint32(p.mode)
	if err != nil {
		return nil, err
	}
	return p.next()
}
func (p *panelCloseEdits) Fields(kind RecordKind, index int) []InspectionField {
	if kind == KindPlayer {
		return []InspectionField{{Key: "name", Label: "Name", Text: true, TextValue: fmt.Sprintf("Player %d", index+1)},
			{Key: "relation", Label: "Relation", Toward: 2 - index, Value: 1}}
	}
	return nil
}

type panelCloseState struct {
	mode             editorMode
	selected, filter int
	list             bool
	message          string
}

func mapPanelCloseStateOf(e *MapEditor) panelCloseState {
	m := e.editorMode
	m.panel = false
	return panelCloseState{m, e.selected, e.filter, e.listShown, e.message}
}

func closeMapPanelWindow(t *testing.T, e *MapEditor, control string) {
	t.Helper()
	if !e.PanelOpen() {
		t.Fatalf("Map window is not shown: %s", e.listKind())
	}
	s, _ := e.listSpec()
	switch control {
	case "close":
		chromeClick(e, center(layoutPopup(s, e.font, e.bodyFont()).Close))
	case "escape":
		for range 4 {
			if !e.PanelOpen() {
				break
			}
			_ = e.Key("escape")
		}
	case "cancel":
		e.cancelPopup(popList)
	case "close-list":
		e.closeList()
	}
}

func TestMapPanelCloseStates(t *testing.T) {
	states := []string{"map-open", "bar-idle", "bar-records", "bar-catalog", "bar-report", "bar-brush", "repeat-open", "owner-frame",
		"prompt-open", "prompt-closed", "notice-open", "notice-closed", "new-mode-open", "new-mode-closed", "path-open", "path-closed",
		"report-open", "report-closed", "test-play-returned", "undo", "new-map", "opened-map", "mode-change"}
	for _, link := range []string{"size", "name", "description", "players", "mode", "start", "victory", "defeat", "report", "test-play"} {
		for _, end := range []string{"open", "closed"} {
			states = append(states, "line-"+link+"-"+end)
		}
	}
	for _, lang := range []string{"en", "russian"} {
		for _, state := range states {
			for _, control := range []string{"close", "escape", "cancel", "close-list"} {
				t.Run(lang+"/"+state+"/"+control, func(t *testing.T) {
					defer func() {
						if p := recover(); p != nil {
							t.Errorf("panic: %v\n%s", p, debug.Stack())
						}
					}()
					p := &mapPanelEdits{facts: playable(), mode: 1, size: 64}
					e, _ := mapPanelEditor(t, p, []EditorMap{{Key: "next", Label: "Next"}})
					h := &panelCloseEdits{mapPanelEdits: p}
					open := e.open
					e.open = func(source string) (*InspectionDocument, error) {
						d, err := open(source)
						if err == nil {
							d.Edits = h
							d.Records = []InspectionRecord{{Kind: KindPlayer, Index: 0, Label: "Self"}, {Kind: KindPlayer, Index: 1, Label: "Enemy"}}
						}
						return d, err
					}
					h.next = func() (*InspectionDocument, error) { return e.open(e.doc.Source) }
					e.SetWords(words.For(words.Language(lang)))
					if err := e.Open("new/64x64"); err != nil {
						t.Fatal(err)
					}
					base := mapPanelCloseStateOf(e)
					want := base
					blocked := func() {
						t.Helper()
						before := mapPanelCloseStateOf(e)
						s, _ := e.listSpec()
						chromeClick(e, center(layoutPopup(s, e.font, e.bodyFont()).Close))
						if !e.PanelOpen() || !reflect.DeepEqual(before, mapPanelCloseStateOf(e)) {
							t.Fatal("Map close changed a modal place")
						}
					}
					finishPrompt := func(leftOpen bool) {
						if leftOpen {
							blocked()
						}
						e.cancelPopup()
					}
					switch state {
					case "bar-idle", "bar-records", "bar-catalog", "bar-report", "bar-brush", "repeat-open":
						e.cancelPopup(popList)
						switch state {
						case "bar-records":
							e.showRecords(editorPlayersFilter)
						case "bar-catalog":
							e.enterCatalog()
						case "bar-report":
							e.reportKey()
						case "bar-brush":
							e.toggleBrush(brushTerrain)
						}
						want = mapPanelCloseStateOf(e)
						chromeClick(e, barPoint(t, e, "panel"))
						if state == "repeat-open" {
							e.openPanel()
							e.openPanel()
						}
					case "owner-frame":
						p.facts = InspectionMapFacts{Name: "Multiplayer Map", Mode: MapModeSingle, Players: 2, Sides: 2, Drops: []image.Point{{X: 32, Y: 32}}}
						p.rows = []InspectionFinding{{Rule: RuleDropCell, Cell: image.Pt(32, 32), HasCell: true, Text: "blocked start"}, {Rule: RuleVictory, Text: "no goals"}}
						lines := e.PanelLines()
						if len(lines) != 10 || !strings.Contains(lines[0], "64 x 64") || !strings.Contains(lines[1], "Multiplayer Map") || !strings.Contains(lines[5], "32,32") {
							t.Fatalf("owner state: %v", lines)
						}
					case "prompt-open", "prompt-closed":
						e.openMetadata("name")
						finishPrompt(state == "prompt-open")
					case "notice-open", "notice-closed":
						p.facts, p.rows = endless()
						if !e.askNotice(noticeTestPlay) {
							t.Fatal("notice missing")
						}
						finishPrompt(state == "notice-open")
					case "new-mode-open", "new-mode-closed":
						e.askNewMode("new/64x64")
						finishPrompt(state == "new-mode-open")
					case "path-open", "path-closed":
						reply := make(chan pathdialog.Result, 1)
						e.SetPathDialog(func(pathdialog.Request) (pathdialog.Result, error) { return <-reply, nil }, "")
						e.askPath(pathdialog.OpenFile)
						if state == "path-open" {
							blocked()
							_ = e.Key("escape")
							if e.pathDialog.pending == nil {
								t.Fatal("dialog did not block Escape")
							}
						}
						reply <- pathdialog.Result{Cancelled: true}
						finishEditorDialog(t, e)
					case "report-open", "report-closed":
						e.reportKey()
						if state == "report-open" {
							want = mapPanelCloseStateOf(e)
							e.openPanel()
						} else {
							e.cancelPopup(popList)
						}
					case "test-play-returned":
						_ = e.keyTestPlay("")
						if p.plays != 1 {
							t.Fatal("test play did not return")
						}
						blocked()
						e.cancelPopup(popPrompt)
					case "undo":
						if _, err := e.commandHistory(false); err != nil {
							t.Fatal(err)
						}
						want = mapPanelCloseStateOf(e)
					case "new-map":
						e.askNewMode("new/64x64")
						if err := e.newModeOpen(false); err != nil {
							t.Fatal(err)
						}
						want = mapPanelCloseStateOf(e)
					case "opened-map":
						if err := e.Open("next"); err != nil {
							t.Fatal(err)
						}
						want = mapPanelCloseStateOf(e)
					case "mode-change":
						e.openMetadata(ModeFieldKey)
						if err := e.setField("4"); err != nil {
							t.Fatal(err)
						}
						want = mapPanelCloseStateOf(e)
					default:
						if strings.HasPrefix(state, "line-") {
							links := map[string]panelLink{"size": linkNone, "name": linkName, "description": linkDescription, "players": linkPlayers, "mode": linkMode, "start": linkStart, "victory": linkGoals, "defeat": linkGoals, "report": linkReport, "test-play": linkTestPlay}
							name := strings.TrimSuffix(strings.TrimSuffix(strings.TrimPrefix(state, "line-"), "-open"), "-closed")
							leftOpen := strings.HasSuffix(state, "-open")
							e.panelFollow(links[name])
							switch {
							case e.prompt.open():
								finishPrompt(leftOpen)
							case e.message != "":
								if leftOpen {
									blocked()
								}
								e.cancelPopup(popPrompt)
							case e.listKind() != "panel":
								if leftOpen {
									want = mapPanelCloseStateOf(e)
									e.openPanel()
								} else {
									e.cancelPopup(popList)
								}
							case e.pick != pickNone:
								if !leftOpen || control == "escape" {
									_ = e.Key("escape")
								} else {
									want = mapPanelCloseStateOf(e)
								}
							}
						}
					}
					if control == "close-list" {
						want = mapPanelCloseStateOf(e)
					}
					doc, raw := e.doc, bytes.Clone(e.doc.SourceBytes)
					undo, redo := h.CanUndo(), h.CanRedo()
					closeMapPanelWindow(t, e, control)
					if e.PanelOpen() || !reflect.DeepEqual(want, mapPanelCloseStateOf(e)) {
						t.Fatalf("close lost the previous or linked place: got %+v want %+v", mapPanelCloseStateOf(e), want)
					}
					if e.doc != doc || !bytes.Equal(e.doc.SourceBytes, raw) || h.CanUndo() != undo || h.CanRedo() != redo {
						t.Fatal("close changed document or history")
					}
					_ = e.ChromeImage()
					e.openPanel()
					e.cancelPopup(popList)
					_ = e.ChromeImage()
				})
			}
		}
	}
}

func TestMapPanelCloseAfterWindowResize(t *testing.T) {
	for _, lang := range []string{"en", "russian"} {
		for _, roundTrip := range []bool{false, true} {
			for _, control := range []string{"close", "escape", "cancel"} {
				t.Run(fmt.Sprintf("%s/round-trip-%v/%s", lang, roundTrip, control), func(t *testing.T) {
					defer func() {
						if p := recover(); p != nil {
							t.Errorf("panic: %v\n%s", p, debug.Stack())
						}
					}()
					e, _ := mapPanelEditor(t, &mapPanelEdits{facts: playable()}, nil)
					e.cancelPopup(popList)
					e.SetWords(words.For(words.Language(lang)))
					draw := func(w, h int) {
						e.Layout(w, h)
						screen := ebiten.NewImage(w, h)
						defer screen.Dispose()
						e.drawChrome(screen)
					}
					t.Cleanup(func() {
						if e.chromeImage != nil {
							e.chromeImage.Dispose()
						}
					})
					draw(640, 480)
					e.openPanel()
					draw(800, 600)
					if roundTrip {
						draw(640, 480)
					}
					live := e.chromeImage
					viewW, viewH := e.doc.Viewer.cam.ViewW, e.doc.Viewer.cam.ViewH
					switch control {
					case "close":
						s, _ := e.listSpec()
						chromeClick(e, center(layoutPopup(s, e.font, e.bodyFont()).Close))
					case "escape":
						_ = e.Key("escape")
					case "cancel":
						e.cancelPopup(popList)
					}
					if e.PanelOpen() {
						t.Fatal("Map window stayed open")
					}
					if e.doc.Viewer.cam.ViewW != viewW || e.doc.Viewer.cam.ViewH != viewH {
						t.Errorf("Cancel restored an obsolete viewport: got %dx%d want %dx%d", e.doc.Viewer.cam.ViewW, e.doc.Viewer.cam.ViewH, viewW, viewH)
					}
					screen := ebiten.NewImage(e.window.X, e.window.Y)
					defer screen.Dispose()
					e.drawChrome(screen)
					if e.chromeImage != live {
						t.Fatalf("Cancel replaced the live chrome image: %p -> %p", live, e.chromeImage)
					}
				})
			}
		}
	}
}
