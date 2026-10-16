package ui

import (
	"errors"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"againrom/pkg/pathdialog"
)

func bindEditorDialog(e *MapEditor, ask pathdialog.Func, base string) {
	if host, ok := any(e).(interface{ SetPathDialog(pathdialog.Func, string) }); ok {
		host.SetPathDialog(ask, base)
	}
}

func finishEditorDialog(t *testing.T, e *MapEditor) {
	t.Helper()
	if host, ok := any(e).(interface{ PollPathDialog() bool }); ok {
		limit := time.Now().Add(5 * time.Second)
		for host.PollPathDialog() {
			if time.Now().After(limit) {
				t.Fatal("dialog worker did not complete")
			}
			runtime.Gosched()
		}
	}
}

func TestEditorOpenDialogKeyAndMenu(t *testing.T) {
	for _, door := range []string{"key", "menu"} {
		t.Run(door, func(t *testing.T) {
			e := editor1092(t)
			base := t.TempDir()
			current := filepath.Join(base, "maps", "current.alm")
			e.doc.Source = current
			chosen := filepath.Join(base, "chosen.alm")
			requests := make(chan pathdialog.Request, 1)
			bindEditorDialog(e, func(r pathdialog.Request) (pathdialog.Result, error) {
				requests <- r
				return pathdialog.Result{Path: chosen}, nil
			}, base)
			if door == "key" {
				_ = e.Key("open")
			} else {
				chromeClick(e, barPoint(t, e, "map"))
				chromeClick(e, menuPoint(t, e, 2))
			}
			finishEditorDialog(t, e)
			select {
			case r := <-requests:
				want := []pathdialog.Filter{{Name: "ALM maps", Extensions: []string{"alm"}}, {Name: "All files", Extensions: []string{"*"}}}
				if r.Kind != pathdialog.OpenFile || r.StartDir != filepath.Dir(current) || !reflect.DeepEqual(r.Filters, want) {
					t.Errorf("request = %+v", r)
				}
			default:
				t.Error("Open did not ask the injected system dialog")
			}
			if e.Document().Source != chosen || e.prompt.open() {
				t.Fatalf("chosen map not opened: %q, prompt=%+v", e.Document().Source, e.prompt)
			}
		})
	}
}

func TestEditorDialogStartFoldersAndSaveName(t *testing.T) {
	base := t.TempDir()
	for _, tc := range []struct{ source, folder, name string }{
		{"", base, "new-map.alm"},
		{"scenario/mission1.alm", base, "mission1.alm"},
		{"new/32x32", base, "new-map.alm"},
		{"loose/nested/author.alm", filepath.Join(base, "nested"), "author.alm"},
		{filepath.Join(base, "authored", "карта.alm"), filepath.Join(base, "authored"), "карта.alm"},
	} {
		for _, key := range []string{"open", "saveas"} {
			t.Run(tc.source+"/"+key, func(t *testing.T) {
				e := editor1092(t)
				e.doc.Source = tc.source
				e.doc.Edits = &editorMoveProbe{doc: e.doc}
				requests := make(chan pathdialog.Request, 1)
				bindEditorDialog(e, func(r pathdialog.Request) (pathdialog.Result, error) {
					requests <- r
					return pathdialog.Result{Cancelled: true}, nil
				}, base)
				_ = e.Key(key)
				finishEditorDialog(t, e)
				select {
				case r := <-requests:
					if r.StartDir != tc.folder || key == "saveas" && (r.DefaultName != tc.name || r.Kind != pathdialog.SaveFile || r.Filters[0].Extensions[0] != "alm") {
						t.Errorf("request = %+v, want folder %q name %q", r, tc.folder, tc.name)
					}
				default:
					t.Fatal("no system dialog request")
				}
			})
		}
	}
	t.Run("no document", func(t *testing.T) {
		e := NewMapEditor(nil, nil, nil)
		requests := make(chan pathdialog.Request, 1)
		bindEditorDialog(e, func(r pathdialog.Request) (pathdialog.Result, error) {
			requests <- r
			return pathdialog.Result{Cancelled: true}, nil
		}, base)
		_ = e.Key("open")
		finishEditorDialog(t, e)
		select {
		case r := <-requests:
			if r.StartDir != base {
				t.Fatalf("start folder %q, want %q", r.StartDir, base)
			}
		default:
			t.Fatal("empty editor did not ask for a file")
		}
	})
}

