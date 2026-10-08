package game

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"

	"againrom/pkg/base"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

type currentSessionData struct {
	Difficulty         *mapload.Difficulty    `json:",omitempty"`
	Game               base.Game              `json:",omitempty"`
	Second             *currentSecondCampaign `json:",omitempty"`
	QuickSpells        []currentQuickSpell    `json:",omitempty"`
	Offered            int
	Won                []int               `json:",omitempty"`
	ConsumedHeroGrants []int               `json:",omitempty"`
	MissionGold        *int                `json:",omitempty"`
	DocumentMission    *int                `json:",omitempty"`
	View               *currentViewResidue `json:",omitempty"`
	Fame               *currentFamePolicy  `json:",omitempty"`
	Taken              []SnapshotOffer     `json:",omitempty"`
	OfferLabels        *currentOfferLabels `json:",omitempty"`
}

type currentQuickSpell struct {
	Slot uint8
	ID   uint32
}

type currentFamePolicy struct {
	Known  bool
	Result *currentFameResult `json:",omitempty"`
}

type currentFameResult struct {
	Name     []byte
	Score    int32
	Recorded bool
}

type currentViewResidue struct {
	Animation                  *ui.AnimationState `json:",omitempty"`
	PressedSpell               uint32             `json:",omitempty"`
	Zoom, FractionX, FractionY float64
	YOriginDelta               *int64 `json:",omitempty"`
	Speed                      int32
	PeriodUS                   int
	Unpaced                    bool
	PlayerPaused               bool
	Panels                     *currentPanelVisibility `json:",omitempty"`
}

type currentPanelVisibility struct {
	DollOpen, WornOpen bool
}

func projectCurrentSession(doc *sav.DocumentData, s Snapshot) error {
	a, err := readCurrentActions(doc)
	if err != nil {
		return err
	}
	if a == nil {
		a = &currentActionData{Version: 1}
	}
	a.Session = &currentSessionData{Game: s.game, Second: s.second.clone(), Offered: s.Offered, Won: slices.Clone(s.Won), ConsumedHeroGrants: slices.Clone(s.ConsumedHeroGrants)}
	if s.game == base.GameROM2 && s.Mission == 0 {
		difficulty := s.Difficulty
		a.Session.Difficulty = &difficulty
	}
	a.Session.Taken, a.Session.OfferLabels = slices.Clone(s.Taken), cloneOfferLabels(s.OfferLabels)
	shortcuts, err := quickSpellsToOriginalIndices(s.QuickSpells)
	if err != nil {
		return err
	}
	for slot, id := range s.QuickSpells {
		if id != 0 && shortcuts[slot] == -1 {
			a.Session.QuickSpells = append(a.Session.QuickSpells, currentQuickSpell{uint8(slot), id})
		}
	}
	fame := fameFromSnapshot(s)
	a.Session.Fame = &currentFamePolicy{Known: fame.Known}
	if fame.Result != nil {
		a.Session.Fame.Result = &currentFameResult{[]byte(fame.Result.Name), fame.Result.Score, fame.Result.Recorded}
	}
	documentMission := s.DocumentMission
	a.Session.DocumentMission = &documentMission
	if s.Mission != 0 {
		gold := s.Gold
		a.Session.MissionGold = &gold
	}
	if s.ApplicationState != nil && !s.ApplicationState.LocalOnly {
		v := s.ApplicationState.View
		raw, err := applicationCurrentRaw(s.ApplicationState)
		if err != nil {
			return err
		}
		a.Session.View = &currentViewResidue{Zoom: v.Zoom, FractionX: v.ViewX - math.Round(v.ViewX), FractionY: v.ViewY - math.Round(v.ViewY), Speed: raw.Speed, PeriodUS: v.PeriodUS, Unpaced: v.Unpaced, PlayerPaused: v.PlayerPaused}
		a.Session.View.Panels = &currentPanelVisibility{DollOpen: v.DollOpen, WornOpen: v.WornOpen}
		if s.mapAnimation != nil {
			animation := *s.mapAnimation
			a.Session.View.Animation = &animation
		}
		if delta := int64(math.Round(v.ViewY)) - int64(raw.ViewY); delta != 0 {
			a.Session.View.YOriginDelta = &delta
		}
		if raw.Pressed == -1 {
			a.Session.View.PressedSpell = v.PressedSpell
		}
	}
	b, err := json.Marshal(a)
	if err != nil {
		return err
	}
	return sav.SetNativeActions(&doc.State, b)
}

