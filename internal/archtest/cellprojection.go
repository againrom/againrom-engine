package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// The cell projection scan. A map cell's corners and centre, in flat and in
// displaced mode, come from one family: cellSpace in pkg/ui/cellprojection.go,
// over terrain.Projection in pkg/render/terrain/project.go. In the production
// files of pkg/ui and pkg/render, outside those two files, it finds two
// shapes:
//   - a multiplication by the cell size (an identifier or field named CellSize,
//     or the integer literal 32): a cell coordinate turned into world pixels;
//   - a call of WorldCorner or AnchorHeight: a mesh corner or a cell's anchor
//     height read past the family.
var cellProjectionRoots = []string{"pkg/ui", "pkg/render"}

// CellProjectionOwners are the two files that compute cell geometry.
var CellProjectionOwners = map[string]string{
	"pkg/ui/cellprojection.go":      "the viewer's cell projection family",
	"pkg/render/terrain/project.go": "the displaced mesh the family reads",
}

// CellProjectionFinding is one site and the shape found there.
type CellProjectionFinding struct {
	File  string
	Line  int
	Shape string
}

func (f CellProjectionFinding) String() string {
	return fmt.Sprintf("%s:%d %s", f.File, f.Line, f.Shape)
}

const (
	shapeCellSizeProduct = "multiplies by the cell size"
	shapeMeshRead        = "reads a mesh corner or anchor height"
)

// CheckCellProjection reports a file whose findings rose above its committed
// count, and a file whose count fell (lower CellProjectionDebt in that commit).
func CheckCellProjection(found []CellProjectionFinding, debt map[string]int) []string {
	held := map[string]int{}
	first := map[string]CellProjectionFinding{}
	for _, f := range found {
		if CellProjectionOwners[f.File] != "" {
			continue
		}
		if held[f.File] == 0 {
			first[f.File] = f
		}
		held[f.File]++
	}
	files := map[string]bool{}
	for name := range held {
		files[name] = true
	}
	for name := range debt {
		files[name] = true
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var out []string
	for _, name := range names {
		switch got, want := held[name], debt[name]; {
		case got > want:
			out = append(out, fmt.Sprintf("%s holds %d cell-geometry sites, above its count of %d (first %s); take a cell corner or centre from cellSpace in pkg/ui/cellprojection.go",
				name, got, want, first[name]))
		case got < want:
			out = append(out, fmt.Sprintf("%s holds %d cell-geometry sites, below its count of %d; lower CellProjectionDebt in this commit", name, got, want))
		}
	}
	return out
}

// LoadCellProjectionFindings reads every production file under pkg/ui and
// pkg/render.
func LoadCellProjectionFindings(root string) ([]CellProjectionFinding, error) {
	var out []CellProjectionFinding
	for _, dir := range cellProjectionRoots {
		seen := 0
		err := filepath.WalkDir(filepath.Join(root, filepath.FromSlash(dir)), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			found, err := CellProjectionFindings(filepath.ToSlash(rel), string(src))
			if err != nil {
				return err
			}
			seen++
			out = append(out, found...)
			return nil
		})
		if err != nil {
			return nil, err
		}
		if seen == 0 {
			return nil, fmt.Errorf("%s holds no Go file; the scan went blind", dir)
		}
	}
	return out, nil
}

// CellProjectionFindings scans one source file named rel.
func CellProjectionFindings(rel, src string) ([]CellProjectionFinding, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, rel, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	var out []CellProjectionFinding
	add := func(pos token.Pos, shape string) {
		out = append(out, CellProjectionFinding{File: rel, Line: fset.Position(pos).Line, Shape: shape})
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.BinaryExpr:
			if n.Op == token.MUL && (isCellSize(n.X) || isCellSize(n.Y)) {
				add(n.OpPos, shapeCellSizeProduct)
			}
		case *ast.CallExpr:
			if sel, ok := n.Fun.(*ast.SelectorExpr); ok && (sel.Sel.Name == "WorldCorner" || sel.Sel.Name == "AnchorHeight") {
				add(n.Pos(), shapeMeshRead)
			}
		}
		return true
	})
	return out, nil
}

// isCellSize reports whether e is the cell size: an identifier or field named
// CellSize, or the integer literal 32.
func isCellSize(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.ParenExpr:
		return isCellSize(e.X)
	case *ast.Ident:
		return e.Name == "CellSize"
	case *ast.SelectorExpr:
		return e.Sel.Name == "CellSize"
	case *ast.BasicLit:
		return e.Kind == token.INT && e.Value == "32"
	}
	return false
}
