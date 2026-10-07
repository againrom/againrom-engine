package game

import (
	"fmt"
	"reflect"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// SnapshotOriginalCity is optional provenance, not canonical simulation state.
// Its baseline fields admit supported source-backed training and city updates.
// Other changed Human/item/roster/session state still requires the native format.
// Only named semantic fields and local object indices are persisted.
type SnapshotOriginalCity struct {
	Version                uint32
	Document               sav.CityData
	Bindings               []SnapshotCityBinding
	Offered                int
	Difficulty             mapload.Difficulty
	CampaignMarkerSelected bool
	// WorldSelectedOnce is a retained wire name; nothing reads it.
	WorldSelectedOnce []int
	Unavailable       string
	// loadedState is the loaded town file's state-store records when its
	// provenance is unavailable, so a save still writes the file's own
	// options and view. It is not persisted.
	loadedState []sav.CityStateRecordData
}

type SnapshotCityBinding struct {
	Identity uint32
	PartyID  string
	Baseline mapload.PartyMember
	Training []uint8
	Sales    []SnapshotCitySale
	// 0 preserves published pre-producer history without guessing Sell groups.
	SalesVersion uint8
	Returned     *SnapshotCityReturn
}

// Baseline policies: 1 template book; 2 saved book; 3 typed book; 4 signed item
// weight; 5 source actor/container; 6 regeneration residues; 7 NPC identities;
// 8 current item spell effects. Old policies never overwrite current progress.
const currentCityPartyImportVersion = 8

func (state *originalCitySaveState) snapshot() *SnapshotOriginalCity {
	if state == nil {
		return nil
	}
	version := state.partyImportVersion
	if version == 0 {
		version = currentCityPartyImportVersion
	}
	out := &SnapshotOriginalCity{Version: version, Offered: state.baselineOffered,
		Difficulty: state.baselineDifficulty, CampaignMarkerSelected: state.baselineCampaignMarkerSelected}
	if state.unavailable != nil {
		out.Unavailable = state.unavailable.Error()
		loaded := state.loadedState
		if document, ok := state.document.(interface{ Data() sav.CityData }); ok && loaded == nil {
			loaded = document.Data().State.ValueRecords
		}
		out.loadedState = append([]sav.CityStateRecordData(nil), loaded...)
		return out
	}
	// Production documents implement this seam. Test doubles for SAVE dispatch
	// can remain non-persistable and cannot masquerade as semantic provenance.
	document, ok := state.document.(interface{ Data() sav.CityData })
	if !ok {
		return nil
	}
	out.Document = document.Data()
	for _, binding := range state.bindings {
		out.Bindings = append(out.Bindings, SnapshotCityBinding{Identity: binding.character.Identity,
			PartyID: binding.partyID, Baseline: mapload.CloneParty([]mapload.PartyMember{binding.baseline})[0], Training: append([]uint8(nil), binding.training...), Sales: cloneCitySales(binding.sales), SalesVersion: binding.salesVersion, Returned: cloneCityReturn(binding.returned)})
	}
	return out
}

// originalCityFromSnapshot prepares structural provenance without install
// context. Native LOAD must also call validateOriginalCityBaseline before
// installCandidate may adopt it. Current party edits do not invalidate native
// saves; marshal separately refuses to export them. A baseline assertion is
// never made trustworthy merely by matching the current party.
func originalCityFromSnapshot(s Snapshot) (*originalCitySaveState, error) {
	dto := s.OriginalCity
	if dto == nil {
		return nil, nil
	}
	if dto.Version < 1 || dto.Version > currentCityPartyImportVersion {
		return nil, fmt.Errorf("original city snapshot version %d, want 1 or %d", dto.Version, currentCityPartyImportVersion)
	}
	if err := boundCitySnapshot(reflect.ValueOf(*dto), new(citySnapshotBudget), 0); err != nil {
		return nil, err
	}
	// Mission AGS keeps the same immutable city source and the previous return
	// graph. The World carries current mission item identities independently.
	if (s.Mission == 0) != (len(s.World) == 0) {
		return nil, fmt.Errorf("original city provenance has inconsistent mission/world presence")
	}
	if _, err := campaignDifficulty(int64(dto.Difficulty)); err != nil {
		return nil, err
	}
	if _, err := worldSelectedOnceFromSnapshot(dto.WorldSelectedOnce); err != nil {
		return nil, err
	}
	if len(dto.Unavailable) > maxLabel {
		return nil, fmt.Errorf("original city unavailable reason exceeds %d bytes", maxLabel)
	}
	state := &originalCitySaveState{baselineOffered: dto.Offered, baselineDifficulty: dto.Difficulty,
		partyImportVersion:             dto.Version,
		baselineCampaignMarkerSelected: dto.CampaignMarkerSelected}
	if dto.Unavailable != "" {
		if !reflect.DeepEqual(dto.Document, sav.CityData{}) || len(dto.Bindings) != 0 {
			return nil, fmt.Errorf("unavailable original city carries a document or bindings")
		}
		state.unavailable = fmt.Errorf("%s", dto.Unavailable)
		return state, nil
	}
	document, err := sav.CityFromData(dto.Document)
	if err != nil {
		return nil, fmt.Errorf("original city provenance: %w", err)
	}
	state.baselineQuickSpells, err = originalCityQuickSpells(document)
	if err != nil {
		return nil, err
	}
	// These are produced by original city import, not free-form authorable
	// session fields. A forged baseline must not hide a changed session latch.
	sourceDifficulty, err := campaignDifficulty(int64(dto.Document.Head[12]))
	if err != nil {
		return nil, err
	}
	baselineDifficulty, _ := campaignDifficulty(int64(dto.Difficulty))
	if dto.Offered != 0 || baselineDifficulty != sourceDifficulty ||
		dto.CampaignMarkerSelected != (len(dto.Document.Campaign.Markers) != 0) {
		return nil, fmt.Errorf("original city session baseline differs from source import")
	}
	roster := document.Roster()
	if len(dto.Bindings) != len(roster) {
		return nil, fmt.Errorf("original city has %d bindings for %d characters", len(dto.Bindings), len(roster))
	}
	byID := make(map[uint32]sav.CityCharacter, len(roster))
	for _, c := range roster {
		byID[c.Identity] = c
	}
	seen := make(map[string]bool, len(roster))
	for _, binding := range dto.Bindings {
		c, ok := byID[binding.Identity]
		if !ok {
			return nil, fmt.Errorf("original city binding has foreign or repeated identity %#x", binding.Identity)
		}
		delete(byID, binding.Identity)
		if binding.PartyID == "" || len(binding.PartyID) > maxLabel || binding.PartyID != binding.Baseline.ID || seen[binding.PartyID] {
			return nil, fmt.Errorf("original city binding has invalid party ID %q", binding.PartyID)
		}
		if c.Hero != binding.Baseline.StartingHero || c.Name != binding.Baseline.Name {
			return nil, fmt.Errorf("original city binding %q does not identify its source character", binding.PartyID)
		}
		seen[binding.PartyID] = true
		if len(binding.Training) > maxCityTraining {
			return nil, fmt.Errorf("original city training exceeds bound")
		}
		if binding.SalesVersion > 1 {
			return nil, fmt.Errorf("original city sale history version %d", binding.SalesVersion)
		}
		for _, slot := range binding.Training {
			if slot < 1 || slot > 5 {
				return nil, fmt.Errorf("original city training contains invalid skill slot %d", slot)
			}
		}
		b := originalCityBinding{character: c, partyID: binding.PartyID,
			baseline: mapload.CloneParty([]mapload.PartyMember{binding.Baseline})[0], training: append([]uint8(nil), binding.Training...), sales: cloneCitySales(binding.Sales), salesVersion: binding.SalesVersion, partyImportVersion: dto.Version}
		bindCityInventory(document, &b)
		b.returned = cloneCityReturn(binding.Returned)
		if b.returned != nil {
			if dto.Version < 6 {
				return nil, fmt.Errorf("legacy city snapshot has a current return")
			}
			if err := b.checkReturn(b.returned); err != nil {
				return nil, err
			}
		}
		if len(b.sales) != 0 {
			if _, err := citySaleRemainders(b, b.sales); err != nil {
				return nil, err
			}
		}
		state.bindings = append(state.bindings, b)
	}
	if err := validateCityPartyIDs(dto, document); err != nil {
		return nil, err
	}
	state.document = document
	return state, nil
}

// Envelopes can check identity construction without an install; the registry
// interpretation itself is verified independently by validateOriginalCityBaseline.
func validateCityPartyIDs(dto *SnapshotOriginalCity, document *sav.CityProvenance) error {
	characters, err := document.SourceParty()
	if err != nil {
		return err
	}
	characters, _, _ = leadFirst(characters)
	byKey := make(map[uint32]SnapshotCityBinding, len(dto.Bindings))
	for _, binding := range dto.Bindings {
		byKey[binding.Identity] = binding
	}
	party := make([]mapload.PartyMember, len(characters))
	for i, c := range characters {
		binding, ok := byKey[c.Key]
		if !ok {
			return fmt.Errorf("original city source character has no identity binding")
		}
		party[i] = binding.Baseline
		party[i].ID = ""
		if dto.Version < 7 {
			party[i].ID, party[i].CompanionNPC = legacyCityPartyIdentity(c)
		}
	}
	mapload.NameParty(party)
	for i, c := range characters {
		if byKey[c.Key].PartyID != party[i].ID {
			return fmt.Errorf("original city binding has foreign source party identity; want %q", party[i].ID)
		}
	}
	return nil
}

func legacyCityPartyIdentity(c sav.Character) (string, int) {
	if c.Hero {
		return "hero", 0
	}
	if c.DefRow == 28 || c.DefRow == 29 {
		return "npc:22", 22
	}
	return "", 0
}

// validateOriginalCityBaseline runs before candidate publication. Saved
// baselines are untrusted assertions: reconstruct ALL member fields using the
// same source characters, table and body context as RestoreOriginal, then
// require exact equality. Current party edits remain valid native progress;
// they cannot change the reconstructed baseline used by the SAV equality gate.
func (in townInstall) validateOriginalCityBaseline(state *originalCitySaveState) error {
	if state == nil || state.unavailable != nil {
		return nil
	}
	document, ok := state.document.(*sav.CityProvenance)
	if !ok {
		return fmt.Errorf("original city baseline has no semantic source")
	}
	characters, err := document.SourceParty()
	if err != nil {
		return fmt.Errorf("original city baseline source: %w", err)
	}
	// restorePartyCharacters filters/reorders its input, so keep an independent
	// source slice for identity binding. A fallback is never export eligible.
	restored, report := restorePartyCharacters(append([]sav.Character(nil), characters...), nil, nil, in.bodies, in.table)
	if report.Err != nil || report.Fallback {
		return fmt.Errorf("original city baseline cannot restore its source roster: %v", report.Err)
	}
	persistent := characters[:0]
	for _, character := range characters {
		if persistentOriginalCharacter(character, in.table) {
			persistent = append(persistent, character)
		}
	}
	persistent, _, _ = leadFirst(persistent)
	if len(restored) != len(persistent) {
		return fmt.Errorf("original city baseline roster differs from source")
	}
	if state.partyImportVersion < 7 {
		for i, c := range persistent {
			restored[i].ID, restored[i].CompanionNPC = legacyCityPartyIdentity(c)
		}
	}
	mapload.NameParty(restored)
	clearRestoredMissionPosition(restored)
	if state.partyImportVersion == 1 {
		if len(restored) != len(persistent) {
			return fmt.Errorf("original city legacy baseline roster differs from source")
		}
		for i, character := range persistent {
			restored[i].KnownSpells = legacyOriginalCityBook(character, in.table)
			restored[i].SpellbookRestored = false
			restored[i].SpellbookPresent = false
		}
	}
	if state.partyImportVersion < 3 {
		for i := range restored {
			restored[i].Book = sim.Spellbook{}
		}
	}
	if state.partyImportVersion < 5 {
		for i := range restored {
			clearLegacySourceEquipment(&restored[i])
			if restored[i].Carry != nil {
				restored[i].Carry.LiveLoad, restored[i].Carry.OrderedStacks = nil, nil
			}
		}
	}
	if state.partyImportVersion < 4 {
		for i := range restored {
			clearLegacyItemWeights(&restored[i])
		}
	}
	if state.partyImportVersion < 6 {
		for i := range restored {
			if c := restored[i].Carry; c != nil && c.LiveLoad != nil {
				c.LiveLoad.HealthHundredths, c.LiveLoad.ManaHundredths = 0, 0
			}
		}
	}
	expected := bindOriginalCity(document, persistent, restored, nil, in.table)
	if expected.unavailable != nil {
		return fmt.Errorf("original city baseline import context: %w", expected.unavailable)
	}
	byID := make(map[uint32]originalCityBinding, len(expected.bindings))
	for i := range expected.bindings {
		expected.bindings[i].partyImportVersion = state.partyImportVersion
		binding := expected.bindings[i]
		byID[binding.character.Identity] = binding
	}
	for _, saved := range state.bindings {
		want, ok := byID[saved.character.Identity]
		comparison := want.baseline
		if saved.baseline.OriginalHuman == nil && len(saved.training) == 0 {
			comparison.OriginalHuman = nil // legacy source baseline, not a current-state repair
		}
		if !ok || saved.partyID != want.partyID {
			return fmt.Errorf("original city baseline for %q differs from semantic source/import context", saved.partyID)
		}
		if state.partyImportVersion < 8 && !reflect.DeepEqual(saved.baseline, comparison) {
			legacy := legacyCityWeaponBaseline(comparison, in.table)
			if reflect.DeepEqual(saved.baseline, legacy) {
				want.baseline = legacyCityWeaponBaseline(want.baseline, in.table)
				comparison = legacy
			}
		}
		if !reflect.DeepEqual(saved.baseline, comparison) {
			if legacy := legacyCityLoadSpeed(comparison); reflect.DeepEqual(saved.baseline, legacy) {
				comparison = legacy
			}
		}
		if !reflect.DeepEqual(saved.baseline, comparison) {
			a, b := reflect.ValueOf(saved.baseline), reflect.ValueOf(comparison)
			for i := 0; i < a.NumField(); i++ {
				if !reflect.DeepEqual(a.Field(i).Interface(), b.Field(i).Interface()) {
					return fmt.Errorf("original city baseline for %q differs from semantic source/import context (%s)", saved.partyID, a.Type().Field(i).Name)
				}
			}
		}
		want.training = append([]uint8(nil), saved.training...)
		want.sales = cloneCitySales(saved.sales)
		want.salesVersion = saved.salesVersion
		want.returned = cloneCityReturn(saved.returned)
		if want.returned != nil {
			if err := want.checkReturn(want.returned); err != nil {
				return err
			}
		}
		if _, err := want.expectedMember(); err != nil {
			return err
		}
		if len(want.sales) != 0 {
			if _, err := citySaleRemainders(want, want.sales); err != nil {
				return err
			}
		}
		byID[saved.character.Identity] = want
	}
	// Adopt only freshly derived baselines after every assertion succeeded.
	state.bindings = expected.bindings
	for i := range state.bindings {
		state.bindings[i] = byID[state.bindings[i].character.Identity]
	}
	return nil
}

// legacyCityLoadSpeed is a baseline as imports before SAV-1116 stored it:
// the Human's actor-load speed without its overload penalty.
func legacyCityLoadSpeed(p mapload.PartyMember) mapload.PartyMember {
	if p.Carry == nil || p.Carry.LiveLoad == nil || !p.Carry.LiveLoad.Movement.Present || p.Carry.LiveLoad.Inventory.Source.Class != 2 {
		return p
	}
	p = mapload.CloneParty([]mapload.PartyMember{p})[0]
	l := p.Carry.LiveLoad
	l.Speed = mapload.SourceHumanState(l.Inventory.Source, l.Inventory.Accumulator).NativeMovementBase()
	l.Movement.NativeSpeed = l.Speed
	return p
}

// Reconstruct the code/template policy for historical baseline assertions only.
// Version 7 also emitted current-effect baselines, so both require full equality.
func legacyCityWeaponBaseline(p mapload.PartyMember, table *mapload.Table) mapload.PartyMember {
	p = mapload.CloneParty([]mapload.PartyMember{p})[0]
	p.Weapon = nil
	if code, ok := wornWeapon(p.Worn); ok && table != nil {
		if weapon, err := data.WeaponFromCode(code, table.Shapes, table.Materials, table.Weapons); err == nil {
			p.Weapon = &weapon
		}
		if p.Mage {
			if staff, err := data.ResolveWeapon(mageWeaponName, table.Shapes, table.Materials, table.Weapons); err == nil && staff.Code == code {
				p.Weapon = &staff
			}
		}
	}
	if p.OriginalHuman != nil {
		p.OriginalHuman.Weapon = p.Weapon
	}
	return p
}

func clearLegacyItemWeights(p *mapload.PartyMember) {
	clear := func(items []sim.ItemInstance) {
		for i := range items {
			items[i].Weight, items[i].WeightPresent = 0, false
		}
	}
	clear(p.CarriedItems)
	clear(p.WornItems[:])
	if p.OriginalHuman != nil {
		clear(p.OriginalHuman.Inventory)
		clear(p.OriginalHuman.Equipment[:])
	}
	if p.Carry != nil {
		clear(p.Carry.ItemInstances)
		clear(p.Carry.EquippedItems[:])
	}
}

func clearLegacySourceEquipment(p *mapload.PartyMember) {
	clear := func(items []sim.ItemInstance) {
		for i := range items {
			items[i].SourceEquipment = sim.SourceEquipment{}
		}
	}
	clear(p.CarriedItems)
	clear(p.WornItems[:])
	if p.OriginalHuman != nil {
		clear(p.OriginalHuman.Inventory)
		clear(p.OriginalHuman.Equipment[:])
	}
	if p.Carry != nil {
		clear(p.Carry.ItemInstances)
		clear(p.Carry.EquippedItems[:])
	}
}

// This is solely the pre-1096 import policy for validating a version-1
// assertion. It is not used to restore a new original SAV or repair live state.
func legacyOriginalCityBook(character sav.Character, table *mapload.Table) uint32 {
	if table == nil || table.Humans == nil || character.Class != "Human" {
		return 0
	}
	i := int(character.DefRow)
	if i <= 0 || i >= table.Humans.Len() {
		return 0
	}
	definition, err := data.NewHumanDef(table.Humans.EntryName(i), table.Humans.EntryParams(i))
	if err != nil {
		return 0
	}
	return definition.KnownSpells
}

type citySnapshotBudget struct{ elements, bytes uint64 }

// The persisted DTO has no recursive pointers; still bound every nested slice
// before cloning a baseline or letting the encoder allocate a payload. Gob's
// independent wire preflight remains mandatory before decoding untrusted bytes.
func boundCitySnapshot(v reflect.Value, budget *citySnapshotBudget, depth int) error {
	if depth > maxSaveGobDepth {
		return fmt.Errorf("original city snapshot nesting exceeds bound")
	}
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			return boundCitySnapshot(v.Elem(), budget, depth+1)
		}
	case reflect.String:
		budget.bytes += uint64(v.Len())
	case reflect.Slice, reflect.Array:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			budget.bytes += uint64(v.Len())
		} else {
			budget.elements += uint64(v.Len())
			if budget.elements > maxSaveGobElements {
				return fmt.Errorf("original city snapshot element budget exceeded")
			}
			for i := 0; i < v.Len(); i++ {
				if err := boundCitySnapshot(v.Index(i), budget, depth+1); err != nil {
					return err
				}
			}
		}
	case reflect.Map:
		budget.elements += uint64(v.Len())
		if budget.elements > maxSaveGobElements {
			return fmt.Errorf("original city snapshot element budget exceeded")
		}
		iter := v.MapRange()
		for iter.Next() {
			if err := boundCitySnapshot(iter.Key(), budget, depth+1); err != nil {
				return err
			}
			if err := boundCitySnapshot(iter.Value(), budget, depth+1); err != nil {
				return err
			}
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if err := boundCitySnapshot(v.Field(i), budget, depth+1); err != nil {
				return err
			}
		}
	}
	if budget.bytes > maxSaveBytes {
		return fmt.Errorf("original city snapshot byte budget exceeded")
	}
	return nil
}

// ExportOriginalSave retains the established API name for the current SAV
// producer; no original-city state selects a separate writer.
func (f *FrontEnd) ExportOriginalSave(s Snapshot, label string) ([]byte, error) {
	return f.ExportCurrentSave(s, label)
}
