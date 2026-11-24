package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBrushPlaneRuleHasOneHome keeps a brush plane's rule and edge ring in the
// format row of pkg/mapedit: no second table in pkg/game and no plane-name
// branch in Band.
func TestBrushPlaneRuleHasOneHome(t *testing.T) {
	check := func(name string, bad ...string) {
		src, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		for _, b := range bad {
			if strings.Contains(string(src), b) {
				t.Errorf("%s holds a second brush plane rule: %q", name, b)
			}
		}
	}
	check("pkg/mapedit/brush.go", "p == PlaneHeight ||")
	check("pkg/game/mapeditrow.go", "secondPaints", "planeRule", "paints:")
	check("pkg/game/mapinspectionbrush.go", "paints[", ".row().paints")
}
