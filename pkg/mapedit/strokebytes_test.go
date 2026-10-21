package mapedit_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

// circleStrokeWant pins the bytes a map saves after each height, object and
// impassable stroke of the default circle brush, taken from the engine
// before the brush had a shape: the shape must not move them.
var circleStrokeWant = map[string]string{
	"height -17 flat size 0":    "85ca5e8b368a955d3b6d24dad09e3d2eb7075528f5d74ea3433bb3cee4a50a8f",
	"height -17 flat size 1":    "df89d2ee9f2e852e419cf79bf02e053b278ba3ae79a24551e10f17eba10a7e8e",
	"height -17 flat size 16":   "c60f559ac680169bb8e8a0f7859986f2bd22924fff189372ac1bd6df93b2e058",
	"height -17 flat size 3":    "f11ec12b47ef32e18905a1e360953ec23b7292b7f138d086bcb7bcb0486daf1c",
	"height -17 flat size 8":    "48c12efadb0e98c8427ce8a495c235add1f43f6395379ba3087dcfa8a35ea72d",
	"height 300 cosine size 0":  "1d0fd63dcb217832c6fa5abe2529e3d9260222d07285c5e61e91bafb9ad144d7",
	"height 300 cosine size 1":  "d4407aba0117e9fb6feee84f5d72a08859199bab6b3bfa64f46671268093ba03",
	"height 300 cosine size 16": "222997b35fd32e37f32d953089445d07b22092aa61df91efb6d40b4763cd5333",
	"height 300 cosine size 3":  "991095a655e70652d54fba68d36442f3caf147653b213b5a8533589c5c83b8ae",
	"height 300 cosine size 8":  "0fb657a1c81ab9e7b361ab1f61082183c06f7f853ac2ef7a66d41f10687b655e",
	"height 40 cosine size 0":   "de72634bfcc1e22fb57bacb1a8bc99a0cdbf44d74daf9b3e3f924e5d0906b499",
	"height 40 cosine size 1":   "da16f166c5dbb09f18be86b612bfa1b889bc94c4b48d9f1800a1c34d4b3a4c5c",
	"height 40 cosine size 16":  "9a17d3b152621d18e3af446a1c2ee40ae8e597115290361d081f90a608d8ff69",
	"height 40 cosine size 3":   "880cc9df5e5dfb2dab4cab7e51dc9a3a52279da7ac300e9edbd67f48c866b931",
	"height 40 cosine size 8":   "59b1b9f4280c8dc4d8118e215257ecf70996e060b544e62d0a62c9166bd3a3bb",
	"impassable 0 size 0":       "aefb525e10f366d0ceb4ac90385c638be63acd9d9833cdd7e7629f88b9cce05d",
	"impassable 0 size 1":       "615175e46745cd7d4fdd57eb568d67e13d6ee6baed08735eb7925be3ac6bbed1",
	"impassable 0 size 16":      "350aba5b8f3ffdbae7da92417db49184c1ffd2d86981bae02ccffd98cf887556",
	"impassable 0 size 3":       "0816c802e2d32269097159cb250f12460c3d5d34f9cf066c7cbc5c5218253b16",
	"impassable 0 size 8":       "7e414c2aba8ada495f6b6f7d07d1eed242fa91c1e0a4ca2c295621dadadb0e2c",
	"impassable 1 size 0":       "aefb525e10f366d0ceb4ac90385c638be63acd9d9833cdd7e7629f88b9cce05d",
	"impassable 1 size 1":       "f8c715c937247abe3843f6b47139b1fcc6fed738c6276af5f14cddf5725afad9",
	"impassable 1 size 16":      "9300b54bdeb1cfb9e6d8a473d90fe7bce1a6f39d303cfb2f578e69f974b67db4",
	"impassable 1 size 3":       "546f75ae8e26d19fea3cf4efe5f0edd2af7c152fea14e254eafb71197f5989fa",
	"impassable 1 size 8":       "6b163fdba271076c49eb121e6c009aff6dcf248d869b74342be1e3cbbc0677fd",
	"object 0 size 0":           "56d79a468627cfe7401940d03e674a5b99792f64a3ac6080640857daec75d159",
	"object 0 size 1":           "82396a195a2ad902c82562fc60be31f000750f37d7b2258995d1995560dbd44f",
	"object 0 size 16":          "936d09689c4f1f180d45019a6eadebab5926da52d171991b99883632bd01eaae",
	"object 0 size 3":           "8ac503a13c321b6d6624f0103c3a63e58ca9fb2f27f3cc5695c060b0ef9722b5",
	"object 0 size 8":           "b4d7c6b8596175712e7bd34ea650d25b87db2b6d851f206aba52a41825fa1737",
	"object 7 size 0":           "56d79a468627cfe7401940d03e674a5b99792f64a3ac6080640857daec75d159",
	"object 7 size 1":           "5c33729ad773de8b0cf490b54dcdf0ec797759a4b72630ec523fd490ec657c66",
	"object 7 size 16":          "a580c08178f3fef55fded9f0ef4188520036b2879f79ef365eef9b26603e7ccf",
	"object 7 size 3":           "dcf6a3c5ea49ee86177b64c82adf8866e345ff0dbf9ab96eebfab716165b9335",
	"object 7 size 8":           "22f50b4606af84c5bc406a3f2955ef2d13954e6c5e513538f203b2cfdf5c8a4c",
}

