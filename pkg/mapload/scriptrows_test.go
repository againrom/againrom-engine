package mapload

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/scriptprofile"
	"againrom/pkg/sim"
)

// rowLine is what the readers say of one operation of a game's family:
// "ops condition id|name|decoded,offered,againrom|signature".
func rowLine(ops scriptprofile.Ops, condition bool, id uint32) string {
	flag := func(on bool, c string) string {
		if on {
			return c
		}
		return "-"
	}
	row, _ := scriptprofile.OpRow(ops, condition, id)
	offered := false
	for _, o := range ScriptOperationsIn(ops, condition) {
		offered = offered || o.Opcode == id
	}
	sig, ok := ScriptSignatureIn(ops, condition, id, 0)
	var parts []string
	for _, s := range sig {
		if s.Type != 0 {
			parts = append(parts, fmt.Sprintf("%d:%s", s.Type, s.Role))
		}
	}
	text := strings.Join(parts, ",")
	switch {
	case !ok:
		text = "none"
	case text == "":
		text = "empty"
	}
	return fmt.Sprintf("%d %v %#x|%s|%s%s%s|%s", ops, condition, id, ScriptOperationNameIn(ops, condition, id),
		flag(ScriptOperationDecoded(ops, condition, id), "D"), flag(offered, "O"), flag(row.Again, "A"), text)
}

// TestScriptRowsGiveTheNamesDecodedFlagsAndSignaturesTheOldTablesGave lists,
// for each game profile and family, every id the old tables could answer
// for; the expected lines were written from the old functions' answers. An
// id with no line has no name, is not decoded or offered and has no
// signature.
func TestScriptRowsGiveTheNamesDecodedFlagsAndSignaturesTheOldTablesGave(t *testing.T) {
	data, err := os.ReadFile("testdata/scriptrows.txt")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		want[line[:strings.Index(line, "|")]] = line
	}
	ids := []uint32{0x10002, 0x10003, 0x4100, 0x4101}
	for id := uint32(0); id <= 40; id++ {
		ids = append(ids, id)
	}
	seen := 0
	for _, ops := range []scriptprofile.Ops{scriptprofile.OpsFirst, scriptprofile.OpsSecond} {
		for _, condition := range []bool{true, false} {
			for _, id := range ids {
				got := rowLine(ops, condition, id)
				key := got[:strings.Index(got, "|")]
				line, ok := want[key]
				if !ok {
					line = key + "|Unknown operation|---|none"
				} else {
					seen++
				}
				if got != line {
					t.Errorf("got %q, want %q", got, line)
				}
			}
		}
	}
	if seen != len(want) {
		t.Errorf("%d expected lines read, %d listed", seen, len(want))
	}
}

// TestScriptOperationsAreOfferedInTheOldOrder pins the editor's operation
// choice order of each game.
func TestScriptOperationsAreOfferedInTheOldOrder(t *testing.T) {
	ids := func(ops scriptprofile.Ops, condition bool) []uint32 {
		var out []uint32
		for _, o := range ScriptOperationsIn(ops, condition) {
			out = append(out, o.Opcode)
		}
		return out
	}
	for _, c := range []struct {
		ops       scriptprofile.Ops
		condition bool
		want      []uint32
	}{
		{scriptprofile.OpsFirst, true, []uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 21, 0x10002, 0x4101}},
		{scriptprofile.OpsFirst, false, []uint32{2, 3, 4, 5, 6, 7, 8, 10, 12, 13, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 28, 29, 30, 32, 33, 34, 0x10002, 0x10003, 0x4100}},
		{scriptprofile.OpsSecond, true, []uint32{1, 2, 4, 5, 6, 7, 8, 9, 10, 14, 15, 16, 17, 18, 19, 21, 23, 24, 25, 26, 27, 0x10002}},
		{scriptprofile.OpsSecond, false, []uint32{2, 3, 4, 5, 6, 8, 10, 12, 13, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 28, 29, 30, 32, 33, 34, 35, 36, 38, 39, 0x10002}},
	} {
		if got := ids(c.ops, c.condition); !reflect.DeepEqual(got, c.want) {
			t.Errorf("ops %d condition %v: offered %v, want %v", c.ops, c.condition, got, c.want)
		}
	}
}

// TestFirstGameRowsOfferWhatTheSimulationSupports keeps the offered flag of
// the first game's rows equal to the simulation's supported arms.
func TestFirstGameRowsOfferWhatTheSimulationSupports(t *testing.T) {
	for _, condition := range []bool{true, false} {
		for id := uint32(1); id <= 34; id++ {
			row, _ := scriptprofile.OpRow(scriptprofile.OpsFirst, condition, id)
			if row.Offered != sim.ScriptOperationSupported(condition, int32(id)) {
				t.Errorf("condition %v id %d: offered %v, supported %v", condition, id, row.Offered, !row.Offered)
			}
		}
	}
}
