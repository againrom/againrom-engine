package scriptprofile

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"testing"
)

// slotDump lists every operation row's slots for first values up to 40.
func slotDump() (string, int) {
	var b strings.Builder
	n := 0
	for _, ops := range []Ops{OpsFirst, OpsSecond} {
		for _, condition := range []bool{true, false} {
			for _, o := range OpRows(ops, condition) {
				for v := uint32(0); v <= 40; v++ {
					slots, ok := o.Slots(v)
					fmt.Fprintf(&b, "%d %t %d %d %t %v\n", ops, condition, o.ID, v, ok, slots)
					n++
				}
			}
		}
	}
	return b.String(), n
}

// TestSlotsOfEveryOperationAreUnchanged pins every row's slots by first value.
func TestSlotsOfEveryOperationAreUnchanged(t *testing.T) {
	dump, n := slotDump()
	if path := os.Getenv("SLOT_DUMP"); path != "" {
		if err := os.WriteFile(path, []byte(dump), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := fmt.Sprintf("%x", sha256.Sum256([]byte(dump)))
	t.Logf("%d slot lists, sha256 %s", n, got)
	const want, wantCount = "4369dcbd69cc7df511531444c27e02aed4ac3296b995ac611b0e1cc2b424c645", 4715
	if got != want || n != wantCount {
		t.Fatalf("slot lists %d sha256 %s, want %d %s", n, got, wantCount, want)
	}
}

// TestVariantsReachEveryOperationWithOne pins the group command and the
// teleport as rows with Variants.
func TestVariantsReachEveryOperationWithOne(t *testing.T) {
	want := []uint32{1, 2, 3, 4, 5, 10, 11, 14, 15, 17, 18}
	for _, c := range []struct {
		ops  Ops
		word string
	}{{OpsFirst, ""}, {OpsSecond, "editor.second.command.%d"}} {
		row, ok := OpRow(c.ops, false, 6)
		if !ok || !row.Grouped || fmt.Sprint(row.Values()) != fmt.Sprint(want) {
			t.Fatalf("ops %d action 6: grouped %t values %v", c.ops, row.Grouped, row.Values())
		}
		for _, v := range row.Variants {
			if c.word != "" && v.Word != fmt.Sprintf(c.word, v.Value) || c.word == "" && v.Word != "" {
				t.Errorf("ops %d command %d word %q", c.ops, v.Value, v.Word)
			}
		}
	}
	row, _ := OpRow(OpsSecond, false, 25)
	v, ok := row.Variant(CellMoveEvent)
	if !ok || row.Grouped || len(row.Variants) != 1 || v.Defaults[1] != 255 || v.Follows[4] != 2 {
		t.Fatalf("teleport variant %+v of %d", v, len(row.Variants))
	}
}
