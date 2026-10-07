package sav

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// CampaignRecord is the common campaign mission record written once for the
// main mission and once for each retained child mission
// (SAV-CAMPAIGN-076). All slices are detached from the save bytes.
type CampaignRecord struct {
	Mission         uint32
	MapObject       uint32
	Payment         uint32
	ShopMin         uint32
	ShopMax         uint32
	Announced       bool
	AddHero         []uint16
	EnableMercenary []uint16
	Age             uint32
}

// CampaignDocument is one persisted campaign document pair. Kind 0 is a
// picture and kind 1 is text (SAV-CAMPAIGN-085).
type CampaignDocument struct {
	Value uint32
	Kind  uint32
}

// CampaignMarker is one selected-mission marker-cache record. Picture is its
// NUL-terminated picture name. Value and the final two dwords are typed as
// record fields but deliberately remain unnamed: research establishes their
// wire shape, not their meaning (SAV-CAMPMARK-073).
type CampaignMarker struct {
	Value   uint32
	Picture string
	Field0  uint32
	Field1  uint32
}

// CampaignProjection is the campaign record after the embedded state store.
// It contains only typed fields established by SAV-CAMPAIGN-076..088. The two
// still-unnamed scalar +0x114 is read for framing. ScoreEvents preserves the
// +0x128 FAME input; Known distinguishes raw decode from older gob snapshots.
//
// Campaign returns a deep copy, so this value is an immutable projection of a
// File rather than a writable view over its bytes.
type CampaignProjection struct {
	Main                 CampaignRecord
	Children             []CampaignRecord
	MercenaryWorking     []uint16
	MercenaryPristine    []uint16
	MercenaryHired       []bool
	Mercenaries          []uint16
	PermanentMercenaries []uint16
	InnNPC               []uint16
	InnMission           []uint16
	TCMission            []uint16
	ShopMission          []uint16
	Documents            []CampaignDocument
	// Payload is the Againrom-only payload decoded from the carrier pairs of
	// the document list, nil when there are none or they do not decode.
	// PayloadError says why they did not. Documents holds vanilla pairs only.
	Payload      *DocPayload
	PayloadError string
	// CarrierRecords is how many carrier pairs the record held when it was
	// read, decoded or not. Writing ignores it.
	CarrierRecords   int
	SelectedMission  uint32
	AutoGetMission   uint32
	LastMission      uint32
	FirstMapPoint    bool
	MissionTime      uint32
	ScoreEvents      uint32
	ScoreEventsKnown bool
	Markers          []CampaignMarker
}

// Campaign reports the decoded campaign projection. A save whose tail did not
// frame as an embedded state store has no projection. A framed store followed
// by a malformed campaign record returns the parse error without exposing a
// partial value.
func (f *File) Campaign() (CampaignProjection, bool, error) {
	if f == nil || f.store == nil || (f.campaign == nil && f.campaignErr == nil) {
		return CampaignProjection{}, false, nil
	}
	if f.campaignErr != nil {
		return CampaignProjection{}, false, f.campaignErr
	}
	return cloneCampaignProjection(*f.campaign), true, nil
}

func cloneCampaignRecord(in CampaignRecord) CampaignRecord {
	out := in
	out.AddHero = append([]uint16(nil), in.AddHero...)
	out.EnableMercenary = append([]uint16(nil), in.EnableMercenary...)
	return out
}

func cloneCampaignProjection(in CampaignProjection) CampaignProjection {
	out := in
	out.Main = cloneCampaignRecord(in.Main)
	out.Children = make([]CampaignRecord, len(in.Children))
	for i := range in.Children {
		out.Children[i] = cloneCampaignRecord(in.Children[i])
	}
	out.MercenaryWorking = append([]uint16(nil), in.MercenaryWorking...)
	out.MercenaryPristine = append([]uint16(nil), in.MercenaryPristine...)
	out.MercenaryHired = append([]bool(nil), in.MercenaryHired...)
	out.Mercenaries = append([]uint16(nil), in.Mercenaries...)
	out.PermanentMercenaries = append([]uint16(nil), in.PermanentMercenaries...)
	out.InnNPC = append([]uint16(nil), in.InnNPC...)
	out.InnMission = append([]uint16(nil), in.InnMission...)
	out.TCMission = append([]uint16(nil), in.TCMission...)
	out.ShopMission = append([]uint16(nil), in.ShopMission...)
	out.Documents = append([]CampaignDocument(nil), in.Documents...)
	out.Payload = in.Payload.Clone()
	out.Markers = append([]CampaignMarker(nil), in.Markers...)
	return out
}

const maxCampaignElements = 4096

