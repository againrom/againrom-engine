package ui

// Function keys consume their action frame after children and cutscenes.
func (a *App) stepGameFunctionKeys(in appInput) bool {
	if in.Unfocused || !in.SaveGame && !in.LoadGame && !in.QuickSave && !in.QuickLoad {
		return false
	}
	phase2 := false
	switch a.flow.screen {
	case ScreenMap:
		v := a.flow.viewer
		if v == nil || v.popupOpen() {
			return false
		}
		phase2 = v.readGameMenuContext().Campaign
	case ScreenTown:
		if _, open := townDialogue(a.flow.town); open {
			return false
		}
		phase2 = true
	default:
		return false
	}
	if (in.QuickSave || in.QuickLoad) && phase2 && (a.flow.screen != ScreenTown || townCanSave(a.flow.town)) {
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
	var action gameMenuAction
	switch {
	case in.SaveGame && phase2 && (a.flow.screen != ScreenTown || townCanSave(a.flow.town)):
		action = gameMenuSave
	case in.LoadGame && phase2:
		action = gameMenuLoad
	case in.LoadGame:
		action = gameMenuDiplomacy
	default:
		return false
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
