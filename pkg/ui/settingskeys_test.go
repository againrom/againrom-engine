package ui

// The mission map's SETTINGS keys and the `Pause` key (round 3).
//
// `keyboard.tsv` gives seven settings toggles, all of them `Ctrl`-modified, and
// a `Pause` key that shows a modal text. This file witnesses the four of them
// this build has a destination for — `Ctrl`+`W` retreat, `Ctrl`+`N` day/night,
// `Ctrl`+`H` show-health and `Ctrl`+`L` flying damage — plus `Pause`, plus the `Ctrl`
// guard that keeps the unmodified numpad speed step off the modified pair the
// original spends on something else.
//
// THE BINDING AND THE ARM ARE WITNESSED SEPARATELY, because no test may open a
// window: the binding is an Ebitengine call and is read from the SOURCE, and
// the arm is driven through a.step with an appInput literal. Neither half alone
// says the key works — a binding with no arm sets a field nothing reads, and an
// arm with no binding is reachable only from a test.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/frame"
)

// bindingSource is the source text of each field of readAppInput's returned
// composite literal, keyed by field name.
//
// IT READS THE SOURCE for TestTheFourKeysAreBoundOnceEach's own reason: the
// bindings are `inpututil.IsKeyJustPressed` calls and no test may open a
// window. The scan is over parsed syntax and the text is cut from the file by
// the node's own byte offsets, so a key named in a comment is not a finding.
func bindingSource(t *testing.T) map[string]string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "app.go", nil, 0)
	if err != nil {
		t.Fatalf("parse app.go: %v", err)
	}
	src, err := os.ReadFile("app.go")
	if err != nil {
		t.Fatalf("read app.go: %v", err)
	}

	out := map[string]string{}
	ast.Inspect(f, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "readAppInput" {
			return true
		}
		ast.Inspect(fn.Body, func(m ast.Node) bool {
			kv, ok := m.(*ast.KeyValueExpr)
			if !ok {
				return true
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok {
				return true
			}
			lo := fset.Position(kv.Value.Pos()).Offset
			hi := fset.Position(kv.Value.End()).Offset
			out[key.Name] = string(src[lo:hi])
			return true
		})
		return false
	})
	if len(out) == 0 {
		t.Fatal("readAppInput's composite literal has no fields — the scan found nothing to check")
	}
	return out
}

// TestTheSettingsKeysCarryTheirModifier is the binding half of G7's settings
// group: each of the three settings this build implements is bound to the key
// `keyboard.tsv` names AND to the `Ctrl` that file requires of it, and the two
// keys that must NOT be modified are not.
//
// THE MODIFIER IS PART OF THE BINDING AND NOT A DETAIL OF IT. `keyboard.tsv`
// rows 47..53 all read "Ctrl held", and rows 21 and 22 read "Ctrl not held";
// row 20's `Pause` names no modifier at all. A binding that names the right
// letter and drops the modifier reaches the setting from a bare press, which
// is the exact defect this round found on bare `N` — a key of this build's
// occupying a letter the original spends under `Ctrl`.
//
// It asserts the guard's POLARITY, not merely that ctrlHeld is mentioned:
// `Faster` and `Slower` must carry the NEGATED read and the three settings the
// plain one, and swapping either way is what the two sub-cases separate.
func TestTheSettingsKeysCarryTheirModifier(t *testing.T) {
	got := bindingSource(t)
	for _, tc := range []struct {
		field   string
		key     string
		ctrl    bool // the plain read must be present
		notCtrl bool // the negated read must be present
		row     string
	}{
		{"Retreat", "ebiten.KeyW", true, false, "MENU-057, Ctrl+W cycle retreat mode"},
		{"Formation", "ebiten.KeyF", true, false, "MENU-057, Ctrl+F cycle formation"},
		{"Smoothing", "ebiten.KeyO", true, false, "MENU-057, Ctrl+O toggle smoothing"},
		{"AutoHealing", "ebiten.KeyU", true, false, "MENU-057, Ctrl+U cycle autohealing"},
		{"TimeFlow", "ebiten.KeyN", true, false, "row 50, Ctrl+N toggle day/night changes"},
		{"ShowHealth", "ebiten.KeyH", true, false, "row 48, Ctrl+H toggle show-health setting"},
		{"Numerals", "ebiten.KeyL", true, false, "MENU-057, Ctrl+L toggle flying damage"},
		{"PauseText", "ebiten.KeyPause", false, false, "row 20, Pause shows main.txt[119]"},
		{"Faster", "ebiten.KeyNumpadAdd", false, true, "row 21, numpad + with Ctrl NOT held"},
		{"Slower", "ebiten.KeyNumpadSubtract", false, true, "row 22, numpad - with Ctrl NOT held"},
	} {
		t.Run(tc.field, func(t *testing.T) {
			expr, ok := got[tc.field]
			if !ok {
				t.Fatalf("readAppInput names no field %s — the binding is gone, "+
					"so %s reaches nothing", tc.field, tc.row)
			}
			if !strings.Contains(expr, tc.key) {
				t.Errorf("%s is bound to %q, which does not name %s (%s)",
					tc.field, expr, tc.key, tc.row)
			}
			plain := strings.Contains(expr, "ctrlHeld()") &&
				!strings.Contains(expr, "!ctrlHeld()")
			negated := strings.Contains(expr, "!ctrlHeld()")
			if tc.ctrl && !plain {
				t.Errorf("%s is bound to %q with no unnegated ctrlHeld() — "+
					"a bare press reaches it, and %s requires Ctrl held", tc.field, expr, tc.row)
			}
			if tc.notCtrl && !negated {
				t.Errorf("%s is bound to %q with no !ctrlHeld() — "+
					"a Ctrl-held press reaches it, and %s requires Ctrl NOT held",
					tc.field, expr, tc.row)
			}
			if !tc.ctrl && !tc.notCtrl && strings.Contains(expr, "ctrlHeld()") {
				t.Errorf("%s is bound to %q, which reads Ctrl — %s names no modifier",
					tc.field, expr, tc.row)
			}
		})
	}
}

