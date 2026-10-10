package sim

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"reflect"
	"testing"
)

// Every value the saved-object layouts cover is written and read exactly as
// encoding/binary writes and reads it, including a read past the data's end.
func TestSavedObjectLayoutsMatchEncodingBinary(t *testing.T) {
	samples := []any{false, SavedObjectOrigin{}, SavedObjectToken{}, SavedObjectOwner{}, ItemEffect{}, SourceItemSpell{},
		SourceEquipment{}, SavedBookRoot{}, uint8(0), uint16(0), uint32(0), uint64(0)}
	rng := rand.New(rand.NewSource(11))
	for _, sample := range samples {
		typ := reflect.TypeOf(sample)
		size := binary.Size(sample)
		for n := 0; n < 300; n++ {
			v := reflect.New(typ)
			randomLayoutValue(rng, v.Elem())
			value := v.Elem().Interface()
			prefix := []byte{0xaa}
			want, err := binary.Append(prefix, binary.LittleEndian, value)
			if err != nil {
				t.Fatal(err)
			}
			if typ.Kind() != reflect.Uint8 && typ.Kind() != reflect.Uint16 && typ.Kind() != reflect.Uint32 && typ.Kind() != reflect.Uint64 {
				got, ok := appendSavedObjectLayout([]byte{0xaa}, value)
				if !ok || !bytes.Equal(got, want) {
					t.Fatalf("%s %+v: %x, want %x", typ, value, got, want)
				}
			}
			raw := make([]byte, size+3)
			rng.Read(raw)
			for _, data := range [][]byte{want[1:], raw, raw[:size-1]} {
				x, y := reflect.New(typ), reflect.New(typ)
				randomLayoutValue(rng, x.Elem())
				y.Elem().Set(x.Elem())
				ref := &savedObjectReader{data: data}
				if b := ref.take(uint64(size)); b != nil {
					_, ref.err = binary.Decode(b, binary.LittleEndian, x.Interface())
				}
				fast := &savedObjectReader{data: data}
				if typ.Kind() == reflect.Bool {
					continue
				}
				if !fast.readSavedObjectLayout(y.Interface()) {
					t.Fatalf("%s has no read layout", typ)
				}
				if !reflect.DeepEqual(x.Elem().Interface(), y.Elem().Interface()) || (ref.err == nil) != (fast.err == nil) || len(ref.data) != len(fast.data) {
					t.Fatalf("%s read %+v (err %v), want %+v (err %v)", typ, y.Elem().Interface(), fast.err, x.Elem().Interface(), ref.err)
				}
			}
		}
	}
}
