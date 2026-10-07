package sav

import "fmt"

// SpellEffect is the typed projection of one object in the top-level
// SpellEffect-list graph (SAV-DOC-053): a bare SpellEffect, or a
// PointEffect, AreaEffect or SpellTransport, chosen by Class exactly as
// Building projects Outpost/Tavern/Shop into one flat shape —
// SAV-CLASSSER-172..177 publish these four programmes together and none
// recurs at more than one graph position. Off is this record's own body
// offset, not counting a typed child's.
type SpellEffect struct {
	Off   int
	Class string // "SpellEffect", "PointEffect", "AreaEffect" or "SpellTransport".

	// SE40/SE41 (+0x40/+0x41) are on every subtype (SAV-CLASSSER-173).
	SE40, SE41 uint8

	// PointEffect only (SAV-CLASSSER-174). PE48 is the typed Effect the
	// load arm expects. PE44 is a raw dword the original resolves through
	// the archive identity map at a post-load step no promoted claim
	// states the effect of (SAV-CLASSSER-174: "What the repaired pointer
	// means to simulation is Unknown") — this package carries it raw and
	// never interprets or resolves it.
	PE48 *Effect
	PE44 uint32

	// AreaEffect only (SAV-CLASSSER-175): four raw bytes, a word, then the
	// typed Effect the load arm expects.
	AE48 [4]byte
	AE4C uint16
	AE44 *Effect

	// SpellTransport only (SAV-CLASSSER-175): the typed SpellEffect and
	// AreaEffect the load arm expects, then a trailing word. ST48's own
	// Class reads "AreaEffect" whenever it is present; this package does
	// not enforce that as a decode-time constraint beyond what the archive
	// stream itself states, on B1 (the wire grammar does not carry a
	// static type check).
	ST44 *SpellEffect
	ST48 *SpellEffect
	ST4C uint16
}

// Effect is the typed projection of an Effect or Effect_DirectDamage object
// reached through a SpellEffect graph reference (SAV-EFFCHAIN-046 gives
// Effect's own Token+E3C+E3D+E40+E0C=44 bytes; SAV-CLASSSER-173 gives
// Effect_DirectDamage's extra 24 raw bytes, 68 total). The same two classes
// also appear, class-checked but not typed, in an Item's own Effects list
// (GroundSacks, SAV-EQUIPEFFECT-553); this projection serves only the
// SpellEffect graph.
type Effect struct {
	Off   int
	Class string // "Effect" or "Effect_DirectDamage".

	E3C, E3D uint8
	E40      uint32
	E0C      uint8

	// Effect_DirectDamage only: 24 raw bytes from +0x48.
	DirectDamage [24]byte
}

// SpellEffects returns the top-level SpellEffect-list object graph in
// archive order. A no-world save differs from an empty saved list, the same
// distinction GroundSacks and Buildings already draw for their own lists.
//
// VirtualCaster has its own closed programme (SAV-CLASSSER-172/173) but no
// promoted claim names a field of SpellEffect, PointEffect, AreaEffect,
// SpellTransport or Effect that references one, and SAV-CLASSSER-177 finds
// zero VirtualCaster records or raw-name hits across the entire permitted
// corpus. This projection therefore has no VirtualCaster site to type; one
// would be invented reach, not a carried field.
func (f *File) SpellEffects() ([]SpellEffect, bool, error) {
	doc, present, err := f.exactDocument()
	if err != nil || !present {
		return nil, present, err
	}
	out := make([]SpellEffect, len(doc.effects))
	for i, r := range doc.effects {
		s, err := spellEffectFromRecord(r, 0)
		if err != nil {
			return nil, true, err
		}
		out[i] = *s
	}
	return out, true, nil
}

// spellEffectMaxDepth bounds SpellTransport's own ST44/ST48 recursion. The
// wire grammar's back-reference arm names any object already read
// (SAV-STREAM-013) with no rule that it be a descendant, and the walker
// registers a record in w.objects before running its own programme
// (program.go's body()), precisely so a reference inside a record's own body
// can name that record — so a cyclic ST44/ST48 chain is a valid *Record graph
// the walker hands back unchanged. The walker's own maxWalkDepth does not
// bound this: a back-reference is not re-read, so it never advances the
// walker's own recursion. Reusing maxWalkDepth's value here turns such a
// cycle into a bounded error instead of a stack overflow, the same rule
// maxWalkDepth already states for the walker itself.
const spellEffectMaxDepth = maxWalkDepth

