package game

import (
	"testing"

	"againrom/pkg/ui"
)

// The palette asks for one identity's picture at a time; each creature keeps
// its own picture whichever was asked first.
func TestEditorIdentityPaletteShowsEveryCreaturePicture(t *testing.T) {
	s, _ := unitIdentityEditor(t)
	for _, order := range [][]int{{1, 3}, {3, 1}} {
		s.inspector.recordPictures = nil
		s.inspector.identityChoices(s.mission())
		for _, row := range order {
			if s.NewRecordPreview(ui.KindUnit, "identity", identityValue(identityCreature, row)) == nil {
				t.Errorf("order %v: creature row %d has no palette picture", order, row)
			}
		}
	}
}
