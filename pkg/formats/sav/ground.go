package sav

import "fmt"

// GroundSack is one top-level Sack, not a dead actor's container. Identity is
// local to this archive. Position uses the packed cell that Sack placement
// reads (SAV-POSLOAD-140); the two fine bytes remain available to a caller.
//
// InsertIndex and Accumulator are the Contents container's own +0x1c/+0x20
// pair (ITEM-SAVE-014: "the CObList elements, then +0x1c, then +0x20 — so
// insert index and load are stored, not recomputed"). No claim states a
// rebuild rule for either value against this build's own ground-sack
// mechanics (sim.Sack's own pourSack always appends at the live tail), so
// they are carried, never enacted (docs/DIVERGENCES.md).
// Token1C is the Sack's own `+0x1c` value slot (ITEM-SACK-010): gold plus
// the sum of every carried item's own price, recomputed by the original's
// own constructor rather than read back out of the container.
type GroundSack struct {
	Identity    uint32
	Cell        uint16
	FineX       uint8
	FineY       uint8
	Gold        uint32
	Items       []Piece
	InsertIndex uint32
	Accumulator int32
	Token1C     uint32
}

// GroundSacks walks the document in serialization order through the Sack list.
// It never scans for a class name or treats a corpse's inventory as ground
// loot. The boolean distinguishes a no-world document from an empty world
// list: the latter must replace, rather than retain, a fresh map's sacks.
//
// SAV-DOC-053 fixes the five top-level collections and the common trailer;
// SAV-TERRKEY-056 fixes the two intervening counted terrain regions. Their
// payloads are skipped by their published widths, not interpreted as live
// terrain or session state. This is a reader, not a world-authoring programme.
func (f *File) GroundSacks() ([]GroundSack, bool, error) {
	doc, present, err := f.exactDocument()
	if err != nil || !present {
		return nil, present, err
	}
	return groundSacks(doc, present)
}

func groundSacks(doc *document, present bool) ([]GroundSack, bool, error) {
	records := uniqueSackRecords(doc.sacks)
	out := make([]GroundSack, 0, len(records))
	identities := make(map[uint32]bool, len(records))
	for i, record := range records {
		identity := record.value("Identity")
		if identity == 0 || identities[identity] {
			return nil, true, fmt.Errorf("sav: Sack %d has zero or repeated identity %#x", i, identity)
		}
		identities[identity] = true
		position := record.Raw["Block12"]
		if len(position) != 12 {
			return nil, true, fmt.Errorf("sav: Sack %d has no complete Position", i)
		}
		sack := GroundSack{Identity: identity, Cell: u16(position, 2),
			FineX: position[4], FineY: position[5], Gold: record.value("S3C"),
			InsertIndex: record.value("Contents1C"), Accumulator: int32(record.value("Contents20")),
			Token1C: record.value("T1C")}
		items := record.Refs["Contents"]
		if len(items) != record.Counts["Contents"] {
			return nil, true, fmt.Errorf("sav: Sack %d contains a null item reference", i)
		}
		for j, item := range items {
			if !groundClass(item.Class, "Item") {
				return nil, true, fmt.Errorf("sav: Sack %d item %d has class %s", i, j, item.Class)
			}
			if len(item.Refs["Effects"]) != item.Counts["Effects"] {
				return nil, true, fmt.Errorf("sav: Sack %d item %d contains a null effect reference", i, j)
			}
			// A non-"Effect" reference (Effect_DirectDamage: SAV-EQUIPEFFECT-553
			// names both classes on an item, but no claim states either is
			// expected here specifically) is per-item unsupported, on piece's
			// own ground — never a reason to refuse the rest of this Sack.
			sack.Items = append(sack.Items, piece(item))
		}
		out = append(out, sack)
	}
	return out, true, nil
}

