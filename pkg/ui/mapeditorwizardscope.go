package ui

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/scriptprofile"
)

// wizardScope is the open trigger as the bindings of the wizard's
// description: its number and name, the once switch, one item for each
// condition and action with its parameters, and the flags that pick the
// headings, add rows, warnings and disabled buttons.
func (e *MapEditor) wizardScope(v scriptView) *popupScope {
	o := &e.obj
	w := o.wiz
	once := "editor.trigger.once.no"
	if strings.TrimSpace(o.buf[trigOnce]) != "0" {
		once = "editor.trigger.once.yes"
	}
	top := &popupScope{
		Values: map[string]string{"number": strconv.Itoa(v.index + 1), "name": e.editorHostText(o.buf[trigName])},
		Words:  map[string]string{"once": once},
		Fields: map[string]int{"name": trigName, "once": trigOnce},
		Lists:  map[string][]popupScope{},
	}
	at := map[string]int{}
	for i, f := range o.fields {
		at[f.Key] = i
	}
	conds, acts, heroes, drops := 0, 0, false, 0
	// Each OR group opens with "When" or "Or when" and its second condition
	// with "And if".
	inGroup, group := 0, uint32(0)
	for r, wr := range w.rows {
		if !wr.condition {
			continue
		}
		item := e.wizardItemScope(r, at, &heroes, &drops)
		if conds > 0 && wr.group != group {
			item.flag("newgroup")
			inGroup = 0
		}
		switch {
		case conds == 0:
			item.flag("first")
		case inGroup == 1:
			item.flag("second")
		}
		group = wr.group
		conds++
		inGroup++
		top.Lists["conditions"] = append(top.Lists["conditions"], item)
	}
	for r, wr := range w.rows {
		if !wr.condition {
			item := e.wizardItemScope(r, at, &heroes, &drops)
			if acts > 0 {
				item.Lists["gap"] = make([]popupScope, actionGapRows)
			}
			acts++
			top.Lists["actions"] = append(top.Lists["actions"], item)
		}
	}
	// The add rows reach the extended record's limits on any map: past the
	// original record's, Apply asks to extend the map.
	most := scriptprofile.Extended
	_, copies := e.copyOpen()
	for _, f := range []struct {
		name string
		on   bool
	}{
		{"noconditions", conds == 0},
		{"pairroom", conds == 0 || conds < most.Pairs},
		{"actionroom", acts < most.Actions},
		{"nocondwarn", conds == 0 && acts > drops},
		{"heroes", heroes && e.multiplayer()},
		{"clean", !e.objectDirty()},
		{"nocopies", !copies},
	} {
		if f.on {
			top.flag(f.name)
		}
	}
	return top
}

// wizardItemScope is wizard row r as a list item: its item choice, the
// "Other" texts of an item the wizard does not know, and its parameters,
// comparison and hint. heroes is set by a hero unit parameter, drops
// counted by a drop zone action.
func (e *MapEditor) wizardItemScope(r int, at map[string]int, heroes *bool, drops *int) popupScope {
	o := &e.obj
	w := o.wiz
	wr := w.rows[r]
	f := at[w.wizardKey(r, "item")]
	item := popupScope{
		Values: map[string]string{"value": choiceText(o.fields[f], o.buf[f])},
		Words:  map[string]string{},
		Fields: map[string]int{"item": f},
		Lists:  map[string][]popupScope{},
	}
	if wr.item < 0 {
		item.flag("other")
		var texts []string
		if wr.condition {
			_, lp := e.slotText(true, strconv.FormatUint(uint64(wr.left), 10))
			_, rp := e.slotText(true, strconv.FormatUint(uint64(wr.right), 10))
			texts = append(texts, lp, rp)
		} else {
			_, p := e.slotText(false, strconv.FormatUint(uint64(wr.act), 10))
			texts = append(texts, p)
		}
		for _, p := range texts {
			if p != "" {
				item.Lists["others"] = append(item.Lists["others"], popupScope{Values: map[string]string{"text": p}})
			}
		}
		return item
	}
	it := wizItems(wr.condition)[wr.item]
	if slices.ContainsFunc(it.params, func(p wizParam) bool { return p.kind == wpMessage }) {
		item.flag("configure")
	}
	if !wr.condition && it.opcode == scriptConstantOp {
		*drops++
	}
	for k, p := range it.params {
		if p.kind == wpMessage {
			continue // the item row's Configure edits the dialog
		}
		i := at[w.wizardKey(r, fmt.Sprintf("p%d", k))]
		param := popupScope{
			Values: map[string]string{"label": o.fields[i].Label, "value": o.buf[i]},
			Fields: map[string]int{"param": i},
		}
		switch p.kind {
		case wpUnit, wpPlayer, wpGroup, wpCommand, wpRelation, wpVariable, wpItem, wpStructure, wpSpell, wpProperty:
			param.flag("choose")
			param.Values["value"] = choiceText(o.fields[i], o.buf[i])
			if p.kind == wpUnit || p.kind == wpGroup || p.kind == wpStructure {
				param.flag("onmap")
			}
			if p.kind == wpUnit {
				if n, err := strconv.ParseInt(strings.TrimSpace(o.buf[i]), 10, 64); err == nil && n > 10000 {
					*heroes = true
				}
			}
		case wpCell:
			if wr.condition && it.hasRadius() {
				param.flag("area")
			} else {
				param.flag("cell")
			}
		case wpBox:
			param.flag("area")
		}
		item.Lists["params"] = append(item.Lists["params"], param)
	}
	if it.chooseCmp {
		i := at[w.wizardKey(r, "cmp")]
		item.flag("cmp")
		item.Values["cmp"] = choiceText(o.fields[i], o.buf[i])
		item.Fields["cmp"] = i
	}
	if it.hintWord != "" {
		item.flag("hint")
		item.Words["hint"] = it.hintWord
	}
	return item
}
