package game

import (
	"fmt"
	"slices"
	"strings"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

type currentCityActorGraft struct {
	Source, Current    uint16
	RetainedHumanTails bool
}

type currentCityPlayerGraft struct {
	Source, Current uint16
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
		actors = append(actors, currentCityActorGraft{Source: from, Current: to, RetainedHumanTails: !currentTails})
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
	var players []currentCityPlayerGraft
	for _, cp := range current.Players {
		if sp := playerSources[cp]; sp != 0 && !ambiguous[cp] {
			players = append(players, currentCityPlayerGraft{Source: sp, Current: cp})
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
// The town document is the party's constructed one; a bound record takes from
// the loaded town file only the unknown-meaning spans savDocumentFallbacks
// names, and the party's own Human tails win over the loaded ones.
func graftCurrentCityResidue(current, retained sav.DocumentData, actors []currentCityActorGraft, players []currentCityPlayerGraft) (sav.DocumentData, error) {
	doc, err := sav.CloneDocumentData(current)
	if err != nil {
		return sav.DocumentData{}, err
	}
	if len(actors) == 0 && len(players) == 0 {
		return doc, nil
	}
	if doc.World != nil {
		return sav.DocumentData{}, fmt.Errorf("city residue requires a city current graph")
	}
	memo, used := map[uint16]bool{}, map[uint16]bool{}
	bind := func(from, to uint16, player bool) error {
		if from == 0 || int(from) > len(retained.Objects) || to == 0 || int(to) > len(doc.Objects) {
			return fmt.Errorf("city residue binding is outside its document")
		}
		if memo[from] || used[to] {
			return fmt.Errorf("city residue binding is ambiguous")
		}
		a, b := &retained.Objects[from-1], &doc.Objects[to-1]
		if player {
			if a.Class != "Player" || b.Class != "Player" {
				return fmt.Errorf("city residue Player binding has another class")
			}
		} else if !cityGraftActorClass(a.Class) || !cityGraftActorClass(b.Class) {
			return fmt.Errorf("city residue actor binding has another class")
		}
		memo[from], used[to] = true, true
		return nil
	}
	spans := unknownRecordSpans()
	for _, b := range actors {
		if err := bind(b.Source, b.Current, false); err != nil {
			return sav.DocumentData{}, err
		}
		mergeCityActorResidue(&doc.Objects[b.Current-1], &retained.Objects[b.Source-1], b, spans)
	}
	for _, b := range players {
		if err := bind(b.Source, b.Current, true); err != nil {
			return sav.DocumentData{}, err
		}
		graftUnknownRecord(&doc.Objects[b.Current-1], &retained.Objects[b.Source-1], spans, nil)
	}
	return doc, nil
}

func cityGraftActorClass(class string) bool {
	return class == "Unit" || class == "Humanoid" || class == "Human"
}

// cityHumanTail names the unknown attack-pair spans a party member's own
// Human tails carry when it has them.
var cityHumanTail = map[string]bool{"U114": true, "UA6": true, "UD4": true}

func mergeCityActorResidue(current, source *sav.DocumentRecordData, binding currentCityActorGraft, spans map[string][]unknownSpan) {
	if !binding.RetainedHumanTails {
		filtered := map[string][]unknownSpan{}
		for pattern, list := range spans {
			field := strings.TrimPrefix(pattern, current.Class+".r.")
			if field != pattern && cityHumanTail[field] {
				continue
			}
			filtered[pattern] = list
		}
		spans = filtered
	}
	graftUnknownRecord(current, source, spans, nil)
}
