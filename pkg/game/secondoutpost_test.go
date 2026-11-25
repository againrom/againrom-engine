package game

import (
	"bytes"
	"testing"
)

// A second-game mission whose map opens the type-10 gate builds the knights'
// Outpost at mission start. Our SAVE writes no Outpost field: the file equals
// the one of the same mission without the gate and the entry, and a LOAD of
// it holds the Outpost in its loaded state, no member, group 0 and reload
// 120 (R2-SESSION-502, R2-SESSION-503).
func TestSecondMissionOutpostSavesNothingAndLoadsEmpty(t *testing.T) {
	c := secondGuardMission()
	c.gate, c.outposts = 4, [][4]uint32{{5, 7, 2, 0}}
	f, app, screen, _ := secondBagFrontOf(t, secondBagSample(), c)
	secondBagEnter(t, f, app, screen)
	built := f.live.world.Outposts()
	if len(built) != 1 || built[0].Owner != 2 || built[0].Group != 5 || built[0].Reload != 7 || len(built[0].Members) != 2 {
		t.Fatalf("mission start Outposts = %+v, want Player 2's Group 5 with reload 7", built)
	}
	raw := secondPageSave(t, f, app)

	plain := secondGuardMission()
	pf, papp, pscreen, _ := secondBagFrontOf(t, secondBagSample(), plain)
	secondBagEnter(t, pf, papp, pscreen)
	if len(pf.live.world.Outposts()) != 0 {
		t.Fatalf("a map without the gate built %+v", pf.live.world.Outposts())
	}
	if !bytes.Equal(raw, secondPageSave(t, pf, papp)) {
		t.Fatal("the SAVE of the Outpost mission differs from the one without it")
	}

	cold := secondMissionLoadBytesOf(t, c, raw)
	if cold.live == nil {
		t.Fatal("the file does not LOAD")
	}
	loaded := cold.live.world.Outposts()
	if len(loaded) != 1 || loaded[0].Owner != 2 || loaded[0].Group != 0 || loaded[0].Reload != 120 || loaded[0].Counting || len(loaded[0].Members) != 0 {
		t.Fatalf("LOAD Outposts = %+v, want the loaded state", loaded)
	}
}
