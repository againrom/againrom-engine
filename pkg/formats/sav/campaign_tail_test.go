package sav

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/formats/reg"
)

func campaignDWords(values ...uint32) []byte {
	var out []byte
	for _, value := range values {
		out = binary.LittleEndian.AppendUint32(out, value)
	}
	return out
}

func campaignPatternOffset(t *testing.T, body, pattern []byte, name string) int {
	t.Helper()
	off := bytes.Index(body, pattern)
	if off < 0 {
		t.Fatalf("fixture has no %s pattern", name)
	}
	return off
}

type campaignTailBuilder struct{ b []byte }

func (w *campaignTailBuilder) u32(v uint32) { w.b = binary.LittleEndian.AppendUint32(w.b, v) }
func (w *campaignTailBuilder) u16(v uint16) { w.b = binary.LittleEndian.AppendUint16(w.b, v) }
func (w *campaignTailBuilder) u16s(v ...uint16) {
	w.u32(uint32(len(v)))
	for _, n := range v {
		w.u16(n)
	}
}
func (w *campaignTailBuilder) record(r CampaignRecord, child bool) {
	for _, v := range [...]uint32{r.Mission, r.MapObject, r.Payment, r.ShopMin, r.ShopMax} {
		w.u32(v)
	}
	if r.Announced {
		w.u32(1)
	} else {
		w.u32(0)
	}
	w.u16s(r.AddHero...)
	w.u16s(r.EnableMercenary...)
	if child {
		w.u32(r.Age)
	}
}

func encodeCampaignProjection(in CampaignProjection, unnamed114, unnamed128 uint32) []byte {
	w := campaignTailBuilder{}
	w.record(in.Main, false)
	w.u32(uint32(len(in.Children)))
	for _, r := range in.Children {
		w.record(r, true)
	}
	w.u32(uint32(len(in.MercenaryWorking)))
	for _, v := range in.MercenaryWorking {
		w.u16(v)
	}
	for _, v := range in.MercenaryPristine {
		w.u16(v)
	}
	w.u32(uint32(len(in.MercenaryHired)))
	for _, v := range in.MercenaryHired {
		if v {
			w.u32(1)
		} else {
			w.u32(0)
		}
	}
	for _, v := range [][]uint16{in.Mercenaries, in.PermanentMercenaries, in.InnNPC,
		in.InnMission, in.TCMission, in.ShopMission} {
		w.u16s(v...)
	}
	w.u32(uint32(len(in.Documents)))
	for _, d := range in.Documents {
		w.u32(d.Value)
		w.u32(d.Kind)
	}
	w.u32(in.SelectedMission)
	w.u32(unnamed114)
	w.u32(in.AutoGetMission)
	w.u32(in.LastMission)
	if in.FirstMapPoint {
		w.u32(1)
	} else {
		w.u32(0)
	}
	w.u32(in.MissionTime)
	w.u32(unnamed128)
	w.u32(uint32(len(in.Markers)))
	for _, m := range in.Markers {
		w.u32(m.Value)
		w.u32(uint32(len(m.Picture) + 1))
		w.b = append(w.b, m.Picture...)
		w.b = append(w.b, 0)
		w.u32(m.Field0)
		w.u32(m.Field1)
	}
	return w.b
}

func campaignProjectionFixture() (CampaignProjection, []byte) {
	want := CampaignProjection{
		Main: CampaignRecord{Mission: 50, MapObject: 5, Payment: 700, ShopMin: 11, ShopMax: 999,
			Announced: true, AddHero: []uint16{22}, EnableMercenary: []uint16{3}},
		Children: []CampaignRecord{{Mission: 41, MapObject: 4, Payment: 701, ShopMin: 12,
			ShopMax: 1000, Announced: true, AddHero: []uint16{8}, EnableMercenary: []uint16{10}, Age: 1}},
		Mercenaries: []uint16{3, 10}, PermanentMercenaries: []uint16{1, 2},
		InnNPC: []uint16{0, 5, 6}, InnMission: []uint16{0, 50, 41},
		TCMission: []uint16{51}, ShopMission: []uint16{60},
		Documents:       []CampaignDocument{{Value: 1, Kind: 1}, {Value: 2, Kind: 0}},
		SelectedMission: 41, AutoGetMission: 50, LastMission: 40,
		FirstMapPoint: true, MissionTime: 1234, ScoreEvents: 0xdeadbeef, ScoreEventsKnown: true,
		Markers: []CampaignMarker{{Value: 41, Picture: "mark.bmp", Field0: 17, Field1: 19}},
	}
	for i := 0; i < 15; i++ {
		want.MercenaryWorking = append(want.MercenaryWorking, uint16(i+1))
		want.MercenaryPristine = append(want.MercenaryPristine, uint16(i+21))
		want.MercenaryHired = append(want.MercenaryHired, i == 2)
	}
	return want, encodeCampaignProjection(want, 0xfeedbeef, 0xdeadbeef)
}

