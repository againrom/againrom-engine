package game

import (
	"slices"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// worldCarrier is one family of state a SAV carries. home names the state
// it lives in. Each hook is one step of the family; a nil hook is a step the
// family does not take here. capture copies onto a Snapshot what the World's
// byte form does not carry. decode puts the family on a World from one
// record, keyed by the record it reads. absence edits a SAVE's World copy
// before any document writer reads it and reports whether it applied.
// transform derives the family again on a loaded World, seat puts it into the
// World a mission starts with, keyed by the start it runs at, and writer
// writes it into the document at its stage.
type worldCarrier struct {
	home      string
	capture   func(live *sim.World, s *Snapshot)
	decode    map[carrierRecord]func(c carrierDecode) error
	absence   func(w *sim.World) bool
	transform func(c carrierLoad)
	seat      map[carrierSeatPoint]func(c carrierSeat) error
	writer    func(c carrierSave) error
	stage     carrierStage
	// build runs inside the generated document builder, keyed by its point.
	build map[carrierBuildPoint]func(c carrierBuild) error
	// town moves the family between the town and its saved forms, keyed by
	// the point it runs at.
	town map[carrierTownPoint]func(c carrierTown) error
}

// carrierRecord is the record a decode reads, and so the point it runs at.
type carrierRecord uint8

const (
	// snapshotRecord is a SAVE's captured Snapshot, decoded onto its World
	// copy before the document writers read it.
	snapshotRecord carrierRecord = iota
	// archiveLiving is an original LOAD's archive walk, once the living
	// actors are admitted.
	archiveLiving
	// archiveDead is an original LOAD's dead actor records, once the living
	// actors are restored.
	archiveDead
	// documentRecord is the SAV document a LOAD binds to its World.
	documentRecord
	// actionsRecord is the engine's actions leaf a LOAD restores, once the
	// session head is restored.
	actionsRecord
	// fileRecord is a SAV file's decoded document, as a LOAD keeps it.
	fileRecord
	// materialRecord is a SAVE's Snapshot as the World-built document
	// replaces its loaded one.
	materialRecord
	// identityRecord is the actions leaf's actor identities, once the absent
	// Players are matched. The identity remap runs here, once.
	identityRecord
	// terminalRecord is the actions leaf's terminal actors, inside the
	// identity remap once the World's actor identities are restored.
	terminalRecord
	// selectionRecord is the actions leaf before the continuation is
	// restored; orderRecord follows it.
	selectionRecord
	orderRecord
	// tokenRecord is the actions leaf once the native actor bases are
	// restored.
	tokenRecord
	// cellRecord is the actions leaf once the script program is restored.
	cellRecord
	// heldRecord is a LOAD's SAV document once the mission holds it and its
	// World holds the loaded record.
	heldRecord
	// resumeRecord is a native resume's Snapshot document once the mission
	// holds it.
	resumeRecord
	// entryRecord is an original LOAD's mission document once the spell book
	// walk resumes, before the restored Groups are counted.
	entryRecord
)

// carrierDecode is what a decode reads: the World it decodes onto and the
// record of its point. A field the point does not have is nil.
type carrierDecode struct {
	w       *sim.World
	s       *Snapshot
	ms      *Mission
	src     *originalSource
	report  *OriginalSaveResume
	state   *SnapshotSAVDocument
	actions *currentActionData
	// terminal is the terminal actor rows of terminalRecord.
	terminal []sim.CurrentTerminalActor
}

// carrierStage is the point of the document producer a writer runs at.
type carrierStage uint8

const (
	// keysStage runs after the document keys are complete.
	keysStage carrierStage = iota
	// lateStage runs after the mod mark and the clamps.
	lateStage
	// graphStage runs on the mission document's graph, after the actor
	// motion and before the removed native bases are projected.
	graphStage
	// graftStage runs on the mission document once the World effect order
	// is written, before the terrain.
	graftStage
)

// carrierSeatPoint is the mission start a seat runs at.
type carrierSeatPoint uint8

const (
	// entrySeat is a mission door's entry, seatWorld.
	entrySeat carrierSeatPoint = iota
	// lineSeat is the mission AdvanceLine starts with no door. It takes the
	// between-mission purse the door installs and nothing else.
	lineSeat
	// settleSeat is a mission door's settle, once the driver is open: the
	// state a resume restores on top of it.
	settleSeat
)

// carrierBuildPoint is the point of the generated document builder a build
// step runs at.
type carrierBuildPoint uint8

const (
	// seatBuild is the builder made, before any key is reserved.
	seatBuild carrierBuildPoint = iota
	// keysBuild is once the World's keys are reserved, before any key is
	// minted.
	keysBuild
	// joinBuild is once the retained keys are claimed.
	joinBuild
	// deadRecordBuild is each dead record, once its dead tuple is written.
	deadRecordBuild
	// playerBuild is each Player record, before its name is written.
	playerBuild
	// effectsBuild is once the projectile leaves are written.
	effectsBuild
	// absentDeadBuild is each dead record whose body the World holds as no
	// entity, once its dead record is built.
	absentDeadBuild
)

// carrierBuild is what a build step reads: the builder, the SAVE's Snapshot
// and World, at deadRecordBuild and absentDeadBuild the dead record, its
// body and its mission, and at playerBuild the Player's current ID and the
// name its record takes.
type carrierBuild struct {
	b      *generatedDocumentBuilder
	s      *Snapshot
	w      *sim.World
	ms     *Mission
	record *sav.DocumentRecordData
	body   sim.OriginalDeadRecord
	entity bool
	player uint32
	name   *string
}

// carrierTownPoint is the point a town step runs at.
type carrierTownPoint uint8

const (
	// townCapture copies the town's family onto a captured Snapshot.
	townCapture carrierTownPoint = iota
	// townRestore puts a Snapshot's family on the town it restores.
	townRestore
	// townArchive puts an original city file's family on the town.
	townArchive
	// townCarry keeps on the town what the ended mission's World holds.
	townCarry
	// townWrite writes a Snapshot's family into its city document.
	townWrite
)

// carrierTown is what a town step reads. A field the point does not have is
// nil.
type carrierTown struct {
	t      *Town
	s      *Snapshot
	w      *sim.World
	player *sav.Record
	doc    *sav.DocumentData
}

// carrierLoad is what a LOAD transform reads: the loaded World, its map and
// the definition table.
type carrierLoad struct {
	w     *sim.World
	m     *alm.Map
	table *mapload.Table
}

// carrierSeat is a mission entry: the request, its campaign rules and the
// started mission.
type carrierSeat struct {
	r     missionRequest
	rules campaignService
	ms    *Mission
	// mw is the opened driver of settleSeat.
	mw *mapWorld
}

// carrierSave is one SAVE's document under construction and its snapshot.
// doc is the finished document of the keys and late stages; state and world
// are the mission graph and the World copy of the graph stage. The graft
// stage reads all three: doc is the mission document there.
type carrierSave struct {
	f     *FrontEnd
	s     *Snapshot
	doc   *sav.DocumentData
	state *SnapshotSAVDocument
	world *sim.World
}

// worldCarrierSet is the one set of carriers, in the order each step runs.
// The SAV producer, the LOAD restore and the mission seat run them through
// the functions below and call no hook of a carrier directly.
func worldCarrierSet() []worldCarrier {
	return []worldCarrier{
		{home: "World.LoadedRecord, written by the SAV LOAD restore",
			capture: func(live *sim.World, s *Snapshot) { s.loadedRecord = live.LoadedRecord() },
			decode: map[carrierRecord]func(carrierDecode) error{
				snapshotRecord: func(c carrierDecode) error { c.w.SetLoadedRecord(c.s.loadedRecord); return nil }}},
		{home: "the loaded document the mission writers and the generated document builder read",
			decode: map[carrierRecord]func(carrierDecode) error{
				materialRecord: func(c carrierDecode) error { c.s.loadedDocument = c.s.SavedDocument; return nil }},
			build:  map[carrierBuildPoint]func(carrierBuild) error{seatBuild: seatLoadedDocument, joinBuild: joinLoadedActors, playerBuild: keepLoadedPlayerName},
			writer: func(c carrierSave) error { return graftUnknownObjects(c.doc, c.state, c.s.loadedDocument, c.world) }, stage: graftStage},
		{home: "World actor identities of the actions leaf, remapped once",
			decode: map[carrierRecord]func(carrierDecode) error{
				identityRecord: func(c carrierDecode) error { return restoreCurrentActorIdentities(c.ms, c.actions) }}},
		{home: "World terminal actors and their held state (CurrentTerminalActor.held)",
			decode: map[carrierRecord]func(carrierDecode) error{terminalRecord: importCurrentTerminalActors}},
		{home: "World actor actions, with the order words U40 and U5C the loaded record keeps for a body no entity holds",
			writer: func(c carrierSave) error { return projectSavedActorActions(c.state, c.world, c.s.loadedDocument) }, stage: graphStage},
		{home: "World saved spell effects and projectiles (DIV-4175)",
			absence: leaveUnboundEffectsOut,
			build: map[carrierBuildPoint]func(carrierBuild) error{
				keysBuild: func(c carrierBuild) error {
					return c.b.reserveWorldEffectKeys(worldEffectSource(c.s.SavedDocument), c.w)
				},
				effectsBuild: func(c carrierBuild) error { return c.b.currentWorldEffects(worldEffectSource(c.s.SavedDocument), c.w) }}},
		{home: "a Pointer's UseAmount, the action slot its map placement derives",
			transform: func(c carrierLoad) { restorePointerSlots(c.w, c.m, c.table) }},
		{home: "World ROM2 scenario state, the second game's Scenario bank",
			seat: map[carrierSeatPoint]func(carrierSeat) error{
				entrySeat: func(c carrierSeat) error { return c.rules.seatWorld(c.r, c.ms) }}},
		{home: "World purse of the local Player, unless the loaded record binds the Player purses",
			seat: map[carrierSeatPoint]func(carrierSeat) error{entrySeat: seatTownPurse, lineSeat: seatTownPurse}},
		{home: "World saved Diaries, by owner (sim.CompareSavedDiaryOwners)",
			decode: map[carrierRecord]func(carrierDecode) error{
				archiveLiving: func(c carrierDecode) error {
					return applyOriginalDiaries(c.ms, c.src.diaryPlayerRec, c.src.diaryChars, c.report)
				},
				documentRecord: func(c carrierDecode) error { return importSavedDiaries(c.ms, c.state) }},
			seat:   map[carrierSeatPoint]func(carrierSeat) error{entrySeat: seatTownDiaries},
			writer: func(c carrierSave) error { return projectSavedDiaries(c.state, c.world) }, stage: graphStage,
			town: map[carrierTownPoint]func(carrierTown) error{
				townCapture: func(c carrierTown) error { c.s.knowledge = slices.Clone(c.t.knowledge); return nil },
				townRestore: func(c carrierTown) error { c.t.knowledge = slices.Clone(c.s.knowledge); return nil },
				townArchive: func(c carrierTown) error { c.t.knowledge = savedPlayerDiary(c.player); return nil },
				townCarry:   carryTownKnowledge,
				townWrite:   func(c carrierTown) error { return projectTownKnowledge(c.doc, c.s.knowledge) }}},
		{home: "World original dead bodies and their held state",
			decode: map[carrierRecord]func(carrierDecode) error{
				archiveDead: func(c carrierDecode) error {
					return applyOriginalDeadWithOrigins(c.ms, c.src.dead, c.report, c.src.missionDocument, c.src.documentOrigins)
				},
				heldRecord: restoreSavedDeadArt},
			writer: func(c carrierSave) error { return projectDeadActors(c.state, c.world, c.s.loadedDocument) }, stage: graphStage,
			build: map[carrierBuildPoint]func(carrierBuild) error{
				keysBuild:       func(c carrierBuild) error { return c.b.reserveRetainedDeadEffectKeys(c.w) },
				deadRecordBuild: retainLoadedDead,
				absentDeadBuild: writeDeadArt}},
		{home: "town spell-effect owners",
			writer: func(c carrierSave) error {
				if c.s.Mission == 0 {
					finalizeCurrentTownEffectOwners(c.doc)
				}
				return nil
			}},
		{home: "the loaded file's Token Reference misses",
			decode: map[carrierRecord]func(carrierDecode) error{
				fileRecord: func(c carrierDecode) error {
					c.state.ReferenceMisses = loadedTokenReferenceMisses(c.state.Document)
					return nil
				}},
			writer: func(c carrierSave) error {
				if c.s.Mission != 0 && c.s.SavedDocument != nil {
					finalizeLoadedTokenReferences(c.doc, c.s.SavedDocument.ReferenceMisses)
				}
				return nil
			}},
		{home: "object identity absence", writer: func(c carrierSave) error { return finalizeCurrentObjectIdentityAbsence(c.doc) }},
		{home: "order spell absence",
			decode: map[carrierRecord]func(carrierDecode) error{
				orderRecord: func(c carrierDecode) error { return matchCurrentOrderSpellAbsence(c.state.Document, c.actions) }},
			writer: func(c carrierSave) error { return finalizeCurrentOrderSpellAbsence(c.doc) }},
		{home: "spell book selection",
			decode: map[carrierRecord]func(carrierDecode) error{
				selectionRecord: func(c carrierDecode) error { return matchCurrentBookSelection(c.state.Document, c.actions) }},
			writer: func(c carrierSave) error { return finalizeCurrentBookSelection(c.doc) }},
		{home: "World session head absence (the actions leaf's AbsentSessionHead)",
			decode: map[carrierRecord]func(carrierDecode) error{
				actionsRecord: func(c carrierDecode) error {
					return restoreAbsentSessionHead(c.ms.World, c.ms.savedDocument.Document, c.actions.AbsentSessionHead)
				},
				resumeRecord: restoreResumedSessionHead}},
		{home: "World carrier absence: cell tails, cell records, motion cells, Sack keys, blocks and Diaries",
			decode: map[carrierRecord]func(carrierDecode) error{
				actionsRecord: func(c carrierDecode) error {
					return restoreCurrentCarrierAbsence(c.ms.World, c.ms.savedDocument.Document, c.actions)
				}},
			writer: func(c carrierSave) error { return finalizeCurrentCarrierAbsence(c.doc, c.s.World) }},
		{home: "cell plane residue, seated with the cell costs",
			decode: map[carrierRecord]func(carrierDecode) error{cellRecord: restoreCurrentPlaneResidue},
			writer: func(c carrierSave) error { return c.f.finalizeCurrentPlaneResidue(c.doc, *c.s) }},
		{home: "World spell-effect Tokens", stage: lateStage,
			decode: map[carrierRecord]func(carrierDecode) error{
				tokenRecord: func(c carrierDecode) error { return restoreCurrentWorldEffectTokens(c.ms, c.actions) }},
			writer: func(c carrierSave) error { return finalizeCurrentWorldEffectTokens(c.doc) }},
		{home: "World session entry, and the item Token T08 it publishes into the loaded record",
			decode: map[carrierRecord]func(carrierDecode) error{
				entryRecord: func(c carrierDecode) error { publishSessionEntry(c.ms); return nil }}},
		{home: "the application state a resume restores, from the loaded record (Presentation)",
			seat: map[carrierSeatPoint]func(carrierSeat) error{settleSeat: settleLoadedApplication}},
	}
}

// seatTownPurse seeds a mission that does not resume a saved World with the
// between-mission purse. Exact Player bindings have already restored the
// current Money; a participant shortcut must not overwrite it or invent a
// SelfSlot.
func seatTownPurse(c carrierSeat) error {
	r, ms := c.r, c.ms
	if r.snap != nil || ms.savedDocument != nil && ms.savedDocument.PlayerPurses != nil {
		return nil
	}
	gold := uint32(r.town.Gold())
	if r.initialPurse != nil {
		gold = *r.initialPurse
	}
	ms.World.SetPurse(sim.SelfSlot, gold)
	return nil
}

// seatTownDiaries gives a fresh mission the Diaries the town's knowledge
// carries.
func seatTownDiaries(c carrierSeat) error {
	if d := c.r.town.knowledgeDiaries(); c.r.fresh() && len(d) != 0 {
		c.ms.World.SetSavedDiaries(d)
	}
	return nil
}

// seatLoadedDocument gives the generated document builder the document the
// mission was loaded from.
func seatLoadedDocument(c carrierBuild) error {
	if l := c.s.SavedDocument; l != nil {
		c.b.loaded = l.Document
	}
	return nil
}

// joinLoadedActors joins each actor to its loaded record by the identity the
// record holds. The identity is read to join, as a cell record's key is; a
// cell key can name a body another actor stands on. A key the World holds for
// a structure, the terrain or a dead actor stays theirs. No minted key takes
// a joined one.
func joinLoadedActors(c carrierBuild) error {
	w, b := c.w, c.b
	held := map[uint32]bool{currentTerrainKey(w): true}
	if rows, _, present := w.SavedStructures(); present {
		for _, row := range rows {
			held[row.SourceKey] = true
		}
	}
	for _, d := range w.OriginalDeadActors() {
		held[d.Source.Identity] = true
	}
	b.joined = map[sim.EntityID]uint32{}
	if l := c.s.SavedDocument; l != nil && l.Document != nil {
		for _, a := range l.Actors {
			if a.ObjectIndex != 0 && int(a.ObjectIndex) <= len(l.Document.Objects) {
				if key, _ := savedStructureValue(&l.Document.Objects[a.ObjectIndex-1], "Identity"); key != 0 && !held[key] {
					b.joined[a.EntityID], b.currentKeys[key] = key, true
				}
			}
		}
	}
	return nil
}

// retainLoadedDead writes onto a dead record what its loaded record keeps and
// the World does not hold: the retained values, the spellbook and the
// unexecuted Effects.
func retainLoadedDead(c carrierBuild) error {
	s := c.body.Source
	if err := c.b.retainDeadValues(c.record, s, c.entity); err != nil {
		return err
	}
	if err := c.b.retainDeadBook(c.record, s, c.entity); err != nil {
		return err
	}
	return c.b.retainDeadEffects(c.record, c.body.ID, s)
}

// keepLoadedPlayerName gives a Player record the name its loaded record
// holds, over the name the map or the party gives it.
func keepLoadedPlayerName(c carrierBuild) error {
	if saved, present := currentPlayerName(c.s.SavedDocument, c.player); present {
		*c.name = saved
	}
	return nil
}

// restoreSavedDeadArt gives each dead body no map unit placed the art type
// its loaded record holds in T0E.
func restoreSavedDeadArt(c carrierDecode) error {
	ms := c.ms
	if ms.savedDocument == nil || ms.savedDocument.Document == nil {
		return nil
	}
	for _, dead := range ms.World.OriginalDeadActors() {
		if dead.Source.MapUnitID != 0 {
			continue
		}
		for i := range ms.savedDocument.Document.Objects {
			record := &ms.savedDocument.Document.Objects[i]
			if record.Class != savedActorClass(dead.Source.Class) {
				continue
			}
			key, err := savedStructureValue(record, "Identity")
			if err != nil || key != dead.Source.Identity {
				continue
			}
			typeID, err := savedStructureValue(record, "T0E")
			if err == nil {
				if ms.DeadArt == nil {
					ms.DeadArt = map[sim.EntityID]uint16{}
				}
				ms.DeadArt[dead.ID] = uint16(typeID)
			}
		}
	}
	return nil
}

// restoreResumedSessionHead gives a native resume's World the absent session
// head its document's actions leaf records, so a later SAVE does not turn the
// retained constructor bytes into live state.
func restoreResumedSessionHead(c carrierDecode) error {
	document := c.state
	if document == nil || document.Document == nil {
		return nil
	}
	actions, err := readCurrentActions(document.Document)
	if err != nil || actions == nil {
		return err
	}
	return restoreAbsentSessionHead(c.ms.World, document.Document, actions.AbsentSessionHead)
}

// writeDeadArt writes into a dead record whose body the World holds as no
// entity the art type its mission keeps for the body.
func writeDeadArt(c carrierBuild) error {
	if typeID, present := c.ms.DeadArt[c.body.ID]; present {
		mustSetValue(c.record, "T0E", uint32(typeID))
	}
	return nil
}

// importCurrentTerminalActors puts the actions leaf's terminal actor rows in
// the World, which holds each row by its held rule, and gives each departed
// actor its contents.
func importCurrentTerminalActors(c carrierDecode) error {
	if len(c.terminal) == 0 {
		return nil
	}
	if err := c.ms.World.ImportCurrentTerminalActors(c.terminal); err != nil {
		return err
	}
	return restoreDepartedContents(c.ms, c.actions, c.terminal)
}

// restoreCurrentPlaneResidue seats the actions leaf's cell costs with the cell
// plane residue that matches the loaded document.
func restoreCurrentPlaneResidue(c carrierDecode) error {
	residue, err := matchingCurrentPlaneResidue(c.ms.savedDocument.Document, c.actions.CellPlaneResidue)
	if err != nil {
		return err
	}
	return c.ms.World.RestoreCurrentCellCosts(c.actions.CellCosts, residue...)
}

// settleLoadedApplication restores, on the opened driver, the application
// state of a native resume or of the loaded document, and the action view of
// an original LOAD.
func settleLoadedApplication(c carrierSeat) error {
	r, ms, mw := c.r, c.ms, c.mw
	var application *SnapshotApplicationState
	if r.snap != nil {
		application = r.snap.ApplicationState
	}
	if err := mw.restoreApplicationState(application, r.snap == nil && r.prepare != nil && ms.savedDocument != nil && ms.savedDocument.Document != nil); err != nil {
		return err
	}
	if r.snap == nil && r.prepare != nil {
		if err := mw.restoreOriginalActionView(); err != nil {
			return err
		}
		mw.seedRestoredRuns()
		mw.stampAfterLoad()
	}
	return nil
}

// carryTownKnowledge keeps the local Player's Diary the mission ended with,
// so the next mission entered from the town opens with it.
func carryTownKnowledge(c carrierTown) error {
	if c.t == nil || c.w == nil {
		return nil
	}
	c.t.knowledge = nil
	for _, d := range c.w.SavedDiaries() {
		if d.Owner.Player {
			c.t.knowledge = []sim.SavedDiary{d}
		}
	}
	return nil
}

// knowledgeDiaries is the Diary a fresh mission starts with: none before a
// mission has been won in this campaign.
func (t *Town) knowledgeDiaries() []sim.SavedDiary {
	if t == nil {
		return nil
	}
	return t.knowledge
}

// projectTownKnowledge writes the town's carried Diary into the local
// Player's Diary record of a city document, over the record's own Self. A
// town with no carried Diary leaves the record as the producer built it.
func projectTownKnowledge(doc *sav.DocumentData, knowledge []sim.SavedDiary) error {
	var carried *sim.SavedDiary
	for i := range knowledge {
		if knowledge[i].Owner.Player {
			carried = &knowledge[i]
		}
	}
	if carried == nil {
		return nil
	}
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Player" {
			continue
		}
		participant, err := savedStructureValue(r, "Participant")
		if err != nil {
			return err
		}
		if participant != 0 {
			continue
		}
		for j := range r.Inline {
			if r.Inline[j].Record.Class != "Diary" {
				continue
			}
			next, err := sav.ProjectDocumentDiary(r.Inline[j].Record, sav.Diary{Length: carried.Length, Entries: savedDiaryEntriesToSav(carried.Entries)})
			if err != nil {
				return err
			}
			r.Inline[j].Record = next
		}
	}
	return nil
}

