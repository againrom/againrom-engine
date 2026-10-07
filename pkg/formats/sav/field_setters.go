package sav

import (
	"fmt"

	"againrom/pkg/formats/reg"
)

// This file adds setters for the four field groups
// pipeline/SAV-OWNER-RUNS.md names as blocking the field-group transplant
// ladder: the embedded state store's options/view/selection, the campaign
// head's map name and player list dword, and the campaign record's own
// scalars and collections. Every setter here changes ONLY the field it names
// — a byte-level diff for one that cannot change length, a decoded-level
// diff (every OTHER field's own value unchanged, though its byte position
// may move) for one that can.

// ---------------------------------------------------------------------------
// Group 1: the embedded state store (view, options, selection).
// ---------------------------------------------------------------------------

// SetStoreInt writes one scalar leaf of the embedded state store, addressed
// exactly as reg.Reg's own Get* accessors are: a section then a key, e.g.
// SetStoreInt("GameOptions", "Speed", 0). It changes exactly that leaf's own
// 4-byte data word (reg.SetInt); no other Store byte moves.
func (f *File) SetStoreInt(section, key string, v int32) error {
	if f.store == nil {
		return fmt.Errorf("sav: file has no state store")
	}
	out, err := reg.SetInt(f.Store, f.store, section, key, v)
	if err != nil {
		return err
	}
	return f.commitStore(out)
}

// SetStoreIntArray replaces one array leaf of the embedded state store, e.g.
// SetStoreIntArray("Objects", "Selection", nil). The replacement's byte
// length need not equal the old one's, so — unlike SetStoreInt — this can
// also shift the heap-offset word of every OTHER String/IntArray leaf that
// sits after it in the store's own heap (reg.SetIntArray); such a leaf's own
// DECODED value is unchanged, only the word that locates it.
func (f *File) SetStoreIntArray(section, key string, values []int32) error {
	if f.store == nil {
		return fmt.Errorf("sav: file has no state store")
	}
	out, err := reg.SetIntArray(f.Store, f.store, section, key, values)
	if err != nil {
		return err
	}
	return f.commitStore(out)
}

// commitStore re-parses an edited store before adopting it, so a File never
// holds Store bytes its own f.store view disagrees with.
func (f *File) commitStore(out []byte) error {
	parsed, err := reg.Parse(out)
	if err != nil {
		return fmt.Errorf("sav: state store edit produced an unparseable store: %w", err)
	}
	f.Store, f.store = out, parsed
	return nil
}

// ---------------------------------------------------------------------------
// Group 2: the campaign head (SAV-OWNER-RUNS.md's "the head").
// ---------------------------------------------------------------------------

