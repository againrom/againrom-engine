package ui

import (
	"slices"
	"strconv"
)

var castSettings = []struct {
	key, word string
	step      int64
}{
	{key: "template", word: "editor.cast.where", step: 1},
	{key: "castkind", word: "editor.cast.kind", step: 1},
	{key: "power", word: "editor.cast.power", step: 5},
	{key: "range", word: "editor.cast.range", step: 5},
}

// The template offer changes with the document, including when a spell is
// deleted or an Undo restores it.
func (e *MapEditor) castTemplateOffer() InspectionNewStep {
	host, ok := e.addHost()
	if !ok {
		return InspectionNewStep{}
	}
	c := &e.place.cast
	if c.doc != e.doc {
		row, _ := addKindOf(KindEnchantment)
		c.offer, _ = e.addOffer(host, &addRun{kind: row.Kind, steps: row.Steps}, 0)
		c.doc = e.doc
	}
	return c.offer
}

func (e *MapEditor) castTemplate() int64 {
	c := &e.place.cast
	if c.source != 0 {
		if id := e.linkedRecord(RecordRef{}, c.source); id >= 0 {
			return int64(e.doc.Records[id].Index)
		}
		return c.constructor
	}
	if v, ok := c.values["template"]; ok {
		return v
	}
	return e.castTemplateOffer().Value
}

func (e *MapEditor) castConstructors() []InspectionChoice {
	offer := e.castTemplateOffer()
	var choices []InspectionChoice
	for _, c := range offer.Choices {
		if where, ok := offer.Templates[c.Value]; ok && where.Constructor == c.Value {
			choices = append(choices, c)
		}
	}
	return choices
}

func (e *MapEditor) castWhere() InspectionChoice {
	offer := e.castTemplateOffer()
	where, ok := offer.Templates[e.castTemplate()]
	if !ok {
		return InspectionChoice{Value: offer.Value}
	}
	for _, c := range offer.Choices {
		if c.Value == where.Constructor {
			return c
		}
	}
	return InspectionChoice{Value: offer.Value}
}

func (e *MapEditor) castOffer(key string) (InspectionNewStep, bool) {
	host, ok := e.addHost()
	if !ok {
		return InspectionNewStep{}, false
	}
	row, _ := addKindOf(KindEnchantment)
	r := &addRun{kind: row.Kind}
	r.setSteps(row.Steps)
	e.placeValues(r)
	for i, st := range r.steps {
		if st.Key == key {
			offer, skip := e.addOffer(host, r, i)
			return offer, !skip
		}
	}
	return InspectionNewStep{}, false
}

func (e *MapEditor) castValue(key string) (int64, InspectionNewStep, bool) {
	if key == "template" {
		choices := e.castConstructors()
		return e.castWhere().Value, InspectionNewStep{Choices: choices}, len(choices) > 0
	}
	offer, asked := e.castOffer(key)
	if v, ok := e.place.cast.values[key]; ok {
		return v, offer, asked
	}
	return offer.Value, offer, asked
}

func (e *MapEditor) castLabel(offer InspectionNewStep, word string) string {
	if offer.Label != "" {
		return offer.Label
	}
	return e.word(word)
}

func (e *MapEditor) castRows() []popupRow {
	if kind, _, _, ok := e.placeGridOf(); !ok || kind != KindEnchantment {
		return nil
	}
	buttons := []string{e.word("editor.brush.less"), e.word("editor.brush.more")}
	var rows []popupRow
	for _, s := range castSettings {
		v, offer, asked := e.castValue(s.key)
		if !asked {
			continue
		}
		row := popupRow{Kind: popupText, Label: e.castLabel(offer, s.word), Value: strconv.FormatInt(v, 10), Buttons: buttons}
		for _, c := range offer.Choices {
			if c.Value == v {
				row.Value = c.Label
				break
			}
		}
		rows = append(rows, row)
	}
	return rows
}

func (e *MapEditor) castRowPress(s popupSpec, n, k int) bool {
	if kind, _, _, ok := e.placeGridOf(); !ok || kind != KindEnchantment || n < 0 || n >= len(s.Rows) {
		return false
	}
	for _, set := range castSettings {
		v, offer, asked := e.castValue(set.key)
		if !asked || s.Rows[n].Label != e.castLabel(offer, set.word) {
			continue
		}
		dir := int64(2*k - 1)
		switch choices := offer.Choices; {
		case len(choices) > 0:
			at := slices.IndexFunc(choices, func(c InspectionChoice) bool { return c.Value == v })
			at = (at + int(dir) + 2*len(choices)) % len(choices)
			if set.key == "template" {
				e.place.cast.source = 0
			}
			e.place.cast.set(set.key, choices[at].Value)
		case offer.Bounds != nil:
			e.place.cast.set(set.key, max(offer.Bounds.Min, min(offer.Bounds.Max, v+dir*set.step)))
		}
		return true
	}
	return false
}

// A selected spell supplies the settings and remains the copy source as
// records change index. Its constructor remains usable if it is deleted.
func (e *MapEditor) castFromRecord(record InspectionRecord) {
	index := int64(record.Index)
	where, ok := e.castTemplateOffer().Templates[index]
	if !ok {
		return
	}
	c := &e.place.cast
	c.values, c.source, c.constructor = nil, record.Key, where.Constructor
	c.set("template", index)
	for _, key := range []string{"spell", "power", "castkind", "range"} {
		if offer, asked := e.castOffer(key); asked {
			c.set(key, offer.Value)
		}
	}
	e.placeStart(KindEnchantment, map[string]int64{"spell": c.values["spell"]})
}
