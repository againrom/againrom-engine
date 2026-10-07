package sim

import (
	"encoding/binary"
	"fmt"
)

const savedObjectsSpanLen = 4
const maxSavedObjectsBytes = 64 << 20
const savedObjectsWireVersion = 2

// Form84 is the outermost suffix after form83 planes. A zero span means no
// registry and no live handles. Present payload: u32 schema, u8 registry
// presence, u32 item-handle count, u32 Sack-handle count, then sparse 12-byte
// (u32 ordinal, u64 ID) rows for each population. The explicit registry follows:
// u32 Version, u64 NextID, nine u32 counts (Items, Effects, Spells, Sacks,
// SackRoots, Containers, ItemRoots, ScrollReservations, BookRoots), followed by those
// populations in the same order. Item roots are ID and owner tuples; scroll
// reservations bind caster IDs to independent session handles. Schema one
// retains its six counts and exclusive Owner rows only at the decode boundary.
// Fixed structures use declaration order, little-endian scalar widths, and
// one byte per boolean. Coverage is u64 Unknown, u16 byte length, then text.
// ID lists are u32 count followed by u64 IDs. No source key is matched on LOAD.
type savedObjectsSection struct {
	registry                 *SavedObjects
	itemHandles, sackHandles []byte
	reservations             []savedScrollReservation
	legacy                   bool
}

type savedScrollReservation struct {
	Caster EntityID
	Handle uint64
}

// Ordinals are the existing instance-weight walk, including empty worn slots.
// Constructor rows are not live items and have no object identity.
func (w *World) eachSavedObjectHandle(visit func(uint32, uint16, *SavedObjectID)) {
	var ordinal uint32
	item := func(i *ItemInstance) { visit(ordinal, i.Code, &i.ObjectID); ordinal++ }
	for si := range w.sacks {
		for ii := range w.sacks[si].ItemInstances {
			item(&w.sacks[si].ItemInstances[ii])
		}
	}
	for ei := range w.carried {
		for si := range w.carried[ei] {
			s := &w.carried[ei][si]
			visit(ordinal, s.Code, &s.ObjectID)
			ordinal++
		}
	}
	for ei := range w.equipment {
		for si := range w.equipment[ei] {
			item(&w.equipment[ei][si])
		}
	}
	for ci := range w.scrollCasts {
		item(&w.scrollCasts[ci].Item)
	}
}

func (w *World) savedObjectsBinarySize() (uint64, error) {
	n, handles := uint64(13), uint64(0)
	var fault error
	w.eachSavedObjectHandle(func(_ uint32, code uint16, id *SavedObjectID) {
		if *id != 0 {
			handles++
			if code == 0 {
				fault = fmt.Errorf("sim: empty item has saved object handle")
			}
		}
	})
	for _, s := range w.sacks {
		if s.ObjectID != 0 {
			handles++
		}
	}
	if fault != nil || handles > MaxSavedObjects {
		return 0, fmt.Errorf("sim: invalid saved object handle population: %v", fault)
	}
	if w.savedObjects == nil {
		if handles != 0 {
			return 0, fmt.Errorf("sim: saved object handles lack registry")
		}
		return 0, nil
	}
	r := w.savedObjects
	n += handles*12 + 48
	n += uint64(len(r.ItemRoots)) * 33
	n += uint64(len(r.BookRoots)) * 228
	for _, cast := range w.scrollCasts {
		if cast.Item.ObjectID != 0 {
			n += 12
		}
	}
	for _, v := range r.Items {
		n += 189 + uint64(len(v.Value.Effects))*6 + uint64(len(v.Effects))*8 + uint64(len(v.Coverage.Unsupported))
	}
	for _, v := range r.Effects {
		n += 76 + uint64(len(v.Coverage.Unsupported))
	}
	for _, v := range r.Spells {
		n += 42 + uint64(len(v.Coverage.Unsupported))
	}
	for _, v := range r.Sacks {
		n += 69 + uint64(len(v.Coverage.Unsupported))
	}
	n += uint64(len(r.SackRoots)) * 8
	for _, v := range r.Containers {
		n += 48 + uint64(len(v.Items))*8 + uint64(len(v.Coverage.Unsupported))
	}
	if n > maxSavedObjectsBytes {
		return 0, fmt.Errorf("sim: saved object suffix exceeds byte bound")
	}
	return n, nil
}