// SetMapName replaces Head.MapName. Its length can differ from the old
// name's, which shifts every body offset after it: the reserved dwords,
// Mission, Difficulty, PlayerListField, PlayerCount, and the whole object
// stream (Players, World, Actors, Trailer) that follows the head. Every one
// of those fields' own DECODED value is unchanged; only their byte position
// moves — exactly the relationship a resized state-store array has to a
// later leaf (SetStoreIntArray).
//
// A length change by an odd number of bytes also flips whether
// SAV-DECPAD-238's single trailing alignment byte belongs at the very end of
// Body (document.go). This setter keeps len(Body) matching that parity
// rather than leaving it wrong by one, which the whole-body length staying
// even cannot by itself reveal — but it does NOT carry the old byte's own
// VALUE across the edit: when the new natural end no longer wants a pad, the
// old one is simply dropped, and when the new natural end newly wants one
// where the old body had none, a fresh literal 0 is written, because
// SAV-DECPAD-238 found no source-pad-to-output-pad copy in ROM1's own writer
// either and the byte's value is otherwise unconstrained. A there-and-back
// pair of odd-delta edits is therefore not a byte no-op on a body whose
// existing alignment byte is non-zero
// (TestSetMapNameOddDeltaPadIsWrittenNotCarried pins this). It re-runs
// f.index() to rebuild every offset-carrying view from the spliced Body,
// the same recompute Open itself does.
func (f *File) SetMapName(name string) error {
	if len(name) >= 0xff {
		return fmt.Errorf("sav: extended map CString is unsupported")
	}
	oldEnd := f.Head.MapNameOff + 1 + len(f.Head.MapName)
	if f.Head.MapNameOff < 0 || oldEnd > len(f.Body) {
		return fmt.Errorf("sav: head map name span [%d,%d) outside a %d-byte body", f.Head.MapNameOff, oldEnd, len(f.Body))
	}
	before, _, err := f.exactDocument()
	if err != nil {
		return fmt.Errorf("sav: cannot locate the body's own natural end: %w", err)
	}
	nameBytes, err := cityAppendCString(nil, name)
	if err != nil {
		return err
	}
	delta := len(name) - len(f.Head.MapName)
	body := make([]byte, 0, len(f.Body)-len(f.Head.MapName)+len(name)+1)
	body = append(body, f.Body[:f.Head.MapNameOff]...)
	body = append(body, nameBytes...)
	body = append(body, f.Body[oldEnd:]...)
	if delta%2 != 0 {
		hadPad := before.end&1 != 0
		switch {
		case hadPad:
			// The suffix carried that pad byte through unchanged; the new
			// natural end no longer wants one, so drop it.
			body = body[:len(body)-1]
		default:
			body = append(body, 0)
		}
	}
	f.Body = body
	return f.index()
}

// SetPlayerListField writes the head's own PlayerListField dword in place;
// nothing else in the body moves. Its own meaning is not decoded (Head docs);
// this setter carries whatever value a caller gives it.
func (f *File) SetPlayerListField(v uint32) error {
	off := f.Head.DifficultyOff + 4
	if off+4 > len(f.Body) {
		return fmt.Errorf("sav: PlayerListField offset %d outside a %d-byte body", off, len(f.Body))
	}
	put32(f.Body, off, v)
	f.Head.PlayerListField = v
	return nil
}

// ---------------------------------------------------------------------------
// Groups 3 and 4: the campaign record's scalars, dwords, arrays, documents
// and children.
// ---------------------------------------------------------------------------

// editCampaign parses TailRest as a campaign record, lets edit mutate the
// detached value, and writes the result back only if edit succeeds.
// serializeCityCampaign never reorders or resizes a field edit did not touch
// (city_campaign.go), so a scalar edit changes only its own fixed-width
// dword and a collection edit reflows only what follows it. It also refreshes
// the cached CampaignProjection (f.campaign/f.campaignErr) exactly as
// splitTail does on Open, so File.Campaign() reflects an edit at once.
func (f *File) editCampaign(edit func(*cityCampaign) error) error {
	if len(f.TailRest) == 0 {
		return fmt.Errorf("sav: file has no campaign record")
	}
	cc, err := parseCityCampaign(f.TailRest)
	if err != nil {
		return fmt.Errorf("sav: campaign record: %w", err)
	}
	if err := edit(cc); err != nil {
		return err
	}
	out, err := serializeCityCampaign(cc)
	if err != nil {
		return err
	}
	f.TailRest = out
	if projection, err := parseCampaignProjection(out); err != nil {
		f.campaign, f.campaignErr = nil, err
	} else {
		f.campaign, f.campaignErr = &projection, nil
	}
	return nil
}

