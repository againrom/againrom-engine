package game

import (
	"fmt"

	"againrom/pkg/formats/sav"
)

// DocPayload returns a copy of the Againrom-only payload the campaign holds,
// or nil. It is read from the loaded SAV's carrier pairs and written back by
// the next SAVE; a feature that wants durable binary state in a SAV sets it
// with SetDocPayload and reads it here.
func (t *Town) DocPayload() *sav.DocPayload {
	if t == nil {
		return nil
	}
	return t.docPayload.Clone()
}

// SetDocPayload replaces the payload every later SAVE writes. Nil or an empty
// payload writes no carrier pairs. A payload that cannot fit the document list
// beside the current documents is refused with an error and the held payload
// is unchanged, so the session never takes in a payload no SAVE could carry.
func (t *Town) SetDocPayload(p *sav.DocPayload) error {
	if t == nil {
		return nil
	}
	if !sav.DocPayloadFits(p, len(t.documents)) {
		return fmt.Errorf("document payload does not fit the document list beside %d documents", len(t.documents))
	}
	t.docPayload = p.Clone()
	t.docPayloadError = ""
	return nil
}

// dropDocPayload is the SAVE-side rule: documents granted after
// SetDocPayload can leave a held payload without room. The SAVE writes the
// vanilla state without the payload, and the payload is dropped with a
// diagnostic in DocPayloadError.
func (t *Town) dropDocPayload(vanillaPairs int) {
	if t == nil {
		return
	}
	t.docPayloadError = fmt.Sprintf("document payload does not fit beside %d document pairs; dropped and not written", vanillaPairs)
	t.docPayload = nil
}

// DocPayloadError is why carrier pairs present in the loaded SAV were ignored,
// or empty. Ignored pairs are not written back.
func (t *Town) DocPayloadError() string {
	if t == nil {
		return ""
	}
	return t.docPayloadError
}
