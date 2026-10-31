package archtest

import (
	"os"
	"strings"
	"testing"
)

// painterTwinReasons is every reason CheckPainterTwins gives for files,
// without the list entries the small fixtures leave unmatched.
func painterTwinReasons(files map[string]string) []string {
	var got []string
	for _, v := range CheckPainterTwins(files) {
		if !strings.Contains(v.Reason, "nothing matches it") {
			got = append(got, v.Reason)
		}
	}
	return got
}

const painterThingPlan = "type thingPlan struct{}\nfunc planThing() thingPlan { thingTiles(); return thingPlan{} }\nfunc thingTiles() {}\n"

func TestCheckPainterTwinsNamesEachShape(t *testing.T) {
	cases := []struct {
		name, src, want string
	}{
		{"a twin pair building its own geometry",
			"package ui\nfunc thingTiles() {}\nfunc drawThingGPU(dst *ebiten.Image) { thingTiles() }\nfunc drawThingCPU(dst *image.RGBA) { thingTiles() }\n",
			"drawThingGPU and pkg/ui/x.go:drawThingCPU are painter twins"},
		{"a twin calling the producer's step beside the plan",
			"package ui\n" + painterThingPlan + "func drawThing(dst *ebiten.Image) { planThing() }\nfunc applyThing(dst *image.RGBA) { planThing(); thingTiles() }\n",
			"calls thingTiles, a step of planThing"},
		{"a frame shadow step outside its producer",
			"package ui\nfunc f() { frameShadowMask(nil, nil) }\n", "uses frameShadowMask, a plan step only planFrameShadows may use"},
		{"the frame shadow level outside the backdrop policy",
			"package ui\nfunc f() int { return frameShadowLevel }\n", "uses frameShadowLevel"},
		{"a counted mask outside the mask builder",
			"package ui\nfunc f(m *image.RGBA) { m.Pix[0]++ }\n", "counts a mask in a picture's Pix"},
		{"a level-6 table built beside the policy",
			"package ui\nfunc f() { backdrop.NewLevel(l, m, 6) }\n", "level-6 table itself"},
		{"the dialogue shadow submission outside the GPU executor",
			"package ui\nfunc (s *dialogueBackdropState) f() { s.submitDialogueShadow() }\n", "uses submitDialogueShadow"},
		{"a native-to-wide walk outside the wide plan",
			"package ui\nfunc f(l wideFrameLayout) { for range l.segments() {} }\n", "uses segments, a plan step only wideFrameLayout.plan may use"},
		{"a picture centred on a wider frame outside the dialogue plan",
			"package ui\nfunc f(w int, s image.Point) image.Point { return image.Pt((w-s.X)/2, (frame.H-s.Y)/2) }\n", "centres a picture on a wider frame"},
		{"the native frame centred on a wider one outside the layout",
			"package ui\nfunc f(w int) int { return (w - frame.W) / 2 }\n", "centres the native frame on a wider one"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := painterTwinReasons(map[string]string{"pkg/ui/x.go": tc.src})
			if len(got) == 0 || !strings.Contains(got[0], tc.want) {
				t.Fatalf("CheckPainterTwins = %v, want one naming %q", got, tc.want)
			}
		})
	}
}

func TestCheckPainterTwinsPassesPlanRunningExecutors(t *testing.T) {
	cases := []struct{ name, src string }{
		{"each executor calls the producer",
			"package ui\n" + painterThingPlan + "func drawThingGPU(dst *ebiten.Image) { p := planThing(); _ = p }\nfunc drawThingCPU(dst *image.RGBA) { planThing() }\n"},
		{"each executor takes the plan",
			"package ui\n" + painterThingPlan + "func (p thingPlan) drawThing(dst *ebiten.Image) {}\nfunc (p thingPlan) paintThing(dst *image.RGBA) {}\n"},
		{"an executor delegating to one that runs the plan",
			"package ui\n" + painterThingPlan + "func drawThing(dst *ebiten.Image) { planThing() }\nfunc applyThing(dst *image.RGBA) { runIt(dst) }\nfunc runIt(dst *image.RGBA) { planThing() }\n"},
		{"each executor asks one presenter",
			"package ui\nfunc (v *Viewer) thingPresent() (*image.RGBA, image.Point, bool) { return nil, image.Point{}, false }\nfunc (v *Viewer) drawThing(s *ebiten.Image) { v.thingPresent() }\nfunc (v *Viewer) paintThing(dst *image.RGBA) { v.thingPresent() }\n"},
		{"the native picture centred on the native frame",
			"package ui\nfunc f(s image.Point) image.Point { return image.Pt((frame.W-s.X)/2, (frame.H-s.Y)/2) }\n"},
		{"a sprite shadow level",
			"package ui\nfunc f() { backdrop.NewLevel(l, m, 4) }\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := painterTwinReasons(map[string]string{"pkg/ui/x.go": tc.src}); len(got) != 0 {
				t.Fatalf("CheckPainterTwins = %v, want none", got)
			}
		})
	}
}

func TestCheckPainterTwinsListsMatchAndGoStale(t *testing.T) {
	listed := map[string]string{
		"pkg/ui/viewer.go":      "package ui\nfunc (v *Viewer) drawFrame(s *ebiten.Image) {}\n",
		"pkg/ui/widgetframe.go": "package ui\nfunc drawFrame(dst *image.RGBA, s frameSpec) {}\n",
	}
	if got := painterTwinReasons(listed); len(got) != 0 {
		t.Fatalf("a listed pair that paints two things = %v, want none", got)
	}
	var stale int
	for _, v := range CheckPainterTwins(listed) {
		if strings.Contains(v.Reason, "nothing matches it") {
			stale++
		}
	}
	if want := len(painterPlanProducers) + len(painterNotTwins) - 1; stale != want {
		t.Fatalf("stale list entries = %d, want %d", stale, want)
	}
	if vs := CheckPainterTwins(nil); len(vs) != 1 || !strings.Contains(vs[0].Reason, "no sources scanned") {
		t.Fatalf("CheckPainterTwins(nil) = %v, want one violation", vs)
	}
}

func TestPainterTwinsRunOnePlan(t *testing.T) {
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
	for _, v := range CheckPainterTwins(files) {
		t.Errorf("%s", v)
	}
}
