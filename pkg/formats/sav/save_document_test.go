package sav

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

func saveDocument1115Literal(t *testing.T) ([]byte, []byte, []byte, []byte) {
	t.Helper()
	body, _ := archiveDocument1115Fixture()
	state := literalStateStore(literalWorldDirectories(266, 7, 266))
	_, campaign := campaignProjectionFixture()
	blob := Compress(body)
	file := make([]byte, 16)
	copy(file, "Asg&")
	binary.LittleEndian.PutUint32(file[4:], uint32(16+len(blob)))
	binary.LittleEndian.PutUint32(file[8:], 0x0bad0002)
	binary.LittleEndian.PutUint32(file[12:], uint32(len(blob)))
	file = append(file, blob...)
	label := bytes.Repeat([]byte{0x77}, 256)
	copy(label, []byte{'s', 0xe0, 'v', 0})
	file = append(file, label...)
	file = append(file, state...)
	file = append(file, campaign...)
	return file, body, state, campaign
}

func TestSaveDocument1115CompleteContainerAfterSourceErasure(t *testing.T) {
	source, _, state, campaign := saveDocument1115Literal(t)
	wantState, _ := literalReadState(state)
	d, err := parseSaveDocument(source)
	if err != nil {
		t.Fatal(err)
	}
	clear(source)
	clear(state)
	got, err := serializeSaveDocument(d)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := Open(got)
	if err != nil {
		t.Fatal(err)
	}
	gotState, _ := literalReadState(loaded.Store)
	if !reflect.DeepEqual(gotState, wantState) || !bytes.Equal(loaded.TailRest, campaign) || !bytes.Equal(loaded.Label, []byte{'s', 0xe0, 'v'}) {
		t.Fatal("whole container lost a typed state/campaign/label value")
	}
	if bytes.Contains(loaded.LabelRegion[4:], []byte{0x77}) || binary.LittleEndian.Uint32(loaded.Store[20:]) != 0 {
		t.Fatal("transport debris was replayed")
	}
	// Independently vary world, app, projectile and campaign values before a
	// new serialization; no original File remains to patch or replay.
	d.archive.world.session.Lost = 3
	d.archive.world.sacks[0].Value["S3C"] = 900
	d.state.values["/View/X"] = cityStateValue{kind: 2, int32: -789}
	d.state.values["/Prj266/actionphase"] = cityStateValue{kind: 2, int32: 27}
	d.state.values["/Fog/Data"] = cityStateValue{kind: 6, bytes: literalStateWords(3, 5, 8, 13)}
	d.campaign.scalars[5] = 3456
	d.label = []byte("current")
	changed, err := serializeSaveDocument(d)
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := parseSaveDocument(changed)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.archive.world.session.Lost != 3 || fresh.archive.world.sacks[1].Value["S3C"] != 900 || fresh.state.values["/View/X"].int32 != -789 || fresh.state.values["/Prj266/actionphase"].int32 != 27 || !bytes.Equal(fresh.state.values["/Fog/Data"].bytes, literalStateWords(3, 5, 8, 13)) || fresh.campaign.scalars[5] != 3456 || string(fresh.label) != "current" {
		t.Fatal("changed values did not reach fresh full-container decode")
	}
	// Determinism covers fresh transport addresses and heap layout as well.
	again, err := serializeSaveDocument(fresh)
	if err != nil || !bytes.Equal(changed, again) {
		t.Fatalf("second complete encoding differs: %v", err)
	}
}

func TestSaveDocument1115SharesCodecWithCity(t *testing.T) {
	d, err := parseSaveDocument(cityTestSource(t))
	if err != nil {
		t.Fatal(err)
	}
	if d.archive.world != nil {
		t.Fatal("city gained a world")
	}
	b, err := serializeSaveDocument(d)
	if err != nil {
		t.Fatal(err)
	}
	f, err := Open(b)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.CityProvenance(); err != nil {
		t.Fatalf("existing city semantic reader rejects shared codec: %v", err)
	}
}

func TestSaveDocument1115RejectsIncompleteAndOversizedContainers(t *testing.T) {
	b, _, _, _ := saveDocument1115Literal(t)
	for _, n := range []int{0, 19, len(b) - 1, len(b) - 24} {
		if d, err := parseSaveDocument(b[:n]); err == nil || d != nil {
			t.Fatalf("accepted truncated container %d", n)
		}
	}
	for name, mutate := range map[string]func(*saveDocument){
		"version":    func(d *saveDocument) { d.version = 0 },
		"label":      func(d *saveDocument) { d.label = []byte{'a', 0, 'b'} },
		"state":      func(d *saveDocument) { delete(d.state.values, "/Fog/Data") },
		"projectile": func(d *saveDocument) { delete(d.state.values, "/Prj266/actionphase") },
		"campaign":   func(d *saveDocument) { d.campaign = nil },
		"marker budget": func(d *saveDocument) {
			text := make([]byte, 1<<20)
			for range 33 {
				d.campaign.markers = append(d.campaign.markers, cityCampaignMarker{text: text})
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			d, err := parseSaveDocument(b)
			if err != nil {
				t.Fatal(err)
			}
			mutate(d)
			if out, err := serializeSaveDocument(d); err == nil || out != nil {
				t.Fatal("invalid container produced output")
			}
		})
	}
	bad := append([]byte(nil), b...)
	binary.LittleEndian.PutUint32(bad[16:], maxArchiveOutput)
	if d, err := parseSaveDocument(bad); err == nil || d != nil {
		t.Fatal("accepted unbounded declared expansion")
	}
	// Actual run expansion must be checked even when its declaration is tiny.
	bad = make([]byte, 20)
	for range maxArchiveOutput/254 + 1 {
		bad = append(bad, 0xff, 1, 2)
	}
	binary.LittleEndian.PutUint32(bad[4:], uint32(len(bad)))
	binary.LittleEndian.PutUint32(bad[16:], 1)
	if err := checkArchiveContainerSpan(bad); err == nil {
		t.Fatal("accepted unbounded actual expansion")
	}
}
