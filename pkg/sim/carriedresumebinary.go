package sim

import (
	"encoding/binary"
	"fmt"
)

// Form85 is the outermost suffix after form84's saved-object footer. It
// gives every field world.go's own struct comments name
// carried-not-wire-form a wire position: rawSessionHead/rawSessionMid,
// savedCellRecords (DIV-932), savedSpellEffects (DIV-938/DIV-939),
// savedProjectiles (DIV-944) and savedDiaries (DIV-956). Before this story
// each was carried ACROSS a decode from the receiver's own prior value
// rather than read from the byte form, so a decode into a fresh receiver —
// the read-only save instruments, and every genuine cross-process native
// `.ags` resume — silently lost all five; a same-receiver decode
// (resumeWorld, pkg/game/resume.go) kept them only because nothing else
// touched the receiver in between, and only because StartMissionFrom never
// populates them in the first place.
//
// A zero span means every one of the six fields below is absent/empty: the
// two spans are all-zero, and the three lists and the projectile store are
// empty. carriedResumeStateEmpty is what MarshalBinary asks before writing
// the fuller form, so the overwhelmingly common case — a mission never
// resumed from an original save — costs four bytes, not this section's own
// minimum 470.
//
// Present payload, in this fixed order:
//
//	RawSessionHead   [sessionRawHeadLen]byte (48, raw)
//	RawSessionMid    [sessionRawMidLen]byte (400, raw)
//	CellRecords      u32 count, then that many 52-byte records, strictly
//	                 ascending by Cell with no duplicate key
//	SpellEffects     u32 count, then that many spell-effect nodes (below)
//	Projectiles      u16 FreeIndex, u32 IDs count + that many u16 ids, u32
//	                 Items count + that many 66-byte records; no two Items
//	                 share an ID, IDs may repeat or name an id Items lacks
//	Diaries          u32 count, then that many records: u8 Player (0/1), u32
//	                 Actor (zero when Player), u32 Length, u32 Entries count
//	                 + that many (u32 Index, u32 Count, u16 Remaining)
//	                 triples, strictly ascending by Index and 0<=Index<Length;
//	                 at most one Player owner and no repeated Actor owner
//
// A cell record's 52 bytes are Cell (u16), LayerCount (u8), Residue0 (u8),
// Residue1 (2 raw bytes), Ground and Air (u32 Key, u32 Entity, u8 Bound each),
// Sack (u32) and six u32 SpellEffects keys, on SavedCellRecord's own field
// order (savedcellrecord.go).
//
// A spell-effect node is 15 fixed bytes — class tag (u8: 0 SpellEffect, 1
// PointEffect, 2 AreaEffect, 3 SpellTransport), SE40, SE41 (u8 each), PE44
// (u32), AE48 (4 raw bytes), AE4C (u16), ST4C (u16) — followed by four
// pointer slots in PE48, AE44, ST44, ST48 order, each a presence byte and,
// if present, PE48/AE44's fixed 33-byte Effect body (class tag, E3C, E3D
// (u8 each), E40 (u32), E0C (u8), DirectDamage (24 raw bytes)) or ST44/ST48's
// own nested spell-effect node. Every field the node's own Class does not
// select is written and read as zero; a decoded node carrying nonzero
// off-class state is refused (savedSpellEffectShapeFault), the same rule
// every other section in this form applies to a byte the encoder could not
// have written.
//
// FIELDS THE STRUCT DOES NOT SELECT BY CLASS ARE STILL ALWAYS PRESENT ON THE
// WIRE, at the same offset regardless of Class: the alternative, a shape that
// varies by tag, would make two decoders of the same bytes disagree about
// where the next field starts unless both first branch on Class identically,
// and this form already refuses the one input that would matter (an
// off-class field set nonzero) at both write and read.
//
// SHARED-OBJECT IDENTITY INSIDE THE SPELL-EFFECT GRAPH IS NOT RECONSTRUCTED.
// DIV-939 review F-1 finds the archive's own back-references already lost by
// the time this project's own converter runs — spellEffectFromRecord and
// effectFromRecord (pkg/formats/sav/spelleffect.go) allocate a fresh Go value
// at every reference site, so two graph positions naming one archive object
// already carry two distinct pointers before spellEffectConverter's own
// pointer-keyed cache (pkg/game/originalspelleffects.go) ever runs, and that
// cache never hits. PE44 is DIV-938's own opaque archive-identity dword,
// carried as a plain uint32 with no lookup attempted. There is therefore no
// live sharing this form could either preserve or invent: PE48, AE44, ST44
// and ST48 are each written and read inline, independently, at the exact
// position their own pointer occupies, on whatever graph is already in
// memory — which is a tree wherever the file was, and a tree everywhere this
// project's own converter is the only writer that has ever populated one.
//
// RECURSION IS BOUNDED. ST44/ST48 nest a SpellTransport's own two typed
// references, and savedSpellEffectMaxDepth (16) mirrors
// pkg/formats/sav/spelleffect.go's spellEffectMaxDepth (itself maxWalkDepth):
// the same bound the archive reader already applies to the same shape, cited
// rather than re-derived, applied again here because this form's own decoder
// reads bytes the archive reader never touches — a native `.ags` file this
// project wrote, not the SAV file the graph was first read from.
//
// A zero span means every field below is absent or empty.
const carriedResumeSpanLen = 4

