package ui

import (
	"bytes"
	"image"
	"os"
	"testing"
	"time"
)

func TestReleaseDamageNumeralsStartAtTheUnitWithInstalledFont(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("AGAINROM_ASSETS not set")
	}
	v := numeralViewer(t)
	v.SetFont(saveDialogInstallFont(t, root))
	v.SetPeriod(20000)
	v.SetLocalOwner(1)
	hp := 1000
	push(v, at0, numeralUnit(1, 2, hp))
	var origin image.Point
	var glyph []byte
	for hit := 0; hit < 25; hit++ {
		hp -= 7
		push(v, at0.Add(time.Duration(hit)*100*time.Millisecond), numeralUnit(1, 2, hp))
		placements := v.numeralPlacements()
		if len(placements) != len(v.numerals) || len(placements) == 0 {
			t.Fatalf("hit %d: placed %d of %d live numerals", hit, len(placements), len(v.numerals))
		}
		newest := placements[len(placements)-1]
		if hit == 0 {
			origin, glyph = newest.At, bytes.Clone(newest.Pic.Pix)
			painted := 0
			for p := 3; p < len(glyph); p += 4 {
				if glyph[p] != 0 {
					painted++
				}
			}
			if painted == 0 {
				t.Fatal("installed numeral has no painted pixels")
			}
		} else if newest.At != origin || !bytes.Equal(newest.Pic.Pix, glyph) {
			t.Fatalf("hit %d: new numeral at %v, want the original %v and unchanged installed glyph", hit, newest.At, origin)
		}
		if hit > 0 && placements[0].At.Y >= newest.At.Y {
			t.Fatal("previous numeral did not continue its own upward flight")
		}
	}
	t.Logf("25 installed-font damage figures start at %v; earlier figures rise and expire independently", origin)
}
