package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestCheckDrawnTextNamesEachViolation drives the pure evaluator with synthetic
// sources, one case per arm, so the rule is pinned without reading the tree.
func TestCheckDrawnTextNamesEachViolation(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "an em dash in a drawn message is a violation",
			src:  "package p\nvar s = \"a \u2014 b\"\n",
			want: "pkg/x.go:2",
		},
		{
			name: "a Cyrillic literal is a violation",
			src:  "package p\nvar s = \"\u0448\u043b\u0435\u043c\"\n",
			want: "pkg/x.go:2",
		},
		{
			name: "an escaped high byte is a violation, because it is one at run time",
			src:  "package p\nvar s = \"\\x98\"\n",
			want: "pkg/x.go:2",
		},
		{
			name: "an em dash in a comment is not a violation",
			src:  "package p\n\n// a \u2014 b\nvar s = \"a - b\"\n",
			want: "",
		},
		{
			name: "plain ASCII is not a violation",
			src:  "package p\nvar s = \"a - b\"\n",
			want: "",
		},
		{
			name: "an import path is a string literal and stays ASCII",
			src:  "package p\n\nimport \"fmt\"\n\nvar _ = fmt.Sprint\n",
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vs := CheckDrawnText(map[string]string{"pkg/x.go": tc.src})
			if tc.want == "" {
				if len(vs) != 0 {
					t.Fatalf("CheckDrawnText = %v, want none", vs)
				}
				return
			}
			if len(vs) != 1 {
				t.Fatalf("CheckDrawnText = %v, want exactly one violation", vs)
			}
			if vs[0].From != tc.want {
				t.Errorf("violation at %q, want %q", vs[0].From, tc.want)
			}
		})
	}
}

// TestCheckDrawnTextEmptyFileSet pins that finding nothing to read is itself a
// violation, so a loader that stops working cannot read as a clean scan.
func TestCheckDrawnTextEmptyFileSet(t *testing.T) {
	for _, files := range []map[string]string{nil, {}} {
		vs := CheckDrawnText(files)
		if len(vs) != 1 {
			t.Fatalf("CheckDrawnText(%v) = %v, want one violation", files, vs)
		}
	}
}

// TestLibrarySourcesDrawOnlyASCII is the live-tree half: every production
// source under pkg/ is scanned.
func TestLibrarySourcesDrawOnlyASCII(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	files, err := LoadDrawnTextSources(root)
	if err != nil {
		t.Fatalf("load library sources: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("loader found no pkg sources; the scan would pass by reading nothing")
	}
	for _, v := range CheckDrawnText(files) {
		t.Errorf("unexpected non-ASCII string literal: %s", v)
	}

	// The tree carries the evidence that separates a syntax scan from a text
	// grep: pkg sources are full of em dashes in prose while holding none in a
	// literal. A grep fails here; this check reads parsed syntax, so it passes.
	prose := 0
	for _, src := range files {
		if strings.Contains(src, "\u2014") {
			prose++
		}
	}
	if prose == 0 {
		t.Error("no pkg source names an em dash in prose any more; the " +
			"syntax-versus-grep distinction is no longer witnessed on the live tree")
	}
}
