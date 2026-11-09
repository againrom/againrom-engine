package editorapp

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"againrom/internal/buildinfo"
)

func keepProgram(t *testing.T) {
	t.Helper()
	saved := program
	t.Cleanup(func() { program = saved })
}

// The editor has no version of its own: -version prints the version the game
// program hands to Run, as "mapedit <game version> <first 12 characters of the
// revision>".
func TestVersionFlagPrintsOneLineAndExits(t *testing.T) {
	keepProgram(t)
	var out bytes.Buffer
	if status := Run("1.2.3", []string{"-version"}, &out, io.Discard); status != 0 {
		t.Fatalf("exit %d", status)
	}
	want := "mapedit 1.2.3 " + buildinfo.Revision() + "\n"
	if out.String() != want || strings.Count(out.String(), " ") != 2 {
		t.Fatalf("stdout %q, want %q", out.String(), want)
	}
}

// A failed start reports the error the way the program always has and exits 1.
func TestRunReportsAFailureAndExitsOne(t *testing.T) {
	keepProgram(t)
	var stderr bytes.Buffer
	if status := Run("1.2.3", []string{"-check"}, io.Discard, &stderr); status != 1 {
		t.Fatalf("exit %d", status)
	}
	if want := "mapedit: -check requires -map\n"; stderr.String() != want {
		t.Fatalf("stderr %q, want %q", stderr.String(), want)
	}
}