// maxCarriedResumeBytes bounds a hostile or corrupt span before it is
// trusted, on savedObjectsbinary.go's maxSavedObjectsBytes' own precedent.
const maxCarriedResumeBytes = 64 << 20

// minCarriedResumePayload is the smallest a present (nonzero-span) payload
// can be: both raw spans plus every count field at zero.
const minCarriedResumePayload = sessionRawHeadLen + sessionRawMidLen + 4 + 4 + 2 + 4 + 4 + 4

// carriedCellRecordLen is one SavedCellRecord's fixed wire width.
const carriedCellRecordLen = 52

// carriedProjectileRecordLen is one SavedProjectile's fixed wire width: a u16
// ID followed by its sixteen int32 leaves.
const carriedProjectileRecordLen = 2 + 16*4

// carriedDiaryMinLen is one SavedDiary's fixed prefix before its entries.
const carriedDiaryMinLen = 1 + 4 + 4 + 4

// carriedSpellEffectMinLen is the smallest one spell-effect node can be: the
// 15-byte fixed prefix plus four absent (one-byte) pointer slots.
const carriedSpellEffectMinLen = 15 + 4

// savedSpellEffectMaxDepth bounds ST44/ST48 recursion, mirroring
// pkg/formats/sav/spelleffect.go's spellEffectMaxDepth.
const savedSpellEffectMaxDepth = 16

// carriedResumeSection is what splitCarriedResumeState hands the decoder:
// always non-nil, with every field at its natural Go zero value when the
// span was absent.
type carriedResumeSection struct {
	head         [sessionRawHeadLen]byte
	mid          [sessionRawMidLen]byte
	cellRecords  []SavedCellRecord
	spellEffects []SavedSpellEffect
	projectiles  SavedProjectiles
	diaries      []SavedDiary
}

func boolFlag(b bool) byte {
	if b {
		return 1
	}
	return 0
}

// carriedResumeStateEmpty reports whether every field this section carries is
// at its absent/empty default, the case appendCarriedResumeState collapses to
// a zero span.
func carriedResumeStateEmpty(w *World) bool {
	return w.rawSessionHead == [sessionRawHeadLen]byte{} &&
		w.rawSessionMid == [sessionRawMidLen]byte{} &&
		len(w.savedCellRecords) == 0 &&
		len(w.savedSpellEffects) == 0 &&
		w.savedProjectiles.FreeIndex == 0 && len(w.savedProjectiles.IDs) == 0 && len(w.savedProjectiles.Items) == 0 &&
		len(w.savedDiaries) == 0
}

func (w *World) appendCarriedResumeState(b []byte) []byte {
	start := len(b)
	if carriedResumeStateEmpty(w) {
		return binary.LittleEndian.AppendUint32(b, 0)
	}
	b = append(b, w.rawSessionHead[:]...)
	b = append(b, w.rawSessionMid[:]...)
	b = appendCarriedCellRecords(b, w.savedCellRecords)
	b = appendCarriedSpellEffects(b, w.savedSpellEffects)
	b = appendCarriedProjectiles(b, w.savedProjectiles)
	b = appendCarriedDiaries(b, w.savedDiaries)
	return binary.LittleEndian.AppendUint32(b, uint32(len(b)-start))
}

