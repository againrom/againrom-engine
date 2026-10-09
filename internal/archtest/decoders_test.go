package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"strings"
	"testing"
)

// TestLiveDecodersStayInTheFormatsTree measures the real tree: no production
// file outside pkg/formats decodes a format, except the composer's falling
// debt list, each entry of which must still hold a finding.
func TestLiveDecodersStayInTheFormatsTree(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	found, err := LoadDecoderFindings(root)
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	for _, v := range CheckDecoders(found, DecoderComposerDebt) {
		t.Error(v)
	}
}

// TestDecoderFindingsSeeEachShape holds the scan against synthetic sources, so
// a scan that went blind fails here rather than passing a dirty tree.
func TestDecoderFindingsSeeEachShape(t *testing.T) {
	// A stub of image/color: a -trimpath test binary has no GOROOT to import
	// the real one from.
	const imgcolor = `package color
type RGBA struct{ R, G, B, A uint8 }`
	const pal = `package pal
type Color struct{ R, G, B uint8 }`
	const spr = `package spr256
type Pixel struct {
	Index  uint8
	Opaque bool
}`
	const user = `package user

import (
	"image/color"

	"againrom/pkg/formats/pal"
	"againrom/pkg/formats/spr256"
)

func wav(b []byte) bool { return string(b[:4]) == "RIFF" }

func bgr(e []byte) color.RGBA { return color.RGBA{R: e[2], G: e[1], B: e[0], A: 255} }

func entry(c pal.Color) color.RGBA { return color.RGBA{R: c.R, G: c.G, B: c.B} }

func write(pix []uint8, c pal.Color) { pix[0] = c.R }

func frame(p spr256.Pixel) bool { return p.Opaque }

func compare(a, b pal.Color) bool { return a.R == b.R && a == b }
`
	fset := token.NewFileSet()
	pkgs := map[string]*types.Package{}
	check := func(path, src string) (*ast.File, *types.Info) {
		f, err := parser.ParseFile(fset, path+".go", src, 0)
		if err != nil {
			t.Fatal(err)
		}
		info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Selections: map[*ast.SelectorExpr]*types.Selection{}}
		conf := types.Config{Importer: importerFunc(func(p string) (*types.Package, error) {
			if pk, ok := pkgs[p]; ok {
				return pk, nil
			}
			return nil, fmt.Errorf("no stub for %q", p)
		})}
		pk, err := conf.Check(path, fset, []*ast.File{f}, info)
		if err != nil {
			t.Fatal(err)
		}
		pkgs[path] = pk
		return f, info
	}
	check("image/color", imgcolor)
	check("againrom/pkg/formats/pal", pal)
	check("againrom/pkg/formats/spr256", spr)
	f, info := check("user", user)

	got := decoderFindings(fset, f, info, "user.go")
	want := []string{"walks RIFF chunks", "converts byte-order colour channels",
		"converts a palette colour", "converts a palette colour", "converts a palette colour",
		"writes a palette colour into pixels", "resolves a decoded sprite pixel"}
	if len(got) != len(want) {
		t.Fatalf("findings = %v, want %d of them", got, len(want))
	}
	for i, w := range want {
		if got[i].Shape != w {
			t.Errorf("finding %d = %v, want %q", i, got[i], w)
		}
	}
}

type importerFunc func(string) (*types.Package, error)

func (f importerFunc) Import(p string) (*types.Package, error) { return f(p) }

// TestCheckDecodersHoldsTheDebtListFalling: a finding outside the list fails,
// a listed file keeps its place only while it still holds a finding.
func TestCheckDecodersHoldsTheDebtListFalling(t *testing.T) {
	debt := map[string]bool{"a.go": true, "b.go": true}
	got := CheckDecoders([]DecoderFinding{{File: "a.go", Line: 1, Shape: "x"}, {File: "c.go", Line: 2, Shape: "y"}}, debt)
	if len(got) != 2 || !strings.HasPrefix(got[0], "c.go:2 y") || !strings.HasPrefix(got[1], "b.go no longer") {
		t.Fatalf("violations = %v", got)
	}
	if got := CheckDecoders([]DecoderFinding{{File: "a.go"}, {File: "b.go"}}, debt); len(got) != 0 {
		t.Fatalf("a clean tree = %v", got)
	}
}
