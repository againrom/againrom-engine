package sav

import (
	"bytes"
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

func payloadPatterns() map[string]func(n int) []byte {
	fill := func(b byte) func(int) []byte {
		return func(n int) []byte { return bytes.Repeat([]byte{b}, n) }
	}
	return map[string]func(int) []byte{
		"zero": fill(0x00), "ones": fill(0xff),
		"alt": func(n int) []byte {
			out := make([]byte, n)
			for i := range out {
				out[i] = [2]byte{0xaa, 0x55}[i&1]
			}
			return out
		},
		"random": func(n int) []byte {
			out := make([]byte, n)
			rand.New(rand.NewSource(int64(n) + 7)).Read(out)
			return out
		},
	}
}

// chunksOf packs a frame the way EncodeDocPayload does, for corruption cases
// that need a frame the encoder would never write.
func chunksOf(frame []byte) []uint32 {
	chunks := make([]uint32, carrierCount(len(frame)))
	for bit := 0; bit < 8*len(frame); bit++ {
		if frame[bit>>3]>>(bit&7)&1 != 0 {
			chunks[bit/31] |= 1 << (bit % 31)
		}
	}
	return chunks
}

func TestDocPayloadRoundTripSizesAndPatterns(t *testing.T) {
	sizes := []int{0, 1, 2, 3, 4, 127, 1024, 100 * 1024}
	for n := 5; n < 130; n++ {
		sizes = append(sizes, n)
	}
	for name, gen := range payloadPatterns() {
		for _, n := range sizes {
			p := &DocPayload{Version: PayloadVersion, Data: gen(n)}
			chunks, err := EncodeDocPayload(p)
			if err != nil {
				t.Fatalf("%s %d: %v", name, n, err)
			}
			if n == 0 {
				if len(chunks) != 0 {
					t.Fatalf("%s: empty payload has %d carriers", name, len(chunks))
				}
				continue
			}
			if want := carrierCount(14 + n); len(chunks) != want {
				t.Fatalf("%s %d: %d carriers, want %d", name, n, len(chunks), want)
			}
			for i, c := range chunks {
				if c > carrierChunkMax {
					t.Fatalf("%s %d: chunk %d exceeds 31 bits", name, n, i)
				}
			}
			again, _ := EncodeDocPayload(p)
			if !reflect.DeepEqual(chunks, again) {
				t.Fatalf("%s %d: encoding is not deterministic", name, n)
			}
			got, err := DecodeDocPayload(chunks)
			if err != nil || got.Version != PayloadVersion || !bytes.Equal(got.Data, p.Data) {
				t.Fatalf("%s %d: round trip = %v, %v", name, n, got, err)
			}
		}
	}
}

func TestDocPayloadBitBoundaries(t *testing.T) {
	// Frames of 30, 31, 32, 62 and 63 bits are not whole bytes; the nearest
	// whole-byte frames put the last data bit on each side of a chunk edge.
	for _, bits := range []int{30, 31, 32, 62, 63} {
		frame := make([]byte, (bits+7)/8)
		for i := range frame {
			frame[i] = 0xff
		}
		if bits%8 != 0 {
			frame[len(frame)-1] &= byte(1<<(bits%8)) - 1
		}
		chunks := chunksOf(frame)
		if len(chunks) != (8*len(frame)+30)/31 {
			t.Fatalf("%d bits: %d chunks", bits, len(chunks))
		}
		set := 0
		for _, c := range chunks {
			for ; c != 0; c &= c - 1 {
				set++
			}
		}
		if set != bits {
			t.Fatalf("%d bits: %d bits set", bits, set)
		}
	}
}

func TestDocPayloadLargest100KiBCarrierCount(t *testing.T) {
	chunks, err := EncodeDocPayload(&DocPayload{Version: PayloadVersion, Data: make([]byte, 100*1024)})
	if err != nil || len(chunks) != 26430 {
		t.Fatalf("100 KiB: %d carriers, %v", len(chunks), err)
	}
}

func TestDocPayloadEveryCarrierPairHasHighBit(t *testing.T) {
	pairs, err := carrierPairs(&DocPayload{Version: PayloadVersion, Data: payloadPatterns()["random"](500)})
	if err != nil || len(pairs) == 0 {
		t.Fatal(pairs, err)
	}
	for i, pair := range pairs {
		if pair[0] != 1 || !IsCarrierKind(pair[1]) {
			t.Fatalf("pair %d = %#x", i, pair)
		}
	}
}

func TestDocPayloadCorruptionNeverPanics(t *testing.T) {
	p := &DocPayload{Version: PayloadVersion, Data: payloadPatterns()["random"](200)}
	good, _ := EncodeDocPayload(p)
	clone := func() []uint32 { return append([]uint32(nil), good...) }
	cases := map[string][]uint32{
		"empty":        nil,
		"missing last": good[:len(good)-1],
		"extra record": append(clone(), 0),
		"extra high":   append(clone(), 1<<31),
		"one record":   good[:1],
	}
	flip := clone()
	flip[3] ^= 1 << 9
	cases["flipped bit"] = flip
	magic := clone()
	magic[0] ^= 1
	cases["bad magic"] = magic
	pad := clone()
	pad[len(pad)-1] |= 1 << 30
	cases["padding bit"] = pad
	over := clone()
	over[0] |= 1 << 31
	cases["over 31 bits"] = over
	frame := payloadFrame(p)
	frame[6], frame[7], frame[8], frame[9] = 0xff, 0xff, 0xff, 0xff
	cases["impossible length"] = chunksOf(frame)
	frame = payloadFrame(p)
	frame[len(frame)-1] ^= 0x80
	cases["crc mismatch"] = chunksOf(frame)
	for name, chunks := range cases {
		if got, err := DecodeDocPayload(chunks); err == nil || got != nil {
			t.Errorf("%s: decoded %v, %v", name, got, err)
		}
	}
	rng := rand.New(rand.NewSource(3))
	for i := 0; i < 2000; i++ {
		junk := make([]uint32, rng.Intn(40))
		for j := range junk {
			junk[j] = rng.Uint32() >> 1
		}
		DecodeDocPayload(junk)
		DecodeDocPayload(junk[:len(junk)/2])
	}
}

func TestDocPayloadUnknownVersionRoundTripsVerbatim(t *testing.T) {
	p := &DocPayload{Version: 9, Data: []byte("later layout")}
	chunks, err := EncodeDocPayload(p)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeDocPayload(chunks)
	if err != nil || got.Version != 9 || string(got.Data) != "later layout" {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestDocPayloadOversizeRefusedOnEncode(t *testing.T) {
	if _, err := EncodeDocPayload(&DocPayload{Version: 1, Data: make([]byte, maxPayloadData+1)}); err == nil {
		t.Fatal("oversize payload accepted")
	}
	if chunks, err := EncodeDocPayload(&DocPayload{Version: 1, Data: make([]byte, maxPayloadData)}); err != nil || len(chunks) > maxCampaignDocumentPairs {
		t.Fatalf("largest payload: %d, %v", len(chunks), err)
	}
}

func projectionWithPairs(pairs [][2]uint32) []byte {
	want, _ := campaignProjectionFixture()
	want.Documents = nil
	for _, p := range pairs {
		want.Documents = append(want.Documents, CampaignDocument{Value: p[0], Kind: p[1]})
	}
	return encodeCampaignProjection(want, 0xfeedbeef, want.ScoreEvents)
}

func TestCampaignProjectionSeparatesCarriersFromVanilla(t *testing.T) {
	payload := &DocPayload{Version: PayloadVersion, Data: payloadPatterns()["random"](100)}
	carriers, err := carrierPairs(payload)
	if err != nil {
		t.Fatal(err)
	}
	vanilla := [][2]uint32{{1, 1}, {2, 1}, {3, 0}}
	// Vanilla pairs sit between carriers; both keep their file order.
	var pairs [][2]uint32
	pairs = append(pairs, vanilla[0])
	pairs = append(pairs, carriers[:2]...)
	pairs = append(pairs, vanilla[1])
	pairs = append(pairs, carriers[2:]...)
	pairs = append(pairs, vanilla[2])
	got, err := parseCampaignProjection(projectionWithPairs(pairs))
	if err != nil {
		t.Fatal(err)
	}
	want := []CampaignDocument{{1, 1}, {2, 1}, {3, 0}}
	if !reflect.DeepEqual(got.Documents, want) {
		t.Fatalf("documents = %v, want %v", got.Documents, want)
	}
	if got.PayloadError != "" || got.Payload == nil || !bytes.Equal(got.Payload.Data, payload.Data) {
		t.Fatalf("payload = %v, %q", got.Payload, got.PayloadError)
	}
	cc, err := parseCityCampaign(projectionWithPairs(pairs))
	if err != nil {
		t.Fatal(err)
	}
	if len(cc.documents) != 3 || !reflect.DeepEqual(cc.carriers, carriers) {
		t.Fatalf("city campaign split = %v / %d carriers", cc.documents, len(cc.carriers))
	}
	out, err := serializeCityCampaign(cc)
	if err != nil {
		t.Fatal(err)
	}
	sorted := append(append([][2]uint32{}, vanilla...), carriers...)
	if !bytes.Equal(out, projectionWithPairs(sorted)) {
		t.Fatal("serialize writes vanilla pairs then one carrier block")
	}
}

func TestCampaignProjectionCarrierPolicies(t *testing.T) {
	good, _ := carrierPairs(&DocPayload{Version: PayloadVersion, Data: []byte("payload")})
	unknown, _ := carrierPairs(&DocPayload{Version: 7, Data: []byte("later")})
	stray := [][2]uint32{{1, 0x80000000 | 0x1234567}}
	cases := []struct {
		name    string
		pairs   [][2]uint32
		payload bool
		version uint16
		errPart string
	}{
		{"none", nil, false, 0, ""},
		{"valid", good, true, PayloadVersion, ""},
		{"unknown version", unknown, true, 7, ""},
		{"stray only", stray, false, 0, "magic"},
		{"valid plus stray", append(append([][2]uint32{}, good...), stray...), false, 0, "does not match"},
		{"truncated", good[:len(good)-1], false, 0, "overruns"},
	}
	for _, tc := range cases {
		got, err := parseCampaignProjection(projectionWithPairs(append([][2]uint32{{1, 1}}, tc.pairs...)))
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if len(got.Documents) != 1 || (got.Payload != nil) != tc.payload {
			t.Fatalf("%s: docs %v payload %v err %q", tc.name, got.Documents, got.Payload, got.PayloadError)
		}
		if tc.payload && got.Payload.Version != tc.version {
			t.Fatalf("%s: version %d", tc.name, got.Payload.Version)
		}
		if !tc.payload && len(tc.pairs) != 0 && got.PayloadError == "" {
			t.Fatalf("%s: no diagnostic", tc.name)
		}
		if !strings.Contains(got.PayloadError, tc.errPart) {
			t.Fatalf("%s: diagnostic %q", tc.name, got.PayloadError)
		}
	}
}

func TestCampaignProjectionKeepsRefusalOfPlainBadKind(t *testing.T) {
	if _, err := parseCampaignProjection(projectionWithPairs([][2]uint32{{1, 2}})); err == nil {
		t.Fatal("kind 2 accepted")
	}
	if _, err := parseCityCampaign(projectionWithPairs([][2]uint32{{1, 0x7fffffff}})); err == nil {
		t.Fatal("kind 0x7fffffff accepted")
	}
}

func TestCampaignDocumentCountLimits(t *testing.T) {
	big := make([][2]uint32, 5000)
	for i := range big {
		big[i] = [2]uint32{uint32(i), 1}
	}
	if _, err := parseCampaignProjection(projectionWithPairs(big)); err == nil {
		t.Fatal("5000 vanilla documents accepted")
	}
	carriers := make([][2]uint32, 40000)
	for i := range carriers {
		carriers[i] = [2]uint32{1, 0x80000000 | uint32(i)}
	}
	got, err := parseCampaignProjection(projectionWithPairs(carriers))
	if err != nil || got.PayloadError == "" {
		t.Fatalf("40000 carriers: %v %q", err, got.PayloadError)
	}
	over := make([][2]uint32, maxCampaignDocumentPairs+1)
	for i := range over {
		over[i] = [2]uint32{1, 0x80000000}
	}
	if _, err := parseCampaignProjection(projectionWithPairs(over)); err == nil {
		t.Fatal("document count above the bound accepted")
	}
}

func TestApplyProjectionWritesCurrentPayloadOnce(t *testing.T) {
	cc, err := parseCityCampaign(projectionWithPairs([][2]uint32{{1, 1}}))
	if err != nil {
		t.Fatal(err)
	}
	proj, _ := campaignProjectionFixture()
	proj.Payload = &DocPayload{Version: PayloadVersion, Data: []byte("first")}
	for i := 0; i < 3; i++ {
		if err := applyCityCampaignProjection(cc, proj); err != nil {
			t.Fatal(err)
		}
	}
	want, _ := carrierPairs(proj.Payload)
	if !reflect.DeepEqual(cc.carriers, want) {
		t.Fatalf("repeated apply accumulated carriers: %d, want %d", len(cc.carriers), len(want))
	}
	proj.Payload = nil
	if err := applyCityCampaignProjection(cc, proj); err != nil || len(cc.carriers) != 0 {
		t.Fatalf("nil payload left %d carriers, %v", len(cc.carriers), err)
	}
	proj.Payload = &DocPayload{Version: PayloadVersion}
	if err := applyCityCampaignProjection(cc, proj); err != nil || len(cc.carriers) != 0 {
		t.Fatalf("empty payload left %d carriers, %v", len(cc.carriers), err)
	}
}

func TestApplyProjectionRefusesCarrierKindInVanillaDocuments(t *testing.T) {
	cc, _ := parseCityCampaign(projectionWithPairs(nil))
	proj, _ := campaignProjectionFixture()
	proj.Documents = []CampaignDocument{{1, 0x80000000}}
	if err := applyCityCampaignProjection(cc, proj); err == nil {
		t.Fatal("a carrier kind was accepted as a vanilla document")
	}
}

func TestDocumentDataCarriesPayloadThroughValidation(t *testing.T) {
	cc, err := parseCityCampaign(projectionWithPairs([][2]uint32{{1, 1}}))
	if err != nil {
		t.Fatal(err)
	}
	p := &DocPayload{Version: PayloadVersion, Data: []byte("abc")}
	pairs, _ := carrierPairs(p)
	cc.carriers = pairs
	data := cityToDataCampaign(*cc, nil)
	if err := validateDocumentCampaign(data); err != nil {
		t.Fatal(err)
	}
	back := cityFromDataCampaign(data, nil)
	if !reflect.DeepEqual(back.carriers, pairs) {
		t.Fatal("carriers lost through the document form")
	}
	var doc DocumentData
	doc.Campaign = data
	got, why := DocumentPayload(doc)
	if why != "" || got == nil || string(got.Data) != "abc" {
		t.Fatalf("DocumentPayload = %v %q", got, why)
	}
	if err := ProjectDocumentPayload(&doc, nil); err != nil || len(doc.Campaign.Carriers) != 0 {
		t.Fatal("nil payload kept carriers")
	}
	data.Carriers = [][2]uint32{{1, 1}}
	if err := validateDocumentCampaign(data); err == nil {
		t.Fatal("a vanilla pair was accepted as a carrier")
	}
}

func TestDocPayloadFitsBeside(t *testing.T) {
	data := func(n int) *DocPayload { return &DocPayload{Version: PayloadVersion, Data: make([]byte, n)} }
	for _, vanilla := range []int{0, 3, 4096} {
		edge := (31*(maxCampaignDocumentPairs-vanilla))/8 - 14
		if !DocPayloadFits(data(edge), vanilla) || DocPayloadFits(data(edge+1), vanilla) {
			t.Fatalf("vanilla %d: edge %d is not the largest fitting payload", vanilla, edge)
		}
		doc := &DocumentData{}
		doc.Campaign.Documents = make([][2]uint32, vanilla)
		if err := ProjectDocumentPayload(doc, data(edge+1)); err == nil {
			t.Fatalf("vanilla %d: ProjectDocumentPayload took a payload past the edge", vanilla)
		}
		if err := ProjectDocumentPayload(doc, data(edge)); err != nil || len(doc.Campaign.Documents)+len(doc.Campaign.Carriers) != maxCampaignDocumentPairs {
			t.Fatalf("vanilla %d: edge payload: %v", vanilla, err)
		}
	}
	if !DocPayloadFits(nil, maxCampaignDocumentPairs) || DocPayloadFits(data(1), maxCampaignDocumentPairs) {
		t.Fatal("empty payload must always fit and a non-empty one must not fit a full list")
	}
}
