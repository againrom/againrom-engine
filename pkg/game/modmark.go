package game

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/rules"
)

// modMarkFormat is the version of the mod mark's payload.
const modMarkFormat = 1

// modMarkName is the leaf name the mark is stored under, as the state store
// spells it. Its presence in a save's bytes is the cheap test for a mark.
var modMarkName = []byte("AgainromMods")

// modMark is the payload of the AgainromMods leaf: the mod set the save was
// written under, and the true value of every actor field that exceeds the
// original game's domain. The ordinary fields of the save hold the projection
// of those values into the original domain.
type modMark struct {
	Format    int    `json:"format"`
	SetDigest string `json:"set_digest"`
	mod.SetRecord
	Domain []modMarkObject `json:"domain,omitempty"`
	// Items are the document objects that hold a mod item's stand-in.
	Items []modMarkItem `json:"items,omitempty"`
	// Layers are the clothing layers party members wear. The document holds
	// the pack unit of each as an ordinary item; only the leaf says it is worn.
	Layers []modMarkLayer `json:"layers,omitempty"`
}

// modMarkLayer is one worn clothing layer: the party member's identity and the
// layer item's own code.
type modMarkLayer struct {
	Member string `json:"member"`
	Code   uint16 `json:"code"`
}

// modMarkLayers lists the layers a party wears, member by member, limited to
// those each member's pack still backs.
func modMarkLayers(party []mapload.PartyMember, t *mapload.Table) []modMarkLayer {
	var out []modMarkLayer
	for _, p := range party {
		for _, code := range mapload.MemberLayers(p, t) {
			out = append(out, modMarkLayer{Member: p.ID, Code: code})
		}
	}
	return out
}

// applyModLayers puts the layers of a mark on the restored party. Each layer
// must name a member of the party and an item an active mod adds as a layer.
func applyModLayers(party []mapload.PartyMember, layers []modMarkLayer, items []mapload.ModItem) error {
	if len(layers) == 0 {
		return nil
	}
	mapload.NameParty(party)
	for _, l := range layers {
		isLayer := false
		for _, m := range items {
			if m.Code == l.Code && m.Layer != 0 {
				isLayer = true
			}
		}
		if !isLayer {
			return modMarkRefusal("the mod mark names layer item %#04x, which no active mod adds as a layer", l.Code)
		}
		i := slices.IndexFunc(party, func(p mapload.PartyMember) bool { return p.ID == l.Member })
		if i < 0 {
			return modMarkRefusal("the mod mark puts a layer on party member %q, whom the save does not have", l.Member)
		}
		party[i].Layers = append(party[i].Layers, l.Code)
	}
	return nil
}

// modMarkItem is one item object of the document that holds the stand-in of a
// mod item: the item's own code and the definition row the object held.
type modMarkItem struct {
	Object int    `json:"object"`
	Code   uint16 `json:"code"`
	Row    uint8  `json:"row,omitempty"`
}

// modMarkObject is the true skill state of one document object, addressed by
// its index in the document.
type modMarkObject struct {
	Object     int       `json:"object"`
	Levels     [6]int32  `json:"levels"`
	BaseLevels [6]int32  `json:"base_levels"`
	XP         [6]uint32 `json:"xp"`
	Experience uint32    `json:"experience"`
}

// Field names and offsets of the skill state in an actor record: the current
// level block, the base level block (u16 per slot after a 2-byte lead), the
// per-slot experience block and the aggregate experience value.
const (
	modLevelBlock = "UA6"
	modBaseBlock  = "U114"
	modXPBlock    = "H1CC"
	modExpValue   = "U130"
)

// originalSkillDomain is the highest level and experience the original game
// stores.
func originalSkillDomain() (level int32, xp uint32) {
	r := rules.Default()
	return r.SkillCap(), uint32(r.SkillXP(r.SkillCap()))
}

func modBlock(r *sav.DocumentRecordData, name string) []byte {
	for _, raw := range r.Raw {
		if raw.Name == name {
			return raw.Bytes
		}
	}
	return nil
}

