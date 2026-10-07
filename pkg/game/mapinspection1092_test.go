package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// Independent raw walk: no alm decoder, loader, UI record builder or rendering
// result supplies expected counts/coordinates. Constants are the framed ALM
// field map; the second comparison below pins the full shipped population.
func rawSections1092(t *testing.T, b []byte) map[uint32][]byte {
	t.Helper()
	sections := map[uint32][]byte{}
	count := int(binary.LittleEndian.Uint32(b[12:16]))
	for off, i := 20, 0; i < count; i++ {
		n := int(binary.LittleEndian.Uint32(b[off+8 : off+12]))
		kind := binary.LittleEndian.Uint32(b[off+12 : off+16])
		sections[kind] = b[off+20 : off+20+n]
		off += 20 + n
	}
	return sections
}

func rawInspection1092(t *testing.T, b []byte) map[string][]image.Point {
	sections := rawSections1092(t, b)
	m := sections[0]
	w := int(binary.LittleEndian.Uint32(m[:4]))
	xy := func(b []byte) image.Point {
		return image.Pt(int(binary.LittleEndian.Uint32(b[:4])/256), int(binary.LittleEndian.Uint32(b[4:8])/256))
	}
	want := map[string][]image.Point{}
	for b := sections[6]; len(b) > 0; b = b[70:] {
		want["Unit"] = append(want["Unit"], xy(b))
	}
	for b := sections[4]; len(b) > 0; {
		want["Structure"] = append(want["Structure"], xy(b))
		n := 20
		if binary.LittleEndian.Uint32(b[8:12]) == 33 {
			n += 8
		}
		b = b[n:]
	}
	for i, b := range sections[3] {
		if b != 0 {
			want["Scenery"] = append(want["Scenery"], image.Pt(i%w, i/w))
		}
	}
	for b := sections[8]; len(b) > 0; {
		n := int(binary.LittleEndian.Uint32(b[:4]))
		kind := "Stock"
		p := image.Point{}
		if binary.LittleEndian.Uint32(b[4:8]) == 0 {
			kind = "Sack"
			p = xy(b[8:16])
		}
		want[kind] = append(want[kind], p)
		b = b[20+n*10:]
	}
	return want
}

func assertInspectionFit1092(t *testing.T, e *ui.MapEditor) {
	t.Helper()
	sections := rawSections1092(t, e.Document().SourceBytes)
	w := int(binary.LittleEndian.Uint32(sections[0]))
	h := int(binary.LittleEndian.Uint32(sections[0][4:]))
	minY, maxY := h*32+128, -128
	for y := 0; y <= h; y++ {
		for x := 0; x <= w; x++ {
			alt := int(int8(sections[2][min(y, h-1)*w+min(x, w-1)]))
			v := y*32 - alt
			minY, maxY = min(minY, v), max(maxY, v)
		}
	}
	worldH := h * 32
	if e.Document().Viewer.Mode() == ui.ModeDisplaced {
		worldH = maxY - minY
	}
	for _, size := range []image.Point{image.Pt(640, 480), image.Pt(1280, 800)} {
		e.Layout(size.X, size.Y)
		e.Fit()
		c := e.Document().Viewer.Camera()
		if c.WorldW() != float64(w*32) || c.WorldH() != float64(worldH) {
			t.Fatal("camera differs from raw authored extent")
		}
		for _, p := range []image.Point{{0, 0}, {w * 32, 0}, {0, worldH}, {w * 32, worldH}} {
			x, y := c.WorldToScreen(float64(p.X), float64(p.Y))
			if x < -1e-8 || y < -1e-8 || x > float64(size.X-320)+1e-8 || y > float64(size.Y)+1e-8 {
				t.Fatalf("%s %dx%d at%v edge%v projects outside canvas: %v,%v zoom%v", e.Document().Source, w, h, size, p, x, y, c.Zoom)
			}
		}
		name := filepath.Base(e.Document().Source)
		if size.X == 640 && (strings.EqualFold(name, "FORESTER.ALM") || strings.EqualFold(name, "Beast.ALM")) {
			t.Logf("Fit %s raw %dx%d: all edges inside320x480 zoom=%v", e.Document().Source, w, h, c.Zoom)
		}
	}
}

