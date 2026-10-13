package launch

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

// mainRecipe is the process the starter built before its launches went
// through this package, recorded line for line: exec.Command of the program
// and its arguments, the program's own folder as the working directory, and
// the console hidden.
func mainRecipe(exe string, args []string) *exec.Cmd {
	cmd := exec.Command(exe, args...)
	cmd.Dir = filepath.Dir(exe)
	HideConsole(cmd)
	return cmd
}

// processOf is what a launch hands the operating system.
type processOf struct {
	Path string
	Args []string
	Dir  string
	Attr string
}

func process(cmd *exec.Cmd) processOf {
	return processOf{Path: cmd.Path, Args: cmd.Args, Dir: cmd.Dir, Attr: attrText(cmd)}
}

// attrText is the process attributes as text; none is the empty string.
func attrText(cmd *exec.Cmd) string {
	if cmd.SysProcAttr == nil {
		return ""
	}
	return fmt.Sprintf("%+v", *cmd.SysProcAttr)
}

func TestTheStarterLaunchesBuildTheRecordedProcess(t *testing.T) {
	dir := t.TempDir()
	game := filepath.Join(dir, "againrom.exe")
	editor := filepath.Join(dir, "mapedit.exe")
	for _, c := range []struct {
		exe  string
		args []string
	}{
		{game, []string{"-assets", `C:\Games\en`, "-base", "rom1-en", "-mods", "alpha", "-mods-dir", filepath.Join(dir, "mods")}},
		{game, []string{"-assets", `C:\Games\en`, "-check"}},
		{game, []string{"-version"}},
		{editor, []string{"-assets", `C:\Games\ru`}},
		{editor, []string{"-assets", `C:\Games\ru`, "-list"}},
	} {
		want := process(mainRecipe(c.exe, c.args))
		got := process(For(c.exe, c.args).Command(context.Background()))
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s %v:\n got %+v\nwant %+v", c.exe, c.args, got, want)
		}
	}
	// Loss control: a launch from another folder, or with one argument
	// changed, is a different process, so the comparison can fail.
	want := process(mainRecipe(game, []string{"-check"}))
	moved := For(game, []string{"-check"})
	moved.Dir = filepath.Join(dir, "elsewhere")
	if reflect.DeepEqual(process(moved.Command(context.Background())), want) {
		t.Fatal("a launch in another folder matched the recorded process")
	}
	if reflect.DeepEqual(process(For(game, []string{"-check=false"}).Command(context.Background())), want) {
		t.Fatal("a launch with another argument matched the recorded process")
	}
}

func TestForCopiesTheArguments(t *testing.T) {
	args := []string{"-a", "b"}
	p := For("x/game", args)
	args[0] = "-z"
	if p.Args[0] != "-a" {
		t.Fatalf("the launch shares its caller's argument slice: %v", p.Args)
	}
}

func TestBesideNamesTheProgramInTheSameFolder(t *testing.T) {
	dir := t.TempDir()
	if got := Beside(filepath.Join(dir, "mapedit.exe"), "againrom", "windows"); got != filepath.Join(dir, "againrom.exe") {
		t.Fatalf("windows: %s", got)
	}
	if got := Beside(filepath.Join(dir, "mapedit"), "againrom", "darwin"); got != filepath.Join(dir, "againrom") {
		t.Fatalf("darwin: %s", got)
	}
}