func spellEffectFromRecord(r *Record, depth int) (*SpellEffect, error) {
	if r == nil {
		return nil, fmt.Errorf("sav: a top-level SpellEffect list entry is null")
	}
	if depth > spellEffectMaxDepth {
		return nil, fmt.Errorf("sav: SpellEffect nesting past %d at %d", spellEffectMaxDepth, r.Off)
	}
	if !groundClass(r.Class, "SpellEffect") {
		return nil, fmt.Errorf("sav: %s is not a SpellEffect reference", r.Class)
	}
	s := &SpellEffect{Off: r.Off, Class: r.Class,
		SE40: uint8(r.value("SE40")), SE41: uint8(r.value("SE41"))}
	switch r.Class {
	case "PointEffect":
		child, err := effectFromRecord(oneRef(r, "PE48"))
		if err != nil {
			return nil, err
		}
		s.PE48, s.PE44 = child, r.value("PE44")
	case "AreaEffect":
		copy(s.AE48[:], r.Raw["AE48"])
		s.AE4C = uint16(r.value("AE4C"))
		child, err := effectFromRecord(oneRef(r, "AE44"))
		if err != nil {
			return nil, err
		}
		s.AE44 = child
	case "SpellTransport":
		st44, err := spellEffectFromRecordOrNil(oneRef(r, "ST44"), depth+1)
		if err != nil {
			return nil, err
		}
		st48, err := spellEffectFromRecordOrNil(oneRef(r, "ST48"), depth+1)
		if err != nil {
			return nil, err
		}
		s.ST44, s.ST48, s.ST4C = st44, st48, uint16(r.value("ST4C"))
	}
	return s, nil
}

// spellEffectFromRecordOrNil is spellEffectFromRecord for a nested reference
// site, where a null reference is a null typed value rather than an error:
// SAV-EFFECTGRAPH-366's own witnessed graph has a null typed AreaEffect
// reference at this exact site.
func spellEffectFromRecordOrNil(r *Record, depth int) (*SpellEffect, error) {
	if r == nil {
		return nil, nil
	}
	return spellEffectFromRecord(r, depth)
}

func effectFromRecord(r *Record) (*Effect, error) {
	if r == nil {
		return nil, nil
	}
	if r.Class != "Effect" && r.Class != "Effect_DirectDamage" {
		return nil, fmt.Errorf("sav: %s is not an Effect reference", r.Class)
	}
	e := &Effect{Off: r.Off, Class: r.Class,
		E3C: uint8(r.value("E3C")), E3D: uint8(r.value("E3D")),
		E40: r.value("E40"), E0C: uint8(r.value("E0C"))}
	if r.Class == "Effect_DirectDamage" {
		copy(e.DirectDamage[:], r.Raw["EDD48"])
	}
	return e, nil
}

func oneRef(r *Record, name string) *Record {
	refs := r.Refs[name]
	if len(refs) == 0 {
		return nil
	}
	return refs[0]
}

// SetSpellEffects patches the top-level SpellEffect-list scalar fields in
// place, from effects matched row for row and reference for reference
// against the file's own current graph shape.
//
// It is a content patch, not a re-encoder. SAV-CLASSSER-172..177 name no
// LOAD-time creation, removal or relocation of a SpellEffect object, so a
// native writer's own list length and reference topology already equal the
// file's — SetCell's own precedent for the fixed-stride cell-record table,
// carried over to this variable-stride one. The archive tag, class-name and
// index bytes an object reference already carries are never rewritten, only
// the named scalar members; a reference this package cannot attribute to a
// fresh nested introduction within the record it is reading (a back-reference,
// or one this package did not itself just decode) is left exactly as the file
// already has it. No corpus record this story reached exercises that arm
// (SAV-CLASSSER-177); the guard exists so a future one fails closed rather
// than silently misplacing a write into an unrelated record.
func (f *File) SetSpellEffects(effects []SpellEffect) error {
	doc, present, err := f.exactDocument()
	if err != nil {
		return err
	}
	if !present {
		return fmt.Errorf("sav: this save has no world session")
	}
	if len(effects) != len(doc.effects) {
		return fmt.Errorf("sav: %d SpellEffect values for %d records", len(effects), len(doc.effects))
	}
	for i, r := range doc.effects {
		if err := f.patchSpellEffect(r, effects[i]); err != nil {
			return fmt.Errorf("sav: SpellEffect %d: %w", i, err)
		}
	}
	return nil
}

