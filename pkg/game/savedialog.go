package game

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"againrom/pkg/formats/textinput"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// ConfigureSaveSeams installs the player dialog and keeps LOAD in the directory
// of the last successful save. Interactive and automatic saves share the
// current SAV producer.
func (f *FrontEnd) ConfigureSaveSeams(app *ui.App, store SaveStore, original OriginalStore, now func() time.Time, observers ...func(string, Snapshot)) {
	// The hall stays in the launch profile even if SAVE later browses elsewhere.
	if store.Dir != "" {
		f.hallStore.Dir = filepath.Dir(store.Dir)
		f.hallStore.profile = store.profile.rootAccess()
	}
	if f.Archives != nil {
		f.hallStore.OriginalDir = f.Archives.Root
	}
	// EVERY SAV LABEL THIS WIRING READS OR WRITES GOES THROUGH ONE INSTALL'S
	// OWN CODE PAGE: the caller supplies bare stores, and this is the one seam
	// that knows which install is actually running.
	store.Selector = f.textSelector()
	original.Selector = store.Selector
	current := store
	// Automatic saves are exported and written off the frame thread, in order.
	// Every later save, load, delete and exit waits for the queue first, so no
	// automatic save finishes after a save the player made later.
	queue := newAutosaveQueue()
	app.SetBackgroundFlush(queue.wait)
	app.SetMapEntryObserver(func(viewer *ui.Viewer) {
		f.autosaveMissionStart(viewer, current, original, queue, observers...)
	})
	fences := []string{original.Dir}
	if f.Archives != nil {
		fences = append(fences, f.Archives.Root)
	}
	f.configureScreenshot(app, store, fences)
	f.configureTimedAutosave(app, &current, fences, now, queue, observers...)
	f.configureQuickSave(app, &current, fences, queue, observers...)
	game := f.savedGame()
	app.SetSaveFiles(func() bool {
		return anyGameSave(current.Dir) || filesOf(game).ownSaves && f.anyLoadableSaveRow(SaveStore{Dir: current.Dir}, OriginalStore{})
	})
	app.SetSaveAny(func() bool { return f.anyLoadableSaveRow(current, original) })
	app.SetSaveDelete(func(token string) bool {
		_, err := loadDeletePath(current.Dir, token, fences, current.profile)
		return err == nil
	}, func(token string) (func() error, error) {
		queue.wait()
		return prepareLoadDelete(current.Dir, token, fences, current.profile)
	})
	app.SetSaveSeams(func(onMap bool) (string, error) {
		queue.wait()
		save, _, _ := f.SaveSeams(current, original, now, observers...)
		return save(onMap)
	}, func() []ui.SaveEntry {
		// A local row keeps its stored label, undrawable runes included, so
		// this window and the SAVE dialog's own browser (SaveDialogSeams.List,
		// same store, never altered) show the same string for the same file.
		return f.saveRows(current, original, false)
	}, func(name string) (ui.MapOpener, bool, error) {
		queue.wait()
		_, _, load := f.SaveSeams(current, original, now)
		return load(name)
	})
	seams := f.SaveDialogSeams(store, original, observers...)
	prepare := seams.Prepare
	prepareDelete := seams.PrepareDelete
	seams.PrepareDelete = func(dir, name string) (func() error, error) {
		queue.wait()
		return prepareDelete(dir, name)
	}
	seams.Prepare = func(request ui.SaveRequest) (ui.PreparedSave, error) {
		queue.wait()
		prepared, err := prepare(request)
		if err != nil {
			return prepared, err
		}
		commit := prepared.Commit
		prepared.Commit = func(overwrite bool) ([]string, error) {
			queue.wait()
			paths, err := commit(overwrite)
			if err == nil {
				current.Dir = request.Directory
			}
			return paths, err
		}
		return prepared, nil
	}
	app.SetSaveDialogSeams(seams)
}

