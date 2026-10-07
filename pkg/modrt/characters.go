package modrt

import "againrom/pkg/mod"

func (d *dataValue) loadCharacters(rel string, src []byte) error {
	look, err := mod.TextLookup(d.dir, d.sh.lang)
	if err != nil {
		return err
	}
	chars, err := mod.ParseCharacters(d.id, rel, src, look, d.sh.lang)
	if err != nil {
		return err
	}
	d.sh.characters.Rows = append(d.sh.characters.Rows, chars.Rows...)
	return nil
}
