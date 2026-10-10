package bmp

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"testing"
)

// DecodeRGBA writes the picture Decode(data).RGBA() holds, and refuses what
// Decode refuses, over random sizes, row paddings, both row orders and
// damaged headers.
func TestDecodeRGBAIsDecodeAtFullOpacity(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	for n := 0; n < 300; n++ {
		w, h := 1+rng.Intn(9), 1+rng.Intn(7)
		rows := make([][]Color, h)
		for y := range rows {
			rows[y] = make([]Color, w)
			for x := range rows[y] {
				rows[y][x] = Color{R: uint8(rng.Uint32()), G: uint8(rng.Uint32()), B: uint8(rng.Uint32())}
			}
		}
		data := build(t, rows, 0, rng.Intn(4))
		if n%2 == 1 {
			stride := (w*3 + 3) &^ 3
			flipped := append([]byte(nil), data[:HeaderLen]...)
			binary.LittleEndian.PutUint32(flipped[22:], uint32(-int32(h)))
			for y := h - 1; y >= 0; y-- {
				flipped = append(flipped, data[HeaderLen+y*stride:HeaderLen+(y+1)*stride]...)
			}
			data = flipped
		}
		if n%5 == 4 {
			data = append([]byte(nil), data...)
			data[rng.Intn(HeaderLen)] ^= byte(1 + rng.Intn(255))
		}
		im, wantErr := Decode(data)
		got, gotErr := DecodeRGBA(data)
		if (wantErr == nil) != (gotErr == nil) || wantErr != nil && wantErr.Error() != gotErr.Error() {
			t.Fatalf("case %d: error %v, want %v", n, gotErr, wantErr)
		}
		if wantErr != nil {
			continue
		}
		want := im.RGBA()
		if got.Rect != want.Rect || got.Stride != want.Stride || !bytes.Equal(got.Pix, want.Pix) {
			t.Fatalf("case %d: picture differs", n)
		}
	}
}
