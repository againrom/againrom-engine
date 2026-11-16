package ui

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/scriptprofile"
)

// wizardSpecBefore is the wizard as the editor built it in code before its
// description: the witness the built window is compared with. It is kept
// as it stood, line for line.
func (e *MapEditor) wizardSpecBefore(v scriptView, s *popupSpec, add func(popupRow, int, ...string), text func(string, string)) {
	o := &e.obj
	w := o.wiz
	s.Title = fmt.Sprintf(e.word("editor.trigger.title"), v.index+1)
	add(popupRow{Kind: popupEdit, Label: o.fields[trigName].Label, Value: e.editorHostText(o.buf[trigName])}, trigName)
	once := e.word("editor.trigger.once.no")
	if strings.TrimSpace(o.buf[trigOnce]) != "0" {
		once = e.word("editor.trigger.once.yes")
	}
	add(popupRow{Kind: popupSwitch, Label: o.fields[trigOnce].Label, Value: once}, trigOnce)
	at := map[string]int{}
	for i, f := range o.fields {
		at[f.Key] = i
	}
	conds, acts, heroes, drops := 0, 0, false, 0
	row := func(r int) {
		wr := w.rows[r]
		f := at[w.wizardKey(r, "item")]
		label := o.fields[f].Label
		ink := editorCondition
		if !wr.condition {
			ink = editorGold
		}
		acts := []string{actRemoveRow}
		if wr.item >= 0 && slices.ContainsFunc(wizItems(wr.condition)[wr.item].params, func(p wizParam) bool { return p.kind == wpMessage }) {
			acts = []string{actConfigure, actRemoveRow}
		}
		add(popupRow{Kind: popupChoice, Label: label, Value: choiceText(o.fields[f], o.buf[f]), Ink: ink}, f, acts...)
		if wr.item < 0 {
			if wr.condition {
				_, lp := e.slotText(true, strconv.FormatUint(uint64(wr.left), 10))
				_, rp := e.slotText(true, strconv.FormatUint(uint64(wr.right), 10))
				for _, p := range []string{lp, rp} {
					if p != "" {
						text("    "+p, "")
					}
				}
			} else if _, p := e.slotText(false, strconv.FormatUint(uint64(wr.act), 10)); p != "" {
				text("    "+p, "")
			}
			text("    "+e.word("editor.wizard.otherhint"), "")
			return
		}
		it := wizItems(wr.condition)[wr.item]
		if !wr.condition && it.opcode == scriptConstantOp {
			drops++
		}
		for k, p := range it.params {
			if p.kind == wpMessage {
				continue
			}
			i := at[w.wizardKey(r, fmt.Sprintf("p%d", k))]
			pr := popupRow{Kind: popupEdit, Label: "  " + o.fields[i].Label, Value: o.buf[i]}
			var buttons []string
			switch p.kind {
			case wpUnit, wpPlayer, wpGroup, wpCommand, wpRelation, wpVariable, wpItem, wpStructure, wpSpell, wpProperty:
				pr.Kind, pr.Value = popupChoice, choiceText(o.fields[i], o.buf[i])
				if p.kind == wpUnit || p.kind == wpGroup || p.kind == wpStructure {
					buttons = []string{actPickParam}
				}
				if p.kind == wpUnit {
					if n, err := strconv.ParseInt(strings.TrimSpace(o.buf[i]), 10, 64); err == nil && n > 10000 {
						heroes = true
					}
				}
			case wpCell:
				buttons = []string{actPick}
				if wr.condition && it.hasRadius() {
					buttons = []string{actDraw, actPickArea}
				}
			case wpBox:
				buttons = []string{actDraw, actPickArea}
			}
			add(pr, i, buttons...)
		}
		if it.chooseCmp {
			i := at[w.wizardKey(r, "cmp")]
			add(popupRow{Kind: popupChoice, Label: "  " + o.fields[i].Label, Value: choiceText(o.fields[i], o.buf[i])}, i)
		}
		if it.hintWord != "" {
			text("    "+e.word(it.hintWord), "")
		}
	}
	inGroup, group := 0, uint32(0)
	for r, wr := range w.rows {
		if !wr.condition {
			continue
		}
		if conds > 0 && wr.group != group {
			add(popupRow{Kind: popupHeading, Label: e.word("editor.wizard.orwhen")}, -1)
			inGroup = 0
		}
		switch {
		case conds == 0:
			add(popupRow{Kind: popupHeading, Label: e.word("editor.wizard.when")}, -1)
		case inGroup == 1:
			add(popupRow{Kind: popupHeading, Label: e.word("editor.wizard.andif")}, -1)
		}
		group = wr.group
		conds++
		inGroup++
		row(r)
	}
	if conds == 0 {
		add(popupRow{Kind: popupHeading, Label: e.word("editor.wizard.when")}, -1)
	}
	if most := scriptprofile.Extended; conds == 0 {
		add(popupRow{Kind: popupText}, -1, actAddPair)
	} else if conds < most.Pairs {
		add(popupRow{Kind: popupText}, -1, actAddPair, actAddGroup)
	}
	add(popupRow{Kind: popupHeading, Label: e.word("editor.wizard.do")}, -1)
	for r, wr := range w.rows {
		if !wr.condition {
			actionGap(add, acts)
			acts++
			row(r)
		}
	}
	if acts < scriptprofile.Extended.Actions {
		add(popupRow{Kind: popupText}, -1, actAddAct)
	}
	if conds == 0 && acts > drops {
		text(e.word("editor.wizard.nocondition"), "warning")
	}
	if heroes && e.multiplayer() {
		text(e.word("editor.wizard.multiplayer"), "warning")
	}
	_, copies := e.copyOpen()
	s.Buttons = []string{e.word("editor.button.apply"), e.word("editor.button.cancel"), e.word("editor.trigger.new"), e.word("editor.trigger.delete"), e.word("editor.wizard.details"), e.word("editor.trigger.copy")}
	s.Disabled = []bool{!e.objectDirty(), false, false, false, false, !copies}
	fitButtons(s, e.bodyFont())
}
