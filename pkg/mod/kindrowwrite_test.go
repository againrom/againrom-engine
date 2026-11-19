package mod

import (
	"bytes"
	"strings"
	"testing"
)

// The missions kind writes through its column the bytes the row writer
// gives, with a before and with none.
func TestMissionsKindWritesTheRowWriterBytes(t *testing.T) {
	source := []byte("\xef\xbb\xbf" + strings.ReplaceAll(goodMission, "gold     = 50", "gold     = 50 # purse"))
	before, err := ParseMissions("m", MissionsFile, source, missionText, "ru")
	if err != nil {
		t.Fatal(err)
	}
	after := []Mission{CloneMission(before.Missions[0])}
	after[0].Gold = 73
	kind, ok := WrittenKind(MissionsFile)
	if !ok {
		t.Fatal("the missions kind has no writer")
	}
	for _, c := range []struct {
		name          string
		source        []byte
		before, after []Mission
	}{{"edit", source, before.Missions, after}, {"same", source, before.Missions, before.Missions}, {"none", nil, nil, after}} {
		want, wantErr := WriteMissions(c.source, c.before, c.after)
		got, err := kind.Write(MissionsFile, c.source, Data{Missions: MissionData{Missions: c.after}, Before: DataBefore{Missions: c.before}})
		if (err != nil) != (wantErr != nil) || !bytes.Equal(got, want) {
			t.Fatalf("%s: kind wrote\n%s\nrow writer wrote\n%s", c.name, got, want)
		}
	}
}

// The towns kind writes through its column the bytes the offer writer gives.
func TestTownsKindWritesTheOfferWriterBytes(t *testing.T) {
	const towns = "# The tavern.\n[[tavern]]\ntown = 1\nchapter = 30\nnpc = 8\nmission = 32\n\n# The shop.\n[[shop]]\ntown = 1\nchapter = 30\nmission = 31\n\n# keep me: the school row\n[[school]]\ntown = 1\nchapter = 30\nmission = 32\n"
	_, before, err := ParseTowns("m", TownsFile, []byte(towns))
	if err != nil {
		t.Fatal(err)
	}
	kind, ok := WrittenKind(TownsFile)
	if !ok {
		t.Fatal("the towns kind has no writer")
	}
	for _, after := range [][]OfferRecord{before, {before[0], before[2]}} {
		want, wantErr := WriteOfferRows([]byte(towns), before, after)
		got, err := kind.Write(TownsFile, []byte(towns), Data{Towns: TownData{Offers: after}, Before: DataBefore{Offers: before}})
		if (err != nil) != (wantErr != nil) || !bytes.Equal(got, want) {
			t.Fatalf("kind wrote\n%s\nrow writer wrote\n%s", got, want)
		}
	}
}
