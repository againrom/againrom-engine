package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The editor's panel profile file is decoded in readPanelProfile only.
func TestPanelProfileIsReadByOneReader(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "pkg", "ui", "mapeditorsidepanels.go")
	findings, err := CheckPanelProfileReader(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		t.Error(f)
	}
}

func TestCheckPanelProfileReaderNamesASecondDecode(t *testing.T) {
	const reader = `package ui

import "encoding/json"

func readPanelProfile(data []byte) { var v int; _ = json.Unmarshal(data, &v) }
`
	for _, tc := range []struct {
		name, source string
		want         string
	}{
		{"only the reader", reader, ""},
		{"a second decode", reader + `
func viewOf(data []byte) { var v string; _ = json.Unmarshal(data, &v) }
`, "viewOf decodes json outside readPanelProfile"},
		{"a method decode", reader + `
func (e *E) load(data []byte) { var v string; _ = json.Unmarshal(data, &v) }
`, "load decodes json outside readPanelProfile"},
		{"no reader", `package ui

func other() {}
`, "readPanelProfile is not declared"},
	} {
		got, err := CheckPanelProfileReader("side.go", tc.source)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if tc.want == "" {
			if len(got) != 0 {
				t.Errorf("%s: findings %v, want none", tc.name, got)
			}
			continue
		}
		if len(got) != 1 || !strings.Contains(got[0], tc.want) {
			t.Errorf("%s: findings %v, want one naming %q", tc.name, got, tc.want)
		}
	}
}
