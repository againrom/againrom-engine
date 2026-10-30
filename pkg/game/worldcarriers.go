package game

import (
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// worldCarrier is one family of World state a SAV carries. home names the
// World state it lives in. Each hook is one step of the family; a nil hook
// is a step the family does not take here. capture copies onto a Snapshot
// what the World's byte form does not carry, and decode puts it on the
// SAVE's copy of the World. absence edits that copy before any document
// writer reads it and reports whether it applied. transform derives the
// family again on a loaded World, seat puts it into the World a mission
// opens with, and writer writes it into the finished document; a late
// writer runs after the mod mark and the clamps.
type worldCarrier struct {
	home      string
	capture   func(live *sim.World, s *Snapshot)
	decode    func(s Snapshot, w *sim.World)
	absence   func(w *sim.World) bool
	transform func(c carrierLoad)
	seat      func(c carrierSeat) error
	writer    func(c carrierSave) error
	late      bool
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
}

// carrierSave is one SAVE's document under construction and its snapshot.
type carrierSave struct {
	f   *FrontEnd
	s   *Snapshot
	doc *sav.DocumentData
}

// worldCarrierSet is the one set of World carriers, in the order each step
// runs. The SAV producer, the LOAD restore and the mission seat run them
// through the functions below and call no hook of a carrier directly.
func worldCarrierSet() []worldCarrier {
	return []worldCarrier{
		{home: "World.LoadedRecord, written by the SAV LOAD restore",
			capture: func(live *sim.World, s *Snapshot) { s.loadedRecord = live.LoadedRecord() },
			decode:  func(s Snapshot, w *sim.World) { w.SetLoadedRecord(s.loadedRecord) }},
		{home: "World saved spell effects and projectiles (DIV-4175)",
			absence: leaveUnboundEffectsOut},
		{home: "a Pointer's UseAmount, the action slot its map placement derives",
			transform: func(c carrierLoad) { restorePointerSlots(c.w, c.m, c.table) }},
		{home: "World ROM2 scenario state, the second game's Scenario bank",
			seat: func(c carrierSeat) error { return c.rules.seatWorld(c.r, c.ms) }},
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
		{home: "World carrier absence", writer: func(c carrierSave) error { return finalizeCurrentCarrierAbsence(c.doc, c.s.World) }},
		{home: "cell plane residue", writer: func(c carrierSave) error { return c.f.finalizeCurrentPlaneResidue(c.doc, *c.s) }},
		{home: "World spell-effect Tokens", late: true,
			writer: func(c carrierSave) error { return finalizeCurrentWorldEffectTokens(c.doc) }},
	}
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
// beside the byte form.
func decodeWorldCarriers(s Snapshot, w *sim.World) {
	for _, c := range worldCarrierSet() {
		if c.decode != nil {
			c.decode(s, w)
		}
	}
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

// seatWorldCarriers seats each family a mission entry seats.
func seatWorldCarriers(c carrierSeat) error {
	for _, carrier := range worldCarrierSet() {
		if carrier.seat != nil {
			if err := carrier.seat(c); err != nil {
				return err
			}
		}
	}
	return nil
}

// writeWorldCarriers runs the document writers of one stage.
func writeWorldCarriers(c carrierSave, late bool) error {
	for _, carrier := range worldCarrierSet() {
		if carrier.writer != nil && carrier.late == late {
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
