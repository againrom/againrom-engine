package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"againrom/internal/buildinfo"
)

func TestVersionIsTheGameVersion(t *testing.T) {
	data, err := os.ReadFile("../againrom/VERSION")
	if err != nil {
		t.Fatal(err)
	}
	game, err := buildinfo.ParseVersion(string(data))
	if err != nil {
		t.Fatal(err)
	}
	if got := program.Version(); got != game {
		t.Fatalf("mapedit reports %q, the game is %q; run go run ./internal/versionres/cmd/versionres", got, game)
	}
}

// The package checks compare this line with "mapedit <game version> <first 12
// characters of the revision>".
func TestVersionFlagPrintsOneLineAndExits(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"-version"}, &out); err != nil {
		t.Fatal(err)
	}
	want := "mapedit " + program.Version() + " " + buildinfo.Revision() + "\n"
	if out.String() != want || strings.Count(out.String(), " ") != 2 {
		t.Fatalf("stdout %q, want %q", out.String(), want)
	}
}
