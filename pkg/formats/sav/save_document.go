package sav

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// saveDocument joins the complete wire document with detached application
// state and campaign values. It owns no File.Body, Store, TailRest, offsets or
// compressed input. Producers still have to supply current rather than stale
// imported values; this codec does not assert a gameplay admission policy.
type saveDocument struct {
	version  uint32
	label    []byte
	archive  *archiveDocument
	state    *cityState
	campaign *cityCampaign
}

func parseSaveDocument(b []byte) (*saveDocument, error) {
	if err := checkArchiveContainerSpan(b); err != nil {
		return nil, err
	}
	f, err := Open(b)
	if err != nil {
		return nil, err
	}
	d := &saveDocument{version: f.Version, label: append([]byte(nil), f.Label...)}
	d.archive, err = parseArchiveDocument(f.Body)
	if err != nil {
		return nil, err
	}
	if d.archive.world != nil {
		d.state, err = parseWorldState(f.Store)
	} else {
		d.state, err = parseCityState(f.Store)
	}
	if err != nil {
		return nil, err
	}
	for path, value := range d.state.values {
		if value.kind != 2 {
			value.int32 = 0 // Source pool offsets are not typed leaf values.
			d.state.values[path] = value
		}
	}
	d.campaign, err = parseCityCampaign(f.TailRest)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// Preflight the compressed extent before Open allocates its output. Both the
// declared count and the actual opcode expansion are bounded; trusting only
// the declared count would permit a small declared span with oversized runs.
func checkArchiveContainerSpan(b []byte) error {
	if len(b) < headerLen+4 || len(b) > maxArchiveOutput {
		return fmt.Errorf("sav: container size %d outside archive bounds", len(b))
	}
	end := uint64(binary.LittleEndian.Uint32(b[4:]))
	if end < headerLen+4 || end > uint64(len(b)) {
		return fmt.Errorf("sav: invalid archive blob extent %d", end)
	}
	declared := uint64(binary.LittleEndian.Uint32(b[headerLen:]))
	if declared > maxArchiveOutput/2 {
		return fmt.Errorf("sav: declared archive body exceeds output bound")
	}
	var words uint64
	for at := headerLen + 4; at < int(end); {
		op := b[at]
		at++
		n, span := int(op), 2*int(op)
		if op&0x80 != 0 {
			n, span = int(op&0x7f), 2
		}
		if span > int(end)-at {
			return fmt.Errorf("sav: archive opcode exceeds blob extent")
		}
		at += span
		words += uint64(n)
		if words > maxArchiveOutput/2 {
			return fmt.Errorf("sav: expanded archive body exceeds output bound")
		}
	}
	if words != declared {
		return fmt.Errorf("sav: archive body declares %d words, opcodes emit %d", declared, words)
	}
	return nil
}

func serializeSaveDocument(d *saveDocument) ([]byte, error) {
	if d == nil || d.archive == nil {
		return nil, fmt.Errorf("sav: missing save document")
	}
	if d.version < MinVersion {
		return nil, fmt.Errorf("sav: document version %#x is below %#x", d.version, MinVersion)
	}
	if len(d.label) >= labelLen || bytes.IndexByte(d.label, 0) >= 0 {
		return nil, fmt.Errorf("sav: document label must fit one NUL-terminated label region")
	}
	body, err := serializeArchiveDocument(d.archive)
	if err != nil {
		return nil, err
	}
	blob := Compress(body)
	stateSize, err := worldStateSerializedSize(d.state)
	if err != nil {
		return nil, err
	}
	campaignSize, err := archiveCampaignSize(d.campaign)
	if err != nil {
		return nil, err
	}
	if uint64(headerLen)+uint64(len(blob))+labelLen+stateSize+campaignSize > maxArchiveOutput {
		return nil, fmt.Errorf("sav: complete output exceeds %d bytes", maxArchiveOutput)
	}
	var state []byte
	if d.archive.world != nil {
		state, err = serializeWorldState(d.state)
	} else {
		state, err = serializeCityState(d.state)
	}
	if err != nil {
		return nil, err
	}
	campaign, err := serializeCityCampaign(d.campaign)
	if err != nil {
		return nil, err
	}
	if uint64(headerLen)+uint64(len(blob))+labelLen+uint64(len(state))+uint64(len(campaign)) > maxArchiveOutput {
		return nil, fmt.Errorf("sav: complete output exceeds %d bytes", maxArchiveOutput)
	}
	out := make([]byte, headerLen, headerLen+len(blob)+labelLen+len(state)+len(campaign))
	copy(out, Magic)
	binary.LittleEndian.PutUint32(out[4:], uint32(headerLen+len(blob)))
	binary.LittleEndian.PutUint32(out[8:], d.version)
	binary.LittleEndian.PutUint32(out[12:], uint32(len(blob)))
	out = append(out, blob...)
	// SAV-LABELTAIL-236: only the NUL-terminated name has source provenance.
	// A new output buffer is allowed; source label debris is not game state.
	region := make([]byte, labelLen)
	copy(region, d.label)
	out = append(out, region...)
	out = append(out, state...)
	out = append(out, campaign...)
	return out, nil
}

func archiveCampaignSize(c *cityCampaign) (uint64, error) {
	if c == nil {
		return 0, fmt.Errorf("sav: missing campaign")
	}
	// The fixed grammar is104 bytes. Every variable element is included before
	// the existing campaign encoder can append it; its semantic checks follow.
	n := uint64(104) + 36*uint64(len(c.children)) + 4*uint64(len(c.dwords)) + 8*uint64(len(c.documents)+len(c.carriers))
	for _, a := range c.base.arrays {
		n += 2 * uint64(len(a))
	}
	for _, child := range c.children {
		for _, a := range child.base.arrays {
			n += 2 * uint64(len(a))
		}
	}
	for _, a := range c.parallel {
		n += 2 * uint64(len(a))
	}
	for _, a := range c.arrays {
		n += 2 * uint64(len(a))
	}
	for _, marker := range c.markers {
		n += 16 + uint64(len(marker.text))
	}
	if n > maxArchiveOutput {
		return 0, fmt.Errorf("sav: campaign output exceeds %d bytes", maxArchiveOutput)
	}
	return n, nil
}