// Repeated archive roots still name one physical Sack. Distinct records with
// equal values remain distinct and undergo the ordinary identity validation.
func uniqueSackRecords(roots []*Record) []*Record {
	seen := make(map[*Record]bool, len(roots))
	records := make([]*Record, 0, len(roots))
	for _, record := range roots {
		if !seen[record] {
			seen[record] = true
			records = append(records, record)
		}
	}
	return records
}

func (w *walker) groundCountedList(name, class string, refOffsets ...*[]int) ([]*Record, error) {
	n, err := w.u32()
	if err != nil {
		return nil, err
	}
	return w.groundList(name, n, class, refOffsets...)
}

func (w *walker) groundList(name string, n uint32, class string, refOffsets ...*[]int) ([]*Record, error) {
	if n > maxListElements || uint64(n) > uint64((len(w.b)-w.p)/2) {
		return nil, fmt.Errorf("sav: %s count %d exceeds bounded remaining data", name, n)
	}
	out := make([]*Record, 0, int(n))
	for i := uint32(0); i < n; i++ {
		for _, offsets := range refOffsets {
			*offsets = append(*offsets, w.p)
		}
		record, err := w.object(0)
		if err != nil {
			return nil, fmt.Errorf("sav: %s %d: %w", name, i, err)
		}
		if record == nil || !groundClass(record.Class, class) {
			return nil, fmt.Errorf("sav: %s %d is null or not a %s", name, i, class)
		}
		out = append(out, record)
	}
	return out, nil
}

func groundClass(class, base string) bool {
	if class == base {
		return true
	}
	switch base {
	case "Unit":
		return class == "Human" || class == "Humanoid"
	case "Building":
		return class == "Outpost" || class == "Tavern" || class == "Shop"
	case "SpellEffect":
		return class == "PointEffect" || class == "AreaEffect" || class == "SpellTransport"
	case "Item":
		return class == "Weapon" || class == "Armor" || class == "Shield"
	}
	return false
}

// GroundContainerTail is one Sack record's own +0x1c/+0x20 pair, matched to
// its file record by Identity. SetGroundContainerTails' own input shape is
// deliberately narrower than GroundSack: ITEM-SAVE-014 names no LOAD-time
// creation, removal or relocation of a Sack or its Items, so the container
// tail is the only Sack content this package ever rewrites — SetSpellEffects'
// own standing for its own narrower LOAD-time guarantee, restated here for a
// different record shape.
type GroundContainerTail struct {
	Identity    uint32
	InsertIndex uint32
	Accumulator int32
}

// SetGroundContainerTails patches every Sack's own container-tail dwords in
// place. Matching is positional, by GroundSacks' own file-order return, with
// an Identity cross-check per entry: GroundSacks already refuses distinct Sacks
// sharing one Identity, so a mismatch here means the caller's own list is out
// of step with this file, not a legitimate rewrite target.
//
// It is a content patch, not a re-encoder, on SetSpellEffects' own contract:
// the archive tag, class-name and index bytes the Contents list already
// carries are never rewritten, and no Sack, Item or Effect record is added,
// removed or relocated.
func (f *File) SetGroundContainerTails(tails []GroundContainerTail) error {
	doc, present, err := f.exactDocument()
	if err != nil {
		return err
	}
	if !present {
		return fmt.Errorf("sav: this save has no world session")
	}
	records := uniqueSackRecords(doc.sacks)
	if len(tails) != len(records) {
		return fmt.Errorf("sav: %d container tails for %d Sack records", len(tails), len(records))
	}
	offsets := make([]int, len(records))
	for i, r := range records {
		t := tails[i]
		if identity := r.value("Identity"); identity != t.Identity {
			return fmt.Errorf("sav: Sack %d identity %#x does not match tail identity %#x", i, identity, t.Identity)
		}
		off, ok := r.ContainerTailOff["Contents"]
		if !ok {
			return fmt.Errorf("sav: Sack %d has no recorded container tail offset", i)
		}
		offsets[i] = off
	}
	for i, t := range tails {
		put32(f.Body, offsets[i], t.InsertIndex)
		put32(f.Body, offsets[i]+4, uint32(t.Accumulator))
	}
	return nil
}
