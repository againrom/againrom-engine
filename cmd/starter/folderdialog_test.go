package main

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/pathdialog"
)

func folderAppState(a *app) string {
	state := *a
	state.deps = deps{}
	state.file = nil
	state.results = nil
	state.folderDialogResults = nil
	return fmt.Sprintf("%#v\n%s", state, a.file.Bytes())
}

func TestFolderDialogButtonRequestAndLocalization(t *testing.T) {
	for _, tc := range []struct{ language, caption, title string }{
		{"english", "Browse", "Choose the base game folder"},
		{"russian", "Обзор", "Выберите папку основной игры"},
	} {
		t.Run(tc.language, func(t *testing.T) {
			r := rigWithLanguage(t, tc.language)
			a := r.app()
			var got []pathdialog.Request
			a.deps.pathDialog = func(req pathdialog.Request) (pathdialog.Result, error) {
				got = append(got, req)
				return pathdialog.Result{Cancelled: true}, nil
			}
			found := false
			for _, it := range a.layout() {
				if it.act.kind == aChooseBase {
					found = true
					if it.kind != iButton || it.text != tc.caption || it.rect.Min.Y != 158 || it.rect.Max.X > 252 {
						t.Fatalf("folder button %+v", it)
					}
				}
			}
			if !found {
				t.Fatal("no folder button beside Add path")
			}
			do(t, a, action{kind: aChooseBase})
			want := pathdialog.Request{Kind: pathdialog.ChooseFolder, Title: tc.title, StartDir: r.root}
			if len(got) != 1 || !reflect.DeepEqual(got[0], want) {
				t.Fatalf("requests %+v, want %+v", got, want)
			}
			a.pollFolderDialog()
		})
	}
	r := newRig(t, "")
	a := r.app()
	var start string
	a.deps.pathDialog = func(req pathdialog.Request) (pathdialog.Result, error) {
		start = req.StartDir
		return pathdialog.Result{Cancelled: true}, nil
	}
	do(t, a, action{kind: aChooseBase})
	if start != r.dir {
		t.Fatalf("empty base list starts at %q, want %q", start, r.dir)
	}
	a.pollFolderDialog()
}

func TestFolderDialogSelectionMatchesTypedAdd(t *testing.T) {
	for _, name := range []string{"valid", "not-install", "missing", "file", "duplicate"} {
		t.Run(name, func(t *testing.T) {
			r := newRig(t, "[starter]\nfuture = kept\n")
			chosen := r.root
			switch name {
			case "not-install":
				chosen = t.TempDir()
			case "missing":
				chosen = filepath.Join(r.dir, "missing")
			case "file":
				chosen = filepath.Join(r.dir, "againrom.exe")
			}
			a, typed := r.app(), r.app()
			if name == "duplicate" {
				for _, app := range []*app{a, typed} {
					app.addText = chosen
					app.addBase()
				}
			}
			a.focus, typed.focus = fExtra, fExtra
			a.addText, typed.addText = "previous text", chosen
			a.deps.pathDialog = func(pathdialog.Request) (pathdialog.Result, error) {
				return pathdialog.Result{Path: chosen}, nil
			}
			do(t, a, action{kind: aChooseBase})
			if !a.pollFolderDialog() || a.folderDialogResults != nil {
				t.Fatal("dialog completion was not consumed")
			}
			do(t, typed, action{kind: aAddBase})
			if got, want := folderAppState(a), folderAppState(typed); got != want {
				t.Fatalf("selected path differs from typed Add\ngot %s\nwant %s", got, want)
			}
			before, err := os.ReadFile(r.ini)
			if err != nil {
				t.Fatal(err)
			}
			if string(before) != "[starter]\nfuture = kept\n" {
				t.Fatalf("Add changed the settings file: %q", before)
			}
			if name == "valid" {
				a.save()
				got, err := os.ReadFile(r.ini)
				if err != nil {
					t.Fatal(err)
				}
				typed.save()
				want, err := os.ReadFile(r.ini)
				if err != nil || !bytes.Equal(got, want) {
					t.Fatalf("saved settings differ: %v\ngot %s\nwant %s", err, got, want)
				}
				if restored := r.app(); !reflect.DeepEqual(restored.s, a.s) {
					t.Fatalf("folder choice did not survive settings reload: %+v", restored.s)
				}
			}
		})
	}
}

