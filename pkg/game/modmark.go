package game

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/rules"
	"againrom/pkg/ui"
)

// modMarkFormat is the version of the mod mark's payload.
const modMarkFormat = 1

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

// modMarkLayer is one worn clothing layer: the party member's identity, the
// layer item's own code, and the mod and key that declare the item.
type modMarkLayer struct {
	Member string `json:"member"`
	Code   uint16 `json:"code"`
	Mod    string `json:"mod,omitempty"`
	Key    string `json:"key,omitempty"`
}

// modMarkLayers lists the layers a party wears, member by member, limited to
// those each member's pack still backs.
func modMarkLayers(party []mapload.PartyMember, t *mapload.Table) []modMarkLayer {
	var out []modMarkLayer
	for _, p := range party {
		for _, code := range mapload.MemberLayers(p, t) {
			m, _ := mapload.LayerItem(t, code)
			out = append(out, modMarkLayer{Member: p.ID, Code: code, Mod: m.Mod, Key: m.Key})
		}
	}
	return out
}

// modItemByKey is the active mod item a stamped item or layer names: the item
// its mod declares under its key now. A weapon declares one item per material
// and shape, so the saved code's material and shape choose among them.
func modItemByKey(items []mapload.ModItem, modID, key string, saved uint16) (mapload.ModItem, bool) {
	var found []mapload.ModItem
	for _, m := range items {
		if m.Mod == modID && m.Key == key {
			found = append(found, m)
		}
	}
	if len(found) == 1 {
		return found[0], true
	}
	c := data.ItemCode(saved)
	for _, m := range found {
		if d := data.ItemCode(m.Code); d.A() == c.A() && d.C() == c.C() {
			return m, true
		}
	}
	return mapload.ModItem{}, false
}

// applyModLayers puts the layers of a mark on the restored party. Each layer
// must name a member of the party and an item an active mod adds as a layer.
// Under changed mods a layer takes the code its mod and key have now, and one
// no active mod adds is left off and returned.
func applyModLayers(party []mapload.PartyMember, layers []modMarkLayer, items []mapload.ModItem, changed bool) ([]modMarkLayer, error) {
	if len(layers) == 0 {
		return nil, nil
	}
	mapload.NameParty(party)
	var off []modMarkLayer
	for _, l := range layers {
		i := slices.IndexFunc(party, func(p mapload.PartyMember) bool { return p.ID == l.Member })
		if changed {
			m, _ := modItemByKey(items, l.Mod, l.Key, l.Code)
			l.Code = m.Code
		}
		if _, isLayer := mapload.LayerItemIn(items, l.Code); !isLayer {
			if changed && i >= 0 {
				off = append(off, l)
				continue
			}
			return nil, modMarkRefusal("the mod mark names layer item %#04x, which no active mod adds as a layer", l.Code)
		}
		if i < 0 {
			return nil, modMarkRefusal("the mod mark puts a layer on party member %q, whom the save does not have", l.Member)
		}
		party[i].Layers = append(party[i].Layers, l.Code)
	}
	return off, nil
}

// modMarkItem is one item object of the document that holds the stand-in of a
// mod item: the item's own code, the definition row the object held, and the
// mod and key that declare the item. A mark written before keys has none.
type modMarkItem struct {
	Object int    `json:"object"`
	Code   uint16 `json:"code"`
	Row    uint8  `json:"row,omitempty"`
	Mod    string `json:"mod,omitempty"`
	Key    string `json:"key,omitempty"`
}