func circleStrokeHashes(t *testing.T) map[string]string {
	t.Helper()
	got := map[string]string{}
	sum := func(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
	e, err := mapedit.NewMap(alm.NewMapSpec{Width: 48, Height: 40, Tile: 0x14, Altitude: 64})
	if err != nil {
		t.Fatal(err)
	}
	strokes := []struct {
		name string
		b    mapedit.Brush
	}{
		{"height 40 cosine", mapedit.Brush{Plane: mapedit.PlaneHeight, Value: 40, Falloff: mapedit.FalloffCosine}},
		{"height -17 flat", mapedit.Brush{Plane: mapedit.PlaneHeight, Value: -17}},
		{"height 300 cosine", mapedit.Brush{Plane: mapedit.PlaneHeight, Value: 300, Falloff: mapedit.FalloffCosine}},
		{"object 7", mapedit.Brush{Plane: mapedit.PlaneObject, Value: 7}},
		{"object 0", mapedit.Brush{Plane: mapedit.PlaneObject, Value: 0}},
		{"impassable 1", mapedit.Brush{Plane: mapedit.PlaneImpassable, Value: 1}},
		{"impassable 0", mapedit.Brush{Plane: mapedit.PlaneImpassable, Value: 0}},
	}
	for i, r := range []int{0, 1, 3, 8, 16} {
		for j, s := range strokes {
			s.b.Radius = r
			at := []mapedit.Cell{{X: 10 + 6*i, Y: 12 + 2*j}, {X: 13 + 6*i, Y: 15 + 2*j}}
			if _, err := e.Paint(s.b, at, nil); err != nil {
				t.Fatalf("%s size %d: %v", s.name, r, err)
			}
			got[fmt.Sprintf("%s size %d", s.name, r)] = sum(e.Bytes())
		}
	}
	return got
}

func TestCircleStrokesKeepTheirSavedBytes(t *testing.T) {
	got := circleStrokeHashes(t)
	if len(circleStrokeWant) == 0 {
		for k, v := range got {
			t.Logf("PIN %q: %q,", k, v)
		}
		t.Fatal("no pinned bytes")
	}
	if len(got) != len(circleStrokeWant) {
		t.Fatalf("%d steps, %d pinned", len(got), len(circleStrokeWant))
	}
	for k, want := range circleStrokeWant {
		if got[k] != want {
			t.Errorf("%s: saved bytes changed", k)
		}
	}
}
