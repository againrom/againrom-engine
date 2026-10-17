package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestLiveCellGeometryStaysInItsFamily measures the real tree: no production
// file of pkg/ui or pkg/render outside the projection family computes a cell
// corner or centre beyond its committed count.
func TestLiveCellGeometryStaysInItsFamily(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	found, err := LoadCellProjectionFindings(root)
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	owned := 0
	for _, f := range found {
		if f.File == "pkg/ui/cellprojection.go" {
			owned++
		}
	}
	if owned == 0 {
		t.Fatal("pkg/ui/cellprojection.go shows no cell geometry; the scan or the family moved")
	}
	for _, v := range CheckCellProjection(found, CellProjectionDebt) {
		t.Error(v)
	}
}

// TestCellProjectionFindingsSeeEachShape holds the scan against synthetic
// sources: each removed site of the old editor and viewer, and the shapes that
// are not cell geometry.
func TestCellProjectionFindingsSeeEachShape(t *testing.T) {
	const src = `package ui

func a(c, r int) (int, int) { return c*32+16, r*32+16 }
func b(c int) float64 { return float64(c) * terrain.CellSize }
func d(c int) int { return CellSize * c }
func e(p *terrain.Projection) int { x, _ := p.WorldCorner(1, 2); return x + p.AnchorHeight(1, 2) }
func f(g float64) float64 { return g * 32.0 }
func h(n int) int { return n * 31 + n/32 }
`
	found, err := CellProjectionFindings("pkg/ui/x.go", src)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 6 {
		t.Fatalf("found %d sites, want 6: %v", len(found), found)
	}
	if v := CheckCellProjection(found, nil); len(v) != 1 || !strings.Contains(v[0], "cellprojection.go") {
		t.Fatalf("an unlisted file: %v", v)
	}
	if v := CheckCellProjection(found, map[string]int{"pkg/ui/x.go": 6}); len(v) != 0 {
		t.Fatalf("at its count: %v", v)
	}
	if v := CheckCellProjection(found, map[string]int{"pkg/ui/x.go": 7}); len(v) != 1 || !strings.Contains(v[0], "below its count") {
		t.Fatalf("below its count: %v", v)
	}
	owned, _ := CellProjectionFindings("pkg/ui/cellprojection.go", src)
	if v := CheckCellProjection(owned, nil); len(v) != 0 {
		t.Fatalf("the family file is counted: %v", v)
	}
}
