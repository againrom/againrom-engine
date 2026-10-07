package sav

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
)

// cityCampaign retains the two not-yet-named campaign scalars as typed u32s,
// while the public CampaignProjection exposes only fields whose meanings are
// established. This is semantic provenance, not a copy of TailRest.
type cityCampaignBase struct {
	dwords [6]uint32
	arrays [2][]uint16
}

type cityCampaignChild struct {
	base cityCampaignBase
	age  uint32
}

type cityCampaignMarker struct {
	value uint32
	text  []byte
	tail  [8]byte
}

type cityCampaign struct {
	base      cityCampaignBase
	children  []cityCampaignChild
	parallel  [2][]uint16
	dwords    []uint32
	arrays    [6][]uint16
	documents [][2]uint32
	// carriers are the document pairs whose kind has the high bit set, in
	// file order. They are written after documents.
	carriers [][2]uint32
	scalars  [7]uint32
	markers  []cityCampaignMarker
}

func parseCityCampaign(raw []byte) (*cityCampaign, error) {
	c := &cityCursor{b: raw}
	readArray := func(what string) ([]uint16, error) {
		n, err := c.u32(what + " count")
		if err != nil {
			return nil, err
		}
		if n > maxCampaignElements || uint64(n) > uint64((len(c.b)-c.p)/2) {
			return nil, fmt.Errorf("sav: city campaign %s count %d exceeds its bounded remaining data", what, n)
		}
		values := make([]uint16, int(n))
		for i := range values {
			values[i], err = c.u16(fmt.Sprintf("%s value %d", what, i))
			if err != nil {
				return nil, err
			}
		}
		return values, nil
	}
	readBase := func(what string) (cityCampaignBase, error) {
		var b cityCampaignBase
		var err error
		for i := range b.dwords {
			b.dwords[i], err = c.u32(fmt.Sprintf("%s dword %d", what, i))
			if err != nil {
				return b, err
			}
		}
		if b.dwords[5] > 1 {
			return b, fmt.Errorf("sav: city campaign %s announced flag is %d", what, b.dwords[5])
		}
		for i := range b.arrays {
			b.arrays[i], err = readArray(fmt.Sprintf("%s array %d", what, i))
			if err != nil {
				return b, err
			}
		}
		return b, nil
	}

	p := &cityCampaign{}
	var err error
	if p.base, err = readBase("base"); err != nil {
		return nil, err
	}
	children, err := c.u32("children count")
	if err != nil {
		return nil, err
	}
	if children > maxCampaignElements || uint64(children) > uint64((len(raw)-c.p)/36) {
		return nil, fmt.Errorf("sav: city campaign child count %d exceeds its bounded remaining data", children)
	}
	p.children = make([]cityCampaignChild, int(children))
	for i := range p.children {
		if p.children[i].base, err = readBase(fmt.Sprintf("child %d", i)); err != nil {
			return nil, err
		}
		if p.children[i].age, err = c.u32(fmt.Sprintf("child %d age", i)); err != nil {
			return nil, err
		}
	}
	parallel, err := c.u32("parallel count")
	if err != nil {
		return nil, err
	}
	if parallel > maxCampaignElements || uint64(parallel) > uint64((len(raw)-c.p)/4) {
		return nil, fmt.Errorf("sav: city campaign parallel count %d exceeds its bounded remaining data", parallel)
	}
	for side := range p.parallel {
		p.parallel[side] = make([]uint16, int(parallel))
		for i := range p.parallel[side] {
			p.parallel[side][i], err = c.u16(fmt.Sprintf("parallel %d value %d", side, i))
			if err != nil {
				return nil, err
			}
		}
	}
	dwords, err := c.u32("mercenary hire count")
	if err != nil {
		return nil, err
	}
	if dwords > maxCampaignElements || uint64(dwords) > uint64((len(raw)-c.p)/4) {
		return nil, fmt.Errorf("sav: city campaign hire count %d exceeds its bounded remaining data", dwords)
	}
	p.dwords = make([]uint32, int(dwords))
	for i := range p.dwords {
		p.dwords[i], err = c.u32(fmt.Sprintf("hire flag %d", i))
		if err != nil {
			return nil, err
		}
		if p.dwords[i] > 1 {
			return nil, fmt.Errorf("sav: city campaign hire flag %d is %d", i, p.dwords[i])
		}
	}
	for i := range p.arrays {
		p.arrays[i], err = readArray(fmt.Sprintf("array %d", i))
		if err != nil {
			return nil, err
		}
	}
	documents, err := c.u32("document count")
	if err != nil {
		return nil, err
	}
	if documents > maxCampaignDocumentPairs || uint64(documents) > uint64((len(raw)-c.p)/8) {
		return nil, fmt.Errorf("sav: city campaign document count %d exceeds its bounded remaining data", documents)
	}
	pairs := make([][2]uint32, int(documents))
	for i := range pairs {
		if pairs[i][0], err = c.u32(fmt.Sprintf("document %d value", i)); err != nil {
			return nil, err
		}
		if pairs[i][1], err = c.u32(fmt.Sprintf("document %d kind", i)); err != nil {
			return nil, err
		}
	}
	if p.documents, p.carriers, err = splitDocumentPairs(pairs); err != nil {
		return nil, fmt.Errorf("sav: city campaign %w", err)
	}
	if p.documents == nil {
		p.documents = [][2]uint32{}
	}
	for i := range p.scalars {
		if p.scalars[i], err = c.u32(fmt.Sprintf("scalar %d", i)); err != nil {
			return nil, err
		}
	}
	if p.scalars[4] > 1 {
		return nil, fmt.Errorf("sav: city campaign first-MapPoint flag is %d", p.scalars[4])
	}
	markers, err := c.u32("marker count")
	if err != nil {
		return nil, err
	}
	if markers > maxCampaignElements || uint64(markers) > uint64((len(raw)-c.p)/17) {
		return nil, fmt.Errorf("sav: city campaign marker count %d exceeds its bounded remaining data", markers)
	}
	p.markers = make([]cityCampaignMarker, int(markers))
	for i := range p.markers {
		m := &p.markers[i]
		if m.value, err = c.u32(fmt.Sprintf("marker %d value", i)); err != nil {
			return nil, err
		}
		n, err := c.u32(fmt.Sprintf("marker %d text length", i))
		if err != nil {
			return nil, err
		}
		if n == 0 || uint64(n) > uint64(len(raw)-c.p) {
			return nil, fmt.Errorf("sav: city campaign marker %d text length is %d", i, n)
		}
		text, err := c.take(int(n), fmt.Sprintf("marker %d text", i))
		if err != nil {
			return nil, err
		}
		if text[len(text)-1] != 0 || bytes.IndexByte(text[:len(text)-1], 0) >= 0 || len(text) == 1 {
			return nil, fmt.Errorf("sav: city campaign marker %d is not one nonempty NUL-terminated string", i)
		}
		m.text = append([]byte(nil), text...)
		tail, err := c.take(8, fmt.Sprintf("marker %d tail", i))
		if err != nil {
			return nil, err
		}
		copy(m.tail[:], tail)
	}
	if c.p != len(raw) {
		return nil, fmt.Errorf("sav: city campaign consumed %d of %d bytes", c.p, len(raw))
	}
	return p, nil
}

