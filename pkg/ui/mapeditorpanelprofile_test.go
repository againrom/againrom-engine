package ui

import (
	"image"
	"os"
	"reflect"
	"testing"
)

// Each part of the panel profile keeps its own absence rule when its entry
// does not read.
func TestReadPanelProfileKeepsEachPartsAbsenceRule(t *testing.T) {
	defaults := readPanelProfile(nil, os.ErrNotExist)
	if !defaults.plain || defaults.windows != nil || defaults.floats != [len(floatWindows)]floatKept{} {
		t.Fatalf("defaults %+v", defaults)
	}
	for _, c := range []struct {
		name string
		data string
		want func(p panelProfile) bool
	}{
		{"a broken side gives every side its default",
			`{"left":{"minimized":false,"width":300},"right":{"width":"wide"},"bottom":{"minimized":false}}`,
			func(p panelProfile) bool { return p.sides == defaults.sides }},
		{"a side that is no object gives every side its default",
			`{"left":{"minimized":false,"width":300},"bottom":5}`,
			func(p panelProfile) bool { return p.sides == defaults.sides }},
		{"a broken side costs no float, view or size",
			`{"left":{"minimized":"x","float":{"x":1,"y":2,"width":300,"height":200}},"right":{"view":"technical","width":"x"},"windows":{"units":{"width":400,"height":300}}}`,
			func(p panelProfile) bool {
				return p.sides == defaults.sides && p.floats[0].at == image.Rect(1, 2, 301, 202) && !p.plain && p.windows["units"] == image.Pt(400, 300)
			}},
		{"a broken float costs that window only",
			`{"left":{"minimized":false,"width":300,"float":{"x":"a"}},"right":{"float":{"x":5,"y":6,"width":100,"height":90}},"toolbar":{"hidden":true,"float":"no"}}`,
			func(p panelProfile) bool {
				return p.sides[panelLeft].Restored && p.floats[0] == floatKept{} && p.floats[1].at == image.Rect(5, 6, 105, 96) && p.floats[1].floating && p.floats[2] == floatKept{}
			}},
		{"a float the window cannot take is docked and keeps hidden",
			`{"toolbar":{"hidden":true,"float":{"x":0,"y":0,"width":0,"height":9}},"left":{"float":{"x":99999,"y":0,"width":9,"height":9}}}`,
			func(p panelProfile) bool {
				return p.floats[2] == floatKept{hidden: true} && p.floats[0] == floatKept{}
			}},
		{"a broken view gives the plain view",
			`{"right":{"view":7}}`, func(p panelProfile) bool { return p.plain }},
		{"a view other than technical is plain",
			`{"right":{"view":"other"}}`, func(p panelProfile) bool { return p.plain }},
		{"the technical view reads",
			`{"right":{"view":"technical"}}`, func(p panelProfile) bool { return !p.plain }},
		{"a broken size gives the default and costs no other",
			`{"windows":{"units":{"width":"x","height":300},"structures":{"width":500,"height":600}}}`,
			func(p panelProfile) bool {
				_, units := p.windows["units"]
				return !units && p.windows["structures"] == image.Pt(500, 600)
			}},
		{"a size beyond the bound gives the default",
			`{"windows":{"units":{"width":16385,"height":300}}}`,
			func(p panelProfile) bool { return p.windows == nil }},
		{"windows that is no object gives no size",
			`{"windows":[1]}`, func(p panelProfile) bool { return p.windows == nil }},
		{"a file that is no object gives the defaults",
			`[1,2]`, func(p panelProfile) bool { return reflect.DeepEqual(p, defaults) }},
	} {
		if p := readPanelProfile([]byte(c.data), nil); !c.want(p) {
			t.Errorf("%s: %+v", c.name, p)
		}
	}
}

// A profile written from a state reads back to the same state, and the bytes
// for that state are fixed.
func TestPanelProfileRoundTripAndBytes(t *testing.T) {
	const profile = `{"bottom":{"minimized":true,"width":180},"left":{"float":{"height":400,"width":300,"x":10,"y":20},"minimized":true,"width":260},"right":{"minimized":false,"view":"technical","width":600},"toolbar":{"float":{"height":40,"width":500,"x":30,"y":40},"hidden":true},"windows":{"structures":{"height":333,"width":444},"units":{"height":300,"width":400}}}` + "\n"
	e, _, _ := panelsEditor(t, 1280, 800, "")
	e.SetPanelProfile([]byte(profile), nil, nil)
	got := e.panelStateBytes()
	if string(got) != profile {
		t.Fatalf("wrote %q, want %q", got, profile)
	}
	f, _, _ := panelsEditor(t, 1280, 800, "")
	f.SetPanelProfile(got, nil, nil)
	if f.panels.state != e.panels.state || f.panels.floats != e.panels.floats || f.panels.toolbarHidden != e.panels.toolbarHidden || f.props.plain != e.props.plain || !reflect.DeepEqual(f.kept, e.kept) {
		t.Fatalf("read back %+v from %+v", f.panels, e.panels)
	}
	if string(f.panelStateBytes()) != profile {
		t.Fatalf("second write %q", f.panelStateBytes())
	}
}
