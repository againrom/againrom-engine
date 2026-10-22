package sim

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"reflect"
	"sync"
)

// wrapperTrailer names how a continuation wrapper ends, and so how the form
// it wraps is found.
type wrapperTrailer uint8

const (
	// trailerSpan32: section, uint32 section length, base version, tag.
	trailerSpan32 wrapperTrailer = iota
	// trailerPairs8: two-byte rows, row count byte, base version, tag.
	trailerPairs8
	// trailerSpan16: section, uint16 section length, tag; the base is
	// formatVersion.
	trailerSpan16
)

// formWrapper is one continuation form of the World byte form: a version
// byte that replaces the wrapped form's own, a section appended after it,
// and a trailer that names the wrapped form's version.
type formWrapper struct {
	version byte
	tag     string
	trailer wrapperTrailer
	base    wrapperBase
	decode  func(*World, []byte, formProof) error
}

// wrapperBase admits the version of the form a wrapper wraps.
type wrapperBase func(own, base byte) bool

func baseOlder(own, base byte) bool { return base < own }

func baseOther(own, base byte) bool { return base != own }

func baseAny(byte, byte) bool { return true }

// baseSpellGraph admits the plain form and the autohealing wrapper.
func baseSpellGraph(_, base byte) bool {
	return base == formatVersion || base == autoHealingFormVersion
}

// formWrappers is the one list of World form wrappers, outermost first.
// CheckSaveForm, unmarshalForm and HasStructureBlockingForm read it. It is
// filled in init because each decoder decodes its base through unmarshalForm.
var formWrappers []formWrapper

func init() {
	formWrappers = []formWrapper{
		{passiveBodyFormVersion, "PBC1", trailerSpan32, baseOlder, (*World).unmarshalPassiveBodies},
		{moverRadiusFormVersion, "MRD1", trailerSpan32, baseOlder, (*World).unmarshalMoverRadii},
		{walkClaimFormVersion, "WCL1", trailerSpan32, baseOlder, (*World).unmarshalWalkClaims},
		{flightRunFormVersion, "FLR1", trailerSpan32, baseOlder, (*World).unmarshalFlightRuns},
		{originalDeadHealthFormVersion, "DMH1", trailerSpan32, baseOlder, (*World).unmarshalOriginalDeadHealth},
		{moverFormVersion, "MVR1", trailerSpan32, baseOlder, (*World).unmarshalMovers},
		{autoCastFormVersion, "ACS1", trailerPairs8, baseOlder, (*World).unmarshalAutoCastSettings},
		{bodyReferenceFormVersion, "BDR1", trailerSpan32, baseOlder, (*World).unmarshalBodyReferences},
		{speedModifierFormVersion, "SPM1", trailerSpan32, baseOlder, (*World).unmarshalSpeedModifiers},
		{heldOrderFormVersion, "HLD1", trailerSpan32, baseOlder, (*World).unmarshalHeldOrders},
		{pursuitSearchFormVersion, "PRS1", trailerSpan32, baseOlder, (*World).unmarshalPursuitSearches},
		{escortFormVersion, "ESC1", trailerSpan32, baseOlder, (*World).unmarshalEscortResidues},
		{turnStateFormVersion, "TRN1", trailerSpan32, baseOlder, (*World).unmarshalTurnStates},
		{nativeScalarFormVersion, "NSC1", trailerSpan32, baseOlder, (*World).unmarshalNativeScalars},
		{nativeItemFormVersion, "NIR1", trailerSpan32, baseOlder, (*World).unmarshalNativeItemRecords},
		{nativeLiveFormVersion, "NLB1", trailerSpan32, baseOlder, (*World).unmarshalNativeLiveBlocks},
		{currentPlayerFormVersion, "CPP1", trailerSpan32, baseOlder, (*World).unmarshalCurrentPlayers},
		{playerParticipantFormVersion, "PPT1", trailerSpan32, baseOlder, (*World).unmarshalPlayerParticipants},
		{nativeBasisFormVersion, "NAB1", trailerSpan32, baseOlder, (*World).unmarshalNativeActorBases},
		{bookSelectionFormVersion, "BSL1", trailerSpan32, baseOlder, (*World).unmarshalBookSelections},
		{nativeClassFormVersion, "CLS1", trailerSpan32, baseOlder, (*World).unmarshalNativeClasses},
		{rom2ScriptFormVersion, "R2S1", trailerSpan32, baseOlder, (*World).unmarshalROM2ScriptState},
		{nativeTrainingFormVersion, "TRN1", trailerSpan32, baseOlder, (*World).unmarshalNativeTraining},
		{areaCostFormVersion, "ACP1", trailerSpan32, baseOlder, (*World).unmarshalAreaCosts},
		{creatureSpellFormVersion, "CSP1", trailerSpan32, baseOlder, (*World).unmarshalCreatureSpells},
		{pendingOrderFormVersion, "ORD1", trailerSpan32, baseOlder, (*World).unmarshalPendingOrders},
		{tacticalFormVersion, "TAC1", trailerSpan32, baseOther, (*World).unmarshalTactical},
		{structureBlockingFormVersion, "SBK1", trailerSpan32, baseOther, (*World).unmarshalStructureBlocking},
		{currentTerminalFormVersion, "CTA1", trailerSpan32, baseAny, (*World).unmarshalCurrentTerminalActors},
		{currentAreaFormVersion, "SPG1", trailerSpan32, baseSpellGraph, (*World).unmarshalSavedSpellGraph},
		{spellGraphFormVersion, "SPG1", trailerSpan32, baseSpellGraph, (*World).unmarshalSavedSpellGraph},
		{autoHealingFormVersion, "AHL1", trailerSpan16, baseAny, (*World).unmarshalAutoHealing},
	}
}

