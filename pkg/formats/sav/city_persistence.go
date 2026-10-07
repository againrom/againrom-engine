package sav

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"sort"
)

const CityDataVersion = 2

const (
	maxCityDataObjects  = 4096
	maxCityDataElements = 1 << 16
	maxCityDataBytes    = 4 << 20
	maxCityDataDepth    = 64
)

// SourceParty decodes the existing character projection from rebuilt semantic
// fields. The temporary body is neither retained nor used as export input.
// This lets native LOAD re-run the ordinary import policy instead of trusting
// a persisted eligibility baseline. Object identities are not reminted here.
func (p *CityProvenance) SourceParty() ([]Character, error) {
	if p == nil || p.document == nil {
		return nil, fmt.Errorf("sav: missing city document")
	}
	// Item references may alias a single large stack/effect list many times.
	// Bound the expanded import, not just the compact graph, before Party or
	// the game restorer materializes those aliases into independent loadouts.
	var expanded uint64
	for _, c := range p.roster {
		u := p.document.objects[p.characterSourceIndex[c.Identity]].unit
		refs := append([]*cityObject{u.reference74, u.reference78}, u.equipment...)
		refs = append(refs, u.container...)
		for _, object := range refs {
			if object == nil || object.item == nil {
				continue
			}
			item := object.item
			stack := max(uint64(1), uint64(binary.LittleEndian.Uint16(item.fields[2:4])))
			expanded += stack * (1 + uint64(len(item.effects)))
			if expanded > maxCityDataElements {
				return nil, fmt.Errorf("sav: city source loadout expansion exceeds element budget")
			}
		}
	}
	body, err := serializeCityDocument(p.document)
	if err != nil {
		return nil, err
	}
	f := &File{Version: p.version, Body: body}
	if err := f.index(); err != nil {
		return nil, err
	}
	return f.Party()
}

// Data returns a detached DTO. No live pointers or source archive indices cross
// this seam; table order alone gives each object a durable local identity.
func (p *CityProvenance) Data() CityData {
	d := p.document
	out := CityData{Version: CityDataVersion, FileVersion: p.version,
		Counter04: d.counter04, Counter00: d.counter00, MapName: d.mapName,
		Head: d.head, PlayerList: d.playerList, Marker: d.marker,
		GlobalDWord: d.globalDWord, TrailerState: cityDataCopy(d.trailerState)}
	indices := make([]int, 0, len(d.objects))
	for index := range d.objects {
		indices = append(indices, int(index))
	}
	sort.Ints(indices)
	ids := make(map[*cityObject]uint16, len(indices))
	for i, index := range indices {
		ids[d.objects[uint16(index)]] = uint16(i + 1)
	}
	ref := func(o *cityObject) uint16 { return ids[o] }
	out.Players, out.DeadActors = cityDataSlice(d.players, ref), cityDataSlice(d.deadActors, ref)
	for _, index := range indices {
		o := d.objects[uint16(index)]
		x := CityObjectData{Class: o.class}
		if o.player != nil {
			v := cityToDataPlayer(*o.player, ref)
			x.Player = &v
		}
		if o.unit != nil {
			v := cityToDataUnit(*o.unit, ref)
			x.Unit = &v
		}
		if o.item != nil {
			v := cityToDataItem(*o.item, ref)
			x.Item = &v
		}
		if o.effect != nil {
			v := cityToDataEffect(*o.effect, ref)
			x.Effect = &v
		}
		if o.spell != nil {
			v := cityToDataSpell(*o.spell, ref)
			x.Spell = &v
		}
		if o.diary != nil {
			v := cityToDataDiary(*o.diary, ref)
			x.Diary = &v
		}
		out.Objects = append(out.Objects, x)
	}
	out.State = CityStateData{RootKind: p.state.rootKind}
	for k, v := range p.state.directoryKinds {
		out.State.DirectoryRecords = append(out.State.DirectoryRecords, CityStateDirectoryData{Path: k, Kind: v})
	}
	for k, v := range p.state.values {
		if v.kind != 2 {
			v.int32 = 0
		} // byte-pool offsets are not semantic state
		out.State.ValueRecords = append(out.State.ValueRecords, CityStateRecordData{Path: k,
			Value: CityStateValueData{v.kind, v.int32, cityDataCopy(v.bytes)}})
	}
	sort.Slice(out.State.DirectoryRecords, func(i, j int) bool { return out.State.DirectoryRecords[i].Path < out.State.DirectoryRecords[j].Path })
	sort.Slice(out.State.ValueRecords, func(i, j int) bool { return out.State.ValueRecords[i].Path < out.State.ValueRecords[j].Path })
	out.Campaign = cityToDataCampaign(*p.campaign, ref)
	return out
}

