package game

import (
	"slices"
	"testing"

	"againrom/pkg/ui"
)

// Over every installed map, each unit the speaker pick refuses carries a
// cause the pick can word, a unit shown carries a record of the speaker list,
// and a unit refused only because another unit is shown first names a record
// of the list and a cell inside the map. The census of the install's causes
// has the shown units too: no map of the install is refused whole.
func TestReleaseMapEditorSpeakerPickNamesWhyEveryRefusedUnit(t *testing.T) {
	x := installedInspector(t)
	var ids []int64
	for _, c := range x.npcChoices(255, true) {
		ids = append(ids, c.Value)
	}
	var shown, refused int
	causes := map[ui.SpeakerCause]int{}
	for _, me := range x.Maps {
		if me.Err != nil {
			continue
		}
		key := ui.MapSourceKey(ui.SourceLoose, me.Source)
		if me.FromArchive {
			key = ui.MapSourceKey(ui.SourceScenario, me.Source)
		}
		d, err := x.Open(key)
		if err != nil {
			continue
		}
		s := d.Edits.(*mapInspectionEdits)
		authored, err := s.model.Map()
		if err != nil {
			continue
		}
		for i := range authored.Units {
			id, why := s.UnitSpeaker(i)
			causes[why.Cause]++
			switch why.Cause {
			case ui.SpeakerShown:
				shown++
				if !slices.Contains(ids, id) {
					t.Fatalf("%s unit %d is shown by speaker %d, not in the speaker list", key, i, id)
				}
			case ui.SpeakerNoMission:
				t.Fatalf("%s unit %d: the mission start of an installed map was not built", key, i)
			case ui.SpeakerNotFirst:
				refused++
				if !slices.Contains(ids, why.Record) || why.Cell.X < 0 || why.Cell.Y < 0 || why.Cell.X >= authored.Width || why.Cell.Y >= authored.Height {
					t.Fatalf("%s unit %d: record %d at %v is not a record of the list inside the map", key, i, why.Record, why.Cell)
				}
			default:
				refused++
			}
		}
	}
	if shown == 0 || refused == 0 {
		t.Fatalf("the install's units: %d shown, %d refused, causes %v", shown, refused, causes)
	}
	t.Logf("%d shown, %d refused, by cause %v", shown, refused, causes)
}
