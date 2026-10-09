package sim

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const structureBlockingFormVersion byte = 100

func HasStructureBlockingForm(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	version := data[0]
	for version == heldOrderFormVersion || version == escortFormVersion || version == turnStateFormVersion || version == nativeScalarFormVersion || version == nativeLiveFormVersion || version == nativeItemFormVersion || version == currentPlayerFormVersion || version == playerParticipantFormVersion || version == nativeBasisFormVersion || version == bookSelectionFormVersion || version == nativeClassFormVersion || version == rom2ScriptFormVersion || version == nativeTrainingFormVersion || version == areaCostFormVersion || version == creatureSpellFormVersion || version == pendingOrderFormVersion || version == tacticalFormVersion {
		if len(data) < headerLen+9 {
			return false
		}
		tag := "TAC1"
		switch version {
		case heldOrderFormVersion:
			tag = "HLD1"
		case escortFormVersion:
			tag = "ESC1"
		case turnStateFormVersion:
			tag = "TRN1"
		case nativeScalarFormVersion:
			tag = "NSC1"
		case nativeLiveFormVersion:
			tag = "NLB1"
		case nativeItemFormVersion:
			tag = "NIR1"
		case currentPlayerFormVersion:
			tag = "CPP1"
		case playerParticipantFormVersion:
			tag = "PPT1"
		case nativeBasisFormVersion:
			tag = "NAB1"
		case bookSelectionFormVersion:
			tag = "BSL1"
		case nativeClassFormVersion:
			tag = "CLS1"
		case rom2ScriptFormVersion:
			tag = "R2S1"
		case nativeTrainingFormVersion:
			tag = "TRN1"
		case pendingOrderFormVersion:
			tag = "ORD1"
		case areaCostFormVersion:
			tag = "ACP1"
		case creatureSpellFormVersion:
			tag = "CSP1"
		}
		end := len(data)
		span := uint64(binary.LittleEndian.Uint32(data[end-9:]))
		if string(data[end-4:]) != tag || data[end-5] >= version || span > uint64(end-headerLen-9) {
			return false
		}
		version, data = data[end-5], data[:end-9-int(span)]
	}
	return version == structureBlockingFormVersion
}

func (w *World) appendStructureBlocking(b []byte) []byte {
	count := 0
	for _, st := range w.structures {
		if st.Blocking != st.Attach {
			count++
		}
	}
	if count == 0 {
		return b
	}
	base := b[0]
	b[0] = structureBlockingFormVersion
	b = binary.LittleEndian.AppendUint32(b, uint32(count))
	for _, st := range w.structures {
		if st.Blocking != st.Attach {
			b = binary.LittleEndian.AppendUint32(b, uint32(st.ID))
			b = binary.LittleEndian.AppendUint32(b, st.Blocking)
		}
	}
	b = binary.LittleEndian.AppendUint32(b, uint32(4+8*count))
	return append(b, base, 'S', 'B', 'K', '1')
}

func (w *World) unmarshalStructureBlocking(data []byte) error {
	fail := func() error { return fmt.Errorf("malformed structure blocking section") }
	if len(data) < headerLen+21 || !bytes.Equal(data[len(data)-4:], []byte("SBK1")) {
		return fail()
	}
	baseVersion := data[len(data)-5]
	if baseVersion == structureBlockingFormVersion {
		return fail()
	}
	span := uint64(binary.LittleEndian.Uint32(data[len(data)-9:]))
	if span < 12 || span > uint64(len(data)-headerLen-9) {
		return fail()
	}
	start := len(data) - 9 - int(span)
	payload := data[start : len(data)-9]
	count := uint64(binary.LittleEndian.Uint32(payload))
	if count == 0 || span != 4+8*count {
		return fail()
	}
	base := bytes.Clone(data[:start])
	base[0] = baseVersion
	next := *w
	if err := next.UnmarshalBinary(base); err != nil {
		return err
	}
	if count > uint64(len(next.structures)) {
		return fail()
	}
	var prior StructureID
	for i := uint64(0); i < count; i++ {
		o := 4 + 8*int(i)
		id := StructureID(binary.LittleEndian.Uint32(payload[o:]))
		mask := binary.LittleEndian.Uint32(payload[o+4:])
		if i > 0 && id <= prior {
			return fail()
		}
		prior = id
		at := indexOfStructure(next.structures, id)
		if at < 0 || mask == next.structures[at].Attach {
			return fail()
		}
		next.structures[at].Blocking = mask
	}
	canonical, err := next.MarshalBinary()
	if err != nil || !bytes.Equal(canonical, data) {
		return fail()
	}
	*w = next
	return nil
}
