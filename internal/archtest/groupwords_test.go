package archtest

import (
	"fmt"
	"os"
	"slices"
	"testing"
)

// Every shape of a write into an AI block outside the setter is named, at
// its position; reads, the setter file and a seam's whole-block copy are not.
func TestGroupWordWritesNameAPlantedWrite(t *testing.T) {
	const head = "package sim\nfunc f(g *SavedGroup, h SavedGroup, w *World) {\n"
	for _, tc := range []struct {
		shape, line string
		col         int
	}{
		{"index store", "g.AI[0x20] = 1", 1},
		{"index store", "g.AI[0x20], h.Selector = 1, 2", 1},
		{"index store", "h.AI[0x2d] += 4", 1},
		{"index store", "w.savedGroupByID(1).AI[0x15] = 2", 1},
		{"index increment", "w.savedGroupByID(1).AI[0x15]++", 1},
		{"PutUint16 into the block", "binary.LittleEndian.PutUint16(g.AI[10:], 3)", 1},
		{"PutUint32 into the block", "binary.LittleEndian.PutUint32(g.AI[0x30], 1)", 1},
		{"copy into the block", "copy(g.AI[0x00:0x02], g.AI[0x28:0x2a])", 1},
		{"clear into the block", "clear(g.AI[0x30:0x38])", 1},
		{"copy into the block", "copy(g.AI[:], h.AI[:])", 1},
		{"word address", "p := &g.AI[0x45]; *p = 1", 6},
		{"block address", "fill(&g.AI)", 6},
		{"slice handed on", "fill(g.AI[0x24:])", 6},
		{"slice handed on", "b := g.AI[:]; b[0] = 1", 6},
		{"whole-block store", "g.AI = h.AI", 1},
		{"AI literal field", "_ = SavedGroup{AI: [76]byte{1}}", 16},
	} {
		t.Run(tc.shape+" "+tc.line, func(t *testing.T) {
			got, err := GroupWordWrites(map[string]string{"pkg/sim/planted.go": head + tc.line + "\n}\n"})
			if err != nil {
				t.Fatal(err)
			}
			want := []string{fmt.Sprintf("pkg/sim/planted.go:3:%d %s", tc.col, tc.shape)}
			if !slices.Equal(got, want) {
				t.Fatalf("GroupWordWrites = %q, want %q", got, want)
			}
		})
	}
	reads := head + "_ = g.AI[0x20] == 1\n_ = binary.LittleEndian.Uint32(g.AI[0x48:])\nb = append(b, g.AI[:]...)\ncopy(raw[:], g.AI[:76])\n}\n"
	whole := head + "copy(g.AI[:], raw[:])\nget(&g.AI)\n_ = Data{Raw: g.AI[:]}\n}\n"
	got, err := GroupWordWrites(map[string]string{
		"pkg/sim/reads.go":               reads,
		GroupWordsSetter:                 head + "g.AI[0x20] = 1\n}\n",
		"pkg/sim/savedgroupsbinary.go":   whole,
		"pkg/game/originalfilesource.go": whole,
		"pkg/sim/planted_test.go":        head + "g.AI[0] = 1\n}\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("reads, the setter, a seam's whole block and a test were named: %q", got)
	}
	got, err = GroupWordWrites(map[string]string{"pkg/game/originalgroups.go": head + "copy(g.AI[2:4], raw[:])\n}\n"})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"pkg/game/originalgroups.go:3:1 copy into the block"}; !slices.Equal(got, want) {
		t.Fatalf("a seam's partial copy: GroupWordWrites = %q, want %q", got, want)
	}
}

// No production file writes a Group record's AI words but the setter, and
// only the seam files copy the whole block.
func TestGroupWordsHaveOneSetter(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadRandomnessSources(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range append([]string{GroupWordsSetter}, GroupWordsSeams...) {
		if _, ok := files[f]; !ok {
			t.Fatalf("%s is missing; the guard names a file the tree no longer holds", f)
		}
	}
	writes, err := GroupWordWrites(files)
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range writes {
		t.Errorf("a Group AI word is written outside %s: %s", GroupWordsSetter, w)
	}
}
