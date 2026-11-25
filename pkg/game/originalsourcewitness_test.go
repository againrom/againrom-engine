package game

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"testing"
)

// The rows and steps of the witness. A town row records its groups (steps
// from 100) and the bytes of the file a resave of it writes (step 1); a
// mission row its read application (step 2), first frame (step 3) and
// captures (steps 4 to 6); a projection row the leaves it wrote (step 7), the
// document's bytes (step 8) and its refusal (step 9).
const (
	sourceRowTownLoaded = iota + 1
	sourceRowTownCurrent
	sourceRowTownSecondGraph
	sourceRowTownSecondCurrent
	sourceRowTownEmpty
	sourceRowTownNoSource
	sourceRowMissionStore
	sourceRowMissionStoreWide
	sourceRowMissionStoreWhole
	sourceRowMissionSecondDocument
	sourceRowMissionNoStore
	sourceRowMissionRefusedIndex
	sourceRowMissionRefusedMissing
	sourceRowMissionDocument
	sourceRowProjectCurrent
	sourceRowProjectAbsent
	sourceRowProjectLocal
	sourceRowProjectRefused
	sourceRowDocumentRefused
)

const (
	sourceStepSave = iota + 1
	sourceStepRead
	sourceStepFrame
	sourceStepCapture
	sourceStepCaptureNil
	sourceStepCaptureLocal
	sourceStepProjected
	sourceStepDocument
	sourceStepRefusal
)

type originalSourceTrace struct{ partyCrossingTrace }

// numbers records v under a row and step, as the party crossings do.
func (w *originalSourceTrace) numbers(row, step int, v any) { w.line(row, step, v) }

// refusal records an error's text as its FNV hash, zero for none.
func (w *originalSourceTrace) refusal(row, step int, err error) {
	var text uint64
	if err != nil {
		nums := []string{err.Error()}
		text = partyCrossingHash(nums)
	}
	fmt.Fprintf(&w.b, "%d %d %d\n", row, step, text)
}

// bytes records the SHA-256 of b as two numbers and its length.
func (w *originalSourceTrace) bytes(row, step int, b []byte) {
	sum := sha256.Sum256(b)
	fmt.Fprintf(&w.b, "%d %d %d %d %d\n", row, step, len(b), binary.BigEndian.Uint64(sum[:8]), binary.BigEndian.Uint64(sum[8:16]))
}

// groups records each group in order: its keys, a hash of its fields and of
// each member's party ID, in order.
func (w *originalSourceTrace) groups(row int, groups []cityLiveGroup) {
	fmt.Fprintf(&w.b, "%d 99 %d\n", row, len(groups))
	for i, g := range groups {
		var nums []string
		partyCrossingNumbers(&nums, reflect.ValueOf(g.Payload))
		fmt.Fprintf(&w.b, "%d %d %d %d %d %d %d", row, 100+i, g.Payload.F1c, g.Payload.F40, g.Payload.F44, partyCrossingHash(nums), len(g.Members))
		for _, member := range g.Members {
			fmt.Fprintf(&w.b, " %d", partyCrossingHash([]string{member}))
		}
		w.b.WriteByte('\n')
	}
}

// Each town fact picks its row in the table's order: a current file with a
// readable provenance binds through its document, the second game's file
// through its graph, an original through its provenance, and none of them
// leaves no source.
func TestTownSourceTableChoosesOneRowPerFile(t *testing.T) {
	bare := &originalSource{campaign: &originalCampaignDecode{}}
	second := &originalSource{campaign: &originalCampaignDecode{fileSession: &currentSessionData{}}}
	refused := errors.New("no provenance")
	for _, c := range []struct {
		name string
		in   originalTownInputs
		want string
	}{
		{"current with provenance", originalTownInputs{src: bare, hasCurrent: true}, "current SAV document"},
		{"current without provenance", originalTownInputs{src: bare, hasCurrent: true, cityErr: refused}, ""},
		{"current with provenance over a file session", originalTownInputs{src: second, hasCurrent: true}, "current SAV document"},
		{"current without provenance over a file session", originalTownInputs{src: second, hasCurrent: true, cityErr: refused}, "file session and actor graph"},
		{"no current over a file session", originalTownInputs{src: second}, "file session and actor graph"},
		{"an original with provenance", originalTownInputs{src: bare}, "city provenance"},
		{"an original without provenance", originalTownInputs{src: bare, cityErr: refused}, "city provenance"},
	} {
		got := ""
		if row := originalTownSource(c.in); row != nil {
			got = row.name
		}
		if got != c.want {
			t.Errorf("%s: row %q, want %q", c.name, got, c.want)
		}
	}
	rows := originalFileSources()
	names := map[string]bool{}
	for _, row := range rows {
		if row.name == "" || names[row.name] {
			t.Errorf("row %q is unnamed or repeated", row.name)
		}
		names[row.name] = true
		if (row.townHolds == nil) != (row.townGroups == nil) || (row.missionHolds == nil) != (row.readApplication == nil) || (row.frameHolds == nil) != (row.frame == nil) || (row.captureHolds == nil) != (row.captureApplication == nil) {
			t.Errorf("row %q holds a test without its reader", row.name)
		}
	}
	if len(rows) != 8 {
		t.Errorf("%d rows, want the three town sources, the state store, the document, capture, casters and Outposts", len(rows))
	}
}
