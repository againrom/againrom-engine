package game

// playerMissionSave encodes the dialog label and writes the captured save
// point through the same producer used by ordinary SAVE and conversion.
func (f *FrontEnd) playerMissionSave(s Snapshot, label string) ([]byte, string, error) {
	encoded, err := encodeSaveLabel(label, f.textSelector())
	if err != nil {
		return nil, "", err
	}
	raw, err := f.ExportCurrentSave(s, encoded)
	return raw, "", err
}
