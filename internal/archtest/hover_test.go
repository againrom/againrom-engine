package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHoverHasOneRectangleWalkAndHotspotSource(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"hoverhit.go", "inspection.go", "missioninput.go", "command.go", "castcursor.go", "cursor.go"} {
		b, err := os.ReadFile(filepath.Join(root, "pkg", "ui", name))
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{"targetAt(", "inspectionAt(", "inspectionPixel(", "hoverSpriteBrightness", "AttackPointerHotspot"} {
			if strings.Contains(string(b), forbidden) {
				t.Errorf("%s retains %s", name, forbidden)
			}
		}
	}
	for name, required := range map[string]string{
		"hoverhit.go":       "func registeredHover(",
		"inspection.go":     "return v.hoverAt(v.cursorX, v.cursorY)",
		"castcursor.go":     "ref, hit := v.hoverAt(x, y)",
		"command.go":        "hit:       v.hover.hit.subject",
		"headlessentity.go": "return v.InspectionPoint(InspectionSubject{InspectionUnit, id})",
		"headless.go":       "v.InspectionPoint(InspectionSubject{InspectionUnit, id})",
	} {
		b, err := os.ReadFile(filepath.Join(root, "pkg", "ui", name))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Count(string(b), required) != 1 {
			t.Errorf("%s must contain one %s", name, required)
		}
	}
}