func TestCampaignProjectionDecodesEveryTypedAxis(t *testing.T) {
	want, body := campaignProjectionFixture()
	got, err := parseCampaignProjection(body)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("projection = %#v, want %#v", got, want)
	}
}

func TestCampaignProjectionAxesVaryIndependently(t *testing.T) {
	base, _ := campaignProjectionFixture()
	cases := []struct {
		name   string
		mutate func(*CampaignProjection)
	}{
		{"record mission", func(p *CampaignProjection) { p.Main.Mission++ }},
		{"record MapObject", func(p *CampaignProjection) { p.Main.MapObject++ }},
		{"record payment", func(p *CampaignProjection) { p.Main.Payment++ }},
		{"record shop minimum", func(p *CampaignProjection) { p.Main.ShopMin++ }},
		{"record shop maximum", func(p *CampaignProjection) { p.Main.ShopMax++ }},
		{"record announcement", func(p *CampaignProjection) { p.Main.Announced = false }},
		{"record AddHero", func(p *CampaignProjection) { p.Main.AddHero[0]++ }},
		{"record EnableMercenary", func(p *CampaignProjection) { p.Main.EnableMercenary[0]++ }},
		{"child age", func(p *CampaignProjection) { p.Children[0].Age = 0 }},
		{"working counts", func(p *CampaignProjection) { p.MercenaryWorking[0]++ }},
		{"pristine counts", func(p *CampaignProjection) { p.MercenaryPristine[0]++ }},
		{"hire flags", func(p *CampaignProjection) { p.MercenaryHired[0] = true }},
		{"mercenary shelf", func(p *CampaignProjection) { p.Mercenaries[0]++ }},
		{"permanent unlocks", func(p *CampaignProjection) { p.PermanentMercenaries[0]++ }},
		{"inn NPCs", func(p *CampaignProjection) { p.InnNPC[0]++ }},
		{"inn missions", func(p *CampaignProjection) { p.InnMission[0]++ }},
		{"school missions", func(p *CampaignProjection) { p.TCMission[0]++ }},
		{"shop missions", func(p *CampaignProjection) { p.ShopMission[0]++ }},
		{"documents", func(p *CampaignProjection) { p.Documents[0].Value++ }},
		{"selected", func(p *CampaignProjection) { p.SelectedMission++ }},
		{"automatic", func(p *CampaignProjection) { p.AutoGetMission++ }},
		{"last", func(p *CampaignProjection) { p.LastMission++ }},
		{"first MapPoint", func(p *CampaignProjection) { p.FirstMapPoint = false }},
		{"mission time", func(p *CampaignProjection) { p.MissionTime++ }},
		{"score events", func(p *CampaignProjection) { p.ScoreEvents++ }},
		{"markers", func(p *CampaignProjection) { p.Markers[0].Field1++ }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want := cloneCampaignProjection(base)
			tc.mutate(&want)
			got, err := parseCampaignProjection(encodeCampaignProjection(want, 0x11111111, want.ScoreEvents))
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("projection = %#v, err %v; want %#v", got, err, want)
			}
		})
	}
}

