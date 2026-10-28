package mod

import (
	"fmt"
	"testing"
)

func readOtherLanguageFixture(t *testing.T, lang, texts string, files map[string]string) (Mission, error) {
	t.Helper()
	dir := t.TempDir()
	writeTreeFile(t, dir, "mission/map.alm", "map")
	for name, body := range files {
		writeTreeFile(t, dir, name, body)
	}
	src := "[[mission]]\nkey = \"map\"\nkind = \"replace\"\nnumber = 20\nmap = \"mission/map.alm\"\n"
	if texts != "" {
		src += fmt.Sprintf("texts = %q\n", texts)
	}
	d, err := ParseMissions("map", MissionsFile, []byte(src), nil, lang)
	if err != nil {
		t.Fatal(err)
	}
	m := d.Missions[0]
	return m, m.ReadFiles(dir)
}

func TestMissionPagesOtherLanguageFillsMissingNames(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		other := map[string]string{"en": "ru", "ru": "en"}[lang]
		t.Run(lang, func(t *testing.T) {
			m, err := readOtherLanguageFixture(t, lang, "", map[string]string{
				"mission/map.text/" + other + "/event01.txt": "Other one",
				"mission/map.text/" + other + "/event02.txt": "Other two",
				"mission/map.text/" + lang + "/event02.txt":  "Own two",
			})
			if err != nil {
				t.Fatal(err)
			}
			if string(m.TextFiles["event01.txt"]) != "Other one" || string(m.TextFiles["event02.txt"]) != "Own two" || len(m.TextFiles) != 2 {
				t.Fatalf("pages=%q", m.TextFiles)
			}
			if !m.OtherLanguageTexts["event01.txt"] || m.OtherLanguageTexts["event02.txt"] {
				t.Fatalf("other-language names=%v", m.OtherLanguageTexts)
			}
		})
	}
}

func TestMissionPagesUnlanguagedBeatsOtherLanguage(t *testing.T) {
	m, err := readOtherLanguageFixture(t, "en", "", map[string]string{
		"mission/map.text/event01.txt":    "Common",
		"mission/map.text/ru/event01.txt": "Russian",
	})
	if err != nil || string(m.TextFiles["event01.txt"]) != "Common" || m.OtherLanguageTexts["event01.txt"] {
		t.Fatalf("pages=%q other=%v: %v", m.TextFiles, m.OtherLanguageTexts, err)
	}
}

func TestMissionPagesDeclaredFolderOtherLanguageFills(t *testing.T) {
	m, err := readOtherLanguageFixture(t, "en", "mission/text/{lang}", map[string]string{
		"mission/text/ru/event01.txt": "Declared Russian one",
		"mission/text/ru/event02.txt": "Declared Russian two",
		"mission/text/en/event02.txt": "Declared English two",
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(m.TextFiles["event01.txt"]) != "Declared Russian one" || string(m.TextFiles["event02.txt"]) != "Declared English two" {
		t.Fatalf("pages=%q", m.TextFiles)
	}
	m, err = readOtherLanguageFixture(t, "en", "mission/text/{lang}", map[string]string{
		"mission/text/ru/event01.txt":         "Declared Russian",
		"mission/map.text/en/event01.txt":     "Beside English",
		"mission/map.text/ru/event01.txt":     "Beside Russian",
		"mission/text/ru/ignored-not-a-page":  "x",
		"mission/map.text/ru/Ignored-Name.md": "x",
	})
	if err != nil || string(m.TextFiles["event01.txt"]) != "Beside English" {
		t.Fatalf("pages=%q: %v", m.TextFiles, err)
	}
}

func TestMissionPagesDeclaredFolderMissingInBothLanguages(t *testing.T) {
	if _, err := readOtherLanguageFixture(t, "en", "mission/text/{lang}", nil); err == nil {
		t.Fatal("declared pages missing in both languages accepted")
	}
}
