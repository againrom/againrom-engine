package game

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/namefold"
	"againrom/pkg/scriptprofile"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// The map's script form in the editor: an original map holds its triggers
// in 184-byte type-7 records, an extended one in the Againrom script record
// (DIV-4599). An edit on an original map that only the extended record
// holds answers ui.ScriptFormNeeded; the editor asks, and ExtendScript
// moves the triggers as one edit. A save never writes an extended map
// under an .alm name (DIV-4603).

// liftedLimit is a refusal in the host's words of a limit the extended
// record lifts; Unwrap gives the limit, so an edit on an original map can
// ask to extend the script.
type liftedLimit struct {
	text  string
	limit *mapedit.ScriptLimitError
}

func (l *liftedLimit) Error() string { return l.text }
func (l *liftedLimit) Unwrap() error { return l.limit }

// lifted is err in the words text, keeping the limit it carries.
func lifted(err error, text string) error {
	var limit *mapedit.ScriptLimitError
	if errors.As(err, &limit) {
		return &liftedLimit{text: text, limit: limit}
	}
	return errors.New(text)
}

// formNeeded is the question an edit refused on original map m asks: the
// refusal carries a limit the extended record lifts. Nil for any other.
func formNeeded(m *mapedit.Editor, err error) error {
	var limit *mapedit.ScriptLimitError
	if err == nil || m.Extended() || !errors.As(err, &limit) || !limit.Lifted() {
		return nil
	}
	return &ui.ScriptFormNeeded{Why: err.Error()}
}

// ExtendScript moves the map's triggers into the Againrom script record as
// one edit.
func (s *mapInspectionEdits) ExtendScript() (*ui.InspectionDocument, error) {
	return s.change(func(m *mapedit.Editor) (bool, error) {
		return true, m.ExtendScript()
	})
}

// FoldScript moves an extended map's triggers back into type-7 records as
// one edit, when the original holds the script again.
func (s *mapInspectionEdits) FoldScript() (*ui.InspectionDocument, error) {
	return s.change(func(m *mapedit.Editor) (bool, error) {
		return true, foldScript(m)
	})
}

// foldScript folds m's script and checks every variable a node addresses
// as the original's registers hold it: below 100, and no check result in
// its register. The variable count falls to 100 first when no node
// addresses a variable past it.
func foldScript(m *mapedit.Editor) error {
	_, sc, err := wizardScript(m)
	if err != nil {
		return err
	}
	need := 0
	for v := range variableRegisters(sc) {
		need = max(need, int(min(v, math.MaxInt32-1))+1)
	}
	if v := m.ScriptVariables(); v > scriptprofile.FirstRegisters && need <= scriptprofile.FirstRegisters {
		if err := m.SetScriptVariables(scriptprofile.FirstRegisters); err != nil {
			return err
		}
	}
	if err := m.FoldScript(); err != nil {
		return err
	}
	_, sc, err = wizardScript(m)
	if err != nil {
		return err
	}
	for v := range variableRegisters(sc) {
		if err := scriptRegisterAllowed(sc, v); err != nil {
			return err
		}
	}
	return nil
}

// raiseVariables raises an extended map's variable count past the highest
// variable a node addresses, so every variable the script names is one the
// record holds; the raise joins the edit it follows.
func raiseVariables(m *mapedit.Editor) error {
	if !m.Extended() {
		return nil
	}
	_, sc, err := wizardScript(m)
	if err != nil {
		return err
	}
	need := m.ScriptVariables()
	for v := range variableRegisters(sc) {
		if v < scriptprofile.ExtendedVariables {
			need = max(need, int(v)+1)
		}
	}
	if need == m.ScriptVariables() {
		return nil
	}
	last := m.Mark() - 1
	if err := m.SetScriptVariables(need); err != nil {
		return err
	}
	m.Squash(last)
	return nil
}

// triggerSlot is a trigger slot key's family and index: left, right, cmp
// and group name a condition, action an action id. ok is false for any
// other key; an index past the extended record's slots is refused by the
// write.
func triggerSlot(key string) (family string, n int, ok bool) {
	for _, f := range []string{"left", "right", "cmp", "group", "action"} {
		if rest, found := strings.CutPrefix(key, f); found {
			n, err := strconv.Atoi(rest)
			if err != nil || n < 0 || n > scriptprofile.ExtendedConditions || rest != strconv.Itoa(n) {
				return "", 0, false
			}
			return f, n, true
		}
	}
	return "", 0, false
}

// wholeTrigger reports that the set writes trigger index's slots as one
// trigger: on an extended map, or when a slot is one the 184-byte record
// does not have (a fourth pair, a fifth action, a group).
func wholeTrigger(m *mapedit.Editor, index int, values []ui.InspectionFieldValue) bool {
	if m.Extended() {
		return true
	}
	fields, err := m.Fields(mapedit.KindTrigger, index)
	if err != nil {
		return false
	}
	held := map[string]bool{}
	for _, f := range fields {
		held[f.Key] = true
	}
	for _, v := range values {
		if _, _, ok := triggerSlot(v.Key); ok && !held[v.Key] {
			return true
		}
	}
	return false
}

