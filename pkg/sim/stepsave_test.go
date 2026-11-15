package sim

import (
	"encoding/binary"
	"testing"
)

// A first-game SAV written during a Speed-24 north-east step carries the
// position the step holds, the octant 1 correction included: stride 16, call
// 8 of 16 leaves x at 15:255 and y at 16:0 (MOVE-174, MOVE-200). Loaded, the
// step continues from those bytes, call by call, to the position the native
// walk reaches (MOVE-201, SAV-1406).
func TestASavedStepCarriesAndContinuesTheCorrectedPosition(t *testing.T) {
	w := mustWorld(t, 1115, Bounds{32, 32}, []Entity{{ID: 7, X: 15, Y: 16, HP: 100, MaxHP: 100, Speed: 24}})
	Step(w, []Command{MoveTo(7, CellPoint{X: 16, Y: 15})})
	for range 7 {
		Step(w, nil)
	}
	e := w.Entities()[0]
	saved, err := ProjectActorMotion(e, SavedActorMotion{Entity: 7}, w.Route(7), true, w.StrideCorrected())
	if err != nil {
		t.Fatal(err)
	}
	word := func(m SavedActorMotion, at int) uint16 { return binary.LittleEndian.Uint16(m.Mover[at:]) }
	if p := saved.Position; p.Cell != 0x100f || p.FineX != 255 || p.FineY != 0 || word(saved, 0xa8) != 24 || word(saved, 0xaa) != 16 ||
		word(saved, 0xac) != 8 || word(saved, 0xae) != 1 || int8(saved.Mover[0xb0]) != 16 || int8(saved.Mover[0xb1]) != -16 {
		t.Fatalf("saved step: position %+v, rate %d, calls %d of %d, octant %d", p, word(saved, 0xa8), word(saved, 0xac), word(saved, 0xaa), word(saved, 0xae))
	}
	if !saved.NativeStrideCompatible(true) || saved.NativeStrideCompatible(false) {
		t.Fatal("the corrected position is not the first game's own stride, or is the second game's")
	}

	loaded, m, cs, bs := savedMotionFixture(t, saved.Position.FineX, saved.Position.FineY, 16, -16, 1, 8, 16)
	m.StaticRoute, m.DynamicRoute, m.ActorAction = nil, []uint16{0x0f10}, 1
	binary.LittleEndian.PutUint16(m.Mover[0xa6:], word(saved, 0xa6))
	binary.LittleEndian.PutUint16(m.Mover[0x80:], word(saved, 0x80))
	binary.LittleEndian.PutUint16(m.Mover[0xa8:], 24)
	cs = append(cs, SavedActorCell{Cell: 0x0f10})
	bs = append(bs, SavedActorBlock{Cell: 0x0f10})
	importSavedMotion(t, loaded, m, cs, bs)
	for call := 9; call < 16; call++ {
		Step(w, nil)
		Step(loaded, nil)
		want, err := ProjectActorMotion(w.Entities()[0], SavedActorMotion{Entity: 7}, w.Route(7), true, w.StrideCorrected())
		if err != nil {
			t.Fatal(err)
		}
		got := loaded.motionFor(7)
		if got.Position.Cell != want.Position.Cell || got.Position.FineX != want.Position.FineX || got.Position.FineY != want.Position.FineY ||
			word(*got, 0xac) != word(want, 0xac) || got.Issue != "" {
			t.Fatalf("call %d: loaded %+v (issue %q), native %+v", call, got.Position, got.Issue, want.Position)
		}
	}
}
