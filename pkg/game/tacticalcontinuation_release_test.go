package game

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseRetreatPendingSAVColdLoadAndNextAction(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("Retreat continuation")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	f.ConfigureSaveSeams(app, SaveStore{Dir: t.TempDir()}, OriginalStore{}, nil)
	party := MissionPartyAs(false, f.StartWeapon.Value(), f.Bodies, f.Table)
	if err := app.OpenMission(f.MissionOpenerWith(releaseRetreatRefusedMission, party)); err != nil {
		t.Fatal(err)
	}
	for k := 0; k < 16 && f.live.mission.open; k++ {
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	hero := f.live.mission.ids[0]
	if err := app.HeadlessSelectEntity(uint32(hero)); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("r"); err != nil {
		t.Fatal(err)
	}
	f.live.tick()
	w := f.live.world
	actions := w.Actions()
	traversal := w.ActorTraversal()
	slices.Reverse(traversal)
	actions.ActorTraversal = &traversal
	var hx, hy int32
	var owner uint32
	for _, e := range w.Entities() {
		if e.ID == hero {
			hx, hy, owner = e.X, e.Y, e.Owner
		}
	}
	var foe sim.EntityID
	found := false
	relations := w.Relations()
	for _, e := range w.Entities() {
		if e.ID != hero && e.Alive() && !e.OffMap && relations.Hostile(owner, e.Owner) {
			foe, found = e.ID, true
			break
		}
	}
	if !found {
		t.Fatal("no installed actor for loaded-cycle control")
	}
	for i := range actions.Actors {
		a := &actions.Actors[i]
		if a.Entity == hero {
			a.HasAttackTarget, a.AttackTarget, a.AttackPhase, a.AttackCountdown = true, foe, sim.AttackCharging, 1
			a.Retreat = &sim.RetreatContinuation{Known: true, Pending: true, X: hx, Y: hy, Progress: 1, Counter: 2, Complete: true}
		}
	}
	if err := w.RestoreActions(actions, nil); err != nil {
		t.Fatal(err)
	}
	dir, raw := castOrderF2Save(t, f, app, "Retreat pending")
	if err := app.HeadlessKey("escape"); err != nil || app.Screen() != ui.ScreenMap {
		t.Fatal("SAVE return to GAME", err, app.Screen())
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	current, err := readCurrentActions(&doc)
	if err != nil || current == nil {
		t.Fatal("current actions absent", err)
	}
	requireTacticalActorBytes(t, &doc, current, hero, hx, hy)
	cold, _ := castOrderSession(t, dir)
	if cold.live.world.Hash() != w.Hash() {
		t.Fatalf("installed cold LOAD changed state: %x != %x", cold.live.world.Hash(), w.Hash())
	}
	current.Actions.ActorTraversal = nil
	for i := range current.Actions.Actors {
		current.Actions.Actors[i].Retreat = nil
	}
	leaf, err := json.Marshal(current)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
		t.Fatal(err)
	}
	lost, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(raw, lost) {
		t.Fatal("loss control changed no installed SAV bytes")
	}
	lossDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(lossDir, "Retreat loss.sav"), lost, 0600); err != nil {
		t.Fatal(err)
	}
	loss, _ := castOrderSession(t, lossDir)
	if loss.live.world.Hash() == w.Hash() {
		t.Fatal("loss control changed no installed decoded state")
	}
	var boundary *FrontEnd
	for tick := range 3 {
		f.live.tick()
		cold.live.tick()
		loss.live.tick()
		if boundary != nil {
			boundary.live.tick()
			if boundary.live.world.Hash() != w.Hash() {
				t.Fatal("installed completion-clear LOAD changed next production action")
			}
		}
		if w.Hash() != cold.live.world.Hash() {
			t.Fatal("installed cold LOAD changed next production action")
		}
		if tick == 0 {
			var continuing, lostActor sim.Entity
			for _, e := range w.Entities() {
				if e.ID == hero {
					continuing = e
				}
			}
			for _, e := range loss.live.world.Entities() {
				if e.ID == hero {
					lostActor = e
				}
			}
			if continuing.HasAttackTarget == lostActor.HasAttackTarget && continuing.AttackPhase == lostActor.AttackPhase && continuing.HasTarget == lostActor.HasTarget {
				t.Fatal("installed action loss control changed only retained state")
			}
			if !continuing.HasAttackTarget || continuing.AttackTarget != foe || continuing.AttackPhase != sim.AttackCharging || continuing.AttackCountdown != 1 || continuing.Retreat.Progress != 0 || !continuing.Retreat.Pending || continuing.HasTarget {
				t.Fatal("installed completion-clear dispatched or lost/advanced its physical carrier", continuing)
			}
			boundaryDir, boundaryRaw := castOrderF2Save(t, f, app, "Retreat completion clear")
			if err := app.HeadlessKey("escape"); err != nil || app.Screen() != ui.ScreenMap {
				t.Fatal("completion-clear SAVE return to GAME", err, app.Screen())
			}
			boundaryDoc, err := sav.DecodeDocumentData(boundaryRaw)
			if err != nil {
				t.Fatal(err)
			}
			boundaryActions, err := readCurrentActions(&boundaryDoc)
			if err != nil || boundaryActions == nil {
				t.Fatal("completion-clear actions absent", err)
			}
			requireCompletionTacticalActorBytes(t, &boundaryDoc, boundaryActions, hero, sim.AttackCharging)
			boundary, _ = castOrderSession(t, boundaryDir)
			if boundary.live.world.Hash() != w.Hash() {
				t.Fatal("installed completion-clear cold LOAD changed state")
			}
		}
	}
	if loss.live.world.Hash() == w.Hash() {
		t.Fatal("loss control changed no installed next action")
	}
}
