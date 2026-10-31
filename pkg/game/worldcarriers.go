package game

import (
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
}

// carrierStage is the point of the document producer a writer runs at.
type carrierStage uint8

const (
	// keysStage runs after the document keys are complete.
	keysStage carrierStage = iota
	// lateStage runs after the mod mark and the clamps.
	lateStage
	// graphStage runs on the mission document's graph, after the actor
	// actions and before the removed native bases are projected.
	graphStage
)

// carrierSeatPoint is the mission start a seat runs at.
type carrierSeatPoint uint8

const (
	// entrySeat is a mission door's entry, seatWorld.
	entrySeat carrierSeatPoint = iota
	// lineSeat is the mission AdvanceLine starts with no door. It takes the
	// between-mission purse the door installs and nothing else.
	lineSeat
)

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
}

// carrierSave is one SAVE's document under construction and its snapshot.
// doc is the finished document of the keys and late stages; state and world
// are the mission graph and the World copy of the graph stage.
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
		{home: "World saved spell effects and projectiles (DIV-4175)",
			absence: leaveUnboundEffectsOut},
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
			writer: func(c carrierSave) error { return projectSavedDiaries(c.state, c.world) }, stage: graphStage},
		{home: "World original dead bodies and their held state",
			decode: map[carrierRecord]func(carrierDecode) error{
				archiveDead: func(c carrierDecode) error {
					return applyOriginalDeadWithOrigins(c.ms, c.src.dead, c.report, c.src.missionDocument, c.src.documentOrigins)
				}},
			writer: func(c carrierSave) error { return projectDeadActors(c.state, c.world, c.s.loadedDocument) }, stage: graphStage},
		{home: "town spell-effect owners",
			writer: func(c carrierSave) error {
				if c.s.Mission == 0 {
					finalizeCurrentTownEffectOwners(c.doc)
				}
				return nil
			}},
		{home: "the loaded file's Token Reference misses",
			writer: func(c carrierSave) error {
				if c.s.Mission != 0 && c.s.SavedDocument != nil {
					finalizeLoadedTokenReferences(c.doc, c.s.SavedDocument.ReferenceMisses)
				}
				return nil
			}},
		{home: "object identity absence", writer: func(c carrierSave) error { return finalizeCurrentObjectIdentityAbsence(c.doc) }},
		{home: "order spell absence", writer: func(c carrierSave) error { return finalizeCurrentOrderSpellAbsence(c.doc) }},
		{home: "spell book selection", writer: func(c carrierSave) error { return finalizeCurrentBookSelection(c.doc) }},
		{home: "World carrier absence: cell tails, cell records, motion cells, Sack keys, blocks and Diaries",
			decode: map[carrierRecord]func(carrierDecode) error{
				actionsRecord: func(c carrierDecode) error {
					return restoreCurrentCarrierAbsence(c.ms.World, c.ms.savedDocument.Document, c.actions)
				}},
			writer: func(c carrierSave) error { return finalizeCurrentCarrierAbsence(c.doc, c.s.World) }},
		{home: "cell plane residue", writer: func(c carrierSave) error { return c.f.finalizeCurrentPlaneResidue(c.doc, *c.s) }},
		{home: "World spell-effect Tokens", stage: lateStage,
			writer: func(c carrierSave) error { return finalizeCurrentWorldEffectTokens(c.doc) }},
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
