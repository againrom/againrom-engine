package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

func TestOneStoredSmoothingKeepsTheMissionPreference(t *testing.T) {
	for _, tc := range []struct {
		stored string
		want   bool
	}{
		{"Smoothing=0\nFrameSmoothing=1\n", false},
		{"Smoothing=1\nFrameSmoothing=0\n", true},
		{"FrameSmoothing=0\n", true},
		{"Smoothing=broken\nFrameSmoothing=0\n", true},
		{"Smoothing=-0\nFrameSmoothing=1\n", false},
	} {
		stored, want := tc.stored, tc.want
		s := OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
		if err := os.WriteFile(s.Path, []byte(stored+"Future=keep\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if on, err := s.FrameSmoothing(); err != nil || on != want {
			t.Fatalf("%q: on %v, err %v, want %v", stored, on, err, want)
		}
		if err := s.SetFrameSmoothing(!want); err != nil {
			t.Fatal(err)
		}
		cold := OptionsStore{Path: s.Path}
		on, err := cold.FrameSmoothing()
		values, _, _ := cold.gameOptions()
		data, _ := os.ReadFile(s.Path)
		if err != nil || on != !want || (values[ui.GameOptionSmoothing] != 0) != on ||
			strings.Contains(string(data), "FrameSmoothing=") || !strings.Contains(string(data), "Future=keep") {
			t.Fatalf("reload on %v err %v, values %v, file %s", on, err, values, data)
		}
	}
}
