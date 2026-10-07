package sav

import (
	"encoding/binary"
	"fmt"
)

// ProjectDocumentCampaign shares the city writer's current campaign producer.
// The two unnamed campaign scalar slots remain explicit retained state.
func ProjectDocumentCampaign(source DocumentData, current CampaignProjection) (DocumentData, error) {
	doc, err := CloneDocumentData(source)
	if err != nil {
		return DocumentData{}, err
	}
	campaign := cityFromDataCampaign(doc.Campaign, nil)
	if err := applyCityCampaignProjection(&campaign, current); err != nil {
		return DocumentData{}, err
	}
	doc.Campaign = cityToDataCampaign(campaign, nil)
	return CloneDocumentData(doc)
}

func walkDocumentKeys(r *DocumentRecordData, fn func(*DocumentRecordData)) {
	fn(r)
	for i := range r.Inline {
		walkDocumentKeys(&r.Inline[i].Record, fn)
	}
	for i := range r.Groups {
		walkDocumentKeys(&r.Groups[i], fn)
	}
}

// ReserveDocumentKeys supplies deterministic keys for current constructors and
// key completion. New graph keys must avoid unresolved retained raw references,
// or an allocation would accidentally bind those references.
func ReserveDocumentKeys(doc DocumentData, count int) ([]uint32, error) {
	if count < 0 || count > 1<<16 {
		return nil, fmt.Errorf("sav: invalid identity reservation count %d", count)
	}
	used := map[uint32]bool{0: true}
	// The allocator also excludes every retained scalar and raw dword. This is
	// collision avoidance, not interpretation or reference repair.
	for i := range doc.Objects {
		walkDocumentKeys(&doc.Objects[i], func(r *DocumentRecordData) {
			for _, v := range r.Values {
				used[v.Value] = true
			}
			for _, b := range r.Raw {
				for at := 0; at+4 <= len(b.Bytes); at++ {
					used[binary.LittleEndian.Uint32(b.Bytes[at:])] = true
				}
			}
		})
	}
	if doc.World != nil {
		used[doc.World.TerrainIdentity] = true
		for _, c := range doc.World.Cells {
			p := c.Payload()
			for at := 0; at+4 <= len(p); at++ {
				used[binary.LittleEndian.Uint32(p[at:])] = true
			}
		}
	}
	next := uint32(0x01000000)
	reserved := make([]uint32, count)
	for i := range reserved {
		for used[next] {
			next++
			if next == 0 {
				return nil, fmt.Errorf("sav: identity space exhausted")
			}
		}
		reserved[i] = next
		next++
		if next == 0 && i+1 < count {
			return nil, fmt.Errorf("sav: identity space exhausted")
		}
	}
	return reserved, nil
}

// CompleteDocumentKeys preserves every current ordinary identity. Only an
// absent object key needs allocation; null and unresolved references retain
// their bytes. Identity keys are scoped by SAV class.
const documentTerrainIdentityClass = "\x00world-terrain"

func CompleteDocumentKeys(source DocumentData) (DocumentData, error) {
	doc, err := CloneDocumentData(source)
	if err != nil {
		return DocumentData{}, err
	}
	type typedIdentity struct {
		class string
		key   uint32
	}
	seen := map[typedIdentity]bool{}
	var missing []*DocumentValueData
	for i := range doc.Objects {
		found := false
		for j := range doc.Objects[i].Values {
			v := &doc.Objects[i].Values[j]
			if v.Name != "Identity" && v.Name != "This" {
				continue
			}
			if found {
				return DocumentData{}, fmt.Errorf("sav: object %d has multiple identity fields", i+1)
			}
			found = true
			if v.Value == 0 {
				missing = append(missing, v)
				continue
			}
			identity := typedIdentity{class: doc.Objects[i].Class, key: v.Value}
			if seen[identity] {
				return DocumentData{}, fmt.Errorf("sav: distinct %s objects share identity %#x", doc.Objects[i].Class, v.Value)
			}
			seen[identity] = true
		}
	}
	if doc.World != nil && (doc.World.TerrainIdentity == 0 || seen[typedIdentity{class: documentTerrainIdentityClass, key: doc.World.TerrainIdentity}]) {
		return DocumentData{}, fmt.Errorf("sav: terrain has absent or colliding identity")
	}
	reserved, err := ReserveDocumentKeys(doc, len(missing))
	if err != nil {
		return DocumentData{}, err
	}
	for i, value := range missing {
		value.Value = reserved[i]
	}
	return CloneDocumentData(doc)
}

