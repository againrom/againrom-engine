package archtest

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"
	"strconv"
	"strings"
)

// The one-decoder-per-format scan. A format is decoded under pkg/formats and
// nowhere else: production code outside that tree walks no WAV chunk, converts
// no palette entry and resolves no sprite pixel. A fix to a palette, a
// transparency rule or a coverage ramp then reaches every caller at once.
//
// It finds three shapes, type-checked:
//   - a "RIFF", "WAVE" or "fmt " string literal: a RIFF chunk walk;
//   - a keyed R, G, B composite literal whose three values are byte reads, or
//     a pal.Color channel written into a composite literal or a byte slice: a
//     palette or pixel conversion;
//   - a read of a decoded .256 or .16a pixel's fields: a frame conversion.
const decoderRoot = "pkg/formats/"

// decoderTranslations are the files that read decoded sprite pixels into
// another pixel model the render tier owns, not into colours: index-and-hole
// static frames, which the terrain blitters shade by index, and font glyph
// coverage.
var decoderTranslations = map[string]string{
	"pkg/game/statics.go": "terrain.StaticPixel keeps the index for the shade table",
	"pkg/game/font.go":    "text.Pixel keeps the level as glyph coverage",
}

// DecoderComposerDebt is the files the town composer moves onto the decoders.
// The list may only fall: each entry must still hold a finding, and a file
// that holds none must leave the list in the same commit.
var DecoderComposerDebt = map[string]bool{
	"pkg/game/shopart.go":         true,
	"pkg/game/worldmap.go":        true,
	"cmd/schoolcheck/main.go":     true,
	"cmd/townsquarecheck/main.go": true,
}

// DecoderFinding is one site and the shape found there.
type DecoderFinding struct {
	File  string
	Line  int
	Shape string
}

func (f DecoderFinding) String() string { return fmt.Sprintf("%s:%d %s", f.File, f.Line, f.Shape) }

// CheckDecoders turns the findings into violations: a finding outside the
// debt list, and a debt entry with no finding left.
func CheckDecoders(found []DecoderFinding, debt map[string]bool) []string {
	var out []string
	held := map[string]bool{}
	for _, f := range found {
		if debt[f.File] {
			held[f.File] = true
			continue
		}
		out = append(out, f.String()+"; decode it in "+decoderRoot+" and call that decoder")
	}
	names := make([]string, 0, len(debt))
	for name := range debt {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if !held[name] {
			out = append(out, name+" no longer decodes a format itself; remove it from DecoderComposerDebt")
		}
	}
	return out
}

// LoadDecoderFindings walks every production file of the module outside
// pkg/formats.
func LoadDecoderFindings(root string) ([]DecoderFinding, error) {
	m, err := loadTypeCheckedModule(root)
	if err != nil {
		return nil, err
	}
	var out []DecoderFinding
	for _, cp := range m.Packages {
		for i, f := range cp.Files[:cp.NumProd] {
			rel := cp.RelPaths[i]
			if strings.HasPrefix(rel, decoderRoot) || strings.HasPrefix(rel, "internal/archtest/") || decoderTranslations[rel] != "" {
				continue
			}
			out = append(out, decoderFindings(m.Fset, f, cp.Info, rel)...)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out, nil
}

func decoderFindings(fset *token.FileSet, f *ast.File, info *types.Info, rel string) []DecoderFinding {
	var out []DecoderFinding
	add := func(n ast.Node, shape string) {
		out = append(out, DecoderFinding{File: rel, Line: fset.Position(n.Pos()).Line, Shape: shape})
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.BasicLit:
			if n.Kind == token.STRING {
				if s, err := strconv.Unquote(n.Value); err == nil && (s == "RIFF" || s == "WAVE" || s == "fmt ") {
					add(n, "walks RIFF chunks")
				}
			}
		case *ast.CompositeLit:
			if byteChannels(n) {
				add(n, "converts byte-order colour channels")
			}
			for _, e := range n.Elts {
				if kv, ok := e.(*ast.KeyValueExpr); ok && isPaletteChannel(info, kv.Value) {
					add(kv, "converts a palette colour")
				}
			}
		case *ast.AssignStmt:
			for i, lhs := range n.Lhs {
				if _, ok := lhs.(*ast.IndexExpr); ok && i < len(n.Rhs) && isPaletteChannel(info, n.Rhs[i]) {
					add(n, "writes a palette colour into pixels")
				}
			}
		case *ast.SelectorExpr:
			if isSpritePixelField(info, n) {
				add(n, "resolves a decoded sprite pixel")
			}
		}
		return true
	})
	return out
}

// byteChannels is a keyed literal whose R, G and B are all byte reads with R
// at a higher offset than B: the blue-green-red storage order reversed.
func byteChannels(lit *ast.CompositeLit) bool {
	offset := map[string]int{}
	for _, e := range lit.Elts {
		kv, ok := e.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok || (key.Name != "R" && key.Name != "G" && key.Name != "B") {
			continue
		}
		ix, ok := kv.Value.(*ast.IndexExpr)
		if !ok {
			return false
		}
		offset[key.Name] = indexOffset(ix.Index)
	}
	return len(offset) == 3 && offset["R"] > offset["B"]
}

// indexOffset is the constant part of an index: N for N or e+N, else 0.
func indexOffset(e ast.Expr) int {
	if b, ok := e.(*ast.BinaryExpr); ok && b.Op == token.ADD {
		e = b.Y
	}
	if lit, ok := e.(*ast.BasicLit); ok && lit.Kind == token.INT {
		n, _ := strconv.Atoi(lit.Value)
		return n
	}
	return 0
}

// isPaletteChannel is a read of the R, G or B field of a pal.Color value.
func isPaletteChannel(info *types.Info, e ast.Expr) bool {
	sel, ok := ast.Unparen(e).(*ast.SelectorExpr)
	if !ok || (sel.Sel.Name != "R" && sel.Sel.Name != "G" && sel.Sel.Name != "B") {
		return false
	}
	return isNamed(info.TypeOf(sel.X), "againrom/pkg/formats/pal", "Color")
}

// isSpritePixelField is a read of a field of a decoded .256 or .16a pixel.
func isSpritePixelField(info *types.Info, sel *ast.SelectorExpr) bool {
	t := info.TypeOf(sel.X)
	return isNamed(t, "againrom/pkg/formats/spr256", "Pixel") || isNamed(t, "againrom/pkg/formats/spr16", "PixelA")
}

func isNamed(t types.Type, pkg, name string) bool {
	if t == nil {
		return false
	}
	t = types.Unalias(t)
	if p, ok := t.(*types.Pointer); ok {
		t = types.Unalias(p.Elem())
	}
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj != nil && obj.Pkg() != nil && obj.Pkg().Path() == pkg && obj.Name() == name
}
