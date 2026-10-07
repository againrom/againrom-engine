package ui

import (
	"image"
	"testing"
)

func nonGamePoints(t *testing.T, v *Viewer) map[string]image.Point {
	t.Helper()
	pts := map[string]image.Point{}
	if bar, ok := v.packBarArea(); ok {
		pts["pack bar"] = bar.Min.Add(bar.Max).Div(2)
	}
	if bar, _, ok := v.spellbookBar(); ok {
		pts["book strip"] = bar.Min.Add(bar.Max).Div(2)
	}
	if bar, ok := v.commandPanelBar(); ok {
		pts["command panel"] = bar.Min.Add(bar.Max).Div(2)
	}
	if g, ok := v.minimapGeometry(); ok {
		pts["minimap"] = g.Box.Min.Add(g.Box.Max).Div(2)
	}
	pts["right column"] = image.Pt(v.cam.ViewW+MissionPanelW/2, 40)
	for _, name := range []string{"pack bar", "book strip", "right column"} {
		if _, ok := pts[name]; !ok {
			t.Fatalf("setup: no %s position", name)
		}
	}
	return pts
}

func TestOrderModeAndOrdinaryCursorsShowOnlyOverTheGameArea(t *testing.T) {
	modes := []struct {
		name string
		arm  func(t *testing.T, a *App, v *Viewer)
		want string
	}{
		{"ordinary", func(*testing.T, *App, *Viewer) {}, "move"},
		{"attack", func(t *testing.T, a *App, _ *Viewer) { pressPanelKey(t, a, "a") }, "attack"},
		{"move", func(_ *testing.T, _ *App, v *Viewer) { v.pressCommandPanelCell(commandCellMove, true) }, "move"},
		{"defend", func(t *testing.T, a *App, _ *Viewer) { pressPanelKey(t, a, "d") }, "defend"},
		{"patrol", func(t *testing.T, a *App, _ *Viewer) { pressPanelKey(t, a, "p") }, "patrol"},
		{"swarm", func(_ *testing.T, _ *App, v *Viewer) { v.pressCommandPanelCell(commandCellSwarm, true) }, "swarm"},
		{"cast", func(t *testing.T, a *App, v *Viewer) {
			chooseSpell(t, v, castArea)
		}, "cast"},
	}
	for _, m := range modes {
		t.Run(m.name, func(t *testing.T) {
			a, v := castCursorApp(t)
			pressPanelKey(t, a, "space")
			m.arm(t, a, v)
			if got := cursorAtCell(t, a, v, castCells.ground[0], castCells.ground[1]); got != m.want {
				t.Fatalf("%s over ground: cursor %q, want %q", m.name, got, m.want)
			}
			for where, p := range nonGamePoints(t, v) {
				hoverFrame(t, a, v, p)
				if got := shownCursor(v); got != "default" {
					t.Fatalf("%s over the %s: cursor %q, want the arrow", m.name, where, got)
				}
				if _, _, drawn := v.attackPointerPresent(); drawn {
					t.Fatalf("%s over the %s: the attack picture is drawn", m.name, where)
				}
			}
			if got := cursorAtCell(t, a, v, castCells.ground[0], castCells.ground[1]); got != m.want {
				t.Fatalf("%s back over ground: cursor %q, want %q", m.name, got, m.want)
			}
		})
	}
}
