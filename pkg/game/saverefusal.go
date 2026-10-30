package game

import (
	"errors"

	"againrom/pkg/ui"
)

// saveRefusal is a refused save point and the engine word that tells the
// player why (DIV-4297). Logs and witnesses read errSavingUnavailable.
type saveRefusal struct{ reasonWord string }

func (r saveRefusal) Error() string { return errSavingUnavailable.Error() }

func (r saveRefusal) Is(target error) bool { return target == errSavingUnavailable }

var (
	refusedTalk     = saveRefusal{reasonWord: "save.refused.talk"}
	refusedNotice   = saveRefusal{reasonWord: "save.refused.notice"}
	refusedWorldMap = saveRefusal{reasonWord: "save.refused.world_map"}
	refusedLost     = saveRefusal{reasonWord: "save.refused.lost"}
	refusedHere     = saveRefusal{reasonWord: "save.refused.here"}
)

// playerSaveError is err as the save dialog shows it: a refusal is its
// reason in the install's language, in UTF-8; any other error is itself.
func (f *FrontEnd) playerSaveError(err error) error {
	var r saveRefusal
	if errors.As(err, &r) {
		return errors.New(f.Words.Engine.Text(r.reasonWord))
	}
	return err
}

// admitSave is the admission ExportCurrentSave applies before it builds a
// document: whether the captured point admits a SAVE at all.
func (f *FrontEnd) admitSave(s Snapshot) error {
	s.game = s.game.Normal()
	if f.campaign().ownsGame(s.game) != nil {
		return errors.New("captured save game differs from installed game")
	}
	rules := campaignOf(s.game)
	if err := rules.exportable(s); err != nil {
		return err
	}
	if s.WorldMapReturn != nil {
		return errors.New("return to the city before saving")
	}
	return rules.refuseCompleted(f, s)
}

// saveAdmission answers the save dialog's admission request without
// building a file. Only a refusal is a reason; any other error answers nil.
func (f *FrontEnd) saveAdmission(request ui.SaveRequest) error {
	err := f.captureSave(captureTentative, request.OnMap, func(s Snapshot, _ string) error {
		return f.admitSave(s)
	})
	if !errors.As(err, new(saveRefusal)) {
		return nil
	}
	return f.playerSaveError(err)
}
