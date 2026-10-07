package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"os"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// The fixture writes offsets independently, without Script, a binder or a
// production writer. Distinct ID/value/tag sentinels expose index substitution.
func triggerPayload1093() []byte {
	b := make([]byte, 12+3*796+4*796+2*184)
	put := func(off int, value uint32) { binary.LittleEndian.PutUint32(b[off:], value) }
	put(0, 3)
	node := func(off int, label string, op, id uint32, values, tags []uint32) {
		copy(b[off:], label)
		put(off+0x40, op)
		put(off+0x44, id)
		for i, value := range values {
			put(off+0x4c+4*i, value)
		}
		for i, tag := range tags {
			put(off+0x74+4*i, tag)
		}
	}
	node(4, "drop", 65538, 42, []uint32{17, 26, 123456}, []uint32{5, 6, 0})
	node(4+796, "group", 22, 65537, []uint32{5, 1, 65537, 999, 7, 2, 0xffffffff, 65543, 65537}, []uint32{2, 3, 4, 4, 9, 8, 99, 9, 3})
	node(4+2*796, "duplicate", 999, 42, nil, nil)
	co := 4 + 3*796
	put(co, 4)
	node(co+4, "zero", 65538, 1, []uint32{0}, []uint32{1})
	node(co+4+796, "external", 7, 65537, []uint32{10001, 10005, 11001}, []uint32{4, 4, 4})
	node(co+4+2*796, "unit", 5, 8, []uint32{2}, []uint32{4})
	node(co+4+3*796, "wide coords", 3, 9, []uint32{2, 0xffffffff, 7, 0xffffffff}, []uint32{4, 5, 6, 1})
	tr := co + 4 + 4*796
	put(tr, 2)
	tr += 4
	copy(b[tr:], "all references")
	put(tr+0x80, 1)
	put(tr+0x84, 65537)
	put(tr+0x88, 8)
	put(tr+0x8c, 9)
	put(tr+0x90, 0)
	put(tr+0x94, 444) // half-empty is not discarded
	put(tr+0x98, 65537)
	put(tr+0x9c, 42)
	put(tr+0xa0, 65537)
	put(tr+0xa8, 5)
	put(tr+0xac, 0xffffffff)
	put(tr+0xb0, 2)
	put(tr+0xb4, 7)
	tr += 184
	copy(b[tr:], "empty first pair")
	put(tr+0x98, 65537)
	return b
}

func inspectText1093(r ui.InspectionRecord) string {
	var out []string
	out = append(out, r.Label)
	out = append(out, r.Details...)
	for _, line := range r.Lines {
		out = append(out, line.Text)
	}
	return strings.Join(out, "\n")
}

func inspectRecord1093(t *testing.T, d *ui.InspectionDocument, kind string, index int) ui.InspectionRecord {
	t.Helper()
	for _, r := range d.Records {
		if r.Kind == kind && r.Index == index {
			return r
		}
	}
	t.Fatalf("missing %s:%d", kind, index)
	return ui.InspectionRecord{}
}