func appendSavedObjectFixed(dst []byte, value any) []byte {
	out, err := binary.Append(dst, binary.LittleEndian, value)
	if err != nil {
		panic(err) // Only statically declared fixed-width structures reach here.
	}
	return out
}

func appendSavedObjectCoverage(dst []byte, c SavedObjectCoverage) []byte {
	dst = binary.LittleEndian.AppendUint64(dst, uint64(c.Unknown))
	dst = binary.LittleEndian.AppendUint16(dst, uint16(len(c.Unsupported)))
	return append(dst, c.Unsupported...)
}

func appendSavedObjectIDs(dst []byte, ids []SavedObjectID) []byte {
	dst = binary.LittleEndian.AppendUint32(dst, uint32(len(ids)))
	for _, id := range ids {
		dst = binary.LittleEndian.AppendUint64(dst, uint64(id))
	}
	return dst
}

func appendSavedObjectStack(dst []byte, v ItemStack) []byte {
	dst = binary.LittleEndian.AppendUint64(dst, uint64(v.ObjectID))
	dst = binary.LittleEndian.AppendUint32(dst, v.Count)
	dst = appendItemBytes(dst, v.Instance())
	dst = appendSavedObjectFixed(dst, v.WeightPresent)
	dst = binary.LittleEndian.AppendUint16(dst, uint16(v.Weight))
	return appendSavedObjectFixed(dst, v.SourceEquipment)
}

