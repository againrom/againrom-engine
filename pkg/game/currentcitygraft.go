package game

import (
	"encoding/binary"
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

type currentCityActorGraft struct {
	Source, Current    uint16
	Returned           bool
	CurrentFacing      bool
	CurrentDiary       bool
	CurrentHire        bool
	RetainedHumanTails bool
	CurrentCell        bool
	Cell               uint16
	CurrentMapUnit     bool
	MapUnitID          uint16
}

type currentCityPlayerGraft struct {
	Source, Current  uint16
	CurrentAutoheal  bool
	CurrentFormation bool
	CurrentDiary     bool
}

func graftSnapshotCityResidue(current sav.DocumentData, snapshot Snapshot) (sav.DocumentData, error) {
	retained := snapshot.OriginalCity
	if retained == nil || retained.Unavailable != "" {
		if err := avoidCurrentCityGraphKeyCollisions(current, snapshot.CityObjects); err != nil {
			return sav.DocumentData{}, err
		}
		return current, nil
	}
	city, err := sav.CityFromData(retained.Document)
	if err != nil {
		return sav.DocumentData{}, err
	}
	source, err := city.DocumentData()
	if err != nil {
		return sav.DocumentData{}, err
	}
	identities := func(doc sav.DocumentData, name string) map[uint32]uint16 {
		indices, duplicates := map[uint32]uint16{}, map[uint32]bool{}
		for i, record := range doc.Objects {
			key, err := savedStructureValue(&record, name)
			if err != nil || key == 0 {
				continue
			}
			if indices[key] != 0 {
				duplicates[key] = true
			}
			indices[key] = uint16(i + 1)
		}
		for key := range duplicates {
			delete(indices, key)
		}
		return indices
	}
	sourceActors, currentActors := identities(source, "Identity"), identities(current, "Identity")
	sourcePlayers, currentPlayers := identities(source, "This"), identities(current, "This")
	byParty, counts := map[string]SnapshotCityBinding{}, map[string]int{}
	for _, b := range retained.Bindings {
		byParty[b.PartyID] = b
		counts[b.PartyID]++
	}
	currentCounts := map[string]int{}
	for _, p := range snapshot.Party {
		currentCounts[p.ID]++
	}
	var actors []currentCityActorGraft
	playerSources, ambiguous := map[uint16]uint16{}, map[uint16]bool{}
	for i, p := range snapshot.Party {
		b, ok := byParty[p.ID]
		if !ok || p.ID == "" || counts[p.ID] != 1 || currentCounts[p.ID] != 1 {
			continue
		}
		from, to := sourceActors[b.Identity], currentActors[nativeCityIdentity(i)]
		if from == 0 || to == 0 || !cityGraftActorClass(source.Objects[from-1].Class) {
			continue
		}
		_, currentTails := currentCityHumanTails(p)
		graft := currentCityActorGraft{Source: from, Current: to, Returned: b.Returned != nil, RetainedHumanTails: !currentTails, CurrentHire: p.Hired()}
		if p.Saved != nil {
			graft.CurrentMapUnit, graft.MapUnitID = true, p.Saved.MapUnitID
		}
		if c := p.Saved; c != nil && c.Cell.X >= 0 && c.Cell.X <= 0xff && c.Cell.Y >= 0 && c.Cell.Y <= 0xff {
			graft.CurrentCell, graft.Cell = true, uint16(c.Cell.Y)<<8|uint16(c.Cell.X)
		}
		actors = append(actors, graft)
		oldOwner, _ := savedStructureValue(&source.Objects[from-1], "Reference")
		newOwner, _ := savedStructureValue(&current.Objects[to-1], "Reference")
		sp, cp := sourcePlayers[oldOwner], currentPlayers[newOwner]
		if sp == 0 || cp == 0 {
			continue
		}
		if previous := playerSources[cp]; previous != 0 && previous != sp {
			ambiguous[cp] = true
		}
		playerSources[cp] = sp
	}
	_, autoheal, err := carriedAutoHealing(snapshot.Party)
	if err != nil {
		return sav.DocumentData{}, err
	}
	var players []currentCityPlayerGraft
	for _, cp := range current.Players {
		if sp := playerSources[cp]; sp != 0 && !ambiguous[cp] {
			players = append(players, currentCityPlayerGraft{Source: sp, Current: cp, CurrentAutoheal: autoheal, CurrentFormation: snapshot.ApplicationState != nil})
		}
	}
	out, err := graftCurrentCityResidue(current, source, actors, players)
	if err != nil {
		return sav.DocumentData{}, err
	}
	if err := avoidCurrentCityGraphKeyCollisions(out, snapshot.CityObjects); err != nil {
		return sav.DocumentData{}, err
	}
	return out, nil
}

// Current city item/effect/spell records retain native token keys in the
// detached topology. Those keys are outside DocumentData, so the current
// producer's deterministic key reservation cannot see them. If a retained
// token key is one of the keys the producer may mint (or is already owned by a
// current document object), clear only that token's own key and let the
// producer mint a fresh one while preserving the rest of its token state.
func avoidCurrentCityGraphKeyCollisions(doc sav.DocumentData, graph *cityObjectTopology) error {
	if graph == nil {
		return nil
	}
	blocked := map[uint32]bool{0: true}
	for _, record := range doc.Objects {
		for _, value := range record.Values {
			if value.Name == "Identity" || value.Name == "This" {
				blocked[value.Value] = true
			}
		}
	}
	itemIDs := make([]sim.SavedObjectID, 0, len(graph.ItemRecords))
	for id := range graph.ItemRecords {
		itemIDs = append(itemIDs, id)
	}
	slices.Sort(itemIDs)
	for _, id := range itemIDs {
		record := graph.ItemRecords[id]
		if record.Token.Identity != 0 && blocked[record.Token.Identity] {
			record.Token.Identity = 0
			graph.ItemRecords[id] = record
			continue
		}
		if record.Token.Identity != 0 {
			blocked[record.Token.Identity] = true
		}
	}
	effectIDs := make([]sim.SavedObjectID, 0, len(graph.EffectRecords))
	for id := range graph.EffectRecords {
		effectIDs = append(effectIDs, id)
	}
	slices.Sort(effectIDs)
	for _, id := range effectIDs {
		record := graph.EffectRecords[id]
		if record.Token.Identity != 0 && blocked[record.Token.Identity] {
			record.Token.Identity = 0
			graph.EffectRecords[id] = record
			continue
		}
		if record.Token.Identity != 0 {
			blocked[record.Token.Identity] = true
		}
	}
	spellIDs := make([]sim.SavedObjectID, 0, len(graph.SpellRecords))
	for id := range graph.SpellRecords {
		spellIDs = append(spellIDs, id)
	}
	slices.Sort(spellIDs)
	for _, id := range spellIDs {
		record := graph.SpellRecords[id]
		if record.This != 0 && blocked[record.This] {
			record.This = 0
			graph.SpellRecords[id] = record
			continue
		}
		if record.This != 0 {
			blocked[record.This] = true
		}
	}
	return nil
}

// reserveCurrentCityDocumentKeys keeps detached native graph identities out of
// the producer's actual allocation stream. ReserveDocumentKeys cannot see the
// detached graph, and reserving a fixed upper bound here would erase perfectly
// valid native keys merely because they happen to fall in that unused window.
func reserveCurrentCityDocumentKeys(doc sav.DocumentData, graph *cityObjectTopology, count int) ([]uint32, error) {
	if graph == nil {
		return sav.ReserveDocumentKeys(doc, count)
	}
	reservation := doc
	reservation.Objects = slices.Clone(doc.Objects)
	appendKey := func(key uint32) {
		if key == 0 {
			return
		}
		reservation.Objects = append(reservation.Objects, sav.DocumentRecordData{
			Values: []sav.DocumentValueData{{Name: "Identity", Value: key}},
		})
	}
	for _, record := range graph.ItemRecords {
		appendKey(record.Token.Identity)
	}
	for _, record := range graph.EffectRecords {
		appendKey(record.Token.Identity)
	}
	for _, record := range graph.SpellRecords {
		appendKey(record.This)
	}
	return sav.ReserveDocumentKeys(reservation, count)
}

// Bindings identify surviving objects, never equal values or old roster slots.
// Current roots and memberships remain authoritative; only Diary edges add
// retained objects. The caller supplies the newest available retained document.
func graftCurrentCityResidue(current, retained sav.DocumentData, actors []currentCityActorGraft, players []currentCityPlayerGraft) (sav.DocumentData, error) {
	doc, err := sav.CloneDocumentData(current)
	if err != nil {
		return sav.DocumentData{}, err
	}
	if len(actors) == 0 && len(players) == 0 {
		return doc, nil
	}
	source, err := sav.CloneDocumentData(retained)
	if err != nil {
		return sav.DocumentData{}, err
	}
	if doc.World != nil {
		return sav.DocumentData{}, fmt.Errorf("city residue requires a city current graph")
	}
	// Temporary keys must avoid retained missing references as well as identities.
	reserve := doc
	reserve.Objects = append(slices.Clone(doc.Objects), source.Objects...)
	reserve.World = source.World
	keys, err := sav.ReserveDocumentKeys(reserve, len(doc.Objects))
	if err != nil {
		return sav.DocumentData{}, err
	}
	currentKeys := map[uint32]uint32{}
	for i := range doc.Objects {
		for j := range doc.Objects[i].Values {
			v := &doc.Objects[i].Values[j]
			if v.Name != "Identity" && v.Name != "This" {
				continue
			}
			if v.Value != 0 {
				if _, exists := currentKeys[v.Value]; exists {
					return sav.DocumentData{}, fmt.Errorf("city current identity is ambiguous")
				}
				currentKeys[v.Value] = keys[i]
			}
			v.Value = keys[i]
		}
	}
	for i := range doc.Objects {
		translateCityGraftKeys(&doc.Objects[i], currentKeys)
	}
	memo := map[uint16]uint16{0: 0}
	used := map[uint16]bool{}
	retainedKeys := map[uint32]uint32{}
	bind := func(from, to uint16, player bool) error {
		if from == 0 || int(from) > len(source.Objects) || to == 0 || int(to) > len(doc.Objects) {
			return fmt.Errorf("city residue binding is outside its document")
		}
		if _, found := memo[from]; found || used[to] {
			return fmt.Errorf("city residue binding is ambiguous")
		}
		a, b := &source.Objects[from-1], &doc.Objects[to-1]
		identity := "Identity"
		if player {
			if a.Class != "Player" || b.Class != "Player" {
				return fmt.Errorf("city residue Player binding has another class")
			}
			identity = "This"
		} else if !cityGraftActorClass(a.Class) || !cityGraftActorClass(b.Class) {
			return fmt.Errorf("city residue actor binding has another class")
		}
		old, e := savedStructureValue(a, identity)
		if e != nil {
			return e
		}
		next, e := savedStructureValue(b, identity)
		if e != nil {
			return e
		}
		if old != 0 {
			if prior, exists := retainedKeys[old]; exists && prior != next {
				return fmt.Errorf("city retained identity is ambiguous")
			}
			retainedKeys[old] = next
		}
		memo[from], used[to] = to, true
		return nil
	}
	for _, b := range actors {
		if err := bind(b.Source, b.Current, false); err != nil {
			return sav.DocumentData{}, err
		}
	}
	for _, b := range players {
		if err := bind(b.Source, b.Current, true); err != nil {
			return sav.DocumentData{}, err
		}
	}
	for i := range source.Objects {
		translateCityGraftKeys(&source.Objects[i], retainedKeys)
	}
	retainDiary := func(index uint16) (uint16, error) {
		if index == 0 {
			return 0, nil
		}
		if int(index) > len(source.Objects) || source.Objects[index-1].Class != "Diary" {
			return 0, fmt.Errorf("city retained Diary edge has another class")
		}
		if found, ok := memo[index]; ok {
			return found, nil
		}
		if len(doc.Objects) >= 0x7fff {
			return 0, fmt.Errorf("city residue exceeds object reference range")
		}
		doc.Objects = append(doc.Objects, source.Objects[index-1])
		memo[index] = uint16(len(doc.Objects))
		return memo[index], nil
	}
	for _, b := range actors {
		old := &source.Objects[b.Source-1]
		r := doc.Objects[b.Current-1]
		mergeCityActorResidue(&r, old, b)
		if !b.CurrentDiary {
			for i := range r.RefSlots {
				slot := &r.RefSlots[i]
				if slot.Name != "Diary" || len(slot.Objects) != 1 || slot.Objects[0] != 0 {
					continue
				}
				for _, prior := range old.RefSlots {
					if prior.Name != "Diary" || len(prior.Objects) != 1 {
						continue
					}
					index, err := retainDiary(prior.Objects[0])
					if err != nil {
						return sav.DocumentData{}, err
					}
					slot.Objects[0] = index
				}
			}
		}
		doc.Objects[b.Current-1] = r
	}
	for _, b := range players {
		mergeCityPlayerResidue(&doc.Objects[b.Current-1], &source.Objects[b.Source-1], b)
	}
	doc, _, err = sav.ReindexDocumentData(doc)
	return doc, err
}

func cityGraftActorClass(class string) bool {
	return class == "Unit" || class == "Humanoid" || class == "Human"
}

func cityGraftCurrentActorValue(name string) bool {
	switch name {
	case "Identity", "Reference", "T0C", "T18", "U4B", "U4C", "HasInventory", "Inventory1C", "Inventory20", "HasSpellbook",
		"Body", "Reaction", "Mind", "Spirit", "Speed", "U8E", "U90", "Capacity", "Health", "HealthMax", "HealthRegen", "Mana", "ManaMax", "ManaRegen",
		"UA0", "UA4", "UA2", "UA3", "U12C", "U130", "U134", "U135":
		return true
	}
	return false
}

func mergeCityActorResidue(current, source *sav.DocumentRecordData, binding currentCityActorGraft) {
	for i := range current.Values {
		v := &current.Values[i]
		for _, old := range source.Values {
			if old.Name != v.Name {
				continue
			}
			if v.Name == "U4C" {
				v.Value = (old.Value &^ 6) | (v.Value & 6)
			} else if v.Name == "T08" && binding.CurrentMapUnit {
				v.Value = (old.Value &^ 0xffff) | uint32(binding.MapUnitID)
			} else if v.Name == "U148" && binding.CurrentHire {
				v.Value = (old.Value &^ 0xff) | (v.Value & 0xff)
			} else if !cityGraftCurrentActorValue(v.Name) {
				v.Value = old.Value
			}
			break
		}
	}
	for i := range current.Raw {
		r := &current.Raw[i]
		switch r.Name {
		case "H1CC", "UBE":
			continue
		case "UA6", "U114", "UD4":
			if binding.RetainedHumanTails {
				offset := 22
				if r.Name == "UD4" {
					offset = 40
				}
				for _, old := range source.Raw {
					if old.Name == r.Name && len(old.Bytes) >= offset+2 && len(r.Bytes) >= offset+2 {
						copy(r.Bytes[offset:offset+2], old.Bytes[offset:offset+2])
						break
					}
				}
			}
			continue
		}
		for _, old := range source.Raw {
			if old.Name != r.Name {
				continue
			}
			if r.Name == "U154" {
				speed, facing := r.Bytes[10], r.Bytes[0]
				r.Bytes = slices.Clone(old.Bytes)
				r.Bytes[10] = speed
				if binding.CurrentFacing {
					r.Bytes[0] = facing
				}
			} else {
				r.Bytes = slices.Clone(old.Bytes)
			}
			if r.Name == "Block12" && binding.CurrentCell && len(r.Bytes) >= 2 {
				binary.LittleEndian.PutUint16(r.Bytes, binding.Cell)
			}
			break
		}
	}
	for i := range current.Counts {
		count := &current.Counts[i]
		switch count.Name {
		case "U15C", "U178", "U158_90":
			for _, old := range source.Counts {
				if old.Name == count.Name {
					count.Count = old.Count
				}
			}
		}
	}
	if binding.Returned {
		for _, name := range []string{"Stage", "U5C", "U64", "U44", "U40"} {
			mustSetValue(current, name, 0)
		}
		mustSetRefs(current, "U68", []uint16{0})
	}
}

func mergeCityPlayerResidue(current, source *sav.DocumentRecordData, binding currentCityPlayerGraft) {
	for i := range current.Values {
		v := &current.Values[i]
		switch v.Name {
		case "This", "Hero", "Money", "Slot", "SlotAgain", "Participant", "F2C":
			continue
		case "F58":
			if binding.CurrentAutoheal {
				continue
			}
		}
		for _, old := range source.Values {
			if old.Name == v.Name {
				v.Value = old.Value
			}
		}
	}
	for i := range current.Raw {
		r := &current.Raw[i]
		for _, old := range source.Raw {
			if old.Name != r.Name {
				continue
			}
			if r.Name == "PRaw32" && binding.CurrentFormation {
				mode := r.Bytes[31]
				r.Bytes = slices.Clone(old.Bytes)
				r.Bytes[31] = mode
			} else {
				r.Bytes = slices.Clone(old.Bytes)
			}
		}
	}
	if !binding.CurrentDiary {
		for i := range current.Inline {
			if current.Inline[i].Name != "Diary" {
				continue
			}
			for _, old := range source.Inline {
				if old.Name == "Diary" {
					current.Inline[i].Record = old.Record
				}
			}
		}
	}
}

// Translate only fields whose programme declares a reference. Other retained
// dwords participate in allocation avoidance, never in reference repair.
func translateCityGraftKeys(r *sav.DocumentRecordData, keys map[uint32]uint32) {
	translate := func(old uint32) uint32 {
		if next, ok := keys[old]; ok {
			return next
		}
		return old
	}
	for i := range r.Values {
		v := &r.Values[i]
		switch v.Name {
		case "Reference", "Hero", "D2C", "G40", "G44", "PE44":
			v.Value = translate(v.Value)
		case "U5C", "U64", "U44", "U40":
			if cityGraftActorClass(r.Class) {
				v.Value = translate(v.Value)
			}
		}
	}
	for i := range r.Raw {
		raw := &r.Raw[i]
		var offsets []int
		switch raw.Name {
		case "Block12":
			if len(raw.Bytes) == 12 {
				offsets = []int{8}
			}
		case "U158":
			if len(raw.Bytes) == 148 {
				offsets = []int{0x0c, 0x10, 0x18, 0x20, 0x28, 0x30, 0x68}
			}
		}
		for _, at := range offsets {
			binary.LittleEndian.PutUint32(raw.Bytes[at:], translate(binary.LittleEndian.Uint32(raw.Bytes[at:])))
		}
	}
	for i := range r.Inline {
		translateCityGraftKeys(&r.Inline[i].Record, keys)
	}
	for i := range r.Groups {
		translateCityGraftKeys(&r.Groups[i], keys)
	}
}