func cloneCityCampaign(source *cityCampaign) *cityCampaign {
	cloneBase := func(source cityCampaignBase) cityCampaignBase {
		out := source
		for i := range out.arrays {
			out.arrays[i] = append([]uint16(nil), source.arrays[i]...)
		}
		return out
	}
	out := &cityCampaign{base: cloneBase(source.base), dwords: append([]uint32(nil), source.dwords...), documents: append([][2]uint32(nil), source.documents...), carriers: append([][2]uint32(nil), source.carriers...), scalars: source.scalars}
	out.children = make([]cityCampaignChild, len(source.children))
	for i := range source.children {
		out.children[i] = cityCampaignChild{base: cloneBase(source.children[i].base), age: source.children[i].age}
	}
	for i := range source.parallel {
		out.parallel[i] = append([]uint16(nil), source.parallel[i]...)
	}
	for i := range source.arrays {
		out.arrays[i] = append([]uint16(nil), source.arrays[i]...)
	}
	out.markers = make([]cityCampaignMarker, len(source.markers))
	for i := range source.markers {
		out.markers[i] = source.markers[i]
		out.markers[i].text = append([]byte(nil), source.markers[i].text...)
	}
	return out
}

func serializeCityCampaign(p *cityCampaign) ([]byte, error) {
	if p == nil {
		return nil, fmt.Errorf("sav: city campaign is nil")
	}
	if len(p.parallel[0]) != len(p.parallel[1]) {
		return nil, fmt.Errorf("sav: city campaign parallel arrays have %d and %d elements", len(p.parallel[0]), len(p.parallel[1]))
	}
	validateCount := func(what string, n int) error {
		if n > maxCampaignElements {
			return fmt.Errorf("sav: city campaign %s count %d exceeds %d", what, n, maxCampaignElements)
		}
		return nil
	}
	appendArray := func(dst []byte, what string, values []uint16) ([]byte, error) {
		if err := validateCount(what, len(values)); err != nil {
			return nil, err
		}
		dst = cityAppendU32(dst, uint32(len(values)))
		for _, v := range values {
			dst = cityAppendU16(dst, v)
		}
		return dst, nil
	}
	appendBase := func(dst []byte, what string, b cityCampaignBase) ([]byte, error) {
		if b.dwords[5] > 1 {
			return nil, fmt.Errorf("sav: city campaign %s announced flag is %d", what, b.dwords[5])
		}
		for _, v := range b.dwords {
			dst = cityAppendU32(dst, v)
		}
		var err error
		for i, values := range b.arrays {
			dst, err = appendArray(dst, fmt.Sprintf("%s array %d", what, i), values)
			if err != nil {
				return nil, err
			}
		}
		return dst, nil
	}
	var out []byte
	var err error
	if out, err = appendBase(out, "base", p.base); err != nil {
		return nil, err
	}
	for _, pair := range []struct {
		name string
		n    int
	}{{"children", len(p.children)}, {"parallel", len(p.parallel[0])}, {"hire flags", len(p.dwords)}, {"documents", len(p.documents)}, {"markers", len(p.markers)}} {
		if err := validateCount(pair.name, pair.n); err != nil {
			return nil, err
		}
	}
	out = cityAppendU32(out, uint32(len(p.children)))
	for i, child := range p.children {
		if out, err = appendBase(out, fmt.Sprintf("child %d", i), child.base); err != nil {
			return nil, err
		}
		out = cityAppendU32(out, child.age)
	}
	out = cityAppendU32(out, uint32(len(p.parallel[0])))
	for _, values := range p.parallel {
		for _, v := range values {
			out = cityAppendU16(out, v)
		}
	}
	out = cityAppendU32(out, uint32(len(p.dwords)))
	for i, v := range p.dwords {
		if v > 1 {
			return nil, fmt.Errorf("sav: city campaign hire flag %d is %d", i, v)
		}
		out = cityAppendU32(out, v)
	}
	for i, values := range p.arrays {
		if out, err = appendArray(out, fmt.Sprintf("array %d", i), values); err != nil {
			return nil, err
		}
	}
	if len(p.documents)+len(p.carriers) > maxCampaignDocumentPairs {
		return nil, fmt.Errorf("sav: city campaign document count %d exceeds %d", len(p.documents)+len(p.carriers), maxCampaignDocumentPairs)
	}
	out = cityAppendU32(out, uint32(len(p.documents)+len(p.carriers)))
	for i, document := range p.documents {
		if document[1] > 1 {
			return nil, fmt.Errorf("sav: city campaign document %d kind is %d", i, document[1])
		}
		out = cityAppendU32(out, document[0])
		out = cityAppendU32(out, document[1])
	}
	for i, carrier := range p.carriers {
		if !IsCarrierKind(carrier[1]) {
			return nil, fmt.Errorf("sav: city campaign carrier %d kind is %d", i, carrier[1])
		}
		out = cityAppendU32(out, carrier[0])
		out = cityAppendU32(out, carrier[1])
	}
	if p.scalars[4] > 1 {
		return nil, fmt.Errorf("sav: city campaign first-MapPoint flag is %d", p.scalars[4])
	}
	for _, v := range p.scalars {
		out = cityAppendU32(out, v)
	}
	out = cityAppendU32(out, uint32(len(p.markers)))
	for i, marker := range p.markers {
		if len(marker.text) < 2 || marker.text[len(marker.text)-1] != 0 || bytes.IndexByte(marker.text[:len(marker.text)-1], 0) >= 0 {
			return nil, fmt.Errorf("sav: city campaign marker %d is not one nonempty NUL-terminated string", i)
		}
		out = cityAppendU32(out, marker.value)
		out = cityAppendU32(out, uint32(len(marker.text)))
		out = append(out, marker.text...)
		out = append(out, marker.tail[:]...)
	}
	return out, nil
}

