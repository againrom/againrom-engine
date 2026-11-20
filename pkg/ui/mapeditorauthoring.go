package ui

import (
	"errors"
	"image"
	"strconv"
	"strings"
)

type InspectionRecords interface {
	AddStock(unit int) (*InspectionDocument, error)
	AddItemRecipe(kind RecordKind, index int, item int64) (*InspectionDocument, error)
}

type InspectionRecordChoices interface {
	RecordChoices(kind RecordKind, index int, key string) []InspectionChoice
}

// recipeItemField is the item recipe prompt's one field: the selected sack's
// items by their place in it, each with the picture its item field's item
// shows, so the prompt shows them as a picture grid.
func (e *MapEditor) recipeItemField() []InspectionField {
	f := InspectionField{Key: "recipeitem", Label: e.plainWord("editor.prompt.recipe")}
	kind, index := e.selectedRecord()
	if host, ok := Seam[InspectionRecordChoices](e.doc.Edits); ok {
		f.Choices = host.RecordChoices(kind, index, f.Key)
	}
	look, ok := Seam[InspectionItemLook](e.doc.Edits)
	if !ok || len(f.Choices) == 0 || index < 0 {
		return []InspectionField{f}
	}
	codes := map[string]int64{}
	for _, g := range e.doc.Edits.Fields(kind, index) {
		codes[g.Key] = g.Value
	}
	f.Icons = map[int64]*image.RGBA{}
	for _, c := range f.Choices {
		if _, icon := look.ItemLook(codes["item"+strconv.FormatInt(c.Value, 10)]); icon != nil {
			f.Icons[c.Value] = icon
		}
	}
	return []InspectionField{f}
}

func (e *MapEditor) keyRecords(key string) error {
	if !e.editable() || e.catalog {
		return nil
	}
	if key == "structure" {
		e.showRecords(2)
		return nil
	}
	if _, ok := Seam[InspectionRecords](e.doc.Edits); !ok {
		return nil
	}
	e.leave(leaveDrag | leavePick | leavePrompt | leaveBrush)
	kind, index := e.selectedRecord()
	if key == "stock" {
		if kind != KindUnit {
			e.message = e.plainWord("editor.stock.selectunit")
			return nil
		}
		// The new stock opens in its window as the unit's sack row opens
		// it: Back returns to the unit.
		back := append(e.objectBack, e.objectPlace())
		ch, err := e.commandAddStock(index)
		if err != nil {
			e.message = err.Error()
			return err
		}
		if ch.doc != nil {
			e.selectNewest(KindStock)
			e.objectBack = back
			e.message = e.plainWord("editor.stock.created")
		}
		return nil
	}
	if kind != KindStock && kind != KindSack {
		e.message = e.plainWord("editor.recipe.selectsack")
		return nil
	}
	e.openPrompt(editorPrompt{kind: promptItemRecipe})
	return nil
}

func (e *MapEditor) setItemRecipe(text string) error {
	n, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
	if err != nil {
		e.message = e.plainWord("editor.recipe.choose")
		return errors.New(e.message)
	}
	kind, index := e.selectedRecord()
	if _, ok := Seam[InspectionRecords](e.doc.Edits); !ok {
		return nil
	}
	ch, err := e.commandAddItemRecipe(itemRecipe{Kind: kind, Index: index, Item: n})
	if err != nil {
		e.message = err.Error()
		return err
	}
	e.leave(leavePrompt)
	if ch.doc != nil {
		e.selectNewest(KindEnchantment)
		e.message = e.plainWord("editor.recipe.created")
	}
	return nil
}
