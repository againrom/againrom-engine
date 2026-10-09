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
				c.talkTo(secondInnOption{kind: 3, topic: 10, npc: 517})
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

func TestCurrentSecondOrdinaryMissionsKeepTheirCampaign(t *testing.T) {
	for mission := 1; mission <= 127; mission++ {
		source := &secondCampaign{current: secondLocation{1, mission}, available: []secondLocation{{1, mission}}}
		source.bank[768], source.bank[1000] = 50, -113
		if mission == 21 {
			source.bank[768], source.bank[916], source.bank[772] = 30, 1, 1
			source.available = []secondLocation{{2, 2}, {1, 21}}
		}
		captured := captureSecondCampaign(source)
		raw, err := json.Marshal(captured)
		if err != nil {
			t.Fatal(err)
		}
		var decoded currentSecondCampaign
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatalf("ordinary mission %d refused: %v", mission, err)
		}
		if !reflect.DeepEqual(source, decoded.restore()) {
			t.Fatalf("ordinary mission %d lost its campaign", mission)
		}
	}
	for _, mission := range []int{-1, 0, 128, 129} {
		if secondSaveMission(mission) {
			t.Fatalf("ordinary mission %d admitted outside the bank", mission)
		}
	}
}

func TestCurrentSecondLaterMissionKeepsEveryAvailableRecordInOrder(t *testing.T) {
	source := &secondCampaign{current: secondLocation{1, 31}}
	for id := 3; id > 0; id-- {
		source.available = append(source.available, secondLocation{2, id})
	}
	for id := 127; id > 0; id-- {
		source.available = append(source.available, secondLocation{1, id})
	}
	for i := range source.bank {
		source.bank[i] = int32(i) * -101
	}
	source.bank[775] = 0
	captured := captureSecondCampaign(source)
	cloned := captured.clone()
	raw, err := json.Marshal(struct {
		Bank      [1024]int32
		Current   currentSecondLocation
		Available []currentSecondLocation
	}{captured.Bank, captured.Current, captured.Available})
	if err != nil {
		t.Fatal(err)
	}
	var decoded currentSecondCampaign
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal("complete bounded availability refused", err)
	}
	if !reflect.DeepEqual(source, decoded.restore()) || !reflect.DeepEqual(captured, cloned) {
		t.Fatal("mission bank or available record order changed")
	}
	source.bank[1000]++
	source.available[0].id = 1
	decoded.Available[1].ID = 1
	if !reflect.DeepEqual(captured, cloned) {
		t.Fatal("captured campaign aliases live, decoded or cloned state")
	}
	cloned.Available[0].ID = 1
	if captured.Available[0].ID != 3 {
		t.Fatal("cloned campaign aliases the capture")
	}
}

func TestCurrentSecondLaterMissionRefusesMalformedAvailabilityAtomically(t *testing.T) {
	valid := &currentSecondCampaign{Current: currentSecondLocation{1, 31}, Available: []currentSecondLocation{{2, 2}, {1, 31}, {1, 32}}}
	for _, tc := range []struct {
		name   string
		change func(*currentSecondCampaign)
	}{
		{"current kind", func(c *currentSecondCampaign) { c.Current.Kind = 3 }},
		{"current zero", func(c *currentSecondCampaign) { c.Current.ID = 0 }},
		{"current outside bank", func(c *currentSecondCampaign) { c.Current.ID = 128 }},
		{"absent list", func(c *currentSecondCampaign) { c.Available = nil }},
		{"current absent", func(c *currentSecondCampaign) { c.Available = c.Available[:1] }},
		{"duplicate current", func(c *currentSecondCampaign) { c.Available = append(c.Available, c.Current) }},
		{"duplicate other", func(c *currentSecondCampaign) { c.Available = append(c.Available, c.Available[0]) }},
		{"available kind", func(c *currentSecondCampaign) { c.Available[0].Kind = 3 }},
		{"mission zero", func(c *currentSecondCampaign) { c.Available[2].ID = 0 }},
		{"mission outside bank", func(c *currentSecondCampaign) { c.Available[2].ID = 128 }},
		{"town zero", func(c *currentSecondCampaign) { c.Available[0].ID = 0 }},
		{"unknown town", func(c *currentSecondCampaign) { c.Available[0].ID = 4 }},
		{"restoration", func(c *currentSecondCampaign) { c.Bank[775] = 1 }},
		{"town room", func(c *currentSecondCampaign) { room := secondTownSquare; c.Room = &room }},
		{"list overflow", func(c *currentSecondCampaign) {
			c.Available = make([]currentSecondLocation, 131)
			for i := range c.Available {
				c.Available[i] = c.Current
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed := valid.clone()
			tc.change(changed)
			raw, err := json.Marshal(changed)
			if err != nil {
				t.Fatal(err)
			}
			decoded := valid.clone()
			if json.Unmarshal(raw, decoded) == nil {
				t.Fatal("malformed campaign admitted")
			}
			if !reflect.DeepEqual(valid, decoded) {
				t.Fatal("failed decode replaced the campaign")
			}
		})
	}
	for _, raw := range []string{`{"Bank":`, `{"Bank":[],"Current":{"Kind":1,"ID":31},"Available":[]}`} {
		decoded := valid.clone()
		if json.Unmarshal([]byte(raw), decoded) == nil || !reflect.DeepEqual(valid, decoded) {
			t.Fatal("malformed JSON replaced the campaign")
		}
	}
}
