package ui

import (
	"fmt"
	"image"
	"testing"
	"time"
)

func hoverFogFixture(t *testing.T, second bool) (*App, *Viewer, []byte) {
	t.Helper()
	a, v := hoverHitFixture(t, second)
	plane := make([]byte, 32*32)
	plane[6*32+8] = FogExplored
	v.SetFog(plane, 32, 32)
	return a, v, plane
}

func checkFogHoverReaders(t *testing.T, v *Viewer, visible bool) {
	t.Helper()
	wantID, wantCursor := uint32(1), "move"
	if visible {
		wantID, wantCursor = 7, "attack"
	}
	if got := v.fogGateEntity(2, 8, 6); got != visible {
		t.Errorf("enemy visible=%v, want %v", got, visible)
	}
	if p, ok := v.InspectionPanel(); !ok || p.Kind != InspectionUnit || p.ID != wantID {
		t.Errorf("panel kind=%v id=%d/%v, want unit %d", p.Kind, p.ID, ok, wantID)
	}
	if _, lit := v.hoverSpriteFactor(7); lit != visible {
		t.Errorf("enemy highlight=%v, want %v", lit, visible)
	}
	if _, lit := v.hoverSpriteFactor(1); lit {
		t.Error("selected fallback received hover shading")
	}
	if name, _ := v.missionHoverCursor(); name != wantCursor {
		t.Errorf("cursor=%q, want %q", name, wantCursor)
	}
	if _, _, ok := v.panelPresent(); !ok || v.panelKey.subject.ID != wantID {
		t.Errorf("drawn panel id=%d/%v, want unit %d", v.panelKey.subject.ID, ok, wantID)
	}
	if _, _, ok := v.missionCardPresent(); !ok || v.missionCardKey.subject.ID != wantID {
		t.Errorf("drawn card id=%d/%v, want unit %d", v.missionCardKey.subject.ID, ok, wantID)
	}
	v.refreshHoverLighting()
	if _, lit := v.hoverSpriteFactor(7); lit != visible {
		t.Errorf("draw refresh highlight=%v, want %v", lit, visible)
	}
}

func TestHoverRevealInputRefreshesReaders(t *testing.T) {
	for _, second := range []bool{false, true} {
		for _, initial := range []bool{false, true} {
			t.Run(fmt.Sprintf("second=%v/initial=%v", second, initial), func(t *testing.T) {
				a, v, _ := hoverFogFixture(t, second)
				v.SetFogReveal(initial)
				in := atFrame(280, 212)
				a.step(in, time.Unix(1, 0))
				checkFogHoverReaders(t, v, initial)
				in.Reveal = true
				a.step(in, time.Unix(1, 1))
				if v.FogRevealed() == initial {
					t.Fatal("reveal input did not toggle the override")
				}
				checkFogHoverReaders(t, v, !initial)
			})
		}
	}
}

func TestHoverRevealReleaseAttacksVisibleEnemy(t *testing.T) {
	for _, second := range []bool{false, true} {
		t.Run(fmt.Sprintf("second=%v", second), func(t *testing.T) {
			a, v, _ := hoverFogFixture(t, second)
			moves := 0
			var attacks [][2]uint32
			a.flow.order = func(uint32, int, int) { moves++ }
			a.flow.attack = func(entity, victim, spell uint32, x, y int, cell bool) {
				if spell != 0 || cell {
					t.Errorf("attack became spell=%d cell=%v at (%d,%d)", spell, cell, x, y)
				}
				attacks = append(attacks, [2]uint32{entity, victim})
			}
			press := atFrame(280, 212)
			press.PrimaryPressed, press.Viewer.PrimaryDown = true, true
			a.step(press, time.Unix(1, 0))
			release := atFrame(280, 212)
			release.PrimaryReleased, release.Reveal = true, true
			a.step(release, time.Unix(1, 1))
			if moves != 0 || len(attacks) != 1 || attacks[0] != [2]uint32{1, 7} {
				t.Errorf("moves=%d attacks=%v, want one attack 1 -> 7", moves, attacks)
			}
			checkFogHoverReaders(t, v, true)
		})
	}
}

