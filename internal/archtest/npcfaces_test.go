package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestCheckNPCFacesNamesEachShape drives the scan with synthetic sources, one
// case per shape and per exemption.
func TestCheckNPCFacesNamesEachShape(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"a Portraits call", "pkg/game/a.go",
			"package game\nfunc f(t *T) { _ = t.NPCCatalog.Portraits() }\n", "Portraits reads the catalog's faces"},
		{"a Portraits call on a local", "pkg/game/a.go",
			"package game\nfunc f(c *C) { _ = c.Portraits()[3] }\n", "Portraits reads the catalog's faces"},
		{"the Faces field through the table", "pkg/game/a.go",
			"package game\nfunc f(t *T) { _ = t.NPCCatalog.Faces }\n", "Faces map is read"},
		{"the Faces field through a name", "pkg/game/a.go",
			"package game\nfunc f(t *T) {\n\tif catalog := t.NPCCatalog; catalog != nil {\n\t\t_ = catalog.Faces\n\t}\n}\n", "Faces map is read"},
		{"the Faces field through a loader call", "pkg/game/a.go",
			"package game\nfunc f() { _ = data.LoadNPCCatalog(r).Faces }\n", "Faces map is read"},
		{"LoadNPCFaces", "pkg/game/a.go",
			"package game\nfunc f() { _ = data.LoadNPCFaces(r) }\n", "LoadNPCFaces reads"},
		{"a tool under cmd", "cmd/a/a.go",
			"package main\nfunc f(t *T) { _ = t.NPCCatalog.Portraits() }\n", "Portraits reads the catalog's faces"},
		{"the face source", "pkg/game/gamefiles.go",
			"package game\nfunc f(t *T) { _ = t.NPCCatalog.Portraits() }\n", ""},
		{"the catalog's own package", "pkg/data/npccatalog.go",
			"package data\nfunc f() { _ = LoadNPCCatalog(r).Faces }\n", ""},
		{"a Faces field of another type", "pkg/game/a.go",
			"package game\nfunc f(s *S) { _ = s.in.Faces; _ = a.Faces }\n", ""},
		{"an unrelated name", "pkg/game/a.go",
			"package game\nfunc f(s *S) { catalog := s.shop; _ = catalog.Faces }\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, v := range CheckNPCFaces(map[string]string{tc.file: tc.src}) {
				got = append(got, v.Reason)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckNPCFaces = %v, want none", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0], tc.want) {
				t.Fatalf("CheckNPCFaces = %v, want one naming %q", got, tc.want)
			}
		})
	}
	if vs := CheckNPCFaces(nil); len(vs) != 1 {
		t.Fatalf("CheckNPCFaces(nil) = %v, want one violation", vs)
	}
}

// TestNPCFacesHaveOneSource is the live-tree half.
func TestNPCFacesHaveOneSource(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadNPCFaceSources(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range CheckNPCFaces(files) {
		t.Errorf("%s", v)
	}
}
