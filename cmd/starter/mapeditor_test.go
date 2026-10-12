package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/game"
)

func writeEditor(t *testing.T, r *rig) string {
	t.Helper()
	exe := filepath.Join(r.dir, "mapedit.exe")
	if err := os.WriteFile(exe, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	return exe
}

func editorCaption(t *testing.T, a *app) string {
	t.Helper()
	for _, it := range a.layout() {
		if it.kind == iButton && it.act.kind == aMapEditor {
			return it.text
		}
	}
	t.Fatal("no map editor button")
	return ""
}

func TestMapEditorChecksTheBaseThenStartsBesideTheStarter(t *testing.T) {
	r := newRig(t, "[starter]\nlast-base = en\n[bases]\nen = {root}\n[mods]\nenabled = alpha\n[options]\nsound = off\nextra = -skill axe\n")
	writeModFolder(t, filepath.Join(r.dir, "mods"), "alpha", goodMod("alpha"))
	exe := writeEditor(t, r)
	a := r.app()
	r.fakes.ran = nil
	do(t, a, action{kind: aMapEditor})
	if len(r.fakes.started) != 0 || a.status != "checking mapedit.exe on en ..." {
		t.Fatalf("started before the check: %q %q", r.fakes.started, a.status)
	}
	a.poll()
	wantCheck := []string{exe, "-assets", r.root, "-list"}
	if len(r.fakes.ran) != 1 || !reflect.DeepEqual(r.fakes.ran[0], wantCheck) {
		t.Fatalf("ran %q\nwant %q", r.fakes.ran, wantCheck)
	}
	want := []string{exe, "-assets", r.root}
	if len(r.fakes.started) != 1 || !reflect.DeepEqual(r.fakes.started[0], want) {
		t.Fatalf("started %q\nwant %q", r.fakes.started, want)
	}
	if a.statusBad || a.status != "started mapedit.exe" || a.closing {
		t.Fatalf("status %q bad=%v closing=%v", a.status, a.statusBad, a.closing)
	}
	// Close after Play belongs to Play.
	a.s.CloseOnPlay = true
	do(t, a, action{kind: aMapEditor})
	a.poll()
	if a.closing || len(r.fakes.started) != 2 {
		t.Fatalf("closing=%v started %d", a.closing, len(r.fakes.started))
	}
	r.fakes.startErr = errors.New("boom")
	do(t, a, action{kind: aMapEditor})
	a.poll()
	if !a.statusBad || !strings.Contains(a.status, "boom") {
		t.Fatalf("%q", a.status)
	}
}

// A base the editor cannot load, such as a second-game install, fails the map
// list check: the starter shows the editor's own error and starts nothing.
func TestMapEditorCheckFailureShowsTheEditorsErrorAndStartsNothing(t *testing.T) {
	r := newRig(t, "[starter]\nlast-base = en\n[bases]\nen = {root}\n")
	writeEditor(t, r)
	a := r.app()
	r.fakes.runErr = errors.New("exit status 1")
	r.fakes.runOut["-list"] = "mapedit: world/data/data.bin: cannot read\r\n"
	do(t, a, action{kind: aMapEditor})
	a.poll()
	if len(r.fakes.started) != 0 || !a.statusBad || a.status != "the map editor cannot open this base: exit status 1" {
		t.Fatalf("%v %q", r.fakes.started, a.status)
	}
	if !reflect.DeepEqual(a.output, []string{"mapedit: world/data/data.bin: cannot read"}) {
		t.Fatalf("output %q", a.output)
	}
}

func TestMapEditorRefusalsStartNothing(t *testing.T) {
	r := newRig(t, "")
	a := r.app()
	writeEditor(t, r)
	r.fakes.ran = nil
	do(t, a, action{kind: aMapEditor})
	a.poll()
	if len(r.fakes.started) != 0 || len(r.fakes.ran) != 0 || !a.statusBad || !strings.Contains(a.status, "no base game selected") {
		t.Fatalf("%v %v %q", r.fakes.started, r.fakes.ran, a.status)
	}
	bad := filepath.Join(r.dir, "bad")
	os.MkdirAll(bad, 0o755)
	a.s.addBase(bad)
	a.inspectBase(bad)
	a.s.LastBase = "bad"
	do(t, a, action{kind: aMapEditor})
	a.poll()
	if len(r.fakes.started) != 0 || len(r.fakes.ran) != 0 || !strings.Contains(a.status, "not an install") {
		t.Fatalf("%v %v %q", r.fakes.started, r.fakes.ran, a.status)
	}

	r2 := newRig(t, "[starter]\nlast-base = en\n[bases]\nen = {root}\n")
	a2 := r2.app()
	r2.fakes.ran = nil
	do(t, a2, action{kind: aMapEditor})
	a2.poll()
	missing := filepath.Join(r2.dir, "mapedit.exe")
	if len(r2.fakes.started) != 0 || len(r2.fakes.ran) != 0 || !a2.statusBad || a2.status != "cannot open the map editor: map editor not found: "+missing {
		t.Fatalf("%v %v %q", r2.fakes.started, r2.fakes.ran, a2.status)
	}
}

func TestMapEditorCaptionFollowsTheBaseLanguage(t *testing.T) {
	r := newRig(t, "[starter]\nlast-base = en\n[bases]\nen = {root}\n")
	if got := editorCaption(t, r.app()); got != "Map editor" {
		t.Fatalf("english base: %q", got)
	}
	r.fakes.validRoots[r.root] = game.InstallInfo{Language: "russian"}
	if got := editorCaption(t, r.app()); got != "Редактор карт" {
		t.Fatalf("russian base: %q", got)
	}
	if got := editorCaption(t, newRig(t, "").app()); got != "Map editor" {
		t.Fatalf("no base: %q", got)
	}
}

func TestEditorExecutableIsBesideTheStarter(t *testing.T) {
	dir := filepath.Join("games", "againrom")
	if got := editorExecutable(filepath.Join(dir, "starter.exe"), "windows"); got != filepath.Join(dir, "mapedit.exe") {
		t.Fatal(got)
	}
	if got := editorExecutable(filepath.Join(dir, "starter"), "darwin"); got != filepath.Join(dir, "mapedit") {
		t.Fatal(got)
	}
}