// savedPlayerDiary is the local Player's Diary a loaded city file carries, as
// the town's carried Diary. An all-default Diary carries no knowledge and
// leaves the town without one.
func savedPlayerDiary(player *sav.Record) []sim.SavedDiary {
	d, ok, err := sav.PlayerDiary(player)
	if err != nil || !ok || len(d.Entries) == 0 {
		return nil
	}
	return []sim.SavedDiary{{Owner: sim.SavedDiaryOwner{Player: true}, Length: d.Length, Entries: savDiaryEntriesToSaved(d.Entries)}}
}

// captureWorldCarriers copies onto a captured Snapshot what the live World's
// byte form does not carry.
func captureWorldCarriers(live *sim.World, s *Snapshot) {
	for _, c := range worldCarrierSet() {
		if c.capture != nil {
			c.capture(live, s)
		}
	}
}

// decodeWorldCarriers puts on a SAVE's World copy what its snapshot carried
// beside the byte form. A snapshot decode does not fail.
func decodeWorldCarriers(s Snapshot, w *sim.World) {
	_ = decodeWorldCarriersFrom(snapshotRecord, carrierDecode{w: w, s: &s})
}

// decodeWorldCarriersFrom runs each decode that reads the record from, in
// row order.
func decodeWorldCarriersFrom(from carrierRecord, c carrierDecode) error {
	for _, carrier := range worldCarrierSet() {
		if decode := carrier.decode[from]; decode != nil {
			if err := decode(c); err != nil {
				return err
			}
		}
	}
	return nil
}