func TestHoverRevealSettersRefreshReaders(t *testing.T) {
	for _, second := range []bool{false, true} {
		for _, tc := range []struct {
			name    string
			initial bool
			change  func(*Viewer)
		}{
			{"set on", false, func(v *Viewer) { v.SetFogReveal(true) }},
			{"set off", true, func(v *Viewer) { v.SetFogReveal(false) }},
			{"toggle on", false, func(v *Viewer) { v.ToggleFogReveal() }},
			{"toggle off", true, func(v *Viewer) { v.ToggleFogReveal() }},
			{"cheat on", false, func(v *Viewer) { v.SetCheatMapReveal(true) }},
			{"cheat off", true, func(v *Viewer) { v.SetCheatMapReveal(false) }},
		} {
			t.Run(fmt.Sprintf("second=%v/%s", second, tc.name), func(t *testing.T) {
				a, v, _ := hoverFogFixture(t, second)
				v.SetFogReveal(tc.initial)
				a.step(atFrame(280, 212), time.Unix(1, 0))
				checkFogHoverReaders(t, v, tc.initial)
				tc.change(v)
				checkFogHoverReaders(t, v, !tc.initial)
			})
		}
	}
}

func TestHoverFogAndSnapshotRefreshBetweenReaders(t *testing.T) {
	for _, second := range []bool{false, true} {
		t.Run(fmt.Sprintf("second=%v", second), func(t *testing.T) {
			a, v, plane := hoverFogFixture(t, second)
			a.step(atFrame(280, 212), time.Unix(1, 0))
			checkFogHoverReaders(t, v, false)
			plane[6*32+8] = FogVisible
			v.SetFog(plane, 32, 32)
			checkFogHoverReaders(t, v, true)
			plane[6*32+8] = FogExplored
			v.SetFog(plane, 32, 32)
			checkFogHoverReaders(t, v, false)
			v.SetCheatMapReveal(true)
			checkFogHoverReaders(t, v, true)
			v.SetCheatMapReveal(false)
			checkFogHoverReaders(t, v, true)
			v.RefreshCheatFog()
			checkFogHoverReaders(t, v, false)
			v.SetFogReveal(true)
			checkFogHoverReaders(t, v, true)
			original := append([]MapEntity(nil), v.entities...)
			moved := append([]MapEntity(nil), original...)
			moved[1].Cell = image.Pt(20, 20)
			v.SetEntities(moved)
			if p, ok := v.InspectionPanel(); !ok || p.ID != 1 {
				t.Errorf("moved snapshot panel id=%d/%v", p.ID, ok)
			}
			v.SetEntities(original)
			checkFogHoverReaders(t, v, true)
		})
	}
}

func TestHoverCameraJumpAfterResolveRefreshesReaders(t *testing.T) {
	for _, second := range []bool{false, true} {
		t.Run(fmt.Sprintf("second=%v", second), func(t *testing.T) {
			a, v, _ := hoverFogFixture(t, second)
			v.SetFogReveal(true)
			v.entities[0].Cell = image.Pt(24, 24)
			v.groups[0] = selection{1}
			in := atFrame(280, 212)
			a.step(in, time.Unix(1, 0))
			checkFogHoverReaders(t, v, true)
			before := *v.Camera()
			in.GroupKey, in.Viewer.Alt = true, true
			a.step(in, time.Unix(1, 1))
			if v.Camera().X == before.X && v.Camera().Y == before.Y {
				t.Fatal("group input did not move the camera")
			}
			if p, ok := v.InspectionPanel(); !ok || p.ID != 1 {
				t.Errorf("camera jump panel id=%d/%v, want selected unit", p.ID, ok)
			}
			if _, lit := v.hoverSpriteFactor(7); lit {
				t.Error("camera jump retained the enemy highlight")
			}
			if name, _ := v.missionHoverCursor(); name != "move" {
				t.Errorf("camera jump cursor=%q, want move", name)
			}
		})
	}
}

func TestHoverQuickLoadReplacesCachedObject(t *testing.T) {
	for _, second := range []bool{false, true} {
		t.Run(fmt.Sprintf("second=%v", second), func(t *testing.T) {
			a, live, _ := hoverFogFixture(t, second)
			live.SetFogReveal(true)
			a.step(atFrame(280, 212), time.Unix(1, 0))
			checkFogHoverReaders(t, live, true)
			_, restored, _ := hoverFogFixture(t, second)
			a.SetQuickSaveControls(QuickSaveControls{Load: func() (MapOpener, bool, error) {
				return func() (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
					return restored, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil
				}, false, nil
			}})
			in := atFrame(280, 212)
			in.QuickLoad = true
			a.step(in, time.Unix(1, 1))
			if a.flow.viewer != restored {
				t.Fatal("quickload did not replace the viewer")
			}
			if p, ok := restored.InspectionPanel(); !ok || p.ID != 1 {
				t.Errorf("loaded panel id=%d/%v, want selected unit", p.ID, ok)
			}
			if _, lit := restored.hoverSpriteFactor(7); lit {
				t.Error("loaded viewer inherited the enemy highlight")
			}
			in.QuickLoad = false
			a.step(in, time.Unix(1, 2))
			checkFogHoverReaders(t, restored, false)
		})
	}
}
