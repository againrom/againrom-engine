package game

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/formats/alm"
	"againrom/pkg/ui"
)

func TestEachGameNamesItsMusicDescription(t *testing.T) {
	for _, p := range base.Profiles {
		d := GameMusic(p)
		if d == nil || d.Music != string(p.GameOf()) {
			t.Errorf("%s: music description %v", p.ID, d)
		}
	}
	if GameMusic(base.Profile{}).Archive.Path != "Allods/"+MusicArchive {
		t.Errorf("first game archive %q", GameMusic(base.Profile{}).Archive.Path)
	}
}

func typeTwelve(records ...[7]int32) []byte {
	var out []byte
	for _, r := range records {
		for _, v := range r {
			out = binary.LittleEndian.AppendUint32(out, uint32(v))
		}
	}
	return out
}

// The head joins the area array only when its first theme names an entry.
func TestMissionMusicAreasAdmitTheHeadByItsFirstTheme(t *testing.T) {
	area := [7]int32{40, 41, 6, 1, -1, -1, -1}
	m := &alm.Map{}
	m.Extension[2] = typeTwelve([7]int32{0, 0, 0, 4, 5, -1, -1}, area)
	got := missionMusicAreas(m)
	want := []ui.MusicArea{{Themes: [4]int32{4, 5, -1, -1}}, {X: 40, Y: 41, Radius: 6, Themes: [4]int32{1, -1, -1, -1}}}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("areas %+v, want %+v", got, want)
	}
	m.Extension[2] = typeTwelve([7]int32{0, 0, 0, -1, -1, -1, -1}, area)
	if got := missionMusicAreas(m); len(got) != 1 || got[0] != want[1] {
		t.Fatalf("areas without an admitted head %+v", got)
	}
	if got := missionMusicAreas(&alm.Map{}); got != nil {
		t.Fatalf("a first-game map answered areas %+v", got)
	}
}
