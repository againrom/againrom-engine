package data

import (
	"fmt"
	"testing"

	"againrom/pkg/rules"
)

// The first game's rule functions as written before the table, kept as an
// independent oracle: the first set must answer them for every identifier.
func oracleFlight(picture, dist int) int {
	if dist < 0 {
		dist = 0
	}
	switch picture {
	case 10:
		return atLeastOne(dist / 200)
	case 12:
		return atLeastOne(dist / 384)
	case 20, 30:
		return 1
	case 34, 36:
		return 13
	case 60:
		return 21
	}
	return 0
}

func oracleTrail(picture int) int {
	switch picture {
	case 10:
		return 0
	case 12:
		return 1
	}
	return -1
}

func oracleBurstLife(picture int) int {
	switch picture {
	case 27:
		return 18
	case 13:
		return 22
	}
	return 16
}

func TestTheFirstSetIsTheFirstGamesRulesUnchanged(t *testing.T) {
	s := FirstSpellPresentation()
	for id := -2; id <= 80; id++ {
		for _, dist := range []int{-5, 0, 1, 199, 200, 383, 384, 1000, 5000} {
			if got, want := s.Flight(int32(id), dist, 9), oracleFlight(id, dist); got != want {
				t.Errorf("flight %d at %d: %d, want %d", id, dist, got, want)
			}
		}
		if got, want := s.Travels(int32(id)), id == 10 || id == 12; got != want {
			t.Errorf("travels %d: %v", id, got)
		}
		if got, want := s.DrawsPath(int32(id)), id == 34 || id == 36; got != want {
			t.Errorf("path %d: %v", id, got)
		}
		if got, want := s.TrailSlot(int32(id)), oracleTrail(id); got != want {
			t.Errorf("trail %d: %d, want %d", id, got, want)
		}
		if got, want := s.CellLife(int32(id)), oracleBurstLife(id); got != want {
			t.Errorf("burst life %d: %d, want %d", id, got, want)
		}
		if got, want := s.Effect(int32(id)).Hidden, id == 20; got != want {
			t.Errorf("hidden %d: %v", id, got)
		}
		if got, want := s.Effect(int32(id)).SecondRecord, id == 60; got != want {
			t.Errorf("second record %d: %v", id, got)
		}
		for frame := 0; frame < 20; frame++ {
			if drawn, raise := s.DrawnFrame(int32(id), frame); drawn != frame || raise != 0 {
				t.Errorf("drawn frame %d at %d: %d raised %d", id, frame, drawn, raise)
			}
		}
	}
	for row := -1; row <= 30; row++ {
		_, overlay := s.OverlayPicture(row)
		if want := row == 3 || row == 7 || row == 8 || row == 19; overlay != want {
			t.Errorf("overlay row %d: %v", row, overlay)
		}
		cast, wantCast := s.CastSound(row), 0
		slot, delay := s.CellSound(row)
		wantSlot, wantDelay := 0, 0
		if row >= 1 && row <= 28 {
			wantCast, wantSlot = 500+2*row+8, 500+2*row+9
			if wantSlot == 551 {
				wantDelay = 8
			}
		}
		if cast != wantCast || slot != wantSlot || delay != wantDelay {
			t.Errorf("sounds row %d: %d %d %d", row, cast, slot, delay)
		}
	}
}