// CityFromData validates the complete DTO before publishing any provenance.
// It first bounds the acyclic allocation shape, then proves graph closure and
// traversal depth, then uses the ordinary semantic serializers/readers for all
// field widths and programmes. Derived roster caches are rebuilt, not trusted.
func CityFromData(data CityData) (*CityProvenance, error) {
	if data.Version != 1 && data.Version != CityDataVersion {
		return nil, fmt.Errorf("sav: city data version %d, want 1 or %d", data.Version, CityDataVersion)
	}
	if data.FileVersion < MinVersion {
		return nil, fmt.Errorf("sav: city file version %#x is unsupported", data.FileVersion)
	}
	if data.Head[11] != 0 {
		return nil, fmt.Errorf("sav: city data names mission %d", data.Head[11])
	}
	if len(data.Objects) == 0 || len(data.Objects) > maxCityDataObjects {
		return nil, fmt.Errorf("sav: city object count %d outside 1..%d", len(data.Objects), maxCityDataObjects)
	}
	budget := cityDataBudget{}
	if err := budget.check(reflect.ValueOf(data)); err != nil {
		return nil, err
	}
	if err := validateCityDataGraph(data); err != nil {
		return nil, err
	}
	d := &cityDocument{counter04: data.Counter04, counter00: data.Counter00,
		mapName: data.MapName, head: data.Head, playerList: data.PlayerList,
		marker: data.Marker, globalDWord: data.GlobalDWord, trailerState: cityDataCopy(data.TrailerState),
		objects: make(map[uint16]*cityObject, len(data.Objects))}
	for i, x := range data.Objects {
		d.objects[uint16(i+1)] = &cityObject{sourceIndex: uint16(i + 1), class: x.Class}
	}
	ref := func(index uint16) *cityObject { return d.objects[index] }
	for i, x := range data.Objects {
		o := d.objects[uint16(i+1)]
		if x.Player != nil {
			v := cityFromDataPlayer(*x.Player, ref)
			o.player = &v
		}
		if x.Unit != nil {
			v := cityFromDataUnit(*x.Unit, ref)
			o.unit = &v
		}
		if x.Item != nil {
			v := cityFromDataItem(*x.Item, ref)
			o.item = &v
		}
		if x.Effect != nil {
			v := cityFromDataEffect(*x.Effect, ref)
			o.effect = &v
		}
		if x.Spell != nil {
			v := cityFromDataSpell(*x.Spell, ref)
			o.spell = &v
		}
		if x.Diary != nil {
			v := cityFromDataDiary(*x.Diary, ref)
			o.diary = &v
		}
	}
	d.players, d.deadActors = cityDataSlice(data.Players, ref), cityDataSlice(data.DeadActors, ref)
	state, err := cityStateFromData(data.State, data.Version)
	if err != nil {
		return nil, err
	}
	campaign := cityFromDataCampaign(data.Campaign, ref)
	body, err := serializeCityDocument(d)
	if err != nil {
		return nil, err
	}
	if _, err := parseCityDocument(body); err != nil {
		return nil, err
	}
	check := &File{Body: body}
	if err := check.index(); err != nil {
		return nil, err
	}
	if _, err := serializeCityState(state); err != nil {
		return nil, err
	}
	if _, err := serializeCityCampaign(&campaign); err != nil {
		return nil, err
	}
	return newCityProvenance(data.FileVersion, d, state, &campaign)
}

func cityStateFromData(data CityStateData, version uint32) (*cityState, error) {
	state := &cityState{rootKind: data.RootKind, directoryKinds: map[string]uint32{}, values: map[string]cityStateValue{}}
	if version == 1 {
		if len(data.DirectoryRecords) != 0 || len(data.ValueRecords) != 0 {
			return nil, fmt.Errorf("sav: legacy city state carries version 2 records")
		}
		for path, kind := range data.Directories {
			state.directoryKinds[path] = kind
		}
		for path, value := range data.Values {
			state.values[path] = cityStateValue{value.Kind, value.Int32, cityDataCopy(value.Bytes)}
		}
	} else {
		if len(data.Directories) != 0 || len(data.Values) != 0 {
			return nil, fmt.Errorf("sav: city state version 2 carries legacy maps")
		}
		previous := ""
		for _, record := range data.DirectoryRecords {
			if record.Path <= previous {
				return nil, fmt.Errorf("sav: city directory records are not strictly path ordered")
			}
			state.directoryKinds[record.Path], previous = record.Kind, record.Path
		}
		previous = ""
		for _, record := range data.ValueRecords {
			if record.Path <= previous {
				return nil, fmt.Errorf("sav: city value records are not strictly path ordered")
			}
			v := record.Value
			state.values[record.Path], previous = cityStateValue{v.Kind, v.Int32, cityDataCopy(v.Bytes)}, record.Path
		}
	}
	for path, v := range state.values {
		if (v.kind == 2 && len(v.bytes) != 0) || (v.kind != 2 && v.int32 != 0) {
			return nil, fmt.Errorf("sav: city state %s has inactive fields", path)
		}
		if v.kind == 6 && len(v.bytes)%4 != 0 {
			return nil, fmt.Errorf("sav: city state %s has unaligned integer array", path)
		}
	}
	return state, nil
}

