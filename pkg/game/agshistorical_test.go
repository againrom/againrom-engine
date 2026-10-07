package game

// Historical field recovery for the in-package AGS fixture codec, so fixtures
// written under the historical spelling decode.

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"againrom/pkg/ui"
)

// historicalApplicationField and historicalOptionsField are the two field
// names a payload written under the historical spelling contains verbatim.
// gob writes a field name as an uncompressed length-prefixed ASCII string in
// the struct's type descriptor, so a stream that declares the field contains
// these bytes and a stream that does not cannot. Using them as a gate can only
// over-trigger, and an over-trigger costs one decode that adopts nothing.
const (
	historicalApplicationField = "Application1170"
	historicalOptionsField     = "GameOptions1186"
)

// historicalSnapshotWire declares only the Snapshot fields whose wire name
// changed. gob ignores every other field in the stream, so this is a partial
// view of the same payload and not a second Snapshot.
type historicalSnapshotWire struct {
	Application1170 *historicalApplicationWire
	Residue         historicalResidueWire
}

// historicalResidueWire is SnapshotResidue's one renamed field and nothing
// else.
type historicalResidueWire struct {
	GameOptions1186 []PendingGameOption
}

// historicalApplicationWire mirrors SnapshotApplicationState under the one
// field name that changed. The other five names are unchanged on the wire and
// are repeated here because gob needs somewhere to put them.
type historicalApplicationWire struct {
	LocalOnly1186 bool
	Version       uint32
	View          ui.SaveApplicationState
	Baseline      ui.SaveApplicationState
	Original      OriginalStateData
	WimpyBaseline int
}

// adoptHistoricalNames recovers the two Snapshot values whose gob field names
// changed after saves carrying them had already been written. body is the same
// payload the primary decode read. It decodes a second time only when the
// current name produced nothing AND the historical name is present in the
// stream, so an ordinary current save pays two bytes.Contains and no decode.
func adoptHistoricalNames(s *Snapshot, body []byte) error {
	wantApplication := s.ApplicationState == nil && bytes.Contains(body, []byte(historicalApplicationField))
	wantOptions := len(s.Residue.PendingGameOptions) == 0 && bytes.Contains(body, []byte(historicalOptionsField))
	if !wantApplication && !wantOptions {
		return nil
	}
	var historical historicalSnapshotWire
	if err := gob.NewDecoder(bytes.NewReader(body)).Decode(&historical); err != nil {
		return fmt.Errorf("save is corrupt: historical field names: %w", err)
	}
	if wantApplication && historical.Application1170 != nil {
		s.ApplicationState = &SnapshotApplicationState{
			LocalOnly:     historical.Application1170.LocalOnly1186,
			Version:       historical.Application1170.Version,
			View:          historical.Application1170.View,
			Baseline:      historical.Application1170.Baseline,
			Original:      historical.Application1170.Original,
			WimpyBaseline: historical.Application1170.WimpyBaseline,
		}
	}
	if wantOptions && len(historical.Residue.GameOptions1186) != 0 {
		s.Residue.PendingGameOptions = historical.Residue.GameOptions1186
	}
	return nil
}