func (w *World) appendSavedObjects(dst []byte) []byte {
	start := len(dst)
	if w.savedObjects == nil {
		hasHandle := false
		w.eachSavedObjectHandle(func(_ uint32, _ uint16, id *SavedObjectID) { hasHandle = hasHandle || *id != 0 })
		for _, s := range w.sacks {
			hasHandle = hasHandle || s.ObjectID != 0
		}
		if !hasHandle {
			return binary.LittleEndian.AppendUint32(dst, 0)
		}
	}
	dst = binary.LittleEndian.AppendUint32(dst, savedObjectsWireVersion)
	dst = appendSavedObjectFixed(dst, w.savedObjects != nil)
	dst = append(dst, make([]byte, 8)...)
	var ni, ns uint32
	w.eachSavedObjectHandle(func(ordinal uint32, _ uint16, id *SavedObjectID) {
		if *id != 0 {
			dst = binary.LittleEndian.AppendUint32(dst, ordinal)
			dst = binary.LittleEndian.AppendUint64(dst, uint64(*id))
			ni++
		}
	})
	for ordinal, s := range w.sacks {
		if s.ObjectID != 0 {
			dst = binary.LittleEndian.AppendUint32(dst, uint32(ordinal))
			dst = binary.LittleEndian.AppendUint64(dst, uint64(s.ObjectID))
			ns++
		}
	}
	binary.LittleEndian.PutUint32(dst[start+5:], ni)
	binary.LittleEndian.PutUint32(dst[start+9:], ns)
	r := w.savedObjects
	if r == nil {
		// Marshal refuses these orphan handles, but Hash must still see the
		// malformed live state rather than silently treating it as absent.
		return binary.LittleEndian.AppendUint32(dst, uint32(len(dst)-start))
	}
	dst = binary.LittleEndian.AppendUint32(dst, r.Version)
	dst = binary.LittleEndian.AppendUint64(dst, uint64(r.NextID))
	reservations := 0
	for _, cast := range w.scrollCasts {
		if cast.Item.ObjectID != 0 {
			reservations++
		}
	}
	for _, n := range []int{len(r.Items), len(r.Effects), len(r.Spells), len(r.Sacks), len(r.SackRoots), len(r.Containers), len(r.ItemRoots), reservations, len(r.BookRoots)} {
		dst = binary.LittleEndian.AppendUint32(dst, uint32(n))
	}
	for _, v := range r.Items {
		dst = binary.LittleEndian.AppendUint64(dst, uint64(v.ID))
		dst = appendSavedObjectFixed(dst, v.Origin)
		dst = appendSavedObjectFixed(dst, v.Retired)
		dst = binary.LittleEndian.AppendUint32(dst, v.InFlight)
		dst = appendSavedObjectStack(dst, v.Value)
		dst = appendSavedObjectFixed(dst, v.Token)
		dst = append(dst, v.F45, v.F46, v.F47)
		dst = binary.LittleEndian.AppendUint16(dst, v.F48)
		dst = appendSavedObjectIDs(dst, v.Effects)
		dst = binary.LittleEndian.AppendUint64(dst, uint64(v.Spell))
		dst = appendSavedObjectCoverage(dst, v.Coverage)
	}
	for _, v := range r.Effects {
		dst = binary.LittleEndian.AppendUint64(dst, uint64(v.ID))
		dst = appendSavedObjectFixed(dst, v.Origin)
		dst = appendSavedObjectFixed(dst, v.Token)
		dst = appendSavedObjectFixed(dst, v.Value)
		dst = append(dst, v.E0C)
		dst = binary.LittleEndian.AppendUint32(dst, v.ExternalReferences)
		dst = appendSavedObjectFixed(dst, v.Retired)
		dst = appendSavedObjectCoverage(dst, v.Coverage)
	}
	for _, v := range r.Spells {
		dst = binary.LittleEndian.AppendUint64(dst, uint64(v.ID))
		dst = appendSavedObjectFixed(dst, v.Origin)
		dst = appendSavedObjectFixed(dst, v.Value)
		dst = binary.LittleEndian.AppendUint32(dst, v.This)
		dst = binary.LittleEndian.AppendUint32(dst, v.ExternalReferences)
		dst = appendSavedObjectFixed(dst, v.Retired)
		dst = appendSavedObjectCoverage(dst, v.Coverage)
	}
	for _, v := range r.Sacks {
		dst = binary.LittleEndian.AppendUint64(dst, uint64(v.ID))
		dst = appendSavedObjectFixed(dst, v.Origin)
		dst = appendSavedObjectFixed(dst, v.Token)
		dst = binary.LittleEndian.AppendUint32(dst, v.Gold)
		dst = appendSavedObjectFixed(dst, v.Retired)
		dst = appendSavedObjectCoverage(dst, v.Coverage)
	}
	for _, id := range r.SackRoots {
		dst = binary.LittleEndian.AppendUint64(dst, uint64(id))
	}
	for _, v := range r.Containers {
		dst = appendSavedObjectFixed(dst, v.Owner)
		dst = appendSavedObjectFixed(dst, v.Present)
		dst = binary.LittleEndian.AppendUint32(dst, v.InsertIndex)
		dst = binary.LittleEndian.AppendUint32(dst, uint32(v.Accumulator))
		dst = appendSavedObjectIDs(dst, v.Items)
		dst = appendSavedObjectCoverage(dst, v.Coverage)
	}
	for _, root := range r.ItemRoots {
		dst = binary.LittleEndian.AppendUint64(dst, uint64(root.ID))
		dst = appendSavedObjectFixed(dst, root.Owner)
	}
	for _, cast := range w.scrollCasts {
		if cast.Item.ObjectID != 0 {
			dst = binary.LittleEndian.AppendUint32(dst, uint32(cast.Caster))
			dst = binary.LittleEndian.AppendUint64(dst, cast.Reservation)
		}
	}
	for _, root := range r.BookRoots {
		dst = appendSavedObjectFixed(dst, root)
	}

	return binary.LittleEndian.AppendUint32(dst, uint32(len(dst)-start))
}

