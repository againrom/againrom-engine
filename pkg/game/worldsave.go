package game

import (
	"encoding/binary"
	"fmt"
	"slices"
	"strings"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// worldSaveUnsupportedError reports a missing representation or invalid binding.
type worldSaveUnsupportedError struct{ reason string }

func (e *worldSaveUnsupportedError) Error() string {
	return "current world SAV unavailable: " + e.reason
}
func worldSaveUnsupportedf(format string, args ...any) error {
	return &worldSaveUnsupportedError{fmt.Sprintf(format, args...)}
}

// All save points use ExportCurrentSave.
func (f *FrontEnd) ExportCurrentWorldSave(s Snapshot, label string) ([]byte, error) {
	return f.ExportCurrentSave(s, label)
}

// fresh selects the scoped native block-plane projection and initial repairs.
func currentWorldDocument(s Snapshot, fresh bool, tables ...*mapload.Table) (sav.DocumentData, error) {
	var table *mapload.Table
	if len(tables) != 0 {
		table = tables[0]
	}
	if err := validatePendingGameOptions(s.Residue.PendingGameOptions); err != nil {
		return sav.DocumentData{}, err
	}
	fail := func(reason string) (sav.DocumentData, error) {
		return sav.DocumentData{}, worldSaveUnsupportedf("%s", reason)
	}
	if s.Mission == 0 || len(s.World) == 0 {
		return fail("one mission World is required")
	}
	state, err := cloneSavedDocument(s.SavedDocument)
	if err != nil {
		return sav.DocumentData{}, err
	}
	if state == nil {
		return fail("never-imported worlds have no source-backed constructors")
	}
	if state.Document == nil {
		return fail(state.Unavailable)
	}
	difficulty, err := campaignDifficulty(int64(s.Difficulty))
	if err != nil {
		return sav.DocumentData{}, err
	}
	state.Document.Head.Difficulty = uint32(difficulty)
	var world sim.World
	if err = world.UnmarshalBinary(s.World); err != nil {
		return sav.DocumentData{}, err
	}
	actions, err := readCurrentActions(state.Document)
	if err != nil {
		return sav.DocumentData{}, err
	}
	if actions != nil {
		if err := retireCurrentAreaDocument(&Mission{World: &world, savedDocument: state}, actions.NativeAreas, actions.NativeDeliveries); err != nil {
			return sav.DocumentData{}, err
		}
	}
	policy := world.CurrentPolicy()
	policy.Ghost = s.ghost
	if len(tables) != 0 {
		policy.KeepSpellDeviations(mapload.SpellRules(tables[0]))
	}
	if err := armCarriedSpellGraph(state, &world); err != nil {
		return sav.DocumentData{}, err
	}
	if err := rootDeadActors(state, &world); err != nil {
		return sav.DocumentData{}, err
	}
	// Recompute supported fields from World even when a caller supplies an old
	// retained Document. Bindings, not equal field values, select each object.
	state, err = snapshotSavedDocument(&Mission{World: &world, savedDocument: state, Start: mapload.Start{ConstructionTable: table}}, fresh)
	if err != nil {
		return sav.DocumentData{}, fmt.Errorf("capture current document: %w", err)
	}
	if err := projectCurrentPartyActorFields(state, &world, s, table); err != nil {
		return sav.DocumentData{}, err
	}
	if err := retireCurrentActors(state, &world); err != nil {
		return sav.DocumentData{}, fmt.Errorf("retire current actors: %w", err)
	}
	if state.GroupBindings == nil {
		return fail("current Group/Player bindings are absent")
	}
	if state.GroupBindings.Unavailable != "" {
		return fail(state.GroupBindings.Unavailable)
	}
	if state.PlayerPurses == nil {
		return fail("current purse bindings are absent")
	}
	if err := projectDeadActorContainers(state, &world); err != nil {
		return sav.DocumentData{}, err
	}
	if err := projectItems(state, &world, tables...); err != nil {
		return sav.DocumentData{}, fmt.Errorf("project current holdings: %w", err)
	}
	if err := admitEffects(state, &world); err != nil {
		return sav.DocumentData{}, err
	}
	if err := sav.SetNativeRandomState(&state.Document.State, world.RandomState()); err != nil {
		return fail(err.Error())
	}
	if err := projectActionClocks(state, &world); err != nil {
		return sav.DocumentData{}, err
	}
	if err := projectMotion(state, &world); err != nil {
		return sav.DocumentData{}, err
	}
	if err := projectSavedActorActions(state, &world); err != nil {
		return sav.DocumentData{}, err
	}
	if err := projectSavedDiaries(state, &world); err != nil {
		return sav.DocumentData{}, err
	}
	if err := projectDeadActors(state, &world); err != nil {
		return sav.DocumentData{}, err
	}
	if err := projectRemovedNativeBasis(state, &world); err != nil {
		return sav.DocumentData{}, err
	}
	terminalBindings, err := currentTerminalActorBindings(state, &world)
	if err != nil {
		return sav.DocumentData{}, err
	}
	removed := map[uint32]bool{}
	for _, object := range terminalBindings {
		key, _ := savedStructureValue(&state.Document.Objects[object-1], "Identity")
		removed[key] = true
	}
	if err := projectCurrentCastWire(state, &world); err != nil {
		return sav.DocumentData{}, err
	}
	state.Document.World.Session.Results = world.ScriptRegisters()
	for i := range state.Document.Objects {
		r := &state.Document.Objects[i]
		if r.Class == "Player" {
			participant, _ := savedStructureValue(r, "Participant")
			if participant == 0 {
				savedObjectSetValue(r, "Outcome", uint32(world.Outcome()))
			}
		}
	}
	// Group/root reindexing also moves actor bindings used by selection.
	s.SavedDocument = state
	if err := projectApplicationState(state.Document, s, &world); err != nil {
		return sav.DocumentData{}, err
	}
	doc := *state.Document
	if err := projectCurrentActions(&doc, state, &world, s, table, policy); err != nil {
		return sav.DocumentData{}, err
	}
	if err := projectCurrentCreatureSpells(&doc, state, &world); err != nil {
		return sav.DocumentData{}, err
	}
	if s.CampaignState {
		doc, err = sav.ProjectDocumentCampaign(doc, s.Campaign)
		if err != nil {
			return sav.DocumentData{}, err
		}
	}
	if err := projectNativeAreas(&doc, &world); err != nil {
		return sav.DocumentData{}, err
	}
	if err := projectNativeSpellDeliveries(&doc, state, &world); err != nil {
		return sav.DocumentData{}, err
	}
	if err := projectWorldEffectOrder(&doc, &world); err != nil {
		return sav.DocumentData{}, err
	}
	if err := graftUnknownObjects(&doc, state, s.loadedDocument, &world); err != nil {
		return sav.DocumentData{}, err
	}
	projectCurrentTerrain(&doc, &world, fresh)
	if err := projectTerminalActorRegistry(&doc, removed); err != nil {
		return sav.DocumentData{}, err
	}
	// Bind the next checkpoint to this exact projected document.
	doc, permutation, err := sav.ReindexDocumentData(doc)
	if err != nil {
		return sav.DocumentData{}, err
	}
	state.Document = &doc
	if err := remapSavedSackDocument(state, permutation); err != nil {
		return sav.DocumentData{}, err
	}
	return doc, nil
}

func admitEffects(state *SnapshotSAVDocument, world *sim.World) error {
	if state.ActorEffects != nil && state.ActorEffects.Unavailable != "" {
		return worldSaveUnsupportedf("%s", state.ActorEffects.Unavailable)
	}
	if state.WorldEffects != nil {
		// These two accepted native policies do not omit an active object or
		// field. Other notices are specific missing graph/callback producers.
		for _, notice := range strings.Split(state.WorldEffects.Unavailable, "; ") {
			if notice == "" || notice == unboundProjectileConsumers || notice == "native removed-target policy retains the last current aim" || notice == "original target lifetime remains unproved" || notice == "projectile direction-sector and target-altitude arithmetic remain unproved" || notice == "retained direction fields are unchanged" {
				continue
			}
			if notice == "AreaEffect payload application is unbound: EDD48 field bridge or unsupported ordinary effect" && currentNativeAreasReplaceRetiredGraph(state, world) {
				// The ordinary AreaEffect node that raised this notice has
				// already been retired into a native current area.  The current
				// area producer writes its payload from the live row below, so
				// the stale source notice no longer describes this save point.
				continue
			}
			return worldSaveUnsupportedf("%s", notice)
		}
	}
	return nil
}

func currentNativeAreasReplaceRetiredGraph(state *SnapshotSAVDocument, world *sim.World) bool {
	if state == nil || state.WorldEffects == nil || len(state.WorldEffects.SpellNodes) != 0 || world == nil {
		return false
	}
	areas, err := world.NativeAreaSaveStates()
	return err == nil && len(areas) != 0
}

// Source archive indices bind the import once. Current dead records retain a
// unique source key after their entity has been removed; never join by
// MapUnitID, value equality, or the local object number.
func currentDeadSourceRoot(state *SnapshotSAVDocument, d sim.OriginalDeadRecord) (*sav.DocumentRecordData, error) {
	var index uint16
	for _, root := range state.Document.DeadActors {
		if root == 0 {
			continue
		}
		r := &state.Document.Objects[root-1]
		key, _ := savedStructureValue(r, "Identity")
		if key == d.Source.Identity {
			if index != 0 && index != root {
				return nil, worldSaveUnsupportedf("dead actor %d has ambiguous source identity", d.ID)
			}
			index = root
		}
	}
	if index == 0 {
		return nil, worldSaveUnsupportedf("dead actor %d has no exact source root", d.ID)
	}
	return &state.Document.Objects[index-1], nil
}

func projectDeadContainer(r *sav.DocumentRecordData, d sim.OriginalDeadRecord) {
	if d.Source.ContainerPresent {
		savedObjectSetValue(r, "HasInventory", 1)
		savedObjectSetValue(r, "Inventory1C", d.Source.ContainerTail[0])
		savedObjectSetValue(r, "Inventory20", d.Source.ContainerTail[1])
	} else {
		savedObjectSetValue(r, "HasInventory", 0)
		r.Values = slices.DeleteFunc(r.Values, func(v sav.DocumentValueData) bool { return v.Name == "Inventory1C" || v.Name == "Inventory20" })
		r.Counts = slices.DeleteFunc(r.Counts, func(v sav.DocumentCountData) bool { return v.Name == "Inventory" })
		r.RefSlots = slices.DeleteFunc(r.RefSlots, func(v sav.DocumentRefsData) bool { return v.Name == "Inventory" })
	}
}

func projectDeadActorContainers(state *SnapshotSAVDocument, world *sim.World) error {
	for _, d := range world.OriginalDeadActors() {
		r, err := currentDeadSourceRoot(state, d)
		if err != nil {
			return err
		}
		projectDeadContainer(r, d)
	}
	return nil
}

func projectDeadActors(state *SnapshotSAVDocument, world *sim.World) error {
	for _, d := range world.OriginalDeadActors() {
		r, err := currentDeadSourceRoot(state, d)
		if err != nil {
			return err
		}
		c := d.Current
		for _, v := range []sav.DocumentValueData{
			{Name: "Stage", Value: uint32(c.Stage)},
			{Name: "Health", Value: uint32(uint16(c.HP))},
			{Name: "RuntimeID", Value: c.RuntimeID},
			{Name: "U6C", Value: uint32(uint8(c.Timer))},
			{Name: "U40", Value: d.Source.References[4]},
		} {
			savedObjectSetValue(r, v.Name, v.Value)
		}
		projectDeadContainer(r, d)
		p, err := savedMotionRaw(r, "Block12", 12)
		if err != nil {
			return err
		}
		p[0], p[1], p[2], p[3], p[4], p[5] = byte(c.Cell), byte(c.Cell>>8), byte(c.Cell), byte(c.Cell>>8), c.FineX, c.FineY
		binary.LittleEndian.PutUint32(p[8:], d.Source.TerrainKey)
	}
	terminalBindings, err := currentTerminalActorBindings(state, world)
	if err != nil {
		return err
	}
	for _, terminal := range world.CurrentTerminalActors() {
		object, bound := terminalBindings[terminal.ID]
		if !bound {
			continue // departed before any SAV object bound it
		}
		if object == 0 || int(object) > len(state.Document.Objects) {
			return worldSaveUnsupportedf("current terminal actor %d has no exact ordinary SAV object", terminal.ID)
		}
		if terminal.HP < -32768 || terminal.HP > 32767 {
			return worldSaveUnsupportedf("current terminal actor %d health exceeds SAV width", terminal.ID)
		}
		r := &state.Document.Objects[object-1]
		savedObjectSetValue(r, "Stage", uint32(terminal.Stage))
		savedObjectSetValue(r, "Health", uint32(uint16(int16(terminal.HP))))
		p, err := savedMotionRaw(r, "Block12", 12)
		if err != nil {
			return err
		}
		p[0], p[1], p[2], p[3] = byte(terminal.Cell), byte(terminal.Cell>>8), byte(terminal.Cell), byte(terminal.Cell>>8)
	}
	bound := map[sim.EntityID]bool{}
	dead := map[sim.EntityID]bool{}
	for _, d := range world.OriginalDeadActors() {
		dead[d.ID] = true
	}
	for _, terminal := range world.CurrentTerminalActors() {
		dead[terminal.ID] = true
	}
	for _, a := range state.Actors {
		bound[a.EntityID] = true
		if a.Retired && !dead[a.EntityID] {
			return worldSaveUnsupportedf("retired actor %d has no current terminal tuple", a.EntityID)
		}
	}
	for _, d := range world.OriginalDeadActors() {
		bound[d.ID] = true
	}
	for _, e := range world.Entities() {
		if !bound[e.ID] {
			return worldSaveUnsupportedf("actor %d has no persisted source-backed class constructor", e.ID)
		}
		if e.Alive() {
			for _, a := range state.Actors {
				if a.EntityID == e.ID {
					savedObjectSetValue(&state.Document.Objects[a.ObjectIndex-1], "Stage", uint32(sim.DecayNone))
					break
				}
			}
		}
		if !e.Alive() && e.Decay >= 1 {
			for _, a := range state.Actors {
				if a.EntityID == e.ID {
					r := &state.Document.Objects[a.ObjectIndex-1]
					savedObjectSetValue(r, "Stage", uint32(e.Decay))
					savedObjectSetValue(r, "Health", uint32(uint16(e.HP)))
					savedObjectSetValue(r, "U6C", uint32(e.Dwell))
					if e.SourceBinding.Class != 0 {
						savedObjectSetValue(r, "RuntimeID", e.SourceBinding.RuntimeID)
					}
					p, err := savedMotionRaw(r, "Block12", 12)
					if err != nil {
						return err
					}
					// projectMotion has already selected the authoritative ordinary
					// Position. Keep its fractional coordinates for a superseded
					// dead mover; only this value owner updates the integer cell.
					p[0], p[1], p[2], p[3] = byte(e.X), byte(e.Y), byte(e.X), byte(e.Y)
				}
			}
		}
	}
	return nil
}

// Reconcile dead roots with current actors before Group projection. A bound
// Stage-1 actor cannot remain a dead root; other roots retain their order and
// duplicates. Actor records and children stay untouched.
func rootDeadActors(state *SnapshotSAVDocument, world *sim.World) error {
	stageOne := map[sim.EntityID]bool{}
	for _, e := range world.Entities() {
		if !e.Alive() && e.Decay == sim.DecayFallen {
			stageOne[e.ID] = true
		}
	}
	stale := map[uint16]bool{}
	for _, a := range state.Actors {
		if !a.Retired && stageOne[a.EntityID] {
			stale[a.ObjectIndex] = true
		}
	}
	state.Document.DeadActors = slices.DeleteFunc(state.Document.DeadActors, func(index uint16) bool {
		return stale[index]
	})
	add := func(index uint16) {
		if !slices.Contains(state.Document.DeadActors, index) {
			state.Document.DeadActors = append(state.Document.DeadActors, index)
		}
	}
	terminalBindings, err := currentTerminalActorBindings(state, world)
	if err != nil {
		return err
	}
	for _, terminal := range world.CurrentTerminalActors() {
		if object, bound := terminalBindings[terminal.ID]; bound {
			add(object)
		}
	}
	for _, d := range world.OriginalDeadActors() {
		var index uint16
		for i, r := range state.Document.Objects {
			if r.Class != savedActorClass(d.Source.Class) {
				continue
			}
			key, err := savedStructureValue(&r, "Identity")
			if err == nil && key == d.Source.Identity {
				if index != 0 {
					return worldSaveUnsupportedf("dead source key %#x is ambiguous", key)
				}
				index = uint16(i + 1)
			}
		}
		if index == 0 {
			return worldSaveUnsupportedf("dead actor %d has no exact current binding", d.ID)
		}
		add(index)
	}
	for _, e := range world.Entities() {
		if !e.Dead() || e.Decay < sim.DecayBones {
			continue
		}
		for _, a := range state.Actors {
			if a.EntityID == e.ID && !a.Retired {
				add(a.ObjectIndex)
			}
		}
	}
	return nil
}

func projectItems(state *SnapshotSAVDocument, world *sim.World, tables ...*mapload.Table) error {
	r := world.SavedObjects()
	if r == nil {
		r = &sim.SavedObjects{Version: sim.SavedObjectsVersion, NextID: 1}
	}
	priorKeys := make(map[sim.SavedObjectID]uint32)
	var bindings map[sim.SavedObjectID]uint16
	if state.Objects != nil {
		bindings = savedObjectIndices(state.Objects)
	}
	for _, row := range r.Sacks {
		if row.Retired || row.Token.Identity != 0 || bindings[row.ID] == 0 {
			continue
		}
		index := bindings[row.ID]
		if int(index) > len(state.Document.Objects) || state.Document.Objects[index-1].Class != "Sack" {
			return fmt.Errorf("current Sack %d has a different ordinary object", row.ID)
		}
		key, err := savedStructureValue(&state.Document.Objects[index-1], "Identity")
		if err != nil {
			return err
		}
		priorKeys[row.ID] = key
	}
	if err := projectCurrentItemGraphForSave(state, world, tables...); err != nil {
		return err
	}
	indices := savedObjectIndices(state.Objects)
	check := func(id sim.SavedObjectID, c sim.SavedObjectCoverage, allowed sim.SavedObjectUnknown) error {
		if c.Unsupported != "" || c.Unknown & ^allowed != 0 {
			return worldSaveUnsupportedf("object %d: %s", id, savedObjectCoverage(c))
		}
		if indices[id] == 0 {
			return worldSaveUnsupportedf("current object %d is detached from the complete SAV graph", id)
		}
		return nil
	}
	keys, err := sav.ReserveDocumentKeys(*state.Document, len(r.Sacks))
	if err != nil {
		return err
	}
	keyIndex := 0
	planes, hasPlanes := world.SavedCellPlanes()
	for _, row := range r.Sacks {
		if row.Retired {
			continue
		}
		allowed := sim.SavedUnknownIdentity | sim.SavedUnknownToken | sim.SavedUnknownContainerLoad
		if err := check(row.ID, row.Coverage, allowed); err != nil {
			return err
		}
		if row.Token.Identity != 0 {
			continue
		}
		var ground *sim.Sack
		for _, sack := range world.Sacks() {
			if sack.ObjectID == row.ID {
				copy := sack
				ground = &copy
				break
			}
		}
		if ground == nil {
			return worldSaveUnsupportedf("Sack %d has no current ground root", row.ID)
		}
		// The current record belongs to this exact Sack binding. Reuse its
		// ordinary key while the native key remains absent across LOAD.
		key := priorKeys[row.ID]
		if key == 0 {
			key = keys[keyIndex]
			keyIndex++
		}
		object := &state.Document.Objects[indices[row.ID]-1]
		savedObjectSetValue(object, "Identity", key)
		cell := uint16(ground.X) | uint16(ground.Y)<<8
		found := false
		for i := range state.Document.World.Cells {
			c := &state.Document.World.Cells[i]
			if c.Cell == cell {
				if c.Sack != 0 && c.Sack != key {
					return worldSaveUnsupportedf("new Sack %d collides with existing cell root", row.ID)
				}
				c.Sack = key
				found = true
			}
		}
		if found {
			// SAV-SACKENTRY-590: existing-node reuse changes only the Sack
			// slot. Its baselines and current planes remain independent.
			continue
		}
		cost, static, dynamic := newSackCellPlanes(world, state.Document.World, planes, cell)
		state.Document.World.Cells = append(state.Document.World.Cells, sav.DocumentCellData{Cell: cell, Cost: cost, Static: static, Sack: key})
		// The new Cell keeps its baseline; recomputation marks the Block.
		blockStatic := static
		if !hasPlanes {
			blockStatic |= 0x20
		}
		block := sav.BlockRecord{Cell: cell, Dyn: dynamic, Static: blockStatic}
		found = false
		for i := range state.Document.World.Blocks {
			if state.Document.World.Blocks[i].Cell == cell {
				state.Document.World.Blocks[i] = block
				found = true
			}
		}
		if !found {
			state.Document.World.Blocks = append(state.Document.World.Blocks, block)
		}
	}
	slices.SortFunc(state.Document.World.Cells, func(a, b sav.DocumentCellData) int { return int(a.Cell) - int(b.Cell) })
	slices.SortFunc(state.Document.World.Blocks, func(a, b sav.BlockRecord) int { return int(a.Cell) - int(b.Cell) })
	return nil
}

// newSackCellPlanes reads live or carried planes.
func newSackCellPlanes(world *sim.World, doc *sav.DocumentWorldData, planes *sim.SavedCellPlanes, cell uint16) (cost, static, dynamic byte) {
	if planes != nil {
		return planes.Cost[cell], planes.Static[cell], planes.Dynamic[cell]
	}
	terrain := world.CurrentPolicy().Terrain
	at := int(cell>>8)*int(world.Bounds().Width) + int(cell&0xff)
	for _, block := range doc.Blocks {
		if block.Cell == cell {
			static = block.Static
			break
		}
	}
	grid := terrain.Block[at]
	static = static&^7 | grid&3 | (grid&8)>>1
	return terrain.Cost[at], static, static | 0x20
}
