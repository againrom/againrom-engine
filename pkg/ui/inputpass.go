package ui

// inputGroup is a group of appInput fields a capturing surface passes on.
type inputGroup uint8

const (
	passCursor inputGroup = 1 << iota
	passButtons
	passAlt
	passCtrlShift
	passFocus
	passKeys
	passHelp
)

const (
	passCheatChat   = passCursor | passAlt | passFocus | passHelp
	passKeysCapture = passCursor | passButtons | passAlt | passCtrlShift | passFocus | passKeys
)

// passing keeps the named groups of in and drops the rest.
func (in appInput) passing(groups inputGroup) appInput {
	var out appInput
	if groups&passCursor != 0 {
		out.CursorX, out.CursorY = in.CursorX, in.CursorY
		out.Viewer.CursorX, out.Viewer.CursorY = in.Viewer.CursorX, in.Viewer.CursorY
	}
	if groups&passButtons != 0 {
		out.PrimaryPressed, out.PrimaryReleased = in.PrimaryPressed, in.PrimaryReleased
		out.SecondaryPressed, out.SecondaryReleased = in.SecondaryPressed, in.SecondaryReleased
	}
	if groups&passAlt != 0 {
		out.Viewer.Alt = in.Viewer.Alt
	}
	if groups&passCtrlShift != 0 {
		out.Viewer.Ctrl, out.Viewer.Shift = in.Viewer.Ctrl, in.Viewer.Shift
	}
	if groups&passFocus != 0 {
		out.Unfocused, out.Viewer.Unfocused = in.Unfocused, in.Unfocused
	}
	if groups&passKeys != 0 {
		out.AnyKey, out.Keys, out.Close, out.Escape = in.AnyKey, in.Keys, in.Close, in.Escape
	}
	if groups&passHelp != 0 {
		out.Help, out.AltLetter = in.Help, in.AltLetter
	}
	return out
}