func splitCarriedResumeState(data []byte) ([]byte, *carriedResumeSection, error) {
	if len(data) < headerLen+carriedResumeSpanLen {
		return nil, nil, fmt.Errorf("sim: truncated carried resume-state footer")
	}
	end := len(data) - carriedResumeSpanLen
	n := uint64(binary.LittleEndian.Uint32(data[end:]))
	if n == 0 {
		return data[:end], &carriedResumeSection{}, nil
	}
	if n > maxCarriedResumeBytes || n > uint64(end-headerLen) || n < minCarriedResumePayload {
		return nil, nil, fmt.Errorf("sim: invalid carried resume-state span")
	}
	start := end - int(n)
	r := &savedObjectReader{data: data[start:end]}
	var s carriedResumeSection
	if b := r.take(sessionRawHeadLen); b != nil {
		copy(s.head[:], b)
	}
	if b := r.take(sessionRawMidLen); b != nil {
		copy(s.mid[:], b)
	}
	s.cellRecords = readCarriedCellRecords(r)
	s.spellEffects = readCarriedSpellEffects(r)
	s.projectiles = readCarriedProjectiles(r)
	s.diaries = readCarriedDiaries(r)
	if r.err != nil {
		return nil, nil, r.err
	}
	if len(r.data) != 0 {
		return nil, nil, fmt.Errorf("sim: extra carried resume-state payload bytes")
	}
	return data[:start], &s, nil
}

// --- cell records --------------------------------------------------------

func appendCarriedCellRecords(dst []byte, records []SavedCellRecord) []byte {
	dst = binary.LittleEndian.AppendUint32(dst, uint32(len(records)))
	for _, c := range records {
		dst = binary.LittleEndian.AppendUint16(dst, c.Cell)
		dst = append(dst, c.LayerCount, c.Residue0, c.Residue1[0], c.Residue1[1])
		dst = binary.LittleEndian.AppendUint32(dst, c.Ground.Key)
		dst = binary.LittleEndian.AppendUint32(dst, uint32(c.Ground.Entity))
		dst = append(dst, boolFlag(c.Ground.Bound))
		dst = binary.LittleEndian.AppendUint32(dst, c.Air.Key)
		dst = binary.LittleEndian.AppendUint32(dst, uint32(c.Air.Entity))
		dst = append(dst, boolFlag(c.Air.Bound))
		dst = binary.LittleEndian.AppendUint32(dst, c.Sack)
		for _, se := range c.SpellEffects {
			dst = binary.LittleEndian.AppendUint32(dst, se)
		}
	}
	return dst
}

func readCarriedCellRecords(r *savedObjectReader) []SavedCellRecord {
	n := r.u32()
	if r.err != nil {
		return nil
	}
	if uint64(n)*carriedCellRecordLen > uint64(len(r.data)) {
		r.err = fmt.Errorf("sim: carried cell record count exceeds payload")
		return nil
	}
	if n == 0 {
		return nil
	}
	out := make([]SavedCellRecord, n)
	for i := range out {
		c := &out[i]
		c.Cell = r.u16()
		c.LayerCount, c.Residue0 = r.u8(), r.u8()
		if b := r.take(2); b != nil {
			c.Residue1[0], c.Residue1[1] = b[0], b[1]
		}
		c.Ground.Key = r.u32()
		c.Ground.Entity = EntityID(r.u32())
		c.Ground.Bound = r.boolean()
		c.Air.Key = r.u32()
		c.Air.Entity = EntityID(r.u32())
		c.Air.Bound = r.boolean()
		c.Sack = r.u32()
		for k := range c.SpellEffects {
			c.SpellEffects[k] = r.u32()
		}
		if r.err != nil {
			return nil
		}
		if i > 0 && out[i-1].Cell >= c.Cell {
			r.err = fmt.Errorf("sim: carried cell records are not strictly ascending")
			return nil
		}
	}
	return out
}

func savedCellRecordOrderFault(records []SavedCellRecord) error {
	for i := 1; i < len(records); i++ {
		if records[i-1].Cell >= records[i].Cell {
			return fmt.Errorf("sim: carried cell records are not strictly ascending")
		}
	}
	return nil
}

// --- spell-effect graph ---------------------------------------------------

