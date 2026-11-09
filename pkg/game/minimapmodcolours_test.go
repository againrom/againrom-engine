package game

import (
	"image/color"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/mod"
)

// The picker's dressing and an install with no lookup take the mod groups'
// colours from the one paint: a row with its own colour keeps it, a row
// without one stays out, and the lookup never reaches the picker. Loss
// control: a table with no mod terrain gives no paint.
func TestModGroupColoursReachAViewerWithNoPaintData(t *testing.T) {
	own := color.RGBA{R: 7, G: 180, B: 90, A: 255}
	table := &mapload.Table{}
	table.Mods.Terrain = mod.TerrainData{Groups: []mod.TerrainGroup{
		{Group: 20, Minimap: own, HasMinimap: true},
		{Group: 21},
	}}
	in := &InstallResources{Table: table}
	p := missionMinimapPaint(in)
	if p == nil || p.Colours != nil {
		t.Fatalf("paint %+v, want a paint with no lookup", p)
	}
	if p.GroupColours[20] != own {
		t.Errorf("own colour row: %v, want %v", p.GroupColours[20], own)
	}
	if p.GroupColours[21] != (color.RGBA{}) {
		t.Errorf("a row without a colour and no lookup: %v, want none", p.GroupColours[21])
	}
	picker := pickerMinimapPaint(in)
	if picker == nil || picker.Colours != nil || picker.GroupColours != p.GroupColours {
		t.Fatalf("picker paint %+v, want only the group colours %v", picker, p.GroupColours)
	}
	empty := &InstallResources{Table: &mapload.Table{}}
	if missionMinimapPaint(empty) != nil || pickerMinimapPaint(empty) != nil {
		t.Error("loss control: a table with no mod terrain produced a paint")
	}
}