// setTriggerSlots writes the slot values of the set as trigger index's
// conditions and actions, through SetTriggerSlots: each value over the
// trigger as it stands, a new node made for a slot asking for one. On an
// extended map a condition with both sides empty and an empty action go,
// and the groups are numbered again from 0; on an original map only the
// empty slots at the end go. It answers the values it did not write.
func (s *mapInspectionEdits) setTriggerSlots(m *mapedit.Editor, index int, values []ui.InspectionFieldValue) ([]ui.InspectionFieldValue, error) {
	_, sc, err := wizardScript(m)
	if err != nil {
		return nil, err
	}
	if index < 0 || index >= len(sc.Triggers) {
		return nil, fmt.Errorf("no trigger %d", index+1)
	}
	old := sc.Triggers[index]
	t := alm.ScriptTrigger{Name: old.Name, Once: old.Once}
	conds, acts := old.Pairs(), len(old.Acts)
	for _, v := range values {
		if family, n, ok := triggerSlot(v.Key); ok {
			if family == "action" {
				acts = max(acts, n+1)
			} else {
				conds = max(conds, n+1)
			}
		}
	}
	// A condition past the trigger's joins its last group unless the set
	// names its group.
	for p := range conds {
		left, right, cmp := old.Pair(p)
		t.Left, t.Right, t.Cmp, t.Group = append(t.Left, left), append(t.Right, right), append(t.Cmp, cmp), append(t.Group, old.GroupOf(min(p, old.Pairs()-1)))
	}
	for a := range acts {
		t.Acts = append(t.Acts, old.Act(a))
	}
	var rest []ui.InspectionFieldValue
	for _, v := range values {
		family, n, ok := triggerSlot(v.Key)
		if !ok {
			rest = append(rest, v)
			continue
		}
		value := v.Value
		if v.New {
			node := mapedit.KindCondition
			if family == "action" {
				node = mapedit.KindAction
			}
			if family == "cmp" || family == "group" {
				return nil, fmt.Errorf("no new node for the field %q", v.Key)
			}
			id, err := addScriptRecord(m, node)
			if err != nil {
				return nil, err
			}
			value = int64(id)
		}
		if value < 0 || value > math.MaxUint32 {
			return nil, fmt.Errorf("%s: %d is outside 0..%d", v.Key, value, uint32(math.MaxUint32))
		}
		if err := scriptFieldAllowed(m, mapedit.KindTrigger, index, v.Key, value); err != nil {
			return nil, err
		}
		switch family {
		case "left":
			t.Left[n] = uint32(value)
		case "right":
			t.Right[n] = uint32(value)
		case "cmp":
			t.Cmp[n] = uint32(value)
		case "group":
			t.Group[n] = uint32(value)
		case "action":
			t.Acts[n] = uint32(value)
		}
	}
	if err := m.SetTriggerSlots(index, packTrigger(t, m.Extended())); err != nil {
		return nil, err
	}
	return rest, nil
}

// packTrigger drops t's empty slots as the form holds them: on an extended
// map every condition with both sides empty and every empty action, the
// groups numbered again so each condition is in its predecessor's group
// or the next; on an original map the empty slots at the end alone, so a
// slot keeps its place in the record.
func packTrigger(t alm.ScriptTrigger, extended bool) alm.ScriptTrigger {
	out := alm.ScriptTrigger{Name: t.Name, Once: t.Once}
	keep := func(p int) bool { return t.Left[p] != 0 || t.Right[p] != 0 }
	conds, acts := len(t.Left), len(t.Acts)
	for !extended && conds > 0 && !keep(conds-1) {
		conds--
	}
	for !extended && acts > 0 && t.Acts[acts-1] == 0 {
		acts--
	}
	var prev uint32
	for p := range conds {
		if extended && !keep(p) {
			continue
		}
		g := uint32(0)
		if n := len(out.Left); n > 0 {
			g = out.Group[n-1]
			if t.Group[p] != prev {
				g++
			}
		}
		prev = t.Group[p]
		out.Left, out.Right, out.Cmp, out.Group = append(out.Left, t.Left[p]), append(out.Right, t.Right[p]), append(out.Cmp, t.Cmp[p]), append(out.Group, g)
	}
	for _, id := range t.Acts[:acts] {
		if !extended || id != 0 {
			out.Acts = append(out.Acts, id)
		}
	}
	return out
}

// scriptSaveForm is the form question or refusal of a save of the model to
// path: an extended map saves under an .almx name; under an .alm name it is
// refused, unless the original records hold its script again and its texts
// are not inside, when the save asks to fold it first.
func (s *mapInspectionEdits) scriptSaveForm(path string) error {
	if !s.model.Extended() || namefold.HostEqual(filepath.Ext(path), extendedMapExt) {
		return nil
	}
	t := words.For(s.inspector.pageLanguage())
	if s.TextsInside() {
		return errors.New(t.Text("editor.save.textsinside.almx"))
	}
	trial := s.model.Clone()
	if err := foldScript(trial); err != nil {
		return fmt.Errorf(t.Text("editor.scriptform.almx"), err)
	}
	return &ui.ScriptFormNeeded{Fold: true, Why: t.Text("editor.scriptform.fits")}
}

// extendedName refuses map bytes that carry the extended script record or
// the texts record under a name that is not .almx.
func extendedName(data []byte, name string) error {
	m, err := alm.Open(data)
	if err != nil || !m.HasScriptRecord() && !m.HasTextsRecord() || namefold.HostEqual(filepath.Ext(name), extendedMapExt) {
		return nil
	}
	return fmt.Errorf("%s: an extended map is saved under an %s name; the original game does not open it", filepath.Base(name), extendedMapExt)
}
