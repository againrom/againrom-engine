package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/mod"
)

func TestMapReportNamesAMisnamedTextFile(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "map.alm")
	for _, name := range []string{"Event01.txt", "desktop.ini"} {
		folder := filepath.Join(dir, "map.text")
		if err := os.MkdirAll(folder, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(folder, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		x := checkInspector()
		m := built(t, cleanOptions(), nil, nil)
		rows := x.report(m, source, x.editorPages(source))
		if len(rows) != 1 || !rows[0].Error || rows[0].Rule != ruleTextsFolder || rows[0].Kind != "" || !strings.Contains(rows[0].Text, name) {
			t.Fatalf("%s: %+v", name, rows)
		}
		if err := os.Remove(filepath.Join(folder, name)); err != nil {
			t.Fatal(err)
		}
	}
	x := checkInspector()
	m := built(t, cleanOptions(), nil, nil)
	if rows := x.report(m, source, x.editorPages(source)); len(rows) != 0 {
		t.Fatalf("an empty texts folder reports %+v", rows)
	}
}

func TestMapReportNamesAMapTheModelCannotDecode(t *testing.T) {
	x := checkInspector()
	rows := x.CheckModMap(mod.Mission{MapBytes: []byte("not a map")}, "mod/x")
	if len(rows) != 1 || !rows[0].Error || rows[0].Rule != ruleMapDecode || rows[0].Text == "" {
		t.Fatalf("%+v", rows)
	}
}

func TestMapStopRulesAreRowsOfTheTable(t *testing.T) {
	for _, id := range []string{ruleTextsFolder, ruleMapDecode} {
		found := false
		for _, r := range mapRules {
			found = found || r.ID == id && r.Error
		}
		if !found {
			t.Fatalf("no error row %q", id)
		}
	}
}
