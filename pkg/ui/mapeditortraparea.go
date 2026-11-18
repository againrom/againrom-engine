package ui

import "image"

func (e *MapEditor) trapAreaTemplate() (int64, bool) {
	offer := e.castTemplateOffer()
	for _, c := range e.castConstructors() {
		if offer.Templates[c.Value].Target == "" {
			return c.Value, true
		}
	}
	return 0, false
}

func (e *MapEditor) keyTrapArea(string) error {
	if e.pick == pickTrapArea {
		e.leave(leavePick | leaveDrag)
		return nil
	}
	if _, ok := e.trapAreaTemplate(); !ok {
		return nil
	}
	e.leave(leaveDrag | leavePrompt | leavePick | leaveReport | leaveBrush)
	e.armPick(pickTrapArea)
	return nil
}

// trapAreaDrawn fills box with traps of the Traps and spells palette's spell
// and power at once, as one history entry.
func (e *MapEditor) trapAreaDrawn(box image.Rectangle) {
	e.leave(leavePick)
	spell, _, _ := e.castValue("spell")
	power, _, _ := e.castValue("power")
	template, ok := e.trapAreaTemplate()
	if !ok {
		return
	}
	ch, err := e.commandAddRecord(InspectionNewRecord{Kind: KindEnchantment, Area: box, Cell: box.Min, HasCell: true,
		Values: []InspectionFieldValue{{Key: "template", Value: template}, {Key: "spell", Value: spell}, {Key: "power", Value: power}}})
	if err != nil {
		e.message = e.editorHostText(err.Error())
		return
	}
	if ch.doc != nil {
		e.selectNewest(KindEnchantment, ch.doc.AddedRecord)
	}
}