// keyed reports that every stamped item and layer names its key.
func (m modMark) keyed() bool {
	return !slices.ContainsFunc(m.Items, func(i modMarkItem) bool { return i.Key == "" }) &&
		!slices.ContainsFunc(m.Layers, func(l modMarkLayer) bool { return l.Key == "" })
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

// project returns the object clamped into the original domain.
func (m modMarkObject) project() modMarkObject { return m.projectTo(originalSkillDomain()) }

// projectTo returns the object clamped (the five trainable skill slots; slot 0
// is not a skill an award can raise) into a domain: levels at most maxLevel,
// experience at most maxXP. The aggregate experience loses exactly what the
// slots lost.
func (m modMarkObject) projectTo(maxLevel int32, maxXP uint32) modMarkObject {
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

// modMarkError is a refusal of a save over its mod mark: the prose every text
// output shows, and the same refusal as data for the Load page's window. The
// differences are mod.Compare's, the list the prose is written from.
type modMarkError struct {
	text   string
	reason ui.ModMismatchReason
	diffs  []mod.Difference
}

func (e *modMarkError) Error() string { return ErrModMark.Error() + ": " + e.text }

func (e *modMarkError) Unwrap() error { return ErrModMark }

// modMismatchKinds maps each difference kind to the window's row kind.
var modMismatchKinds = map[mod.DifferenceKind]ui.ModMismatchKind{
	mod.DifferenceBase:        ui.ModMismatchBase,
	mod.DifferenceMissing:     ui.ModMismatchMissing,
	mod.DifferenceAdded:       ui.ModMismatchAdded,
	mod.DifferenceVersion:     ui.ModMismatchVersion,
	mod.DifferenceFiles:       ui.ModMismatchFiles,
	mod.DifferenceSettingGone: ui.ModMismatchSettingGone,
	mod.DifferenceSetting:     ui.ModMismatchSetting,
	mod.DifferenceSettingNew:  ui.ModMismatchSettingNew,
	mod.DifferenceOrder:       ui.ModMismatchOrder,
}

// ModMismatch is the refusal as the Load page's window lists it.
func (e *modMarkError) ModMismatch() ui.ModMismatch {
	out := ui.ModMismatch{Reason: e.reason}
	for _, d := range e.diffs {
		out.Rows = append(out.Rows, ui.ModMismatchRow{Kind: modMismatchKinds[d.Kind], Mod: d.Mod, Setting: d.Setting, Saved: d.Saved, Current: d.Current})
	}
	return out
}

// modMarkRefusal refuses a mark that cannot be read or does not fit the save.
func modMarkRefusal(format string, args ...any) error {
	return &modMarkError{text: fmt.Sprintf(format, args...), reason: ui.ModMismatchMark}
}

// unmarkedRefusal refuses a save without a readable mark while mods are
// active: each active mod is one the save did not use.
func unmarkedRefusal(format string, active mod.Set) error {
	return &modMarkError{text: fmt.Sprintf(format, modNames(active)), reason: ui.ModMismatchUnmarked,
		diffs: mod.Compare(mod.Set{Base: active.Base}, active)}
}

// applyModMark checks a save's mod mark against the active mod set (only the
// mods that change game state, mod.Set.Marked) and returns the bytes to load,
// with the true values beyond the original domain restored. The rules:
//   - no mods active and no mark: the bytes are returned untouched;
//   - no mods active and a mark: refused, naming the mods the save used;
//   - mods active and no mark: refused unless the context accepts unmarked saves;
//   - mods active and a mark of the same set: the true values are restored;
//   - a mark that names a mod now state-free counts that mod while it is active;
//   - a town save whose mods changed only files, versions or settings (townAdmits):
//     its items matched by mod and key, refused if its mark names none;
//   - mods active and a mark of another set: refused, naming each difference.
func applyModMark(saved []byte, ctx mapload.ModContext, skills rules.Rules) (modMarkLoad, error) {
	active := ctx.Set.Marked()
	if active.Empty() && !sav.MayCarryNativeMods(saved) {
		return modMarkLoad{saved: saved}, nil
	}
	doc, err := sav.DecodeDocumentData(saved)
	if err != nil {
		if active.Empty() || ctx.AcceptUnmarked {
			return modMarkLoad{saved: saved}, nil
		}
		return modMarkLoad{}, unmarkedRefusal("this save has no readable mod mark and mods are active (%s); start with -mods-accept-unmarked to load it anyway", active)
	}
	mark, present, err := readModMark(doc)
	if err != nil {
		return modMarkLoad{}, modMarkRefusal("%v", err)
	}
	if !present {
		if active.Empty() || ctx.AcceptUnmarked {
			return modMarkLoad{saved: saved}, nil
		}
		return modMarkLoad{}, unmarkedRefusal("this save carries no mod mark and mods are active (%s); start with -mods-accept-unmarked to load it anyway", active)
	}
	saw, err := mark.SetRecord.Set()
	if err != nil {
		return modMarkLoad{}, modMarkRefusal("the mod mark is unreadable: %v", err)
	}
	// A mod the save names stays in the comparison while it is active, even if
	// it changes no state now: the save was marked under it.
	named := map[string]bool{}
	for _, m := range saw.Mods {
		named[m.ID] = true
	}
	active = mod.Set{Base: ctx.Set.Base}
	for _, m := range ctx.Set.Mods {
		if !m.StateFree || named[m.ID] {
			active.Mods = append(active.Mods, m)
		}
	}
	if active.Empty() {
		return modMarkLoad{}, &modMarkError{reason: ui.ModMismatchSet, diffs: mod.Compare(saw, mod.Set{Base: saw.Base}),
			text: fmt.Sprintf("this save was written with mods active (%s); start with the same mods to load it", modNames(saw))}
	}
	diffs := mod.Compare(saw, active)
	changed := len(diffs) != 0
	if changed && !townAdmits(saved, saw, active, diffs) {
		return modMarkLoad{}, &modMarkError{reason: ui.ModMismatchSet, diffs: diffs,
			text: "this save was written under a different mod set: " + strings.Join(mod.Differences(saw, active), "; ")}
	}
	if mark.SetDigest != saw.Digest() {
		return modMarkLoad{}, modMarkRefusal("the mod mark's digest does not match its own mod list")
	}
	if changed && !mark.keyed() {
		return modMarkLoad{}, &modMarkError{reason: ui.ModMismatchKeyless, diffs: diffs,
			text: "this save was written before saves named their mod items, so its items cannot be matched under the changed mods: " +
				strings.Join(mod.Differences(saw, active), "; ")}
	}
	load := modMarkLoad{saved: saved, layers: mark.Layers, changed: changed}
	if len(mark.Domain) == 0 && len(mark.Items) == 0 {
		return load, nil
	}
	var limit *modSkillDomain
	if changed {
		limit = &modSkillDomain{level: skills.SkillCap(), xp: uint32(skills.SkillXP(skills.SkillCap()))}
	}
	if load.lowered, err = restoreModDomain(&doc, mark.Domain, limit); err != nil {
		return modMarkLoad{}, modMarkRefusal("%v", err)
	}
	if load.gone, err = restoreModItems(&doc, mark.Items, ctx.Items, changed); err != nil {
		return modMarkLoad{}, modMarkRefusal("%v", err)
	}
	load.saved, err = sav.EncodeDocumentData(doc)
	return load, err
}

// modMarkLoad is what a save's mod mark gives its LOAD: the bytes to read and
// the worn layers. changed is a town save admitted under mods whose files,
// versions or settings changed; lowered and gone are what the active mods no
// longer hold.
type modMarkLoad struct {
	saved   []byte
	layers  []modMarkLayer
	changed bool
	lowered []modLoweredSkill
	// gone are the stamped item objects whose mod and key no active mod
	// declares now; each still holds its stand-in, and the town takes it away.
	gone []modMarkItem
	// dropped lists the campaign missions and waiting guests the campaign
	// decode left out (pruneUndeclaredMissions, adoptChangedGuests).
	dropped []ui.ModMismatchRow
}

// modLoweredSkill is one skill slot of a document actor that the active cap
// lowered: the level the save held and the level it holds now.
type modLoweredSkill struct {
	Object, Slot int
	Saved, Now   int32
}

// modSkillDomain is the highest level and experience a skill slot holds.
type modSkillDomain struct {
	level int32
	xp    uint32
}

// townAdmits reports that a save whose mods differ from the active ones
// loads all the same: the same base and the same mods in the same order, each
// difference another version, other files or a setting, and a save made in
// town. Another mod or another order is refused, as is any save made inside
// a mission.
func townAdmits(saved []byte, saw, active mod.Set, diffs []mod.Difference) bool {
	if saw.Base != active.Base || len(saw.Mods) != len(active.Mods) {
		return false
	}
	for i := range saw.Mods {
		if saw.Mods[i].ID != active.Mods[i].ID {
			return false
		}
	}
	for _, d := range diffs {
		switch d.Kind {
		case mod.DifferenceVersion, mod.DifferenceFiles, mod.DifferenceSetting, mod.DifferenceSettingGone, mod.DifferenceSettingNew:
		default:
			return false
		}
	}
	sf, err := sav.Open(saved)
	return err == nil && sf.Head.Mission == 0
}

// restoreModDomain puts the true values of the mark back into the document.
// Each object must still hold the projection of the true values it is given, so
// a mark that does not belong to the save is refused. With a cap, the values
// are held to it and each slot it lowered is returned.
func restoreModDomain(doc *sav.DocumentData, domain []modMarkObject, limit *modSkillDomain) ([]modLoweredSkill, error) {
	seen := map[int]bool{}
	var lowered []modLoweredSkill
	for _, t := range domain {
		idx := t.Object
		if idx < 0 || idx >= len(doc.Objects) || seen[idx] {
			return nil, fmt.Errorf("the mod mark names object %d, which the save does not have once", idx)
		}
		seen[idx] = true
		t.Object = 0
		now, hasXP, ok := readModObject(&doc.Objects[idx])
		if !ok || now != t.project() {
			return nil, fmt.Errorf("the mod mark does not match the save's actor %d", idx)
		}
		if limit != nil {
			held := t.projectTo(limit.level, limit.xp)
			for j := 1; j < 6; j++ {
				if saved := max(t.Levels[j], t.BaseLevels[j]); held.Levels[j] != t.Levels[j] || held.BaseLevels[j] != t.BaseLevels[j] {
					lowered = append(lowered, modLoweredSkill{Object: idx, Slot: j, Saved: saved, Now: max(held.Levels[j], held.BaseLevels[j])})
				}
			}
			t = held
		}
		if err := writeModObject(&doc.Objects[idx], t, hasXP); err != nil {
			return nil, err
		}
	}
	return lowered, nil
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
		mark := modMarkItem{Object: i, Code: m.Code, Mod: m.Mod, Key: m.Key}
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
// to the save is refused. Under changed mods each object takes the code and
// row its mod and key have now (modItemByKey), whatever its stand-in; an item
// no active mod declares keeps the stand-in, and its mark is returned.
func restoreModItems(doc *sav.DocumentData, marks []modMarkItem, items []mapload.ModItem, changed bool) ([]modMarkItem, error) {
	byCode := make(map[uint16]mapload.ModItem, len(items))
	for _, m := range items {
		byCode[m.Code] = m
	}
	seen := map[int]bool{}
	var gone []modMarkItem
	for _, mark := range marks {
		idx := mark.Object
		if idx < 0 || idx >= len(doc.Objects) || seen[idx] {
			return nil, fmt.Errorf("the mod mark names item object %d, which the save does not have once", idx)
		}
		seen[idx] = true
		m, ok := byCode[mark.Code]
		row := mark.Row
		if changed {
			m, ok = modItemByKey(items, mark.Mod, mark.Key, mark.Code)
			row = m.Row
		} else if !ok {
			return nil, fmt.Errorf("the mod mark names item code %#04x, which no active mod adds", mark.Code)
		}
		r := &doc.Objects[idx]
		code, err := savedStructureValue(r, "F40")
		if !savedItemClass(r.Class) || err != nil || !changed && uint16(code) != m.StandInCode {
			return nil, fmt.Errorf("the mod mark does not match the save's item object %d", idx)
		}
		if !ok {
			gone = append(gone, mark)
			continue
		}
		next := *r
		next.Values = slices.Clone(r.Values)
		if err := savedStructureSetValue(&next, "F40", uint32(m.Code)); err != nil {
			return nil, err
		}
		if equipmentRecordClass(r.Class) {
			held, err := savedStructureValue(&next, "T0C")
			if err != nil || !changed && uint8(held) != m.StandInRow {
				return nil, fmt.Errorf("the mod mark does not match the save's item object %d", idx)
			}
			if err := savedStructureSetValue(&next, "T0C", uint32(row)); err != nil {
				return nil, err
			}
		}
		*r = next
	}
	return gone, nil
}
