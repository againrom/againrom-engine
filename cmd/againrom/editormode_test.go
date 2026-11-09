package main

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/buildinfo"
)

// Everything after --editor is the editor's, including flags the game does not
// know and flags it knows by another meaning; the game's parser reads none.
func TestEditorModeHandsTheArgumentsAfterTheFlagToTheEditor(t *testing.T) {
	for _, flagName := range []string{"--editor", "-editor"} {
		args := []string{"-assets", "x", "-list", "-check", "not-a-game-flag", "-mcp", "stdio"}
		o, err := parse(append([]string{flagName}, args...))
		if err != nil {
			t.Fatalf("%s: %v", flagName, err)
		}
		if !o.editor || !reflect.DeepEqual(o.editorArgs, args) {
			t.Fatalf("%s: editor=%v args=%q", flagName, o.editor, o.editorArgs)
		}
		if o.assets != "" || o.check {
			t.Fatalf("%s: the game's parser read the editor's arguments: %+v", flagName, o)
		}
	}
}

func TestEditorModeWithNoArgumentsIsTheEditor(t *testing.T) {
	o, err := parse([]string{"--editor"})
	if err != nil || !o.editor || len(o.editorArgs) != 0 {
		t.Fatalf("%v %v %q", err, o.editor, o.editorArgs)
	}
}

// A game flag before --editor would be dropped, so it is refused.
func TestEditorModeRefusesAGameFlagBeforeIt(t *testing.T) {
	var stderr bytes.Buffer
	if status := run([]string{"-window", "--editor", "-list"}, func(string) string { return "" }, &bytes.Buffer{}, &stderr); status != 2 {
		t.Fatalf("exit %d", status)
	}
	if !strings.Contains(stderr.String(), "--editor must come first") {
		t.Fatalf("stderr %q", stderr.String())
	}
}

// The flag is read by the game's parser, so a game flag's value may be spelled
// like it, and a false switch leaves the game.
func TestEditorModeIsAFlagOfTheGamesParser(t *testing.T) {
	o, err := parse([]string{"-saves", "--editor"})
	if err != nil || o.editor || o.saves != "--editor" {
		t.Fatalf("flag value: %v editor=%v saves=%q", err, o.editor, o.saves)
	}
	o, err = parse([]string{"--editor=false", "-window"})
	if err != nil || o.editor || !o.window {
		t.Fatalf("false switch: %v editor=%v window=%v", err, o.editor, o.window)
	}
	if o, err := parse([]string{"-window"}); err != nil || o.editor {
		t.Fatalf("no flag: %v editor=%v", err, o.editor)
	}
}

// Only a start whose one argument is a map path is the editor; every other
// start keeps the game.
func TestEditorModeOpensALoneMapPath(t *testing.T) {
	spaced := `C:\Program Files\Old Maps\first map.Alm`
	for _, row := range []struct {
		name       string
		args       []string
		wantEditor bool
		wantArgs   []string
		wantWindow bool
	}{
		{name: "bare map", args: []string{"map.alm"}, wantEditor: true, wantArgs: []string{"map.alm"}},
		{name: "upper case", args: []string{"MAP.ALM"}, wantEditor: true, wantArgs: []string{"MAP.ALM"}},
		{name: "spaces and folders", args: []string{spaced}, wantEditor: true, wantArgs: []string{spaced}},
		{name: "two positionals", args: []string{"one.alm", "two.alm"}},
		{name: "another extension", args: []string{"map.txt"}},
		{name: "a folder named like a map", args: []string{"maps.alm/notes.txt"}},
		{name: "a flag and a map", args: []string{"-window", "map.alm"}, wantWindow: true},
		{name: "the end of flags and a map", args: []string{"--", "map.alm"}},
	} {
		t.Run(row.name, func(t *testing.T) {
			o, err := parse(row.args)
			if err != nil {
				t.Fatal(err)
			}
			if o.editor != row.wantEditor || !reflect.DeepEqual(o.editorArgs, row.wantArgs) || o.window != row.wantWindow {
				t.Fatalf("editor=%v args=%q window=%v, want editor=%v args=%q window=%v",
					o.editor, o.editorArgs, o.window, row.wantEditor, row.wantArgs, row.wantWindow)
			}
		})
	}
}

// The editor's version line carries the game's version from the game's own
// VERSION file; the game's version line is unchanged.
func TestEditorModeVersionLineIsTheGamesVersion(t *testing.T) {
	var stdout bytes.Buffer
	if status := run([]string{"--editor", "-version"}, func(string) string { return "" }, &stdout, &bytes.Buffer{}); status != 0 {
		t.Fatalf("exit %d", status)
	}
	want := "mapedit " + program.Version() + " " + buildinfo.Revision() + "\n"
	if stdout.String() != want {
		t.Fatalf("stdout %q, want %q", stdout.String(), want)
	}
	stdout.Reset()
	if status := run([]string{"-version"}, func(string) string { return "" }, &stdout, &bytes.Buffer{}); status != 0 {
		t.Fatalf("exit %d", status)
	}
	if want := versionLine() + "\n"; stdout.String() != want || !strings.HasPrefix(want, "againrom ") {
		t.Fatalf("game version line %q, want %q", stdout.String(), want)
	}
}

// A failure of the editor is the editor's message and exit status.
func TestEditorModeFailureIsTheEditorsReport(t *testing.T) {
	var stderr bytes.Buffer
	if status := run([]string{"--editor", "-check"}, func(string) string { return "" }, &bytes.Buffer{}, &stderr); status != 1 {
		t.Fatalf("exit %d", status)
	}
	if want := "mapedit: -check requires -map\n"; stderr.String() != want {
		t.Fatalf("stderr %q, want %q", stderr.String(), want)
	}
}
