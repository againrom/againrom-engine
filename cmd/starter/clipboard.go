package main

import "againrom/pkg/ui"

func clipboardText() string {
	if !starterClipboardEnabled {
		return ""
	}
	s, err := ui.ReadTextClipboard()
	if err != nil {
		return ""
	}
	return s
}
