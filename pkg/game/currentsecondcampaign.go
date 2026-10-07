package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"againrom/pkg/base"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

type currentSecondLocation struct{ Kind, ID int }
type currentSecondCampaign struct {
	Bank      [1024]int32
	Current   currentSecondLocation
	Available []currentSecondLocation
	Room      *secondTownRoom `json:",omitempty"`
}

func (c *currentSecondCampaign) UnmarshalJSON(raw []byte) error {
	var fields struct {
		Bank      json.RawMessage
		Current   currentSecondLocation
		Available json.RawMessage
		Room      *secondTownRoom
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&fields); err != nil {
		return err
	}
	bank, err := currentScriptRows[int32](fields.Bank, 1024)
	if err != nil || len(bank) != 1024 {
		return fmt.Errorf("current second campaign bank must contain 1024 values")
	}
	available, err := currentScriptRows[currentSecondLocation](fields.Available, 2)
	if err != nil {
		return err
	}
	next := currentSecondCampaign{Current: fields.Current, Available: available, Room: fields.Room}
	copy(next.Bank[:], bank)
	if err := next.validate(); err != nil {
		return err
	}
	*c = next
	return nil
}

func (c *currentSecondCampaign) validate() error {
	if c.Current.Kind == 2 {
		return c.validateTown()
	}
	mission := c.Current
	if mission.Kind != 1 || !secondSaveMission(mission.ID) || c.Room != nil {
		return fmt.Errorf("current second campaign requires a supported ordinary mission")
	}
	if mission.ID == 21 {
		if err := c.validateLaterAvailable(); err != nil || len(c.Available) != 2 {
			return fmt.Errorf("current second campaign requires available mission21")
		}
	} else if len(c.Available) != 1 || c.Available[0] != mission {
		return fmt.Errorf("current second campaign requires the selected mission")
	}
	return nil
}

func secondSaveMission(id int) bool { return id == 10 || id == 20 || id == 21 }

func (c *currentSecondCampaign) validateTown() error {
	if c.Current == (currentSecondLocation{2, 2}) {
		if c.Room == nil || *c.Room != secondTownSquare {
			return fmt.Errorf("current second campaign requires a quiet second town")
		}
		return c.validateLaterAvailable()
	}
	town, mission := (currentSecondLocation{2, 1}), (currentSecondLocation{1, 10})
	if c.Current != town || c.Bank[768] != 10 || c.Room == nil || *c.Room > secondTownInn || len(c.Available) < 1 || len(c.Available) > 2 || c.Available[0] != town || len(c.Available) == 2 && c.Available[1] != mission {
		return fmt.Errorf("current second campaign requires a quiet first town")
	}
	return nil
}

func (c *currentSecondCampaign) validateLaterAvailable() error {
	if c.Bank[768] != 30 || c.Bank[916] != 1 || c.Bank[775] != 0 || len(c.Available) < 1 || len(c.Available) > 2 || c.Available[0] != (currentSecondLocation{2, 2}) {
		return fmt.Errorf("current second campaign requires ordinary second-town availability")
	}
	wantMission := c.Bank[772] != 0 && c.Bank[917] == 0
	if (len(c.Available) == 2) != wantMission || len(c.Available) == 2 && c.Available[1] != (currentSecondLocation{1, 21}) {
		return fmt.Errorf("current second campaign mission21 availability disagrees with bank")
	}
	return nil
}
func captureSecondCampaign(c *secondCampaign) *currentSecondCampaign {
	if c == nil {
		return nil
	}
	out := &currentSecondCampaign{Bank: c.bank, Current: currentSecondLocation{c.current.kind, c.current.id}}
	if c.current.kind == 2 {
		room := c.room
		out.Room = &room
	}
	for _, l := range c.available {
		out.Available = append(out.Available, currentSecondLocation{l.kind, l.id})
	}
	return out
}
func (c *currentSecondCampaign) clone() *currentSecondCampaign {
	if c == nil {
		return nil
	}
	out := *c
	out.Available = slices.Clone(c.Available)
	if c.Room != nil {
		room := *c.Room
		out.Room = &room
	}
	return &out
}
func (c *currentSecondCampaign) restore() *secondCampaign {
	if c == nil {
		return nil
	}
	out := &secondCampaign{bank: c.Bank, current: secondLocation{c.Current.Kind, c.Current.ID}}
	if c.Room != nil {
		out.room = *c.Room
	}
	for _, l := range c.Available {
		out.available = append(out.available, secondLocation{l.Kind, l.ID})
	}
	return out
}

