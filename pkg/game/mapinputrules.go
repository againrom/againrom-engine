package game

import (
	"unicode"
	"unicode/utf8"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mod"
)

func mapInputRule(codec alm.TextField) (int, func(rune) int) {
	limit := codec.Width
	if codec.Terminated {
		limit--
	}
	return limit, func(r rune) int {
		b, err := codec.Encode(string(r))
		if err != nil {
			return 0
		}
		return len(b)
	}
}

func (s *mapInspectionEdits) InputRule(scope, kind, key string) (int, func(rune) int) {
	if scope == "metadata" {
		switch key {
		case "name":
			return mapInputRule(alm.MapNameText)
		case "description":
			return mapInputRule(alm.DescriptionText)
		}
	}
	if scope == "package" && key == "title" {
		return 1024, nil
	}
	if key == "name" {
		switch kind {
		case "Player":
			return mapInputRule(alm.PlayerNameText)
		case "Trigger", "Condition", "Action":
			return mapInputRule(alm.ScriptLabelText)
		}
	}
	if scope != "pages" || key != "body" {
		return 0, nil
	}
	x := s.inspector
	selector := 0
	if x.font != nil {
		selector = x.font.Selector
	}
	return mod.MaxFileBytes, func(r rune) int {
		if unicode.IsControl(r) {
			return 0
		}
		if _, dropped := ModTextDropped([]byte(string(r)), selector); dropped {
			return 0
		}
		return utf8.RuneLen(r)
	}
}
