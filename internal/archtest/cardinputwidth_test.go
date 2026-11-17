package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestUnitCardInputWidthHasNoHandDigitCount keeps the card's input width on
// the field's bound: no hand digit count beside the field table.
func TestUnitCardInputWidthHasNoHandDigitCount(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("..", "..", "pkg", "ui", "mapeditorunitcard.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(src)
	for _, bad := range []string{"digits:", "digits int"} {
		if strings.Contains(text, bad) {
			t.Errorf("mapeditorunitcard.go holds a hand digit count %q", bad)
		}
	}
	if !strings.Contains(text, ".ValueWidth") {
		t.Error("mapeditorunitcard.go does not read InspectionField.ValueWidth")
	}
}
