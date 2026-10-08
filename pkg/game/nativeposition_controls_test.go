package game

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/sim"
)

func TestNativePositionAcceptedStrideLossControls(t *testing.T) {
	e := sim.Entity{X: 68, Y: 36, Transit: 8, TransitTotal: 14, Stride: sim.NativeStride{Present: true, FromX: 69, FromY: 35, ToX: 68, ToY: 36, StepX: -19, StepY: 19}}
	raw := make([]byte, 12)
	binary.LittleEndian.PutUint16(raw, 69|35<<8)
	binary.LittleEndian.PutUint16(raw[2:], 69|35<<8)
	raw[4], raw[5] = 14, 242
	if differences := unitNativePositionDifferences(e, raw); len(differences) != 0 {
		t.Fatal("accepted destination confused with near position", differences)
	}
	for _, control := range []struct {
		name string
		edit func(*sim.Entity, []byte)
	}{
		{"destination", func(e *sim.Entity, _ []byte) { e.X++ }},
		{"from", func(e *sim.Entity, _ []byte) { e.Stride.FromX++ }},
		{"step", func(e *sim.Entity, _ []byte) { e.Stride.StepX++ }},
		{"elapsed", func(e *sim.Entity, _ []byte) { e.Transit++ }},
		{"invalid interval", func(e *sim.Entity, _ []byte) { e.TransitTotal = 0 }},
		{"missing stride", func(e *sim.Entity, _ []byte) { e.Stride.Present = false }},
		{"raw near", func(_ *sim.Entity, p []byte) { p[0]++ }},
		{"raw packed", func(_ *sim.Entity, p []byte) { p[2]++ }},
		{"raw fine", func(_ *sim.Entity, p []byte) { p[4]++ }},
	} {
		t.Run(control.name, func(t *testing.T) {
			changed, position := e, append([]byte(nil), raw...)
			control.edit(&changed, position)
			if len(unitNativePositionDifferences(changed, position)) == 0 {
				t.Fatal("accepted corrupted current or ordinary position")
			}
		})
	}
	e.Transit, e.TransitTotal, e.Stride = 0, 0, sim.NativeStride{}
	e.X, e.Y = 69, 35
	if differences := unitNativePositionDifferences(e, raw); len(differences) != 0 {
		t.Fatal("stationary coordinate differs", differences)
	}
	e.X++
	if len(unitNativePositionDifferences(e, raw)) == 0 {
		t.Fatal("stationary position loss accepted")
	}
}
