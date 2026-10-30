package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// checkInspector holds the tables the rules read: two unit classes, one item
// the name table holds, and structure kind 1 as a 2x2 footprint that takes
// every cell it covers.
func checkInspector() *MapInspector {
	row := func(slots map[int]int32) dbEntry {
		p := make([]int32, 40)
		for k, v := range slots {
			p[k] = v
		}
		return dbEntry{name: "row", params: p}
	}
	t := &mapload.Table{
		Units:  dbCollection{{}, row(map[int]int32{0x1d: 40, 0x1e: 2}), row(map[int]int32{0x1d: 41, 0x1e: 2})},
		Humans: dbCollection{{}, row(map[int]int32{0x10: 5, 0x18: 1234})},
	}
	t.Names = data.ItemNames{data.ItemCode(0x0101): "Sword"}
	t.Buildings = dbCollection{{}, {name: "Hut", params: []int32{2, 2, 0, 100, 0, -1}}}
	set := &terrain.StructureSet{}
	set.Classes[1] = &terrain.StructureClass{}
	return &MapInspector{table: t, structures: set}
}

// scriptNode is one action or condition of a synthetic type-7 payload.
type scriptNode struct {
	label        string
	opcode, id   uint32
	values, tags []uint32
}

// scriptPayloadWithDrop is scriptPayload for a fixture the party can start on
// and win: a script with no drop cell gets one after its actions, then an
// instant 4 and one trigger that runs it. A test of the drop-cell or the
// victory rule lays the payload out itself.
func scriptPayloadWithDrop(actions, conditions []scriptNode) []byte {
	if !slices.ContainsFunc(actions, func(n scriptNode) bool { return n.opcode == dropActionOpcode }) {
		actions = append(slices.Clone(actions), scriptNode{label: "Drop location", opcode: dropActionOpcode, id: 99, values: []uint32{16, 16}, tags: []uint32{5, 6}})
	}
	actions = append(slices.Clone(actions), scriptNode{label: "Mission complete", opcode: scriptOpWin, id: winActionID})
	b := scriptPayload(actions, conditions)
	b = binary.LittleEndian.AppendUint32(b[:len(b)-4], 1)
	return append(b, winTrigger("Win", winActionID)...)
}

// winActionID is the id of the fixtures' instant 4.
const winActionID = 97

// winTrigger is a 184-byte trigger record named name that runs action id
// once.
func winTrigger(name string, id uint32) []byte {
	rec := make([]byte, 184)
	copy(rec, name)
	binary.LittleEndian.PutUint32(rec[0x98:], id)
	binary.LittleEndian.PutUint32(rec[0xb4:], 1)
	return rec
}

func scriptPayload(actions, conditions []scriptNode) []byte {
	var b []byte
	put := func(v uint32) { b = binary.LittleEndian.AppendUint32(b, v) }
	for _, family := range [][]scriptNode{actions, conditions} {
		put(uint32(len(family)))
		for _, n := range family {
			rec := make([]byte, 796)
			copy(rec, n.label)
			binary.LittleEndian.PutUint32(rec[0x40:], n.opcode)
			binary.LittleEndian.PutUint32(rec[0x44:], n.id)
			for i, v := range n.values {
				binary.LittleEndian.PutUint32(rec[0x4c+4*i:], v)
			}
			for i, v := range n.tags {
				binary.LittleEndian.PutUint32(rec[0x74+4*i:], v)
			}
			b = append(b, rec...)
		}
	}
	return append(b, 0, 0, 0, 0)
}

// cleanOptions is a 32x32 map every rule passes: two units owned by the one
// side, with their own unit and group ids, and a script that holds only the
// drop cell.
func cleanOptions() synth.ALMOptions {
	return synth.ALMOptions{Width: 32, Height: 32, Type7Payload: scriptPayloadWithDrop(nil, nil), Units: []synth.ALMUnit{
		{X: 12*256 + 128, Y: 12*256 + 128, ClassID: 40, ClassSubID: 2},
		{X: 16*256 + 128, Y: 18*256 + 128, ClassID: 40, ClassSubID: 2},
	}}
}

