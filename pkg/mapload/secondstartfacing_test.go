package mapload_test

import (
	"slices"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// secondFacingScript is an empty second-game script; a start given it builds
// a second-game World.
func secondFacingScript(t *testing.T) *sim.Script {
	t.Helper()
	s, _, err := mapload.CompileROM2ScriptFrom(alm.Script{}, mapload.ScriptRefs{})
	if err != nil || s.Dialect() != sim.ScriptROM2 {
		t.Fatalf("second-game script: %v", err)
	}
	return s
}

// secondFacingMap is the facing fixture with five placed actors: the join
// fixture's Human and four creatures beside it.
func secondFacingMap(t *testing.T) *alm.Map {
	t.Helper()
	m := facingMap(t)
	for i := range 4 {
		m.Units = append(m.Units, alm.Unit{X: uint32(14+i)<<8 | 0x80, Y: 0x0E80, ClassID: joinCreatureKey, ClassSubID: 1,
			UnitID: uint16(20 + i), Owner: 4})
	}
	return m
}

func secondFacingStart(t *testing.T, m *alm.Map, s *sim.Script, draws *sim.Draws) (*sim.World, mapload.Start) {
	t.Helper()
	w, st, err := mapload.StartCampaignMissionScriptedWith(m, joinTable(), mapload.DifficultyNormal,
		joinParty(), 0, s, draws)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	return w, st
}

// crtRange is the claim's range draw written out from its text: the core
// s = 214013 s + 2531011 yields (s >> 16) & 0x7fff, and the draw over 0..n
// is (core * (n + 1)) >> 15, with no core call for n = 0 (R2-ENGINE-959).
func crtRange(s *uint32, n int32) int32 {
	if n == 0 {
		return 0
	}
	*s = *s*214013 + 2531011
	return int32((int64(*s>>16&0x7fff) * int64(n+1)) >> 15)
}

// TestSecondGameStartDrawsEachPlacedActorsFacingInRecordOrder: a second-game
// start builds each map actor through Unit initialization, which stores
// 0x40 plus a 0..128 draw as the facing and copies it to the heading, one
// draw per actor in record order, from the stream the mission seeded; the
// join walk's drop pick follows (R2-ENGINE-959, R2-ENGINE-960). The expected
// facings come from the claim's arithmetic at the same state.
func TestSecondGameStartDrawsEachPlacedActorsFacingInRecordOrder(t *testing.T) {
	cells := mapload.DropCells(secondFacingMap(t))
	for _, seed := range []uint32{0x1234567, 0x89abcdef, 7} {
		s := seed
		var want []uint8
		for range secondFacingMap(t).Units {
			want = append(want, uint8(0x40+crtRange(&s, 0x80)))
		}
		drop := cells[crtRange(&s, int32(len(cells)-1))]
		w, st := secondFacingStart(t, secondFacingMap(t), secondFacingScript(t), sim.NewOriginalDraws(seed))
		if w.Script() == nil || w.Script().Dialect() != sim.ScriptROM2 {
			t.Fatal("the start built no second-game World")
		}
		var got []uint8
		for _, e := range w.Entities()[:len(want)] {
			got = append(got, e.Facing)
			if e.DesiredFacing != e.Facing || e.Facing < 0x40 || e.Facing > 0xC0 {
				t.Errorf("seed %#x: actor %d faces %#x heading %#x, want 0x40..0xC0 and equal", seed, e.ID, e.Facing, e.DesiredFacing)
			}
		}
		if !slices.Equal(got, want) {
			t.Errorf("seed %#x: facings %#x, want %#x", seed, got, want)
		}
		if st.Drop != drop {
			t.Errorf("seed %#x: drop %v, want %v after %d facing draws", seed, st.Drop, drop, len(want))
		}
		again, _ := secondFacingStart(t, secondFacingMap(t), secondFacingScript(t), sim.NewOriginalDraws(seed))
		if again.Hash() != w.Hash() {
			t.Errorf("seed %#x: two starts at one state differ", seed)
		}
	}
}

// TestSecondGameSeededStartKeysEachFacingAndKeepsTheStream: in seeded mode a
// second-game start gives each map actor the facing the first game's start
// of the same map gives it, in 0x40..0xC0 and equal to its heading, and its
// drop pick and World stream are those of the same map with no placed actor.
func TestSecondGameSeededStartKeysEachFacingAndKeepsTheStream(t *testing.T) {
	second, st := secondFacingStart(t, secondFacingMap(t), secondFacingScript(t), nil)
	first, _ := secondFacingStart(t, secondFacingMap(t), nil, nil)
	n := len(secondFacingMap(t).Units)
	distinct := map[uint8]bool{}
	for i, e := range second.Entities()[:n] {
		if e.Facing < 0x40 || e.Facing > 0xC0 || e.DesiredFacing != e.Facing {
			t.Errorf("actor %d faces %#x heading %#x, want 0x40..0xC0 and equal", e.ID, e.Facing, e.DesiredFacing)
		}
		if f := first.Entities()[i].Facing; e.Facing != f {
			t.Errorf("actor %d faces %#x, the first game's start %#x", e.ID, e.Facing, f)
		}
		distinct[e.Facing] = true
	}
	if len(distinct) < 2 {
		t.Errorf("%d actors share one facing", n)
	}
	bare := secondFacingMap(t)
	bare.Units = nil
	none, sn := secondFacingStart(t, bare, secondFacingScript(t), nil)
	if st.Drop != sn.Drop || second.RandomState() != none.RandomState() {
		t.Errorf("the placed actors moved the seeded draws: drop %v/%v, World stream %#x/%#x",
			st.Drop, sn.Drop, second.RandomState(), none.RandomState())
	}
	if hero := second.Entities()[st.IDs[0]]; hero.Facing != 0 || hero.DesiredFacing != 0 {
		t.Errorf("the party hero faces %#x, want 0 (R2-SESSION-363)", hero.Facing)
	}
}