func TestMapInspection1093RawIDsSlotsAndUnresolvedData(t *testing.T) {
	b := synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Type7Payload: triggerPayload1093(), Units: []synth.ALMUnit{{X: 3*256 + 128, Y: 4 * 256}, {X: 5 * 256, Y: 6 * 256}}, Objects: []synth.ALMObject{{X: 8 * 256, Y: 9 * 256, Field12: 7}, {X: 10 * 256, Y: 11 * 256, Field12: 7}}})
	sections := rawSections1092(t, b)
	for i := 0; i < 2; i++ {
		binary.LittleEndian.PutUint16(sections[6][i*70+0x40:], 2)
		binary.LittleEndian.PutUint32(sections[6][i*70+0x42:], 5)
	}
	x := &MapInspector{tiles: &terrain.Tileset{}}
	d, err := x.inspect(b, "raw")
	if err != nil {
		t.Fatal(err)
	}
	r := inspectRecord1093(t, d, "Trigger", 0)
	text := inspectText1093(r)
	for _, want := range []string{"1: C1 <= C65537 [cmp 5]", "2: C8 unknown C9 [cmp 4294967295]", "3: C0 > C444 [cmp 2] / incomplete pair", "1: A65537 / Give group", "3: A65537 / Give group", "4: A0 / empty", "Once 7: at most once", "Ambiguous Action ID 42", "Unresolved Condition ID 444", "External hero ordinal 1", "External hero ordinal 5", "External static unit 11001", "Ambiguous Unit ID 2", "External static unit 65537", "Unresolved Structure ID 65543", "Unresolved player slot 65537", "Unresolved Unit ID 999", "Ambiguous Structure ID 7", "Player slot 1", "Par6 Unknown [tag 99] = 4294967295", "Item offset 2 / code 0x0e1a", "outside map", "Live state unknown"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in\n%s", want, text)
		}
	}
	if len(r.References) != 1 || len(r.References[0].Targets) != 2 || r.References[0].Targets[0].Cell != image.Pt(3, 4) || r.References[0].Targets[1].Cell != image.Pt(5, 6) {
		t.Fatalf("group identity or ambiguous reference fabricated: %+v", r.References)
	}
	if !strings.Contains(inspectText1093(inspectRecord1093(t, d, "Trigger", 1)), "1: A65537") {
		t.Fatal("empty first pair trigger discarded")
	}
	for i := 0; i < 3; i++ {
		inspectRecord1093(t, d, "Action", i)
	}
	for i := 0; i < 4; i++ {
		inspectRecord1093(t, d, "Condition", i)
	}
	if !strings.Contains(inspectText1093(inspectRecord1093(t, d, "Action", 0)), "Par2 None [tag 0] = 123456") {
		t.Fatal("tag-0 raw value disappeared")
	}
	if !bytes.Equal(d.SourceBytes, b) {
		t.Fatal("inspection mutated source")
	}
}

func TestMapInspection1093MalformedAndEmptyScriptStatus(t *testing.T) {
	for _, payload := range [][]byte{make([]byte, 12), {0xff, 0xff, 0xff, 0xff}, triggerPayload1093()[:100], append(make([]byte, 12), 99)} {
		b := synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Type7Payload: payload})
		x := &MapInspector{tiles: &terrain.Tileset{}}
		d, err := x.inspect(b, "script-status")
		if err != nil {
			t.Fatal(err)
		}
		valid := len(payload) == 12
		if valid != (d.ScriptNotice == "No authored triggers") {
			t.Fatalf("status %q", d.ScriptNotice)
		}
		for _, r := range d.Records {
			if r.Kind == "Section" && r.Section == 7 && !valid && r.Warning == "" {
				t.Fatal("decode failure not surfaced")
			}
			if r.Kind == "Trigger" || r.Kind == "Action" || r.Kind == "Condition" {
				t.Fatal("partial failed decode exposed as valid nodes")
			}
		}
		if !bytes.Equal(b, d.SourceBytes) {
			t.Fatal("malformed script bytes changed")
		}
	}
}

// Independent raw section walk over every installed map: counts, identities,
// comparison slots and raw argument arrays never come from the adapter.
func TestReleaseMapInspection1093RawTriggerPopulation(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: needs lawful install")
	}
	x, err := NewMapInspector(root)
	if err != nil {
		t.Fatal(err)
	}
	totals := [3]int{}
	for _, entry := range x.Maps {
		key := "loose/" + entry.Source
		if entry.FromArchive {
			key = "scenario/" + entry.Source
		}
		d, err := x.Open(key)
		if err != nil {
			t.Fatal(err)
		}
		b := rawSections1092(t, d.SourceBytes)[7]
		if len(b) == 0 {
			if !strings.Contains(d.ScriptNotice, "absent") {
				t.Fatal("absent not distinct from empty")
			}
			continue
		}
		word := func(off int) uint32 { return binary.LittleEndian.Uint32(b[off:]) }
		off := 0
		counts := [3]int{}
		for family, kind := range []string{"Action", "Condition", "Trigger"} {
			count := int(word(off))
			off += 4
			counts[family] = count
			totals[family] += count
			for i := 0; i < count; i++ {
				r := inspectRecord1093(t, d, kind, i)
				text := inspectText1093(r)
				if family < 2 {
					var values, tags [10]uint32
					for j := range values {
						values[j], tags[j] = word(off+0x4c+4*j), word(off+0x74+4*j)
					}
					for _, want := range []string{fmt.Sprintf("ID %d:", word(off+0x44)), fmt.Sprintf("opcode %d", word(off+0x40)), fmt.Sprintf("Raw values: %v", values), fmt.Sprintf("Raw tags: %v", tags)} {
						if !strings.Contains(text, want) {
							t.Fatalf("%s %s:%d lacks %s", key, kind, i, want)
						}
					}
					off += 796
				} else {
					for p := 0; p < 3; p++ {
						if !strings.Contains(text, fmt.Sprintf("%d: C%d ", p+1, word(off+0x80+8*p))) || !strings.Contains(text, fmt.Sprintf("C%d [cmp %d]", word(off+0x84+8*p), word(off+0xa8+4*p))) {
							t.Fatalf("%s Trigger:%d lost pair %d", key, i, p)
						}
					}
					for p := 0; p < 4; p++ {
						if !strings.Contains(text, fmt.Sprintf("%d: A%d /", p+1, word(off+0x98+4*p))) {
							t.Fatalf("%s Trigger:%d lost action%d", key, i, p)
						}
					}
					if !strings.Contains(text, fmt.Sprintf("Once %d:", word(off+0xb4))) {
						t.Fatal("once word changed")
					}
					off += 184
				}
			}
		}
		if off != len(b) {
			t.Fatalf("%s raw walk %d/%d", key, off, len(b))
		}
		if key == "scenario/10.alm" {
			if counts != [3]int{28, 16, 13} || len(b) != 37428 {
				t.Fatalf("M10 population %v size%d", counts, len(b))
			}
			t.Logf("M10 raw A/C/T=%v payload=%d; zero-first-pair trigger1 retained", counts, len(b))
		}
	}
	t.Logf("%d installed maps raw A/C/T totals=%v", len(x.Maps), totals)
}