func appendSavedEffectPtr(dst []byte, e *SavedEffect) []byte {
	if e == nil {
		return append(dst, 0)
	}
	dst = append(dst, 1)
	var tag byte
	switch e.Class {
	case "Effect":
		tag = 0
	case "Effect_DirectDamage":
		tag = 1
	default:
		panic("sim: invalid SavedEffect class") // carriedResumeStateFault already refused this.
	}
	dst = append(dst, tag, e.E3C, e.E3D)
	dst = binary.LittleEndian.AppendUint32(dst, e.E40)
	dst = append(dst, e.E0C)
	return append(dst, e.DirectDamage[:]...)
}

func readSavedEffectPtr(r *savedObjectReader) *SavedEffect {
	if r.err != nil || !r.boolean() {
		return nil
	}
	tag := r.u8()
	e := &SavedEffect{}
	switch tag {
	case 0:
		e.Class = "Effect"
	case 1:
		e.Class = "Effect_DirectDamage"
	default:
		r.err = fmt.Errorf("sim: invalid carried Effect class tag %d", tag)
		return nil
	}
	e.E3C, e.E3D = r.u8(), r.u8()
	e.E40 = r.u32()
	e.E0C = r.u8()
	if b := r.take(24); b != nil {
		copy(e.DirectDamage[:], b)
	}
	if r.err != nil {
		return nil
	}
	if err := savedEffectClassFault(*e); err != nil {
		r.err = err
		return nil
	}
	return e
}

func savedEffectClassFault(e SavedEffect) error {
	switch e.Class {
	case "Effect":
		if e.DirectDamage != [24]byte{} {
			return fmt.Errorf("sim: carried Effect carries nonzero direct-damage bytes")
		}
	case "Effect_DirectDamage":
	default:
		return fmt.Errorf("sim: invalid carried Effect class %q", e.Class)
	}
	return nil
}

func appendSpellEffectPtr(dst []byte, s *SavedSpellEffect) []byte {
	if s == nil {
		return append(dst, 0)
	}
	dst = append(dst, 1)
	return appendSpellEffectNode(dst, *s)
}

func appendSpellEffectNode(dst []byte, s SavedSpellEffect) []byte {
	var tag byte
	switch s.Class {
	case "SpellEffect":
		tag = 0
	case "PointEffect":
		tag = 1
	case "AreaEffect":
		tag = 2
	case "SpellTransport":
		tag = 3
	default:
		panic("sim: invalid SavedSpellEffect class") // carriedResumeStateFault already refused this.
	}
	dst = append(dst, tag, s.SE40, s.SE41)
	dst = binary.LittleEndian.AppendUint32(dst, s.PE44)
	dst = append(dst, s.AE48[:]...)
	dst = binary.LittleEndian.AppendUint16(dst, s.AE4C)
	dst = binary.LittleEndian.AppendUint16(dst, s.ST4C)
	dst = appendSavedEffectPtr(dst, s.PE48)
	dst = appendSavedEffectPtr(dst, s.AE44)
	dst = appendSpellEffectPtr(dst, s.ST44)
	dst = appendSpellEffectPtr(dst, s.ST48)
	return dst
}

func readSpellEffectPtrNode(r *savedObjectReader, depth int) *SavedSpellEffect {
	if r.err != nil || !r.boolean() {
		return nil
	}
	s := readSpellEffectNode(r, depth)
	if r.err != nil {
		return nil
	}
	return &s
}

func readSpellEffectNode(r *savedObjectReader, depth int) SavedSpellEffect {
	if r.err != nil {
		return SavedSpellEffect{}
	}
	if depth > savedSpellEffectMaxDepth {
		r.err = fmt.Errorf("sim: carried SpellEffect nesting past %d", savedSpellEffectMaxDepth)
		return SavedSpellEffect{}
	}
	tag := r.u8()
	var s SavedSpellEffect
	switch tag {
	case 0:
		s.Class = "SpellEffect"
	case 1:
		s.Class = "PointEffect"
	case 2:
		s.Class = "AreaEffect"
	case 3:
		s.Class = "SpellTransport"
	default:
		r.err = fmt.Errorf("sim: invalid carried SpellEffect class tag %d", tag)
		return SavedSpellEffect{}
	}
	s.SE40, s.SE41 = r.u8(), r.u8()
	s.PE44 = r.u32()
	if b := r.take(4); b != nil {
		copy(s.AE48[:], b)
	}
	s.AE4C = r.u16()
	s.ST4C = r.u16()
	s.PE48 = readSavedEffectPtr(r)
	s.AE44 = readSavedEffectPtr(r)
	s.ST44 = readSpellEffectPtrNode(r, depth+1)
	s.ST48 = readSpellEffectPtrNode(r, depth+1)
	if r.err != nil {
		return SavedSpellEffect{}
	}
	if err := savedSpellEffectShapeFault(s); err != nil {
		r.err = err
		return SavedSpellEffect{}
	}
	return s
}

