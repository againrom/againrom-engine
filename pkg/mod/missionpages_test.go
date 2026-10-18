package mod

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMissionPagesLanguageTailCompatibility(t *testing.T) {
	for _, declared := range []string{"mission/{lang}", "mission/en", "{lang}", "en"} {
		for _, lang := range []string{"en", "ru"} {
			t.Run(declared+"/"+lang, func(t *testing.T) {
				dir := t.TempDir()
				prefix := ""
				if strings.Contains(declared, "/") {
					prefix = "mission/"
				}
				for name, data := range map[string]string{"map.alm": "map", "en/event01.txt": "EN page", "ru/event01.txt": "RU page", "event02.txt": "Common"} {
					name = prefix + name
					p := filepath.Join(dir, filepath.FromSlash(name))
					if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(p, []byte(data), 0600); err != nil {
						t.Fatal(err)
					}
				}
				src := fmt.Sprintf("[[mission]]\nkey = \"m\"\nkind = \"replace\"\nnumber = 20\nmap = %q\ntexts = %q\n", prefix+"map.alm", declared)
				d, err := ParseMissions("m", MissionsFile, []byte(src), nil, lang)
				if err != nil {
					t.Fatal(err)
				}
				m := d.Missions[0]
				want := "EN page"
				if lang == "ru" && strings.Contains(declared, "{lang}") {
					want = "RU page"
				}
				if err := m.ReadFiles(dir); err != nil || string(m.TextFiles["event01.txt"]) != want || string(m.TextFiles["event02.txt"]) != "Common" {
					t.Fatalf("pages=%q: %v", m.TextFiles, err)
				}
			})
		}
	}
}

func TestMissionPagesLanguageAndLegacyFallback(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, arm := range []string{"beside", "language", "legacy", "missing"} {
			t.Run(lang+"/"+arm, func(t *testing.T) {
				dir := t.TempDir()
				write := func(name, data string) {
					t.Helper()
					p := filepath.Join(dir, filepath.FromSlash(name))
					if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(p, []byte(data), 0600); err != nil {
						t.Fatal(err)
					}
				}
				write("mission/map.alm", "map bytes")
				want := ""
				if arm != "missing" {
					write("mission/text/event01.txt", "Legacy")
					want = "Legacy"
				}
				if arm == "language" || arm == "beside" {
					write("mission/text/"+lang+"/event01.txt", "Localized "+lang)
					want = "Localized " + lang
				}
				if arm == "beside" {
					write("mission/map.text/"+lang+"/event01.txt", "Beside "+lang)
					want = "Beside " + lang
				}
				src := fmt.Sprintf("[[mission]]\nkey = \"map\"\nkind = \"replace\"\nnumber = 20\nmap = \"mission/map.alm\"\ntexts = \"mission/text/{lang}\"\n")
				d, err := ParseMissions("map", MissionsFile, []byte(src), nil, lang)
				if err != nil {
					t.Fatal(err)
				}
				m := d.Missions[0]
				err = m.ReadFiles(dir)
				if arm == "missing" {
					if err == nil {
						t.Fatal("missing declared pages accepted")
					}
					return
				}
				if err != nil || string(m.TextFiles["event01.txt"]) != want {
					t.Fatalf("page=%q want=%q: %v", m.TextFiles["event01.txt"], want, err)
				}
			})
		}
	}
}

func TestMissionPagesErrorNamesFileOnce(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "map.text", "en"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "map.alm"), []byte("map"), 0600); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(dir, "map.text", "en", "Event01.txt")
	if err := os.WriteFile(bad, []byte("page"), 0600); err != nil {
		t.Fatal(err)
	}
	m := Mission{Map: "map.alm"}
	err := m.ReadFiles(dir)
	if err == nil || !strings.Contains(err.Error(), bad) || strings.Count(err.Error(), "texts:") != 1 {
		t.Fatalf("unactionable page error: %v", err)
	}
}
