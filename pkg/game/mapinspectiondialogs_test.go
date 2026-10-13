package game

import (
	"bytes"
	"encoding/binary"
	"image"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"againrom/pkg/pathdialog"
	"againrom/pkg/ui"
)

func installInspectionDialog(e *ui.MapEditor, base string, result pathdialog.Result) <-chan pathdialog.Request {
	called := make(chan pathdialog.Request, 1)
	if host, ok := any(e).(interface {
		SetPathDialog(pathdialog.Func, string)
	}); ok {
		host.SetPathDialog(func(request pathdialog.Request) (pathdialog.Result, error) {
			called <- request
			return result, nil
		}, base)
	}
	return called
}

func finishInspectionDialog(t *testing.T, e *ui.MapEditor, called <-chan pathdialog.Request, kind pathdialog.Kind) pathdialog.Request {
	t.Helper()
	if host, ok := any(e).(interface{ PollPathDialog() bool }); ok {
		deadline := time.Now().Add(5 * time.Second)
		for host.PollPathDialog() {
			if time.Now().After(deadline) {
				t.Fatal("fake path dialog did not finish")
			}
			runtime.Gosched()
		}
	}
	select {
	case request := <-called:
		if request.Kind != kind {
			t.Fatalf("dialog kind = %v, want %v", request.Kind, kind)
		}
		return request
	default:
		t.Fatal("editor did not ask the path dialog")
		return pathdialog.Request{}
	}
}

func TestMapEditorDialogSaveMatchesTextPathBytes(t *testing.T) {
	b := inspectionFixtureBytes(t)
	_, textEditor, _ := openTestMapEditor(t, b)
	_, dialogEditor, source := openTestMapEditor(t, b)
	for _, e := range []*ui.MapEditor{textEditor, dialogEditor} {
		e.Select(0, true)
		dragInspectionRecord(e, image.Pt(3, 4), image.Pt(5, 6))
		if !e.Dirty() || !e.Document().Edits.CanUndo() {
			t.Fatal("fixture did not retain the authored move")
		}
	}
	dir := t.TempDir()
	textPath := filepath.Join(dir, "typed.alm")
	if err := textEditor.Key("saveas"); err != nil {
		t.Fatal(err)
	}
	textEditor.TextInput(textPath)
	if err := textEditor.Key("enter"); err != nil {
		t.Fatal(err)
	}
	dialogPath := filepath.Join(dir, "chosen.alm")
	called := installInspectionDialog(dialogEditor, dir, pathdialog.Result{Path: dialogPath})
	before := dialogEditor.Document()
	if err := dialogEditor.Key("saveas"); err != nil {
		t.Fatal(err)
	}
	request := finishInspectionDialog(t, dialogEditor, called, pathdialog.SaveFile)
	if request.DefaultName != filepath.Base(source) || request.StartDir != filepath.Dir(source) {
		t.Fatalf("save request = %+v, source %q", request, source)
	}
	typed, err := os.ReadFile(textPath)
	if err != nil {
		t.Fatal(err)
	}
	chosen, err := os.ReadFile(dialogPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(chosen, typed) || !bytes.Equal(chosen, before.SourceBytes) {
		t.Fatal("dialog save differs from the text-path write or authored bytes")
	}
	if dialogEditor.Document() != before || dialogEditor.Document().Source != dialogPath || dialogEditor.Dirty() || !before.Edits.CanUndo() {
		t.Fatal("dialog save lost the document, save checkpoint or undo history")
	}
	unchanged, err := os.ReadFile(source)
	if err != nil || !bytes.Equal(unchanged, b) {
		t.Fatalf("dialog save changed the source: %v", err)
	}
}

func TestMapEditorDialogOpenLoadsChosenBytes(t *testing.T) {
	b := inspectionFixtureBytes(t)
	chosen := bytes.Clone(b)
	binary.LittleEndian.PutUint32(chosen[inspectionSectionOffsets(t, chosen)[6]:], 9*256+128)
	path := filepath.Join(t.TempDir(), "chosen.alm")
	if err := os.WriteFile(path, chosen, 0600); err != nil {
		t.Fatal(err)
	}
	for _, route := range []string{"key", "rail"} {
		t.Run(route, func(t *testing.T) {
			_, e, source := openTestMapEditor(t, b)
			e.Select(0, true)
			before := e.Document()
			called := installInspectionDialog(e, filepath.Dir(source), pathdialog.Result{Path: path})
			if route == "key" {
				if err := e.Key("open"); err != nil {
					t.Fatal(err)
				}
			} else {
				in := ui.Input{CursorX: 1280 - 320 + 150, CursorY: 20, PrimaryDown: true}
				e.Pointer(in, time.Unix(1, 0))
				in.PrimaryDown = false
				e.Pointer(in, time.Unix(1, 0))
			}
			request := finishInspectionDialog(t, e, called, pathdialog.OpenFile)
			if request.StartDir != filepath.Dir(source) {
				t.Fatalf("open start = %q, want %q", request.StartDir, filepath.Dir(source))
			}
			d := e.Document()
			if d == before || d.Source != path || !bytes.Equal(d.SourceBytes, chosen) || d.Records[0].Cell != image.Pt(9, 4) {
				t.Fatalf("chosen map was not loaded: source %q cell %v", d.Source, d.Records[0].Cell)
			}
			if e.Dirty() || d.Edits.CanUndo() || d.Edits.CanRedo() || e.Selected() != -1 || e.Message() != "" {
				t.Fatal("chosen map did not start with the ordinary open state")
			}
		})
	}
}

func TestMapEditorDialogInstallRefusalMatchesTextPath(t *testing.T) {
	root := filepath.Join(t.TempDir(), "install")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "refused.alm")
	b := inspectionFixtureBytes(t)
	x, textEditor, _ := openTestMapEditor(t, b)
	y, dialogEditor, _ := openTestMapEditor(t, b)
	x.archives, y.archives = &Archives{Root: root}, &Archives{Root: root}
	for _, e := range []*ui.MapEditor{textEditor, dialogEditor} {
		e.Select(0, true)
		dragInspectionRecord(e, image.Pt(3, 4), image.Pt(5, 6))
	}
	if err := textEditor.Key("saveas"); err != nil {
		t.Fatal(err)
	}
	textEditor.TextInput(path)
	textErr := textEditor.Key("enter")
	if textErr == nil {
		t.Fatal("typed path into the synthetic install was accepted")
	}
	before := dialogEditor.Document()
	beforeBytes := bytes.Clone(before.SourceBytes)
	beforeSource := before.Source
	called := installInspectionDialog(dialogEditor, root, pathdialog.Result{Path: path})
	if err := dialogEditor.Key("saveas"); err != nil {
		t.Fatal(err)
	}
	finishInspectionDialog(t, dialogEditor, called, pathdialog.SaveFile)
	if dialogEditor.Message() != textEditor.Message() || dialogEditor.Message() != textErr.Error() {
		t.Fatalf("dialog refusal %q differs from typed refusal %q", dialogEditor.Message(), textErr)
	}
	if dialogEditor.Document() != before || before.Source != beforeSource || !bytes.Equal(before.SourceBytes, beforeBytes) || !dialogEditor.Dirty() || !before.Edits.CanUndo() || dialogEditor.Selected() != 0 {
		t.Fatal("refused dialog save changed document, selection or history")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("refused install path exists: %v", err)
	}
}