type campaignCursor struct {
	b   []byte
	off int
}

func (c *campaignCursor) need(n int, what string) error {
	if n < 0 || c.off > len(c.b)-n {
		return fmt.Errorf("sav: campaign %s at %d overruns %d bytes", what, c.off, len(c.b))
	}
	return nil
}

func (c *campaignCursor) u32(what string) (uint32, error) {
	if err := c.need(4, what); err != nil {
		return 0, err
	}
	v := binary.LittleEndian.Uint32(c.b[c.off:])
	c.off += 4
	return v, nil
}

func (c *campaignCursor) count(what string, minimumBytes int) (int, error) {
	v, err := c.u32(what + " count")
	if err != nil {
		return 0, err
	}
	if v > maxCampaignElements {
		return 0, fmt.Errorf("sav: campaign %s count %d exceeds %d", what, v, maxCampaignElements)
	}
	n := int(v)
	if minimumBytes > 0 && n > (len(c.b)-c.off)/minimumBytes {
		return 0, fmt.Errorf("sav: campaign %s count %d exceeds the remaining %d bytes", what, n, len(c.b)-c.off)
	}
	return n, nil
}

// documentCount reads the document list count. The bound covers vanilla pairs
// and carriers together; splitDocumentPairs bounds the vanilla pairs alone.
func (c *campaignCursor) documentCount() (int, error) {
	v, err := c.u32("documents count")
	if err != nil {
		return 0, err
	}
	if v > maxCampaignDocumentPairs {
		return 0, fmt.Errorf("sav: campaign documents count %d exceeds %d", v, maxCampaignDocumentPairs)
	}
	if int(v) > (len(c.b)-c.off)/8 {
		return 0, fmt.Errorf("sav: campaign documents count %d exceeds the remaining %d bytes", v, len(c.b)-c.off)
	}
	return int(v), nil
}

func (c *campaignCursor) u16s(what string) ([]uint16, error) {
	n, err := c.count(what, 2)
	if err != nil {
		return nil, err
	}
	if err := c.need(2*n, what); err != nil {
		return nil, err
	}
	out := make([]uint16, n)
	for i := range out {
		out[i] = binary.LittleEndian.Uint16(c.b[c.off+2*i:])
	}
	c.off += 2 * n
	return out, nil
}

func (c *campaignCursor) record(child bool) (CampaignRecord, error) {
	var r CampaignRecord
	fields := []*uint32{&r.Mission, &r.MapObject, &r.Payment, &r.ShopMin, &r.ShopMax}
	for i, dst := range fields {
		v, err := c.u32(fmt.Sprintf("record field %d", i))
		if err != nil {
			return CampaignRecord{}, err
		}
		*dst = v
	}
	announced, err := c.u32("record announcement latch")
	if err != nil {
		return CampaignRecord{}, err
	}
	if announced > 1 {
		return CampaignRecord{}, fmt.Errorf("sav: campaign announcement latch is %d, want 0 or 1", announced)
	}
	r.Announced = announced != 0
	if r.AddHero, err = c.u16s("record AddHero"); err != nil {
		return CampaignRecord{}, err
	}
	if r.EnableMercenary, err = c.u16s("record EnableMercenary"); err != nil {
		return CampaignRecord{}, err
	}
	if child {
		r.Age, err = c.u32("child age")
		if err != nil {
			return CampaignRecord{}, err
		}
	}
	return r, nil
}

