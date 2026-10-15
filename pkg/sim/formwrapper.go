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
	decode  func(*World, []byte, formProof) error
}

// formWrappers is the one list of World form wrappers, outermost first.
// CheckSaveForm, unmarshalForm and HasStructureBlockingForm read it. It is
// filled in init because each decoder decodes its base through unmarshalForm.
var formWrappers []formWrapper

func init() {
	formWrappers = []formWrapper{
		{moverFormVersion, "MVR1", trailerSpan32, (*World).unmarshalMovers},
		{autoCastFormVersion, "ACS1", trailerPairs8, (*World).unmarshalAutoCastSettings},
		{bodyReferenceFormVersion, "BDR1", trailerSpan32, (*World).unmarshalBodyReferences},
		{speedModifierFormVersion, "SPM1", trailerSpan32, (*World).unmarshalSpeedModifiers},
		{heldOrderFormVersion, "HLD1", trailerSpan32, (*World).unmarshalHeldOrders},
		{pursuitSearchFormVersion, "PRS1", trailerSpan32, (*World).unmarshalPursuitSearches},
		{escortFormVersion, "ESC1", trailerSpan32, (*World).unmarshalEscortResidues},
		{turnStateFormVersion, "TRN1", trailerSpan32, (*World).unmarshalTurnStates},
		{nativeScalarFormVersion, "NSC1", trailerSpan32, (*World).unmarshalNativeScalars},
		{nativeItemFormVersion, "NIR1", trailerSpan32, (*World).unmarshalNativeItemRecords},
		{nativeLiveFormVersion, "NLB1", trailerSpan32, (*World).unmarshalNativeLiveBlocks},
		{currentPlayerFormVersion, "CPP1", trailerSpan32, (*World).unmarshalCurrentPlayers},
		{playerParticipantFormVersion, "PPT1", trailerSpan32, (*World).unmarshalPlayerParticipants},
		{nativeBasisFormVersion, "NAB1", trailerSpan32, (*World).unmarshalNativeActorBases},
		{bookSelectionFormVersion, "BSL1", trailerSpan32, (*World).unmarshalBookSelections},
		{nativeClassFormVersion, "CLS1", trailerSpan32, (*World).unmarshalNativeClasses},
		{rom2ScriptFormVersion, "R2S1", trailerSpan32, (*World).unmarshalROM2ScriptState},
		{nativeTrainingFormVersion, "TRN1", trailerSpan32, (*World).unmarshalNativeTraining},
		{areaCostFormVersion, "ACP1", trailerSpan32, (*World).unmarshalAreaCosts},
		{creatureSpellFormVersion, "CSP1", trailerSpan32, (*World).unmarshalCreatureSpells},
		{pendingOrderFormVersion, "ORD1", trailerSpan32, (*World).unmarshalPendingOrders},
		{tacticalFormVersion, "TAC1", trailerSpan32, (*World).unmarshalTactical},
		{structureBlockingFormVersion, "SBK1", trailerSpan32, (*World).unmarshalStructureBlocking},
		{currentTerminalFormVersion, "CTA1", trailerSpan32, (*World).unmarshalCurrentTerminalActors},
		{currentAreaFormVersion, "SPG1", trailerSpan32, (*World).unmarshalSavedSpellGraph},
		{spellGraphFormVersion, "SPG1", trailerSpan32, (*World).unmarshalSavedSpellGraph},
		{autoHealingFormVersion, "AHL1", trailerSpan16, (*World).unmarshalAutoHealing},
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

// inner returns the wrapped form's version and the bytes that precede the
// wrapper's section. It checks only the tag and the bounds; the wrapper's
// decoder validates the section itself.
func (fw formWrapper) inner(data []byte) (byte, []byte, bool) {
	end := len(data)
	if end < headerLen+9 || string(data[end-4:]) != fw.tag {
		return 0, nil, false
	}
	switch fw.trailer {
	case trailerSpan32:
		span := uint64(binary.LittleEndian.Uint32(data[end-9:]))
		if span > uint64(end-headerLen-9) {
			return 0, nil, false
		}
		return data[end-5], data[:end-9-int(span)], true
	case trailerPairs8:
		start := end - 6 - 2*int(data[end-6])
		if start < headerLen {
			return 0, nil, false
		}
		return data[end-5], data[:start], true
	case trailerSpan16:
		start := end - 6 - int(binary.LittleEndian.Uint16(data[end-6:]))
		if start < headerLen {
			return 0, nil, false
		}
		return formatVersion, data[:start], true
	}
	return 0, nil, false
}
