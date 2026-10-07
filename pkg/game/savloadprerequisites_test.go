package game

import (
	"testing"

	"againrom/pkg/formats/sav"
)

func TestOriginalMapNameRetainsRelativeDirectories(t *testing.T) {
	for _, pair := range [][2]string{
		{"scenario/20.alm", "20.alm"},
		{`ScEnArIo\custom\map.alm`, `custom\map.alm`},
		{"20.alm", "20.alm"},
		{"custom/map.alm", "custom/map.alm"},
		{"scenarioX/map.alm", "scenarioX/map.alm"},
		{"", ""},
	} {
		if got := originalMapName(pair[0]); got != pair[1] {
			t.Errorf("map %q became %q; want %q", pair[0], got, pair[1])
		}
	}
}

func TestSavedParticipantRepairUsesPrimaryIdentityAndKeepsScenarioNames(t *testing.T) {
	player := func(name string, participant uint32) sav.DocumentRecordData {
		return sav.DocumentRecordData{Class: "Player", Texts: []sav.DocumentTextData{{Name: "Name", Value: name}},
			Values: []sav.DocumentValueData{{Name: "Hero", Value: 83}, {Name: "Participant", Value: participant}}}
	}
	hero := func(key uint32, name string) sav.DocumentRecordData {
		return sav.DocumentRecordData{Class: "Human", Texts: []sav.DocumentTextData{{Name: "Name", Value: name}},
			Values: []sav.DocumentValueData{{Name: "Identity", Value: key}}}
	}
	doc := sav.DocumentData{World: &sav.DocumentWorldData{}, Players: []uint16{1, 2, 3}, Objects: []sav.DocumentRecordData{
		player("sELf", 0), player("Computer", 1), player("Existing", 0), hero(41, "Companion"), hero(83, "ActualHero"),
	}}
	repairSavedParticipants(&doc)
	for i, want := range []string{"ActualHero", "Computer", "Existing"} {
		if got := doc.Objects[i].Texts[0].Value; got != want {
			t.Errorf("participant %d name %q; want %q", i, got, want)
		}
		latch, _ := savedStructureValue(&doc.Objects[i], "F3D")
		if wantLatch := []uint32{1, 0, 0}[i]; latch != wantLatch {
			t.Errorf("participant %d placement latch %d; want %d", i, latch, wantLatch)
		}
	}
}
