package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestCheckOrderPhaseReadersNamesEachViolation drives the scan with synthetic
// sources, one case per arm.
func TestCheckOrderPhaseReadersNamesEachViolation(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"a record lookup", "pkg/sim/a.go",
			"package sim\nfunc (w *World) f(e *Entity) bool { _, ok := w.bookCastIndex(e.ID); return ok }\n", "bookCastIndex outside"},
		{"a scroll lookup", "pkg/sim/a.go",
			"package sim\nfunc (w *World) f(e *Entity) bool { _, ok := w.scrollIndex(e.ID); return ok }\n", "scrollIndex outside"},
		{"the old scroll predicate", "pkg/sim/a.go",
			"package sim\nfunc (w *World) f(e *Entity) bool { return w.scrollInFlight(e.ID) }\n", "scrollInFlight"},
		{"a wait comparison", "pkg/sim/a.go",
			"package sim\nfunc f(e *Entity) bool { return e.CastWait != 0 }\n", "CastWait comparison"},
		{"a wait comparison reversed", "pkg/sim/a.go",
			"package sim\nfunc f(e *Entity) bool { return 0 < e.CastWait }\n", "CastWait comparison"},
		{"the pair", "pkg/sim/a.go",
			"package sim\nfunc f(p PendingOrder) bool { return p.Kind == PendingActorCast || p.Kind == PendingCellCast }\n", "queued cast pair"},
		{"a listed record reader", "pkg/sim/actionguard.go",
			"package sim\nfunc (w *World) f(e *Entity) bool { _, ok := w.bookCastIndex(e.ID); return ok }\n", ""},
		{"a second call in a listed file", "pkg/sim/actionguard.go",
			"package sim\nfunc (w *World) f(e *Entity) bool { _, ok := w.bookCastIndex(e.ID); _, ok2 := w.bookCastIndex(e.ID); return ok && ok2 }\n", "bookCastIndex outside"},
		{"the order state itself", "pkg/sim/orderstate.go",
			"package sim\nfunc (w *World) f(e *Entity) bool { _, ok := w.bookCastIndex(e.ID); return ok && e.CastWait != 0 }\n", ""},
		{"a writer of the wait", "pkg/sim/a.go",
			"package sim\nfunc f(e *Entity) { e.CastWait = 0; e.CastWait-- }\n", ""},
		{"another package", "pkg/game/a.go",
			"package game\nfunc f(e *Entity) bool { return e.CastWait != 0 }\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, v := range CheckOrderPhaseReaders(map[string]string{tc.file: tc.src}) {
				got = append(got, v.Reason)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckOrderPhaseReaders = %v, want none", got)
				}
				return
			}
			if len(got) == 0 || !strings.Contains(strings.Join(got, "\n"), tc.want) {
				t.Fatalf("CheckOrderPhaseReaders = %v, want one naming %q", got, tc.want)
			}
		})
	}
	if vs := CheckOrderPhaseReaders(nil); len(vs) != 1 {
		t.Fatalf("CheckOrderPhaseReaders(nil) = %v, want one violation", vs)
	}
}

// TestOrderPhaseReadersAreOrderState is the live-tree half.
func TestOrderPhaseReadersAreOrderState(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadDrawnTextSources(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := files[orderPhaseRuleFile]; !ok {
		t.Fatalf("%s is missing; the scan guards nothing", orderPhaseRuleFile)
	}
	for _, v := range CheckOrderPhaseReaders(files) {
		t.Errorf("%s", v)
	}
}
