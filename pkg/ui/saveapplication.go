package ui

import (
	"fmt"
	"math"
	"slices"

	"againrom/pkg/render/camera"
	"againrom/pkg/render/terrain"
)

// SaveApplicationState is the current mission application state, detached from
// the viewer. ViewX/Y are the scroll origin in cells, before the terrain
// projection's canvas translation. Native saves keep fractional origins and
// zoom; the original SAV writer must separately admit their integer-cell form.
type SaveApplicationState struct {
	Selection                    []uint32
	ViewX, ViewY, Zoom           float64
	InventoryOpen, SpellBookOpen bool
	// MinimapOpen remains a historical wire field; false no longer hides the map.
	DollOpen, WornOpen, MinimapOpen bool
	ShowHealth, FlyingHP, TimeFlow  bool
	PressedSpell                    uint32
	PeriodUS                        int
	Unpaced                         bool
	PlayerPaused                    bool
}

func (v *Viewer) projectionMinY() float64 {
	if v.proj != nil {
		return float64(v.proj.MinV)
	}
	return 0
}

// SaveApplication reads the same selection and switches used by the frame and
// command producers. Returned slices never alias the viewer's selection.
func (v *Viewer) SaveApplication() SaveApplicationState {
	s := SaveApplicationState{
		Selection: v.SelectedUnits(), InventoryOpen: v.hudShown(hudPanelPack),
		SpellBookOpen: v.hudShown(hudPanelBook), DollOpen: v.hudShown(hudPanelDoll),
		WornOpen: v.hudShown(hudPanelWorn), MinimapOpen: true,
		ShowHealth: v.HealthBarsShown(), FlyingHP: v.numeralsOn(), TimeFlow: v.TimeFlow(),
		PressedSpell: v.selectedSpell, PeriodUS: v.anim.Period(), Unpaced: v.unpaced, PlayerPaused: v.playerPaused,
	}
	if v.cam != nil {
		s.ViewX = v.cam.X / camera.CellSize
		s.ViewY = (v.cam.Y + v.projectionMinY()) / camera.CellSize
		s.Zoom = v.cam.Zoom
	}
	return s
}

func ValidateSaveApplication(s SaveApplicationState) error {
	if len(s.Selection) > 1<<16 {
		return fmt.Errorf("saved application selection exceeds 65536 actors")
	}
	for i, id := range s.Selection {
		if i > 0 && id <= s.Selection[i-1] {
			return fmt.Errorf("saved application selection is not a unique ascending entity list")
		}
	}
	for _, x := range []float64{s.ViewX, s.ViewY, s.Zoom} {
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return fmt.Errorf("saved application camera is not finite")
		}
	}
	if s.ViewX < math.MinInt32 || s.ViewX > math.MaxInt32 || s.ViewY < math.MinInt32 || s.ViewY > math.MaxInt32 || s.Zoom < camera.ZoomMin || s.Zoom > camera.ZoomMax {
		return fmt.Errorf("saved application camera is outside its supported range")
	}
	if s.PressedSpell > math.MaxUint16 {
		return fmt.Errorf("saved application current spell exceeds uint16")
	}
	if s.PeriodUS != terrain.CadencePeriod(terrain.CadenceRung(s.PeriodUS)) {
		return fmt.Errorf("saved application cadence is outside the native ladder")
	}
	return nil
}

// RestoreSaveApplication validates every requested actor before changing any
// viewer state. The game tier calls it after restoring fog and entities, then
// pushes selection-dependent content and reapplies the current spell.
func (v *Viewer) RestoreSaveApplication(s SaveApplicationState) error {
	if err := ValidateSaveApplication(s); err != nil {
		return err
	}
	if v == nil || v.cam == nil {
		return fmt.Errorf("saved application needs a mission viewer")
	}
	for _, id := range s.Selection {
		found := 0
		for _, entity := range v.entities {
			if entity.ID == id && selectableEntity(entity) {
				found++
			}
		}
		if found != 1 {
			return fmt.Errorf("saved application actor %d has %d selectable viewer bindings", id, found)
		}
	}
	v.sel = slices.Clone(s.Selection)
	v.hudHidden[hudPanelPack], v.hudHidden[hudPanelBook] = !s.InventoryOpen, !s.SpellBookOpen
	v.hudHidden[hudPanelDoll], v.hudHidden[hudPanelWorn] = !s.DollOpen, !s.WornOpen
	v.healthBarsHidden, v.numeralsHidden = !s.ShowHealth, !s.FlyingHP
	v.SetTimeFlow(s.TimeFlow)
	v.SetCadenceMode(s.PeriodUS, s.Unpaced, false)
	v.setPlayerPaused(s.PlayerPaused)
	v.startArmed = false
	v.cam.Zoom = s.Zoom
	v.syncMapViewport()
	v.cam.X, v.cam.Y = s.ViewX*camera.CellSize, s.ViewY*camera.CellSize-v.projectionMinY()
	v.selectedSpell = s.PressedSpell
	v.spellArmed, v.armed, v.attackHeld = false, false, false
	v.aimed, v.itemCast = commandNone, nil
	if s.PressedSpell != 0 && v.spellAvailable(s.PressedSpell) && canArmAttack(v.sel, v.entities, v.localOwner) {
		v.armSpell()
		v.spellNeedsBook = s.SpellBookOpen
	}
	v.applicationRestored = true
	return nil
}

// RestoreCamera moves the mission camera and nothing else. A save that recorded
// only where the player was looking must not also reinstall a selection, the
// panel switches or an armed spell, which is the rest of what
// RestoreSaveApplication does. The three assignments below are that function's
// own camera lines, unchanged, and the range test is ValidateSaveApplication's.
func (v *Viewer) RestoreCamera(x, y, zoom float64) error {
	if v == nil || v.cam == nil {
		return fmt.Errorf("saved camera needs a mission viewer")
	}
	for _, value := range []float64{x, y, zoom} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return fmt.Errorf("saved camera is not finite")
		}
	}
	if x < math.MinInt32 || x > math.MaxInt32 || y < math.MinInt32 || y > math.MaxInt32 || zoom < camera.ZoomMin || zoom > camera.ZoomMax {
		return fmt.Errorf("saved camera is outside its supported range")
	}
	// DISARMING THE START VIEW IS WHAT MAKES THE MOVE STICK, and it is why
	// this is not three assignments a caller could inline. SetStartView arms
	// every mission open, including a LOAD, and the first Layout applies that
	// arming -- so a camera placed before Layout and not disarmed is replaced
	// by the map's own start position one frame later. RestoreSaveApplication
	// clears the same flag for the same reason. It is cleared after the range
	// test, so a refused camera leaves the opening view armed.
	v.startArmed = false
	v.cam.Zoom = zoom
	v.syncMapViewport()
	v.cam.X, v.cam.Y = x*camera.CellSize, y*camera.CellSize-v.projectionMinY()
	return nil
}

// A prepared LOAD carries its own cadence. Adopting that viewer must not write
// the process preference or replace its saved speed with the menu's default.
func (f *flow) adoptRestoredApplication(v *Viewer) bool {
	if v == nil || !v.applicationRestored {
		return false
	}
	v.applicationRestored = false
	f.rung, f.unpaced = terrain.CadenceRung(v.anim.Period()), v.unpaced
	f.stopped = v.playerPaused
	f.preferredRung, f.preferredRungSet = f.rung, true
	f.farRung = -1
	f.syncCadence(false)
	return true
}