// SetCampaignScalar writes one of the campaign record's seven raw top-level
// scalars, in their file order: 0 SelectedMission, 1 an unnamed +0x114 word,
// 2 AutoGetMission, 3 LastMission, 4 the FirstMapPoint flag (0 or 1), 5
// MissionTime, 6 an unnamed +0x128 word. Indices 1 and 6 have no named field
// in CampaignProjection (campaign_tail.go) and are reachable only here.
func (f *File) SetCampaignScalar(i int, v uint32) error {
	return f.editCampaign(func(cc *cityCampaign) error {
		if i < 0 || i >= len(cc.scalars) {
			return fmt.Errorf("sav: campaign scalar %d, want 0..%d", i, len(cc.scalars)-1)
		}
		if i == 4 && v > 1 {
			return fmt.Errorf("sav: campaign first-MapPoint flag %d, want 0 or 1", v)
		}
		cc.scalars[i] = v
		return nil
	})
}

// SetCampaignBaseDWord writes one of the main campaign record's own six
// dwords, in file order: 0 Mission, 1 MapObject, 2 Payment, 3 ShopMin, 4
// ShopMax, 5 the Announced latch (0 or 1).
func (f *File) SetCampaignBaseDWord(i int, v uint32) error {
	return f.editCampaign(func(cc *cityCampaign) error {
		if i < 0 || i >= len(cc.base.dwords) {
			return fmt.Errorf("sav: campaign base dword %d, want 0..%d", i, len(cc.base.dwords)-1)
		}
		if i == 5 && v > 1 {
			return fmt.Errorf("sav: campaign announced latch %d, want 0 or 1", v)
		}
		cc.base.dwords[i] = v
		return nil
	})
}

// SetCampaignArray replaces one of the campaign record's six top-level
// uint16 lists, in file order: 0 Mercenaries, 1 PermanentMercenaries, 2
// InnNPC, 3 InnMission, 4 TCMission, 5 ShopMission. A length change reflows
// every field that follows it in the record — exactly as an emptied
// Objects/Selection reflows the state store's heap (SetStoreIntArray): every
// other field's own decoded value is unchanged.
func (f *File) SetCampaignArray(i int, values []uint16) error {
	return f.editCampaign(func(cc *cityCampaign) error {
		if i < 0 || i >= len(cc.arrays) {
			return fmt.Errorf("sav: campaign array %d, want 0..%d", i, len(cc.arrays)-1)
		}
		if len(values) > maxCampaignElements {
			return fmt.Errorf("sav: campaign array %d has %d elements, exceeds %d", i, len(values), maxCampaignElements)
		}
		cc.arrays[i] = append([]uint16(nil), values...)
		return nil
	})
}

// SetCampaignDocuments replaces the campaign record's whole persisted
// document list (CampaignDocument: Value plus a 0/1 Kind).
func (f *File) SetCampaignDocuments(docs []CampaignDocument) error {
	return f.editCampaign(func(cc *cityCampaign) error {
		if len(docs) > maxCampaignElements {
			return fmt.Errorf("sav: campaign documents: %d elements exceeds %d", len(docs), maxCampaignElements)
		}
		out := make([][2]uint32, len(docs))
		for i, d := range docs {
			if d.Kind > 1 {
				return fmt.Errorf("sav: campaign document %d kind %d, want 0 or 1", i, d.Kind)
			}
			out[i] = [2]uint32{d.Value, d.Kind}
		}
		cc.documents = out
		return nil
	})
}

// SetCampaignChildren replaces the campaign record's whole child-mission
// list, sharing campaignRecordToBase's field mapping with
// applyCityCampaignProjection so the two writers cannot disagree about it.
func (f *File) SetCampaignChildren(children []CampaignRecord) error {
	return f.editCampaign(func(cc *cityCampaign) error {
		if len(children) > maxCampaignElements {
			return fmt.Errorf("sav: campaign children: %d elements exceeds %d", len(children), maxCampaignElements)
		}
		out := make([]cityCampaignChild, len(children))
		for i, r := range children {
			base, err := campaignRecordToBase(r)
			if err != nil {
				return fmt.Errorf("sav: campaign child %d: %w", i, err)
			}
			out[i] = cityCampaignChild{base: base, age: r.Age}
		}
		cc.children = out
		return nil
	})
}