// TestBareNIsNoLongerBound is the other half of the day/night move: the letter
// the binding LEFT is left free rather than kept beside the new one.
//
// A key named zero times is a binding that moved off and never landed; a key
// named twice is one that was copied rather than moved. `ebiten.KeyN` must be
// named exactly once, by TimeFlow, and that one naming must carry Ctrl — which
// is the case above. Together the two say bare `N` does nothing.
func TestBareNIsNoLongerBound(t *testing.T) {
	got := bindingSource(t)
	naming := []string{}
	for field, expr := range got {
		if strings.Contains(expr, "ebiten.KeyN)") {
			naming = append(naming, field)
		}
	}
	if len(naming) != 1 || naming[0] != "TimeFlow" {
		t.Errorf("ebiten.KeyN is named by %v, want exactly [TimeFlow] — "+
			"a second naming is a bare-N binding this build no longer has", naming)
	}
}

// TestShowHealthKeyHidesTheHealthBars is the arm half of `Ctrl`+`H`: the
// snapshot's field, pushed through the front-end's own map arm, stops the bars
// being placed at all and a second press brings them back.
//
// IT COUNTS PLACED RECTANGLES rather than asking the viewer what its flag says.
// A flag that moves and a walk that ignores it is exactly the shape this
// witness exists to catch, so the observed value is healthBarScreenRects'
// output — what the frame would paint — and not HealthBarsShown.
func TestShowHealthKeyHidesTheHealthBars(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	v := fiViewer(t)
	v.SetEntities(fiSnapshot())
	a := NewApp("t", appAssets(t), appRows(1), nil)
	a.Layout(frame.W, frame.H)
	a.flow.viewer = v
	a.flow.screen = ScreenMap

	before, _ := v.healthBarScreenRects()
	if len(before) == 0 {
		t.Fatal("setup: no health bars are placed at all, so hiding them would prove nothing")
	}

	a.step(appInput{ShowHealth: true}, now)
	if got, _ := v.healthBarScreenRects(); len(got) != 0 {
		t.Fatalf("%d health bars survived the show-health key, want 0", len(got))
	}
	if v.HealthBarsShown() {
		t.Error("HealthBarsShown still reports on after the key")
	}

	// A frame naming no key moves nothing, which is what says the arm reads an
	// edge rather than a level.
	a.step(appInput{}, now)
	if got, _ := v.healthBarScreenRects(); len(got) != 0 {
		t.Fatalf("%d health bars came back over a frame with no key, want 0", len(got))
	}

	a.step(appInput{ShowHealth: true}, now)
	if got, _ := v.healthBarScreenRects(); len(got) != len(before) {
		t.Fatalf("%d health bars after the second press, want %d — the switch is a flip, "+
			"and nothing accrues while it is off", len(got), len(before))
	}
}