func TestMapEditorDialogCancelSavePreservesStateAndFiles(t *testing.T) {
	_, e, _ := openTestMapEditor(t, inspectionFixtureBytes(t))
	e.Select(0, true)
	dragInspectionRecord(e, image.Pt(3, 4), image.Pt(5, 6))
	dragInspectionRecord(e, image.Pt(5, 6), image.Pt(7, 8))
	if err := e.Key("undo"); err != nil {
		t.Fatal(err)
	}
	if err := e.Key("close"); err != nil {
		t.Fatal(err)
	}
	before, message, selected := e.Document(), e.Message(), e.Selected()
	beforeBytes, beforeSource := bytes.Clone(before.SourceBytes), before.Source
	if !e.Dirty() || !before.Edits.CanUndo() || !before.Edits.CanRedo() || message == "" {
		t.Fatal("cancel fixture lacks dirty state, both history directions or a message")
	}
	dir := t.TempDir()
	called := installInspectionDialog(e, dir, pathdialog.Result{Path: filepath.Join(dir, "cancelled.alm"), Cancelled: true})
	if err := e.Key("saveas"); err != nil {
		t.Fatal(err)
	}
	finishInspectionDialog(t, e, called, pathdialog.SaveFile)
	if e.Document() != before || before.Source != beforeSource || !bytes.Equal(before.SourceBytes, beforeBytes) || e.Selected() != selected || e.Message() != message || !e.Dirty() || !before.Edits.CanUndo() || !before.Edits.CanRedo() {
		t.Fatal("cancelled save changed document, selection, message or history")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("cancelled save wrote files: %v, %v", entries, err)
	}
}
