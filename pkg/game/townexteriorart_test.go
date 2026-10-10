package game

import (
	"encoding/binary"
	"fmt"
	"image/color"
	"strings"
	"testing"
)

func townBirdSheet(frames int) []byte {
	out := make([]byte, 1024)
	for i := 0; i < frames; i++ {
		out = binary.LittleEndian.AppendUint32(out, 1)
		out = binary.LittleEndian.AppendUint32(out, 1)
		out = binary.LittleEndian.AppendUint32(out, 4)
		out = binary.LittleEndian.AppendUint16(out, 1)          // one literal
		out = binary.LittleEndian.AppendUint16(out, 5<<1|15<<9) // opaque palette index 5
	}
	return binary.LittleEndian.AppendUint32(out, uint32(frames)|0x80000000)
}

func TestTownExteriorArtDropsOnlyIncompleteFamily(t *testing.T) {
	src := townSquareSource()
	for i := 0; i < 9; i++ {
		src[townSquareArtPrefix+fmt.Sprintf("door/t%02d.bmp", i)] = synthBMP(36, 48, color.RGBA{A: 255})
	}
	for i := 0; i < 10; i++ {
		src[townSquareArtPrefix+fmt.Sprintf("sign/v%02d.bmp", i)] = synthBMP(40, 32, color.RGBA{A: 255})
	}
	src[townSquareArtPrefix+"sign/v04.bmp"] = []byte("corrupt")
	a, err := LoadTownSquareArtFor(ROM1TownDescription(), src)
	if err != nil || len(a.Pictures("base")) != 1 || a.Mask == nil || len(a.Pictures("door")) != 9 || len(a.Pictures("sign")) != 0 {
		t.Fatalf("independent fallback: %v / %+v", err, a)
	}
	if !strings.Contains(strings.Join(a.Problems, "\n"), "sign/v04.bmp") {
		t.Fatal("missing named optional-art error")
	}
}

func TestTownAmbientArtLoadsBirdFamiliesLocallyAndStarsAtomically(t *testing.T) {
	src := townSquareSource()
	src[graphicsPrefix+"interface/townbirds/birds1/sprites.16a"] = townBirdSheet(57)
	src[graphicsPrefix+"interface/townbirds/birds2/sprites.16a"] = townBirdSheet(56)
	for i := 0; i < 9; i++ {
		src[townSquareArtPrefix+fmt.Sprintf("stars/s%02d.bmp", i)] = synthBMP(64, 44, color.RGBA{R: byte(i + 1), A: 255})
	}
	src[townSquareArtPrefix+"stars/s04.bmp"] = []byte("corrupt")
	a, err := LoadTownSquareArtFor(ROM1TownDescription(), src)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Pictures("birds/0")) != 57 || len(a.Pictures("birds/1")) != 0 {
		t.Fatalf("bird family fallback = %d/%d", len(a.Pictures("birds/0")), len(a.Pictures("birds/1")))
	}
	if len(a.Pictures("stars")) != 0 {
		t.Fatal("partial star family escaped atomic fallback")
	}
	problems := strings.Join(a.Problems, "\n")
	if !strings.Contains(problems, "birds2/sprites.16a") || !strings.Contains(problems, "stars/s04.bmp") {
		t.Fatalf("named optional failures missing:\n%s", problems)
	}

	src[townSquareArtPrefix+"stars/s04.bmp"] = synthBMP(64, 44, color.RGBA{R: 5, A: 255})
	a, err = LoadTownSquareArtFor(ROM1TownDescription(), src)
	if err != nil || len(a.Pictures("stars")) != 9 {
		t.Fatalf("complete star family = %d, %v", len(a.Pictures("stars")), err)
	}
}
