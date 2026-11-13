package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The compatibility accessor uses the shared Smoothing row and preserves unrelated keys.
func TestFrameSmoothingPreference(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.txt")
	s := OptionsStore{Path: path}
	if on, err := s.FrameSmoothing(); !on || err != nil {
		t.Fatalf("missing file: FrameSmoothing = %v, %v; want true, nil", on, err)
	}
	for _, c := range []struct {
		stored string
		want   bool
	}{{"TipsMode=0\n", true}, {"FrameSmoothing=0\n", true}, {"Smoothing=0\n", false}, {"FrameSmoothing=1\n", true}, {"FrameSmoothing=x\n", true}} {
		if err := os.WriteFile(path, []byte(c.stored), 0o644); err != nil {
			t.Fatal(err)
		}
		if on, _ := s.FrameSmoothing(); on != c.want {
			t.Errorf("%q: FrameSmoothing = %v, want %v", c.stored, on, c.want)
		}
	}

	if err := os.WriteFile(path, []byte("TipsMode=0\nTextSmoothing=0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.SetFrameSmoothing(false); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	for _, want := range []string{"TipsMode=0", "TextSmoothing=0", "Smoothing=0"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("after SetFrameSmoothing(false) the file %q lacks %q", b, want)
		}
	}
	if on, _ := s.FrameSmoothing(); on {
		t.Error("FrameSmoothing after SetFrameSmoothing(false) = true")
	}
}

// Loaded Smoothing reaches the App, regardless of the obsolete frame key.
func TestFrontEndPlumbsFrameSmoothingToTheApp(t *testing.T) {
	var f FrontEnd
	f.LoadOptions()
	if a := f.App("t"); !a.FrameSmoothing() {
		t.Fatal("an App from a FrontEnd that loaded defaults has FrameSmoothing off")
	}
	path := filepath.Join(t.TempDir(), "options.txt")
	if err := os.WriteFile(path, []byte("Smoothing=0\nFrameSmoothing=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f.Options = OptionsStore{Path: path}
	f.LoadOptions()
	if a := f.App("t"); a.FrameSmoothing() {
		t.Fatal("an App from a FrontEnd that loaded Smoothing=0 has Smoothing on")
	}
}
