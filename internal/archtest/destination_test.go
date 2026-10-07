package archtest

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

var wantDestinationWriters = []string{
	"attachMoveOrder", "unmarshalBinary", "walkTo", "issueSavedGroupDestination", "armSwarm", "armPatrol", "walkHome",
	"guardWalkHome", "escortClose", "escortStepAway", "withdrawFromAny", "stepScrollCasts", "savedMove",
	"savedDecision", "beginSavedRouteContinuation", "standDown", "retainCycleForState", "dispatchRetreatPending",
}

func diffDestinationWriters(findings []Finding, want []string) []string {
	got := make(map[string]Finding, len(findings))
	for _, f := range findings {
		got[f.Func] = f
	}
	wantSet := make(map[string]bool, len(want))
	for _, w := range want {
		wantSet[w] = true
	}

	var msgs []string
	for _, w := range want {
		if _, ok := got[w]; !ok {
			msgs = append(msgs, fmt.Sprintf(
				"FR-2 writer %q is gone from pkg/sim: it no longer gives an entity's destination flag true, so FR-1's exactness (a unit is under command exactly while it holds a destination and holds no victim) is no longer backed by this property — reconcile spec.md FR-1/FR-2 with this pin, do not just delete the name",
				w))
		}
	}
	for _, f := range findings {
		if !wantSet[f.Func] {
			msgs = append(msgs, fmt.Sprintf(
				"pkg/sim gained a destination writer FR-2 does not name: %s now gives an entity's destination flag true — spec.md FR-2 says a future writer of a destination on a unit holding no victim inherits FR-1's state and must be triaged as originating it, restoring it, or (like the approach) unable to originate it, before this pin is widened to include it",
				f.String()))
		}
	}
	return msgs
}

// sourceWithWriters builds a single synthetic pkg/sim-shaped file holding one
// function per name, each giving an entity's destination flag true through
// the multi-value shape three of the four real writers use
// (`e.TargetX, e.TargetY, e.HasTarget = x, y, true`) — so a case built from it
// exercises the position-pairing CheckDestinationWriters depends on, not just
// the single-value form.
func sourceWithWriters(names ...string) string {
	var b strings.Builder
	b.WriteString("package sim\n\n")
	for _, n := range names {
		fmt.Fprintf(&b, "func %s(e *Entity, x, y int32) {\n\te.TargetX, e.TargetY, e.HasTarget = x, y, true\n}\n\n", n)
	}
	return b.String()
}