// This walk only receives CityData, whose type graph cannot contain pointer
// cycles. Object graph edges are integers and are checked separately below.
type cityDataBudget struct{ elements, bytes uint64 }

func (b *cityDataBudget) check(v reflect.Value) error {
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			return b.check(v.Elem())
		}
	case reflect.String:
		b.bytes += uint64(v.Len())
	case reflect.Slice, reflect.Array:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			b.bytes += uint64(v.Len())
		} else {
			b.elements += uint64(v.Len())
			if b.elements > maxCityDataElements {
				return fmt.Errorf("sav: city data element budget exceeded")
			}
			for i := 0; i < v.Len(); i++ {
				if err := b.check(v.Index(i)); err != nil {
					return err
				}
			}
		}
	case reflect.Map:
		b.elements += uint64(v.Len())
		if b.elements > maxCityDataElements {
			return fmt.Errorf("sav: city data element budget exceeded")
		}
		iter := v.MapRange()
		for iter.Next() {
			if err := b.check(iter.Key()); err != nil {
				return err
			}
			if err := b.check(iter.Value()); err != nil {
				return err
			}
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if err := b.check(v.Field(i)); err != nil {
				return err
			}
		}
	}
	if b.bytes > maxCityDataBytes {
		return fmt.Errorf("sav: city data byte budget exceeded")
	}
	return nil
}

func cityDataReferences(x CityObjectData) []uint16 {
	var refs []uint16
	if p := x.Player; p != nil {
		for _, g := range p.Groups {
			refs = append(refs, g.Actors...)
		}
	}
	if u := x.Unit; u != nil {
		refs = append(refs, u.Effects...)
		refs = append(refs, u.Reference74, u.Reference78, u.Reference68)
		refs = append(refs, u.Container...)
		refs = append(refs, u.Spells...)
		refs = append(refs, u.Equipment...)
	}
	if i := x.Item; i != nil {
		refs = append(refs, i.Effects...)
		refs = append(refs, i.WeaponExtra)
	}
	return refs
}

func validateCityDataGraph(data CityData) error {
	for i, x := range data.Objects {
		bodies := 0
		for _, present := range []bool{x.Player != nil, x.Unit != nil, x.Item != nil, x.Effect != nil, x.Spell != nil, x.Diary != nil} {
			if present {
				bodies++
			}
		}
		valid := false
		switch x.Class {
		case "Player":
			valid = x.Player != nil
		case "Human", "Humanoid", "Unit":
			valid = x.Unit != nil
		case "Item", "Armor", "Shield", "Weapon":
			valid = x.Item != nil
		case "Effect":
			valid = x.Effect != nil
		case "Spell":
			valid = x.Spell != nil
		case "Diary":
			valid = x.Diary != nil
		}
		if bodies != 1 || !valid {
			return fmt.Errorf("sav: city object %d class %q has invalid body union", i+1, x.Class)
		}
		if u := x.Unit; u != nil {
			if u.ContainerFlag == 0 && (len(u.Container) != 0 || u.ContainerTails != [2]uint32{}) {
				return fmt.Errorf("sav: city object %d has absent-container fields", i+1)
			}
			if u.SpellbookFlag == 0 && (u.SpellbookDWord != 0 || u.SpellbookCount != 0 || len(u.Spells) != 0) {
				return fmt.Errorf("sav: city object %d has absent-spellbook fields", i+1)
			}
			if x.Class == "Unit" && (len(u.XP) != 0 || len(u.Equipment) != 0) {
				return fmt.Errorf("sav: city Unit has Humanoid-only fields")
			}
		}
		if x.Item != nil && x.Class != "Weapon" && x.Item.WeaponExtra != 0 {
			return fmt.Errorf("sav: city non-Weapon has Weapon reference")
		}
	}
	seen := make(map[uint16]bool, len(data.Objects))
	var visit func(uint16, int) error
	visit = func(index uint16, depth int) error {
		if index == 0 {
			return nil
		}
		if int(index) > len(data.Objects) {
			return fmt.Errorf("sav: city reference %d exceeds %d objects", index, len(data.Objects))
		}
		if seen[index] {
			return nil
		}
		if depth > maxCityDataDepth {
			return fmt.Errorf("sav: city graph depth exceeds %d", maxCityDataDepth)
		}
		seen[index] = true
		for _, child := range cityDataReferences(data.Objects[index-1]) {
			if err := visit(child, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	for _, index := range append(cityDataCopy(data.Players), data.DeadActors...) {
		if err := visit(index, 1); err != nil {
			return err
		}
	}
	if len(seen) != len(data.Objects) {
		return fmt.Errorf("sav: city has %d unreachable objects", len(data.Objects)-len(seen))
	}
	return nil
}
