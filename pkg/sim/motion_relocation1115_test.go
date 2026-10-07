package sim

import (
	"encoding/binary"
	"testing"
)

func importedRelocationWorld1115(t *testing.T) *World {
	t.Helper()
	w := strideRelocationWorld1115(t)
	o := SavedActorOrder{Entity: 1, State: 0xb}
	o.Raw[8], o.Raw[9] = 1, 3
	if err := w.ImportSavedGroups(nil, []SavedActorOrder{o}); err != nil {
		t.Fatal(err)
	}
	m := SavedActorMotion{Entity: 1, Position: SavedActorPosition{Cell: 0x0203, PackedCell: 0x0203, FineX: 224, FineY: 128}}
	m.Mover[0], m.Mover[1] = 64, 64
	for _, at := range []int{0x80, 0xa6} {
		binary.LittleEndian.PutUint16(m.Mover[at:], 0x0204)
	}
	binary.LittleEndian.PutUint16(m.Mover[0xaa:], 8)
	binary.LittleEndian.PutUint16(m.Mover[0xac:], 3)
	binary.LittleEndian.PutUint16(m.Mover[0xae:], 2)
	m.Mover[0xb0] = 32
	cs := []SavedActorCell{{Cell: 0x0203, Ground: SavedActorSlot{Key: 0x1004, Entity: 1, Bound: true}}, {Cell: 0x0204}}
	binary.LittleEndian.PutUint32(cs[0].Payload[4:], 0x1004)
	importMotion1115(t, w, m, cs, []SavedActorBlock{{Cell: 0x0203, Dyn: 0x40}, {Cell: 0x0204, Dyn: 0x40}})
	return w
}

func TestSavedMotion1115EveryNativeRelocationInvalidatesCurrentAuthority(t *testing.T) {
	for _, kind := range []string{"unit-effect", "point-book", "point-scroll", "unit-scroll", "script-place", "off-map", "headless", "death"} {
		t.Run(kind, func(t *testing.T) {
			w := importedRelocationWorld1115(t)
			switch kind {
			case "unit-effect":
				if !w.ordinaryEffectPayload(0, 1, w.spells[0], 30) {
					t.Fatal("effect failed")
				}
			case "point-book":
				if !w.castBookAt(0, 7, 5, 26, nil) {
					t.Fatal("book release failed")
				}
			case "point-scroll":
				w.releaseScroll(0, ScrollCast{Caster: 1, AtCell: true, X: 7, Y: 5}, w.spells[0], 30, nil)
			case "unit-scroll":
				w.releaseScroll(0, ScrollCast{Caster: 1, Target: 2, X: 7, Y: 5}, w.spells[0], 30, nil)
			case "script-place":
				if !w.placeAt(0, 7, 5) {
					t.Fatal("placement failed")
				}
			case "off-map":
				if !w.takeOffMap(0) {
					t.Fatal("off-map failed")
				}
			case "headless":
				if err := w.HeadlessPlace(1, 7, 5); err != nil {
					t.Fatal(err)
				}
			case "death":
				if err := w.HeadlessKill(1); err != nil {
					t.Fatal(err)
				}
			}
			if m := w.motionFor(1); m.Current || m.Active || m.Issue == "" {
				t.Fatal("native writer claims stale current motion", m)
			}
			if _, _, ok := w.ActorFinePosition(1); ok {
				t.Fatal("UI retains original fine authority")
			}
			_ = mustMarshal(t, w)
		})
	}
}

func TestSavedMotion1115ActiveBlocksAutomaticCastButNotOneFineUpdate(t *testing.T) {
	w := importedRelocationWorld1115(t)
	if !w.actorActionBusy(0) || !w.retainedCastBusy(0) {
		t.Fatal("active crossing admits conflicting action")
	}
	if w.beginBookSpellAt(0, 7, 5, 26) {
		t.Fatal("ordinary admission interrupted imported movement")
	}
	Step(w, nil)
	if len(w.bookCasts) != 0 || w.entities[0].Facing != 64 || w.motionFor(1).Position.Cell != 0x0204 || w.motionFor(1).Position.FineX != 0 {
		t.Fatal("cast stole or suppressed admitted crossing")
	}
}

func TestSavedMotion1115FootprintsAndAirSlotsStayDistinct(t *testing.T) {
	for _, air := range []bool{false, true} {
		w, m, _, _ := motionFixture1115(t, 224, 128, 32, 0, 2, 3, 8)
		w.entities[0].TokenSize = 2
		layer, mask := 0, byte(0x40)
		if air {
			w.entities[0].Domain = DomainAir
			layer, mask = 1, 0x80
		}
		var cs []SavedActorCell
		var bs []SavedActorBlock
		for y := 16; y <= 17; y++ {
			for x := 15; x <= 17; x++ {
				c := SavedActorCell{Cell: uint16(y*256 + x)}
				if x <= 16 {
					*motionSlot(&c, layer) = SavedActorSlot{Key: 0x1004, Entity: 7, Bound: true}
					binary.LittleEndian.PutUint32(c.Payload[4+4*layer:], 0x1004)
				}
				cs = append(cs, c)
				bs = append(bs, SavedActorBlock{Cell: c.Cell, Dyn: mask})
			}
		}
		importMotion1115(t, w, m, cs, bs)
		Step(w, nil)
		scratch := newRouteScratch(w)
		for y := int32(16); y <= 17; y++ {
			for x := int32(15); x <= 17; x++ {
				at, _ := w.cellIndex(x, y)
				if scratch.at(layer, at) != 1 || scratch.at(1-layer, at) != 0 {
					t.Fatalf("air%t cell%d,%d occupancy not union of slots and reservations", air, x, y)
				}
				ground, above := w.cellLayerOccupants(x, y)
				own, other := ground, above
				if air {
					own, other = above, ground
				}
				want := 0
				if x >= 16 {
					want = 1
				}
				if len(own) != want || len(other) != 0 {
					t.Fatal("footprint slot layer differs")
				}
			}
		}
		for range 4 {
			Step(w, nil)
		}
		scratch.occupy(w)
		for y := int32(16); y <= 17; y++ {
			at, _ := w.cellIndex(15, y)
			if scratch.at(layer, at) != 0 {
				t.Fatal("old footprint reserved after center")
			}
		}
		_ = mustMarshal(t, w)
	}
}
