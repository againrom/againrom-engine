package sav

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
)

// DocPayload is an Againrom-only byte string carried inside an ordinary SAV.
// The campaign record's document list is `count` then `count` pairs of
// (value, kind) read and written raw by the original game. A pair whose kind
// has its high bit set is a carrier: it holds 31 bits of the payload envelope
// in the low bits of kind, and the original game keeps it across LOAD and
// SAVE. The codec knows nothing about what the bytes mean.
type DocPayload struct {
	// Version is the envelope version. A version other than PayloadVersion
	// round-trips verbatim; the codec does not interpret Data.
	Version uint16
	Data    []byte
}

const (
	// PayloadVersion is the envelope version this package writes for new
	// payloads.
	PayloadVersion = 1

	carrierBit      = uint32(1) << 31
	carrierValue    = 1
	carrierChunkMax = carrierBit - 1

	payloadHeaderLen  = 10 // magic, version, length
	payloadTrailerLen = 4  // CRC-32
	payloadMagic      = "ARSV"

	// maxCampaignDocumentPairs bounds the whole document list, vanilla pairs
	// and carriers together. 26426 carriers hold 100 KiB.
	maxCampaignDocumentPairs = 65536
	// maxPayloadData is the largest payload whose carriers fit the list.
	maxPayloadData = (maxCampaignDocumentPairs*31)/8 - payloadHeaderLen - payloadTrailerLen
)

// IsCarrierKind reports whether a document pair's kind marks it as a carrier.
func IsCarrierKind(kind uint32) bool { return kind&carrierBit != 0 }

// Clone returns a deep copy; a nil receiver clones to nil.
func (p *DocPayload) Clone() *DocPayload {
	if p == nil {
		return nil
	}
	return &DocPayload{Version: p.Version, Data: append([]byte(nil), p.Data...)}
}

// empty reports a payload that is written as zero carriers: nil, or the
// current version with no bytes.
func (p *DocPayload) empty() bool {
	return p == nil || (len(p.Data) == 0 && p.Version <= PayloadVersion)
}

func payloadFrame(p *DocPayload) []byte {
	frame := make([]byte, 0, payloadHeaderLen+len(p.Data)+payloadTrailerLen)
	frame = append(frame, payloadMagic...)
	frame = binary.LittleEndian.AppendUint16(frame, p.Version)
	frame = binary.LittleEndian.AppendUint32(frame, uint32(len(p.Data)))
	frame = append(frame, p.Data...)
	return binary.LittleEndian.AppendUint32(frame, crc32.ChecksumIEEE(frame[len(payloadMagic):]))
}

// carrierCount is the number of 31-bit chunks that hold n bytes.
func carrierCount(n int) int { return (8*n + 30) / 31 }

// DocPayloadFits reports whether the payload's carriers fit the document list
// next to the given number of vanilla pairs. A nil or empty payload always
// fits.
func DocPayloadFits(p *DocPayload, vanillaPairs int) bool {
	if p.empty() {
		return true
	}
	if len(p.Data) > maxPayloadData || vanillaPairs < 0 {
		return false
	}
	return vanillaPairs+carrierCount(payloadHeaderLen+len(p.Data)+payloadTrailerLen) <= maxCampaignDocumentPairs
}

// EncodeDocPayload packs the payload envelope into 31-bit chunks, least
// significant bit first, in order. A nil or empty payload has no chunks. The
// output is a pure function of the payload. A payload larger than the list
// bound yields an error.
func EncodeDocPayload(p *DocPayload) ([]uint32, error) {
	if p.empty() {
		return nil, nil
	}
	if len(p.Data) > maxPayloadData {
		return nil, fmt.Errorf("sav: payload of %d bytes exceeds the %d-byte limit", len(p.Data), maxPayloadData)
	}
	frame := payloadFrame(p)
	chunks := make([]uint32, carrierCount(len(frame)))
	for bit := 0; bit < 8*len(frame); bit++ {
		if frame[bit>>3]>>(bit&7)&1 != 0 {
			chunks[bit/31] |= 1 << (bit % 31)
		}
	}
	return chunks, nil
}

