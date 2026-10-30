package game

import (
	"bytes"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
)

// writtenPlayerGroupSizes is the actor count of each group the town document's
// single Player lists.
func writtenPlayerGroupSizes(t *testing.T, raw []byte) []int {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	p, err := file.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	city := p.Data()
	if len(city.Players) != 1 || city.Objects[city.Players[0]-1].Player == nil {
		t.Fatal("town SAV has no single Player")
	}
	var sizes []int
	for _, g := range city.Objects[city.Players[0]-1].Player.Groups {
		sizes = append(sizes, len(g.Actors))
	}
	return sizes
}

// A committed town SAVE of a campaign that never LOADed its groups does not
// write a group whose members all left (SAV-1251: the town's Player has +28
// zero), and leaves the live group list as it was. A cold LOAD of the file
// SAVEs the same bytes.
func TestNeverLoadedTownSaveDropsAnEmptiedGroupAndKeepsTheTown(t *testing.T) {
	f := currentTrainingCity(t)
	var members []string
	for _, m := range f.Carried {
		members = append(members, m.ID)
	}
	f.Town.cityGroupsLoaded = false
	f.Town.cityGroups = []cityLiveGroup{
		{Payload: cityHireGroup(), Members: []string{"left"}},
		{Payload: cityConstructedGroup(), Members: members},
	}
	before := liveGroupIDs(f.Town.cityGroups)
	var raw []byte
	if err := f.captureSave(captureCommitted, false, func(s Snapshot, label string) error {
		var err error
		raw, err = f.ExportOriginalSave(s, label)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if got := liveGroupIDs(f.Town.cityGroups); !reflect.DeepEqual(got, before) || f.Town.cityGroupsLoaded {
		t.Fatalf("SAVE changed the live town groups: %v -> %v", before, got)
	}
	if got := writtenPlayerGroupSizes(t, raw); !reflect.DeepEqual(got, []int{len(members)}) {
		t.Fatalf("written group sizes %v, want one group of %d", got, len(members))
	}
	cold := &FrontEnd{InstallResources: f.InstallResources, RuntimeServices: f.RuntimeServices}
	if _, town, err := cold.RestoreOriginal(raw); err != nil || !town {
		t.Fatal("cold city LOAD", town, err)
	}
	s, label, err := cold.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	again, err := cold.ExportOriginalSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(again, raw) {
		t.Fatalf("the SAVE of the cold LOAD differs: %d bytes, want %d", len(again), len(raw))
	}
}