// campaignRecordToBase converts one CampaignRecord (the public, named shape)
// to the raw six-dword-plus-two-array cityCampaignBase the wire format
// stores, shared by the main record and every child. It is the one place
// that maps Mission/MapObject/Payment/ShopMin/ShopMax/Announced onto their
// fixed dword slots, so applyCityCampaignProjection and a direct
// File.SetCampaignChildren edit cannot disagree about the mapping.
func campaignRecordToBase(record CampaignRecord) (cityCampaignBase, error) {
	if len(record.AddHero) > maxCampaignElements || len(record.EnableMercenary) > maxCampaignElements {
		return cityCampaignBase{}, fmt.Errorf("campaign record arrays exceed %d elements", maxCampaignElements)
	}
	b := cityCampaignBase{dwords: [6]uint32{record.Mission, record.MapObject, record.Payment, record.ShopMin, record.ShopMax}}
	if record.Announced {
		b.dwords[5] = 1
	}
	b.arrays[0] = append([]uint16(nil), record.AddHero...)
	b.arrays[1] = append([]uint16(nil), record.EnableMercenary...)
	return b, nil
}

func applyCityCampaignProjection(dst *cityCampaign, update CampaignProjection) error {
	if dst == nil {
		return fmt.Errorf("sav: city campaign provenance is nil")
	}
	if update.Main.Age != 0 {
		return fmt.Errorf("sav: city campaign main record cannot carry Age=%d", update.Main.Age)
	}
	if len(update.Children) > maxCampaignElements || len(update.MercenaryWorking) > maxCampaignElements || len(update.MercenaryHired) > maxCampaignElements ||
		len(update.Documents) > maxCampaignElements || len(update.Markers) > maxCampaignElements {
		return fmt.Errorf("sav: city campaign update exceeds the %d-element bound", maxCampaignElements)
	}
	if len(update.MercenaryWorking) != len(update.MercenaryPristine) {
		return fmt.Errorf("sav: city campaign working/pristine pools have %d and %d elements", len(update.MercenaryWorking), len(update.MercenaryPristine))
	}
	base, err := campaignRecordToBase(update.Main)
	if err != nil {
		return err
	}
	children := make([]cityCampaignChild, len(update.Children))
	for i, record := range update.Children {
		children[i].base, err = campaignRecordToBase(record)
		if err != nil {
			return fmt.Errorf("sav: city campaign child %d: %w", i, err)
		}
		children[i].age = record.Age
	}
	for name, n := range map[string]int{
		"Mercenaries": len(update.Mercenaries), "PermanentMercenaries": len(update.PermanentMercenaries),
		"InnNPC": len(update.InnNPC), "InnMission": len(update.InnMission), "TCMission": len(update.TCMission), "ShopMission": len(update.ShopMission),
	} {
		if n > maxCampaignElements {
			return fmt.Errorf("sav: city campaign %s has %d elements", name, n)
		}
	}
	dst.base = base
	dst.children = children
	dst.parallel[0] = append([]uint16(nil), update.MercenaryWorking...)
	dst.parallel[1] = append([]uint16(nil), update.MercenaryPristine...)
	dst.dwords = make([]uint32, len(update.MercenaryHired))
	for i, hired := range update.MercenaryHired {
		if hired {
			dst.dwords[i] = 1
		}
	}
	dst.arrays[0] = append([]uint16(nil), update.Mercenaries...)
	dst.arrays[1] = append([]uint16(nil), update.PermanentMercenaries...)
	dst.arrays[2] = append([]uint16(nil), update.InnNPC...)
	dst.arrays[3] = append([]uint16(nil), update.InnMission...)
	dst.arrays[4] = append([]uint16(nil), update.TCMission...)
	dst.arrays[5] = append([]uint16(nil), update.ShopMission...)
	carriers, err := carrierPairs(update.Payload)
	if err != nil {
		return err
	}
	dst.carriers = carriers
	dst.documents = make([][2]uint32, len(update.Documents))
	for i, document := range update.Documents {
		if document.Kind > 1 {
			return fmt.Errorf("sav: city campaign document %d kind is %d", i, document.Kind)
		}
		dst.documents[i] = [2]uint32{document.Value, document.Kind}
	}
	// Scalar 1 remains the unnamed lawful-source value. Old gob projections
	// lack ScoreEventsKnown and must not zero the source's FAME counter.
	dst.scalars[0] = update.SelectedMission
	dst.scalars[2] = update.AutoGetMission
	dst.scalars[3] = update.LastMission
	if update.FirstMapPoint {
		dst.scalars[4] = 1
	} else {
		dst.scalars[4] = 0
	}
	dst.scalars[5] = update.MissionTime
	if update.ScoreEventsKnown {
		dst.scalars[6] = update.ScoreEvents
	}
	dst.markers = make([]cityCampaignMarker, len(update.Markers))
	for i, marker := range update.Markers {
		if marker.Picture == "" || strings.IndexByte(marker.Picture, 0) >= 0 {
			return fmt.Errorf("sav: city campaign marker %d picture is empty or contains NUL", i)
		}
		dst.markers[i].value = marker.Value
		dst.markers[i].text = append(append([]byte(nil), marker.Picture...), 0)
		binary.LittleEndian.PutUint32(dst.markers[i].tail[0:4], marker.Field0)
		binary.LittleEndian.PutUint32(dst.markers[i].tail[4:8], marker.Field1)
	}
	return nil
}