type savedObjectReader struct {
	data []byte
	err  error
}

func (r *savedObjectReader) take(n uint64) []byte {
	if r.err != nil {
		return nil
	}
	if n > uint64(len(r.data)) {
		r.err = fmt.Errorf("sim: truncated saved object suffix")
		return nil
	}
	out := r.data[:n]
	r.data = r.data[n:]
	return out
}

func (r *savedObjectReader) fixed(dst any) {
	n := binary.Size(dst)
	if n < 0 {
		r.err = fmt.Errorf("sim: invalid fixed saved object type")
		return
	}
	if b := r.take(uint64(n)); b != nil {
		_, r.err = binary.Decode(b, binary.LittleEndian, dst)
	}
}

func (r *savedObjectReader) u8() (v uint8)     { r.fixed(&v); return }
func (r *savedObjectReader) u16() (v uint16)   { r.fixed(&v); return }
func (r *savedObjectReader) u32() (v uint32)   { r.fixed(&v); return }
func (r *savedObjectReader) u64() (v uint64)   { r.fixed(&v); return }
func (r *savedObjectReader) id() SavedObjectID { return SavedObjectID(r.u64()) }

func (r *savedObjectReader) boolean() bool {
	v := r.u8()
	if v > 1 {
		r.err = fmt.Errorf("sim: noncanonical saved object boolean")
	}
	return v == 1
}

func (r *savedObjectReader) coverage() SavedObjectCoverage {
	c := SavedObjectCoverage{Unknown: SavedObjectUnknown(r.u64())}
	n := r.u16()
	if n > 512 {
		r.err = fmt.Errorf("sim: saved object coverage exceeds text bound")
		return c
	}
	c.Unsupported = string(r.take(uint64(n)))
	return c
}

func (r *savedObjectReader) ids() []SavedObjectID {
	n := uint64(r.u32())
	if r.err != nil || n > MaxSavedObjects || n*8 > uint64(len(r.data)) {
		r.err = fmt.Errorf("sim: saved object edge count exceeds payload")
		return nil
	}
	if n == 0 {
		return nil
	}
	out := make([]SavedObjectID, int(n))
	for i := range out {
		out[i] = r.id()
	}
	return out
}

func (r *savedObjectReader) stack() ItemStack {
	id, count := r.id(), r.u32()
	if r.err != nil || len(r.data) < itemHeadLen || binary.LittleEndian.Uint32(r.data[7:]) > MaxSavedObjects {
		r.err = fmt.Errorf("sim: saved object item head/effect count is invalid")
		return ItemStack{}
	}
	item, used, err := decodeItemBytes(r.data, "saved object item")
	if err != nil {
		r.err = err
		return ItemStack{}
	}
	r.take(uint64(used))
	item.ObjectID, item.WeightPresent, item.Weight = id, r.boolean(), int16(r.u16())
	if b := r.take(sourceEquipmentLen); b != nil {
		if b[49] > 1 || b[70] > 1 || b[76] > 1 {
			r.err = fmt.Errorf("sim: noncanonical saved object equipment boolean")
		} else {
			_, r.err = binary.Decode(b, binary.LittleEndian, &item.SourceEquipment)
		}
	}
	return StackItem(item, count)
}

