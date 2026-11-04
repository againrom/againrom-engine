package ui

import "image"

func (e *MapEditor) keyTrapArea(string) error {
	if e.pick == pickTrapArea {
		e.leave(leavePick | leaveDrag)
		return nil
	}
	if _, ok := e.addHost(); !ok {
		return nil
	}
	e.leave(leaveDrag | leavePrompt | leavePick | leaveReport | leaveBrush)
	e.armPick(pickTrapArea)
	return nil
}

func (e *MapEditor) trapAreaDrawn(box image.Rectangle) {
	e.leave(leavePick)
	row, _ := addKindOf(KindEnchantment)
	run := &addRun{kind: row.Kind, title: e.word("editor.cast.area"), seed: InspectionNewRecord{
		Kind: KindEnchantment, Area: box, Cell: box.Min, HasCell: true,
		Values: []InspectionFieldValue{{Key: "template", Value: -1}},
	}}
	var steps []addStep
	for _, step := range row.Steps {
		if step.Key == "spell" || step.Key == "power" {
			steps = append(steps, step)
		}
	}
	run.setSteps(steps)
	e.openAddRun(run)
}