func TestDestinationWritersMatchFR2(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	files, err := LoadSimSources(root)
	if err != nil {
		t.Fatalf("load sim sources: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("loader found no pkg/sim sources; this check would pass by reading nothing")
	}

	findings := CheckDestinationWriters(files)
	if msgs := diffDestinationWriters(findings, wantDestinationWriters); len(msgs) != 0 {
		t.Fatalf("pkg/sim's destination writers no longer match spec.md FR-2:\n%s", strings.Join(msgs, "\n"))
	}
}

// TestDestinationWriterDiffNamesWhatChanged drives the check over literal
// source strings for the three cases that matter (AC-11): an unexpected
// writer appearing, an expected one vanishing, and none present at all — so
// the check is proven to say which without the live tree having to be
// broken.
func TestDestinationWriterDiffNamesWhatChanged(t *testing.T) {
	// THE TWO SYNTHETIC SOURCES ARE BUILT FROM wantDestinationWriters ITSELF
	// and their expected counts are derived from its length. They were written
	// out as literal name lists and literal counts until 0167 added two
	// writers, at which point the pin, two name lists and two counts all had
	// to move together — four places to keep in step for one fact. What each
	// case asserts is the SHAPE of the diagnostic, which does not depend on how
	// many writers the tree has.
	t.Run("an unexpected writer appears", func(t *testing.T) {
		files := map[string]string{
			"pkg/sim/x.go": sourceWithWriters(append(append([]string{}, wantDestinationWriters...), "sendHome")...),
		}
		findings := CheckDestinationWriters(files)
		if want := len(wantDestinationWriters) + 1; len(findings) != want {
			t.Fatalf("expected %d findings (every pinned writer plus the unexpected one), got %d: %v",
				want, len(findings), findings)
		}
		msgs := diffDestinationWriters(findings, wantDestinationWriters)
		if len(msgs) != 1 {
			t.Fatalf("expected exactly 1 diagnostic naming the unexpected writer, got %d: %v", len(msgs), msgs)
		}
		if !strings.Contains(msgs[0], "sendHome") {
			t.Errorf("diagnostic %q does not name the unexpected writer sendHome", msgs[0])
		}
		if !strings.Contains(msgs[0], "gained") {
			t.Errorf("diagnostic %q does not say a writer appeared", msgs[0])
		}
	})

	t.Run("an expected writer vanishes", func(t *testing.T) {
		var kept []string
		for _, n := range wantDestinationWriters {
			if n != "issueSavedGroupDestination" { // the one left out
				kept = append(kept, n)
			}
		}
		files := map[string]string{"pkg/sim/x.go": sourceWithWriters(kept...)}
		findings := CheckDestinationWriters(files)
		if want := len(wantDestinationWriters) - 1; len(findings) != want {
			t.Fatalf("expected %d findings, got %d: %v", want, len(findings), findings)
		}
		msgs := diffDestinationWriters(findings, wantDestinationWriters)
		if len(msgs) != 1 {
			t.Fatalf("expected exactly 1 diagnostic naming the vanished writer, got %d: %v", len(msgs), msgs)
		}
		if !strings.Contains(msgs[0], "issueSavedGroupDestination") {
			t.Errorf("diagnostic %q does not name the vanished writer issueSavedGroupDestination", msgs[0])
		}
		if !strings.Contains(msgs[0], "gone") {
			t.Errorf("diagnostic %q does not say a writer vanished", msgs[0])
		}
	})

	t.Run("no writers at all", func(t *testing.T) {
		files := map[string]string{
			"pkg/sim/x.go": "package sim\n\nfunc walkTo(e *Entity, x, y int32) {\n\te.TargetX, e.TargetY = x, y\n}\n",
		}
		findings := CheckDestinationWriters(files)
		if len(findings) != 0 {
			t.Fatalf("expected 0 findings (a destination assigned with no HasTarget writer beside it), got %d: %v", len(findings), findings)
		}
		msgs := diffDestinationWriters(findings, wantDestinationWriters)
		if len(msgs) != len(wantDestinationWriters) {
			t.Fatalf("expected one diagnostic per pinned writer (%d), got %d: %v", len(wantDestinationWriters), len(msgs), msgs)
		}
		for _, want := range wantDestinationWriters {
			found := false
			for _, m := range msgs {
				if strings.Contains(m, want) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("no diagnostic names vanished writer %q among: %v", want, msgs)
			}
		}
	})
}

// TestCheckDestinationWritersPairsByPosition pins the shape note in plan.md:
// the check must pair a multi-value assignment's left-hand and right-hand
// sides by position, not treat ".HasTarget appears on the left" and "true
// appears on the right" as independent questions. A check built the wrong
// way would either miss the real three-way writers entirely or be fooled by
// an unrelated clearing assignment; these cases are shown failing and passing
// on purpose, on synthetic sources, so the live tree never has to carry a
// planted false case.
func TestCheckDestinationWritersPairsByPosition(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		want    int
		wantAt  string // exact File:Line, checked when want == 1
		wantFor string // exact enclosing function, checked when want == 1
	}{
		{
			name: "single-value form",
			src:  "package sim\n\nfunc walkTo(e *Entity) {\n\te.HasTarget = true\n}\n",
			want: 1, wantAt: "pkg/sim/x.go:4", wantFor: "walkTo",
		},
		{
			name: "multi-value form, HasTarget last (the real shape in combat.go/engage.go/step.go)",
			src:  "package sim\n\nfunc armSwarm(e *Entity, x, y int32) {\n\te.TargetX, e.TargetY, e.HasTarget = x, y, true\n}\n",
			want: 1, wantAt: "pkg/sim/x.go:4", wantFor: "armSwarm",
		},
		{
			name: "multi-value form, HasTarget first — proves pairing is by position, not by 'last element'",
			src:  "package sim\n\nfunc stepWorld(e *Entity, x, y int32) {\n\te.HasTarget, e.TargetX, e.TargetY = true, x, y\n}\n",
			want: 1, wantAt: "pkg/sim/x.go:4", wantFor: "stepWorld",
		},
		{
			name: "multi-value form where the paired right-hand value is not literal true",
			src:  "package sim\n\nfunc approach(e *Entity, x, y int32, ok bool) {\n\te.TargetX, e.HasTarget = x, ok\n}\n",
			want: 0,
		},
		{
			name: "the real clearing shape (step.go clearTarget) must not match",
			src:  "package sim\n\nfunc (e *Entity) clearTarget() {\n\te.TargetX, e.TargetY, e.HasTarget, e.Stall = 0, 0, false, 0\n}\n",
			want: 0,
		},
		{
			name: "an unrelated selector is not HasTarget",
			src:  "package sim\n\nfunc noop(e *Entity) {\n\te.TargetX = true\n}\n",
			want: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			findings := CheckDestinationWriters(map[string]string{"pkg/sim/x.go": tc.src})
			if len(findings) != tc.want {
				t.Fatalf("expected %d finding(s), got %d: %v", tc.want, len(findings), findings)
			}
			if tc.want != 1 {
				return
			}
			got := findings[0].File + ":" + fmt.Sprint(findings[0].Line)
			if got != tc.wantAt {
				t.Errorf("expected finding at %s, got %s", tc.wantAt, got)
			}
			if findings[0].Func != tc.wantFor {
				t.Errorf("expected enclosing function %q, got %q", tc.wantFor, findings[0].Func)
			}
		})
	}
}

// TestCheckDestinationWritersReportsInFileOrder pins that one input yields one
// report: findings come back in path order and then in source order, never in
// Go's map-iteration order — matching CheckSimDeterminism's guarantee.
func TestCheckDestinationWritersReportsInFileOrder(t *testing.T) {
	files := map[string]string{
		"pkg/sim/c.go": "package sim\n\nfunc stepWorld(e *Entity) {\n\te.HasTarget = true\n}\n",
		"pkg/sim/a.go": "package sim\n\nfunc walkTo(e *Entity) {\n\te.HasTarget = true\n}\n",
		"pkg/sim/b.go": "package sim\n\nfunc groupOrder(e *Entity) {\n\te.HasTarget = true\n}\n",
	}
	want := []string{"pkg/sim/a.go:4", "pkg/sim/b.go:4", "pkg/sim/c.go:4"}
	for range 8 { // map order is randomised per iteration, not once per process
		findings := CheckDestinationWriters(files)
		if len(findings) != len(want) {
			t.Fatalf("expected %d findings, got %v", len(want), findings)
		}
		for i, w := range want {
			got := findings[i].File + ":" + fmt.Sprint(findings[i].Line)
			if got != w {
				t.Fatalf("finding %d: expected %s, got %s (full report %v)", i, w, got, findings)
			}
		}
	}
}