func TestMapInspection1093DistanceRegionByteDomainRetainsRawPoint(t *testing.T) {
	for _, tc := range []struct {
		name         string
		w, h         int
		x, y, radius uint32
		square       bool
	}{
		{"wide-x", 512, 32, 300, 12, 4, false},
		{"wide-y", 32, 512, 12, 300, 4, false},
		{"narrow-center-wide-map", 512, 32, 44, 12, 4, false},
		{"wide-radius", 128, 128, 36, 51, 256, false},
		{"ordinary-square", 128, 128, 36, 51, 4, true},
		{"byte-edge", 256, 256, 255, 255, 4, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// One condition and no triggers/actions, written directly at the
			// file offsets; expected coordinates are not read from the decoder.
			p := make([]byte, 12+796)
			put := func(off int, v uint32) { binary.LittleEndian.PutUint32(p[off:], v) }
			put(4, 1)
			put(8+0x40, 3)
			put(8+0x44, 73)
			for i, v := range []uint32{1, tc.x, tc.y, tc.radius} {
				put(8+0x4c+4*i, v)
			}
			for i, v := range []uint32{4, 5, 6, 1} {
				put(8+0x74+4*i, v)
			}
			b := synth.ALM(synth.ALMOptions{Width: tc.w, Height: tc.h, Type7Payload: p})
			x := &MapInspector{tiles: &terrain.Tileset{}}
			d, err := x.inspect(b, "raw byte-domain fixture")
			if err != nil {
				t.Fatal(err)
			}
			r := inspectRecord1093(t, d, "Condition", 0)
			text := inspectText1093(r)
			for _, want := range []string{fmt.Sprintf("Par1 X [tag 5] = %d", tc.x), fmt.Sprintf("Par2 Y [tag 6] = %d", tc.y), fmt.Sprintf("Raw values: [1 %d %d %d 0 0 0 0 0 0]", tc.x, tc.y, tc.radius), "Raw tags: [4 5 6 1 0 0 0 0 0 0]"} {
				if !strings.Contains(text, want) {
					t.Fatalf("raw data lost %q", want)
				}
			}
			point, square := false, false
			for _, ref := range r.References {
				if strings.HasPrefix(ref.Label, "Cell ") {
					point = true
					if len(ref.Targets) != 1 || ref.Targets[0].Cell != image.Pt(int(tc.x), int(tc.y)) || ref.Targets[0].Size != image.Pt(1, 1) {
						t.Fatalf("authored point masked: %+v", ref)
					}
				}
				if strings.HasPrefix(ref.Label, "Distance square:") {
					square = true
				}
			}
			if !point || square != tc.square || strings.Contains(text, "Distance region withheld:") == tc.square {
				t.Fatalf("point=%v square=%v wantSquare=%v details=%s", point, square, tc.square, text)
			}
			if !bytes.Equal(b, d.SourceBytes) {
				t.Fatal("inspection changed source bytes")
			}
		})
	}
}