// TestPauseKeyShowsTheModalTextAndOneDismissalClosesIt is the arm half of the
// `Pause` key: the words reach the notice window, and the player gets out.
//
// THE SEAM'S ADVANCE COUNT IS THE LOAD-BEARING ASSERTION. A notice this tier
// raised is closed by this tier and the MapAdvance seam is never consulted, and
// the driver behind that seam answers NoticeStay for a notice its own mission
// record does not know about. So a dismissal routed to the seam would leave the
// box up forever in production. noticeSeam.advance clears the notice itself on
// a NoticeStay, which would hide that in a test asking only whether the notice
// closed — the count is what tells the two apart.
func TestPauseKeyShowsTheModalTextAndOneDismissalClosesIt(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	for _, tc := range []struct {
		name string
		in   appInput
	}{
		{"RETURN", appInput{Enter: true}},
		{"ESCAPE", appInput{Escape: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, seam := noticeApp(t)
			words := a.flow.viewer.Words()
			if words.PauseNotice == "" {
				t.Fatal("setup: the viewer's word set states no pause text")
			}

			a.step(appInput{PauseText: true}, now)
			text, kind, open := a.flow.viewer.NoticeState()
			if !open {
				t.Fatal("the Pause key opened no notice")
			}
			if text != words.PauseNotice {
				t.Errorf("the notice states %q, want the word set's own %q",
					text, words.PauseNotice)
			}
			if kind != NoticeDialogue {
				t.Errorf("the notice is kind %v, want NoticeDialogue — "+
					"an outcome notice ends the mission on dismissal", kind)
			}
			if !a.flow.popupOpen() {
				t.Error("the pause notice does not hold the world: popupOpen is false")
			}

			a.step(tc.in, now)
			if _, _, stillOpen := a.flow.viewer.NoticeState(); stillOpen {
				t.Error("one dismissal did not close the pause notice")
			}
			if seam.advances != 0 {
				t.Errorf("%d advances through the MapAdvance seam, want 0 — "+
					"the driver holds no page for a notice this tier raised, and "+
					"answers NoticeStay without clearing it", seam.advances)
			}
			if a.Screen() != ScreenMap {
				t.Errorf("screen = %v, want ScreenMap — closing this notice navigates nowhere",
					a.Screen())
			}
		})
	}
}

// TestPauseKeyIsRefusedWhileANoticeStands is `keyboard.tsv` row 20's "repeat
// blocked by modal state", in this build's terms: the key is read below the map
// arm's popup gate, so a press made while any popup stands never reaches the
// raise.
//
// The case that matters is a MISSION's own dialogue: replacing it would drop a
// page of event text the player has not read, and the driver would then page
// from a part number whose window is gone.
func TestPauseKeyIsRefusedWhileANoticeStands(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	a, seam := noticeApp(t)
	seam.v.SetNotice("the mission's own words", NoticeDialogue)

	a.step(appInput{PauseText: true}, now)
	text, _, open := a.flow.viewer.NoticeState()
	if !open {
		t.Fatal("the standing notice was closed by a Pause press")
	}
	if text != "the mission's own words" {
		t.Errorf("the notice now states %q — the Pause key replaced a mission's dialogue", text)
	}
	if a.flow.selfNotice {
		t.Error("the refused raise still claimed the notice as this tier's, " +
			"so the driver would never be asked to page it")
	}
}