// readModObject reads the skill state of a document object. ok is false for an
// object that has no skill blocks (not an actor).
func readModObject(r *sav.DocumentRecordData) (m modMarkObject, hasXP, ok bool) {
	levels, base := modBlock(r, modLevelBlock), modBlock(r, modBaseBlock)
	if len(levels) != 24 || len(base) != 24 {
		return m, false, false
	}
	for j := 0; j < 6; j++ {
		m.Levels[j] = int32(int16(binary.LittleEndian.Uint16(levels[2+2*j:])))
		m.BaseLevels[j] = int32(int16(binary.LittleEndian.Uint16(base[2+2*j:])))
	}
	if xp := modBlock(r, modXPBlock); len(xp) == 24 {
		hasXP = true
		for j := 0; j < 6; j++ {
			m.XP[j] = binary.LittleEndian.Uint32(xp[4*j:])
		}
	}
	m.Experience, _ = savedStructureValue(r, modExpValue)
	return m, hasXP, true
}

// project returns the object clamped (the five trainable skill slots; slot 0 is
// not a skill an award can raise) into the original domain: levels at most
// the original cap, experience at most the original curve's end. The aggregate
// experience loses exactly what the slots lost.
func (m modMarkObject) project() modMarkObject {
	maxLevel, maxXP := originalSkillDomain()
	p := m
	var lost uint64
	for j := 1; j < 6; j++ {
		p.Levels[j] = min(m.Levels[j], maxLevel)
		p.BaseLevels[j] = min(m.BaseLevels[j], maxLevel)
		if m.XP[j] > maxXP {
			lost += uint64(m.XP[j] - maxXP)
			p.XP[j] = maxXP
		}
	}
	if lost > uint64(m.Experience) {
		lost = uint64(m.Experience)
	}
	p.Experience = m.Experience - uint32(lost)
	return p
}

// writeModObject stores the skill state in the object's record.
func writeModObject(r *sav.DocumentRecordData, m modMarkObject, hasXP bool) error {
	next := *r
	next.Values = slices.Clone(r.Values)
	next.Raw = slices.Clone(r.Raw)
	levels, err := savedActorRaw(&next, modLevelBlock, 24)
	if err != nil {
		return err
	}
	base, err := savedActorRaw(&next, modBaseBlock, 24)
	if err != nil {
		return err
	}
	for j := 0; j < 6; j++ {
		binary.LittleEndian.PutUint16(levels[2+2*j:], uint16(int16(m.Levels[j])))
		binary.LittleEndian.PutUint16(base[2+2*j:], uint16(int16(m.BaseLevels[j])))
	}
	if hasXP {
		xp, err := savedActorRaw(&next, modXPBlock, 24)
		if err != nil {
			return err
		}
		for j := 0; j < 6; j++ {
			binary.LittleEndian.PutUint32(xp[4*j:], m.XP[j])
		}
	}
	if err := savedActorSetValue(&next, modExpValue, m.Experience); err != nil {
		return err
	}
	*r = next
	return nil
}

// projectModDomain clamps every actor of the document into the original domain
// and returns the true state of each object it changed.
func projectModDomain(doc *sav.DocumentData) ([]modMarkObject, error) {
	var out []modMarkObject
	for i := range doc.Objects {
		tv, hasXP, ok := readModObject(&doc.Objects[i])
		if !ok {
			continue
		}
		proj := tv.project()
		if proj == tv {
			continue
		}
		tv.Object = i
		if err := writeModObject(&doc.Objects[i], proj, hasXP); err != nil {
			return nil, fmt.Errorf("mod mark: object %d: %w", i, err)
		}
		out = append(out, tv)
	}
	return out, nil
}

// markDocumentForMods writes the mod mark into a document that is about to be
// encoded: every actor field is projected into the original domain and the true
// values go into the AgainromMods leaf. Without an active mod set the document
// is left exactly as it is.
func markDocumentForMods(doc *sav.DocumentData, set mod.Set, items []mapload.ModItem, layers []modMarkLayer) error {
	if set.Empty() {
		return nil
	}
	domain, err := projectModDomain(doc)
	if err != nil {
		return err
	}
	standIns, err := standInModItems(doc, items)
	if err != nil {
		return err
	}
	mark := modMark{Format: modMarkFormat, SetDigest: set.Digest(), SetRecord: set.Record(), Domain: domain, Items: standIns, Layers: layers}
	payload, err := json.Marshal(mark)
	if err != nil {
		return err
	}
	return sav.SetNativeMods(&doc.State, payload)
}

