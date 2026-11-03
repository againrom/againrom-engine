package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// starter.ini's record: the bytes a save writes, a save over an existing
// file, a save into a folder still to be made, and a save whose folder is a
// file. The record is written inline so a change of any of them fails here.
func TestHostFileRecordStarterIni(t *testing.T) {
	dir := t.TempDir()
	var b strings.Builder
	norm := func(s string) string { return strings.ReplaceAll(filepath.ToSlash(s), filepath.ToSlash(dir), "$T") }
	for i, path := range []string{
		filepath.Join(dir, "starter.ini"),
		filepath.Join(dir, "starter.ini"),
		filepath.Join(dir, "new", "deeper", "starter.ini"),
	} {
		if i == 1 {
			if err := os.WriteFile(path, []byte("; old\nold = 1\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		a := newApp(path, "starter.exe", "windows", "0.0.0", deps{})
		a.save()
		got, err := os.ReadFile(path)
		info, _ := os.Stat(path)
		b.WriteString(norm(a.status) + "\n" + string(got) + "\n")
		if err == nil {
			b.WriteString((info.Mode().Perm() & 0o700).String() + "\n")
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "blocked"), []byte("a file"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := newApp(filepath.Join(dir, "blocked", "starter.ini"), "starter.exe", "windows", "0.0.0", deps{})
	a.save()
	b.WriteString("bad=" + map[bool]string{true: "yes", false: "no"}[a.statusBad] + "\n")
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		b.WriteString(e.Name() + "\n")
	}
	got := b.String()
	if want := os.Getenv("AGAINROM_WRITE_HOSTFILE_RECORD"); want == "1" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join("testdata", "hostfile_record.txt"), []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(filepath.Join("testdata", "hostfile_record.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != got {
		t.Fatalf("starter.ini record differs:\nrecorded\n%s\nnow\n%s", want, got)
	}
}