func splitSavedObjects(data []byte) ([]byte, *savedObjectsSection, error) {
	if len(data) < headerLen+savedObjectsSpanLen {
		return nil, nil, fmt.Errorf("sim: truncated saved object footer")
	}
	end := len(data) - savedObjectsSpanLen
	n := uint64(binary.LittleEndian.Uint32(data[end:]))
	if n == 0 {
		return data[:end], nil, nil
	}
	if n > maxSavedObjectsBytes || n > uint64(end-headerLen) || n < 49 {
		return nil, nil, fmt.Errorf("sim: invalid saved object suffix span")
	}
	start := end - int(n)
	r := savedObjectReader{data: data[start:end]}
	schema := r.u32()
	if schema != 1 && schema != savedObjectsWireVersion || r.u8() != 1 {
		return nil, nil, fmt.Errorf("sim: invalid saved object schema/presence")
	}
	ni, ns := uint64(r.u32()), uint64(r.u32())
	if ni+ns > MaxSavedObjects || (ni+ns)*12+36 > uint64(len(r.data)) {
		return nil, nil, fmt.Errorf("sim: saved object handle counts exceed payload")
	}
	section := &savedObjectsSection{itemHandles: r.take(ni * 12), sackHandles: r.take(ns * 12), legacy: schema == 1}
	for _, rows := range [][]byte{section.itemHandles, section.sackHandles} {
		for off := 0; off < len(rows); off += 12 {
			if binary.LittleEndian.Uint64(rows[off+4:]) == 0 || off != 0 && binary.LittleEndian.Uint32(rows[off:]) <= binary.LittleEndian.Uint32(rows[off-12:]) {
				return nil, nil, fmt.Errorf("sim: noncanonical saved object handle rows")
			}
		}
	}
	g := &SavedObjects{Version: r.u32(), NextID: r.id()}
	var counts [9]uint32
	if schema == 1 {
		var old [6]uint32
		r.fixed(&old)
		copy(counts[:], old[:])
	} else {
		r.fixed(&counts)
	}
	minimum := [...]uint64{189, 76, 42, 69, 8, 48, 33, 12, 228}
	if schema == 1 {
		minimum[0] = 209
	}
	var total, bytes uint64
	for i, count := range counts {
		total += uint64(count)
		bytes += uint64(count) * minimum[i]
	}
	if r.err != nil || g.Version != schema || g.NextID == 0 || total > MaxSavedObjects || bytes > uint64(len(r.data)) {
		return nil, nil, fmt.Errorf("sim: saved object registry header/counts exceed payload")
	}
	for i := uint32(0); i < counts[0] && r.err == nil; i++ {
		v := SavedItemObject{ID: r.id()}
		r.fixed(&v.Origin)
		if schema == 1 {
			r.fixed(&v.Owner)
		} else {
			v.Retired, v.InFlight = r.boolean(), r.u32()
		}
		v.Value = r.stack()
		r.fixed(&v.Token)
		v.F45, v.F46, v.F47, v.F48 = r.u8(), r.u8(), r.u8(), r.u16()
		v.Effects, v.Spell, v.Coverage = r.ids(), r.id(), r.coverage()
		g.Items = append(g.Items, v)
	}
	for i := uint32(0); i < counts[1] && r.err == nil; i++ {
		v := SavedEffectObject{ID: r.id()}
		r.fixed(&v.Origin)
		r.fixed(&v.Token)
		r.fixed(&v.Value)
		v.E0C, v.ExternalReferences, v.Retired, v.Coverage = r.u8(), r.u32(), r.boolean(), r.coverage()
		g.Effects = append(g.Effects, v)
	}
	for i := uint32(0); i < counts[2] && r.err == nil; i++ {
		v := SavedSpellObject{ID: r.id()}
		r.fixed(&v.Origin)
		v.Value.Present = r.boolean()
		v.Value.ID, v.Value.Range, v.Value.Defensive, v.Value.ManaCost = r.u8(), r.u8(), r.u8(), r.u16()
		v.This, v.ExternalReferences, v.Retired, v.Coverage = r.u32(), r.u32(), r.boolean(), r.coverage()
		g.Spells = append(g.Spells, v)
	}
	for i := uint32(0); i < counts[3] && r.err == nil; i++ {
		v := SavedSackObject{ID: r.id()}
		r.fixed(&v.Origin)
		r.fixed(&v.Token)
		v.Gold, v.Retired, v.Coverage = r.u32(), r.boolean(), r.coverage()
		g.Sacks = append(g.Sacks, v)
	}
	for i := uint32(0); i < counts[4] && r.err == nil; i++ {
		g.SackRoots = append(g.SackRoots, r.id())
	}
	for i := uint32(0); i < counts[5] && r.err == nil; i++ {
		var v SavedObjectContainer
		r.fixed(&v.Owner)
		v.Present, v.InsertIndex, v.Accumulator = r.boolean(), r.u32(), int32(r.u32())
		v.Items, v.Coverage = r.ids(), r.coverage()
		g.Containers = append(g.Containers, v)
	}
	for i := uint32(0); i < counts[6] && r.err == nil; i++ {
		root := SavedItemRoot{ID: r.id()}
		r.fixed(&root.Owner)
		g.ItemRoots = append(g.ItemRoots, root)
	}
	for i := uint32(0); i < counts[7] && r.err == nil; i++ {
		v := savedScrollReservation{EntityID(r.u32()), uint64(r.id())}
		if v.Handle == 0 || i != 0 && section.reservations[i-1].Caster >= v.Caster {
			return nil, nil, fmt.Errorf("sim: invalid scroll reservation mapping")
		}
		section.reservations = append(section.reservations, v)
	}
	for i := uint32(0); i < counts[8] && r.err == nil; i++ {
		var root SavedBookRoot
		r.fixed(&root)
		g.BookRoots = append(g.BookRoots, root)
	}

	if r.err != nil {
		return nil, nil, r.err
	}
	if len(r.data) != 0 {
		return nil, nil, fmt.Errorf("sim: extra saved object payload bytes")
	}
	if schema == 1 {
		if err := g.MigrateLegacyOwners(); err != nil {
			return nil, nil, err
		}
	}
	if err := g.ValidateNoInFlight(); err != nil {
		return nil, nil, err
	}
	section.registry = g
	return data[:start], section, nil
}

