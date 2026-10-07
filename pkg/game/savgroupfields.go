package game

import (
	"encoding/binary"
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// These are the existing SAV codec's per-list and local-reference limits,
// not a shipped population. The enclosing document validates aggregate bounds.
const savedGroupFieldListLimit = 1 << 16
const savedGroupFieldObjectLimit = 0x7fff

// projectSavedGroupFields updates one existing inline Group. The caller binds
// members to document-local indices and resolves the two independent keys;
// none is inferred from Group ID, Selector, owner slot or archive index.
// SAV-GRPLOAD-560/GRPOWNER-561/GRPIDENT-562 distinguish those authorities.
// SAV-GRPAI-563 and GRPSAVENEXT-572 replace the AI list pointer on LOAD: its
// last four raw transport bytes remain owned by the imported record, while
// the first76 bytes and both ordered lists come from current saved state.
func projectSavedGroupFields(record *sav.DocumentRecordData, group sim.SavedGroup, members []uint16, referenceKey, ownerKey uint32) error {
	if record == nil || record.Class != "Group" {
		return fmt.Errorf("saved SAV Group fields require an inline Group record")
	}
	if group.Authored {
		return fmt.Errorf("saved SAV Group %d has unsupported native-authored fields", group.ID)
	}
	if len(members) != len(group.Members) || len(members) > savedGroupFieldListLimit || len(group.Words) > savedGroupFieldListLimit || len(group.Path) > savedGroupFieldListLimit {
		return fmt.Errorf("saved SAV Group %d has invalid list counts", group.ID)
	}
	for i, member := range members {
		isNull := !group.Members[i].Bound && group.Members[i].Archive == 0
		if member > savedGroupFieldObjectLimit || (member == 0) != isNull {
			return fmt.Errorf("saved SAV Group %d has invalid member binding at %d", group.ID, i)
		}
	}
	next, sites, err := cloneSavedGroupFieldRecord(record)
	if err != nil {
		return err
	}
	for _, field := range []sav.DocumentValueData{{Name: "G1C", Value: group.Selector}, {Name: "G40", Value: referenceKey}, {Name: "G44", Value: ownerKey}} {
		index, ok := sites.values[field.Name]
		if !ok {
			return fmt.Errorf("saved SAV Group lacks %s", field.Name)
		}
		next.Values[index].Value = field.Value
	}
	ai, err := sites.fixedRaw(&next, "G3C", 80)
	if err != nil {
		return err
	}
	copy(ai[:76], group.AI[:])
	if err := sites.wordList(&next, "G20", group.Words); err != nil {
		return err
	}
	if err := sites.wordList(&next, "G4C", group.Path); err != nil {
		return err
	}
	rawIndex, hasSlots := sites.refs["Actors"]
	countIndex, hasCount := sites.counts["Actors"]
	if !hasSlots || !hasCount || next.Counts[countIndex].Count > savedGroupFieldListLimit || uint64(len(next.RefSlots[rawIndex].Objects)) != uint64(next.Counts[countIndex].Count) {
		return fmt.Errorf("saved SAV Group has inconsistent Actors slots/count")
	}
	for _, member := range next.RefSlots[rawIndex].Objects {
		if member > savedGroupFieldObjectLimit {
			return fmt.Errorf("saved SAV Group has an invalid existing local member index")
		}
	}
	next.RefSlots[rawIndex].Objects = slices.Clone(members)
	next.Counts[countIndex].Count = uint32(len(members))
	*record = next
	return nil
}

// projectSavedOrderFields preserves the imported transport tail separately
// from current order state and the active ordered patrol ring. SAV-GRPPATROL-570
// and PATROLCURSOR-571 distinguish the Group path, actor ring and cell-valued
// cursor. GRPSAVENEXT-572 supplies the current-state writer. Typed native targets
// have no general raw-pointer constructor here, so Authored refuses explicitly.
func projectSavedOrderFields(record *sav.DocumentRecordData, order sim.SavedActorOrder) error {
	if record == nil || (record.Class != "Unit" && record.Class != "Humanoid" && record.Class != "Human") {
		return fmt.Errorf("saved SAV order fields require a Unit-family record")
	}
	if order.Authored {
		return fmt.Errorf("saved SAV actor %d has unsupported native-authored order fields", order.Entity)
	}
	if len(order.Patrol) > savedGroupFieldListLimit {
		return fmt.Errorf("saved SAV actor %d patrol exceeds list bound", order.Entity)
	}
	next, sites, err := cloneSavedGroupFieldRecord(record)
	if err != nil {
		return err
	}
	state, err := sites.fixedRaw(&next, "U50", 4)
	if err != nil {
		return err
	}
	binary.LittleEndian.PutUint32(state, order.State)
	raw, err := sites.fixedRaw(&next, "U158", 148)
	if err != nil {
		return err
	}
	copy(raw[:144], order.Raw[:])
	if err := sites.wordList(&next, "U158_90", order.Patrol); err != nil {
		return err
	}
	*record = next
	return nil
}

type savedGroupFieldSites struct {
	values, raw, counts, refs map[string]int
}

// Only direct record fields participate. Inline records, Group roots, object
// reachability and Player aggregate counts are the enclosing projector's job.
func cloneSavedGroupFieldRecord(record *sav.DocumentRecordData) (sav.DocumentRecordData, savedGroupFieldSites, error) {
	next := *record
	sites := savedGroupFieldSites{}
	var err error
	if sites.values, err = savedGroupFieldNames(record.Values, func(v sav.DocumentValueData) string { return v.Name }); err != nil {
		return next, sites, err
	}
	if sites.raw, err = savedGroupFieldNames(record.Raw, func(v sav.DocumentRawData) string { return v.Name }); err != nil {
		return next, sites, err
	}
	if sites.counts, err = savedGroupFieldNames(record.Counts, func(v sav.DocumentCountData) string { return v.Name }); err != nil {
		return next, sites, err
	}
	if sites.refs, err = savedGroupFieldNames(record.RefSlots, func(v sav.DocumentRefsData) string { return v.Name }); err != nil {
		return next, sites, err
	}
	// Preflight aggregate storage before copying any potentially shared slices.
	// Repeating one large input block must not amplify into an unbounded clone.
	rawBytes, references := 0, 0
	for _, field := range record.Raw {
		if len(field.Bytes) > 64<<20-rawBytes {
			return next, sites, fmt.Errorf("saved SAV record exceeds document byte bound")
		}
		rawBytes += len(field.Bytes)
	}
	for _, field := range record.RefSlots {
		if len(field.Objects) > savedGroupFieldListLimit || len(field.Objects) > 1<<20-references {
			return next, sites, fmt.Errorf("saved SAV reference lists exceed count bound")
		}
		references += len(field.Objects)
	}
	next.Values, next.Raw = slices.Clone(record.Values), slices.Clone(record.Raw)
	next.Counts, next.RefSlots = slices.Clone(record.Counts), slices.Clone(record.RefSlots)
	for i := range next.Raw {
		next.Raw[i].Bytes = slices.Clone(next.Raw[i].Bytes)
	}
	for i := range next.RefSlots {
		next.RefSlots[i].Objects = slices.Clone(next.RefSlots[i].Objects)
	}
	return next, sites, nil
}

func savedGroupFieldNames[T any](fields []T, name func(T) string) (map[string]int, error) {
	if len(fields) > 128 {
		return nil, fmt.Errorf("saved SAV record exceeds field count bound")
	}
	out := make(map[string]int, len(fields))
	previous := ""
	for i, field := range fields {
		key := name(field)
		if key == "" || (i > 0 && key <= previous) {
			return nil, fmt.Errorf("saved SAV fields are not unique and name-sorted at %q", key)
		}
		out[key], previous = i, key
	}
	return out, nil
}

func (sites savedGroupFieldSites) fixedRaw(record *sav.DocumentRecordData, name string, width int) ([]byte, error) {
	index, ok := sites.raw[name]
	if !ok || len(record.Raw[index].Bytes) != width {
		return nil, fmt.Errorf("saved SAV field %s requires %d bytes", name, width)
	}
	return record.Raw[index].Bytes, nil
}

func (sites savedGroupFieldSites) wordList(record *sav.DocumentRecordData, name string, words []uint16) error {
	rawIndex, hasRaw := sites.raw[name]
	countIndex, hasCount := sites.counts[name]
	if !hasRaw || !hasCount || record.Counts[countIndex].Count > savedGroupFieldListLimit || uint64(len(record.Raw[rawIndex].Bytes)) != 2*uint64(record.Counts[countIndex].Count) {
		return fmt.Errorf("saved SAV word list %s has inconsistent bytes/count", name)
	}
	if len(words) > savedGroupFieldListLimit {
		return fmt.Errorf("saved SAV word list %s exceeds list bound", name)
	}
	var raw []byte
	if len(words) != 0 {
		raw = make([]byte, 2*len(words))
	}
	for i, word := range words {
		binary.LittleEndian.PutUint16(raw[2*i:], word)
	}
	record.Raw[rawIndex].Bytes = raw
	record.Counts[countIndex].Count = uint32(len(words))
	return nil
}