// readModMark reads the mark of a document, if it has one.
func readModMark(doc sav.DocumentData) (modMark, bool, error) {
	payload, ok, err := sav.NativeMods(doc.State)
	if err != nil || !ok {
		return modMark{}, false, err
	}
	var mark modMark
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&mark); err != nil {
		return modMark{}, true, fmt.Errorf("the mod mark is unreadable: %v", err)
	}
	if mark.Format != modMarkFormat {
		return modMark{}, true, fmt.Errorf("the mod mark has format %d, this game reads format %d", mark.Format, modMarkFormat)
	}
	return mark, true, nil
}

func modNames(set mod.Set) string {
	var names []string
	for _, m := range set.Mods {
		names = append(names, fmt.Sprintf("%s %s", m.ID, m.Version))
	}
	return strings.Join(names, ", ")
}

// ErrModMark is wrapped by every refusal of a save over its mod mark.
var ErrModMark = errors.New("mod set mismatch")

func modMarkRefusal(format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrModMark}, args...)...)
}

// applyModMark checks a save's mod mark against the active mod set and returns
// the bytes to load: the save itself, or with the true values of every field
// beyond the original domain restored. The rules:
//
//   - no mods active and no mark: the bytes are returned untouched;
//   - no mods active and a mark: refused, naming the mods the save used;
//   - mods active and no mark: refused unless the context accepts unmarked saves;
//   - mods active and a mark of the same set: the true values are restored;
//   - mods active and a mark of another set: refused, naming each difference.
func applyModMark(saved []byte, ctx mapload.ModContext) ([]byte, []modMarkLayer, error) {
	active := ctx.Set
	if active.Empty() && !bytes.Contains(saved, modMarkName) {
		return saved, nil, nil
	}
	doc, err := sav.DecodeDocumentData(saved)
	if err != nil {
		if active.Empty() || ctx.AcceptUnmarked {
			return saved, nil, nil
		}
		return nil, nil, modMarkRefusal("this save has no readable mod mark and mods are active (%s); start with -mods-accept-unmarked to load it anyway", modNames(active))
	}
	mark, present, err := readModMark(doc)
	if err != nil {
		return nil, nil, modMarkRefusal("%v", err)
	}
	if !present {
		if active.Empty() || ctx.AcceptUnmarked {
			return saved, nil, nil
		}
		return nil, nil, modMarkRefusal("this save carries no mod mark and mods are active (%s); start with -mods-accept-unmarked to load it anyway", modNames(active))
	}
	saw, err := mark.SetRecord.Set()
	if err != nil {
		return nil, nil, modMarkRefusal("the mod mark is unreadable: %v", err)
	}
	if active.Empty() {
		return nil, nil, modMarkRefusal("this save was written with mods active (%s); start with the same mods to load it", modNames(saw))
	}
	if diffs := mod.Differences(saw, active); len(diffs) != 0 {
		return nil, nil, modMarkRefusal("this save was written under a different mod set: %s", strings.Join(diffs, "; "))
	}
	if mark.SetDigest != saw.Digest() {
		return nil, nil, modMarkRefusal("the mod mark's digest does not match its own mod list")
	}
	if len(mark.Domain) == 0 && len(mark.Items) == 0 {
		return saved, mark.Layers, nil
	}
	if err := restoreModDomain(&doc, mark.Domain); err != nil {
		return nil, nil, modMarkRefusal("%v", err)
	}
	if err := restoreModItems(&doc, mark.Items, ctx.Items); err != nil {
		return nil, nil, modMarkRefusal("%v", err)
	}
	restored, err := sav.EncodeDocumentData(doc)
	return restored, mark.Layers, err
}

