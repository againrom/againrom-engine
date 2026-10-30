package sim

// LoadedRecord is the kind of SAV record a World was last restored from. The
// SAV LOAD writes it; a SAVE's absence rules and a mission's seat read it, so
// what they leave out or seat follows the World and not a document kept
// beside it.
type LoadedRecord uint8

const (
	// NoLoadedRecord is a World a mission started, or one restored from a
	// snapshot that held no SAV document.
	NoLoadedRecord LoadedRecord = iota
	// LoadedDocument is a World restored from a record whose SAV document
	// decoded.
	LoadedDocument
	// LoadedWithoutDocument is a World restored from a record whose SAV
	// document did not decode: a partial legacy input or a second-game
	// mission file.
	LoadedWithoutDocument
)

// LoadedRecord returns the kind of record the World was last restored from.
func (w *World) LoadedRecord() LoadedRecord { return w.loadedRecord }

// SetLoadedRecord records the kind of record a LOAD restored the World from.
// It is provenance, not play state: no tick reads it, the byte form does not
// carry it and a decode carries the receiver's value across.
func (w *World) SetLoadedRecord(r LoadedRecord) { w.loadedRecord = r }
