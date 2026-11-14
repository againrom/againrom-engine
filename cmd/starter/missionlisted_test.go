package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/res"
)

// writeCampaignInstall writes a scenario.res under root whose registry holds
// a section for each number and whose maps are those numbers' files.
func writeCampaignInstall(t *testing.T, root string, numbers ...int) {
	t.Helper()
	var sections []synth.RegNode
	entries := []res.WriteEntry{}
	for _, n := range numbers {
		sections = append(sections, synth.RegNode{Name: fmt.Sprintf("Mission%d", n), Kind: 0x01, Children: []synth.RegNode{{Name: "LastMission", Kind: 0x02, Int: 0}}})
		entries = append(entries, res.WriteEntry{Path: fmt.Sprintf("%d.alm", n), Data: []byte("map")})
	}
	entries = append(entries, res.WriteEntry{Path: "scenario.reg", Data: synth.Reg(0x11, sections)})
	data, err := res.Write(entries)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scenario.res"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeModFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// A row whose number the campaign lacks stays listed beside a stray
// number-named file: the game warns of the file and plays the row.
func TestStarterListsAGuestRowBesideAStrayNumberedFile(t *testing.T) {
	r := rigWithLanguage(t, "english")
	writeCampaignInstall(t, r.root, 10, 20)
	writeModFiles(t, filepath.Join(r.dir, "mods", "probe"), map[string]string{
		"mod.toml":           goodMod("probe"),
		"data/missions.toml": "[[mission]]\nkey = \"g\"\nnumber = 901\nmap = \"maps/g.alm\"\nafter = 20\nguest = 22\nchapter = 30\n",
		"maps/901.alm":       "map",
	})
	a := r.app()
	do(t, a, action{aToggleMod, 0})
	rows := missionTexts(a)
	if len(rows) != 1 || !strings.HasPrefix(rows[0], "901.") {
		t.Fatalf("mission rows %q, want the row 901; the window shows %q", rows, texts(a))
	}
}

// A number-named file whose number the install's campaign lacks is no
// mission, and the mission list holds the files the game plays only.
func TestStarterListsNoNumberedFileTheCampaignLacks(t *testing.T) {
	r := rigWithLanguage(t, "english")
	writeCampaignInstall(t, r.root, 10, 20)
	writeModFiles(t, filepath.Join(r.dir, "mods", "files"), map[string]string{
		"mod.toml":       goodMod("files"),
		"150.alm":        "map",
		"maps/65535.alm": "map",
		"maps/20.alm":    "map",
	})
	a := r.app()
	do(t, a, action{aToggleMod, 0})
	rows := missionTexts(a)
	if len(rows) != 1 || !strings.HasPrefix(rows[0], "20.") {
		t.Fatalf("mission rows %q, want the mission 20 only; the window shows %q", rows, texts(a))
	}
}

// With no install readable the starter lists a mod's rows and no file.
func TestStarterWithNoInstallListsRowsAndNoFile(t *testing.T) {
	r := rigWithLanguage(t, "english")
	writeModFiles(t, filepath.Join(r.dir, "mods", "probe"), map[string]string{
		"mod.toml":           goodMod("probe"),
		"data/missions.toml": "[[mission]]\nkey = \"g\"\nnumber = 901\nmap = \"maps/g.alm\"\nafter = 20\nguest = 22\nchapter = 30\n",
		"maps/901.alm":       "map",
		"7.alm":              "map",
	})
	a := r.app()
	do(t, a, action{aToggleMod, 0})
	rows := missionTexts(a)
	if len(rows) != 1 || !strings.HasPrefix(rows[0], "901.") {
		t.Fatalf("mission rows %q, want the row 901 only; the window shows %q", rows, texts(a))
	}
}
