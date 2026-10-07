package game

import (
	"encoding/binary"
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

type generatedDocumentBuilder struct {
	doc                  sav.DocumentData
	state                *SnapshotSAVDocument
	nextKey, nextRuntime uint32
	runtimeIDs           map[uint32]bool
	table                *mapload.Table
	reservedKeys         []uint32
}

func constructedGeneratedItem(item sim.ItemInstance, t *mapload.Table) sim.ItemInstance {
	item = mapload.SourceConstructedItem(item, t)
	// The native code-weight policy gives non-equipment documents/tokens zero
	// weight. At construction that current value becomes explicit SAV state.
	if item.Code != 0 && !item.WeightPresent {
		item.WeightPresent = true
	}
	return item
}

func (c *generatedDocumentBuilder) identity() uint32 {
	if len(c.reservedKeys) != 0 {
		key := c.reservedKeys[0]
		c.reservedKeys = c.reservedKeys[1:]
		return key
	}
	c.nextKey += 16
	return c.nextKey
}
func (c *generatedDocumentBuilder) runtime() uint32 {
	if c.runtimeIDs == nil {
		c.runtimeIDs = savedRuntimeIDs(c.doc.Objects)
	}
	for {
		c.nextRuntime++
		if !c.runtimeIDs[c.nextRuntime] {
			c.runtimeIDs[c.nextRuntime] = true
			return c.nextRuntime
		}
	}
}
func (c *generatedDocumentBuilder) append(r sav.DocumentRecordData) (uint16, error) {
	if len(c.doc.Objects) >= 0x7fff {
		return 0, fmt.Errorf("generated world exceeds object reference range")
	}
	c.doc.Objects = append(c.doc.Objects, r)
	return uint16(len(c.doc.Objects)), nil
}
func mustSetValue(r *sav.DocumentRecordData, n string, v uint32) {
	if err := savedStructureSetValue(r, n, v); err != nil {
		panic(err)
	}
}
func mustSetRaw(r *sav.DocumentRecordData, n string, b []byte) {
	if err := savedStructureSetRaw(r, n, b); err != nil {
		panic(err)
	}
}
func mustSetRefs(r *sav.DocumentRecordData, n string, refs []uint16) {
	for i := range r.RefSlots {
		if r.RefSlots[i].Name == n {
			r.RefSlots[i].Objects = slices.Clone(refs)
			return
		}
	}
	panic("missing constructor refs " + n)
}
func mustSetCount(r *sav.DocumentRecordData, n string, v uint32) {
	for i := range r.Counts {
		if r.Counts[i].Name == n {
			r.Counts[i].Count = v
			return
		}
	}
	panic("missing constructor count " + n)
}
func mustCountValue(r *sav.DocumentRecordData, n string) uint32 {
	for _, v := range r.Counts {
		if v.Name == n {
			return v.Count
		}
	}
	panic("missing constructor count " + n)
}
func mustSetText(r *sav.DocumentRecordData, n, v string) {
	for i := range r.Texts {
		if r.Texts[i].Name == n {
			r.Texts[i].Value = v
			return
		}
	}
	panic("missing constructor text " + n)
}
func mustNewRecord(class string, flags ...string) sav.DocumentRecordData {
	r, err := sav.NewDocumentRecord(class, flags...)
	if err != nil {
		panic(err)
	}
	return r
}
func constructedPositionBlock(x, y int32, terrainKey uint32) []byte {
	b := make([]byte, 12)
	cell := uint16(x) | uint16(y)<<8
	binary.LittleEndian.PutUint16(b, cell)
	binary.LittleEndian.PutUint16(b[2:], cell)
	b[4], b[5] = 128, 128
	binary.LittleEndian.PutUint32(b[8:], terrainKey)
	return b
}
func mustSetToken(r *sav.DocumentRecordData, raw []byte) {
	mustSetRaw(r, "Block12", raw[:12])
	for _, v := range []sav.DocumentValueData{{Name: "RuntimeID", Value: binary.LittleEndian.Uint32(raw[12:])}, {Name: "T0C", Value: uint32(raw[16])}, {Name: "T0E", Value: uint32(binary.LittleEndian.Uint16(raw[17:]))}, {Name: "T08", Value: binary.LittleEndian.Uint32(raw[19:])}, {Name: "T18", Value: uint32(binary.LittleEndian.Uint16(raw[23:]))}, {Name: "T1C", Value: binary.LittleEndian.Uint32(raw[25:])}, {Name: "Identity", Value: binary.LittleEndian.Uint32(raw[29:])}, {Name: "Reference", Value: binary.LittleEndian.Uint32(raw[33:])}} {
		mustSetValue(r, v.Name, v.Value)
	}
}
func mustNewDiaryRecord(count int, owner uint32) sav.DocumentRecordData {
	r := mustNewRecord("Diary")
	words := make([]byte, 2*count)
	for i := range count {
		binary.LittleEndian.PutUint16(words[2*i:], 1024)
	}
	mustSetRaw(&r, "Journal", make([]byte, 4*count))
	mustSetRaw(&r, "JournalWords", words)
	mustSetCount(&r, "Journal", uint32(count))
	mustSetCount(&r, "JournalWords", uint32(count))
	mustSetValue(&r, "D2C", owner)
	return r
}

func (f *FrontEnd) generatedCampaignChapter(s Snapshot) int {
	return restoreTown(f.Campaign.Value(), s).Chapter()
}
