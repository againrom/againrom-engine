package ui

import (
	"strings"
)

func textInputKeys(in appInput) TextInputKeys {
	return TextInputKeys{Typed: in.Typed, Backspace: in.Backspace, Delete: in.Delete,
		Home: in.Home, End: in.End, Left: in.Left, Right: in.Right, SelectAll: in.SelectText, Ctrl: in.Viewer.Ctrl}
}

func editorSingleLine(s string, multiline bool) string {
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
	if !multiline {
		s = strings.ReplaceAll(s, "\n", " ")
	}
	return s
}
