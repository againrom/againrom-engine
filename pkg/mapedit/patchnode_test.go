package mapedit

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
)

// TestPatchNodeKeepsWhatItDoesNotNameAndClearsTheSlotsPastParams pins the one
// node writer: a label-less patch keeps the label field and the bytes no field
// names, clears the parameter slots past params, and leaves its base alone.
func TestPatchNodeKeepsWhatItDoesNotNameAndClearsTheSlotsPastParams(t *testing.T) {
	base := make([]byte, alm.ScriptNodeSize)
	copy(base, "Old")
	base[0x9c] = 0xee
	base[alm.ScriptNodeValues+4*9] = 0x55
	rec, err := patchNode(base, nil, 4, 9, []ScriptParam{{Type: 1, Value: 2}})
	if err != nil {
		t.Fatal(err)
	}
	if string(rec[:3]) != "Old" {
		t.Fatalf("the label field is %q, want the kept %q", rec[:3], "Old")
	}
	if rec[0x9c] != 0xee {
		t.Fatal("a byte no field names changed")
	}
	if got := rec[alm.ScriptNodeValues+4*9]; got != 0 {
		t.Fatalf("slot 9 keeps %d, want it cleared", got)
	}
	if base[alm.ScriptNodeOpcode] != 0 || base[alm.ScriptNodeValues+4*9] != 0x55 {
		t.Fatal("the base record was written")
	}
}

// TestNewScriptNodeIsThePatchOfAZeroRecord checks that the constructor and
// the patch give the same bytes for the same node.
func TestNewScriptNodeIsThePatchOfAZeroRecord(t *testing.T) {
	params := []ScriptParam{{Type: 4, Value: 7}, {Type: 5, Value: 9}}
	fresh, err := NewScriptNode("Drop location", 3, 12, params...)
	if err != nil {
		t.Fatal(err)
	}
	patched, err := patchNode(make([]byte, alm.ScriptNodeSize), ptr("Drop location"), 3, 12, params)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(fresh, patched) {
		t.Fatal("NewScriptNode and the patch of a zero record differ")
	}
}

// TestPatchNodeRefusesALongLabelWithTheOneSentinel pins the label refusal
// that the constructor, the rewrite and the copy share.
func TestPatchNodeRefusesALongLabelWithTheOneSentinel(t *testing.T) {
	long := strings.Repeat("x", alm.ScriptLabelLen)
	base := make([]byte, alm.ScriptNodeSize)
	if _, err := patchNode(base, &long, 1, 1, nil); !errors.Is(err, errLabelLong) {
		t.Fatalf("patch refused a long label with %v", err)
	}
	if _, err := NewScriptNode(long, 1, 1); !errors.Is(err, errLabelLong) {
		t.Fatalf("NewScriptNode refused a long label with %v", err)
	}
}

func ptr(s string) *string { return &s }

// TestPatchNodeClearsTheLabelForAnEmptyLabel pins that an empty label clears.
func TestPatchNodeClearsTheLabelForAnEmptyLabel(t *testing.T) {
	base := make([]byte, alm.ScriptNodeSize)
	copy(base, "Old")
	rec, err := patchNode(base, ptr(""), 1, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if rec[0] != 0 || rec[1] != 0 || rec[2] != 0 {
		t.Fatalf("the label field is %q, want it cleared", rec[:3])
	}
}
