package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

func TestTooltipPreferencesColdStartAndOtherKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.txt")
	if err := os.WriteFile(path, []byte("GameSpeed=6\nSoundEnabled=0\nMasterVolume=25\nOther=keep\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, ms := range []int{0, 100, 200, 300, 400, 500} {
		if err := (OptionsStore{Path: path}).SetTooltipDelay(ms); err != nil {
			t.Fatal(err)
		}
		f := missionFrontEnd(t)
		f.Options = OptionsStore{Path: path}
		if got := f.App("cold-tooltip").TooltipDelay(); got != ms {
			t.Fatalf("cold app got %d, want %d", got, ms)
		}
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"GameSpeed=6", "SoundEnabled=0", "MasterVolume=25", "Other=keep"} {
		if !strings.Contains(string(b), s) {
			t.Fatal("lost preference", s)
		}
	}
}

func TestTooltipPreferencesInvalidValuesDefaultWithoutRewriting(t *testing.T) {
	for _, raw := range []string{"", "TooltipDelay=nope\n", "TooltipDelay=-100\n", "TooltipDelay=600\n", "TooltipDelay=150\n"} {
		path := filepath.Join(t.TempDir(), "options.txt")
		if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		s := OptionsStore{Path: path}
		if got, err := s.TooltipDelay(); err != nil || got != ui.DefaultTooltipDelay {
			t.Fatal(raw, got, err)
		}
		if err := s.SetTooltipDelay(150); err == nil {
			t.Fatal("accepted non-step delay")
		}
		b, _ := os.ReadFile(path)
		if string(b) != raw {
			t.Fatal("read or refused write changed options")
		}
	}
	if got, err := (OptionsStore{}).TooltipDelay(); err != nil || got != 500 {
		t.Fatal(got, err)
	}
}
