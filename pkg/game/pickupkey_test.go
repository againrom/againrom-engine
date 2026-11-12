package game

import (
	"image"
	"slices"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

func TestMissionPickupKeyUsesTheProfileAndSpeaker(t *testing.T) {
	for _, tc := range []struct {
		name     string
		game     base.Game
		key      string
		selected bool
		grabs    int
		voices   []int16
	}{
		{"second P selected", base.GameROM2, "p", true, 1, []int16{1}},
		{"second P empty", base.GameROM2, "p", false, 0, nil},
		{"second F selected", base.GameROM2, "f", true, 1, nil},
		{"first P selected", base.GameROM1, "p", true, 0, nil},
		{"first F selected", base.GameROM1, "f", true, 1, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mw := voiceFixture(t, heroMan)
			if tc.game == base.GameROM2 {
				secondVoiceMission(mw, []mapload.PartyMember{voiceHeroMember}, nil)
			}
			f, voices := voiceFront(mw, secondVoiceBank("characters/aldor/pickup/1.wav"))
			containers, err := vfs.Open(nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			f.Archives = &Archives{Containers: containers, Base: base.Match{Profile: base.Profile{Game: tc.game}}}
			f.SetDeterministicFrames(true)
			a := f.App("pickup key")
			script := &scriptedDraw{}
			command, _ := f.runtimeAudio().unitReplies(mw, script.draw)
			mw.view.SetCommandAcknowledgment(command)
			mw.view.SetLocalOwner(sim.SelfSlot)
			mw.view.SetEntities([]ui.MapEntity{{ID: 1, Cell: image.Pt(0, 0), Owner: sim.SelfSlot,
				Life: ui.LifeAlive, HP: 10, MaxHP: 10, PlayerCharacter: true}})
			grabs := 0
			grab := func(id uint32, x, y int, action ui.GrabAction) {
				grabs++
				wantID, wantAction := uint32(0), ui.GrabUnderfoot
				if tc.key == "p" {
					wantID, wantAction = 1, ui.GrabCollect
				}
				if id != wantID || x != 0 || y != 0 || action != wantAction {
					t.Errorf("key grab = (%d,%d,%d,%v), want (%d,0,0,%v)", id, x, y, action, wantID, wantAction)
				}
			}
			if err := a.OpenMission(func() (*ui.Viewer, ui.MapTick, ui.MapOrder, ui.MapCadence, ui.MapAffect, ui.MapAdvance, ui.MapAttack, ui.MapGrab, ui.MapStance, ui.MapMarch, error) {
				return mw.view, nil, nil, nil, nil, nil, nil, grab, nil, nil, nil
			}); err != nil {
				t.Fatal(err)
			}
			if tc.selected {
				if err := a.HeadlessKey("e"); err != nil {
					t.Fatal(err)
				}
				if got := a.HeadlessSelection(); !slices.Equal(got, []uint32{1}) {
					t.Fatalf("selection %v, want [1]", got)
				}
			}
			if err := a.HeadlessKey(tc.key); err != nil {
				t.Fatal(err)
			}
			if got := heard(voices, 0); grabs != tc.grabs || !slices.Equal(got, tc.voices) {
				t.Fatalf("grabs %d, recordings %v; want %d, %v", grabs, got, tc.grabs, tc.voices)
			}
		})
	}
}