// savedSpellEffectShapeFault refuses a node carrying nonzero state outside
// its own Class, the same rule this form applies everywhere else to a shape
// the encoder could not have written. Used at both encode-time
// (savedSpellEffectTreeFault) and decode-time (readSpellEffectNode).
func savedSpellEffectShapeFault(s SavedSpellEffect) error {
	switch s.Class {
	case "PointEffect":
		if s.AE48 != [4]byte{} || s.AE4C != 0 || s.AE44 != nil || s.ST44 != nil || s.ST48 != nil || s.ST4C != 0 {
			return fmt.Errorf("sim: carried PointEffect carries AreaEffect or SpellTransport state")
		}
	case "AreaEffect":
		if s.PE48 != nil || s.PE44 != 0 || s.ST44 != nil || s.ST48 != nil || s.ST4C != 0 {
			return fmt.Errorf("sim: carried AreaEffect carries PointEffect or SpellTransport state")
		}
	case "SpellTransport":
		if s.PE48 != nil || s.PE44 != 0 || s.AE48 != [4]byte{} || s.AE4C != 0 || s.AE44 != nil {
			return fmt.Errorf("sim: carried SpellTransport carries PointEffect or AreaEffect state")
		}
	default: // bare "SpellEffect"
		if s.PE48 != nil || s.PE44 != 0 || s.AE48 != [4]byte{} || s.AE4C != 0 || s.AE44 != nil ||
			s.ST44 != nil || s.ST48 != nil || s.ST4C != 0 {
			return fmt.Errorf("sim: carried bare SpellEffect carries typed subclass state")
		}
	}
	return nil
}

