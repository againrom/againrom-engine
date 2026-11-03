package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestCheckMoverBytesNamesEachShape plants one Mover write of each shape
// outside the producer and checks that the scan names it, and that the
// producer file and reads pass.
func TestCheckMoverBytesNamesEachShape(t *testing.T) {
	cases := []struct {
		name, src, want string
	}{
		{"an index write into a Mover field", "package game\nfunc f(m *sim.SavedActorMotion) { m.Mover[10] = 3 }\n", "writes an index of a Mover"},
		{"an index write into a [180]byte parameter", "package sim\nfunc f(m *[180]byte) { m[5], m[8] = 1, 2 }\n", "writes an index of a Mover"},
		{"an index write into a raw U154 block", "package game\nfunc f(r *sav.DocumentRecordData) { b, _ := savedActorRaw(r, \"U154\", 180); b[0] = 1 }\n", "writes an index of a Mover"},
		{"an increment of a Mover byte", "package sim\nfunc f(m *SavedActorMotion) { m.Mover[0x9d]++ }\n", "writes an index of a Mover"},
		{"a write into a city record's Mover", "package game\nfunc f(u *sav.CityUnitData) { u.Raw154[10] = 9 }\n", "writes an index of a Mover"},
		{"a binary Put into a Mover", "package sim\nfunc f(m *SavedActorMotion) { binary.LittleEndian.PutUint16(m.Mover[0x72:], 4) }\n", "puts a binary value into a Mover"},
		{"a binary Put into a local [MoverLen]byte", "package sim\nfunc f() { var m [MoverLen]byte; binary.LittleEndian.PutUint32(m[0xa0:], 1) }\n", "puts a binary value into a Mover"},
		{"a copy into a Mover", "package sim\nfunc f(m *SavedActorMotion) { copy(m.Mover[0x82:0x86], []byte{1, 2, 3, 4}) }\n", "copies into a Mover"},
		{"a clear of a slice of a Mover", "package sim\nfunc f(m *SavedActorMotion) { clear(m.Mover[0x7c:0x80]) }\n", "clears bytes of a Mover"},
		{"a whole copy into a raw mover", "package game\nfunc f(mover []byte, m [180]byte) { copy(mover, m[:]) }\n", "copies into a Mover"},
		{"a Mover literal holding a byte", "package game\nvar x = [180]byte{5: 0x41}\n", "builds a Mover literal holding a byte"},
		{"a Mover field literal holding a byte", "package game\nfunc f() sav.CityUnitData { return sav.CityUnitData{Raw154: []byte{0, 1}} }\n", "builds a Mover literal holding a byte"},
		{"reads, an empty literal and producer calls", "package game\nfunc f(m [180]byte, k *mapload.KeptActor) byte {\n\tk.Mover = [180]byte{}\n\tsim.EditMoverHead(m[:], func(h *sim.MoverHead) { h.Rate = 0 })\n\treturn m[5] + k.Mover[8]\n}\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, v := range CheckMoverBytes(map[string]string{"pkg/x/a.go": tc.src}) {
				got = append(got, v.Reason)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckMoverBytes = %v, want none", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0], tc.want) {
				t.Fatalf("CheckMoverBytes = %v, want one naming %q", got, tc.want)
			}
		})
	}
	if vs := CheckMoverBytes(map[string]string{MoverProducerFile: cases[1].src}); len(vs) != 0 {
		t.Fatalf("the producer file was refused: %v", vs)
	}
	if vs := CheckMoverBytes(nil); len(vs) != 1 {
		t.Fatalf("CheckMoverBytes(nil) = %v, want one violation", vs)
	}
}

// TestMoverBytesHaveOneProducer is the live-tree half.
func TestMoverBytesHaveOneProducer(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadMoverBytesSources(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range CheckMoverBytes(files) {
		t.Errorf("%s: %s", v.From, v.Reason)
	}
}
