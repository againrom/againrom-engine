package res_test

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/res"
)

func TestOpenFileIndexReadsIndividualPayloads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "MUSIC.RES")
	want := []byte("left-right-stereo")
	archive := synth.Archive([]synth.File{{Path: "B00.wav", Data: want}, {Path: "menu.wav", Data: []byte("menu")}})
	if err := os.WriteFile(path, archive, 0o600); err != nil {
		t.Fatal(err)
	}
	a, err := res.OpenFileIndex(path)
	if err != nil {
		t.Fatalf("OpenFileIndex: %v", err)
	}
	if got := a.Entries(); len(got) != 2 {
		t.Fatalf("entries = %v, want 2", got)
	}
	got, err := a.ReadFile("b00.WAV")
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("ReadFile = %q, %v; want %q", got, err, want)
	}
	if _, err := a.ReadFile("absent.wav"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("absent error = %v, want fs.ErrNotExist", err)
	}
}
