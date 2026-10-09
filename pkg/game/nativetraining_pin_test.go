package game

import (
	"bytes"
	"encoding/binary"
	"testing"

	"againrom/pkg/sim"
)

func beforeNativeTrainingForm(t *testing.T, form []byte) []byte {
	t.Helper()
	// A held pursuit search wraps any earlier form; an earlier form never
	// held one, so its peel drops the records.
	if end := len(form); end > 0 && form[0] == 117 {
		if end < 40 || string(form[end-4:]) != "PRS1" || form[end-5] >= 117 {
			t.Fatal("invalid pursuit search footer")
		}
		span := uint64(binary.LittleEndian.Uint32(form[end-9:]))
		if span < 31 || span > uint64(end-9-34) || (span-4)%27 != 0 {
			t.Fatal("invalid pursuit search span", span)
		}
		base := form[end-5]
		form = bytes.Clone(form[:end-9-int(span)])
		form[0] = base
	}
	form = beforeTurnStateForm(t, form)
	for _, suffix := range []struct {
		version byte
		tag     string
		width   uint64
		prefix  uint64
	}{{112, "NLB1", 59, 4}, {111, "CPP1", 12, 5}, {110, "PPT1", 8, 4}, {109, "NAB1", 107, 4}, {108, "BSL1", 6, 4}} {
		if len(form) == 0 || form[0] != suffix.version {
			continue
		}
		end := len(form)
		if end < 56 || string(form[end-4:]) != suffix.tag || form[end-5] >= suffix.version {
			t.Fatal("invalid native actor compatibility footer", suffix.tag)
		}
		span := uint64(binary.LittleEndian.Uint32(form[end-9:]))
		player := suffix.tag == "PPT1" || suffix.tag == "CPP1"
		if span < suffix.prefix || !player && span < suffix.prefix+suffix.width || span > uint64(end-43) {
			t.Fatal("invalid native actor compatibility span", suffix.tag)
		}
		start := end - 9 - int(span)
		if suffix.tag == "CPP1" && form[start] > 1 {
			t.Fatal("invalid current Player presence")
		}
		count := uint64(binary.LittleEndian.Uint32(form[start+int(suffix.prefix)-4:]))
		if count == 0 && !player || count > 65535 || span != suffix.prefix+suffix.width*count {
			t.Fatal("invalid native actor compatibility population", suffix.tag)
		}
		var prior uint32
		for n := uint64(0); n < count; n++ {
			o := start + int(suffix.prefix+suffix.width*n)
			id := binary.LittleEndian.Uint32(form[o:])
			if n > 0 && id <= prior || player && id == 0 {
				t.Fatal("invalid native actor compatibility order", suffix.tag)
			}
			prior = id
			if suffix.tag == "CPP1" && form[start] == 0 && binary.LittleEndian.Uint32(form[o+8:]) != 0 {
				t.Fatal("absent current Participant has a value")
			}
		}
		out := bytes.Clone(form[:start])
		out[0] = form[end-5]
		form = out
	}
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
	turn := bytes.Clone(form)
	turn[0] = 115
	turn = binary.LittleEndian.AppendUint32(turn, 1)
	turn = binary.LittleEndian.AppendUint32(turn, 7)
	turn = append(turn, 1, 1, 16, 8, 7)
	turn = binary.LittleEndian.AppendUint32(turn, 13)
	turn = append(turn, 105, 'T', 'R', 'N', '1')
	if got := beforeNativeTrainingForm(t, turn); !bytes.Equal(got, base) {
		t.Fatal("turn and training peel changed predecessor bytes")
	}
}