func (f *FrontEnd) SaveDialogSeams(store SaveStore, original OriginalStore, observers ...func(string, Snapshot)) ui.SaveDialogSeams {
	fences := []string{original.Dir}
	if f.Archives != nil {
		fences = append(fences, f.Archives.Root)
	}
	return ui.SaveDialogSeams{Directory: store.Dir,
		List: func(dir string) (ui.SaveDirectory, error) {
			if game := f.savedGame(); filesOf(game).ownSaves {
				return listOwnSaveDirectory(dir, fences, store.Selector, game, store.profile)
			}
			return listSaveDirectory(dir, fences, store.Selector, store.profile)
		},
		CanDelete: func(dir, name string) bool {
			_, err := loadDeletePath(dir, saveDialogDeleteToken(name), fences, store.profile)
			return err == nil
		},
		PrepareDelete: func(dir, name string) (func() error, error) {
			return prepareLoadDelete(dir, saveDialogDeleteToken(name), fences, store.profile)
		},
		Prepare: func(request ui.SaveRequest) (ui.PreparedSave, error) {
			if request.Format != ui.SaveSAV {
				return ui.PreparedSave{}, fmt.Errorf("save format must be SAV")
			}
			name, err := namedSaveBase(request.Name)
			if err != nil {
				return ui.PreparedSave{}, err
			}
			if _, err := namedSaveDirectory(request.Directory, fences, store.profile); err != nil {
				return ui.PreparedSave{}, err
			}
			label := request.Label
			if label == "" {
				label = string(name)
			}
			// The file is built from the pruned lists, but the live World is
			// pruned only when the file is written (SAV-1251): a cancelled
			// overwrite must leave the World and its hash as they were.
			var raw []byte
			var notice string
			err = f.captureSave(captureTentative, request.OnMap, func(s Snapshot, _ string) error {
				notifySaveCapture(observers, "f2", s)
				var err error
				raw, notice, err = f.playerMissionSave(s, label)
				return err
			})
			if err != nil {
				return ui.PreparedSave{}, err
			}
			payloads := []namedSavePayload{{".sav", raw}}
			prepared, err := prepareNamedSave(request.Directory, name, payloads, fences, nil, store.profile)
			if err != nil {
				return prepared, err
			}
			prepared.Notice = notice
			if commit := prepared.Commit; commit != nil {
				prepared.Commit = func(overwrite bool) ([]string, error) {
					paths, err := commit(overwrite)
					if err == nil {
						_ = f.captureSave(captureCommitted, request.OnMap, nil)
					}
					return paths, err
				}
			}
			return prepared, nil
		}}
}

func notifySaveCapture(observers []func(string, Snapshot), route string, captured Snapshot) {
	for _, observer := range observers {
		if observer != nil {
			observer(route, captured)
		}
	}
}

func saveDialogDeleteToken(name string) string {
	if IsOriginal(name) {
		return localOriginalSaveToken(name)
	}
	return name
}

