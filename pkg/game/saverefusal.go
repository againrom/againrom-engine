package game

import (
	"errors"
	"strconv"

	"againrom/pkg/ui"
)

// refusalCondition is a closed save point: the key of a refusal table row
// and the target errors.Is matches the row by.
type refusalCondition uint8

const (
	refusedNoGame refusalCondition = iota + 1
	refusedCompleted
	refusedWorldMapReturn
	refusedListMap
	refusedTalk
	refusedNotice
	refusedWorldMap
	refusedLost
	refusedHere
	refusalConditions
)

func (c refusalCondition) Error() string { return "save refusal " + strconv.Itoa(int(c)) }

// saveRefusal is a refusal table row: condition, the engine word naming the
// reason (DIV-4297) and the error text. A row without a word shows its text
// and names no reason.
type saveRefusal struct {
	condition  refusalCondition
	reasonWord string
	text       string
}

func (r saveRefusal) Error() string { return r.text }

func (r saveRefusal) Is(target error) bool {
	return target == r.condition || r.reasonWord != "" && target == errSavingUnavailable
}

// refusalTable is one game's save point refusals, keyed by condition.
type refusalTable [refusalConditions]saveRefusal

func refusals(rows ...saveRefusal) *refusalTable {
	var t refusalTable
	for _, r := range rows {
		t[r.condition] = r
	}
	return &t
}

func (t *refusalTable) refuse(c refusalCondition) error { return t[c] }

var (
	firstRefusals = refusals(
		saveRefusal{condition: refusedNoGame, text: "there is no game to save yet"},
		saveRefusal{condition: refusedCompleted, text: "the campaign is complete; there is no save point after its ending"},
		saveRefusal{condition: refusedWorldMapReturn, text: "return to the city before saving"},
		saveRefusal{condition: refusedListMap, text: "a map opened from the list is not a game and cannot be saved"},
	)
	secondRefusals = refusals(
		firstRefusals[refusedWorldMapReturn],
		firstRefusals[refusedListMap],
		saveRefusal{condition: refusedTalk, reasonWord: "save.refused.talk", text: errSavingUnavailable.Error()},
		saveRefusal{condition: refusedNotice, reasonWord: "save.refused.notice", text: errSavingUnavailable.Error()},
		saveRefusal{condition: refusedWorldMap, reasonWord: "save.refused.world_map", text: errSavingUnavailable.Error()},
		saveRefusal{condition: refusedLost, reasonWord: "save.refused.lost", text: errSavingUnavailable.Error()},
		saveRefusal{condition: refusedHere, reasonWord: "save.refused.here", text: errSavingUnavailable.Error()},
	)
)

func refusalReason(err error) (saveRefusal, bool) {
	var r saveRefusal
	return r, errors.As(err, &r) && r.reasonWord != ""
}

// playerSaveError is err as the save dialog shows it: a refusal's reason in
// the install's language, in UTF-8; any other error is itself.
func (f *FrontEnd) playerSaveError(err error) error {
	if r, ok := refusalReason(err); ok {
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
		return rules.refusals().refuse(refusedWorldMapReturn)
	}
	return rules.refuseCompleted(f, s)
}

// saveAdmission answers the save dialog's admission request without
// building a file. Only a refusal is a reason; any other error answers nil.
func (f *FrontEnd) saveAdmission(request ui.SaveRequest) error {
	err := f.captureSave(captureTentative, request.OnMap, func(s Snapshot, _ string) error {
		return f.admitSave(s)
	})
	if _, ok := refusalReason(err); !ok {
		return nil
	}
	return f.playerSaveError(err)
}
