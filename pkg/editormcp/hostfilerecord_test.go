package editormcp

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// The editor profile file's record: a new file, an equal file left alone, a
// replaced file, a file whose folder is missing, and what each leaves.
func TestHostFileRecordProfileFile(t *testing.T) {
	dir := t.TempDir()
	digits := regexp.MustCompile(`[0-9]{5,}`)
	var b strings.Builder
	norm := func(s string) string {
		return digits.ReplaceAllString(strings.ReplaceAll(filepath.ToSlash(s), filepath.ToSlash(dir), "$T"), "#")
	}
	tree := func() {
		var rows []string
		_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
			if err != nil || p == dir {
				return nil
			}
			rel, _ := filepath.Rel(dir, p)
			body, _ := os.ReadFile(p)
			info, _ := d.Info()
			rows = append(rows, norm(filepath.ToSlash(rel)+" "+(info.Mode().Perm()&0o700).String()+" "+string(body)))
			return nil
		})
		sort.Strings(rows)
		b.WriteString(strings.Join(rows, "\n") + "\n")
	}
	file := filepath.Join(dir, "panel.json")
	step := func(what string, content string, at string) {
		err := WriteProfileFile(at, ".panel-*", "the panel state", []byte(content))
		msg := "ok"
		if err != nil {
			msg = err.Error()
		}
		b.WriteString(what + ": " + norm(msg) + "\n")
		tree()
	}
	step("new", "one", file)
	before, _ := os.Stat(file)
	step("equal", "one", file)
	after, _ := os.Stat(file)
	b.WriteString("equal left the file: " + map[bool]string{true: "yes", false: "no"}[os.SameFile(before, after)] + "\n")
	step("replace", "two", file)
	step("missing folder", "x", filepath.Join(dir, "absent", "panel.json"))
	got := b.String()
	golden := filepath.Join("testdata", "hostfile_record.txt")
	if os.Getenv("AGAINROM_WRITE_HOSTFILE_RECORD") == "1" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != got {
		t.Fatalf("profile file record differs:\nrecorded\n%s\nnow\n%s", want, got)
	}
}