func validateAuthoredGame(doc *sav.DocumentData, a *currentActionData, game base.Game) error {
	authored := base.GameROM1
	if a != nil && a.Session != nil && a.Session.Game != "" {
		authored = a.Session.Game
	}
	if authored != game {
		return fmt.Errorf("save game %s does not match installed game %s", authored, game)
	}
	if authored == base.GameROM2 {
		if doc.Head.Mission == 0 {
			if a.Session.Difficulty != nil {
				native, err := campaignDifficulty(int64(*a.Session.Difficulty))
				if err != nil || uint32(native) != doc.Head.Difficulty {
					return fmt.Errorf("ROM2 city difficulty disagrees with its header")
				}
			}
			if doc.World != nil || a.Session.Second == nil || a.Session.Second.validateTown() != nil || a.Program != nil || a.Policy != nil || a.Fog != nil || a.Pending != nil || len(a.PendingMessages) != 0 || len(a.Actions.Actors) != 0 || len(a.Held) != 0 || len(a.Bolts) != 0 || len(a.Heals) != 0 || len(a.Runs) != 0 || len(a.Options) != 0 || len(a.Animation) != 0 || len(a.DeathAges) != 0 || a.Manifest != nil || len(a.TerminalMotions) != 0 || len(a.VisualIdentities) != 0 || a.VisualNext != 0 || a.WorldMapReturn != nil || a.Session.MissionGold != nil || len(a.Party) == 0 || len(a.Roster) != 0 {
				return fmt.Errorf("ROM2 save lacks its current town continuation")
			}
			for _, p := range a.Party {
				if p.City == nil || p.Policy == nil || p.Base == nil {
					return fmt.Errorf("ROM2 town lacks complete current city party")
				}
			}
		} else if !secondSaveMission(int(doc.Head.Mission)) || doc.World == nil || a.Session.Second == nil || a.Session.Second.Current != (currentSecondLocation{1, int(doc.Head.Mission)}) || a.Session.Second.validate() != nil || a.Program == nil || a.Program.Dialect != sim.ScriptROM2 || a.Policy == nil || a.Policy.ROM2 == nil || a.Fog == nil {
			return fmt.Errorf("ROM2 save lacks its current mission continuation")
		}
	} else if a != nil && (a.Session != nil && a.Session.Second != nil || a.Program != nil && a.Program.Dialect != sim.ScriptROM1 || a.Policy != nil && a.Policy.ROM2 != nil) {
		return fmt.Errorf("ROM1 save contains ROM2 continuation")
	}
	return nil
}

func validateOriginalGame(saved []byte, game base.Game) error {
	doc, err := sav.DecodeDocumentData(saved)
	if err != nil {
		if game == base.GameROM2 {
			return fmt.Errorf("ROM2 LOAD requires an authored current SAV: %w", err)
		}
		sf, parseErr := sav.Open(saved)
		if parseErr != nil {
			return parseErr
		}
		if store, present := sf.StateStore(); present {
			for _, section := range store.Root.Children {
				if !strings.EqualFold(section.Name, "CurrentState") {
					continue
				}
				for _, leaf := range section.Children {
					if strings.EqualFold(leaf.Name, "AgainromActions") {
						return fmt.Errorf("authored current SAV document is malformed: %w", err)
					}
				}
			}
		}
		return nil
	}
	a, err := readCurrentActions(&doc)
	if err != nil {
		return err
	}
	return validateAuthoredGame(&doc, a, game)
}
