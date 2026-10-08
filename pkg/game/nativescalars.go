package game

import (
	"encoding/binary"
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

var nativeScalarFields = [sim.ScalarCount]string{
	"T0C", "T08", "T18", "T1C", "Reference", "U4B", "U4C", "U6C", "U8E", "U60", "U61", "UA0", "UA4", "U130", "U136", "U138", "U148", "U144", "U50", "U54", "U58",
}

func readNativeScalar(r *sav.DocumentRecordData, slot int) (uint32, error) {
	if slot >= sim.ScalarU50 {
		raw, err := savedActorRaw(r, nativeScalarFields[slot], 4)
		if err != nil {
			return 0, err
		}
		return binary.LittleEndian.Uint32(raw), nil
	}
	value, err := savedStructureValue(r, nativeScalarFields[slot])
	if slot == sim.ScalarT08High {
		value >>= 16
	}
	return value, err
}

func observeNativeScalars(b sim.NativeActorBasis, r *sav.DocumentRecordData) (sim.NativeActorBasis, error) {
	if !b.ScalarsPresent {
		for slot := range b.Scalars {
			value, err := readNativeScalar(r, slot)
			if err != nil {
				return b, err
			}
			b.Scalars[slot] = value
		}
		b.ScalarsPresent, b.ScalarKnown = true, 1<<sim.ScalarCount-1
	}
	if !b.BlockPresent {
		raw, err := savedActorRaw(r, "Block12", 12)
		if err != nil {
			return b, err
		}
		copy(b.Block[:], raw[2:])
		b.BlockPresent, b.BlockKnown = true, 0x3ff
	}
	return b, b.Validate()
}

func nativeScalarRawCopy(r *sav.DocumentRecordData, name string, size int) ([]byte, error) {
	raw, err := savedActorRaw(r, name, size)
	if err != nil {
		return nil, err
	}
	for i := range r.Raw {
		if r.Raw[i].Name == name {
			r.Raw[i].Bytes = slices.Clone(raw)
			return r.Raw[i].Bytes, nil
		}
	}
	return nil, fmt.Errorf("native scalar raw field is missing")
}

func projectNativeScalars(r *sav.DocumentRecordData, b sim.NativeActorBasis) error {
	for slot, value := range b.Scalars {
		if !b.ScalarIsKnown(slot) {
			continue
		}
		name := nativeScalarFields[slot]
		if slot >= sim.ScalarU50 {
			raw, err := nativeScalarRawCopy(r, name, 4)
			if err != nil {
				return err
			}
			binary.LittleEndian.PutUint32(raw, value)
			continue
		}
		if slot == sim.ScalarT08High {
			low, err := savedStructureValue(r, name)
			if err != nil {
				return err
			}
			value = value<<16 | low&0xffff
		}
		if err := savedActorSetValue(r, name, value); err != nil {
			return err
		}
	}
	if b.BlockKnown != 0 {
		raw, err := nativeScalarRawCopy(r, "Block12", 12)
		if err != nil {
			return err
		}
		for n, value := range b.Block {
			if b.BlockByteKnown(n) {
				raw[n+2] = value
			}
		}
	}
	return nil
}

func captureNativeScalars(b *sim.NativeActorBasis, r *sav.DocumentRecordData) error {
	for slot := range b.Scalars {
		if b.ScalarIsKnown(slot) {
			if _, err := readNativeScalar(r, slot); err != nil {
				return err
			}
			b.Scalars[slot] = 0
		}
	}
	if b.BlockKnown != 0 {
		if _, err := savedActorRaw(r, "Block12", 12); err != nil {
			return err
		}
		for n := range b.Block {
			if b.BlockByteKnown(n) {
				b.Block[n] = 0
			}
		}
	}
	return nil
}

func nativeScalarMetadata(b sim.NativeActorBasis, unbound bool) (sim.NativeActorBasis, error) {
	if err := b.Validate(); err != nil {
		return b, err
	}
	if !unbound {
		for slot := range b.Scalars {
			if b.ScalarIsKnown(slot) {
				b.Scalars[slot] = 0
			}
		}
		for n := range b.Block {
			if b.BlockByteKnown(n) {
				b.Block[n] = 0
			}
		}
	}
	return b, nil
}

func matchNativeScalars(b *sim.NativeActorBasis, r *sav.DocumentRecordData) error {
	for slot := range b.Scalars {
		if !b.ScalarIsKnown(slot) {
			continue
		}
		if b.Scalars[slot] != 0 {
			return fmt.Errorf("native scalar metadata duplicates an ordinary value")
		}
		value, err := readNativeScalar(r, slot)
		if err != nil {
			return err
		}
		b.Scalars[slot] = value
	}
	if b.BlockKnown != 0 {
		raw, err := savedActorRaw(r, "Block12", 12)
		if err != nil {
			return err
		}
		for n := range b.Block {
			if !b.BlockByteKnown(n) {
				continue
			}
			if b.Block[n] != 0 {
				return fmt.Errorf("native Block metadata duplicates an ordinary byte")
			}
			b.Block[n] = raw[n+2]
		}
	}
	return b.Validate()
}