func savedSpellEffectTreeFault(s SavedSpellEffect, depth int) error {
	if depth > savedSpellEffectMaxDepth {
		return fmt.Errorf("sim: SpellEffect nesting past %d", savedSpellEffectMaxDepth)
	}
	switch s.Class {
	case "SpellEffect", "PointEffect", "AreaEffect", "SpellTransport":
	default:
		return fmt.Errorf("sim: invalid SpellEffect class %q", s.Class)
	}
	if err := savedSpellEffectShapeFault(s); err != nil {
		return err
	}
	if s.PE48 != nil {
		if err := savedEffectClassFault(*s.PE48); err != nil {
			return err
		}
	}
	if s.AE44 != nil {
		if err := savedEffectClassFault(*s.AE44); err != nil {
			return err
		}
	}
	if s.ST44 != nil {
		if err := savedSpellEffectTreeFault(*s.ST44, depth+1); err != nil {
			return err
		}
	}
	if s.ST48 != nil {
		if err := savedSpellEffectTreeFault(*s.ST48, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func appendCarriedSpellEffects(dst []byte, effects []SavedSpellEffect) []byte {
	dst = binary.LittleEndian.AppendUint32(dst, uint32(len(effects)))
	for _, e := range effects {
		dst = appendSpellEffectNode(dst, e)
	}
	return dst
}

func readCarriedSpellEffects(r *savedObjectReader) []SavedSpellEffect {
	n := r.u32()
	if r.err != nil {
		return nil
	}
	if uint64(n)*carriedSpellEffectMinLen > uint64(len(r.data)) {
		r.err = fmt.Errorf("sim: carried SpellEffect count exceeds payload")
		return nil
	}
	if n == 0 {
		return nil
	}
	out := make([]SavedSpellEffect, n)
	for i := range out {
		out[i] = readSpellEffectNode(r, 1)
		if r.err != nil {
			return nil
		}
	}
	return out
}

// --- projectiles -----------------------------------------------------------

func appendCarriedProjectiles(dst []byte, p SavedProjectiles) []byte {
	dst = binary.LittleEndian.AppendUint16(dst, p.FreeIndex)
	dst = binary.LittleEndian.AppendUint32(dst, uint32(len(p.IDs)))
	for _, id := range p.IDs {
		dst = binary.LittleEndian.AppendUint16(dst, id)
	}
	dst = binary.LittleEndian.AppendUint32(dst, uint32(len(p.Items)))
	for _, it := range p.Items {
		dst = binary.LittleEndian.AppendUint16(dst, it.ID)
		for _, v := range [16]int32{
			it.X, it.Y, it.Z, it.Picture, it.Dir, it.Phase, it.LastAction, it.Action,
			it.ActionDir, it.ActionTarget, it.ActionX, it.ActionY, it.ActionZ, it.ActionPhase, it.ActionSegments, it.ActionSpell,
		} {
			dst = binary.LittleEndian.AppendUint32(dst, uint32(v))
		}
	}
	return dst
}

func readCarriedProjectiles(r *savedObjectReader) SavedProjectiles {
	var p SavedProjectiles
	p.FreeIndex = r.u16()
	nids := r.u32()
	if r.err != nil {
		return SavedProjectiles{}
	}
	if uint64(nids)*2 > uint64(len(r.data)) {
		r.err = fmt.Errorf("sim: carried projectile id count exceeds payload")
		return SavedProjectiles{}
	}
	if nids > 0 {
		p.IDs = make([]uint16, nids)
		for i := range p.IDs {
			p.IDs[i] = r.u16()
		}
	}
	nItems := r.u32()
	if r.err != nil {
		return SavedProjectiles{}
	}
	if uint64(nItems)*carriedProjectileRecordLen > uint64(len(r.data)) {
		r.err = fmt.Errorf("sim: carried projectile item count exceeds payload")
		return SavedProjectiles{}
	}
	if nItems > 0 {
		p.Items = make([]SavedProjectile, nItems)
		seen := make(map[uint16]bool, nItems)
		for i := range p.Items {
			it := &p.Items[i]
			it.ID = r.u16()
			vals := [16]*int32{
				&it.X, &it.Y, &it.Z, &it.Picture, &it.Dir, &it.Phase, &it.LastAction, &it.Action,
				&it.ActionDir, &it.ActionTarget, &it.ActionX, &it.ActionY, &it.ActionZ, &it.ActionPhase, &it.ActionSegments, &it.ActionSpell,
			}
			for _, ptr := range vals {
				*ptr = int32(r.u32())
			}
			if r.err != nil {
				return SavedProjectiles{}
			}
			if seen[it.ID] {
				r.err = fmt.Errorf("sim: carried projectiles have duplicate id %04x", it.ID)
				return SavedProjectiles{}
			}
			seen[it.ID] = true
		}
	}
	return p
}

// --- diaries -----------------------------------------------------------

func appendCarriedDiaries(dst []byte, diaries []SavedDiary) []byte {
	dst = binary.LittleEndian.AppendUint32(dst, uint32(len(diaries)))
	for _, d := range diaries {
		dst = append(dst, boolFlag(d.Owner.Player))
		dst = binary.LittleEndian.AppendUint32(dst, uint32(d.Owner.Actor))
		dst = binary.LittleEndian.AppendUint32(dst, uint32(d.Length))
		dst = binary.LittleEndian.AppendUint32(dst, uint32(len(d.Entries)))
		for _, e := range d.Entries {
			dst = binary.LittleEndian.AppendUint32(dst, uint32(e.Index))
			dst = binary.LittleEndian.AppendUint32(dst, e.Count)
			dst = binary.LittleEndian.AppendUint16(dst, e.Remaining)
		}
	}
	return dst
}

func readCarriedDiaries(r *savedObjectReader) []SavedDiary {
	n := r.u32()
	if r.err != nil {
		return nil
	}
	if uint64(n)*carriedDiaryMinLen > uint64(len(r.data)) {
		r.err = fmt.Errorf("sim: carried diary count exceeds payload")
		return nil
	}
	if n == 0 {
		return nil
	}
	out := make([]SavedDiary, n)
	seenPlayer := false
	seenActor := make(map[EntityID]bool, n)
	for i := range out {
		d := &out[i]
		d.Owner.Player = r.boolean()
		actor := r.u32()
		d.Owner.Actor = EntityID(actor)
		d.Length = int(r.u32())
		if r.err != nil {
			return nil
		}
		if d.Owner.Player {
			if actor != 0 {
				r.err = fmt.Errorf("sim: carried Player diary carries a nonzero Actor id")
				return nil
			}
			if seenPlayer {
				r.err = fmt.Errorf("sim: carried diaries have two Player owners")
				return nil
			}
			seenPlayer = true
		} else {
			if seenActor[d.Owner.Actor] {
				r.err = fmt.Errorf("sim: carried diaries have duplicate owner entity %d", d.Owner.Actor)
				return nil
			}
			seenActor[d.Owner.Actor] = true
		}
		nEntries := r.u32()
		if r.err != nil {
			return nil
		}
		if uint64(nEntries)*10 > uint64(len(r.data)) {
			r.err = fmt.Errorf("sim: carried diary entry count exceeds payload")
			return nil
		}
		if nEntries == 0 {
			continue
		}
		d.Entries = make([]SavedDiaryEntry, nEntries)
		for k := range d.Entries {
			idx := int(r.u32())
			d.Entries[k].Index = idx
			d.Entries[k].Count = r.u32()
			d.Entries[k].Remaining = r.u16()
			if r.err != nil {
				return nil
			}
			if idx < 0 || idx >= d.Length {
				r.err = fmt.Errorf("sim: carried diary entry index %d outside its own %d-element arrays", idx, d.Length)
				return nil
			}
			if k > 0 && d.Entries[k-1].Index >= idx {
				r.err = fmt.Errorf("sim: carried diary entries are not strictly ascending by index")
				return nil
			}
		}
	}
	return out
}

func savedDiaryOwnerFault(diaries []SavedDiary) error {
	seenPlayer := false
	seenActor := make(map[EntityID]bool, len(diaries))
	for _, d := range diaries {
		if d.Owner.Player {
			if d.Owner.Actor != 0 {
				return fmt.Errorf("sim: carried Player diary carries a nonzero Actor id")
			}
			if seenPlayer {
				return fmt.Errorf("sim: carried diaries have two Player owners")
			}
			seenPlayer = true
			continue
		}
		if seenActor[d.Owner.Actor] {
			return fmt.Errorf("sim: carried diaries have duplicate owner entity %d", d.Owner.Actor)
		}
		seenActor[d.Owner.Actor] = true
	}
	return nil
}

// carriedResumeStateFault is MarshalBinary's own pre-encode check: every
// invariant this file's decoder later refuses a byte form for is checked here
// first, so a world built through an unvalidated setter (SetSavedCellRecords
// and its four siblings, script.go's SetRawSessionHead/SetRawSessionMid) that
// somehow reached an invalid shape is refused at MarshalBinary rather than
// producing bytes its own UnmarshalBinary cannot read back.
func (w *World) carriedResumeStateFault() error {
	if err := savedCellRecordOrderFault(w.savedCellRecords); err != nil {
		return err
	}
	for i, e := range w.savedSpellEffects {
		if err := savedSpellEffectTreeFault(e, 1); err != nil {
			return fmt.Errorf("sim: carried SpellEffect %d: %w", i, err)
		}
	}
	seen := make(map[uint16]bool, len(w.savedProjectiles.Items))
	for _, it := range w.savedProjectiles.Items {
		if seen[it.ID] {
			return fmt.Errorf("sim: carried projectiles have duplicate id %04x", it.ID)
		}
		seen[it.ID] = true
	}
	if err := savedDiaryOwnerFault(w.savedDiaries); err != nil {
		return err
	}
	for i, d := range w.savedDiaries {
		if d.Length < 0 || d.Length > 0xFFFFFFFF {
			return fmt.Errorf("sim: carried diary %d length %d does not fit the wire form", i, d.Length)
		}
		for k, e := range d.Entries {
			if e.Index < 0 || e.Index >= d.Length {
				return fmt.Errorf("sim: carried diary %d entry %d index %d outside its own %d-element arrays",
					i, k, e.Index, d.Length)
			}
			if k > 0 && d.Entries[k-1].Index >= e.Index {
				return fmt.Errorf("sim: carried diary %d entries are not strictly ascending by index", i)
			}
		}
	}
	return nil
}
