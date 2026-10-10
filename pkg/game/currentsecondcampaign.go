package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"againrom/pkg/base"
	"againrom/pkg/formats/sav"
)

type currentSecondLocation struct{ Kind, ID int }
type currentSecondCampaign struct {
	Bank      [1024]int32
	Current   currentSecondLocation
	Available []currentSecondLocation
	Room      *secondTownRoom `json:",omitempty"`
	Aux       *secondAux      `json:",omitempty"`
}

const maxSecondAvailable = 130

func (c *currentSecondCampaign) UnmarshalJSON(raw []byte) error {
	var fields struct {
		Bank      json.RawMessage
		Current   currentSecondLocation
		Available json.RawMessage
		Room      *secondTownRoom
		Aux       json.RawMessage
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
	available, err := currentScriptRows[currentSecondLocation](fields.Available, maxSecondAvailable)
	if err != nil {
		return err
	}
	next := currentSecondCampaign{Current: fields.Current, Available: available, Room: fields.Room}
	copy(next.Bank[:], bank)
	if next.Aux, err = decodeSecondAux(fields.Aux); err != nil {
		return err
	}
	if err := next.validate(); err != nil {
		return err
	}
	*c = next
	return nil
}

// decodeSecondAux reads exactly eight entries of exactly five fields. An
// absent or all-zero array decodes as absent, the form capture writes.
func decodeSecondAux(raw json.RawMessage) (*secondAux, error) {
	if raw == nil {
		return nil, nil
	}
	rows, err := currentScriptRows[json.RawMessage](raw, 8)
	if err != nil || rows == nil || len(rows) != 8 {
		return nil, fmt.Errorf("current second campaign auxiliary array must hold 8 entries")
	}
	var aux secondAux
	for i, row := range rows {
		fields, err := currentScriptRows[int32](row, 5)
		if err != nil || len(fields) != 5 {
			return nil, fmt.Errorf("current second campaign auxiliary entry must hold 5 fields")
		}
		copy(aux[i][:], fields)
	}
	if aux == (secondAux{}) {
		return nil, nil
	}
	return &aux, nil
}

func (c *currentSecondCampaign) validate() error {
	if c.Current.Kind == 2 {
		return c.validateTown()
	}
	mission := c.Current
	if mission.Kind != 1 || !secondSaveMission(mission.ID) || c.Room != nil {
		return fmt.Errorf("current second campaign requires a supported ordinary mission")
	}
	if mission.ID == 10 || mission.ID == 20 {
		if len(c.Available) != 1 || c.Available[0] != mission {
			return fmt.Errorf("current second campaign requires the selected mission")
		}
	} else {
		return c.validateOrdinaryAvailable()
	}
	return nil
}

func secondSaveMission(id int) bool { return id > 0 && id < 128 }

func (c *currentSecondCampaign) validateOrdinaryAvailable() error {
	if c.Bank[775] != 0 || len(c.Available) == 0 || len(c.Available) > maxSecondAvailable {
		return fmt.Errorf("current second campaign requires bounded ordinary availability")
	}
	seen := make(map[currentSecondLocation]bool, len(c.Available))
	for _, location := range c.Available {
		valid := location.Kind == 1 && secondSaveMission(location.ID) || location.Kind == 2 && location.ID >= 1 && location.ID <= 3
		if !valid || seen[location] {
			return fmt.Errorf("current second campaign contains an invalid or duplicate destination")
		}
		seen[location] = true
	}
	if !seen[c.Current] {
		return fmt.Errorf("current second campaign requires the selected mission")
	}
	return nil
}

func (c *currentSecondCampaign) validateTown() error {
	if c.Current == (currentSecondLocation{2, 2}) || c.Current == (currentSecondLocation{2, 3}) {
		if c.Room == nil || *c.Room > secondTownInn {
			return fmt.Errorf("current second campaign requires a quiet later town")
		}
		return c.validateOrdinaryAvailable()
	}
	town, mission := (currentSecondLocation{2, 1}), (currentSecondLocation{1, 10})
	if c.Current != town || c.Bank[768] != 10 || c.Room == nil || *c.Room > secondTownInn || len(c.Available) < 1 || len(c.Available) > 2 || c.Available[0] != town || len(c.Available) == 2 && c.Available[1] != mission {
		return fmt.Errorf("current second campaign requires a quiet first town")
	}
	return nil
}

func captureSecondCampaign(c *secondCampaign) *currentSecondCampaign {
	if c == nil {
		return nil
	}
	out := &currentSecondCampaign{Bank: c.bank, Current: currentSecondLocation{c.current.kind, c.current.id}}
	if c.aux != (secondAux{}) {
		aux := c.aux
		out.Aux = &aux
	}
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
	if c.Aux != nil {
		aux := *c.Aux
		out.Aux = &aux
	}
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
	if c.Aux != nil {
		out.aux = *c.Aux
	}
	if c.Room != nil {
		out.room = *c.Room
	}
	for _, l := range c.Available {
		out.available = append(out.available, secondLocation{l.Kind, l.ID})
	}
	return out
}

func validateAuthoredGame(doc *sav.DocumentData, a *currentActionData, game base.Game) error {
	var authored base.Game
	if a != nil && a.Session != nil {
		authored = a.Session.Game
	}
	if !base.SameGame(authored, game) {
		return fmt.Errorf("save game %s does not match installed game %s", authored.Normal(), game.Normal())
	}
	return campaignOf(authored).validateAuthored(doc, a)
}

func validateOriginalGame(saved []byte, game base.Game) error {
	doc, err := sav.DecodeDocumentData(saved)
	if err != nil {
		if refusal := campaignOf(game).undecodedSave(err); refusal != nil {
			return refusal
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
