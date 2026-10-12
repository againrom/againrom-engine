package sim

import "encoding/binary"

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
	decode  func(*World, []byte) error
}

// formWrappers is the one list of World form wrappers, outermost first.
// CheckSaveForm, unmarshalForm and HasStructureBlockingForm read it. It is
// filled in init because each decoder decodes its base through unmarshalForm.
var formWrappers []formWrapper

func init() {
	formWrappers = []formWrapper{
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
