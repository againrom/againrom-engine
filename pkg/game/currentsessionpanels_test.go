package game

import (
	"encoding/json"
	"testing"

	"againrom/pkg/ui"
)

func TestCurrentPanelVisibilityPresence(t *testing.T) {
	for _, test := range []struct {
		name, panels string
		doll, worn   bool
	}{
		{"old-absent", "", true, false},
		{"explicit-hidden", `,"Panels":{"DollOpen":false,"WornOpen":false}`, false, false},
		{"explicit-visible", `,"Panels":{"DollOpen":true,"WornOpen":true}`, true, true},
		{"independent", `,"Panels":{"DollOpen":false,"WornOpen":true}`, false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var data currentActionData
			raw := `{"Version":1,"Session":{"View":{"Zoom":1,"PeriodUS":1000` + test.panels + `}}}`
			if err := json.Unmarshal([]byte(raw), &data); err != nil {
				t.Fatal(err)
			}
			view := ui.SaveApplicationState{DollOpen: true, WornOpen: false}
			applyCurrentView(&view, OriginalStateData{}, &data)
			if view.DollOpen != test.doll || view.WornOpen != test.worn {
				t.Fatalf("panel presence: %+v", view)
			}
		})
	}
}
