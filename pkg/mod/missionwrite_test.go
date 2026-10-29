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

func TestManifestTitleWriterKeepsCommentsAndMetadata(t *testing.T) {
	src := []byte("id = 'm'\r\ntitle = 'Old' # keep\r\nversion = '7'\r\nauthor = 'Owner'\r\nentry = 'custom.star'\r\n")
	got, err := WriteManifestTitle(src, "Новый")
	want := bytes.Replace(src, []byte("'Old'"), []byte(`"Новый"`), 1)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("manifest: %v\n%s", err, got)
	}
}