// citySnapshotFromMission performs only the survivor boundary on a detached World.
// It does not complete a mission, consume its objective, pay its reward,
// advance campaign selection, or touch the running driver.
func (f *FrontEnd) citySnapshotFromMission(captured Snapshot) (*FrontEnd, Snapshot, *sim.World, error) {
	n := *f
	var err error
	if n.Town, err = restoreTown(f.Campaign.Value(), captured); err != nil {
		return nil, Snapshot{}, nil, err
	}
	n.townUI = nil
	// The emitted header is a city header. Apply the same location admission
	// as cold LOAD before normalizing any survivor or preparing output. A
	// pre-town campaign remains unfinished; saving must not advance it.
	location := &campaignProgress{campaignRecords: campaignRecords{main: campaignProgressRecord{mission: n.Town.Chapter()}}}
	if err := validateCampaignLocation(n.Campaign.Value(), location, 0, "city SAV"); err != nil {
		return nil, Snapshot{}, nil, originalCityUnsupportedf("%v", err)
	}
	if len(captured.WorldSelectedOnce) != 0 {
		screen := n.TownScreen().(*townScreen)
		screen.worldSelectedOnce, _ = worldSelectedOnceFromSnapshot(captured.WorldSelectedOnce)
		screen.worldSelectedOrder = append([]int(nil), captured.WorldSelectedOnce...)
	}
	n.originalCity, err = originalCityFromSnapshot(captured)
	if err != nil {
		return nil, Snapshot{}, nil, err
	}
	n.Carried = mapload.CloneParty(captured.Party)
	if captured.Mission == 0 {
		return &n, captured, nil, nil
	}
	if f.live == nil || f.live.world == nil || f.live.mission == nil || f.live.mission.state == nil || captured.Mission != f.liveMission {
		return nil, Snapshot{}, nil, fmt.Errorf("mission save lacks its current party binding")
	}
	world := *f.live.world
	mercenaries := liveMercenaries(f.live.mission.party, &world, f.live.mission.ids)
	if err := world.NormalizeMissionSurvivors(sim.SelfSlot); err != nil {
		return nil, Snapshot{}, nil, err
	}
	mission := f.live.mission
	party := mapload.CloneParty(mission.party)
	ids := append([]sim.EntityID(nil), mission.ids...)
	packs := captureCityItemGraphs(party, &world, ids)
	worn := captureCityEquipmentGraphs(party, &world, ids)
	carryParty, carryIDs := withoutFallenBodies(party, ids, &world)
	n.Carried, n.Town.cityObjects, err = currentMissionCityParty(&world, mission.state.savedDocument, carryParty, carryIDs, mission.state.Start.Roster, n.Table)
	if err != nil {
		return nil, Snapshot{}, nil, err
	}
	n.Town.mercenaryBoundary(mercenaries)
	n.Town.gold = int(world.Purse(sim.SelfSlot))
	n.Town.playerColor = uint8(missionPlayerColor(mission.state.Map, sim.SelfSlot))
	n.Town.open = true
	// An active mission is restartable in this detached city even if a tool
	// originally opened it before its ordinary announcement was presented.
	n.Town.announceMission(captured.Mission)
	if progress := n.Town.progress; progress != nil {
		progress.firstMapPoint = true
		if progress.main.mission == captured.Mission {
			progress.main.announced = true
		}
		for i := range progress.children {
			if progress.children[i].mission == captured.Mission {
				progress.children[i].announced = true
			}
		}
	}
	if len(n.Carried) == 0 {
		return nil, Snapshot{}, nil, originalCityUnsupportedf("city save needs a surviving party")
	}
	live := *f.live
	live.world = &world
	n.live = &live
	n.captureMissionReturn(captured.Mission, party, &world, ids, true, packs, worn)
	// CITY LOAD ordinarily materializes a pending companion for this chapter.
	// Do that same arrival work on the copy now, so its grant latch and actual
	// Human record agree on the first save and do not duplicate on cold LOAD.
	n.addChapterCompanions(n.Town.Chapter())
	var city Snapshot
	fame := fameFromSnapshot(captured)
	city.Fame = &fame
	city.Difficulty, city.QuickSpells = captured.Difficulty, captured.QuickSpells
	city.Party = mapload.CloneParty(n.Carried)
	city.Offered = n.Town.Chapter()
	n.Offered = city.Offered
	snapshotTown(n.Town, &city)
	if city.CampaignState && fame.Known {
		city.Campaign.MissionTime = fame.Time
		city.Campaign.ScoreEvents, city.Campaign.ScoreEventsKnown = fame.Events, true
	}
	city.WorldSelectedOnce = append([]int(nil), captured.WorldSelectedOnce...)
	city.OriginalCity = n.originalCity.snapshot()
	return &n, city, &world, nil
}

// encodeSaveLabel uses the same byte alphabet as installed display text.
// Unrepresentable characters reject the label before output is prepared.
func encodeSaveLabel(label string, selector int) (string, error) {
	encoded, err := textinput.EncodeText(label, selector, textinput.Refuse)
	var refused textinput.RefusedError
	if errors.As(err, &refused) {
		// %U, not %q: the rune this reports is by definition one the
		// alphabet the message itself is drawn in cannot show, so a
		// player-legible message needs its ASCII code point instead.
		return "", fmt.Errorf("SAV label cannot represent %U; choose a different name", refused.Rune)
	}
	return encoded, err
}

// decodeSaveLabel is encodeSaveLabel's inverse: text in the install's
// alphabet as UTF-8. It answers false when a byte has no rune or the UTF-8
// does not encode back to s.
func decodeSaveLabel(s string, selector int) (string, bool) {
	runes := make([]rune, 0, len(s))
	for i := 0; i < len(s); i++ {
		r, ok := textinput.DecodeByte(s[i], selector)
		if !ok {
			return "", false
		}
		runes = append(runes, r)
	}
	back, err := encodeSaveLabel(string(runes), selector)
	return string(runes), err == nil && back == s
}

// playerCitySave retains the existing dialog seam and shares the current
// save-point and label policy with playerMissionSave.
func (f *FrontEnd) playerCitySave(captured Snapshot, label string) ([]byte, string, error) {
	return f.playerMissionSave(captured, label)
}
