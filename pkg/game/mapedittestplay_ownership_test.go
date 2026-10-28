package game

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

func writeTestPlayInput(t *testing.T, name string, b []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestTestPlayRefusesAnAuthoredDestination(t *testing.T) {
	for _, selected := range []bool{false, true} {
		t.Run(map[bool]string{false: "unselected", true: "selected-and-opened"}[selected], func(t *testing.T) {
			f := newTestPlayFixture(t)
			folder := filepath.Join(f.profile, TestPlayModID, TestPlayModID)
			manifest := strings.Replace(editorModManifest, `id = "ambush"`, `id = "testplay"`, 1)
			declaration := "[[mission]]\nkind = \"replace\"\nkey = \"authored\"\nnumber = 20\nmap = \"mission/map.alm\"\ntexts = \"mission/text\"\n"
			opened := filepath.Join(folder, filepath.FromSlash(testPlayMap))
			original := synth.ALM(synth.ALMOptions{Width: 4, Height: 4, Name: "authored"})
			for name, b := range map[string][]byte{
				mod.ManifestName: []byte(manifest), "main.star": []byte(testPlayEntry),
				mod.MissionsFile: []byte(declaration), testPlayMap: original,
				"mission/text/event01.txt": []byte("Author page"),
			} {
				writeTestPlayInput(t, filepath.Join(folder, filepath.FromSlash(name)), b)
			}
			source := "scenario/20.alm"
			if selected {
				if err := f.x.SetMod(folder); err != nil {
					t.Fatal(err)
				}
				source = opened
			}
			s := editorEdits(t, f.x, source)
			s.saved = s.model.Bytes()
			if err := s.model.SetAltitude(1, 1, 7); err != nil {
				t.Fatal(err)
			}
			before := tree(t, f.profile)
			_, err := s.TestPlay()
			if err == nil || !strings.Contains(err.Error(), folder) {
				t.Fatalf("destination refusal must name %s: %v", folder, err)
			}
			if !reflect.DeepEqual(before, tree(t, f.profile)) || len(f.started) != 0 || !s.Dirty() {
				t.Fatal("refusal changed an authored byte, started a game or dropped the unsaved edit")
			}
		})
	}
}

func TestTestPlayPreservesUnownedPagesAndRemovesOnlyRecordedPages(t *testing.T) {
	t.Run("unowned-destination", func(t *testing.T) {
		f := newTestPlayFixture(t)
		folder := filepath.Join(f.profile, TestPlayModID, TestPlayModID)
		writeTestPlayInput(t, filepath.Join(folder, "mission", "text", "event01.txt"), []byte("Author page"))
		before := tree(t, f.profile)
		_, err := editorEdits(t, f.x, "scenario/20.alm").TestPlay()
		if err == nil || !strings.Contains(err.Error(), folder) || len(f.started) != 0 || !reflect.DeepEqual(before, tree(t, f.profile)) {
			t.Fatalf("unowned page was not preserved: %v; starts=%d", err, len(f.started))
		}
	})
	t.Run("owned-destination", func(t *testing.T) {
		f := newTestPlayFixture(t)
		s := editorEdits(t, f.x, "scenario/20.alm")
		if _, err := s.TestPlay(); err != nil {
			t.Fatal(err)
		}
		folder := filepath.Join(f.profile, TestPlayModID, TestPlayModID)
		foreign := filepath.Join(folder, "mission", "text", "event01.txt")
		writeTestPlayInput(t, foreign, []byte("Author page added after test play"))
		s = editorEdits(t, f.x, "new.alm")
		if err := s.model.SetAltitude(1, 1, 7); err != nil {
			t.Fatal(err)
		}
		if _, err := s.TestPlay(); err != nil {
			t.Fatal(err)
		}
		got := tree(t, folder)
		if got[testPlayMap] != string(mustSaveBytes(t, s)) || got["mission/text/event01.txt"] != "Author page added after test play" {
			t.Fatal("owned replacement lost the edited map or unowned page")
		}
		if _, exists := got["mission/text/event02.txt"]; exists || len(f.started) != 2 {
			t.Fatal("the recorded stale page remained or the replacement did not start")
		}
	})
}

func TestTestPlayRefusesAModLoadBeforeReplacingTheDestination(t *testing.T) {
	f := newTestPlayFixture(t)
	if _, err := editorEdits(t, f.x, "scenario/20.alm").TestPlay(); err != nil {
		t.Fatal(err)
	}
	f.started = nil
	modDir := writeEditorMod(t, t.TempDir(), editorModMissions+"texts = \"mission/{lang}/text\"\n")
	opened := filepath.Join(modDir, "mission", "en", "map.alm")
	writeTestPlayInput(t, opened, synth.ALM(synth.ALMOptions{Width: 4, Height: 4, Name: "authored"}))
	page := filepath.Join(modDir, "mission", "en", "text", "event01.txt")
	writeTestPlayInput(t, page, bytes.Repeat([]byte("A"), mod.MaxFileBytes+1))
	if err := f.x.SetMod(modDir); err != nil {
		t.Fatal(err)
	}
	before, source := tree(t, f.profile), tree(t, modDir)
	s := editorEdits(t, f.x, opened)
	_, err := s.TestPlay()
	var refusal *ui.ReportRefusal
	if !errors.As(err, &refusal) || len(refusal.Findings) != 1 || !strings.Contains(refusal.Findings[0].Text, "mission/en/text/event01.txt is larger than 1048576 bytes") {
		t.Fatalf("page source refusal: %v (%+v)", err, refusal)
	}
	if len(f.started) != 0 || !reflect.DeepEqual(before, tree(t, f.profile)) || !reflect.DeepEqual(source, tree(t, modDir)) {
		t.Fatal("mod-load refusal published files, changed the source or started the game")
	}
	writeTestPlayInput(t, page, []byte("Page in bounds"))
	if err := f.x.SetMod(modDir); err != nil {
		t.Fatal(err)
	}
	if _, err := s.TestPlay(); err != nil || len(f.started) != 1 {
		t.Fatalf("in-bound loss control: %v; starts=%d", err, len(f.started))
	}
}

func TestTestPlayLoadsRetainedUnrecordedPagesBeforePublishing(t *testing.T) {
	f, folder := ownedTestPlayFixture(t)
	page := filepath.Join(folder, "mission", "text", "event01.txt")
	writeTestPlayInput(t, page, bytes.Repeat([]byte("A"), mod.MaxFileBytes+1))
	before := testPlayPathSnapshot(t, f.profile)
	_, err := editorEdits(t, f.x, "scenario/20.alm").TestPlay()
	if err == nil || !strings.Contains(err.Error(), "mission/text/event01.txt is larger than 1048576 bytes") {
		t.Fatalf("retained-page loader refusal: %v", err)
	}
	if len(f.started) != 0 || !reflect.DeepEqual(before, testPlayPathSnapshot(t, f.profile)) {
		t.Fatal("loader refusal changed or started the destination containing an unrecorded page")
	}
	writeTestPlayInput(t, page, []byte("Retained in-bound page"))
	if _, err := editorEdits(t, f.x, "scenario/20.alm").TestPlay(); err != nil || len(f.started) != 1 {
		t.Fatalf("retained in-bound loss control: %v; starts=%d", err, len(f.started))
	}
}

func TestTestPlayInstallProfileWritesOnlyThePrivateFolderAndReplacesIt(t *testing.T) {
	f := newTestPlayFixture(t)
	install := profileInstall(t)
	profile, err := ResolveRuntimeProfile("", filepath.Join(install, "mapedit.exe"), t.TempDir(), install)
	if err != nil {
		t.Fatal(err)
	}
	before := tree(t, install)
	s := editorEdits(t, f.x, "scenario/20.alm")
	for _, height := range []uint8{3, 7} {
		if err := s.model.SetAltitude(1, 1, height); err != nil {
			t.Fatal(err)
		}
		m, err := f.x.testPlayMission(s.source, mustSaveBytes(t, s), nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := writeTestMod(profile, install, m); err != nil {
			t.Fatal(err)
		}
		for name, b := range tree(t, install) {
			if original, exists := before[name]; exists {
				if b != original {
					t.Fatalf("installed byte changed: %s", name)
				}
			} else if !strings.HasPrefix(name, "Againrom/testplay/") {
				t.Fatalf("file outside private test play folder: %s", name)
			}
		}
		got := tree(t, filepath.Join(profile.Directory, TestPlayModID, TestPlayModID))
		if len(got) != 6 || got[testPlayRecord] == "" || got[testPlayMap] != string(m.MapBytes) {
			t.Fatalf("private test mod: %v", keys(got))
		}
	}
}
