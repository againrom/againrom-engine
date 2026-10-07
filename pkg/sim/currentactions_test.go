package sim

import (
	"encoding/binary"
	"testing"
)

func TestCurrentMoveKeepsIndependentOrderCursor(t *testing.T) {
	for _, mode := range []string{"with-mover", "without-mover"} {
		t.Run(mode, func(t *testing.T) {
			w := importedRelocationWorld1115(t)
			if mode == "without-mover" {
				w.savedMotion = nil
			}
			binary.LittleEndian.PutUint16(w.savedOrder(1).Raw[2:], 0x0507)
			Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 9, Y: 2}})
			o := w.savedOrder(1)
			if binary.LittleEndian.Uint16(o.Raw[2:]) != 0x0507 || binary.LittleEndian.Uint16(o.Raw[10:]) != 0x0209 {
				t.Fatal("Move confused its destination with the independent patrol cursor", o.Raw[2:4], o.Raw[10:12])
			}
			mustMarshal(t, w)
		})
	}
}
