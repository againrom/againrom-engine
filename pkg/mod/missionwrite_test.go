package mod

import (
	"bytes"
	"strings"
	"testing"
)

func TestMissionWriterKeepsSourceAndChangesOnlyTheSelectedValue(t *testing.T) {
	source := []byte("\xef\xbb\xbf" + strings.ReplaceAll(goodMission, "gold     = 50", "gold     = 50 # purse"))
	before, err := ParseMissions("m", MissionsFile, source, missionText, "ru")
	if err != nil {
		t.Fatal(err)
	}
	got, err := WriteMissions(source, before.Missions, before.Missions)
	if err != nil || !bytes.Equal(got, source) {
		t.Fatalf("no-op changed source: %v\n%s", err, got)
	}
	after := []Mission{CloneMission(before.Missions[0])}
	after[0].Gold = 73
	got, err = WriteMissions(source, before.Missions, after)
	want := bytes.Replace(source, []byte("50 # purse"), []byte("73 # purse"), 1)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("value edit: %v\n%s", err, got)
	}
	if _, err := ParseMissions("m", MissionsFile, got, missionText, "ru"); err != nil {
		t.Fatal(err)
	}
}

func TestMissionWriterPartyOptionalsAndReorder(t *testing.T) {
	empty := ""
	a := Mission{Kind: MissionGuest, Key: "first", Number: 25, Map: "a.alm", After: 20, Party: []GuestMember{{Character: "npc22", Weapon: &empty, Armour: []string{}}}}
	b := Mission{Kind: MissionGuest, Key: "second", Number: 26, Map: "b.alm", After: 25, Party: []GuestMember{{Character: "sellsword", Level: 2}}}
	src, err := WriteMissions(nil, nil, []Mission{a, b})
	if err != nil {
		t.Fatal(err)
	}
	d, err := ParseMissions("m", MissionsFile, src, nil, "en")
	if err != nil {
		t.Fatal(err)
	}
	if d.Missions[0].Party[0].Armour == nil || d.Missions[0].Party[0].Weapon == nil || d.Missions[1].Party[0].Weapon != nil {
		t.Fatal("lost absence or empty")
	}
	reordered, err := WriteMissions(src, d.Missions, []Mission{d.Missions[1], d.Missions[0]})
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := ParseMissions("m", MissionsFile, reordered, nil, "en")
	if err != nil || reopened.Missions[0].Key != "second" || reopened.Missions[0].After != 25 {
		t.Fatalf("reorder changed chain: %v", err)
	}
	clone := CloneMission(d.Missions[0])
	*clone.Party[0].Weapon = "changed"
	if *d.Missions[0].Party[0].Weapon != "" {
		t.Fatal("clone aliases optional")
	}
}

// A removed middle row leaves with its header and pairs alone: the blank
// line and the comment that head the next row stay, in the missions file
// and in the towns file the same writer serves.
func TestRowWriterRemovingAMiddleRowKeepsTheNextRowsComment(t *testing.T) {
	a := Mission{Kind: MissionGuest, Key: "first", Number: 25, Map: "a.alm", After: 20, Party: []GuestMember{{Character: "npc22"}}}
	b := Mission{Kind: MissionGuest, Key: "second", Number: 26, Map: "b.alm", After: 25, Party: []GuestMember{{Character: "npc23"}}}
	c := Mission{Kind: MissionGuest, Key: "third", Number: 27, Map: "c.alm", After: 26, Party: []GuestMember{{Character: "npc24"}}}
	plain, err := WriteMissions(nil, nil, []Mission{a, b, c})
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(string(plain), "[[mission]]\n")
	if len(parts) != 4 || parts[0] != "" {
		t.Fatalf("three tables expected:\n%s", plain)
	}
	middle := "[[mission]]\n" + parts[2]
	src := "# head\n[[mission]]\n" + parts[1] + "\n# The second.\n" + middle + "\n# The third.\n[[mission]]\n" + parts[3]
	d, err := ParseMissions("m", MissionsFile, []byte(src), nil, "en")
	if err != nil {
		t.Fatal(err)
	}
	got, err := WriteMissions([]byte(src), d.Missions, []Mission{d.Missions[0], d.Missions[2]})
	want := strings.Replace(src, middle, "", 1)
	if err != nil || string(got) != want {
		t.Fatalf("missions: %v\n%s\nwant\n%s", err, got, want)
	}

	const towns = "# The tavern.\n[[tavern]]\ntown = 1\nchapter = 30\nnpc = 8\nmission = 32\n\n# The shop.\n[[shop]]\ntown = 1\nchapter = 30\nmission = 31\n\n# keep me: the school row\n[[school]]\ntown = 1\nchapter = 30\nmission = 32\n"
	_, before, err := ParseTowns("m", TownsFile, []byte(towns))
	if err != nil {
		t.Fatal(err)
	}
	got, err = WriteOfferRows([]byte(towns), before, []OfferRecord{before[0], before[2]})
	want = strings.Replace(towns, "[[shop]]\ntown = 1\nchapter = 30\nmission = 31\n", "", 1)
	if err != nil || string(got) != want {
		t.Fatalf("towns: %v\n%s\nwant\n%s", err, got, want)
	}
}

func TestManifestTitleWriterKeepsCommentsAndMetadata(t *testing.T) {
	src := []byte("id = 'm'\r\ntitle = 'Old' # keep\r\nversion = '7'\r\nauthor = 'Owner'\r\nentry = 'custom.star'\r\n")
	got, err := WriteManifestTitle(src, "Новый")
	want := bytes.Replace(src, []byte("'Old'"), []byte(`"Новый"`), 1)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("manifest: %v\n%s", err, got)
	}
}