func TestReleaseMapInspection1092EveryAuthoredPlacement(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: needs lawful install")
	}
	x, err := NewMapInspector(root)
	if err != nil {
		t.Fatal(err)
	}
	wantMaps := 38
	if x.font.Selector != 0 {
		wantMaps = 34
	}
	if len(x.Maps) != wantMaps {
		t.Fatalf("map catalogue %d, want %d", len(x.Maps), wantMaps)
	}
	totals := map[string]int{}
	art := map[string]int{}
	multiItemWitness := false
	for _, entry := range x.Maps {
		key := "loose/" + entry.Source
		if entry.FromArchive {
			key = "scenario/" + entry.Source
		}
		d, err := x.Open(key)
		if err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		want := rawInspection1092(t, d.SourceBytes)
		got := map[string][]image.Point{}
		for _, r := range d.Records {
			if _, ok := want[r.Kind]; ok {
				got[r.Kind] = append(got[r.Kind], r.Cell)
				totals[r.Kind]++
				if r.Preview != nil {
					art[r.Kind]++
				} else if r.Spatial && r.Warning == "" {
					t.Fatalf("%s unmarked missing art: %+v", key, r)
				}
			}
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%s raw authored positions/counts differ", key)
		}
		for b, i := rawSections1092(t, d.SourceBytes)[8], 0; len(b) > 0; i++ {
			n := int(binary.LittleEndian.Uint32(b))
			gold := binary.LittleEndian.Uint32(b[16:])
			owner := binary.LittleEndian.Uint32(b[4:])
			var rec *ui.InspectionRecord
			for j := range d.Records {
				r := &d.Records[j]
				if (r.Kind == "Sack" || r.Kind == "Stock") && r.Index == i {
					rec = r
					break
				}
			}
			if rec == nil {
				t.Fatalf("%s missing loot record %d", key, i)
			}
			details := strings.Join(rec.Details, "\n")
			if !strings.Contains(details, fmt.Sprintf("Owner: %d  Gold: %d", owner, gold)) || !strings.Contains(details, fmt.Sprintf("Items: %d", n)) {
				t.Fatalf("%s loot %d header not inspectable", key, i)
			}
			for k := 0; k < n; k++ {
				e := b[20+k*10:]
				code, raw, link := binary.LittleEndian.Uint16(e), binary.LittleEndian.Uint16(e[4:]), binary.LittleEndian.Uint32(e[6:])
				if !strings.Contains(details, fmt.Sprintf("%d: code %#04x", k, code)) || !strings.Contains(details, fmt.Sprintf("raw04 %d enchantment %d", raw, link)) {
					t.Fatalf("%s loot %d element %d not inspectable", key, i, k)
				}
			}
			if owner == 0 && n > 3 && !multiItemWitness {
				t.Logf("multi-item UI witness %s Sack:%d items=%d", key, i, n)
				multiItemWitness = true
			}
			b = b[20+n*10:]
		}
		before := append([]byte(nil), d.SourceBytes...)
		e := x.Editor()
		if err := e.Open(key); err != nil {
			t.Fatal(err)
		}
		assertInspectionFit1092(t, e)
		for i := range e.Document().Records {
			e.Select(i, false)
		}
		if !bytes.Equal(before, e.Document().SourceBytes) {
			t.Fatal("inspection changed source bytes")
		}
		after, err := x.Open(key)
		if err != nil || !bytes.Equal(after.SourceBytes, before) {
			t.Fatalf("source changed: %v", err)
		}
		if key == "scenario/10.alm" || key == "scenario/20.alm" {
			t.Logf("%s counts U/B/S/Stock/T=%d/%d/%d/%d/%d first-unit=%v", key, len(want["Unit"]), len(want["Structure"]), len(want["Sack"]), len(want["Stock"]), len(want["Scenery"]), want["Unit"][0])
		}
	}
	t.Logf("%d maps independent placement totals=%v actual art=%v", len(x.Maps), totals, art)
	wantTotals := map[string]int{"Unit": 8094, "Structure": 3141, "Sack": 137, "Stock": 43, "Scenery": 71099}
	if wantMaps == 34 {
		wantTotals = map[string]int{"Unit": 3991, "Structure": 1681, "Sack": 133, "Stock": 43, "Scenery": 54954}
	}
	if !reflect.DeepEqual(totals, wantTotals) {
		t.Fatalf("raw placement census %v want %v", totals, wantTotals)
	}
	for _, k := range []string{"Unit", "Structure", "Sack", "Scenery"} {
		if art[k] == 0 {
			t.Fatalf("no actual %s art", k)
		}
	}
	if !multiItemWitness {
		t.Fatal("no multi-item installed sack witness")
	}
}

