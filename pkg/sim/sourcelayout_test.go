package sim

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"reflect"
	"testing"
)

func randomLayoutValue(rng *rand.Rand, v reflect.Value) {
	switch v.Kind() {
	case reflect.Bool:
		v.SetBool(rng.Intn(2) == 1)
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(int64(rng.Uint64()))
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(rng.Uint64())
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			randomLayoutValue(rng, v.Index(i))
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			randomLayoutValue(rng, v.Field(i))
		}
	default:
		panic("unexpected kind " + v.Kind().String())
	}
}

// The field-by-field source codecs write and read exactly what
// encoding/binary writes and reads, over every field and over arbitrary bytes.
func TestSourceLayoutsMatchEncodingBinary(t *testing.T) {
	if n := binary.Size(SourceEquipment{}); n != sourceEquipmentLen {
		t.Fatalf("SourceEquipment encodes to %d bytes, layout has %d", n, sourceEquipmentLen)
	}
	if n := binary.Size(SourceActor{}); n != sourceActorLen {
		t.Fatalf("SourceActor encodes to %d bytes, layout has %d", n, sourceActorLen)
	}
	if n := binary.Size(SourceBinding{}); n != sourceBindingLen {
		t.Fatalf("SourceBinding encodes to %d bytes, layout has %d", n, sourceBindingLen)
	}
	rng := rand.New(rand.NewSource(7))
	for n := 0; n < 500; n++ {
		var sb SourceBinding
		randomLayoutValue(rng, reflect.ValueOf(&sb).Elem())
		wantB := make([]byte, sourceBindingLen)
		_, _ = binary.Encode(wantB, binary.LittleEndian, sb)
		gotB := make([]byte, sourceBindingLen)
		putSourceBinding(gotB, &sb)
		if !bytes.Equal(gotB, wantB) {
			t.Fatalf("SourceBinding %+v: %x, want %x", sb, gotB, wantB)
		}
		rawB := make([]byte, sourceBindingLen)
		rng.Read(rawB)
		for _, b := range [][]byte{wantB, rawB} {
			var x, y SourceBinding
			randomLayoutValue(rng, reflect.ValueOf(&y).Elem())
			_, _ = binary.Decode(b, binary.LittleEndian, &x)
			getSourceBinding(b, &y)
			if x != y {
				t.Fatalf("SourceBinding read %+v, want %+v", y, x)
			}
		}

		var e SourceEquipment
		randomLayoutValue(rng, reflect.ValueOf(&e).Elem())
		want := make([]byte, sourceEquipmentLen)
		_, _ = binary.Encode(want, binary.LittleEndian, e)
		got := make([]byte, sourceEquipmentLen)
		putSourceEquipment(got, &e)
		if !bytes.Equal(got, want) {
			t.Fatalf("SourceEquipment %+v: %x, want %x", e, got, want)
		}
		raw := make([]byte, sourceEquipmentLen)
		rng.Read(raw)
		for _, b := range [][]byte{want, raw} {
			var x, y SourceEquipment
			randomLayoutValue(rng, reflect.ValueOf(&y).Elem())
			_, _ = binary.Decode(b, binary.LittleEndian, &x)
			getSourceEquipment(b, &y)
			if x != y {
				t.Fatalf("SourceEquipment read %+v, want %+v", y, x)
			}
		}

		var a SourceActor
		randomLayoutValue(rng, reflect.ValueOf(&a).Elem())
		want = make([]byte, sourceActorLen)
		_, _ = binary.Encode(want, binary.LittleEndian, a)
		got = make([]byte, sourceActorLen)
		putSourceActor(got, &a)
		if !bytes.Equal(got, want) {
			t.Fatalf("SourceActor %+v: %x, want %x", a, got, want)
		}
		raw = make([]byte, sourceActorLen)
		rng.Read(raw)
		for _, b := range [][]byte{want, raw} {
			var x, y SourceActor
			randomLayoutValue(rng, reflect.ValueOf(&y).Elem())
			_, _ = binary.Decode(b, binary.LittleEndian, &x)
			getSourceActor(b, &y)
			if x != y {
				t.Fatalf("SourceActor read %+v, want %+v", y, x)
			}
		}
	}
}
