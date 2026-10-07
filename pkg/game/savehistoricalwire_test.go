package game

import (
	"bytes"
	"reflect"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

func historicalTestView() ui.SaveApplicationState {
	return ui.SaveApplicationState{
		Selection:     []uint32{82},
		ViewX:         41.9375,
		ViewY:         103.8125,
		Zoom:          1,
		InventoryOpen: true,
		SpellBookOpen: true,
		DollOpen:      true,
		WornOpen:      true,
		MinimapOpen:   true,
		ShowHealth:    true,
		FlyingHP:      true,
		TimeFlow:      true,
		PeriodUS:      terrain.CadencePeriod(terrain.DefaultCadenceRung),
	}
}

// TestEncodeSaveRoundTripsCurrentFieldNames proves the historical path does
// not interfere with what this build writes: a current encode carries the
// current names and decodes back to the same values.
func TestEncodeSaveRoundTripsCurrentFieldNames(t *testing.T) {
	view := historicalTestView()
	s := Snapshot{
		ApplicationState: &SnapshotApplicationState{
			LocalOnly:     true,
			Version:       applicationStateVersion,
			View:          view,
			Baseline:      view,
			WimpyBaseline: 1,
		},
		Residue: SnapshotResidue{
			PendingGameOptions: []PendingGameOption{{Option: ui.GameOptionRetreat, Value: 2}},
		},
	}
	b, err := EncodeSave(s, "current")
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if bytes.Contains(b, []byte("Application1170")) || bytes.Contains(b, []byte("GameOptions1186")) {
		t.Error("a fresh encode still writes a historical field name")
	}
	got, _, err := DecodeSave(b)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ApplicationState == nil || !got.ApplicationState.LocalOnly || got.ApplicationState.WimpyBaseline != 1 {
		t.Fatalf("application record is %+v, want LocalOnly with WimpyBaseline 1", got.ApplicationState)
	}
	if !reflect.DeepEqual(got.ApplicationState.View, view) {
		t.Errorf("View is %+v, want %+v", got.ApplicationState.View, view)
	}
	if len(got.Residue.PendingGameOptions) != 1 || got.Residue.PendingGameOptions[0].Value != 2 {
		t.Errorf("PendingGameOptions is %+v, want one retreat option with value 2", got.Residue.PendingGameOptions)
	}
}
