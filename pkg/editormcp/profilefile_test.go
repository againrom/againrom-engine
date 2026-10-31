package editormcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The MCP record goes through the shared profile writer byte for byte as it
// did on its own: the JSON object and one newline, the folder left with the
// record alone, and an equal record left as it is.
func TestProfileWriterKeepsTheMCPRecordBytes(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, RecordName)
	r := record{URL: "http://127.0.0.1:4567/mcp", Token: strings.Repeat("ab", 32), Port: 4567}
	if err := writeRecord(file, r); err != nil {
		t.Fatal(err)
	}
	want := `{"url":"http://127.0.0.1:4567/mcp","token":"` + strings.Repeat("ab", 32) + `","port":4567}` + "\n"
	got, err := os.ReadFile(file)
	if err != nil || string(got) != want {
		t.Fatalf("record %q, %v", got, err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Fatalf("the folder holds %d files", len(entries))
	}
	before, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeRecord(file, r); err != nil {
		t.Fatal(err)
	}
	if after, err := os.Stat(file); err != nil || !os.SameFile(before, after) || !after.ModTime().Equal(before.ModTime()) {
		t.Fatal("an equal record was written again")
	}
	// Another profile file is written the same way beside it.
	other := filepath.Join(dir, "mapedit-panels.json")
	if err := WriteProfileFile(other, ".mapedit-panels-*.tmp", "panel state", []byte("{}\n")); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(other); string(b) != "{}\n" {
		t.Fatalf("panel state %q", b)
	}
	if b, _ := os.ReadFile(file); string(b) != want {
		t.Fatalf("the record changed to %q", b)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 2 {
		t.Fatalf("the folder holds %d files", len(entries))
	}
	// A folder that does not exist fails the write.
	if err := WriteProfileFile(filepath.Join(dir, "missing", "x.json"), ".x-*.tmp", "panel state", []byte("1")); err == nil {
		t.Fatal("a write into a missing folder succeeded")
	}
}
