package game

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestWorldMapPagePolicyRejectsUnknownSteps(t *testing.T) {
	for _, tc := range []struct {
		field string
		value any
	}{
		{"open", "another"}, {"build", "another"}, {"point", "another"},
		{"font", "another"}, {"leave", "another"}, {"mode", "another"},
		{"mode", ""}, {"route-mode", "another"}, {"records", []string{"another"}},
		{"records", []string{}}, {"arrival", map[string]any{"town": "another", "mission": "location"}},
	} {
		t.Run(tc.field, func(t *testing.T) {
			var d map[string]any
			if err := json.Unmarshal(rom1WorldMapJSON, &d); err != nil {
				t.Fatal(err)
			}
			d["page"].(map[string]any)[tc.field] = tc.value
			raw, err := json.Marshal(d)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := decodeWorldMap(raw); err == nil || !strings.Contains(err.Error(), "page") {
				t.Fatalf("bad page %s accepted: %v", tc.field, err)
			}
		})
	}
}
