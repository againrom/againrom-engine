package data_test

import (
	"testing"

	"againrom/pkg/data"
)

// A placed person's figure directory, from the three columns its row states.
//
// THE SEX AXIS IS THE GENDER COLUMN AND NOT THE FACE BYTE. The other arm's
// test of bit 0x80 reads a byte no shipped row carries — 0 of 216 have it
// set — and a reader that used it drew all 462 campaign placements as men.
func TestFigureForReadsTheGenderColumnAndTheMageRange(t *testing.T) {
	for _, tc := range []struct {
		name    string
		typeID  int32
		face    int32
		gender  int32
		wantDir data.FigureDir
		wantFce int
	}{
		{"a swordsman, face 5", 3, 5, 0, data.FigureDirManFighter, 5},
		{"an archer, face 1", 14, 1, 0, data.FigureDirManFighter, 1},
		// The two mage type ids, and only those two.
		{"a mage at the low end of the range", 0x17, 2, 0, data.FigureDirManMage, 2},
		{"a mage at the high end of the range", 0x18, 2, 0, data.FigureDirManMage, 2},
		{"one below the mage range", 0x16, 2, 0, data.FigureDirManFighter, 2},
		{"one above the mage range", 0x19, 2, 0, data.FigureDirManFighter, 2},
		// The gender cell is the sex axis. 1 is female; nothing else is.
		{"a woman fighter", 3, 8, 1, data.FigureDirWomanFighter, 8},
		{"a woman mage", 0x18, 5, 1, data.FigureDirWomanMage, 5},
		{"an empty cell keeps the man", 3, 8, 0, data.FigureDirManFighter, 8},
		{"a cell that is neither is not female", 3, 8, 2, data.FigureDirManFighter, 8},
		// THE FACE IS THE COLUMN WHOLE. A reader still splitting it would answer
		// face 8 here and a woman's directory; both halves of that are wrong.
		{"a face byte with its top bit set is that face", 3, 0x88, 0, data.FigureDirManFighter, 0x88},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir, face := data.FigureFor(tc.typeID, tc.face, tc.gender)
			if dir != tc.wantDir || face != tc.wantFce {
				t.Errorf("FigureFor(%d, %#x, %d) = (%q, %d), want (%q, %d)",
					tc.typeID, tc.face, tc.gender, dir, face, tc.wantDir, tc.wantFce)
			}
		})
	}
}

// IT IS TOTAL, and the two axes are INDEPENDENT: whatever three integers arrive,
// the answer is one of the four shipped directories, the class axis moves only
// with the type id and the sex axis only with the gender cell.
func TestFigureForIsTotalAndItsTwoAxesAreIndependent(t *testing.T) {
	dirs := map[data.FigureDir]bool{
		data.FigureDirManFighter:   true,
		data.FigureDirManMage:      true,
		data.FigureDirWomanFighter: true,
		data.FigureDirWomanMage:    true,
	}
	for typeID := int32(-2); typeID <= 40; typeID++ {
		for face := int32(-1); face <= 0xff; face++ {
			for _, gender := range []int32{-1, 0, 1, 2} {
				dir, f := data.FigureFor(typeID, face, gender)
				if !dirs[dir] {
					t.Fatalf("FigureFor(%d, %d, %d) answered directory %q", typeID, face, gender, dir)
				}
				if int32(f) != face {
					t.Fatalf("FigureFor(%d, %d, %d) answered face %d", typeID, face, gender, f)
				}
				// The face cannot reach either axis: hold the other two and the
				// directory must not move with it.
				if base, _ := data.FigureFor(typeID, 0, gender); base != dir {
					t.Fatalf("FigureFor(%d, %d, %d) = %q but face 0 gives %q — the face reached an axis",
						typeID, face, gender, dir, base)
				}
			}
		}
	}
}
