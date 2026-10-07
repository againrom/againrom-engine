# 1345 — SAV document carrier

## Result

The engine can keep Againrom-only bytes inside an ordinary SAV and read them
back after the original game loads and resaves the file. No feature uses it.
A SAV with no payload is written byte for byte as before; no World, scenario or
document hash moves. `againrom.exe` goes to 0.75.0; `starter.exe` is unchanged.

## Authority

- Owner direction: keep a persistence mechanism ready, carried by the campaign
  Valuable Documents collection, with no feature on it.
- `MISSION-DOC-021`, `SAV-CAMPAIGN-085`, `SAV-CAMPTAIL-070`: the campaign record
  writes documents as `u32 count` then `count` pairs `(u32 value, u32 kind)`.
- Owner-observed on the original EN game (kit `review/owner-docs-carrier-kit`):
  SAVs with 4, 1000 and 26426 appended pairs `(1, 0x80000000 | x)` load; the
  original's resave keeps every pair bit for bit and in order. Its documents
  panel shows each pair as one more page of document 1; 26426 pairs take about
  4 s to open the panel. LOAD and SAVE show no delay.
- Seat inference, not a promoted claim: the original reads and writes the pairs
  raw (no kind check, count limit or dedup), its panel tests kind against zero
  only, and a registry grant appends at the end, so a later vanilla grant lands
  after the carriers.
- Unknown: the original's behaviour above 26426 pairs, and whether the panel
  cost grows with the carrier count beyond the one measured size.

## As built

Codec, `pkg/formats/sav/docpayload.go`, no gameplay knowledge:

- `type DocPayload struct{ Version uint16; Data []byte }`.
- `EncodeDocPayload(*DocPayload) ([]uint32, error)` and
  `DecodeDocPayload([]uint32) (*DocPayload, error)` pack and unpack 31-bit chunks.
- Envelope: `ARSV`, version (u16), exact length (u32), data, CRC-32 (IEEE, over
  version, length and data), all little endian, packed least significant bit
  first into 31-bit chunks; the last chunk is zero padded. An empty payload is
  zero carriers. Output is a pure function of the payload.
- Decode checks the chunk count, each chunk's width, the magic, the length
  against the chunk count (before any allocation sized from the file), the
  padding bits and the CRC. It never panics.
- A carrier pair is `(1, 0x80000000 | chunk)`. `maxCampaignDocumentPairs` is
  65536 for the whole document list; vanilla pairs keep the old limit of 4096.
  The payload limit follows from it (about 253 KB).

SAV boundary, `pkg/formats/sav`:

- A document pair whose kind has the high bit set is a carrier; high bit clear
  and kind above 1 is refused as before.
- `CampaignProjection.Documents` holds vanilla pairs in file order wherever the
  carriers sit. `Payload`, `PayloadError` and `CarrierRecords` carry the decoded
  result. The editable campaign record keeps raw carrier pairs apart from the
  vanilla documents and writes them after.
- `applyCityCampaignProjection` and `ProjectDocumentPayload` write the current
  payload as one block and drop the old one.

Session, `pkg/game`: `Town` holds the payload from LOAD to the next SAVE
(`DocPayload`, `SetDocPayload`, `DocPayloadError`). It travels in `Snapshot`
(`DocPayload`, additive gob field, nil on older payloads). Every SAVE route
writes it: the town and mission projection, and the final step of
`ExportCurrentSave`.

| carrier pairs read | engine state | next SAVE writes |
|---|---|---|
| none | no payload | none |
| one valid envelope, version 1 | payload | the payload, one block |
| one valid envelope, unknown version | payload with that version, bytes untouched | the same envelope verbatim |
| invalid: bad magic, CRC or length, truncated, extra record | no payload, diagnostic on stderr and `DocPayloadError` | none |
| valid envelope plus stray pairs | whole sequence is invalid: no payload, diagnostic | none |
| payload set to nil or empty | no payload | none |

The carrier sequence is judged as one unit; a valid prefix followed by stray
pairs is invalid, and nothing is partly kept.

`savtool campaign` prints a payload line only when carrier pairs are present.

## Proof

- `pkg/formats/sav/docpayload_test.go`: payload sizes 0 to 129 bytes, 127 B,
  1 KiB, 100 KiB (26430 carriers with the envelope); patterns 0x00, 0xff,
  0xaa/0x55, seeded random; frames of 30, 31, 32, 62 and 63 bits; every carrier
  has the high bit; vanilla pairs interleaved with carriers keep their order and
  serialise as vanilla then one block; corruption cases (missing and extra
  record, flipped bit, bad magic, padding bit, impossible length, CRC mismatch,
  empty sequence, random junk) decode to an error without a panic; repeated
  apply does not accumulate carriers.
- `pkg/game/docpayload_test.go`: a town SAVE through the production route, LOAD
  in a fresh front end, three further SAVE and LOAD cycles with the same carrier
  count, then LOAD in a separate process. Loss control in the same test: a
  session whose payload is dropped writes no carriers and loads none. The panel
  document list is checked equal to the vanilla list after every LOAD. Stray and
  unknown-version cases follow the policy table.
- `TestReleaseDocPayloadOwnerResaves` (variable `AGAINROM_DOCS_CARRIER_KIT`):
  the three owner resaves decode their vanilla documents `(1,1)(2,1)(3,1)` and
  4, 1000 and 26426 carrier records through the production reader; their
  carriers are raw sequence values, so they take the stray path. The files stay
  outside the repository.
- `TestReleaseDocPayloadMissionSAVRoundTrip` (EN and RU, corpus variable
  `AGAINROM_SAVE_CORPUS`): the original mission SAV `game0001-from-9281` loads,
  takes a payload, SAVEs through `ExportCurrentSave`, cold LOADs with payload and
  documents intact and SAVEs again with one block. With
  `AGAINROM_DOCS_CARRIER_OUT` it writes the owner kit candidate.

## Open debt

- No feature uses the payload. A feature must state its size and accept that the
  original shows one page of document 1 per carrier (DIV-2342).
- The diagnostic for ignored carriers goes to stderr and `Town.DocPayloadError`;
  no on-screen notice exists.
- DIV-2343 through DIV-2345 are returned unused.