// RemintDocumentKeys gives every keyed object and terrain a fresh,
// deterministic key. Unlike CompleteDocumentKeys, it needs old keys globally
// unambiguous: rewrites use an untyped key map. Document indices and
// runtime IDs are separate namespaces.
// SAV-PTRMAP-035's missing-reference rule is qualified: a missing order key
// remains missing. Raw spans only exclude allocation collisions; they are never
// interpreted as references or rewritten without a named field program.
func RemintDocumentKeys(source DocumentData) (DocumentData, error) {
	doc, err := CloneDocumentData(source)
	if err != nil {
		return DocumentData{}, err
	}
	reserved, err := ReserveDocumentKeys(doc, len(doc.Objects)+1)
	if err != nil {
		return DocumentData{}, err
	}
	cursor := 0
	mint := func() uint32 {
		key := reserved[cursor]
		cursor++
		return key
	}
	keys := map[uint32]uint32{}
	identities := make([]uint32, len(doc.Objects))
	for i := range doc.Objects {
		for _, v := range doc.Objects[i].Values {
			if v.Name != "Identity" && v.Name != "This" {
				continue
			}
			if identities[i] != 0 {
				return DocumentData{}, fmt.Errorf("sav: object %d has multiple identity fields", i+1)
			}
			if v.Value != 0 && keys[v.Value] != 0 {
				return DocumentData{}, fmt.Errorf("sav: distinct objects share identity %#x", v.Value)
			}
			key := mint()
			identities[i] = key
			if v.Value != 0 {
				keys[v.Value] = key
			}
		}
	}
	if doc.World != nil {
		old := doc.World.TerrainIdentity
		if old == 0 || keys[old] != 0 {
			return DocumentData{}, fmt.Errorf("sav: terrain has absent or colliding identity")
		}
		key := mint()
		keys[old] = key
		doc.World.TerrainIdentity = key
	}
	translate := func(v uint32) uint32 {
		if key := keys[v]; key != 0 {
			return key
		}
		return v
	}
	for i := range doc.Objects {
		walkDocumentKeys(&doc.Objects[i], func(r *DocumentRecordData) {
			for j := range r.Values {
				v := &r.Values[j]
				switch v.Name {
				case "Identity", "This":
					v.Value = identities[i]
				case "Reference", "Hero", "D2C", "G40", "G44", "PE44":
					v.Value = translate(v.Value)
				case "U5C", "U64", "U44", "U40":
					// SAV-DEADLOAD-125 and SAV-908 name these actor lifecycle
					// repairs. They are raw dwords with separate pointer-map
					// consumers, not arbitrary raw spans or Effect+44.
					if r.Class == "Unit" || r.Class == "Human" || r.Class == "Humanoid" {
						v.Value = translate(v.Value)
					}
				}
			}
			for j := range r.Raw {
				b := &r.Raw[j]
				switch b.Name {
				case "Block12":
					if len(b.Bytes) == 12 {
						binary.LittleEndian.PutUint32(b.Bytes[8:], translate(binary.LittleEndian.Uint32(b.Bytes[8:])))
					}
				case "U158":
					if len(b.Bytes) == 148 {
						for _, at := range []int{0x0c, 0x10, 0x18, 0x20, 0x28, 0x30, 0x68} {
							binary.LittleEndian.PutUint32(b.Bytes[at:], translate(binary.LittleEndian.Uint32(b.Bytes[at:])))
						}
					}
				}
			}
		})
	}
	if doc.World != nil {
		for i := range doc.World.Cells {
			c := &doc.World.Cells[i]
			c.GroundActor = translate(c.GroundActor)
			c.AirActor = translate(c.AirActor)
			c.Building = translate(c.Building)
			c.Sack = translate(c.Sack)
			for j := range c.Layers {
				c.Layers[j] = translate(c.Layers[j])
			}
		}
	}
	return CloneDocumentData(doc)
}
