package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSoundMenuPersistsMuteAndVolumeBesideOtherPreferences(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.txt")
	if err := os.WriteFile(path, []byte("GameSpeed=6\nOther=retained\n"), 0600); err != nil {
		t.Fatal(err)
	}
	f := &FrontEnd{RuntimeServices: RuntimeServices{Sound: SoundOptions{Enabled: true, Volume: 100}}, PersistenceContext: PersistenceContext{Options: OptionsStore{Path: path}}}
	if err := f.setGameMenuSound(false, 25); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{"SoundEnabled=0\n", "MasterVolume=25\n", "GameSpeed=6\n", "Other=retained\n"} {
		if !strings.Contains(string(b), line) {
			t.Fatalf("menu change did not retain %q in %q", line, b)
		}
	}
	stored, err := (OptionsStore{Path: path}).SoundOptions(SoundOptions{Enabled: true, Volume: 100})
	if err != nil || stored != f.Sound {
		t.Fatalf("fresh reader sound=%+v want %+v: %v", stored, f.Sound, err)
	}
}

func TestSoundPreferencesInvalidFieldsAndReadFailure(t *testing.T) {
	defaults := SoundOptions{Enabled: true, Volume: 100}
	for _, tc := range []struct {
		input string
		want  SoundOptions
	}{
		{"", defaults},
		{"SoundEnabled=0\nMasterVolume=-1\n", SoundOptions{Enabled: false, Volume: 100}},
		{"SoundEnabled=maybe\nMasterVolume=25\n", SoundOptions{Enabled: true, Volume: 25}},
		{"SoundEnabled=2\nMasterVolume=101\n", defaults},
		{"SoundEnabled=0\r\nMasterVolume=100\r\nMasterVolume=0\r\n", SoundOptions{Enabled: false, Volume: 0}},
	} {
		path := filepath.Join(t.TempDir(), "options.txt")
		if err := os.WriteFile(path, []byte(tc.input), 0600); err != nil {
			t.Fatal(err)
		}
		got, err := (OptionsStore{Path: path}).SoundOptions(defaults)
		if err != nil || got != tc.want {
			t.Fatalf("read %q = %+v want %+v: %v", tc.input, got, tc.want, err)
		}
		b, err := os.ReadFile(path)
		if err != nil || string(b) != tc.input {
			t.Fatalf("read repaired file: %q: %v", b, err)
		}
	}
	// A directory at the configured path reproduces a read failure on every
	// supported OS, without relying on root/Windows chmod semantics.
	store := OptionsStore{Path: t.TempDir()}
	if got, err := store.SoundOptions(defaults); err == nil || got != defaults {
		t.Fatal("read failure lost defaults or its error", got, err)
	}
	f := &FrontEnd{RuntimeServices: RuntimeServices{Sound: defaults}, PersistenceContext: PersistenceContext{Options: store}}
	if err := f.setGameMenuSound(false, 25); err == nil || f.Sound != defaults {
		t.Fatal("failed persistence must retain prior settings and return the error", f.Sound, err)
	}
	if err := store.SetSoundOptions(SoundOptions{Volume: 101}); err == nil {
		t.Fatal("out-of-range write accepted")
	}
}