// leaveAbsentCarriersOut applies each absence rule to a SAVE's World copy and
// reports whether any applied.
func leaveAbsentCarriersOut(w *sim.World) bool {
	applied := false
	for _, c := range worldCarrierSet() {
		if c.absence != nil && c.absence(w) {
			applied = true
		}
	}
	return applied
}

// transformWorldCarriers derives each family a LOAD derives again.
func transformWorldCarriers(c carrierLoad) {
	for _, carrier := range worldCarrierSet() {
		if carrier.transform != nil {
			carrier.transform(c)
		}
	}
}

// seatWorldCarriers runs each seat of the mission start at, in row order.
func seatWorldCarriers(at carrierSeatPoint, c carrierSeat) error {
	for _, carrier := range worldCarrierSet() {
		if seat := carrier.seat[at]; seat != nil {
			if err := seat(c); err != nil {
				return err
			}
		}
	}
	return nil
}

// buildWorldCarriers runs each build step of the builder point at, in row
// order.
func buildWorldCarriers(at carrierBuildPoint, c carrierBuild) error {
	for _, carrier := range worldCarrierSet() {
		if build := carrier.build[at]; build != nil {
			if err := build(c); err != nil {
				return err
			}
		}
	}
	return nil
}

// seatTownCarriers runs each town step of the point at, in row order.
func seatTownCarriers(at carrierTownPoint, c carrierTown) error {
	for _, carrier := range worldCarrierSet() {
		if step := carrier.town[at]; step != nil {
			if err := step(c); err != nil {
				return err
			}
		}
	}
	return nil
}

// writeWorldCarriers runs the document writers of one stage.
func writeWorldCarriers(c carrierSave, stage carrierStage) error {
	for _, carrier := range worldCarrierSet() {
		if carrier.writer != nil && carrier.stage == stage {
			if err := carrier.writer(c); err != nil {
				return err
			}
		}
	}
	return nil
}

// loadedRecordOf is the kind of record a SAV LOAD restores a World from.
func loadedRecordOf(state *SnapshotSAVDocument) sim.LoadedRecord {
	switch {
	case state == nil:
		return sim.NoLoadedRecord
	case state.Document == nil:
		return sim.LoadedWithoutDocument
	}
	return sim.LoadedDocument
}
