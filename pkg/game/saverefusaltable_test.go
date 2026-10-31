package game

import (
	"errors"
	"testing"

	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// A first-game row is today's text and no reason; a second-game row is
// today's word and text.
func TestSaveRefusalTablesGiveTodaysAnswers(t *testing.T) {
	untyped := map[refusalCondition]string{
		refusedNoGame:         "there is no game to save yet",
		refusedCompleted:      "the campaign is complete; there is no save point after its ending",
		refusedWorldMapReturn: "return to the city before saving",
		refusedListMap:        "a map opened from the list is not a game and cannot be saved",
	}
	typed := map[refusalCondition]string{
		refusedTalk:     "save.refused.talk",
		refusedNotice:   "save.refused.notice",
		refusedWorldMap: "save.refused.world_map",
		refusedLost:     "save.refused.lost",
		refusedHere:     "save.refused.here",
	}
	ru := words.For("russian")
	f := &FrontEnd{}
	f.Words.Engine = ru
	tables := []struct {
		name  string
		table *refusalTable
		rows  []refusalCondition
	}{
		{"first", firstRefusals, []refusalCondition{refusedNoGame, refusedCompleted, refusedWorldMapReturn, refusedListMap}},
		{"second", secondRefusals, []refusalCondition{refusedWorldMapReturn, refusedListMap, refusedTalk, refusedNotice, refusedWorldMap, refusedLost, refusedHere}},
	}
	for _, tc := range tables {
		held := map[refusalCondition]bool{}
		for _, c := range tc.rows {
			held[c] = true
			err := tc.table.refuse(c)
			if !errors.Is(err, c) {
				t.Errorf("%s %d: errors.Is does not match its condition", tc.name, c)
			}
			if text, ok := untyped[c]; ok {
				_, reason := refusalReason(err)
				if err.Error() != text || reason || errors.Is(err, errSavingUnavailable) {
					t.Errorf("%s %d: %q reason %v unavailable %v, want %q and no reason", tc.name, c, err, reason, errors.Is(err, errSavingUnavailable), text)
				}
				if got := f.playerSaveError(err); got != err {
					t.Errorf("%s %d: dialog shows %v, want the error itself", tc.name, c, got)
				}
				continue
			}
			word := typed[c]
			if word == "" || tc.name != "second" {
				t.Fatalf("%s %d: no expected row", tc.name, c)
			}
			r, reason := refusalReason(err)
			if !reason || r.reasonWord != word || err.Error() != "saving is not available for this game yet" || !errors.Is(err, errSavingUnavailable) {
				t.Errorf("%s %d: %q word %q unavailable %v, want %q", tc.name, c, err, r.reasonWord, errors.Is(err, errSavingUnavailable), word)
			}
			if got, want := f.playerSaveError(err).Error(), ru.Text(word); got != want {
				t.Errorf("%s %d: dialog shows %q, want %q", tc.name, c, got, want)
			}
		}
		for c := refusalCondition(0); c < refusalConditions; c++ {
			if !held[c] && tc.table[c] != (saveRefusal{}) {
				t.Errorf("%s holds a row for condition %d it does not refuse", tc.name, c)
			}
		}
	}
}

// The rules return their game's rows; the first game's dialog keeps its text.
func TestSaveRefusalRoutesReadTheirTable(t *testing.T) {
	f := &FrontEnd{}
	if err := (firstCampaignRules{}).townSavePoint(f); err != firstRefusals.refuse(refusedNoGame) {
		t.Fatalf("first-game town save point: %v", err)
	}
	if err := (secondCampaignRules{}).townSavePoint(f); err != secondRefusals.refuse(refusedHere) {
		t.Fatalf("second-game town save point: %v", err)
	}
	if err := (secondCampaignRules{}).exportable(Snapshot{}); err != secondRefusals.refuse(refusedHere) {
		t.Fatalf("second-game exportable with no bank: %v", err)
	}
	f.Words.Engine = words.For("")
	dir := t.TempDir()
	seams := f.SaveDialogSeams(SaveStore{Dir: dir}, OriginalStore{})
	if _, err := seams.Prepare(ui.SaveRequest{Admission: true}); err != nil {
		t.Fatalf("first-game admission: %v, want nil", err)
	}
	_, err := seams.Prepare(ui.SaveRequest{Directory: dir, Name: "refused", Format: ui.SaveSAV})
	if err == nil || err.Error() != "there is no game to save yet" {
		t.Fatalf("first-game named SAVE: %v, want today's text", err)
	}
}