func TestFolderDialogCancelPreservesStateAndExcludesInput(t *testing.T) {
	r := newRig(t, "[starter]\nlast-base = en\n[bases]\nen = {root}\n")
	a := r.app()
	a.focus, a.addText, a.status, a.statusBad = fAddBase, r.root, "existing status", true
	a.output = []string{"existing output"}
	a.dirty = false
	r.fakes.clipboard = "clipboard input"
	var jobs []func()
	a.deps.async = func(f func()) { jobs = append(jobs, f) }
	var calls int
	a.deps.pathDialog = func(pathdialog.Request) (pathdialog.Result, error) {
		calls++
		return pathdialog.Result{Path: "ignored on cancel", Cancelled: true}, nil
	}
	before := folderAppState(a)
	settingsBefore, err := os.ReadFile(r.ini)
	if err != nil {
		t.Fatal(err)
	}
	do(t, a, action{kind: aChooseBase})
	if len(jobs) != 1 || calls != 0 || a.folderDialogResults == nil {
		t.Fatalf("dialog did not dispatch off input handling: jobs=%d calls=%d", len(jobs), calls)
	}
	do(t, a, action{kind: aChooseBase})
	a.apply(action{kind: aRemoveBase})
	a.apply(action{kind: aSave})
	a.typed("typed input")
	a.backspace()
	a.enter()
	a.escape()
	a.tab()
	a.pasteText()
	a.wheel(image.Pt(100, 100), 2)
	a.click(image.Pt(winW-1, winH-1))
	if len(jobs) != 1 || folderAppState(a) != before {
		t.Fatalf("pending dialog changed app state or accepted input:\n%s\n%s", folderAppState(a), before)
	}
	w := &window{a: a}
	if !w.blockDialogInput(false) || a.folderDialogResults == nil {
		t.Fatal("pending dialog did not keep the input loop modal")
	}
	jobs[0]()
	if !w.blockDialogInput(true) || a.folderDialogResults != nil || calls != 1 {
		t.Fatal("completion frame accepted input or did not consume cancellation")
	}
	if !w.blockDialogInput(true) || !w.blockDialogInput(false) || w.blockDialogInput(false) {
		t.Fatal("dialog dismissal was not held until a released-input frame")
	}
	if after := folderAppState(a); after != before {
		t.Fatalf("cancel changed app state:\ngot %s\nwant %s", after, before)
	}
	settingsAfter, err := os.ReadFile(r.ini)
	if err != nil || !bytes.Equal(settingsBefore, settingsAfter) {
		t.Fatalf("cancel changed settings file: %v", err)
	}
}

func TestFolderDialogFailureFallsBackToField(t *testing.T) {
	for _, language := range []string{"english", "russian"} {
		for _, unavailable := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/unavailable=%v", language, unavailable), func(t *testing.T) {
				r := rigWithLanguage(t, language)
				a := r.app()
				errDialog := errors.New("test dialog failure")
				if unavailable {
					errDialog = pathdialog.ErrUnavailable
				}
				a.addText, a.focus = "existing text", fExtra
				settingsBefore := fmt.Sprintf("%#v", a.s)
				a.deps.pathDialog = func(pathdialog.Request) (pathdialog.Result, error) {
					return pathdialog.Result{}, errDialog
				}
				do(t, a, action{kind: aChooseBase})
				if !a.pollFolderDialog() {
					t.Fatal("no dialog result")
				}
				if a.focus != fAddBase || a.addText != "existing text" || !a.statusBad || a.status != a.t(cFolderDialogFailed, errDialog) || fmt.Sprintf("%#v", a.s) != settingsBefore {
					t.Fatalf("fallback focus=%d field=%q status=%q settings=%+v", a.focus, a.addText, a.status, a.s)
				}
				a.addText = r.root
				a.enter()
				if a.statusBad || a.addText != "" || a.s.LastBase != "en" {
					t.Fatalf("typed Add failed after dialog fallback: %q", a.status)
				}
			})
		}
	}
	a := newRig(t, "").app()
	do(t, a, action{kind: aChooseBase})
	if a.focus != fAddBase || a.folderDialogResults != nil || a.status != a.t(cFolderDialogFailed, pathdialog.ErrUnavailable) {
		t.Fatalf("missing dependency did not fall back: %q", a.status)
	}
}
