//go:build !windows && !darwin

package systemclick

// read is Fallback: this platform's setting is not read.
func read() Setting { return Fallback }
