package archtest

import (
	"os"
	"strings"
	"testing"
)

// One planted instance per shape, and the forms the scan leaves alone.
func TestCheckHostFilesNamesEachShape(t *testing.T) {
	const game = "package game\nimport \"os\"\n"
	cases := []struct {
		name, file, src, shape string
	}{
		{"rename", "pkg/game/a.go", game + "func f() { _ = os.Rename(\"a\", \"b\") }\n", "os.Rename"},
		{"temp file", "pkg/game/a.go", game + "func f() { _, _ = os.CreateTemp(\"\", \"x\") }\n", "os.CreateTemp"},
		{"temp folder", "pkg/mod/a.go", "package mod\nimport \"os\"\nfunc f() { _, _ = os.MkdirTemp(\"\", \"x\") }\n", "os.MkdirTemp"},
		{"exclusive flag", "pkg/game/a.go", game + "var flag = os.O_EXCL\n", "os.O_EXCL"},
		{"move without replace by name", "pkg/game/a_windows.go", "package game\nimport \"syscall\"\nvar p = syscall.NewLazyDLL(\"kernel32.dll\").NewProc(\"MoveFileExW\")\n", "MoveFileEx"},
		{"move without replace by identifier", "pkg/game/a.go", "package game\nfunc f() { moveFileExW() }\n", "MoveFileEx"},
		{"hard link", "cmd/starter/a.go", "package main\nimport \"os\"\nfunc f() { _ = os.Link(\"a\", \"b\") }\n", "os.Link"},
		{"whole-file write", "cmd/mapedit/a.go", "package main\nimport \"os\"\nfunc f() { _ = os.WriteFile(\"a\", nil, 0o600) }\n", "os.WriteFile"},
		{"create", "cmd/againrom/a.go", "package main\nimport \"os\"\nfunc f() { _, _ = os.Create(\"a\") }\n", "os.Create"},
		{"open to create", "pkg/ui/a.go", "package ui\nimport \"os\"\nfunc f() { _, _ = os.OpenFile(\"a\", os.O_WRONLY|os.O_CREATE, 0o600) }\n", "os.OpenFile"},
		{"open with unknown flags", "pkg/ui/a.go", "package ui\nimport \"os\"\nfunc f(flag int) { _, _ = os.OpenFile(\"a\", flag, 0o600) }\n", "os.OpenFile"},
		{"a function value", "pkg/game/a.go", game + "var publish = os.Rename\n", "os.Rename"},
		{"an aliased import", "pkg/game/a.go", "package game\nimport files \"os\"\nfunc f() { _ = files.WriteFile(\"a\", nil, 0o600) }\n", "os.WriteFile"},
		{"a dot import", "pkg/game/a.go", "package game\nimport . \"os\"\nfunc f() { _ = Rename(\"a\", \"b\") }\n", "dot import of os"},
		{"system rename", "pkg/game/a.go", "package game\nimport \"syscall\"\nfunc f() { _ = syscall.Rename(\"a\", \"b\") }\n", "syscall.Rename"},
		{"system create", "pkg/game/a_windows.go", "package game\nimport \"syscall\"\nfunc f(p *uint16) { _, _ = syscall.CreateFile(p, 0, 0, nil, syscall.CREATE_NEW, 0, 0) }\n", "syscall.CreateFile"},
	}
	for _, tc := range cases {
		_, vs := CheckHostFiles(map[string]string{tc.file: tc.src}, nil)
		if len(vs) != 1 || !strings.Contains(vs[0], tc.shape) {
			t.Errorf("%s: violations %v, want one naming %s", tc.name, vs, tc.shape)
		}
	}
	clean := []struct{ name, file, src string }{
		{"open to read", "pkg/game/a.go", game + "func f() { _, _ = os.OpenFile(\"a\", os.O_RDONLY, 0) }\n"},
		{"system open of an existing file", "pkg/mod/a_windows.go", "package mod\nimport \"syscall\"\nfunc f(p *uint16) { _, _ = syscall.CreateFile(p, 0, 0, nil, syscall.OPEN_EXISTING, 0, 0) }\n"},
		{"the transaction", "pkg/hostfile/hostfile.go", "package hostfile\nimport \"os\"\nfunc f() { _ = os.Rename(\"a\", \"b\"); _, _ = os.CreateTemp(\"\", \"x\") }\n"},
		{"a primitive file", "pkg/hostfile/publish_windows.go", "package hostfile\nimport \"syscall\"\nvar p = syscall.NewLazyDLL(\"kernel32.dll\").NewProc(\"MoveFileExW\")\n"},
		{"a test file", "pkg/game/a_test.go", game + "func f() { _ = os.Rename(\"a\", \"b\") }\n"},
		{"a developer tool", "cmd/maptool/a.go", "package main\nimport \"os\"\nfunc f() { _ = os.WriteFile(\"a\", nil, 0o600) }\n"},
		{"another package's Rename", "pkg/game/a.go", "package game\nimport \"strings\"\nvar r = strings.NewReplacer(\"a\", \"b\").Replace\nfunc f(x interface{ Rename() }) { x.Rename() }\n"},
	}
	for _, tc := range clean {
		if _, vs := CheckHostFiles(map[string]string{tc.file: tc.src}, nil); len(vs) != 0 {
			t.Errorf("%s: violations %v, want none", tc.name, vs)
		}
	}
	named := map[string]string{"pkg/game/a.go os.CreateTemp": "a probe"}
	if _, vs := CheckHostFiles(map[string]string{"pkg/game/a.go": game + "func f() { _, _ = os.CreateTemp(\"\", \"x\") }\n"}, named); len(vs) != 0 {
		t.Errorf("a named entry: violations %v, want none", vs)
	}
	if _, vs := CheckHostFiles(map[string]string{"pkg/game/a.go": game}, named); len(vs) != 1 || !strings.Contains(vs[0], "names nothing") {
		t.Errorf("a stale entry: violations %v, want one", vs)
	}
}

// The live tree writes every kept file through the transaction.
func TestHostFilesAreOneTransaction(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadRandomnessSources(root)
	if err != nil {
		t.Fatal(err)
	}
	hits, vs := CheckHostFiles(files, HostFileEntries)
	if hits < len(HostFileEntries)+8 {
		t.Fatalf("host-file shapes read = %d, want at least %d (the entries and the transaction's own); the scan guards nothing", hits, len(HostFileEntries)+8)
	}
	for _, v := range vs {
		t.Error(v)
	}
}