// DecodeDocPayload reads an envelope from the whole sequence of chunks. Any
// shortfall, surplus, non-zero padding, bad magic, impossible length or CRC
// mismatch is an error and yields no payload. It never panics and sizes
// nothing from the chunks before checking them against the chunk count.
func DecodeDocPayload(chunks []uint32) (*DocPayload, error) {
	if len(chunks) == 0 {
		return nil, errors.New("sav: payload has no carrier records")
	}
	if len(chunks) > maxCampaignDocumentPairs {
		return nil, fmt.Errorf("sav: payload has %d carrier records, limit %d", len(chunks), maxCampaignDocumentPairs)
	}
	for i, c := range chunks {
		if c > carrierChunkMax {
			return nil, fmt.Errorf("sav: carrier record %d holds more than 31 bits", i)
		}
	}
	raw := make([]byte, (31*len(chunks))/8)
	for bit := 0; bit < 8*len(raw); bit++ {
		if chunks[bit/31]>>(bit%31)&1 != 0 {
			raw[bit>>3] |= 1 << (bit & 7)
		}
	}
	if len(raw) < payloadHeaderLen+payloadTrailerLen || string(raw[:len(payloadMagic)]) != payloadMagic {
		return nil, errors.New("sav: payload envelope magic is missing")
	}
	length := uint64(binary.LittleEndian.Uint32(raw[6:10]))
	frameLen := uint64(payloadHeaderLen+payloadTrailerLen) + length
	if frameLen > uint64(len(raw)) {
		return nil, fmt.Errorf("sav: payload length %d overruns its %d carrier records", length, len(chunks))
	}
	if carrierCount(int(frameLen)) != len(chunks) {
		return nil, fmt.Errorf("sav: payload length %d does not match its %d carrier records", length, len(chunks))
	}
	n := int(frameLen)
	for bit := 8 * n; bit < 31*len(chunks); bit++ {
		if chunks[bit/31]>>(bit%31)&1 != 0 {
			return nil, errors.New("sav: payload padding bits are not zero")
		}
	}
	crcAt := n - payloadTrailerLen
	if crc32.ChecksumIEEE(raw[len(payloadMagic):crcAt]) != binary.LittleEndian.Uint32(raw[crcAt:n]) {
		return nil, errors.New("sav: payload CRC-32 does not match")
	}
	return &DocPayload{
		Version: binary.LittleEndian.Uint16(raw[4:6]),
		Data:    append([]byte(nil), raw[payloadHeaderLen:crcAt]...),
	}, nil
}

// splitDocumentPairs separates a campaign document list into vanilla pairs,
// in file order, and carrier chunks, in file order wherever they sit between
// the vanilla pairs. A pair with the high bit clear and a kind above 1 is not
// a carrier and is refused.
func splitDocumentPairs(pairs [][2]uint32) (vanilla, carriers [][2]uint32, err error) {
	for i, pair := range pairs {
		switch {
		case IsCarrierKind(pair[1]):
			carriers = append(carriers, pair)
		case pair[1] > 1:
			return nil, nil, fmt.Errorf("document %d kind is %d, want 0 or 1, or a carrier", i, pair[1])
		default:
			vanilla = append(vanilla, pair)
		}
	}
	if len(vanilla) > maxCampaignElements {
		return nil, nil, fmt.Errorf("document count %d exceeds %d", len(vanilla), maxCampaignElements)
	}
	return vanilla, carriers, nil
}

// decodeCarrierPairs decodes the carrier pairs of one record. No carriers is
// no payload and no error.
func decodeCarrierPairs(carriers [][2]uint32) (*DocPayload, string) {
	if len(carriers) == 0 {
		return nil, ""
	}
	chunks := make([]uint32, len(carriers))
	for i, pair := range carriers {
		chunks[i] = pair[1] &^ carrierBit
	}
	payload, err := DecodeDocPayload(chunks)
	if err != nil {
		return nil, err.Error()
	}
	return payload, ""
}

func carrierPairs(p *DocPayload) ([][2]uint32, error) {
	chunks, err := EncodeDocPayload(p)
	if err != nil || len(chunks) == 0 {
		return nil, err
	}
	pairs := make([][2]uint32, len(chunks))
	for i, c := range chunks {
		pairs[i] = [2]uint32{carrierValue, carrierBit | c}
	}
	return pairs, nil
}

// ProjectDocumentPayload replaces the document's carrier pairs with the
// current payload: one block, the old block dropped. A nil or empty payload
// leaves none. A payload that does not fit beside the vanilla pairs is an
// error.
func ProjectDocumentPayload(doc *DocumentData, p *DocPayload) error {
	if !DocPayloadFits(p, len(doc.Campaign.Documents)) {
		return fmt.Errorf("sav: payload of %d bytes does not fit beside %d document pairs", len(p.Data), len(doc.Campaign.Documents))
	}
	pairs, err := carrierPairs(p)
	if err != nil {
		return err
	}
	doc.Campaign.Carriers = pairs
	return nil
}

// DocumentPayload decodes the payload carried by a document's carrier pairs.
// The second result is why a present block did not decode; a decoded payload
// and an empty reason, or neither, mean the block is valid or absent.
func DocumentPayload(doc DocumentData) (*DocPayload, string) {
	return decodeCarrierPairs(doc.Campaign.Carriers)
}