func validateCurrentSession(s *currentSessionData) error {
	if s == nil {
		return nil
	}
	if s.Game != "" && s.Game != base.GameROM1 && s.Game != base.GameROM2 {
		return fmt.Errorf("invalid current save game")
	}
	if (s.Second != nil) != (s.Game == base.GameROM2) {
		return fmt.Errorf("current save game and campaign disagree")
	}
	if s.Second != nil {
		if err := s.Second.validate(); err != nil {
			return err
		}
	}
	if s.Difficulty != nil && (s.Game != base.GameROM2 || s.Second == nil || s.Second.Current.Kind != 2 || *s.Difficulty < 0 || *s.Difficulty > mapload.DifficultyHard) {
		return fmt.Errorf("invalid current city difficulty")
	}
	var ids [4]uint32
	for _, row := range s.QuickSpells {
		if int(row.Slot) >= len(ids) || row.ID == 0 || ids[row.Slot] != 0 {
			return fmt.Errorf("invalid current quick-spell slot")
		}
		indices, err := quickSpellsToOriginalIndices([4]uint32{row.ID})
		if err != nil || indices[0] != -1 {
			return fmt.Errorf("invalid current custom quick-spell binding")
		}
		ids[row.Slot] = row.ID
	}
	if err := validateQuickSpells(ids); err != nil {
		return err
	}
	if s.Fame != nil && s.Fame.Result != nil && (!s.Fame.Known || len(s.Fame.Result.Name) > 65536) {
		return fmt.Errorf("invalid current campaign result")
	}
	if len(s.Taken) > 65536 {
		return fmt.Errorf("invalid consumed offer population")
	}
	seen := map[SnapshotOffer]bool{}
	for _, r := range s.Taken {
		if r.Chapter <= 0 || r.Building < int(TownTavern) || r.Building > int(TownSchool) || r.Index < 0 || r.Index > 65535 || seen[r] {
			return fmt.Errorf("invalid consumed offer history")
		}
		seen[r] = true
	}
	if labels := s.OfferLabels; labels != nil {
		if labels.Chapter < 0 || uint64(labels.Chapter) > math.MaxUint32 {
			return fmt.Errorf("invalid offer chapter anchor")
		}
		for b, rows := range labels.Buildings {
			if len(rows) > 65536 {
				return fmt.Errorf("invalid offer index population")
			}
			indices := map[int]bool{}
			for _, r := range rows {
				if r.Index < 0 || r.Index > 65535 || indices[r.Index] || r.Mission < 0 || r.Mission > 65535 || r.NPC < 0 || r.NPC > 65535 || b != int(TownTavern) && r.NPC != 0 {
					return fmt.Errorf("invalid offer index anchor")
				}
				indices[r.Index] = true
			}
		}
	}
	if len(s.Won) > 65536 || s.MissionGold != nil && (*s.MissionGold < 0 || uint64(*s.MissionGold) > math.MaxUint32) {
		return fmt.Errorf("invalid current campaign residue")
	}
	if s.DocumentMission != nil && (*s.DocumentMission < 0 || uint64(*s.DocumentMission) > math.MaxUint32) {
		return fmt.Errorf("invalid current document collection guard")
	}
	for i, n := range s.Won {
		if n <= 0 || i > 0 && s.Won[i-1] >= n {
			return fmt.Errorf("invalid current completed-mission set")
		}
	}
	if len(s.ConsumedHeroGrants) > 65536 {
		return fmt.Errorf("invalid consumed companion grant population")
	}
	for i, n := range s.ConsumedHeroGrants {
		if n <= 0 || i > 0 && s.ConsumedHeroGrants[i-1] >= n {
			return fmt.Errorf("invalid consumed companion grant set")
		}
	}
	if v := s.View; v != nil {
		if v.Animation != nil {
			if err := ui.ValidateAnimationState(*v.Animation); err != nil {
				return err
			}
		}
		if v.PressedSpell != 0 {
			indices, err := quickSpellsToOriginalIndices([4]uint32{v.PressedSpell})
			if err != nil || indices[0] != -1 {
				return fmt.Errorf("invalid current custom pressed spell")
			}
		}
		if v.Zoom <= 0 || math.IsNaN(v.Zoom) || math.IsInf(v.Zoom, 0) || math.Abs(v.FractionX) > 0.5 || math.Abs(v.FractionY) > 0.5 || v.PeriodUS <= 0 ||
			v.YOriginDelta != nil && (*v.YOriginDelta < -math.MaxUint32 || *v.YOriginDelta > math.MaxUint32) {
			return fmt.Errorf("invalid current view residue")
		}
	}
	return nil
}

func fameFromCurrentTown(t *Town, s *currentSessionData) SnapshotFame {
	fame := fameFromTown(t)
	if s != nil && s.Fame != nil {
		fame.Known = s.Fame.Known
		if r := s.Fame.Result; r != nil {
			fame.Result = &FameResult{string(r.Name), r.Score, r.Recorded}
		}
	}
	return fame
}

func applyCurrentView(v *ui.SaveApplicationState, raw OriginalStateData, a *currentActionData) {
	if a == nil || a.Session == nil || a.Session.View == nil {
		return
	}
	r := a.Session.View
	if r.Panels != nil {
		v.DollOpen, v.WornOpen = r.Panels.DollOpen, r.Panels.WornOpen
	}
	if r.PressedSpell != 0 && raw.Pressed == -1 {
		v.PressedSpell = r.PressedSpell
	}
	v.Zoom, v.ViewX, v.ViewY = r.Zoom, v.ViewX+r.FractionX, v.ViewY+r.FractionY
	if r.YOriginDelta != nil {
		v.ViewY += float64(*r.YOriginDelta)
	}
	if raw.Speed == r.Speed {
		v.PeriodUS, v.Unpaced, v.PlayerPaused = r.PeriodUS, r.Unpaced, r.PlayerPaused
	}
}

func currentQuickSpells(ordinary [4]uint32, session *currentSessionData) ([4]uint32, error) {
	if session != nil {
		for _, row := range session.QuickSpells {
			if ordinary[row.Slot] == 0 {
				ordinary[row.Slot] = row.ID
			}
		}
	}
	return ordinary, validateQuickSpells(ordinary)
}