func TestEditorDialogCancelPreservesStateAndHoldsInput(t *testing.T) {
	for _, key := range []string{"open", "saveas"} {
		t.Run(key, func(t *testing.T) {
			e := editor1092(t)
			p := &editorMoveProbe{doc: e.doc, dirty: true}
			e.doc.Edits = p
			e.Select(20, true)
			e.message = "prior message"
			e.armPick(pickClone)
			e.message = "prior message"
			doc, mode, selected, message := e.doc, e.editorMode, e.selected, e.message
			camera := *e.doc.Viewer.cam
			release := make(chan struct{})
			bindEditorDialog(e, func(pathdialog.Request) (pathdialog.Result, error) {
				<-release
				return pathdialog.Result{Cancelled: true}, nil
			}, t.TempDir())
			_ = e.Key(key)
			_ = e.Key("undo")
			_ = e.Key("escape")
			_ = e.Key("open")
			e.TextInput("ignored")
			e.Pointer(Input{CursorX: 50, CursorY: 50, WheelY: 1, PrimaryDown: true}, time.Time{})
			close(release)
			finishEditorDialog(t, e)
			if e.doc != doc || !reflect.DeepEqual(e.editorMode, mode) || e.selected != selected || e.message != message || p.undo != 0 || !p.dirty || !reflect.DeepEqual(camera, *e.doc.Viewer.cam) {
				t.Fatalf("cancel or modal input changed document, selection, history, mode, camera or message: mode=%+v message=%q undo=%d", e.editorMode, e.message, p.undo)
			}
		})
	}
}

func TestEditorDialogFailureKeepsTypedRoute(t *testing.T) {
	for _, failure := range []error{pathdialog.ErrUnavailable, errors.New("native service failed")} {
		for _, key := range []string{"open", "saveas"} {
			t.Run(key+"/"+failure.Error(), func(t *testing.T) {
				e := editor1092(t)
				e.doc.Edits = &editorMoveProbe{doc: e.doc}
				bindEditorDialog(e, func(pathdialog.Request) (pathdialog.Result, error) {
					return pathdialog.Result{}, failure
				}, t.TempDir())
				_ = e.Key(key)
				finishEditorDialog(t, e)
				if !e.prompt.open() || !strings.Contains(e.message, failure.Error()) || !strings.Contains(e.message, "Type a path") || (e.prompt.kind == promptSave) != (key == "saveas") {
					t.Fatalf("fallback not visible: prompt=%+v message=%q", e.prompt, e.message)
				}
				e.TextInput("typed.alm")
				err := e.Key("enter")
				if key == "open" && (err != nil || e.doc.Source != "typed.alm") || key == "saveas" && (err == nil || !strings.Contains(err.Error(), "test writer refused")) {
					t.Fatalf("typed route not reached: %v, %q", err, e.doc.Source)
				}
			})
		}
	}
}

func TestEditorModSaveBypassesPathDialog(t *testing.T) {
	e, calls := newModesEditor(t, "mod")
	bindEditorDialog(e, func(pathdialog.Request) (pathdialog.Result, error) {
		panic("mod save asked for a system path")
	}, t.TempDir())
	if err := e.Key("saveas"); err != nil || !reflect.DeepEqual(calls.lines, []string{"SaveIntoMod"}) {
		t.Fatalf("mod save route changed: %v, %v", err, calls.lines)
	}
}