// restoreModDomain puts the true values of the mark back into the document.
// Each object must still hold the projection of the true values it is given, so
// a mark that does not belong to the save is refused.
func restoreModDomain(doc *sav.DocumentData, domain []modMarkObject) error {
	seen := map[int]bool{}
	for _, t := range domain {
		idx := t.Object
		if idx < 0 || idx >= len(doc.Objects) || seen[idx] {
			return fmt.Errorf("the mod mark names object %d, which the save does not have once", idx)
		}
		seen[idx] = true
		t.Object = 0
		now, hasXP, ok := readModObject(&doc.Objects[idx])
		if !ok || now != t.project() {
			return fmt.Errorf("the mod mark does not match the save's actor %d", idx)
		}
		if err := writeModObject(&doc.Objects[idx], t, hasXP); err != nil {
			return err
		}
	}
	return nil
}

// standInModItems puts the stand-in of each mod item in the item objects that
// hold one and returns what the mark needs to put the item back. An item
// object is an Item, Weapon, Armor or Shield record whose code field holds the
// mod item's code; its definition row follows the code.
func standInModItems(doc *sav.DocumentData, items []mapload.ModItem) ([]modMarkItem, error) {
	if len(items) == 0 {
		return nil, nil
	}
	byCode := make(map[uint16]mapload.ModItem, len(items))
	for _, m := range items {
		byCode[m.Code] = m
	}
	var out []modMarkItem
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if !savedItemClass(r.Class) {
			continue
		}
		code, err := savedStructureValue(r, "F40")
		if err != nil {
			continue
		}
		m, ok := byCode[uint16(code)]
		if !ok {
			continue
		}
		next := *r
		next.Values = slices.Clone(r.Values)
		mark := modMarkItem{Object: i, Code: m.Code}
		if err := savedStructureSetValue(&next, "F40", uint32(m.StandInCode)); err != nil {
			return nil, fmt.Errorf("mod mark: object %d: %w", i, err)
		}
		if equipmentRecordClass(r.Class) {
			row, err := savedStructureValue(&next, "T0C")
			if err != nil {
				return nil, fmt.Errorf("mod mark: object %d: %w", i, err)
			}
			mark.Row = uint8(row)
			if err := savedStructureSetValue(&next, "T0C", uint32(m.StandInRow)); err != nil {
				return nil, fmt.Errorf("mod mark: object %d: %w", i, err)
			}
		}
		*r = next
		out = append(out, mark)
	}
	return out, nil
}

// restoreModItems puts the mod items back. Each named object must still hold
// the stand-in the active mods give that item, so a mark that does not belong
// to the save is refused.
func restoreModItems(doc *sav.DocumentData, marks []modMarkItem, items []mapload.ModItem) error {
	byCode := make(map[uint16]mapload.ModItem, len(items))
	for _, m := range items {
		byCode[m.Code] = m
	}
	seen := map[int]bool{}
	for _, mark := range marks {
		idx := mark.Object
		if idx < 0 || idx >= len(doc.Objects) || seen[idx] {
			return fmt.Errorf("the mod mark names item object %d, which the save does not have once", idx)
		}
		seen[idx] = true
		m, ok := byCode[mark.Code]
		if !ok {
			return fmt.Errorf("the mod mark names item code %#04x, which no active mod adds", mark.Code)
		}
		r := &doc.Objects[idx]
		code, err := savedStructureValue(r, "F40")
		if !savedItemClass(r.Class) || err != nil || uint16(code) != m.StandInCode {
			return fmt.Errorf("the mod mark does not match the save's item object %d", idx)
		}
		next := *r
		next.Values = slices.Clone(r.Values)
		if err := savedStructureSetValue(&next, "F40", uint32(m.Code)); err != nil {
			return err
		}
		if equipmentRecordClass(r.Class) {
			row, err := savedStructureValue(&next, "T0C")
			if err != nil || uint8(row) != m.StandInRow {
				return fmt.Errorf("the mod mark does not match the save's item object %d", idx)
			}
			if err := savedStructureSetValue(&next, "T0C", uint32(mark.Row)); err != nil {
				return err
			}
		}
		*r = next
	}
	return nil
}
