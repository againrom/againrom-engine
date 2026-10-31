package ui

import (
	"fmt"
	"image"
	"strings"
	"testing"
)

// savePointShopTown is a town showing its Shop page over a mission.
type savePointShopTown struct{ *stubTown }

func (savePointShopTown) OpenMissionShop() bool { return true }
func (savePointShopTown) MissionShopOpen() bool { return true }

var savePointRowNames = [...]string{"open", "no screen", "no viewer", "no town", "map popup", "map notice",
	"town dialogue", "mission shop", "town cannot save", "not a campaign"}

// TestSavePointGateRows drives each gate row for each save kind and states
// its answer, written apart from the gate's table: open, silent, or asked,
// where asked shows the admission's words on the screen that asked.
func TestSavePointGateRows(t *testing.T) {
	const reason = "Close the message before saving."
	rows := []TownRow{{Text: "town", Choosable: true}}
	town := func(h *haltFix, s TownScreen) {
		h.a.SetTown(s)
		if !h.a.flow.showTown("") {
			t.Fatal("town unavailable")
		}
	}
	menuOver := func(back Screen) func(h *haltFix) {
		return func(h *haltFix) { h.a.flow.screen, h.a.flow.menuBack = ScreenGameMenu, back }
	}
	const (
		o = "open"
		n = "silent/no screen"
	)
	// Kinds: menu action, menu window, menu row, Save key, Save key with
	// Load, Load key, quick keys, autosave.
	cases := []struct {
		name  string
		setup func(h *haltFix)
		want  [saveKinds]string
	}{
		{"main menu", func(h *haltFix) { h.a.flow.screen = ScreenMenu },
			[saveKinds]string{n, n, o, n, n, n, n, n}},
		{"map without a viewer", func(h *haltFix) { h.a.flow.viewer = nil },
			[saveKinds]string{n, n, o, "silent/no viewer", "silent/no viewer", "silent/no viewer", "silent/no viewer", "silent/no viewer"}},
		{"campaign map", func(h *haltFix) {},
			[saveKinds]string{n, n, o, o, o, o, o, o}},
		{"map notice", func(h *haltFix) { h.v.SetNotice("a message", NoticeDialogue) },
			[saveKinds]string{n, n, o, "asked/map popup", "asked/map popup", "silent/map popup", "silent/map popup", "silent/map notice"}},
		{"map popup without a notice", func(h *haltFix) { h.v.menuUp = true },
			[saveKinds]string{n, n, o, "asked/map popup", "asked/map popup", "silent/map popup", "silent/map popup", o}},
		{"map not a campaign", func(h *haltFix) { h.v.SetGameMenuContext(func() GameMenuContext { return GameMenuContext{} }) },
			[saveKinds]string{n, n, o, "silent/not a campaign", "silent/not a campaign", "silent/not a campaign", "silent/not a campaign", o}},
		{"town screen without a town", func(h *haltFix) { h.a.flow.screen, h.a.flow.town = ScreenTown, nil },
			[saveKinds]string{n, n, o, o, o, o, o, "silent/no town"}},
		{"town dialogue", func(h *haltFix) {
			town(h, &savePointTalkTown{stubTown: &stubTown{rows: rows}, pic: image.NewRGBA(image.Rect(0, 0, 40, 20))})
		}, [saveKinds]string{n, n, o, "asked/town dialogue", "asked/town dialogue", "silent/town dialogue", "silent/town dialogue", "silent/town dialogue"}},
		{"mission shop", func(h *haltFix) { town(h, savePointShopTown{&stubTown{rows: rows}}) },
			[saveKinds]string{n, n, o, "silent/mission shop", "silent/mission shop", "silent/mission shop", "silent/mission shop", o}},
		{"town that cannot save", func(h *haltFix) { town(h, &savePointTown{stubTown: &stubTown{rows: rows}}) },
			[saveKinds]string{n, n, "silent/town cannot save", "asked/town cannot save", "silent/town cannot save", o, "silent/town cannot save", "silent/town cannot save"}},
		{"town that can save", func(h *haltFix) { town(h, &savePointTown{stubTown: &stubTown{rows: rows}, ready: true}) },
			[saveKinds]string{n, n, o, o, o, o, o, o}},
		{"game menu over the map", menuOver(ScreenMap),
			[saveKinds]string{o, o, o, n, n, n, n, n}},
		{"game menu over a map that is not a campaign", func(h *haltFix) {
			h.v.SetGameMenuContext(func() GameMenuContext { return GameMenuContext{} })
			menuOver(ScreenMap)(h)
		}, [saveKinds]string{o, o, o, n, n, n, n, n}},
		{"game menu over a town that cannot save", func(h *haltFix) {
			town(h, &savePointTown{stubTown: &stubTown{rows: rows}})
			menuOver(ScreenTown)(h)
		}, [saveKinds]string{"asked/town cannot save", "silent/town cannot save", "silent/town cannot save", n, n, n, n, n}},
		{"game menu over a town that can save", func(h *haltFix) {
			town(h, &savePointTown{stubTown: &stubTown{rows: rows}, ready: true})
			menuOver(ScreenTown)(h)
		}, [saveKinds]string{o, o, o, n, n, n, n, n}},
		{"game menu over the picker", menuOver(ScreenPicker),
			[saveKinds]string{n, n, o, n, n, n, n, n}},
	}
	for _, tc := range cases {
		for k := saveKind(0); k < saveKinds; k++ {
			t.Run(fmt.Sprintf("%s/kind %d", tc.name, k), func(t *testing.T) {
				h := newHaltFix(t, haltOpts{})
				tc.setup(h)
				seams := &refusalSeams{reason: reason}
				seams.install(h.a)
				f := h.a.flow
				f.msg = ""
				onMap := f.screen == ScreenMap && f.viewer != nil
				lines := 0
				if onMap {
					lines = len(f.viewer.MessageLines())
				}
				g := f.savePoint(k)
				got := "open"
				if !g.open() {
					got = "silent/" + savePointRowNames[g.row]
					if g.shown {
						got = "asked/" + savePointRowNames[g.row]
					}
				}
				if got != tc.want[k] {
					t.Fatalf("answer %q, want %q", got, tc.want[k])
				}
				shown := f.msg
				if onMap {
					shown = ""
					if l := f.viewer.MessageLines(); len(l) > lines {
						shown = l[len(l)-1].Text
					}
				}
				if !strings.HasPrefix(got, "asked/") {
					if len(seams.asked) != 0 || shown != "" {
						t.Fatalf("a %s point asked %+v and showed %q", got, seams.asked, shown)
					}
					return
				}
				if len(seams.asked) != 1 || !seams.asked[0].Admission || seams.asked[0].OnMap != (f.screen == ScreenMap) {
					t.Fatalf("requests %+v, want one admission on map %v", seams.asked, f.screen == ScreenMap)
				}
				if shown != reason {
					t.Fatalf("refusal line %q, want the admission's %q", shown, reason)
				}
			})
		}
	}
}
