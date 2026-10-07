package sav

import "testing"

func TestDocumentCell1115LiteralPayload(t *testing.T) {
	var p [52]byte
	for i := range p {
		p[i] = byte(i + 1)
	}
	c := DocumentCellFromPayload(0x5432, p)
	if c.Cell != 0x5432 || c.Cost != 1 || c.Static != 2 || c.LayerCount != 3 || c.Residue03 != 4 ||
		c.GroundActor != 0x08070605 || c.AirActor != 0x0c0b0a09 || c.Building != 0x100f0e0d || c.Sack != 0x14131211 ||
		c.Layers != ([6]uint32{0x18171615, 0x1c1b1a19, 0x201f1e1d, 0x24232221, 0x28272625, 0x2c2b2a29}) ||
		c.Operation != 45 || c.Power != 46 || c.SourceX != 47 || c.SourceY != 48 || c.TargetX != 49 || c.TargetY != 50 || c.Residue32 != 0x3433 {
		t.Fatalf("payload fields differ: %+v", c)
	}
	if got := c.Payload(); got != p {
		t.Fatalf("literal payload changed: %x", got)
	}
	// Every source bit participates independently, including unnamed residue.
	for i := range p {
		for bit := uint(0); bit < 8; bit++ {
			var one [52]byte
			one[i] = 1 << bit
			if got := DocumentCellFromPayload(0xffff, one).Payload(); got != one {
				t.Fatalf("lost payload bit %d:%d", i, bit)
			}
		}
	}
}
