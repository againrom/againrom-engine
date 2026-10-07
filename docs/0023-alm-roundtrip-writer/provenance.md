# Provenance — lossless ALM round-trip writer

Pinned at research `3d95f2a`, frozen for the story; `claims/retracted.md` read first — the five
retracted ALM rows (`ALM-TRL-005`, `ALM-HDR-006`, `ALM-GRID-011` in part, `ALM-HDR-029`,
`ALM-TRL-030`) are pre-corrected-framing readings the landed reader already embodies.
Re-read at `03a9448`, 2026-07-30: `ALM-SEC-003`'s count/set/order clauses are retracted **as a
contract**, and the reader this story delegates acceptance to has widened to the loader's own
gates. `spec.md` follows it and pins no record count of its own.

## Backing

| `spec.md` anchor | Claim | Confidence |
|---|---|---|
| The 20-byte file header and 20-byte record header, every field named; hdrLen a live self-describing length | `ALM-FRAME-031`, `ALM-HDR-001`, `ALM-SEC-002` | High — six independent proofs incl. a bytes-only refutation; chain/stride carried by the loader's own `Read(hdr,0x14)` |
| recordCount is the loader's loop bound, gated `>= 3`; formatVersion its version gate, `<= 1001`, `1000` the header-skipping dialect (refused here) | `ALM-META-024` (amended) | High — each gate a named read. 990 and count 10 are what the gates see on the 38 EN maps, not acceptance bounds |
| typeId is the dispatch word | `ALM-SEC-003` | High — record-header dword 3, 10-entry table at `L02078` |
| `{type0, type1, type2}` required, every other id optional; a repeat last-wins, an id at or above 10 stepped over, bytes past the last record unread; **an absent type-3 manufactured** as a `W·H` zero plane, so the decoded view can hold a section the bytes do not | `ALM-REQ-055`, `ALM-REQ-056` | High — the post-loop block's two failing tests and each absence arm are named instructions. `ALM-SEC-003`'s "exactly ten, `{0..9}` once each" is **retracted** as a contract: a shipped four-record map loads (`ALM-CORP-060`) |
| The shipped physical order `0,1,2,3,5,4,9,8,6,7` | `ALM-SEC-003` (**retracted**), `ALM-ORD-057` | Was **Medium**; the permutation is unenforced but three precedence relations hold at High, and preserving file order satisfies all three |
| `dataSize` (+0x08): `= 4·W·H + 72` on 38/38, the `+72` unexplained | `ALM-HDR-001` | **Medium** (the identity) → preserved verbatim, never recomputed or validated |
| The per-map constant (+0x10): a record-header field, byte-identical across a map's records, 3-value corpus domain | `ALM-FRAME-031`, `ALM-SEC-003`, `ALM-META-009` | High that it is a header field / **Medium** (byte-identity and domain — corpus) → carried as raw bits, never regenerated from the float |
| The type-0 angle (+0x08) read into the map object | `ALM-META-027` | High (read) / Medium (sun-angle naming) → raw bits preserved; the NaN-payload AC |

The four fields the baseline held unknown — file `+0x08`, record `+0x00`/`+0x04`/`+0x10` — are
cited for what they ARE, never as licence to interpret.

## Ours by choice

| What the spec fixes | What the evidence says |
|---|---|
| Preservation, not regeneration: the writer emits retained bytes, recomputes nothing and supplies no record the stream lacked | The identity `Write(OpenDocument(b)) == b` is then arithmetic over the record walk's partition — no new format fact consumed; the Medium rows (`dataSize`, order, per-map constant) are carried, not built on. Supplying nothing is also what makes the manufactured type-3 plane safe to expose |
| The document's acceptance is exactly `Open`'s, whatever `Open` accepts — delegated, never restated | The reader has since widened to the loader's own four gates — magic, `recordCount >= 3`, `formatVersion <= 1001`, then type-1 and type-2 present (`ALM-REQ-055`, `ALM-META-024`) — keeping three refusals it argues in its own contract. This story owns no accept set, so the two cannot drift |
| API vocabulary: `OpenDocument`, records, `RecordCount`/`RecordTypeIDs`/`RecordPayload` | The tree's entry points are `Open`/`OpenInfo` over records; `Parse` and "section" exist nowhere |
| Copy-in, copy-out ownership (FR-3) | Mirrors the shipped `Map`, which aliases nothing; an aliasing document would make the identity contingent on caller discipline |
| type-7/8/9 bodies stay opaque | Research has since closed their grammar over the corpus (`ALM-TRIG-044…050`, Medium, no consuming instruction) — not adopted: a lossless writer needs no grammar, and Medium structure under a byte-identity contract buys nothing |

## Open / undecoded

- **The widened accept set over shipped bytes.** `ALM-CORP-060` scopes the shipped corpus at 44
  names / 72 walked files over both preserved roots, 71 of ten records and one of four; the corpus
  run here covers the EN root's 38, all ten-record. The widened shapes have a synthetic witness
  and no corpus one, and no criterion asks for one.
- **`dataSize` semantics.** Unread by the loader; the arithmetic identity is Medium
  (`ALM-HDR-001`, restorable to High by a `rom.exe` site that writes or validates it). Verbatim
  preservation is correct under every outcome.
- **The per-map constant's meaning.** Domain and per-map byte-identity are corpus Medium
  (`ALM-META-009`). Raw bits dodge the question.
- **type-0 `+0x28`** is read-and-discarded, meaning Unknown (`ALM-META-025`); already exposed raw
  by the shipped `Meta`.

## Removed from the baseline and why

- **The unread-header-bytes premise** ("12 bytes of every section header and 4 of the file header
  unread"). This reader reads and validates or stores all of them; rewritten as the
  pinned-vs-free split the contract states.
- **Its AC-2** (non-zero bytes in section-header +0x00/+0x04 surviving). A rejected input here —
  tag and hdrLen are validated. Replaced by the freedom that exists: dataSize, version, the
  per-map constants, the permutation.
- **The 16-bit coordinate/type-id truncation clause and its AC-4.** No truncation in this reader
  (u32 throughout; `ClassID int16` is the loader's own MOVSX, bijective).
- **The `RawSection` aliasing sentence.** No such type; the shipped `Map` copies everything.
- **"Sections of every kind including unknown/raw ids" (in its FR-2).** Dropped while `Open`
  refused a typeId outside {0..9}, a duplicate and a wrong count. `Open` has since widened; the
  document was always indifferent, holding whatever records the stream carried.
- **"General name/author".** The fields here are the type-0 name and description, plus the type-5
  names.
- **`ParseDocument`/`SectionCount`/`SectionIDs`/`SectionBody`.** Renamed to the tree's
  vocabulary. `SectionCount` was dropped as a constant accessor while acceptance pinned ten; the
  count is a file field now, and `RecordCount` is back as `RecordPayload`'s bound.
- **Its R-1 block and the ROM2 lead labels** (`alm_offsetplayers`, `sec_junk1`,
  `sec_headersize`, `sec_junk2`). Third-party reimplementation labels, not citable here; the
  question is since answered from the game's own loader for three of the four fields
  (`ALM-FRAME-031`), the fourth staying open above. No research request remains.
- **`OPENROM_ALM_CORPUS`.** No such convention here; the corpus check is `almtool`'s new verb.
