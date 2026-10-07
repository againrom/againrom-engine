package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestFrameLogStats(t *testing.T) {
	mean, p95, max, over16, over33 := frameLogStats([]float64{10, 20, 40, 10})
	if mean != 20 || max != 40 || p95 != 40 || over16 != 2 || over33 != 1 {
		t.Fatal(mean, p95, max, over16, over33)
	}
	if m, _, _, _, _ := frameLogStats(nil); m != 0 {
		t.Fatal("empty input")
	}
}

func TestGameForFrameLogOff(t *testing.T) {
	a := &App{}
	g, closeLog := gameFor(a, "")
	closeLog()
	if g != ebiten.Game(a) {
		t.Fatal("empty path must run the App unwrapped")
	}
	if _, ok := g.(*frameLog); ok {
		t.Fatal("empty path wrapped the App")
	}
	dir := t.TempDir()
	g, closeLog = gameFor(a, filepath.Join(dir, "missing", "f.log"))
	closeLog()
	if g != ebiten.Game(a) {
		t.Fatal("unwritable path must fall back to the App")
	}
	if ents, _ := os.ReadDir(dir); len(ents) != 0 {
		t.Fatal("off or failed path wrote files", ents)
	}
}
