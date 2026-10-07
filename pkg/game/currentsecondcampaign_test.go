package game

import (
	"encoding/json"
	"reflect"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestCurrentSecondCampaignJSONBoundsAndDetachment(t *testing.T) {
	source := &secondCampaign{current: secondLocation{1, 10}, available: []secondLocation{{1, 10}}}
	for i := range source.bank {
		source.bank[i] = int32(i) * -101
	}
	captured := captureSecondCampaign(source)
	raw, err := json.Marshal(captured)
	if err != nil {
		t.Fatal(err)
	}
	var decoded currentSecondCampaign
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(source, decoded.restore()) {
		t.Fatal("campaign bank/destination changed")
	}
	source.bank[1000]++
	source.available[0].id = 20
	if captured.Bank[1000] == source.bank[1000] || captured.Available[0].ID == 20 {
		t.Fatal("campaign capture aliases live state")
	}
	for _, count := range []int{0, 1023, 1025} {
		raw, _ := json.Marshal(struct {
			Bank      []int32
			Current   currentSecondLocation
			Available []currentSecondLocation
		}{make([]int32, count), currentSecondLocation{1, 10}, []currentSecondLocation{{1, 10}}})
		if json.Unmarshal(raw, &decoded) == nil {
			t.Fatalf("campaign bank length%d admitted", count)
		}
	}
	if json.Unmarshal([]byte(`{"Bank":[0],"Current":{"Kind":1,"ID":10},"Available":[]}`), &decoded) == nil {
		t.Fatal("truncated bank admitted")
	}
}

func TestCurrentAuthoredGameHistoricalPolicy(t *testing.T) {
	doc := &sav.DocumentData{}
	if err := validateAuthoredGame(doc, nil, base.GameROM1); err != nil {
		t.Fatal("unmarked historical ROM1 refused", err)
	}
	if err := validateAuthoredGame(doc, nil, base.GameROM2); err == nil {
		t.Fatal("unmarked input guessed as ROM2")
	}
	a := &currentActionData{Session: &currentSessionData{Game: base.GameROM1}, Program: &currentScriptProgram{}}
	if err := validateAuthoredGame(doc, a, base.GameROM1); err != nil {
		t.Fatal("explicit empty historical ROM1 refused", err)
	}
	a.Program.Dialect = sim.ScriptROM2
	if err := validateAuthoredGame(doc, a, base.GameROM1); err == nil {
		t.Fatal("ROM1 marker admitted ROM2 dialect")
	}
	a.Program = nil
	a.Session.Game = ""
	if err := validateAuthoredGame(doc, a, base.GameROM1); err != nil {
		t.Fatal("historical absent fields refused", err)
	}
}

func TestCurrentMissionFogBounds(t *testing.T) {
	for _, fog := range []currentMissionFog{{Cols: 257, Rows: 1, Explored: []byte{1}, Visible: []byte{1}}, {Cols: 1, Rows: 1, Explored: []byte{0}, Visible: []byte{1}}, {Cols: 1, Rows: 2, Explored: []byte{1}, Visible: []byte{1}}, {Cols: 1, Rows: 1, Explored: []byte{1}}} {
		if fog.validate() == nil {
			t.Fatal("invalid current visibility admitted", fog.Cols, fog.Rows)
		}
	}
}

func TestCurrentSecondTownRoomAndAvailabilityBounds(t *testing.T) {
	for _, room := range []secondTownRoom{secondTownSquare, secondTownInn} {
		for _, unlocked := range []bool{false, true} {
			c := newSecondCampaign()
			c.room = room
			if unlocked {
				c.talk()
			}
			a := captureSecondCampaign(c)
			raw, err := json.Marshal(a)
			if err != nil {
				t.Fatal(err)
			}
			var cold currentSecondCampaign
			if err := json.Unmarshal(raw, &cold); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(c, cold.restore()) {
				t.Fatal("typed quiet room changed")
			}
			clone := a.clone()
			*cold.Room = secondTownRoom(8)
			if cold.validate() == nil {
				t.Fatal("foreign logical room admitted")
			}
			c.room = secondTownRoom(8)
			c.available[0] = secondLocation{2, 2}
			if *a.Room != room || !reflect.DeepEqual(a, clone) {
				t.Fatal("captured room/list aliases live or cloned state")
			}
		}
	}
}
