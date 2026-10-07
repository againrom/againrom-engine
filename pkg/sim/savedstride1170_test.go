package sim

import (
	"encoding/binary"
	"strings"
	"testing"
)

func TestSavedStride1170QualifiedBoundaryAndNearMatchGuards(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*SavedActorMotion)
	}{
		{"qualified", nil},
		{"position", func(m *SavedActorMotion) { m.Position.FineX-- }},
		{"packed-position", func(m *SavedActorMotion) { m.Position.PackedCell++ }},
		{"old-cell", func(m *SavedActorMotion) { m.Mover[0xa6]++ }},
		{"accepted-cell", func(m *SavedActorMotion) { m.Mover[0x80]++ }},
		{"rate", func(m *SavedActorMotion) { m.Mover[0xa8]++ }},
		{"wide-rate", func(m *SavedActorMotion) { m.Mover[0xa9]++ }},
		{"duration", func(m *SavedActorMotion) { m.Mover[0xaa]++ }},
		{"elapsed", func(m *SavedActorMotion) { m.Mover[0xac]-- }},
		{"signed-step", func(m *SavedActorMotion) { m.Mover[0xb0]-- }},
		{"direction", func(m *SavedActorMotion) { m.Mover[0xae]++ }},
		{"wide-direction", func(m *SavedActorMotion) { m.Mover[0xaf]++ }},
		{"turn-flag", func(m *SavedActorMotion) { m.Mover[0xa0] = 1 }},
		{"turn-counter", func(m *SavedActorMotion) { m.Mover[0x9d] = 1 }},
		{"turn-estimate", func(m *SavedActorMotion) { m.Mover[0xa4] = 1 }},
		{"static-route", func(m *SavedActorMotion) { m.StaticRoute = []uint16{0x1010} }},
		{"dynamic-center", func(m *SavedActorMotion) { m.DynamicRoute = []uint16{0x1011} }},
		{"actor-action", func(m *SavedActorMotion) { m.ActorAction = 0 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Independent wire operands: east from 15:128, three 32/256
			// steps paid out of eight, then centers 16 and 17. The older
			// fixture deliberately carries a different +a6/static/action
			// shape; its original callback-guard witnesses stay unchanged.
			w, m, cs, bs := motionFixture1115(t, 224, 128, 32, 0, 2, 3, 8)
			m.StaticRoute, m.DynamicRoute, m.ActorAction = nil, []uint16{0x1010, 0x1011}, 1
			binary.LittleEndian.PutUint16(m.Mover[0xa6:], 0x100f)
			if tc.mutate != nil {
				tc.mutate(&m)
			}
			if m.NativeStrideCompatible() != (tc.mutate == nil) {
				t.Fatal("stride qualification admitted a near-match or refused the complete tuple")
			}
			importMotion1115(t, w, m, cs, bs)
			before := w.motionFor(7)
			if !before.Active {
				if before.Issue == "" || tc.mutate == nil {
					t.Fatal("unqualified import lost its existing admission guard")
				}
				return
			}
			for range 3 {
				Step(w, nil)
				if w.motionFor(7).Position.Cell != 0x100f {
					break
				}
			}
			got := w.motionFor(7)
			if got.Position.Cell == 0x100f {
				t.Fatal("guard control never crossed its boundary")
			}
			if tc.mutate == nil {
				if got.Issue != "" || !got.NativeStrideCompatible() {
					t.Fatal("qualified crossing acquired an original callback refusal", got.Issue)
				}
				for range 4 {
					Step(w, nil)
				}
				if got.Current || len(w.Route(7)) != 1 || w.Route(7)[0] != [2]int32{17, 16} {
					t.Fatal("completed crossing lost its remaining native route")
				}
			} else if !strings.Contains(got.Issue, "original boundary speed callback") {
				t.Fatal("near-match bypassed the original boundary callback guard", got.Issue)
			}
		})
	}
}

func TestSavedStride1170QualifiedDiagonalKeepsNativeBoundary(t *testing.T) {
	// Native NE at rate 12 has signed axis steps +8/-8. Its sixteenth
	// sample lands exactly on both cell boundaries; the original mover's
	// diagonal zero-corner adjustment would move X back by 1/256.
	w, m, cs, bs := motionFixture1115(t, 248, 8, 8, -8, 1, 15, 32)
	m.StaticRoute, m.DynamicRoute, m.ActorAction = nil, []uint16{0x0f10}, 1
	binary.LittleEndian.PutUint16(m.Mover[0xa6:], 0x100f)
	binary.LittleEndian.PutUint16(m.Mover[0x80:], 0x0f10)
	binary.LittleEndian.PutUint16(m.Mover[0xa8:], 12)
	cs = append(cs, SavedActorCell{Cell: 0x0f10})
	bs = append(bs, SavedActorBlock{Cell: 0x0f10})
	importMotion1115(t, w, m, cs, bs)
	for tick := 16; tick <= 32; tick++ {
		if !w.motionFor(7).NativeStrideCompatible() {
			t.Fatalf("qualified NE stride lost its operands before sample %d", tick)
		}
		Step(w, nil)
		got := w.motionFor(7)
		want := [2]int32{15*256 + 128 + int32(tick)*8, 16*256 + 128 - int32(tick)*8}
		actual := [2]int32{int32(got.Position.Cell&255)*256 + int32(got.Position.FineX), int32(got.Position.Cell>>8)*256 + int32(got.Position.FineY)}
		if actual != want || got.Issue != "" {
			t.Fatalf("native NE sample %d: %v want %v; issue %q", tick, actual, want, got.Issue)
		}
	}
}
