package game

import (
	"bytes"
	"encoding/binary"
	"testing"

	"againrom/pkg/sim"
)

func beforeNativeTrainingForm(t *testing.T, form []byte) []byte {
	t.Helper()
	if len(form) != 0 && form[0] == 107 {
		end := len(form)
		if end < 56 || string(form[end-4:]) != "CLS1" || form[end-5] >= 107 {
			t.Fatal("invalid native class compatibility footer")
		}
		span := uint64(binary.LittleEndian.Uint32(form[end-9:]))
		if span < 9 || span > uint64(end-43) {
			t.Fatal("invalid native class compatibility span")
		}
		start := end - 9 - int(span)
		count := uint64(binary.LittleEndian.Uint32(form[start:]))
		if count == 0 || count > 65535 || span != 4+5*count {
			t.Fatal("invalid native class compatibility population")
		}
		var prior uint32
		for n := uint64(0); n < count; n++ {
			o := start + 4 + 5*int(n)
			id := binary.LittleEndian.Uint32(form[o:])
			if n > 0 && id <= prior || form[o+4] > 1 {
				t.Fatal("invalid native class compatibility actor")
			}
			prior = id
		}
		out := bytes.Clone(form[:start])
		out[0] = form[end-5]
		form = out
	}
	if len(form) == 0 || form[0] != 105 {
		return form
	}
	end := len(form)
	if end < 75 || string(form[end-4:]) != "TRN1" || form[end-5] >= 105 {
		t.Fatal("invalid native training compatibility footer")
	}
	span := uint64(binary.LittleEndian.Uint32(form[end-9:]))
	if span < 32 || span > uint64(end-43) {
		t.Fatal("invalid native training compatibility span")
	}
	start := end - 9 - int(span)
	count := uint64(binary.LittleEndian.Uint32(form[start:]))
	if count == 0 || span != 4+28*count {
		t.Fatal("invalid native training compatibility population")
	}
	var prior uint32
	for n := uint64(0); n < count; n++ {
		id := binary.LittleEndian.Uint32(form[start+4+28*int(n):])
		if n > 0 && id <= prior {
			t.Fatal("invalid native training compatibility actor order")
		}
		prior = id
	}
	out := bytes.Clone(form[:start])
	out[0] = form[end-5]
	return out
}

func TestNativeTrainingCompatibilityPeelIsExact(t *testing.T) {
	w, err := sim.NewWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil,
		[]sim.Entity{{ID: 7, HP: 10, MaxHP: 10, TokenSize: 1}})
	if err != nil {
		t.Fatal(err)
	}
	base, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	form := bytes.Clone(base)
	form[0] = 105
	form = binary.LittleEndian.AppendUint32(form, 1)
	form = binary.LittleEndian.AppendUint32(form, 7)
	for _, level := range []uint32{0, 70, 45, 15, 82, 63} {
		form = binary.LittleEndian.AppendUint32(form, level)
	}
	form = binary.LittleEndian.AppendUint32(form, 32)
	form = append(form, base[0], 'T', 'R', 'N', '1')
	if got := beforeNativeTrainingForm(t, form); !bytes.Equal(got, base) {
		t.Fatal("training peel changed predecessor bytes")
	}
}
