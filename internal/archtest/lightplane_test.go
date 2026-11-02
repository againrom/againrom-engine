package archtest

import (
	"os"
	"strings"
	"testing"
)

// lightPlaneSource is the one plane's sites as the guard admits them.
const lightPlaneSource = "package ui\n" +
	"type lightPlane struct{ cellLevels, cellTerrain map[P]float32; vertexStamps, pathStamps map[P]uint8 }\n" +
	"type Viewer struct{ frameLight lightPlane }\n" +
	"func (v *Viewer) SetLightSources(rows []LightSource) { v.frameLight = buildLightPlane(rows) }\n" +
	"func (v *Viewer) SetLightClock(t uint64) {}\n" +
	"func buildLightPlane(rows []LightSource) lightPlane { var p lightPlane; p.cellLevels, p.cellTerrain, p.vertexStamps, p.pathStamps = nil, nil, nil, nil; return p }\n" +
	"func (v *Viewer) mergeLight(r R, c P) ([4]float32, float32) { _, _, _, _ = v.frameLight.cellLevels, v.frameLight.cellTerrain, v.frameLight.vertexStamps, v.frameLight.pathStamps; return [4]float32{}, 1 }\n" +
	"func (v *Viewer) cornerScales(x, y int) [4]float32 { sc, _ := v.mergeLight(0, P{}); return sc }\n" +
	"func (v *Viewer) spellSpriteFactor(c P) float32 { _, f := v.mergeLight(1, c); return f }\n"

const lightPushSource = "package game\nfunc (mw *mapWorld) push() { mw.view.SetLightSources(mw.lightSources(nil)) }\n"

// TestCheckLightPlaneNamesEachCopy plants each shape a second light setter,
// plane or merge takes, and each site the one plane keeps.
func TestCheckLightPlaneNamesEachCopy(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"the one plane's sites", lightPlaneFile, lightPlaneSource, ""},
		{"the one call", lightPushFile, lightPushSource, ""},
		{"a second light setter", "pkg/ui/a.go",
			"package ui\nfunc (v *Viewer) SetTorchLighting(c []C) {}\n", "second light setter"},
		{"the setter without its plane file", "pkg/ui/a.go",
			"package ui\nfunc (v *Viewer) SetLightSources(rows []LightSource) {}\n", ""},
		{"a former setter called", "pkg/game/a.go",
			"package game\nfunc f(mw *mapWorld) { mw.view.SetLightStamps(nil) }\n", "former light setter"},
		{"the cell setter called", "pkg/game/a.go",
			"package game\nfunc f(mw *mapWorld) { mw.view.SetSpellLighting(nil) }\n", "former light setter"},
		{"the setter called outside the push", "pkg/game/a.go",
			"package game\nfunc (mw *mapWorld) relight() { mw.view.SetLightSources(nil) }\n", "light setter is called here"},
		{"the setter called twice in the push", lightPushFile,
			"package game\nfunc (mw *mapWorld) push() { mw.view.SetLightSources(nil); mw.view.SetLightSources(nil) }\n", "called twice"},
		{"the plane read by another reader", "pkg/ui/a.go",
			"package ui\nfunc (v *Viewer) glow(c P) uint8 { return v.frameLight.vertexStamps[c] }\n", "light plane is read"},
		{"the plane field read in the plane's file", lightPlaneFile,
			lightPlaneSource + "func (v *Viewer) torch(c P) float32 { return v.frameLight.cellLevels[c] }\n", "light plane is read"},
		{"a former plane field", "pkg/ui/a.go",
			"package ui\nfunc (v *Viewer) f() { v.boltLightStamps = nil }\n", "former light plane field"},
		{"a second plane", "pkg/ui/a.go",
			"package ui\nvar spare lightPlane\n", "second light plane"},
		{"the builder asked elsewhere", "pkg/ui/a.go",
			"package ui\nfunc (v *Viewer) f(r []LightSource) { _ = buildLightPlane(r) }\n", "light plane is built"},
		{"the merge asked elsewhere", "pkg/ui/a.go",
			"package ui\nfunc (v *Viewer) minimapLight(c P) float32 { _, f := v.mergeLight(1, c); return f }\n", "light merge is asked"},
		{"the plane file without its setter", lightPlaneFile,
			"package ui\nfunc buildLightPlane(rows []LightSource) lightPlane { var p lightPlane; return p }\n", "SetLightSources is gone"},
		{"the push without its call", lightPushFile,
			"package game\nfunc (mw *mapWorld) push() {}\n", "no longer calls"},
		{"another package's light field", "pkg/sim/a.go",
			"package sim\nfunc f(a arm) SpellLighting { return a.light }\n", ""},
		{"a test file", "pkg/ui/a_test.go",
			"package ui\nfunc (v *Viewer) SetLightStamps(s []LightStamp) { v.frameLight = buildLightPlane(nil) }\n", ""},
		{"outside pkg", "cmd/x/a.go",
			"package main\nfunc f(v *ui.Viewer) { v.SetLightStamps(nil) }\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vs := CheckLightPlane(map[string]string{tc.file: tc.src})
			if tc.want == "" {
				if len(vs) != 0 {
					t.Fatalf("CheckLightPlane = %v, want none", vs)
				}
				return
			}
			if len(vs) != 1 || !strings.Contains(vs[0].Reason, tc.want) || !strings.HasPrefix(vs[0].From, tc.file) {
				t.Fatalf("CheckLightPlane = %v, want one in %s naming %q", vs, tc.file, tc.want)
			}
		})
	}
	if vs := CheckLightPlane(nil); len(vs) != 1 {
		t.Fatalf("CheckLightPlane(nil) = %v, want one violation", vs)
	}
}

// TestLightPlaneHasOneSetter is the live-tree half.
func TestLightPlaneHasOneSetter(t *testing.T) {
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
	for _, f := range []string{lightPlaneFile, lightPushFile} {
		if _, ok := files[f]; !ok {
			t.Fatalf("%s is not among the scanned sources", f)
		}
	}
	for _, v := range CheckLightPlane(files) {
		t.Errorf("%s", v)
	}
}
