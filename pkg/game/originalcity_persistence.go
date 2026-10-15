package game

import (
	"fmt"
	"reflect"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
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

// originalCityFromSnapshot prepares bounded structural provenance for current
// city capture and SAVE. Source bindings and return history are checked before
// the detached city state is used.
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
	// A mission snapshot keeps the immutable city source and previous return
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

// validateCityPartyIDs checks binding identity construction against the
// detached source roster.
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
		if dto.Version < 7 && c.Hero {
			party[i].ID, party[i].CompanionNPC = "hero", 0
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

type citySnapshotBudget struct{ elements, bytes uint64 }

// The persisted DTO has no recursive pointers; still bound every nested slice
// before cloning a baseline. SAV decoding separately bounds untrusted input
// before constructing this DTO.
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
