package ui

import (
	"fmt"
	"strconv"
	"strings"
)

type InspectionRecords interface {
	AddStock(unit int) (*InspectionDocument, error)
	AddItemRecipe(kind string, index int, item int64) (*InspectionDocument, error)
}

type InspectionRecordChoices interface {
	RecordChoices(kind string, index int, key string) []InspectionChoice
}

func (e *MapEditor) recipeItemField() []InspectionField {
	f := InspectionField{Key: "recipeitem", Label: "Item number for magic recipe"}
	kind, index := e.selectedRecord()
	if host, ok := e.doc.Edits.(InspectionRecordChoices); ok {
		f.Choices = host.RecordChoices(kind, index, f.Key)
	}
	return []InspectionField{f}
}

func (e *MapEditor) keyRecords(key string) error {
	if !e.editable() || e.catalog {
		return nil
	}
	if key == "structure" {
		e.leave(leaveReport)
		e.openAdd("Structure")
		return nil
	}
	host, ok := e.doc.Edits.(InspectionRecords)
	if !ok {
		return nil
	}
	e.leave(leaveDrag | leavePick | leavePrompt | leaveBrush | leaveReport)
	kind, index := e.selectedRecord()
	if key == "stock" {
		if kind != "Unit" {
			e.message = "Select a unit first; K creates its stock."
			return nil
		}
		next, err := host.AddStock(index)
		if err := e.adoptEdit(next, err); err != nil {
			return err
		}
		if next != nil {
			e.selectNewest("Stock")
			e.message = "Stock created. E adds items; I adds an item recipe."
		}
		return nil
	}
	if kind != "Stock" && kind != "Sack" {
		e.message = "Select a sack or stock first; I creates an item recipe."
		return nil
	}
	e.openPrompt(editorPrompt{kind: promptItemRecipe})
	return nil
}

func (e *MapEditor) setItemRecipe(text string) error {
	n, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
	if err != nil {
		e.message = "Enter the item number, starting at 1."
		return fmt.Errorf("%s", e.message)
	}
	kind, index := e.selectedRecord()
	host, ok := e.doc.Edits.(InspectionRecords)
	if !ok {
		return nil
	}
	next, err := host.AddItemRecipe(kind, index, n)
	if err := e.adoptEdit(next, err); err != nil {
		return err
	}
	e.leave(leavePrompt)
	if next != nil {
		e.selectNewest("Enchantment")
		e.message = "Recipe created. E edits its spell and ordered effects."
	}
	return nil
}
