package main

import "againrom/pkg/ui/widgetkit"

func clipboardText() string {
	if !starterClipboardEnabled {
		return ""
	}
	s, err := widgetkit.ReadTextClipboard()
	if err != nil {
		return ""
	}
	return s
}
