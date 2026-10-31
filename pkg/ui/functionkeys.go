package ui

// Function keys consume their action frame after children and cutscenes.
func (a *App) stepGameFunctionKeys(in appInput) bool {
	if in.Unfocused || !in.SaveGame && !in.LoadGame && !in.QuickSave && !in.QuickLoad {
		return false
	}
	if (in.QuickSave || in.QuickLoad) && a.flow.savePoint(saveQuickKey).open() {
		a.clearShopDrag()
		a.townSurfacePress.Clear()
		if v := a.flow.viewer; v != nil {
			v.cancelPointerGesture()
		}
		if (in.PrimaryPressed || in.Viewer.PrimaryDown) && !in.PrimaryReleased {
			a.suppressPrimaryRelease = true
		}
		if (in.SecondaryPressed || in.Viewer.SecondaryDown) && !in.SecondaryReleased {
			a.suppressSecondaryRelease = true
		}
		a.stepQuickSave(in.QuickSave)
		return true
	}
	// The Load key opens Load at an open point and Diplomacy on a map that is
	// not a campaign. Pressed with Load, a Save the town refuses gives way to
	// Load without a reason.
	load, save := saveGate{row: pointNoScreen}, saveGate{row: pointNoScreen}
	if in.LoadGame {
		load = a.flow.savePoint(saveLoadKey)
	}
	if in.SaveGame {
		kind := saveKey
		if in.LoadGame {
			kind = saveKeyWithLoad
		}
		save = a.flow.savePoint(kind)
	}
	var action gameMenuAction
	switch {
	case save.open():
		action = gameMenuSave
	case load.open():
		action = gameMenuLoad
	case load.row == pointNotCampaign:
		action = gameMenuDiplomacy
	default:
		return save.shown
	}
	a.clearShopDrag()
	a.townSurfacePress.Clear()
	a.flow.viewer.cancelGold()
	// Navigation also owns a gesture that began on an earlier frame. The
	// physical level is sampled for every screen, including town; looking
	// only at this frame's press edge lets its release activate the new list.
	if (in.PrimaryPressed || in.Viewer.PrimaryDown) && !in.PrimaryReleased {
		a.suppressPrimaryRelease = true
	}
	a.flow.openGameMenu(a.flow.screen)
	a.beforeGameMenuAction(action)
	a.flow.applyGameMenuAction(action)
	a.syncViewerLayout()
	return true
}
