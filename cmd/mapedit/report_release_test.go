package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/game"
	"againrom/pkg/ui"
)

// The command line prints the findings the window lists, for every installed map.
func TestReleaseCheckPrintsTheFindingsTheWindowLists(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: needs lawful install")
	}
	x, err := game.NewMapInspector(root)
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, m := range x.Maps {
		key := "loose/" + m.Source
		if m.FromArchive {
			key = "scenario/" + m.Source
		}
		e := x.Editor()
		if err := e.Open(key); err != nil {
			t.Fatal(err)
		}
		var want []string
		for _, f := range e.Document().Edits.(ui.InspectionChecker).Report() {
			want = append(want, f.Line())
		}
		var out bytes.Buffer
		if err := run([]string{"-assets", root, "-map", key, "-check"}, &out); err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		var got []string
		for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
			if strings.HasPrefix(line, "error ") || strings.HasPrefix(line, "warning ") {
				got = append(got, line)
			}
		}
		if strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Fatalf("%s: -check printed\n%s\nthe window lists\n%s", key, strings.Join(got, "\n"), strings.Join(want, "\n"))
		}
		if !strings.Contains(out.String(), "report: 0 errors,") {
			t.Fatalf("%s: no count line in %q", key, out.String())
		}
		total += len(got)
	}
	t.Logf("%d maps, %d findings printed", len(x.Maps), total)
}