// secondSetMismatches checks s against R2-ENGINE-445..448 and R2-ENGINE-394
// and names every miss.
func secondSetMismatches(s *SpellPresentation) []string {
	var miss []string
	check := func(ok bool, format string, args ...any) {
		if !ok {
			miss = append(miss, fmt.Sprintf(format, args...))
		}
	}
	for row := 1; row <= 29; row++ {
		slot, delay := s.CellSound(row)
		wantDelay := 0
		if row == 7 {
			wantDelay = 9
		}
		check(s.CastSound(row) == 500+2*row+8, "cast sound row %d", row)
		check(slot == 500+2*row+9 && delay == wantDelay, "cell sound row %d", row)
	}
	check(s.CastSound(30) == 0, "row 30 voices")
	// R2-ENGINE-447: trails, the chain, Heal, Blizzard's 16 updates.
	check(s.TrailSlot(10) == 0 && s.TrailSlot(12) == 1 && s.TrailSlot(18) == -1, "trails")
	check(s.Flight(30, 0, 35) == 13 && s.DrawsPath(30) && s.Effect(30).Chain, "chain")
	check(s.DrawsPath(28) && !s.DrawsPath(34) && !s.DrawsPath(36), "paths")
	check(s.Effect(56).Hidden && !s.Effect(20).Hidden, "hidden")
	check(s.CellLife(23) == 16, "Blizzard cell life")
	// The falling piece: frame 8 raised 28-4f while f < 8, then frame f-8.
	for frame := 0; frame < 16; frame++ {
		wantDrawn, wantRaise := frame-8, 0
		if frame < 8 {
			wantDrawn, wantRaise = 8, 28-4*frame
		}
		drawn, raise := s.DrawnFrame(23, frame)
		check(drawn == wantDrawn && raise == wantRaise, "Blizzard frame %d", frame)
	}
	for _, id := range []int32{10, 13, 21, 25, 28, 54} {
		drawn, raise := s.DrawnFrame(id, 3)
		check(drawn == 3 && raise == 0, "unlifted %d", id)
	}
	// R2-ENGINE-448: the light byte map's writers.
	for _, id := range []int32{10, 18, 40} {
		e := s.Effect(id)
		check(e.LightKind == LightPoint && e.Light == LightStep{0, 16}, "light %d", id)
	}
	e := s.Effect(12)
	check(e.LightKind == LightPoint && e.Light == LightStep{1, 16}, "light 12")
	check(s.Effect(13).LightKind == LightBurst && s.Effect(28).LightKind == LightPath && s.Effect(30).LightKind == LightPath, "light 13 28 30")
	for _, id := range []int32{15, 16, 20, 21, 23, 24, 27, 34, 43, 46, 48, 54, 56, 60, 62, 64} {
		check(s.Effect(id).LightKind == LightNone, "light %d", id)
	}
	want13 := []LightStep{{1, 16}, {2, 8}, {3, 0}, {3, 0}, {3, 8}, {3, 16}, {3, 24}, {3, 32}, {3, 40}, {3, 46}, {3, 46}, {3, 0}, {3, 0}}
	for frame, w := range want13 {
		got, ok := s.BurstLightAt(frame)
		check(ok && got == w, "burst light frame %d", frame)
	}
	check(s.PathLightStep == 10, "path light step")
	// R2-ENGINE-445: the travelling casts carry distance lives.
	check(s.Travels(10) && s.Travels(12) && s.Travels(18) && s.Travels(40) && !s.Travels(28), "travel")
	return miss
}

func TestTheSecondSetFollowsItsClaims(t *testing.T) {
	if miss := secondSetMismatches(SpellPresentationOf(rules.SpellGameSecond)); len(miss) != 0 {
		t.Fatalf("misses: %v", miss)
	}
	// Loss control: the same rows read one spell row off.
	shifted := secondSpellPresentation
	shifted.effects = nil
	for _, e := range secondSpellPresentation.effects {
		e.ID += 2
		shifted.effects = append(shifted.effects, e)
	}
	if miss := secondSetMismatches(&shifted); len(miss) == 0 {
		t.Fatal("a set shifted by one row still passes")
	}
	// Loss control: a fall one step steeper misses.
	steep := secondSpellPresentation
	steep.effects = nil
	for _, e := range secondSpellPresentation.effects {
		if e.FallFrames > 0 {
			e.FallStep++
		}
		steep.effects = append(steep.effects, e)
	}
	if miss := secondSetMismatches(&steep); len(miss) == 0 {
		t.Fatal("a steeper fall still passes")
	}
	// Loss control: Blizzard's cell sound one update early misses.
	early := secondSpellPresentation
	early.effects = nil
	for _, e := range secondSpellPresentation.effects {
		if e.SoundDelay > 0 {
			e.SoundDelay--
		}
		early.effects = append(early.effects, e)
	}
	if miss := secondSetMismatches(&early); len(miss) == 0 {
		t.Fatal("an earlier Blizzard cell sound still passes")
	}
	// Loss control: the first set is not the second.
	if miss := secondSetMismatches(FirstSpellPresentation()); len(miss) == 0 {
		t.Fatal("the first set passes the second game's claims")
	}
}

func TestTheChosenSecondLivesAreData(t *testing.T) {
	s := SpellPresentationOf(rules.SpellGameSecond)
	for _, tc := range []struct {
		id           int32
		dist, phases int
		want         int
	}{
		{10, 1000, 4, 5}, {12, 1000, 4, 2}, {18, 1000, 6, 5}, {40, 0, 4, 1},
		{28, 5000, 5, 13}, {16, 0, 6, 6}, {48, 0, 5, 5}, {54, 0, 21, 21}, {60, 0, 9, 9},
		{56, 0, 7, 1}, {8, 1000, 9, 0},
	} {
		if got := s.Flight(tc.id, tc.dist, tc.phases); got != tc.want {
			t.Errorf("flight %d: %d, want %d", tc.id, got, tc.want)
		}
	}
	if !s.Effect(54).SecondRecord || s.Effect(60).SecondRecord {
		t.Error("second record")
	}
	for row, want := range map[int]bool{3: true, 6: true, 17: true, 7: false, 8: false, 19: false} {
		if _, got := s.OverlayPicture(row); got != want {
			t.Errorf("overlay row %d: %v", row, got)
		}
	}
}

func TestAnEditionNamesItsSet(t *testing.T) {
	for name, want := range map[string]rules.SpellGame{"rom1": rules.SpellGameFirst, "rom2": rules.SpellGameSecond, "": rules.SpellGameFirst} {
		if got := SpellPresentationNamed(name).Game; got != want {
			t.Errorf("%q: %d, want %d", name, got, want)
		}
	}
}