// canonicalForms holds the byte forms a wrapper decoder writes only to compare
// with its input; no buffer outlives the comparison.
var canonicalForms sync.Pool

// formProof is true when the bytes being decoded already decoded once into a
// zero World and passed every wrapper's canonical comparison on the way.
// Decoding reads only the bytes and the receiver, so the same bytes decoded
// into a zero World pass the same comparisons again.
type formProof bool

// marshalsTo reports whether w marshals without error to exactly data, the
// canonical re-encoding every wrapper decoder requires of its input. A proven
// form has passed it already.
func (w *World) marshalsTo(data []byte, proof formProof) bool {
	if proof {
		return true
	}
	p, _ := canonicalForms.Get().(*[]byte)
	if p == nil {
		p = new([]byte)
	}
	canonical, err := w.MarshalBinaryInto((*p)[:0])
	equal := err == nil && bytes.Equal(canonical, data)
	if err == nil {
		*p = canonical
	}
	canonicalForms.Put(p)
	return equal
}

// provenFormCount bounds the proven-form memo; one SAVE or LOAD decodes the
// same form several times in a row.
const provenFormCount = 16

// provenForms holds the SHA-256 digests of the wrapper forms most recently
// decoded into a zero World without error, newest first. It holds no World
// and no form bytes, so it hands out nothing that can alias decoded state, and
// a changed byte changes the digest.
var provenForms provenFormMemo

type provenFormMemo struct {
	mu      sync.Mutex
	n       int
	digests [provenFormCount][sha256.Size]byte
}

func (m *provenFormMemo) has(d [sha256.Size]byte) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.n {
		if m.digests[i] == d {
			copy(m.digests[1:i+1], m.digests[:i])
			m.digests[0] = d
			return true
		}
	}
	return false
}

func (m *provenFormMemo) add(d [sha256.Size]byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	at := m.n
	for i := range m.n {
		if m.digests[i] == d {
			at = i
			break
		}
	}
	if at == len(m.digests) {
		at--
	} else if at == m.n {
		m.n++
	}
	copy(m.digests[1:at+1], m.digests[:at])
	m.digests[0] = d
}

