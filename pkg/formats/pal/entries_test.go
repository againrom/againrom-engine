package pal_test

import (
	"image/color"
	"testing"

	"againrom/pkg/formats/pal"
)

func TestEntriesReadsBGRAndDropsTheReservedByteAndATail(t *testing.T) {
	got := pal.Entries([]byte{3, 2, 1, 99, 6, 5, 4, 98, 7})
	want := []pal.Color{{R: 1, G: 2, B: 3}, {R: 4, G: 5, B: 6}}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("Entries = %+v, want %+v", got, want)
	}
	px := pal.Pixels([]byte{3, 2, 1, 6, 5, 4, 7, 8})
	if len(px) != 2 || px[0] != want[0] || px[1] != want[1] {
		t.Fatalf("Pixels = %+v, want %+v", px, want)
	}
}

func TestOpaqueTableOfAndPicture(t *testing.T) {
	if c := (pal.Color{R: 1, G: 2, B: 3}).Opaque(); c != (color.RGBA{R: 1, G: 2, B: 3, A: 0xff}) {
		t.Fatalf("Opaque = %+v", c)
	}
	cols := make([]pal.Color, pal.EntryCount)
	cols[255] = pal.Color{B: 9}
	tbl, ok := pal.TableOf(cols)
	if !ok || tbl.Opaque()[255] != (color.RGBA{B: 9, A: 0xff}) || tbl.Opaque()[0] != (color.RGBA{A: 0xff}) {
		t.Fatalf("TableOf/Opaque = %v %+v", ok, tbl.Opaque()[255])
	}
	if _, ok := pal.TableOf(cols[:255]); ok {
		t.Fatal("a 255-entry palette made a table")
	}
	pic := pal.Picture(2, 1, []color.RGBA{{R: 1, A: 2}, {G: 3, A: 4}, {B: 5, A: 6}})
	if pic.RGBAAt(0, 0) != (color.RGBA{R: 1, A: 2}) || pic.RGBAAt(1, 0) != (color.RGBA{G: 3, A: 4}) {
		t.Fatalf("Picture = %v", pic.Pix)
	}
	short := pal.Picture(2, 1, []color.RGBA{{R: 1, A: 2}})
	if short.RGBAAt(1, 0) != (color.RGBA{}) {
		t.Fatal("a missing pixel is not transparent")
	}
}