// TestMarqueeCursorFollowsTheClaimsOwnRectTest is `AI-CURSOR-230`'s replacement
// (1) as that claim states it: the marquee flag non-zero AND `IsRectEmpty`
// false on the normalised rectangle.
//
// THE BAND IS THE POINT. Through this story's round 2 the cursor read
// marqueeScreenRects, whose threshold is the RELEASE's — `screenW*10/640`, 16
// pixels here (`AI-INPUT-121`) — so a drag of 1 through 16 pixels showed the
// ordinary hover cursor where the original shows `default`. The first case is
// inside that band and fails against the old predicate; the last two are the
// Win32 predicate's own zero-area arms, which a "any movement at all" reading
// would get wrong in the other direction.
func TestMarqueeCursorFollowsTheClaimsOwnRectTest(t *testing.T) {
	for _, tc := range []struct {
		name           string
		dx, dy         int
		want           bool
		why            string
		skipUnlessSlop bool
	}{
		{name: "one pixel on each axis", dx: 1, dy: 1, want: true,
			why: "IsRectEmpty is false as soon as both sides are non-zero"},
		{name: "inside the release threshold", dx: 5, dy: 5, want: true,
			why: "the release threshold does not gate the cursor", skipUnlessSlop: true},
		{name: "beyond the release threshold", dx: 40, dy: 40, want: true,
			why: "a full marquee replaces the cursor"},
		{name: "straight down, no width", dx: 0, dy: 40, want: false,
			why: "a normalised rectangle with zero width IS empty"},
		{name: "straight across, no height", dx: 40, dy: 0, want: false,
			why: "a normalised rectangle with zero height IS empty"},
		{name: "no travel at all", dx: 0, dy: 0, want: false,
			why: "the rectangle is a point"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := marqueeCursorFixture(t)
			if tc.skipUnlessSlop && tc.dx > v.marqueeSlop() {
				t.Skipf("this fixture's marqueeSlop is %d, so dx=%d is not inside the band",
					v.marqueeSlop(), tc.dx)
			}
			v.dragging, v.boxing = true, true
			v.pressX, v.pressY = 100, 100
			v.dragX, v.dragY = 100+tc.dx, 100+tc.dy
			if got := v.marqueeCursorLive(); got != tc.want {
				t.Errorf("marqueeCursorLive = %v, want %v — %s", got, tc.want, tc.why)
			}
			wantName := "select"
			if tc.want {
				wantName = "default"
			}
			if name, ok := v.missionHoverCursor(); !ok || name != wantName {
				t.Errorf("missionHoverCursor over the plain unit with a %dx%d drag = %q,%v, want %q,true — %s",
					tc.dx, tc.dy, name, ok, wantName, tc.why)
			}
		})
	}

	// The three conditions besides the rectangle, each one alone enough to say
	// there is no live marquee at all.
	for _, tc := range []struct {
		name  string
		setup func(v *Viewer)
	}{
		{"no gesture", func(v *Viewer) { v.dragging = false }},
		{"not a boxing gesture", func(v *Viewer) { v.boxing = false }},
		{"the inventory window took it", func(v *Viewer) { v.invGrab = true }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := marqueeCursorFixture(t)
			v.dragging, v.boxing = true, true
			v.pressX, v.pressY = 100, 100
			v.dragX, v.dragY = 140, 140
			if !v.marqueeCursorLive() {
				t.Fatal("setup: the unmodified gesture is not live")
			}
			if name, _ := v.missionHoverCursor(); name != "default" {
				t.Fatalf("setup: the unmodified gesture gives %q, want default", name)
			}
			tc.setup(v)
			if v.marqueeCursorLive() {
				t.Error("marqueeCursorLive is true with no live marquee")
			}
			if name, ok := v.missionHoverCursor(); !ok || name != "select" {
				t.Errorf("missionHoverCursor with no live marquee = %q,%v, want select,true", name, ok)
			}
		})
	}
}

// marqueeCursorFixture puts the pointer on a plain unit with nothing selected,
// where the decoded cascade answers `select`. That is what makes the assertions
// above observe the PRODUCTION result rather than the predicate: `default` at
// this hover point can only have come from missionHoverCursor's marquee arm,
// and `select` can only mean that arm did not fire.
func marqueeCursorFixture(t *testing.T) *Viewer {
	t.Helper()
	_, v, _ := atOnMap(t)
	v.SetEntities(mcEntities())
	x, y := cellPoint(v, mcPlainCol, mcPlainRow)
	v.cursorX, v.cursorY, v.hasCursor = x, y, true
	if name, ok := v.missionHoverCursor(); !ok || name != "select" {
		t.Fatalf("premise: the plain-unit hover with no gesture = %q,%v, want select,true", name, ok)
	}
	return v
}
