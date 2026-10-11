package ui

import (
	"strings"

	"againrom/pkg/render/camera"
	"againrom/pkg/render/terrain"
)

// MapWeaponSwap is the seam the weapon swap key leaves this package through:
// the selected units the local player owns, in selection order. The far side
// decides for each unit whether it has a previous weapon to take back.
type MapWeaponSwap func(entities []uint32)

// SetWeaponSwapSink hands the viewer the seam above. A viewer never given one
// swaps nothing.
func (v *Viewer) SetWeaponSwapSink(fn MapWeaponSwap) { v.weaponSwapSink = fn }

// swapWeapons is W on the map: every living selected unit the local player
// owns is named to the sink once (DIV-2948).
func (v *Viewer) swapWeapons() {
	if v.weaponSwapSink == nil {
		return
	}
	var ids []uint32
	for _, e := range presentSelected(v.sel, v.entities) {
		if e.Life != LifeDead && (v.localOwner == 0 || e.Owner == v.localOwner) {
			ids = append(ids, e.ID)
		}
	}
	if len(ids) != 0 {
		v.weaponSwapSink(ids)
	}
}

// defaultMapZoom is the zoom a mission opens at: the base viewport across the
// authored start columns, or native scale when no column count is set.
func defaultMapZoom() float64 {
	if cols := AuthoredStartColumns(); cols > 0 {
		return float64(MissionViewportSize().X) / float64(cols*camera.CellSize)
	}
	return 1
}

// resetZoom is Home on the map: the zoom returns to the opening zoom and the
// world point at the centre of the view stays there, to the camera's own
// clamp (DIV-2950).
func (v *Viewer) resetZoom() {
	cx, cy := v.cam.ScreenToWorld(float64(v.cam.ViewW)/2, float64(v.cam.ViewH)/2)
	v.cam.SetZoom(defaultMapZoom())
	v.cam.CenterOn(cx, cy)
}

// resetCadence is End on the map: the game speed returns to the shipped
// default rung through the same preference and far-side write the speed keys
// use. The unlimited loop, when selected, is left with its phase reset, as
// Ctrl and numpad minus leave it. A popup refuses the key, as it refuses the
// speed keys. It reports whether the key acted.
func (f *flow) resetCadence() bool {
	if f.popupOpen() {
		return false
	}
	reset := f.unpaced
	f.rung, f.unpaced = terrain.DefaultCadenceRung, false
	f.rememberCadenceRung(f.rung)
	f.syncCadence(reset)
	return true
}

// helpBody is the F1 panel's text: the install's help text, then a blank line
// and the engine's own key lines in the install's language and code page, so
// the keys this build adds are listed beside the original's (DIV-2951). An
// install without help text still opens no panel. The appended lines take the
// install text's own line break.
func (f *flow) helpBody(install string) string {
	if install == "" {
		return install
	}
	nl := "\n"
	if strings.Contains(install, "\r\n") {
		nl = "\r\n"
	}
	var b strings.Builder
	b.WriteString(strings.TrimRight(install, "\r\n"))
	b.WriteString(nl)
	for _, line := range []string{f.word("help.keys"), f.word("help.swap"), f.word("help.autocast"),
		f.word("help.zoom"), f.word("help.speed")} {
		b.WriteString(nl)
		b.WriteString(f.menuText(line))
	}
	return b.String()
}
