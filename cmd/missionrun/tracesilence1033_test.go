package main

// The readout's SILENCE REASONS and its STRUCTURE REFERENCE (1033, adversarial
// pass 1 W-2 and the label beside it). Nothing here reads an install.
//
// Two reasons pkg/sim can give were unnamed by silenceText and printed as the
// word "unknown": ScriptSilenceNoPlayer, which predates this story, and
// ScriptSilenceNoStructure, which this story added. A reader of the trace could
// not tell them apart, and could not tell either from a reason the table simply
// had not been updated for.

import (
	"strings"
	"testing"

	"againrom/pkg/sim"
)

// traceSilenceReasons is every reason pkg/sim declares today. It is a hand list
// and it cannot see a reason added later; what covers that case is the
// production fallback, asserted by the second test below, which prints an
// unnamed reason's own number instead of a word shared with every other unnamed
// reason.
var traceSilenceReasons = []sim.ScriptSilence{
	sim.ScriptSilenceUnsupported,
	sim.ScriptSilenceNoUnit,
	sim.ScriptSilenceNoGroup,
	sim.ScriptSilenceNoPlayer,
	sim.ScriptSilenceNoStructure,
}

// TestEverySilenceReasonIsNamedAndNoTwoReadAlike is the property that failed:
// each reason produces its own text, and none produces the fallback. Asserting
// distinctness rather than five literal strings is what makes the test catch a
// reason accidentally routed to another reason's arm, which is the shape a
// copied case statement takes.
func TestEverySilenceReasonIsNamedAndNoTwoReadAlike(t *testing.T) {
	t.Parallel()

	seen := map[string]sim.ScriptSilence{}
	for _, why := range traceSilenceReasons {
		got := silenceText(why)
		if got == "" {
			t.Errorf("reason %d renders as the empty string", why)
			continue
		}
		if strings.Contains(got, "unnamed silence reason") || strings.Contains(got, "unknown") {
			t.Errorf("reason %d renders as %q, which names no reason", why, got)
		}
		if prior, dup := seen[got]; dup {
			t.Errorf("reasons %d and %d both render as %q; a reader cannot tell them apart",
				prior, why, got)
		}
		seen[got] = why
	}
}

// TestAnUnnamedSilenceReasonPrintsItsOwnNumber covers the reason this table does
// not have yet. The number is what tells the next reader which arm to add; the
// word "unknown" was what let two reasons go unnamed here.
func TestAnUnnamedSilenceReasonPrintsItsOwnNumber(t *testing.T) {
	t.Parallel()

	next := sim.ScriptSilence(len(traceSilenceReasons))
	got := silenceText(next)
	if !strings.Contains(got, "unnamed silence reason") {
		t.Errorf("an undeclared reason renders as %q, want the fallback form", got)
	}
	if !strings.Contains(got, "4") && !strings.Contains(got, "5") {
		t.Errorf("the fallback %q carries no number", got)
	}
	if got == silenceText(next+1) {
		t.Errorf("two undeclared reasons both render as %q; the fallback carries no number", got)
	}
}

// TestAStructureReferenceIsNotPrintedAsAnAbsentUnit is the label. checkDesc's
// fallback arm renders a check's UNIT reference, and a check-21 node names no
// unit -- its one reference is a structure. Through that arm the node printed
// `structfield(NO UNIT)`, which states a true fact about units in the place
// every other arm puts its own subject, and reads as a dangling reference.
func TestAStructureReferenceIsNotPrintedAsAnAbsentUnit(t *testing.T) {
	t.Parallel()

	s, err := sim.NewScript(
		[]sim.ScriptCheck{
			{Op: sim.ScriptCheckStructField, Register: 0, Structure: 6, HasStructure: true},
			{Op: sim.ScriptCheckStructField, Register: 1},
		},
		nil, nil,
	)
	if err != nil {
		t.Fatalf("NewScript: %v", err)
	}
	tr := newTracer(true, nil, nil, s)

	got := tr.checkText(0)
	if strings.Contains(got, "NO UNIT") {
		t.Errorf("a resolved structure reference prints as %q", got)
	}
	if !strings.Contains(got, "structfield") || !strings.Contains(got, "6") {
		t.Errorf("checkText(structfield) = %q, want the arm and the structure it names", got)
	}

	// An absent structure reference says so in the structure's own vocabulary,
	// so a reader can tell it from a node whose UNIT reference failed.
	absent := tr.checkText(1)
	if !strings.Contains(absent, "NO STRUCTURE") {
		t.Errorf("checkText(absent structure) = %q, want the structure's own word", absent)
	}
	if absent == got {
		t.Errorf("a named and an unnamed structure both print as %q", got)
	}
}