func parseCampaignProjection(b []byte) (CampaignProjection, error) {
	c := campaignCursor{b: b}
	var out CampaignProjection
	var err error
	if out.Main, err = c.record(false); err != nil {
		return CampaignProjection{}, err
	}
	children, err := c.count("children", 36)
	if err != nil {
		return CampaignProjection{}, err
	}
	out.Children = make([]CampaignRecord, children)
	for i := range out.Children {
		if out.Children[i], err = c.record(true); err != nil {
			return CampaignProjection{}, fmt.Errorf("sav: campaign child %d: %w", i, err)
		}
	}
	parallel, err := c.count("mercenary pool", 4)
	if err != nil {
		return CampaignProjection{}, err
	}
	if err := c.need(parallel*4, "paired mercenary pools"); err != nil {
		return CampaignProjection{}, err
	}
	out.MercenaryWorking = make([]uint16, parallel)
	out.MercenaryPristine = make([]uint16, parallel)
	for i := range out.MercenaryWorking {
		out.MercenaryWorking[i] = binary.LittleEndian.Uint16(c.b[c.off+2*i:])
	}
	c.off += 2 * parallel
	for i := range out.MercenaryPristine {
		out.MercenaryPristine[i] = binary.LittleEndian.Uint16(c.b[c.off+2*i:])
	}
	c.off += 2 * parallel
	hired, err := c.count("mercenary hire flags", 4)
	if err != nil {
		return CampaignProjection{}, err
	}
	out.MercenaryHired = make([]bool, hired)
	for i := range out.MercenaryHired {
		v, err := c.u32("mercenary hire flag")
		if err != nil {
			return CampaignProjection{}, err
		}
		if v > 1 {
			return CampaignProjection{}, fmt.Errorf("sav: campaign mercenary hire flag %d is %d, want 0 or 1", i, v)
		}
		out.MercenaryHired[i] = v != 0
	}
	arrays := []*[]uint16{&out.Mercenaries, &out.PermanentMercenaries, &out.InnNPC,
		&out.InnMission, &out.TCMission, &out.ShopMission}
	names := [...]string{"Mercenaries", "permanent mercenaries", "InnNPC", "InnMission", "TCMission", "ShopMission"}
	for i := range arrays {
		*arrays[i], err = c.u16s(names[i])
		if err != nil {
			return CampaignProjection{}, err
		}
	}
	documents, err := c.documentCount()
	if err != nil {
		return CampaignProjection{}, err
	}
	pairs := make([][2]uint32, documents)
	for i := range pairs {
		if pairs[i][0], err = c.u32("document value"); err != nil {
			return CampaignProjection{}, err
		}
		if pairs[i][1], err = c.u32("document kind"); err != nil {
			return CampaignProjection{}, err
		}
	}
	vanilla, carriers, err := splitDocumentPairs(pairs)
	if err != nil {
		return CampaignProjection{}, fmt.Errorf("sav: campaign %w", err)
	}
	out.Documents = make([]CampaignDocument, len(vanilla))
	for i, pair := range vanilla {
		out.Documents[i] = CampaignDocument{Value: pair[0], Kind: pair[1]}
	}
	out.Payload, out.PayloadError = decodeCarrierPairs(carriers)
	out.CarrierRecords = len(carriers)
	if out.SelectedMission, err = c.u32("selected mission"); err != nil {
		return CampaignProjection{}, err
	}
	if _, err = c.u32("unnamed +0x114 scalar"); err != nil {
		return CampaignProjection{}, err
	}
	if out.AutoGetMission, err = c.u32("AutoGetMission"); err != nil {
		return CampaignProjection{}, err
	}
	if out.LastMission, err = c.u32("LastMission"); err != nil {
		return CampaignProjection{}, err
	}
	first, err := c.u32("first MapPoint flag")
	if err != nil {
		return CampaignProjection{}, err
	}
	if first > 1 {
		return CampaignProjection{}, fmt.Errorf("sav: campaign first-MapPoint flag is %d, want 0 or 1", first)
	}
	out.FirstMapPoint = first != 0
	if out.MissionTime, err = c.u32("mission time"); err != nil {
		return CampaignProjection{}, err
	}
	if out.ScoreEvents, err = c.u32("score event counter"); err != nil {
		return CampaignProjection{}, err
	}
	out.ScoreEventsKnown = true
	markers, err := c.count("markers", 17)
	if err != nil {
		return CampaignProjection{}, err
	}
	out.Markers = make([]CampaignMarker, markers)
	for i := range out.Markers {
		m := &out.Markers[i]
		if m.Value, err = c.u32("marker value"); err != nil {
			return CampaignProjection{}, err
		}
		n, err := c.count("marker picture", 1)
		if err != nil {
			return CampaignProjection{}, err
		}
		if n == 0 {
			return CampaignProjection{}, fmt.Errorf("sav: campaign marker %d picture has no NUL byte", i)
		}
		if err := c.need(n, "marker picture"); err != nil {
			return CampaignProjection{}, err
		}
		text := c.b[c.off : c.off+n]
		if text[n-1] != 0 || bytes.IndexByte(text[:n-1], 0) >= 0 {
			return CampaignProjection{}, fmt.Errorf("sav: campaign marker %d picture is not one NUL-terminated string", i)
		}
		if n == 1 {
			return CampaignProjection{}, fmt.Errorf("sav: campaign marker %d picture is empty", i)
		}
		m.Picture = string(text[:n-1])
		c.off += n
		if m.Field0, err = c.u32("marker field 0"); err != nil {
			return CampaignProjection{}, err
		}
		if m.Field1, err = c.u32("marker field 1"); err != nil {
			return CampaignProjection{}, err
		}
	}
	if c.off != len(c.b) {
		return CampaignProjection{}, fmt.Errorf("sav: campaign record consumed %d of %d bytes", c.off, len(c.b))
	}
	return out, nil
}
