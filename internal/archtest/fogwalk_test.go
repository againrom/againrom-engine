package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestCheckFogWalkNamesTheOneWalk drives the scan with synthetic sources.
func TestCheckFogWalkNamesTheOneWalk(t *testing.T) {
	const reader = "package sim\nvar fogSight = sightReader{}\n"
	const stamp = "func (w *World) fogStamp(s []byte, v uint32, e *Entity) { var acc [1]int32; w.marchInto(fogSight, nil, &acc, cellOf(e), 1) }\n"
	cases := []struct {
		name, file, src, want string
	}{
		{"the one walk", "pkg/sim/sight.go", reader + stamp, ""},
		{"a second walk", "pkg/sim/sight.go", reader + stamp +
			"func (w *World) other(s []byte, e *Entity) { w.marchSight(fogSight, s, cellOf(e), 6) }\n", "fogSight is used outside fogStamp"},
		{"a walk in another file", "pkg/sim/a.go",
			"package sim\nfunc (w *World) other(s []byte, e *Entity) { w.marchSight(fogSight, s, cellOf(e), 6) }\n", "fogSight is used outside fogStamp"},
		{"a game branch", "pkg/sim/sight.go", reader +
			"func (w *World) fogStamp(s []byte, v uint32, e *Entity) { if w.SecondGame() { return }; var acc [1]int32; w.marchInto(fogSight, nil, &acc, cellOf(e), 1) }\n", "fogStamp reads the running game (SecondGame)"},
		{"two marches", "pkg/sim/sight.go", reader +
			"func (w *World) fogStamp(s []byte, v uint32, e *Entity) { var acc [1]int32; w.marchInto(fogSight, nil, &acc, cellOf(e), 1); w.marchSight(fogSight, s, cellOf(e), 6) }\n", "fogStamp marches 2 times"},
		{"no stamp", "pkg/sim/sight.go", reader, "no fogStamp"},
		{"a test", "pkg/sim/sight_test.go",
			"package sim\nfunc f(w *World) { w.marchSight(fogSight, nil, cell{}, 6) }\n", "no fogStamp"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := map[string]string{tc.file: tc.src}
			if tc.file != "pkg/sim/sight.go" && tc.name != "a test" {
				files["pkg/sim/sight.go"] = reader + stamp
			}
			var got []string
			for _, v := range CheckFogWalk(files) {
				got = append(got, v.Reason)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckFogWalk = %v, want none", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0], tc.want) {
				t.Fatalf("CheckFogWalk = %v, want one naming %q", got, tc.want)
			}
		})
	}
}

// TestFogWalkIsOneWalk is the live-tree half.
func TestFogWalkIsOneWalk(t *testing.T) {
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
	if !strings.Contains(files[fogWalkFile], "func (w *World) fogStamp(") {
		t.Fatal("pkg/sim/sight.go holds no fogStamp; the scan guards nothing")
	}
	for _, v := range CheckFogWalk(files) {
		t.Errorf("%s: %s", v.From, v.Reason)
	}
}