// Attach only to the exact decoded ordinal before stack canonicality checks.
// This runs on the private candidate; a late failure never changes the receiver.
func (w *World) applySavedObjects(section *savedObjectsSection) error {
	if section == nil {
		return nil
	}
	off := 0
	var fault error
	w.eachSavedObjectHandle(func(ordinal uint32, code uint16, id *SavedObjectID) {
		rows := section.itemHandles
		if off < len(rows) && binary.LittleEndian.Uint32(rows[off:]) == ordinal {
			if code == 0 {
				fault = fmt.Errorf("sim: saved object handle names empty item")
			}
			*id = SavedObjectID(binary.LittleEndian.Uint64(rows[off+4:]))
			off += 12
		}
	})
	if fault != nil || off != len(section.itemHandles) {
		return fmt.Errorf("sim: saved object item handle is outside live population: %v", fault)
	}
	for off := 0; off < len(section.sackHandles); off += 12 {
		ordinal := binary.LittleEndian.Uint32(section.sackHandles[off:])
		if uint64(ordinal) >= uint64(len(w.sacks)) {
			return fmt.Errorf("sim: saved object Sack handle is outside live population")
		}
		w.sacks[ordinal].ObjectID = SavedObjectID(binary.LittleEndian.Uint64(section.sackHandles[off+4:]))
	}
	w.savedObjects = section.registry
	at := 0
	for i := range w.scrollCasts {
		cast := &w.scrollCasts[i]
		if cast.Item.ObjectID == 0 {
			continue
		}
		if section.legacy {
			cast.Reservation = w.savedObjects.sessionHandle(cast.Item.ObjectID)
		} else {
			if at >= len(section.reservations) || section.reservations[at].Caster != cast.Caster {
				return fmt.Errorf("sim: scroll reservation lacks exact caster")
			}
			cast.Reservation = section.reservations[at].Handle
			at++
		}
		if !w.savedObjects.HasRoot(cast.Item.ObjectID, SavedObjectOwner{Kind: SavedOwnerSession, SessionHandle: cast.Reservation}) {
			return fmt.Errorf("sim: invalid scroll reservation root")
		}
	}
	if at != len(section.reservations) {
		return fmt.Errorf("sim: extra scroll reservation mapping")
	}
	return nil
}