// payloadOffsets maps each record type of the file b to the offset of its payload.
func payloadOffsets(t *testing.T, b []byte) map[uint32]int {
	t.Helper()
	offsets := map[uint32]int{}
	off := 20
	for n := binary.LittleEndian.Uint32(b[12:]); n > 0 && off+20 <= len(b); n-- {
		offsets[binary.LittleEndian.Uint32(b[off+12:])] = off + 20
		off += 20 + int(binary.LittleEndian.Uint32(b[off+8:]))
	}
	return offsets
}

// built is a synthetic map after the model's own edits and a raw patch.
func built(t *testing.T, o synth.ALMOptions, edit func(*mapedit.Editor), patch func(b []byte, payload func(typeID uint32) int)) *mapedit.Editor {
	t.Helper()
	m, err := mapedit.New(synth.ALM(o))
	if err != nil {
		t.Fatal(err)
	}
	if edit != nil {
		edit(m)
	}
	b := m.Bytes()
	spans := payloadOffsets(t, b)
	// Every unit gets owner slot 1, its own unit id and its own group id.
	for i := 0; i < len(o.Units); i++ {
		rec := spans[6] + i*70
		binary.LittleEndian.PutUint32(b[rec+0x14:], 1)
		binary.LittleEndian.PutUint16(b[rec+0x40:], uint16(10+i))
		binary.LittleEndian.PutUint32(b[rec+0x42:], uint32(20+i))
	}
	for i := range o.Objects {
		rec := spans[4] + i*20
		if o.Objects[i].Kind == 0x21 {
			rec = spans[4] + i*28
		}
		if o.Objects[i].Field0E == 0 {
			binary.LittleEndian.PutUint32(b[rec+0x0e:], 1)
		}
	}
	if patch != nil {
		patch(b, func(id uint32) int { return spans[id] })
	}
	m, err = mapedit.New(b)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func findings(t *testing.T, x *MapInspector, m *mapedit.Editor, source string) []ui.InspectionFinding {
	t.Helper()
	rows, err := x.CheckMap(m, source)
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

func ofRule(rows []ui.InspectionFinding, rule string) []ui.InspectionFinding {
	var out []ui.InspectionFinding
	for _, f := range rows {
		if f.Rule == rule {
			out = append(out, f)
		}
	}
	return out
}

func TestMapRuleTableIsDataWithAKindPerRow(t *testing.T) {
	errs := []string{"terrain-planes", "texts-folder", "map-decode", "mission-build", "record-outside", "script-reference", "owner-slot", "item-code", "structure-kind"}
	warnings := []string{"unit-terrain", "structure-overlap", "structure-border", "sack-duplicate", "drop-cell", "victory", "stock-link", "mod-message-text", "message-overlap"}
	var gotErrs, gotWarnings []string
	seen := map[string]bool{}
	for _, r := range mapRules {
		if seen[r.ID] || r.Authority == "" || r.Meaning == "" || r.run == nil {
			t.Fatalf("rule row %+v is incomplete or repeated", r.ID)
		}
		seen[r.ID] = true
		if r.Error {
			gotErrs = append(gotErrs, r.ID)
		} else {
			gotWarnings = append(gotWarnings, r.ID)
		}
	}
	if !slices.Equal(errs, gotErrs) || !slices.Equal(warnings, gotWarnings) {
		t.Fatalf("rule kinds: errors %v warnings %v", gotErrs, gotWarnings)
	}
	// A rule moves between kinds by its row alone.
	x := checkInspector()
	m := built(t, cleanOptions(), nil, func(b []byte, payload func(uint32) int) {
		binary.LittleEndian.PutUint32(b[payload(6)+0x14:], 0)
	})
	if rows := findings(t, x, m, "x"); len(rows) != 1 || !rows[0].Error || rows[0].Rule != "owner-slot" {
		t.Fatalf("default table: %+v", rows)
	}
	moved := slices.Clone(mapRules)
	for i := range moved {
		if moved[i].ID == "owner-slot" {
			moved[i].Error = false
		}
	}
	x.rules = moved
	if rows := findings(t, x, m, "x"); len(rows) != 1 || rows[0].Error {
		t.Fatalf("owner-slot moved to a warning: %+v", rows)
	}
	x.rules = []mapRule{}
	if rows := findings(t, x, m, "x"); len(rows) != 0 {
		t.Fatalf("empty table: %+v", rows)
	}
}

func TestMapRulesFindTheirDefectAndNothingInTheCleanMap(t *testing.T) {
	x := checkInspector()
	if rows := findings(t, x, built(t, cleanOptions(), nil, nil), "x"); len(rows) != 0 {
		t.Fatalf("the clean map reports %+v", rows)
	}
	ground := func(m *mapedit.Editor, cx, cy uint32, item uint16) {
		if _, err := m.Place(mapedit.KindLoot, m.GroundSack(cx*256+128, cy*256+128, item)); err != nil {
			t.Fatal(err)
		}
	}
	hut := func(cx, cy uint32, kind uint32) synth.ALMObject {
		return synth.ALMObject{X: cx*256 + 128, Y: cy*256 + 128, Kind: kind, Field12: 7}
	}
	with := func(f func(*synth.ALMOptions)) synth.ALMOptions {
		o := cleanOptions()
		f(&o)
		return o
	}
	scriptWith := func(actions, conditions []scriptNode) synth.ALMOptions {
		return with(func(o *synth.ALMOptions) { o.Type7Payload = scriptPayloadWithDrop(actions, conditions) })
	}
	alive := func(unit uint32) []scriptNode {
		return []scriptNode{{label: "alive", opcode: 5, id: 3, values: []uint32{unit}, tags: []uint32{4}}}
	}
	for _, tc := range []struct {
		name, rule string
		kind       string
		index      int
		defect     func() *mapedit.Editor
		control    func() *mapedit.Editor
	}{
		{"a unit outside the map", "record-outside", "Unit", 1,
			func() *mapedit.Editor {
				return built(t, with(func(o *synth.ALMOptions) { o.Units[1].X = 40*256 + 5 }), nil, nil)
			},
			func() *mapedit.Editor { return built(t, cleanOptions(), nil, nil) }},
		{"a structure outside the map", "record-outside", "Structure", 0,
			func() *mapedit.Editor {
				return built(t, with(func(o *synth.ALMOptions) { o.Objects = []synth.ALMObject{hut(60, 12, 1)} }), nil, nil)
			},
			func() *mapedit.Editor {
				return built(t, with(func(o *synth.ALMOptions) { o.Objects = []synth.ALMObject{hut(14, 12, 1)} }), nil, nil)
			}},
		{"a structure footprint that leaves the map", "record-outside", "Structure", 0,
			func() *mapedit.Editor {
				return built(t, with(func(o *synth.ALMOptions) { o.Objects = []synth.ALMObject{hut(31, 31, 1)} }), nil, nil)
			},
			func() *mapedit.Editor {
				return built(t, with(func(o *synth.ALMOptions) { o.Objects = []synth.ALMObject{hut(22, 22, 1)} }), nil, nil)
			}},
		{"a structure footprint that meets the border ring", "structure-border", "Structure", 0,
			func() *mapedit.Editor {
				return built(t, with(func(o *synth.ALMOptions) { o.Objects = []synth.ALMObject{hut(23, 12, 1)} }), nil, nil)
			},
			func() *mapedit.Editor {
				return built(t, with(func(o *synth.ALMOptions) { o.Objects = []synth.ALMObject{hut(22, 12, 1)} }), nil, nil)
			}},
		{"a sack outside the map", "record-outside", "Sack", 0,
			func() *mapedit.Editor {
				return built(t, cleanOptions(), func(m *mapedit.Editor) { ground(m, 40, 12, 0x0101) }, nil)
			},
			func() *mapedit.Editor {
				return built(t, cleanOptions(), func(m *mapedit.Editor) { ground(m, 14, 14, 0x0101) }, nil)
			}},
		{"a script naming a unit the map lacks", "script-reference", "Condition", 0,
			func() *mapedit.Editor { return built(t, scriptWith(nil, alive(77)), nil, nil) },
			func() *mapedit.Editor { return built(t, scriptWith(nil, alive(10)), nil, nil) }},
		{"a script naming a unit above the hero band", "script-reference", "Condition", 0,
			func() *mapedit.Editor { return built(t, scriptWith(nil, alive(12345)), nil, nil) },
			func() *mapedit.Editor { return built(t, scriptWith(nil, alive(10)), nil, nil) }},
		{"a script naming a structure the map lacks", "script-reference", "Condition", 0,
			func() *mapedit.Editor {
				return built(t, with(func(o *synth.ALMOptions) {
					o.Objects = []synth.ALMObject{hut(14, 12, 1)}
					o.Type7Payload = scriptPayloadWithDrop(nil, []scriptNode{{label: "structure", opcode: 21, id: 3, values: []uint32{99}, tags: []uint32{9}}})
				}), nil, nil)
			},
			func() *mapedit.Editor {
				return built(t, with(func(o *synth.ALMOptions) {
					o.Objects = []synth.ALMObject{hut(14, 12, 1)}
					o.Type7Payload = scriptPayloadWithDrop(nil, []scriptNode{{label: "structure", opcode: 21, id: 3, values: []uint32{7}, tags: []uint32{9}}})
				}), nil, nil)
			}},
		{"a script naming a group no unit holds", "script-reference", "Action", 0,
			func() *mapedit.Editor {
				return built(t, scriptWith([]scriptNode{{label: "group", opcode: 6, id: 4, values: []uint32{99}, tags: []uint32{2}}}, nil), nil, nil)
			},
			func() *mapedit.Editor {
				return built(t, scriptWith([]scriptNode{{label: "group", opcode: 6, id: 4, values: []uint32{21}, tags: []uint32{2}}}, nil), nil, nil)
			}},
		{"a unit with an owner outside the roster", "owner-slot", "Unit", 0,
			func() *mapedit.Editor {
				return built(t, cleanOptions(), nil, func(b []byte, payload func(uint32) int) {
					binary.LittleEndian.PutUint32(b[payload(6)+0x14:], 2)
				})
			},
			func() *mapedit.Editor { return built(t, cleanOptions(), nil, nil) }},
		{"a structure with an owner outside the roster", "owner-slot", "Structure", 0,
			func() *mapedit.Editor {
				return built(t, with(func(o *synth.ALMOptions) { o.Objects = []synth.ALMObject{hut(14, 12, 1)} }), nil, func(b []byte, payload func(uint32) int) {
					binary.LittleEndian.PutUint32(b[payload(4)+0x0e:], 0)
				})
			},
			func() *mapedit.Editor {
				return built(t, with(func(o *synth.ALMOptions) { o.Objects = []synth.ALMObject{hut(14, 12, 1)} }), nil, nil)
			}},
		{"a sack of an item the tables lack", "item-code", "Sack", 0,
			func() *mapedit.Editor {
				return built(t, cleanOptions(), func(m *mapedit.Editor) { ground(m, 14, 14, 0x0999) }, nil)
			},
			func() *mapedit.Editor {
				return built(t, cleanOptions(), func(m *mapedit.Editor) { ground(m, 14, 14, 0x0101) }, nil)
			}},
		{"a structure kind the tables lack", "structure-kind", "Structure", 0,
			func() *mapedit.Editor {
				return built(t, with(func(o *synth.ALMOptions) { o.Objects = []synth.ALMObject{hut(14, 12, 77)} }), nil, nil)
			},
			func() *mapedit.Editor {
				return built(t, with(func(o *synth.ALMOptions) { o.Objects = []synth.ALMObject{hut(14, 12, 1)} }), nil, nil)
			}},
		{"a unit on a cell it cannot enter", "unit-terrain", "Unit", 0,
			func() *mapedit.Editor {
				return built(t, with(func(o *synth.ALMOptions) {
					o.Overlay = make([]uint8, 32*32)
					o.Overlay[12*32+12] = 1
				}), nil, nil)
			},
			func() *mapedit.Editor {
				return built(t, with(func(o *synth.ALMOptions) {
					o.Overlay = make([]uint8, 32*32)
					o.Overlay[20*32+20] = 1
				}), nil, nil)
			}},
		{"structures over the same cells", "structure-overlap", "Structure", 1,
			func() *mapedit.Editor {
				return built(t, with(func(o *synth.ALMOptions) { o.Objects = []synth.ALMObject{hut(14, 12, 1), hut(15, 13, 1)} }), nil, nil)
			},
			func() *mapedit.Editor {
				return built(t, with(func(o *synth.ALMOptions) { o.Objects = []synth.ALMObject{hut(14, 12, 1), hut(20, 12, 1)} }), nil, nil)
			}},
		{"two sacks on one cell", "sack-duplicate", "Sack", 1,
			func() *mapedit.Editor {
				return built(t, cleanOptions(), func(m *mapedit.Editor) { ground(m, 14, 14, 0x0101); ground(m, 14, 14, 0x0101) }, nil)
			},
			func() *mapedit.Editor {
				return built(t, cleanOptions(), func(m *mapedit.Editor) { ground(m, 14, 14, 0x0101); ground(m, 15, 14, 0x0101) }, nil)
			}},
		{"a drop cell outside the map", "drop-cell", "Action", 0,
			func() *mapedit.Editor {
				return built(t, scriptWith([]scriptNode{{label: "drop", opcode: 0x10002, id: 1, values: []uint32{40, 40}, tags: []uint32{5, 6}}}, nil), nil, nil)
			},
			func() *mapedit.Editor {
				return built(t, scriptWith([]scriptNode{{label: "drop", opcode: 0x10002, id: 1, values: []uint32{14, 14}, tags: []uint32{5, 6}}}, nil), nil, nil)
			}},
		{"a drop cell on blocked ground", "drop-cell", "Action", 0,
			func() *mapedit.Editor {
				o := scriptWith([]scriptNode{{label: "drop", opcode: 0x10002, id: 1, values: []uint32{14, 14}, tags: []uint32{5, 6}}}, nil)
				o.Overlay = make([]uint8, 32*32)
				o.Overlay[14*32+14] = 1
				return built(t, o, nil, nil)
			},
			func() *mapedit.Editor {
				return built(t, scriptWith([]scriptNode{{label: "drop", opcode: 0x10002, id: 1, values: []uint32{14, 14}, tags: []uint32{5, 6}}}, nil), nil, nil)
			}},
		{"a unit whose stock link names another unit's stock", "stock-link", "Unit", 0,
			func() *mapedit.Editor { return stockLinked(t, 11).Editor },
			func() *mapedit.Editor { return stockLinked(t, 10).Editor }},
		{"a unit whose stock link names no record", "stock-link", "Unit", 0,
			func() *mapedit.Editor { return stockLinked(t, 10).withLink(t, 9) },
			func() *mapedit.Editor { return stockLinked(t, 10).Editor }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ofRule(findings(t, x, tc.defect(), "scenario/10.alm"), tc.rule)
			if len(got) != 1 || got[0].Kind != tc.kind || got[0].Index != tc.index {
				t.Fatalf("%s findings: %+v; want one on %s #%d", tc.rule, got, tc.kind, tc.index)
			}
			if rows := findings(t, x, tc.control(), "scenario/10.alm"); len(rows) != 0 {
				t.Fatalf("the map without the defect reports %+v", rows)
			}
		})
	}
	// The party is not a placed unit.
	if rows := findings(t, x, built(t, scriptWith(nil, alive(10001)), nil, nil), "x"); len(rows) != 0 {
		t.Fatalf("a condition naming the party: %+v", rows)
	}
}

type linkedMap struct{ *mapedit.Editor }

// stockLinked is the clean map with one stock record owned by unit id owner and
// unit 0's stock link naming it.
func stockLinked(t *testing.T, owner uint32) linkedMap {
	t.Helper()
	return linkedMap{built(t, cleanOptions(), func(m *mapedit.Editor) {
		rec := m.GroundSack(0, 0, 0x0101)
		binary.LittleEndian.PutUint32(rec[0x04:], owner)
		if _, err := m.Place(mapedit.KindLoot, rec); err != nil {
			t.Fatal(err)
		}
	}, func(b []byte, payload func(uint32) int) {
		binary.LittleEndian.PutUint32(b[payload(6)+0x18:], 1)
	})}
}

// withLink returns the map with unit 0's stock link set to link.
func (l linkedMap) withLink(t *testing.T, link uint32) *mapedit.Editor {
	t.Helper()
	b := l.Bytes()
	binary.LittleEndian.PutUint32(b[payloadOffsets(t, b)[6]+0x18:], link)
	m, err := mapedit.New(b)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func worldFS(t *testing.T, files ...synth.File) *vfs.FS {
	t.Helper()
	path := filepath.Join(t.TempDir(), WorldArchive)
	if err := os.WriteFile(path, synth.Archive(files), 0o600); err != nil {
		t.Fatal(err)
	}
	fsys, err := vfs.Open([]string{path}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return fsys
}

func TestMapReportRunsTheMissionBuildAndNamesItsRefusal(t *testing.T) {
	// A unit definition row too short for the columns the build reads is a
	// refusal of the mission build; the same map over whole rows builds.
	x := checkInspector()
	m := built(t, cleanOptions(), nil, nil)
	if rows := findings(t, x, m, "scenario/10.alm"); len(rows) != 0 {
		t.Fatalf("whole rows: %+v", rows)
	}
	x.table = authoringTable()
	got := ofRule(findings(t, x, m, "scenario/10.alm"), "mission-build")
	if len(got) != 1 || !got[0].Error || got[0].Kind != "" || !strings.Contains(got[0].Text, "slots 0 to 37 are required") {
		t.Fatalf("short rows: %+v", got)
	}
}

func TestMapReportRunsTheInstalledTerrainRefusal(t *testing.T) {
	x := checkInspector()
	x.archives = &Archives{Containers: worldFS(t, synth.File{Path: "data/map.reg", Data: synth.Reg(0, nil)})}
	small := built(t, synth.ALMOptions{Width: 4, Height: 4, Type7Payload: make([]byte, 12)}, nil, nil)
	got := ofRule(findings(t, x, small, "x"), "terrain-planes")
	if len(got) != 1 || got[0].Kind != "" || !got[0].Error || !strings.Contains(got[0].Text, "dimensions in 8..256") {
		t.Fatalf("a 4x4 map: %+v", got)
	}
	// The mission-build rule leaves the refusal to the rule that names it.
	if rows := findings(t, x, small, "x"); len(ofRule(rows, "mission-build")) != 0 {
		t.Fatalf("the refusal is reported twice: %+v", rows)
	}
	tiles := synth.ALMOptions{Width: 32, Height: 32, Type7Payload: make([]byte, 12)}
	if rows := findings(t, x, built(t, tiles, nil, nil), "x"); len(ofRule(rows, "terrain-planes")) != 0 {
		t.Fatalf("a 32x32 map: %+v", rows)
	}
	// Terrain pairs of strip groups 13 to 15 were never initialised.
	bad := built(t, tiles, nil, func(b []byte, payload func(uint32) int) {
		binary.LittleEndian.PutUint16(b[payload(1)+2*17:], 13<<6)
	})
	if got := ofRule(findings(t, x, bad, "x"), "terrain-planes"); len(got) != 1 || !strings.Contains(got[0].Text, "uninitialized terrain pair") {
		t.Fatalf("a tile of strip group 13: %+v", got)
	}
}

func TestMapReportIsMemoisedOnTheBytesAndGatesBothSaves(t *testing.T) {
	x := checkInspector()
	m := built(t, cleanOptions(), nil, func(b []byte, payload func(uint32) int) {
		binary.LittleEndian.PutUint32(b[payload(6)+0x14:], 5)
	})
	s := &mapInspectionEdits{inspector: x, model: m, source: "scenario/20.alm", saved: m.Bytes()}
	first := s.Report()
	if len(first) != 1 || &first[0] != &s.Report()[0] {
		t.Fatalf("report %+v is not kept for the same bytes", first)
	}
	target := filepath.Join(t.TempDir(), "out.alm")
	_, err := s.SaveAs(target)
	var refused *ui.ReportRefusal
	if r, ok := err.(*ui.ReportRefusal); !ok || len(r.Findings) != 1 {
		t.Fatalf("Save As: %v", err)
	} else {
		refused = r
	}
	if !strings.Contains(refused.Error(), "1 errors and 0 warnings; nothing was saved") {
		t.Fatalf("refusal text %q", refused.Error())
	}
	if _, err := os.Stat(target); err == nil {
		t.Fatal("a refused Save As wrote a file")
	}
	dir := writeEditorMod(t, t.TempDir(), editorModMissions)
	if err := x.SetMod(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveIntoMod(); err == nil || !strings.Contains(err.Error(), "nothing was saved") {
		t.Fatalf("Save into the mod: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "mission")); err == nil {
		t.Fatal("a refused Save into the mod wrote into the mod")
	}
	// An edit changes the bytes, so the report is made again.
	if err := s.model.SetUnitField(0, "owner", 1); err != nil {
		t.Fatal(err)
	}
	if rows := s.Report(); len(rows) != 0 {
		t.Fatalf("after the owner was set: %+v", rows)
	}
	if got, err := s.SaveAs(target); err != nil || got != target {
		t.Fatalf("Save As of the repaired map: %q %v", got, err)
	}
}

func TestMapReportKeepsInstalledFactsAsWarnings(t *testing.T) {
	// A report of only warnings does not refuse a save.
	x := checkInspector()
	m := built(t, cleanOptions(), func(m *mapedit.Editor) {
		for i := 0; i < 2; i++ {
			if _, err := m.Place(mapedit.KindLoot, m.GroundSack(14*256+128, 14*256+128, 0x0101)); err != nil {
				t.Fatal(err)
			}
		}
	}, nil)
	s := &mapInspectionEdits{inspector: x, model: m, source: "x", saved: m.Bytes()}
	rows := s.Report()
	if errs, warnings := ui.CountFindings(rows); errs != 0 || warnings != 1 {
		t.Fatalf("%+v", rows)
	}
	if err := s.gate(false); err != nil {
		t.Fatal(err)
	}
}

// The unit-terrain warning for a ring unit says the game removes it.
func TestMapRuleUnitInTheBorderRingSaysTheGameRemovesIt(t *testing.T) {
	x := checkInspector()
	const ring = "the game removes a unit placed there when the mission loads"
	for _, c := range []struct {
		name string
		x, y int
		ring bool
	}{
		{"a unit in the ring", 3, 3, true},
		{"a unit on blocked ground inside the ring", 12, 12, false},
	} {
		o := cleanOptions()
		o.Units = []synth.ALMUnit{{X: uint32(c.x*256 + 128), Y: uint32(c.y*256 + 128), ClassID: 40, ClassSubID: 2}}
		o.Overlay = make([]uint8, 32*32)
		o.Overlay[c.y*32+c.x] = 1
		rows := ofRule(findings(t, x, built(t, o, nil, nil), "x"), "unit-terrain")
		if len(rows) != 1 || strings.Contains(rows[0].Text, ring) != c.ring {
			t.Errorf("%s: unit-terrain findings %+v, want one, naming the removal: %v", c.name, rows, c.ring)
		}
	}
}
