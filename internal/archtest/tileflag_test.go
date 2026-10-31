package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

// tileFlagOwner spells the tile word's impassable bit; every other tile
// reader takes alm.TileImpassable or alm.Impassable.
const tileFlagOwner = "pkg/formats/alm/"

// tileFlagExceptions are the files outside pkg/formats/alm where a 0x2000
// literal means something other than the tile flag.
var tileFlagExceptions = map[string]string{
	"pkg/video/audioprobe_windows_386.go": "the media stream query flag that selects an audio track",
}

// checkTileFlag reports each hex literal of value 0x2000 outside the alm
// package and the exceptions, and each exception that no longer holds one.
func checkTileFlag(files map[string]string) (scanned int, vs []string) {
	held := map[string]bool{}
	for name, src := range files {
		if strings.HasPrefix(name, tileFlagOwner) || strings.HasSuffix(name, "_test.go") {
			continue
		}
		scanned++
		f, err := parser.ParseFile(token.NewFileSet(), name, src, parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, name+": "+err.Error())
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.INT || !strings.HasPrefix(strings.ToLower(lit.Value), "0x") {
				return true
			}
			if v, err := strconv.ParseUint(strings.ReplaceAll(lit.Value[2:], "_", ""), 16, 64); err != nil || v != 0x2000 {
				return true
			}
			if _, ok := tileFlagExceptions[name]; ok {
				held[name] = true
				return true
			}
			vs = append(vs, name+": spells 0x2000; the tile flag is alm.TileImpassable")
			return true
		})
	}
	for name := range tileFlagExceptions {
		if _, ok := files[name]; ok && !held[name] {
			vs = append(vs, name+": listed as a 0x2000 exception but holds no such literal")
		}
	}
	return scanned, vs
}

func TestCheckTileFlag(t *testing.T) {
	cases := []struct {
		name, file, src string
		bad             bool
	}{
		{"the schema", "pkg/formats/alm/schema.go", "package alm\nconst TileImpassable = 0x2000\n", false},
		{"a reader taking the schema", "pkg/ui/a.go", "package ui\nfunc f(w uint16) bool { return alm.Impassable(w) }\n", false},
		{"a reader spelling the bit", "pkg/ui/a.go", "package ui\nfunc f(w uint16) bool { return w&0x2000 != 0 }\n", true},
		{"an upper-case spelling", "pkg/mapedit/a.go", "package mapedit\nconst blocked = 0X02000\n", true},
		{"the listed exception", "pkg/video/audioprobe_windows_386.go", "package video\nconst audioTrackFlag = 0x2000\n", false},
		{"a stale exception", "pkg/video/audioprobe_windows_386.go", "package video\nconst audioTrackFlag = 1\n", true},
		{"another value", "pkg/game/a.go", "package game\nconst c = 0x2001\n", false},
	}
	for _, tc := range cases {
		_, vs := checkTileFlag(map[string]string{tc.file: tc.src})
		if (len(vs) != 0) != tc.bad {
			t.Errorf("%s: violations %v, want bad %v", tc.name, vs, tc.bad)
		}
	}
}

func TestTileFlagIsSpelledOnce(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadDrawnTextSources(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{"pkg/mapedit/", "pkg/mapload/", "pkg/render/terrain/", "pkg/ui/", "pkg/game/"} {
		found := false
		for name := range files {
			found = found || strings.HasPrefix(name, dir)
		}
		if !found {
			t.Fatalf("no source under %s; the scan guards nothing there", dir)
		}
	}
	scanned, vs := checkTileFlag(files)
	if scanned < 100 {
		t.Fatalf("scanned %d sources, want at least 100", scanned)
	}
	for _, v := range vs {
		t.Error(v)
	}
}