// formDigest returns the SHA-256 of data and whether the proven-form memo
// applies to this decode: a wrapper form into a zero World. Any other
// receiver carries its own fields into the decoded World, and a base form has
// no canonical comparison to skip.
func (w *World) formDigest(data []byte) ([sha256.Size]byte, bool) {
	if len(data) == 0 {
		return [sha256.Size]byte{}, false
	}
	if _, ok := formWrapperOf(data[0]); !ok || !reflect.ValueOf(w).Elem().IsZero() {
		return [sha256.Size]byte{}, false
	}
	return sha256.Sum256(data), true
}

func formWrapperOf(version byte) (formWrapper, bool) {
	for _, fw := range formWrappers {
		if fw.version == version {
			return fw, true
		}
	}
	return formWrapper{}, false
}

// wrapperEnvelope is a wrapper read back: section data[start:end] after
// the wrapped form data[:start] of version baseVersion.
type wrapperEnvelope struct {
	data        []byte
	start, end  int
	baseVersion byte
}

func (e wrapperEnvelope) section() []byte { return e.data[e.start:e.end] }

func (e wrapperEnvelope) base() []byte {
	b := bytes.Clone(e.data[:e.start])
	b[0] = e.baseVersion
	return b
}

// envelope reads fw's tag and the section bounds its trailer names.
func (fw formWrapper) envelope(data []byte) (wrapperEnvelope, bool) {
	end := len(data)
	if end < headerLen+6 || string(data[end-4:]) != fw.tag {
		return wrapperEnvelope{}, false
	}
	switch fw.trailer {
	case trailerSpan32:
		if end < headerLen+9 {
			return wrapperEnvelope{}, false
		}
		span := uint64(binary.LittleEndian.Uint32(data[end-9:]))
		if span > uint64(end-headerLen-9) {
			return wrapperEnvelope{}, false
		}
		return wrapperEnvelope{data, end - 9 - int(span), end - 9, data[end-5]}, true
	case trailerPairs8:
		start := end - 6 - 2*int(data[end-6])
		if start < headerLen {
			return wrapperEnvelope{}, false
		}
		return wrapperEnvelope{data, start, end - 6, data[end-5]}, true
	case trailerSpan16:
		start := end - 6 - int(binary.LittleEndian.Uint16(data[end-6:]))
		if start < headerLen {
			return wrapperEnvelope{}, false
		}
		return wrapperEnvelope{data, start, end - 6, formatVersion}, true
	}
	return wrapperEnvelope{}, false
}

// readWrapper reads the envelope of the given wrapper version and checks its
// base version; the wrapper's decoder validates the section.
func readWrapper(data []byte, version byte) (wrapperEnvelope, bool) {
	fw, ok := formWrapperOf(version)
	if !ok {
		return wrapperEnvelope{}, false
	}
	e, ok := fw.envelope(data)
	if !ok || !fw.base(fw.version, e.baseVersion) {
		return wrapperEnvelope{}, false
	}
	return e, true
}

// inner returns the wrapped form's version and the bytes that precede the
// wrapper's section. It checks only the tag and the bounds.
func (fw formWrapper) inner(data []byte) (byte, []byte, bool) {
	if len(data) < headerLen+9 {
		return 0, nil, false
	}
	e, ok := fw.envelope(data)
	if !ok {
		return 0, nil, false
	}
	return e.baseVersion, data[:e.start], true
}

// wrapperStart is the wrapped form's version and where a section begins.
type wrapperStart struct {
	base byte
	at   int
}

func beginWrapper(b []byte) wrapperStart { return wrapperStart{b[0], len(b)} }

// endWrapper appends the version's trailer after the section begun at s.
func endWrapper(b []byte, s wrapperStart, version byte) []byte {
	fw, ok := formWrapperOf(version)
	if !ok {
		panic("sim: unlisted form wrapper version")
	}
	b[0] = version
	n := len(b) - s.at
	switch fw.trailer {
	case trailerSpan32:
		b = binary.LittleEndian.AppendUint32(b, uint32(n))
		b = append(b, s.base)
	case trailerPairs8:
		b = append(b, byte(n/2), s.base)
	case trailerSpan16:
		b = binary.LittleEndian.AppendUint16(b, uint16(n))
	}
	return append(b, fw.tag...)
}
