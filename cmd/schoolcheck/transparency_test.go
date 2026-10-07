package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"testing"
)

// syntheticPatch authors one padded, bottom-up, 24-bit BMP row directly.
// It uses neither a production encoder nor game art/keying helpers.
func syntheticPatch(rgb ...[3]byte) []byte {
	stride := (len(rgb)*3 + 3) &^ 3
	raw := make([]byte, 54+stride)
	copy(raw, "BM")
	binary.LittleEndian.PutUint32(raw[2:], uint32(len(raw)))
	binary.LittleEndian.PutUint32(raw[10:], 54)
	binary.LittleEndian.PutUint32(raw[14:], 40)
	binary.LittleEndian.PutUint32(raw[18:], uint32(len(rgb)))
	binary.LittleEndian.PutUint32(raw[22:], 1)
	binary.LittleEndian.PutUint16(raw[26:], 1)
	binary.LittleEndian.PutUint16(raw[28:], 24)
	binary.LittleEndian.PutUint32(raw[34:], uint32(stride))
	for i := 54; i < len(raw); i++ {
		raw[i] = 0xa7 // Non-black padding must not enter the population.
	}
	for i, c := range rgb {
		raw[54+3*i], raw[55+3*i], raw[56+3*i] = c[2], c[1], c[0]
	}
	return raw
}

func TestPatchTransparencyBoundaries(t *testing.T) {
	for _, tt := range []struct {
		name                    string
		rgb                     [3]byte
		black, zero565, zero555 int
	}{
		{"true-black", [3]byte{0, 0, 0}, 1, 0, 0},
		{"red-near-black", [3]byte{1, 0, 0}, 0, 1, 1},
		{"green-near-black", [3]byte{0, 1, 0}, 0, 1, 1},
		{"blue-near-black", [3]byte{0, 0, 1}, 0, 1, 1},
		{"last-zero-565", [3]byte{7, 3, 7}, 0, 1, 1},
		{"green-four-is-555-only", [3]byte{0, 4, 0}, 0, 0, 1},
		{"last-zero-555", [3]byte{7, 7, 7}, 0, 0, 1},
		{"red-eight", [3]byte{8, 0, 0}, 0, 0, 0},
		{"green-eight", [3]byte{0, 8, 0}, 0, 0, 0},
		{"blue-eight", [3]byte{0, 0, 8}, 0, 0, 0},
		{"red-overrides-small-green-blue", [3]byte{8, 3, 7}, 0, 0, 0},
		{"white", [3]byte{255, 255, 255}, 0, 0, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := patchTransparency(syntheticPatch(tt.rgb))
			if err != nil {
				t.Fatal(err)
			}
			want := transparencyCounts{patches: 1, pixels: 1, black: tt.black,
				nonblackZero565: tt.zero565, nonblackZero555: tt.zero555}
			if got != want {
				t.Fatalf("RGB %v: got %+v, want %+v", tt.rgb, got, want)
			}
		})
	}
}

func TestPatchTransparencyRejectsInvalidBMP(t *testing.T) {
	for _, tt := range []struct {
		name   string
		mutate func([]byte) []byte
	}{
		{"empty", func([]byte) []byte { return nil }},
		{"short-header", func(b []byte) []byte { return b[:53] }},
		{"missing-pixels", func(b []byte) []byte { return b[:54] }},
		{"wrong-magic", func(b []byte) []byte { b[0] = 'X'; return b }},
		{"wrong-depth", func(b []byte) []byte { binary.LittleEndian.PutUint16(b[28:], 16); return b }},
		{"zero-width", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[18:], 0); return b }},
		{"zero-height", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[22:], 0); return b }},
		{"too-wide", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[18:], 81); return b }},
		{"too-high", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[22:], 37); return b }},
		{"negative-height", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[22:], 0xffffffff); return b }},
		{"huge-dimensions", func(b []byte) []byte {
			binary.LittleEndian.PutUint32(b[18:], 0x7fffffff)
			binary.LittleEndian.PutUint32(b[22:], 0x7fffffff)
			return b
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := patchTransparency(tt.mutate(syntheticPatch([3]byte{0, 0, 0})))
			if err == nil || got != (transparencyCounts{}) {
				t.Fatalf("invalid BMP returned counts %+v, error %v", got, err)
			}
		})
	}
}

