package game

import (
	"testing"

	"againrom/pkg/sim"
)

// Under the show-map cheat a second-game creature card is drawn at the full
// level, a first-game one still follows its Diary (R2-ENGINE-1269).
func TestShowMapDrawsTheSecondGameCardFull(t *testing.T) {
	for _, tc := range []struct {
		name   string
		second bool
		want   int
	}{{"second game", true, sim.KnowledgeFull}, {"first game", false, 0}} {
		t.Run(tc.name, func(t *testing.T) {
			var script *sim.Script
			if tc.second {
				var err error
				if script, err = sim.NewROM2Script(nil, nil, nil); err != nil {
					t.Fatal(err)
				}
			}
			hero := sim.Entity{ID: 1, Class: 35, TypeID: 35, X: 2, Y: 2, HP: 100, MaxHP: 100, ScanRange: 5, Owner: sim.SelfSlot, TokenSize: 1}
			monster := sim.Entity{ID: 2, Class: 1, TypeID: 66, X: 6, Y: 6, HP: 50, MaxHP: 50, ScanRange: 5, Owner: 3, TokenSize: 1, MapUnitID: 2}
			w, err := sim.NewSpelledWorld(5, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, nil, []sim.Entity{hero, monster}, script, nil)
			if err != nil {
				t.Fatal(err)
			}
			w.SetDiaryUnits(119, map[uint16]sim.DiaryUnit{2: {Row: 70, Face: 2}})
			mw := &mapWorld{world: w}
			if got := mw.cardKnowledge(monster); got != 0 {
				t.Fatalf("without the cheat: level %d, want 0", got)
			}
			mw.cheats.showMap = true
			if got := mw.cardKnowledge(monster); got != tc.want {
				t.Fatalf("with show map: level %d, want %d", got, tc.want)
			}
		})
	}
}