func TestUnnamedCampaignScalarNeverEntersProjection(t *testing.T) {
	want, _ := campaignProjectionFixture()
	a, err := parseCampaignProjection(encodeCampaignProjection(want, 1, want.ScoreEvents))
	if err != nil {
		t.Fatal(err)
	}
	b, err := parseCampaignProjection(encodeCampaignProjection(want, 0xffffffff, want.ScoreEvents))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, b) || !reflect.DeepEqual(a, want) {
		t.Fatalf("unnamed scalars changed canonical projection: first=%+v second=%+v", a, b)
	}
}

func TestCampaignProjectionIsDetached(t *testing.T) {
	want, _ := campaignProjectionFixture()
	f := &File{store: &reg.Reg{}, campaign: &want}
	first, ok, err := f.Campaign()
	if err != nil || !ok {
		t.Fatalf("Campaign = ok %v err %v", ok, err)
	}
	first.Main.AddHero[0] = 999
	first.Children[0].EnableMercenary[0] = 999
	first.InnMission[0] = 999
	second, _, _ := f.Campaign()
	if second.Main.AddHero[0] != 22 || second.Children[0].EnableMercenary[0] != 10 || second.InnMission[0] != 0 {
		t.Fatalf("caller mutated the file projection: %+v", second)
	}
}

func TestCampaignProjectionRejectsMalformedSetsWithoutAValue(t *testing.T) {
	_, body := campaignProjectionFixture()
	hireBlock := campaignPatternOffset(t, body, campaignDWords(15, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0), "hire flags")
	documentBlock := campaignPatternOffset(t, body, campaignDWords(2, 1, 1, 2, 0), "documents")
	selectedBlock := campaignPatternOffset(t, body,
		campaignDWords(41, 0xfeedbeef, 50, 40, 1, 1234, 0xdeadbeef), "selected scalars")
	markerPicture := campaignPatternOffset(t, body, []byte("mark.bmp\x00"), "marker picture")
	cases := []struct {
		name string
		edit func([]byte) []byte
		want string
	}{
		{"truncated", func(b []byte) []byte { return b[:len(b)-1] }, "overruns"},
		{"trailing", func(b []byte) []byte { return append(b, 0) }, "consumed"},
		{"announcement", func(b []byte) []byte {
			binary.LittleEndian.PutUint32(b[20:], 2)
			return b
		}, "announcement latch"},
		{"child count ceiling", func(b []byte) []byte {
			binary.LittleEndian.PutUint32(b[36:], maxCampaignElements+1)
			return b
		}, "exceeds 4096"},
		{"hire flag", func(b []byte) []byte {
			binary.LittleEndian.PutUint32(b[hireBlock+4:], 2)
			return b
		}, "hire flag 0"},
		{"document kind", func(b []byte) []byte {
			binary.LittleEndian.PutUint32(b[documentBlock+8:], 2)
			return b
		}, "document 0 kind"},
		{"first MapPoint flag", func(b []byte) []byte {
			binary.LittleEndian.PutUint32(b[selectedBlock+16:], 2)
			return b
		}, "first-MapPoint flag"},
		{"zero marker string", func(b []byte) []byte {
			binary.LittleEndian.PutUint32(b[markerPicture-4:], 0)
			return b
		}, "has no NUL byte"},
		{"empty marker string", func([]byte) []byte {
			projection, _ := campaignProjectionFixture()
			projection.Markers[0].Picture = ""
			return encodeCampaignProjection(projection, 0xfeedbeef, 0xdeadbeef)
		}, "picture is empty"},
		{"marker string count ceiling", func(b []byte) []byte {
			binary.LittleEndian.PutUint32(b[markerPicture-4:], maxCampaignElements+1)
			return b
		}, "exceeds 4096"},
		{"marker string missing terminator", func(b []byte) []byte {
			b[markerPicture+len("mark.bmp")] = 'x'
			return b
		}, "not one NUL-terminated string"},
		{"marker string embedded NUL", func(b []byte) []byte {
			b[markerPicture+2] = 0
			return b
		}, "not one NUL-terminated string"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mut := tc.edit(append([]byte(nil), body...))
			got, err := parseCampaignProjection(mut)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("projection = %+v, err %v, want %q", got, err, tc.want)
			}
			if !reflect.DeepEqual(got, CampaignProjection{}) {
				t.Fatalf("malformed parse returned partial state: %+v", got)
			}
		})
	}
}