type transparencySource struct {
	files map[string][]byte
	reads map[string]int
}

func (s *transparencySource) ReadFile(addr string) ([]byte, error) {
	s.reads[addr]++
	raw, ok := s.files[addr]
	if !ok {
		return nil, os.ErrNotExist
	}
	return raw, nil
}

// The fixture independently lists the expected population. Reusing skillName
// or stateName here would let an incomplete census silently shrink its test.
func completeTransparencySource() *transparencySource {
	src := &transparencySource{files: map[string][]byte{}, reads: map[string]int{}}
	for _, skill := range []string{
		"fighter/sword", "fighter/axe", "fighter/club", "fighter/pike", "fighter/bow",
		"mage/fire", "mage/water", "mage/air", "mage/earth", "mage/astral",
	} {
		for _, state := range []string{"on", "shine", "shine_on"} {
			addr := "graphics/interface/training/column/" + skill + "/" + state + ".bmp"
			src.files[addr] = syntheticPatch(
				[3]byte{0, 0, 0}, [3]byte{1, 1, 1}, [3]byte{0, 4, 0}, [3]byte{8, 0, 0})
		}
	}
	return src
}

func TestCensusTransparencyCompletePopulation(t *testing.T) {
	src := completeTransparencySource()
	var out bytes.Buffer
	if err := censusTransparency(src, &out); err != nil {
		t.Fatal(err)
	}
	want := "transparency scope: 10 skills x 3 states; standard RGB565/RGB555 channel truncation only\n" +
		"transparency patches 30 pixels 120 pure-black 30 nonblack-zero-rgb565 30 nonblack-zero-rgb555 60\n" +
		"schoolcheck: transparency census complete\n"
	if out.String() != want {
		t.Fatalf("census output:\n%s\nwant:\n%s", out.String(), want)
	}
	if len(src.reads) != 30 {
		t.Fatalf("read %d distinct patches, want 30", len(src.reads))
	}
	for addr := range src.files {
		if src.reads[addr] != 1 {
			t.Errorf("%s read %d times, want exactly once", addr, src.reads[addr])
		}
	}
}

func TestCensusTransparencyRejectsEveryMissingPatch(t *testing.T) {
	for missing := range completeTransparencySource().files {
		t.Run(missing, func(t *testing.T) {
			src := completeTransparencySource()
			delete(src.files, missing)
			var out bytes.Buffer
			err := censusTransparency(src, &out)
			if err == nil || !strings.Contains(err.Error(), missing) || out.Len() != 0 {
				t.Fatalf("missing patch: error %v, output %q", err, out.String())
			}
		})
	}
}

func TestCensusTransparencyRejectsEmptyAndInvalidPopulation(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		t.Run(fmt.Sprintf("invalid-final-patch=%v", invalid), func(t *testing.T) {
			src := completeTransparencySource()
			if invalid {
				src.files["graphics/interface/training/column/mage/astral/shine_on.bmp"] = []byte("invalid BMP")
			} else {
				src.files = nil
			}
			var out bytes.Buffer
			if err := censusTransparency(src, &out); err == nil || out.Len() != 0 {
				t.Fatalf("incomplete population: error %v, output %q", err, out.String())
			}
		})
	}
}

func TestTransparencyOnlyRejectsOutputAndSelection(t *testing.T) {
	for _, args := range [][]string{
		{"-transparency-only", "-png", "unused"},
		{"-transparency-only", "fighter/sword/on.bmp"},
	} {
		var out bytes.Buffer
		err := run(args, &out)
		if err == nil || !strings.Contains(err.Error(), "accepts no -png output or positional patch selection") || out.Len() != 0 {
			t.Fatalf("args %v: error %v, output %q", args, err, out.String())
		}
	}
}
