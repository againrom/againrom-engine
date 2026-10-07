package ui

// TownDialogueLifecycle owns the current spoken line separately from the
// dialogue's pixels and the room's ambient sounds.
type TownDialogueLifecycle interface {
	TownDialogueActive(bool)
}
