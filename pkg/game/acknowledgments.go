package game

import "strconv"

const acknowledgmentKey = "Acknowledgement"

// Missing or malformed preferences retain the enabled startup policy. Any
// valid nonzero integer follows the original chooser's zero/nonzero gate.
func (s OptionsStore) Acknowledgments() (bool, error) {
	m, err := s.readAll()
	if err != nil {
		return true, err
	}
	v, err := strconv.Atoi(m[acknowledgmentKey])
	return err != nil || v != 0, nil
}

func (f *FrontEnd) setAcknowledgments(on bool) error {
	if f.Options.Path != "" {
		m, err := f.Options.readAll()
		if err != nil {
			return err
		}
		m[acknowledgmentKey] = strconv.Itoa(boolOption(on))
		if err := f.Options.writeAll(m); err != nil {
			return err
		}
	}
	f.acknowledgmentsOff = !on
	return nil
}