// patchSpellEffect writes s's own scalar fields into the bytes r already
// occupies, then recurses into whichever typed child r's own class
// introduces there, guarded by refSpan below.
func (f *File) patchSpellEffect(r *Record, s SpellEffect) error {
	if r == nil || r.Class != s.Class {
		return fmt.Errorf("record class does not match value class %q", s.Class)
	}
	p := f.Body
	// r.Off is this record's own first byte, the start of its embedded
	// 37-byte Token head (Class.Fixed's own rule: Head adds TokenLen ahead
	// of Extent) — SE40 is the first byte SpellEffect's own programme
	// contributes, past that head.
	base := r.Off + TokenLen
	p[base], p[base+1] = s.SE40, s.SE41
	off := base + 2
	switch r.Class {
	case "PointEffect":
		child := oneRef(r, "PE48")
		if (child == nil) != (s.PE48 == nil) {
			return fmt.Errorf("PE48 reference presence does not match the record")
		}
		end, fresh := refSpan(off, child)
		if fresh {
			if err := f.patchEffect(child, *s.PE48); err != nil {
				return err
			}
		}
		put32(p, end, s.PE44)
	case "AreaEffect":
		copy(p[off:off+4], s.AE48[:])
		put16(p, off+4, s.AE4C)
		child := oneRef(r, "AE44")
		if (child == nil) != (s.AE44 == nil) {
			return fmt.Errorf("AE44 reference presence does not match the record")
		}
		if _, fresh := refSpan(off+6, child); fresh {
			return f.patchEffect(child, *s.AE44)
		}
	case "SpellTransport":
		st44 := oneRef(r, "ST44")
		if (st44 == nil) != (s.ST44 == nil) {
			return fmt.Errorf("ST44 reference presence does not match the record")
		}
		end44, fresh44 := refSpan(off, st44)
		if fresh44 {
			if err := f.patchSpellEffect(st44, *s.ST44); err != nil {
				return err
			}
		}
		st48 := oneRef(r, "ST48")
		if (st48 == nil) != (s.ST48 == nil) {
			return fmt.Errorf("ST48 reference presence does not match the record")
		}
		end48, fresh48 := refSpan(end44, st48)
		if fresh48 {
			if err := f.patchSpellEffect(st48, *s.ST48); err != nil {
				return err
			}
		}
		put16(p, end48, s.ST4C)
	}
	return nil
}

func (f *File) patchEffect(r *Record, e Effect) error {
	if r == nil || r.Class != e.Class {
		return fmt.Errorf("record class does not match value class %q", e.Class)
	}
	p := f.Body
	// Same TokenLen shift as patchSpellEffect: Effect and Effect_DirectDamage
	// both open with the embedded Token head (Effect_DirectDamage's own
	// programme chains through Effect, which chains through Token), so E3C
	// begins TokenLen bytes past r.Off, and Effect_DirectDamage's own 24 raw
	// bytes begin TokenLen+7 past it (E3C, E3D, E40, E0C = 7 bytes).
	base := r.Off + TokenLen
	p[base], p[base+1] = e.E3C, e.E3D
	put32(p, base+2, e.E40)
	p[base+6] = e.E0C
	if r.Class == "Effect_DirectDamage" {
		copy(p[base+7:base+31], e.DirectDamage[:])
	}
	return nil
}

// refSpan answers where the bytes right after one object reference at
// tagStart begin, and whether that reference is a fresh introduction a
// caller may recurse a scalar patch into. A null reference and a
// back-reference are both exactly two bytes here and neither is fresh: a
// back-reference's own record was fully written wherever it was first
// introduced, which is never at or after tagStart.
func refSpan(tagStart int, child *Record) (int, bool) {
	if child == nil || child.Off <= tagStart {
		return tagStart + 2, false
	}
	return child.End, true
}
