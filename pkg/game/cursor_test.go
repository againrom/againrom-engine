package game_test

import (
	"encoding/binary"
	"errors"
	"image"
	"io/fs"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/game"
	"againrom/pkg/vfs"
)

// The .16a control-word grammar, spelled here rather than imported: the decoder
// is what is under test, so a fixture built from its own constants would agree
// with it by construction.
const (
	curOpLiteral = 0 << 14
	curOpSkip    = 2 << 14
)

// curPixel packs one literal pixel word: palette index in bits 1-8, level in
// bits 9-12. Bit 0 and bits 13-15 are left clear, exactly as every shipped word
// leaves them.
func curPixel(index, level int) uint16 { return uint16(index<<1) | uint16(level<<9) }

// curPalette is a 1024-byte [B,G,R,x] block whose entry k is a triple derived
// from k, so a loader that read the wrong entry — or read the channels in the
// on-disk order — could not produce the expectations below.
func curPalette() []byte {
	b := make([]byte, 1024)
	for k := 0; k < 256; k++ {
		b[k*4+0] = byte(3 * k) // B
		b[k*4+1] = byte(2 * k) // G
		b[k*4+2] = byte(k)     // R
	}
	return b
}

// curSheet builds a one-frame .16a stream: the palette block, a w x h record
// carrying the given control stream, and the trailer with the count and the
// palette flag the shipped sheets carry.
func curSheet(w, h int, controls []uint16) []byte {
	block := make([]byte, 0, len(controls)*2)
	for _, c := range controls {
		block = binary.LittleEndian.AppendUint16(block, c)
	}

	out := curPalette()
	out = binary.LittleEndian.AppendUint32(out, uint32(w))
	out = binary.LittleEndian.AppendUint32(out, uint32(h))
	out = binary.LittleEndian.AppendUint32(out, uint32(len(block)))
	out = append(out, block...)
	// One frame, with bit 31 set: the palette-bearing form.
	return binary.LittleEndian.AppendUint32(out, 1|0x80000000)
}

// curSource wraps a stream as the cursor's entry in a container, off the
// loader's own address rather than one spelled again.
func curSource(t *testing.T, stream []byte) *vfs.FS {
	t.Helper()
	files := []synth.File{{Path: graphicsEntry(t, game.AttackCursorPath), Data: stream}}
	return openContainers(t, synth.Archive(files))
}

// AC-7: the palette gives the colour, the level gives the coverage, and an
// unpainted cell is fully transparent.
//
// The four cells are a 2x2 grid: one skipped, then three literals whose levels
// span the range — the top, the bottom and one in between. The expectations are
// written out rather than recomputed from the rule, so a change to the rule
// fails here instead of agreeing with itself.
func TestLoadAttackPointerResolvesPaletteAndCoverage(t *testing.T) {
	stream := curSheet(2, 2, []uint16{
		curOpSkip | 1,
		curOpLiteral | 3,
		curPixel(5, 15),
		curPixel(7, 0),
		curPixel(9, 7),
	})

	pic, err := game.LoadAttackPointer(curSource(t, stream))
	if err != nil {
		t.Fatalf("LoadAttackPointer: %v", err)
	}
	if got := pic.Bounds(); got != image.Rect(0, 0, 2, 2) {
		t.Fatalf("picture bounds %v, want a 2x2", got)
	}

	for _, tc := range []struct {
		name           string
		at             int
		r, g, b, alpha uint8
	}{
		// The skipped cell. Not "black at zero alpha" as a colour — no pixel at
		// all, which is what keeps a painted level 0 distinguishable from it.
		{"the skipped cell is wholly transparent", 0, 0, 0, 0, 0},
		// index 5 at level 15: coverage 16/16, so the palette entry survives
		// premultiplication untouched.
		{"level 15 is the palette entry at full coverage", 1, 5, 10, 15, 255},
		// index 7 at level 0: coverage 1/16. A PAINTED pixel, nearly invisible,
		// and emphatically not a hole.
		{"level 0 is painted at the lowest coverage", 2, 0, 0, 1, 15},
		// index 9 at level 7: coverage 8/16.
		{"a middle level is premultiplied by its coverage", 3, 4, 8, 13, 127},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := tc.at * 4
			gotR, gotG, gotB, gotA := pic.Pix[o], pic.Pix[o+1], pic.Pix[o+2], pic.Pix[o+3]
			if gotR != tc.r || gotG != tc.g || gotB != tc.b || gotA != tc.alpha {
				t.Errorf("cell %d = (%d,%d,%d,%d), want (%d,%d,%d,%d)",
					tc.at, gotR, gotG, gotB, gotA, tc.r, tc.g, tc.b, tc.alpha)
			}
		})
	}
}

// AC-7's other half, and AC-11's premise: every failure is an error and none is
// a partial picture.
func TestLoadAttackPointerRefusesWhatItCannotResolve(t *testing.T) {
	t.Run("no source", func(t *testing.T) {
		if _, err := game.LoadAttackPointer(nil); err == nil {
			t.Error("a nil source loaded a picture")
		}
	})

	t.Run("no entry, and the source's own path error survives", func(t *testing.T) {
		src := openContainers(t, synth.Archive(nil))
		_, err := game.LoadAttackPointer(src)
		if err == nil {
			t.Fatal("an archive with no cursor entry loaded a picture")
		}
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("error %v does not carry fs.ErrNotExist", err)
		}
	})

	t.Run("a stream that will not decode, named by its address", func(t *testing.T) {
		// A literal run announcing more words than its block holds: refused at
		// the read, before any cursor move.
		stream := curSheet(2, 2, []uint16{curOpLiteral | 9})
		pic, err := game.LoadAttackPointer(curSource(t, stream))
		if err == nil {
			t.Fatal("a malformed stream loaded a picture")
		}
		if pic != nil {
			t.Error("a failed load returned a picture as well as an error")
		}
		if !strings.Contains(err.Error(), game.AttackCursorPath) {
			t.Errorf("error %v does not name the address it failed at", err)
		}
	})

	t.Run("a sheet with no frame to draw", func(t *testing.T) {
		// The palette block and a trailer declaring zero frames.
		stream := append(curPalette(), 0, 0, 0, 0x80)
		if _, err := game.LoadAttackPointer(curSource(t, stream)); err == nil {
			t.Error("a sheet holding no frame loaded a picture")
		}
	})

	// A pointer with no pixels is indistinguishable at the draw from the mode
	// being down, which is the failure this whole story exists to remove.
	t.Run("a zero-area frame", func(t *testing.T) {
		stream := curSheet(0, 0, nil)
		if _, err := game.LoadAttackPointer(curSource(t, stream)); err == nil {
			t.Error("a zero-area frame loaded a picture that would draw nothing")
		}
	})
}

// The address is built in one place and carries the archive identity segment, so
// a caller need not know which container the cursor art lives in.
func TestAttackCursorPath(t *testing.T) {
	if got := game.AttackCursorPath; got != "graphics/cursors/attack/sprites.16a" {
		t.Errorf("AttackCursorPath = %q", got)
	}
}
