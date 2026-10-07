//go:build !windows

package main

// clipboardText has no source off Windows; pasting does nothing there.
func clipboardText() string { return "" }