func TestMapInspection1092UnknownDuplicateAndMalformedSectionsStayListed(t *testing.T) {
	b := synth.ALM(synth.ALMOptions{Width: 32, Height: 32})
	appendSection := func(id uint32, payload []byte) {
		h := make([]byte, 20)
		binary.LittleEndian.PutUint32(h, 7)
		binary.LittleEndian.PutUint32(h[4:], 20)
		binary.LittleEndian.PutUint32(h[8:], uint32(len(payload)))
		binary.LittleEndian.PutUint32(h[12:], id)
		b = append(append(b, h...), payload...)
		binary.LittleEndian.PutUint32(b[12:], binary.LittleEndian.Uint32(b[12:])+1)
	}
	appendSection(41, []byte{1, 2, 3, 4})
	appendSection(8, []byte{1}) // zero authored count with trailing payload
	x := &MapInspector{tiles: &terrain.Tileset{}}
	d, err := x.inspect(b, "sections")
	if err != nil {
		t.Fatal(err)
	}
	var unknown, duplicate, malformedScript, malformedLoot bool
	for _, r := range d.Records {
		if r.Kind != "Section" {
			continue
		}
		unknown = unknown || r.Section == 41 && strings.Contains(r.Warning, "Unknown section")
		duplicate = duplicate || strings.Contains(r.Warning, "Overridden duplicate")
		malformedScript = malformedScript || r.Section == 7 && r.Warning != ""
		malformedLoot = malformedLoot || r.Section == 8 && strings.Contains(r.Warning, "walk consumed")
	}
	if !unknown || !duplicate || !malformedScript || !malformedLoot || !bytes.Equal(b, d.SourceBytes) {
		t.Fatalf("section accounting unknown=%v duplicate=%v script=%v loot=%v", unknown, duplicate, malformedScript, malformedLoot)
	}
}

func TestMapInspection1092ReadOnlyAndUnresolvedRecords(t *testing.T) {
	b := synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Units: []synth.ALMUnit{{X: 0x0380, Y: 0x04ff, ClassID: 300}, {X: 0xffff0000, Y: 256, ClassID: 301}}, Objects: []synth.ALMObject{{X: 512, Y: 768, Kind: 254}}})
	x := &MapInspector{tiles: &terrain.Tileset{}}
	d, err := x.inspect(b, "synthetic")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, d.SourceBytes) {
		t.Fatal("model changed accepted bytes")
	}
	if d.Records[0].Cell != image.Pt(3, 4) || d.Records[0].Warning == "" {
		t.Fatal("fractional unit placement or missing-art warning")
	}
	if !strings.Contains(d.Records[1].Warning, "outside map") {
		t.Fatal("out-of-map authored record silently omitted")
	}
	path := filepath.Join(t.TempDir(), "map.alm")
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	e := ui.NewMapEditor(nil, nil, x.Open)
	if err := e.Open(path); err != nil {
		t.Fatal(err)
	}
	old := e.Document()
	for _, bad := range []string{path + ".missing", filepath.Join(t.TempDir(), "bad.alm")} {
		if strings.HasSuffix(bad, "bad.alm") {
			if err := os.WriteFile(bad, []byte("bad"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		if err := e.Open(bad); err == nil || e.Document() != old || e.Message() == "" {
			t.Fatal("failed open replaced old view or hid error")
		}
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(b, after) {
		t.Fatalf("read-only path changed: %v", err)
	}
}
